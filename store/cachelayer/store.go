package cachelayer

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"
)

func init() {
	sql.Register(SQLiteCustomDriverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(sc *sqlite3.SQLiteConn) error {
			sc.RegisterAuthorizer(func(op int, arg1, arg2, db string) int {
				if op == sqlite3.SQLITE_ATTACH {
					return sqlite3.SQLITE_DENY
				}

				return sqlite3.SQLITE_OK
			})

			return nil
		},
	})
}

const (
	CacheSize   = 100
	CacheExpire = 25 * 1000
)

const (
	SQLiteModeRO           string = "ro"
	SQLiteModeRW           string = "rw"
	SQLiteCustomDriverName string = "sqlite3_sandboxed"
)

type ExternalStoreManagerOption func(*ExternalStoreManager)

func WithSQLLiteSandboxDir(sandbox string) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) {
		esm.sqlLiteSandboxDir = sandbox
	}
}

func WithWriteOperationsEnabled(enabled bool) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) {
		esm.writeEnabled = enabled
	}
}

func WithLogger(logger *wlog.Logger) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) {
		esm.logger = logger
	}
}

type ExternalStoreManager struct {
	sqlLiteSandboxDir string
	writeEnabled      bool
	cache             model.ObjectCache
	logger            *wlog.Logger
	sf                singleflight.Group
}

type ExternalDb struct {
	db *sql.DB
}

type ExternalSQLStoreConnectConfig struct {
	Driver   string
	DSN      string
	Writable bool
}

func NewExternalStoreManager(options ...ExternalStoreManagerOption) *ExternalStoreManager {
	m := &ExternalStoreManager{
		cache: model.NewLruWithParams(CacheSize, "external", CacheExpire, ""),
	}

	for _, o := range options {
		o(m)
	}

	return m
}

func (e *ExternalStoreManager) Connect(_ context.Context, cfg *ExternalSQLStoreConnectConfig) (*ExternalDb, *model.AppError) {
	cfg.Writable = cfg.Writable && e.writeEnabled

	connStr, werr := e.PrepareConnectionString(cfg)
	if werr != nil {
		return nil, werr
	}

	key := cfg.Driver + "|" + connStr

	if d, ok := e.cache.Get(key); ok {
		return d.(*ExternalDb), nil
	}

	v, err, _ := e.sf.Do(key, func() (interface{}, error) {
		if d, ok := e.cache.Get(key); ok {
			return d, nil
		}

		db, err := sql.Open(cfg.Driver, connStr)
		if err != nil {
			return nil, err
		}
		db.SetConnMaxIdleTime(time.Second * (CacheExpire - 60))

		ext := &ExternalDb{db: db}

		e.cache.AddWithDefaultExpires(key, ext)
		e.logger.Info("store cache db", wlog.String("driver", cfg.Driver))

		return ext, nil
	})

	if err != nil {
		return nil, model.NewAppError("Cache", "cache.connect.open_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return v.(*ExternalDb), nil
}

func (e *ExternalStoreManager) PrepareConnectionString(connectCfg *ExternalSQLStoreConnectConfig) (string, *model.AppError) {
	if !isSQLiteDriver(connectCfg.Driver) {
		return connectCfg.DSN, nil
	}

	connectCfg.Driver = SQLiteCustomDriverName

	return e.buildSQLiteDSN(connectCfg.DSN, connectCfg.Writable)
}

func (d *ExternalDb) Query(ctx context.Context, text string, params []interface{}) (map[string]interface{}, *model.AppError) {
	rows, err := d.db.QueryContext(ctx, text, params...)

	if err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.err", nil, err.Error(), http.StatusInternalServerError)
	}

	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.columns.err", nil, err.Error(), http.StatusInternalServerError)
	}

	cols := make([]interface{}, len(colNames))
	colPtrs := make([]interface{}, len(colNames))
	for i := 0; i < len(colNames); i++ {
		colPtrs[i] = &cols[i]
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, model.NewAppError(
				"Query",
				"cachelayer.store.external_db.rows_next",
				nil,
				fmt.Sprintf("iterating rows result: %+v", err),
				500,
			)
		}

		return make(map[string]any), nil
	}

	if err = rows.Scan(colPtrs...); err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.scan.err", nil, err.Error(), http.StatusInternalServerError)
	}

	result := make(map[string]any)

	for i, col := range cols {
		switch col.(type) {
		case []uint8, []uint32, []uint:
			result[colNames[i]] = fmt.Sprintf("%s", col)
		default:
			result[colNames[i]] = fmt.Sprint(col)
		}
	}

	return result, nil
}

func isSQLiteDriver(driver string) bool {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite", "sqlite3", SQLiteCustomDriverName:
		return true
	default:
		return false
	}
}

func (e *ExternalStoreManager) buildSQLiteDSN(userPath string, writable bool) (string, *model.AppError) {
	if e.sqlLiteSandboxDir == "" {
		return "", model.NewAppError(
			"BuildSQLiteDSN",
			"cachelayer.store.build_sqlite_dsn.empty",
			nil,
			"sqlite is disabled: sandbox directory not configured",
			400,
		)
	}

	real, err := e.resolveSanbdoxPath(userPath, writable)
	if err != nil {
		return "", err
	}

	mode := SQLiteModeRO
	if writable {
		mode = SQLiteModeRW
	}

	u := url.URL{
		Scheme:   "file",
		OmitHost: true,
		Path:     real,
		RawQuery: "mode=" + mode,
	}

	return u.String(), nil
}

func (e *ExternalStoreManager) resolveSanbdoxPath(p string, writable bool) (string, *model.AppError) {
	if p = strings.TrimSpace(p); p == "" {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.empty_path",
			nil,
			"empty sqllite path",
			400,
		)
	}

	lower := strings.ToLower(p)
	if strings.HasPrefix(lower, ":") || strings.HasPrefix(lower, "file:") {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.not_allowed_params",
			nil,
			"memory/URI forms are not allowed",
			400,
		)
	}

	if strings.ContainsAny(p, "\x00?#") {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.forbidden_characters",
			nil,
			"forbidden characters in path",
			400,
		)
	}

	if !filepath.IsAbs(p) {
		p = filepath.Join(e.sqlLiteSandboxDir, p)
	}

	p = filepath.Clean(p)

	sandboxRoot := filepath.Clean(e.sqlLiteSandboxDir)
	rootWithSep := sandboxRoot + string(os.PathSeparator)
	if p != sandboxRoot && !strings.HasPrefix(p, rootWithSep) {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.escaping_sandbox",
			nil,
			"path escape sandbox",
			403,
		)
	}

	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.eval_symlinks",
			nil,
			fmt.Sprintf("file existance: %+v", err),
			404,
		)
	}

	if real != sandboxRoot && !strings.HasPrefix(real, rootWithSep) {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.escaping_sandbox",
			nil,
			"path escape sandbox via symlink",
			403,
		)
	}

	fi, serr := os.Stat(real)
	if serr != nil || !fi.Mode().IsRegular() {
		return "", model.NewAppError(
			"ResolveSandboxPath",
			"cachelayer.store.resolve_sandbox_path.stat",
			nil,
			fmt.Sprintf("file stat: %+v", serr),
			400,
		)
	}

	return real, nil
}
