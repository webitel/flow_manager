package cachelayer

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/webitel/wlog"
)

type sandboxEnv struct {
	root    string
	outside string
}

func newSandboxEnv(t *testing.T) sandboxEnv {
	t.Helper()

	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval tempdir: %v", err)
	}

	env := sandboxEnv{
		root:    filepath.Join(base, "sandbox"),
		outside: filepath.Join(base, "outside"),
	}

	for _, d := range []string{env.root, env.outside} {
		if err := os.Mkdir(d, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	return env
}

func (env sandboxEnv) manager(opts ...ExternalStoreManagerOption) *ExternalStoreManager {
	all := append([]ExternalStoreManagerOption{WithSQLLiteSandboxDir(env.root), WithLogger(wlog.NewLogger(&wlog.LoggerConfiguration{}))}, opts...)

	return NewExternalStoreManager(all...)
}

const defaultFixtureSQL = `
CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO items (id, name) VALUES (1, 'alpha'), (2, 'beta');
`

func createSQLiteDB(t *testing.T, path string) {
	t.Helper()
	createSQLiteDBWithSQL(t, path, defaultFixtureSQL)
}

func createSQLiteDBWithSQL(t *testing.T, path, ddl string) {
	t.Helper()

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer db.Close()

	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("init fixture %s: %v", path, err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are not supported here: %v", err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.Mkdir(path, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustConnect(t *testing.T, m *ExternalStoreManager, dsn string, writable bool) *ExternalDb {
	t.Helper()

	d, err := m.Connect(context.Background(), &ExternalSQLStoreConnectConfig{
		Driver:   "sqlite3",
		DSN:      dsn,
		Writable: writable,
	})
	if err != nil {
		t.Fatalf("connect %q (writable=%v): %v", dsn, writable, err)
	}

	t.Cleanup(func() { _ = d.db.Close() })

	return d
}

func TestResolveSandboxPath_Allowed(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	dbPath := filepath.Join(env.root, "a.db")
	createSQLiteDB(t, dbPath)

	mustMkdir(t, filepath.Join(env.root, "sub"))
	nested := filepath.Join(env.root, "sub", "b.db")
	createSQLiteDB(t, nested)

	mustSymlink(t, dbPath, filepath.Join(env.root, "link.db"))

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"relative", "a.db", dbPath},
		{"absolute inside", dbPath, dbPath},
		{"nested relative", "sub/b.db", nested},
		{"dotdot that stays inside", "sub/../a.db", dbPath},
		{"surrounding whitespace", "  a.db  ", dbPath},
		{"symlink inside sandbox resolves to its target", "link.db", dbPath},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.resolveSanbdoxPath(tc.input, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveSandboxPath_Denied(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "ok.db"))

	secret := filepath.Join(env.outside, "secret.db")
	createSQLiteDB(t, secret)

	evilDir := env.root + "-evil"
	mustMkdir(t, evilDir)
	evil := filepath.Join(evilDir, "x.db")
	createSQLiteDB(t, evil)

	mustMkdir(t, filepath.Join(env.root, "dir.db"))
	mustSymlink(t, secret, filepath.Join(env.root, "escape.db"))
	mustSymlink(t, env.outside, filepath.Join(env.root, "escape_dir"))
	mustSymlink(t, filepath.Join(env.outside, "nope.db"), filepath.Join(env.root, "dangling.db"))

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "   "},
		{"memory", ":memory:"},
		{"colon prefix", ":anything"},
		{"file uri relative", "file:ok.db"},
		{"file uri upper case", "FILE:ok.db"},
		{"file uri absolute", "file:///etc/passwd"},
		{"query string", "ok.db?mode=rwc"},
		{"query string rw", "ok.db?mode=rw"},
		{"fragment", "ok.db#x"},
		{"nul byte", "ok.db\x00.txt"},

		{"dotdot", "../outside/secret.db"},
		{"dotdot after missing dir", "sub/../../outside/secret.db"},
		{"absolute outside", secret},
		{"etc passwd", "/etc/passwd"},
		{"sibling with common prefix", evil},
		{"url-encoded dotdot", "%2e%2e/outside/secret.db"},

		{"symlink to outside file", "escape.db"},
		{"symlink to outside dir", "escape_dir/secret.db"},
		{"dangling symlink", "dangling.db"},

		{"sandbox root itself (relative)", "."},
		{"sandbox root itself (absolute)", env.root},
		{"directory", "dir.db"},
		{"missing file", "missing.db"},
	}

	for _, writable := range []bool{false, true} {
		for _, tc := range tests {
			t.Run(fmt.Sprintf("%s/writable=%v", tc.name, writable), func(t *testing.T) {
				got, err := m.resolveSanbdoxPath(tc.input, writable)
				if err == nil {
					t.Fatalf("expected denial, got path %q", got)
				}

				if got != "" {
					t.Fatalf("path must be empty on error, got %q", got)
				}
			})
		}
	}
}

