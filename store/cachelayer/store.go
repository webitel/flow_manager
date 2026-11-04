package cachelayer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"
	"modernc.org/sqlite"
)

const (
	CacheSize   = 100
	CacheExpire = 25 * 1000

	SQLiteDriverName = "sqlite_sandboxed"
)

func init() {
	drv := new(sqlite.Driver)
	drv.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		r, ok := conn.(sqlite.AuthorizerRegisterer)
		if !ok {
			return errors.New("sqlite: authorizer is not supported")
		}
		return r.RegisterAuthorizer(sqliteAuthorizer)
	})
	sql.Register(SQLiteDriverName, drv)
}

func sqliteAuthorizer(action sqlite.AuthorizerActionCode, _, _, _, _ string) sqlite.AuthorizerReturnCode {
	switch action {
	case sqlite.AuthSelect, sqlite.AuthRead, sqlite.AuthFunction, sqlite.AuthRecursive,
		sqlite.AuthInsert, sqlite.AuthUpdate, sqlite.AuthDelete:
		return sqlite.AuthorizerOK
	default:
		return sqlite.AuthorizerDeny
	}
}

type ExternalStoreManagerOption func(*ExternalStoreManager)

func WithSQLiteSandboxDir(sandbox string) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) { esm.sqliteSandboxDir = sandbox }
}

func WithWriteOperationsEnabled(enabled bool) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) { esm.sqliteWriteEnabled = enabled }
}

func WithLogger(logger *wlog.Logger) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) { esm.logger = logger }
}

type ExternalStoreManager struct {
	sqliteSandboxDir   string
	sqliteWriteEnabled bool
	cache              model.ObjectCache
	logger             *wlog.Logger
	sf                 singleflight.Group
}

type ExternalDb struct {
	db *sql.DB
}

type ExternalSQLStoreConnectConfig struct {
	Driver string
	DSN    string
}

func NewExternalStoreManager(options ...ExternalStoreManagerOption) *ExternalStoreManager {
	m := &ExternalStoreManager{
		cache:  model.NewLruWithParams(CacheSize, "external", CacheExpire, ""),
		logger: wlog.GlobalLogger().With(wlog.Namespace("external_sql")),
	}
	for _, o := range options {
		o(m)
	}
	return m
}

func (e *ExternalStoreManager) Connect(_ context.Context, cfg ExternalSQLStoreConnectConfig) (*ExternalDb, *model.AppError) {
	driver, dsn, appErr := e.resolveConnection(cfg)
	if appErr != nil {
		return nil, appErr
	}

	key := driver + "|" + dsn

	if d, ok := e.cache.Get(key); ok {
		return d.(*ExternalDb), nil
	}

	v, err, _ := e.sf.Do(key, func() (any, error) {
		if d, ok := e.cache.Get(key); ok {
			return d, nil
		}

		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, err
		}
		db.SetConnMaxIdleTime(time.Second * (CacheExpire - 60))

		ext := &ExternalDb{db: db}
		e.cache.AddWithDefaultExpires(key, ext)
		e.logger.Info("store cache db", wlog.String("driver", driver))

		return ext, nil
	})
	if err != nil {
		return nil, model.NewAppError("Cache", "cache.connect.open_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return v.(*ExternalDb), nil
}

func (e *ExternalStoreManager) resolveConnection(cfg ExternalSQLStoreConnectConfig) (driver, dsn string, appErr *model.AppError) {
	if !isSQLiteDriver(cfg.Driver) {
		return cfg.Driver, cfg.DSN, nil
	}

	dsn, appErr = e.buildSQLiteDSN(cfg.DSN)
	return SQLiteDriverName, dsn, appErr
}

func (d *ExternalDb) Query(ctx context.Context, text string, params []any) (map[string]any, *model.AppError) {
	rows, err := d.db.QueryContext(ctx, text, params...)
	if err != nil {
		return nil, queryError("cache.query.err", err)
	}
	defer rows.Close()

	names, err := rows.Columns()
	if err != nil {
		return nil, queryError("cache.query.columns.err", err)
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, queryError("cachelayer.store.external_db.rows_next", err)
		}
		return map[string]any{}, nil
	}

	values := make([]any, len(names))
	ptrs := make([]any, len(names))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if err := rows.Scan(ptrs...); err != nil {
		return nil, queryError("cache.query.scan.err", err)
	}

	result := make(map[string]any, len(names))
	for i, v := range values {
		if b, ok := v.([]byte); ok {
			result[names[i]] = string(b)
		} else {
			result[names[i]] = fmt.Sprint(v)
		}
	}

	return result, nil
}

func queryError(id string, err error) *model.AppError {
	return model.NewAppError("Cache.Query", id, nil, err.Error(), http.StatusInternalServerError)
}

func isSQLiteDriver(driver string) bool {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite", "sqlite3", SQLiteDriverName:
		return true
	default:
		return false
	}
}

func (e *ExternalStoreManager) buildSQLiteDSN(userPath string) (string, *model.AppError) {
	if e.sqliteSandboxDir == "" {
		return "", model.NewAppError("BuildSQLiteDSN", "cachelayer.store.build_sqlite_dsn.empty", nil,
			"sqlite is disabled: sandbox directory not configured", http.StatusBadRequest)
	}

	realPath, err := e.resolveSandboxPath(userPath)
	if err != nil {
		return "", err
	}

	mode := "ro"
	if e.sqliteWriteEnabled {
		mode = "rw"
	}

	q := url.Values{}
	q.Set("mode", mode)
	q.Add("_pragma", "trusted_schema(0)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_defensive", "1")

	u := url.URL{Scheme: "file", OmitHost: true, Path: realPath, RawQuery: q.Encode()}
	return u.String(), nil
}

func (e *ExternalStoreManager) resolveSandboxPath(p string) (string, *model.AppError) {
	fail := func(code, detail string, status int) (string, *model.AppError) {
		return "", model.NewAppError("ResolveSandboxPath", "cachelayer.store.resolve_sandbox_path."+code, nil, detail, status)
	}

	if p = strings.TrimSpace(p); p == "" {
		return fail("empty_path", "empty sqlite path", http.StatusBadRequest)
	}

	root, err := filepath.Abs(e.sqliteSandboxDir)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return fail("sandbox_root", "sandbox directory is not accessible", http.StatusInternalServerError)
	}

	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}

	realPath, err := filepath.EvalSymlinks(p)
	if err != nil || !isWithin(root, realPath) {
		return fail("not_accessible", "file is not accessible", http.StatusNotFound)
	}

	if fi, err := os.Stat(realPath); err != nil || !fi.Mode().IsRegular() {
		return fail("not_regular", "not a regular file", http.StatusBadRequest)
	}

	return realPath, nil
}

func isWithin(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
