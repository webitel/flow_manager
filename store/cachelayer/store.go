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
	_ "modernc.org/sqlite"
)

const (
	CacheSize   = 100
	CacheExpire = 25 * 1000
)

const (
	SQLiteDriverName        = "sqlite"
	sqliteModeRO     string = "ro"
	sqliteModeRW     string = "rw"
)

type ExternalStoreManagerOption func(*ExternalStoreManager)

func WithSQLiteSandboxDir(sandbox string) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) { esm.sqliteSandboxDir = sandbox }
}

func WithWriteOperationsEnabled(enabled bool) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) { esm.writeEnabled = enabled }
}

func WithLogger(logger *wlog.Logger) ExternalStoreManagerOption {
	return func(esm *ExternalStoreManager) { esm.logger = logger }
}

type ExternalStoreManager struct {
	sqliteSandboxDir string
	writeEnabled     bool
	cache            model.ObjectCache
	logger           *wlog.Logger
	sf               singleflight.Group
}

type ExternalDb struct {
	db           *sql.DB
	useAllowList bool
	writable     bool
}

type ExternalSQLStoreConnectConfig struct {
	Driver string
	DSN    string
}

type resolvedConnection struct {
	Driver   string
	DSN      string
	SQLite   bool
	Writable bool
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
	rc, werr := e.resolveConnection(cfg)
	if werr != nil {
		return nil, werr
	}

	key := rc.Driver + "|" + rc.DSN

	if d, ok := e.cache.Get(key); ok {
		return d.(*ExternalDb), nil
	}

	v, err, _ := e.sf.Do(key, func() (any, error) {
		if d, ok := e.cache.Get(key); ok {
			return d, nil
		}

		db, err := sql.Open(rc.Driver, rc.DSN)
		if err != nil {
			return nil, err
		}
		db.SetConnMaxIdleTime(time.Second * (CacheExpire - 60))

		ext := &ExternalDb{db: db, useAllowList: rc.SQLite, writable: rc.Writable}

		e.cache.AddWithDefaultExpires(key, ext)
		e.logger.Info("store cache db", wlog.String("driver", rc.Driver))

		return ext, nil
	})
	if err != nil {
		return nil, model.NewAppError("Cache", "cache.connect.open_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return v.(*ExternalDb), nil
}

func (e *ExternalStoreManager) resolveConnection(cfg ExternalSQLStoreConnectConfig) (resolvedConnection, *model.AppError) {
	if !isSQLiteDriver(cfg.Driver) {
		return resolvedConnection{Driver: cfg.Driver, DSN: cfg.DSN}, nil
	}

	dsn, err := e.buildSQLiteDSN(cfg.DSN, e.writeEnabled)
	if err != nil {
		return resolvedConnection{}, err
	}

	return resolvedConnection{
		Driver:   SQLiteDriverName,
		DSN:      dsn,
		SQLite:   true,
		Writable: e.writeEnabled,
	}, nil
}

func (d *ExternalDb) Query(ctx context.Context, text string, params []any) (map[string]any, *model.AppError) {
	if d.useAllowList {
		if err := checkSQLiteStatement(text, d.writable); err != nil {
			return nil, model.NewAppError("Cache.Query", "cache.query.sqlite_forbidden", nil, err.Error(), http.StatusBadRequest)
		}
	}

	rows, err := d.db.QueryContext(ctx, text, params...)
	if err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.err", nil, err.Error(), http.StatusInternalServerError)
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.columns.err", nil, err.Error(), http.StatusInternalServerError)
	}

	cols := make([]any, len(colNames))
	colPtrs := make([]any, len(colNames))
	for i := range colNames {
		colPtrs[i] = &cols[i]
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, model.NewAppError("Query", "cachelayer.store.external_db.rows_next", nil,
				fmt.Sprintf("iterating rows result: %+v", err), http.StatusInternalServerError)
		}
		return make(map[string]any), nil
	}

	if err = rows.Scan(colPtrs...); err != nil {
		return nil, model.NewAppError("Cache.Query", "cache.query.scan.err", nil, err.Error(), http.StatusInternalServerError)
	}

	result := make(map[string]any, len(cols))
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
	case "sqlite", "sqlite3":
		return true
	default:
		return false
	}
}

func (e *ExternalStoreManager) buildSQLiteDSN(userPath string, writable bool) (string, *model.AppError) {
	if e.sqliteSandboxDir == "" {
		return "", model.NewAppError("BuildSQLiteDSN", "cachelayer.store.build_sqlite_dsn.empty", nil,
			"sqlite is disabled: sandbox directory not configured", http.StatusBadRequest)
	}

	realPath, err := e.resolveSandboxPath(userPath)
	if err != nil {
		return "", err
	}

	mode := sqliteModeRO
	q := url.Values{}
	if writable {
		mode = sqliteModeRW
	} else {
		q.Add("_pragma", "query_only(1)")
	}
	q.Set("mode", mode)
	q.Add("_pragma", "trusted_schema(0)")
	q.Add("_pragma", "busy_timeout(5000)")

	u := url.URL{
		Scheme:   "file",
		OmitHost: true,
		Path:     realPath,
		RawQuery: q.Encode(),
	}

	return u.String(), nil
}
func isWithin(root, p string) bool {
	return p == root || strings.HasPrefix(p, root+string(os.PathSeparator))
}