func TestResolveSandboxPath_NoExistenceOracle(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	existing := filepath.Join(env.outside, "secret.db")
	createSQLiteDB(t, existing)
	missing := filepath.Join(env.outside, "missing.db")

	_, e1 := m.resolveSanbdoxPath(existing, false)
	_, e2 := m.resolveSanbdoxPath(missing, false)

	if e1 == nil || e2 == nil {
		t.Fatalf("both must be denied: %v / %v", e1, e2)
	}

	if e1.StatusCode != e2.StatusCode || e1.Id != e2.Id {
		t.Fatalf("different responses leak file existence: (%d,%s) vs (%d,%s)",
			e1.StatusCode, e1.Id, e2.StatusCode, e2.Id)
	}

	for _, e := range []error{e1, e2} {
		if strings.Contains(e.Error(), env.outside) {
			t.Fatalf("error leaks real path: %v", e)
		}
	}
}

func TestBuildSQLiteDSN_Disabled_WhenNoSandbox(t *testing.T) {
	m := NewExternalStoreManager()

	dsn, err := m.buildSQLiteDSN("a.db", false)
	if err == nil || dsn != "" {
		t.Fatalf("sqlite must be disabled without sandbox, got dsn=%q err=%v", dsn, err)
	}
}

func TestBuildSQLiteDSN_Mode(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	dbPath := filepath.Join(env.root, "a.db")
	createSQLiteDB(t, dbPath)

	tests := []struct {
		writable bool
		mode     string
	}{
		{false, "ro"},
		{true, "rw"},
	}

	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			dsn, err := m.buildSQLiteDSN("a.db", tc.writable)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			u, perr := url.Parse(dsn)
			if perr != nil {
				t.Fatalf("parse %q: %v", dsn, perr)
			}

			if u.Scheme != "file" {
				t.Fatalf("scheme = %q", u.Scheme)
			}

			if u.Path != dbPath {
				t.Fatalf("path = %q, want %q", u.Path, dbPath)
			}

			q := u.Query()
			if q.Get("mode") != tc.mode {
				t.Fatalf("mode = %q, want %q", q.Get("mode"), tc.mode)
			}

			if len(q) != 1 {
				t.Fatalf("unexpected extra params: %v", q)
			}
		})
	}
}

func TestBuildSQLiteDSN_SpecialCharsInFileName(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	name := "my db & co.db"
	dbPath := filepath.Join(env.root, name)
	createSQLiteDB(t, dbPath)

	dsn, err := m.buildSQLiteDSN(name, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, perr := url.Parse(dsn)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}

	if u.Path != dbPath || len(u.Query()) != 1 || u.Query().Get("mode") != "ro" {
		t.Fatalf("bad dsn %q (path=%q query=%v)", dsn, u.Path, u.Query())
	}

	d := mustConnect(t, m, name, false)
	if _, qerr := d.Query(context.Background(), "SELECT 1 AS one", nil); qerr != nil {
		t.Fatalf("query: %v", qerr)
	}
}

func TestPrepareConnectionString_Drivers(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	for _, drv := range []string{"sqlite3", "sqlite", "SQLITE3", " Sqlite "} {
		t.Run("sqlite variant "+drv, func(t *testing.T) {
			cfg := &ExternalSQLStoreConnectConfig{Driver: drv, DSN: "a.db"}

			dsn, err := m.PrepareConnectionString(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Driver != SQLiteCustomDriverName {
				t.Fatalf("driver must be replaced with sandboxed one, got %q", cfg.Driver)
			}

			if !strings.HasPrefix(dsn, "file:") {
				t.Fatalf("dsn must be built by service, got %q", dsn)
			}
		})
	}

	t.Run("non-sqlite is passed through", func(t *testing.T) {
		raw := "user:pass@tcp(127.0.0.1:3306)/db"
		cfg := &ExternalSQLStoreConnectConfig{Driver: "mysql", DSN: raw}

		dsn, err := m.PrepareConnectionString(cfg)
		if err != nil || dsn != raw || cfg.Driver != "mysql" {
			t.Fatalf("got dsn=%q driver=%q err=%v", dsn, cfg.Driver, err)
		}
	})
}

func TestConnect_ReadOnlyByDefault(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db", false)

	row, err := d.Query(context.Background(), "SELECT id, name FROM items ORDER BY id", nil)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if row["id"] != "1" || row["name"] != "alpha" {
		t.Fatalf("unexpected row: %v", row)
	}

	_, werr := d.db.ExecContext(context.Background(), "INSERT INTO items (id, name) VALUES (3, 'gamma')")
	if werr == nil || !strings.Contains(strings.ToLower(werr.Error()), "readonly") {
		t.Fatalf("write must fail with readonly error, got: %v", werr)
	}
}

func TestConnect_WriteDowngraded_WhenServiceDisablesWrites(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db", true)

	_, werr := d.db.ExecContext(context.Background(), "INSERT INTO items (id, name) VALUES (3, 'gamma')")
	if werr == nil {
		t.Fatal("write must be impossible when service has writes disabled")
	}
}

func TestConnect_WriteAllowed_WhenEnabledAndRequested(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db", true)

	if _, err := d.db.ExecContext(context.Background(), "INSERT INTO items (id, name) VALUES (3, 'gamma')"); err != nil {
		t.Fatalf("write must succeed: %v", err)
	}

	row, qerr := d.Query(context.Background(), "SELECT COUNT(*) AS c FROM items", nil)
	if qerr != nil || row["c"] != "3" {
		t.Fatalf("row=%v err=%v", row, qerr)
	}
}

func TestConnect_RW_DoesNotCreateMissingFile(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	d, err := m.Connect(context.Background(), &ExternalSQLStoreConnectConfig{
		Driver: "sqlite3", DSN: "new.db", Writable: true,
	})
	if err == nil {
		_ = d.db.Close()
		t.Fatal("must not connect to a non-existing file")
	}

	if _, serr := os.Stat(filepath.Join(env.root, "new.db")); serr == nil {
		t.Fatal("file must not be created")
	}
}

func TestConnect_Cache(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	ro1 := mustConnect(t, m, "a.db", false)
	ro2 := mustConnect(t, m, "a.db", false)
	rw := mustConnect(t, m, "a.db", true)

	if ro1 != ro2 {
		t.Fatal("same config must hit the cache")
	}

	if ro1 == rw {
		t.Fatal("ro and rw connections must not share a cache entry")
	}

	abs := mustConnect(t, m, filepath.Join(env.root, "a.db"), false)
	if abs != ro1 {
		t.Fatal("relative and absolute forms of the same file must share a cache entry")
	}
}

func TestConnect_Rejected(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	secret := filepath.Join(env.outside, "secret.db")
	createSQLiteDB(t, secret)
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	inputs := []string{
		"", ":memory:", "file:a.db", "file:" + secret, "a.db?mode=rwc",
		"../outside/secret.db", secret, "/etc/passwd",
	}

	for _, writable := range []bool{false, true} {
		for _, in := range inputs {
			t.Run(fmt.Sprintf("%q/writable=%v", in, writable), func(t *testing.T) {
				d, err := m.Connect(context.Background(), &ExternalSQLStoreConnectConfig{
					Driver: "sqlite3", DSN: in, Writable: writable,
				})
				if err == nil {
					_ = d.db.Close()
					t.Fatal("expected rejection")
				}

				if d != nil {
					t.Fatal("db must be nil on rejection")
				}
			})
		}
	}
}

func TestConnect_SQLiteDisabled_WithoutSandbox(t *testing.T) {
	m := NewExternalStoreManager(WithLogger(wlog.NewLogger(&wlog.LoggerConfiguration{})))

	d, err := m.Connect(context.Background(), &ExternalSQLStoreConnectConfig{Driver: "sqlite3", DSN: "a.db"})
	if err == nil {
		_ = d.db.Close()
		t.Fatal("sqlite must be disabled when sandbox is not configured")
	}
}

func TestConnect_CannotBypassWithInternalDriverName(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	secret := filepath.Join(env.outside, "secret.db")
	createSQLiteDB(t, secret)

	dsns := []string{
		secret,
		"file:" + secret + "?mode=rw",
		":memory:",
	}

	for _, dsn := range dsns {
		t.Run(dsn, func(t *testing.T) {
			d, err := m.Connect(context.Background(), &ExternalSQLStoreConnectConfig{
				Driver:   SQLiteCustomDriverName,
				DSN:      dsn,
				Writable: true,
			})
			if err == nil {
				_ = d.db.Close()
				t.Fatalf("driver %q with user-supplied DSN bypasses sandbox validation", SQLiteCustomDriverName)
			}
		})
	}
}

func TestConnect_Concurrent_SingleInstancePerKey(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	const n = 32

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		got = make(map[*ExternalDb]struct{})
	)

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			d, err := m.Connect(context.Background(), &ExternalSQLStoreConnectConfig{Driver: "sqlite3", DSN: "a.db"})
			if err != nil {
				t.Errorf("connect: %v", err)
				return
			}

			mu.Lock()
			got[d] = struct{}{}
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 instance, got %d", len(got))
	}

	for d := range got {
		_ = d.db.Close()
	}
}