func (e *ExternalStoreManager) resolveSandboxPath(p string) (string, *model.AppError) {
	fail := func(code, detail string, status int) (string, *model.AppError) {
		return "", model.NewAppError("ResolveSandboxPath", "cachelayer.store.resolve_sandbox_path."+code, nil, detail, status)
	}

	if p = strings.TrimSpace(p); p == "" {
		return fail("empty_path", "empty sqlite path", http.StatusBadRequest)
	}

	lower := strings.ToLower(p)
	if strings.HasPrefix(lower, ":") || strings.HasPrefix(lower, "file:") {
		return fail("not_allowed_params", "memory/URI forms are not allowed", http.StatusBadRequest)
	}

	if strings.ContainsAny(p, "\x00?#") {
		return fail("forbidden_characters", "forbidden characters in path", http.StatusBadRequest)
	}

	rawRoot := filepath.Clean(e.sqliteSandboxDir)
	root := rawRoot
	if r, err := filepath.EvalSymlinks(rawRoot); err == nil {
		root = r
	}

	if !filepath.IsAbs(p) {
		p = filepath.Join(rawRoot, p)
	}
	p = filepath.Clean(p)

	if !isWithin(rawRoot, p) && !isWithin(root, p) {
		return fail("escaping_sandbox", "path escapes sandbox", http.StatusForbidden)
	}

	realPath, err := filepath.EvalSymlinks(p)
	if err != nil {
		return fail("eval_symlinks", "file is not accessible", http.StatusNotFound)
	}

	if !isWithin(root, realPath) {
		return fail("escaping_sandbox", "path escapes sandbox", http.StatusForbidden)
	}

	fi, serr := os.Stat(realPath)
	if serr != nil {
		return fail("stat", "file is not accessible", http.StatusBadRequest)
	}
	if !fi.Mode().IsRegular() {
		return fail("not_regular", "not a regular file", http.StatusBadRequest)
	}

	return realPath, nil
}

var (
	sqliteReadStmts  = map[string]struct{}{"SELECT": {}, "WITH": {}, "VALUES": {}}
	sqliteWriteStmts = map[string]struct{}{"INSERT": {}, "UPDATE": {}, "DELETE": {}, "REPLACE": {}}
)

func checkSQLiteStatement(q string, allowWrite bool) error {
	i := skipSpaceAndComments(q, 0)

	start := i
	for i < len(q) && (q[i] >= 'a' && q[i] <= 'z' || q[i] >= 'A' && q[i] <= 'Z') {
		i++
	}
	kw := strings.ToUpper(q[start:i])

	if _, ok := sqliteReadStmts[kw]; !ok {
		if _, ok := sqliteWriteStmts[kw]; !(ok && allowWrite) {
			return fmt.Errorf("statement %q is not allowed", kw)
		}
	}

	return ensureSingleStatement(q, i)
}

func ensureSingleStatement(q string, i int) error {
	for i < len(q) {
		switch c := q[i]; {
		case c == '-' && strings.HasPrefix(q[i:], "--"), c == '/' && strings.HasPrefix(q[i:], "/*"):
			i = skipSpaceAndComments(q, i)
		case c == '\'' || c == '"' || c == '`':
			end, err := skipQuoted(q, i, c)
			if err != nil {
				return err
			}
			i = end
		case c == '[':
			end := strings.IndexByte(q[i:], ']')
			if end < 0 {
				return errors.New("unterminated identifier")
			}
			i += end + 1
		case c == ';':
			if skipSpaceAndComments(q, i+1) != len(q) {
				return errors.New("multiple statements are not allowed")
			}
			return nil
		default:
			i++
		}
	}
	return nil
}

func skipQuoted(q string, i int, quote byte) (int, error) {
	for j := i + 1; j < len(q); j++ {
		if q[j] != quote {
			continue
		}
		if j+1 < len(q) && q[j+1] == quote {
			j++
			continue
		}
		return j + 1, nil
	}
	return 0, errors.New("unterminated quoted string")
}

func skipSpaceAndComments(q string, i int) int {
	for i < len(q) {
		switch {
		case q[i] == ' ' || q[i] == '\t' || q[i] == '\n' || q[i] == '\r' || q[i] == '\f' || q[i] == '\v':
			i++
		case strings.HasPrefix(q[i:], "--"):
			nl := strings.IndexByte(q[i:], '\n')
			if nl < 0 {
				return len(q)
			}
			i += nl + 1
		case strings.HasPrefix(q[i:], "/*"):
			end := strings.Index(q[i+2:], "*/")
			if end < 0 {
				return len(q)
			}
			i += 2 + end + 2
		default:
			return i
		}
	}
	return i
}