func TestAuthorizer_BlocksAttach(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	secret := filepath.Join(env.outside, "secret.db")
	createSQLiteDBWithSQL(t, secret, `CREATE TABLE secrets (v TEXT); INSERT INTO secrets VALUES ('TOP');`)

	before, err := os.ReadFile(secret)
	if err != nil {
		t.Fatalf("read secret: %v", err)
	}

	tests := []struct {
		name   string
		sql    string
		params []interface{}
	}{
		{"plain", fmt.Sprintf("ATTACH DATABASE '%s' AS x", secret), nil},
		{"lower case", fmt.Sprintf("attach '%s' as x", secret), nil},
		{"leading comment", fmt.Sprintf("/* c */ ATTACH '%s' AS x", secret), nil},
		{"after another statement", fmt.Sprintf("SELECT 1; ATTACH '%s' AS x", secret), nil},
		{"bound parameter", "ATTACH DATABASE ? AS x", []interface{}{secret}},
		{"uri form", fmt.Sprintf("ATTACH 'file:%s?mode=rw' AS x", secret), nil},
		{"memory", "ATTACH ':memory:' AS x", nil},
		{"temp db", "ATTACH '' AS x", nil},
	}

	for _, writable := range []bool{false, true} {
		d := mustConnect(t, m, "a.db", writable)

		for _, tc := range tests {
			t.Run(fmt.Sprintf("%s/writable=%v", tc.name, writable), func(t *testing.T) {
				if _, qerr := d.Query(context.Background(), tc.sql, tc.params); qerr == nil {
					t.Fatal("ATTACH must be denied")
				}

				if _, qerr := d.Query(context.Background(), "SELECT v FROM x.secrets", nil); qerr == nil {
					t.Fatal("attached schema must not be reachable")
				}
			})
		}
	}

	after, err := os.ReadFile(secret)
	if err != nil {
		t.Fatalf("read secret: %v", err)
	}

	if string(before) != string(after) {
		t.Fatal("outside file was modified")
	}
}

func TestSQLite_LoadExtensionDisabled(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db", false)

	if _, err := d.Query(context.Background(), "SELECT load_extension('/nonexistent/lib.so')", nil); err == nil {
		t.Fatal("load_extension must not be available")
	}
}

func TestQuery_Semantics(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db", false)

	ctx := context.Background()

	t.Run("empty result gives empty map", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT id FROM items WHERE id = 999", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if row == nil || len(row) != 0 {
			t.Fatalf("want empty non-nil map, got %#v", row)
		}
	})

	t.Run("only first row is returned", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT name FROM items ORDER BY id", nil)
		if err != nil || row["name"] != "alpha" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("bound parameters", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT name FROM items WHERE id = ?", []interface{}{2})
		if err != nil || row["name"] != "beta" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("sql injection via parameter is inert", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT name FROM items WHERE name = ?", []interface{}{"x' OR '1'='1"})
		if err != nil || len(row) != 0 {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("blob is converted to string", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT x'414243' AS b", nil)
		if err != nil || row["b"] != "ABC" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("NULL current representation", func(t *testing.T) {
		// фіксуємо поточну поведінку: fmt.Sprint(nil) == "<nil>"
		row, err := d.Query(ctx, "SELECT NULL AS n", nil)
		if err != nil || row["n"] != "<nil>" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("invalid sql returns error", func(t *testing.T) {
		if _, err := d.Query(ctx, "SELEC nonsense", nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		cancel()

		if _, err := d.Query(cctx, "SELECT 1", nil); err == nil {
			t.Fatal("expected error on canceled context")
		}
	})
}
