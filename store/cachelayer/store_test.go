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
	if wlog.GlobalLogger() == nil {
		wlog.InitGlobalLogger(wlog.NewLogger(&wlog.LoggerConfiguration{}))
	}

	all := append([]ExternalStoreManagerOption{
		WithSQLiteSandboxDir(env.root),
	}, opts...)

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

	db, err := sql.Open(SQLiteDriverName, path)
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

func sqliteCfg(dsn string) ExternalSQLStoreConnectConfig {
	return ExternalSQLStoreConnectConfig{Driver: "sqlite3", DSN: dsn}
}

func mustConnect(t *testing.T, m *ExternalStoreManager, dsn string) *ExternalDb {
	t.Helper()

	d, err := m.Connect(context.Background(), sqliteCfg(dsn))
	if err != nil {
		t.Fatalf("connect %q: %v", dsn, err)
	}

	t.Cleanup(func() { _ = d.db.Close() })

	return d
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
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
			got, err := m.resolveSandboxPath(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveSandboxPath_SandboxRootIsSymlink(t *testing.T) {
	env := newSandboxEnv(t)

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	linkRoot := filepath.Join(filepath.Dir(env.root), "sandbox-link")
	mustSymlink(t, env.root, linkRoot)

	m := NewExternalStoreManager(
		WithSQLiteSandboxDir(linkRoot),
		WithLogger(wlog.NewLogger(&wlog.LoggerConfiguration{})),
	)

	want := filepath.Join(env.root, "a.db")

	for name, in := range map[string]string{
		"relative":               "a.db",
		"absolute via link":      filepath.Join(linkRoot, "a.db"),
		"absolute via real root": want,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := m.resolveSandboxPath(in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != want {
				t.Fatalf("got %q, want %q", got, want)
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := m.resolveSandboxPath(tc.input)
			if err == nil {
				t.Fatalf("expected denial, got path %q", got)
			}

			if got != "" {
				t.Fatalf("path must be empty on error, got %q", got)
			}
		})
	}
}

func TestResolveSandboxPath_NoExistenceOracle(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	existing := filepath.Join(env.outside, "secret.db")
	createSQLiteDB(t, existing)
	missing := filepath.Join(env.outside, "missing.db")

	_, e1 := m.resolveSandboxPath(existing)
	_, e2 := m.resolveSandboxPath(missing)

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

func TestBuildSQLiteDSN_ModeAndPragmas(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	dbPath := filepath.Join(env.root, "a.db")
	createSQLiteDB(t, dbPath)

	tests := []struct {
		name          string
		writable      bool
		mode          string
		wantPragmas   []string
		forbidPragmas []string
	}{
		{
			name: "ro", writable: false, mode: "ro",
			wantPragmas: []string{"query_only(1)", "trusted_schema(0)", "busy_timeout(5000)"},
		},
		{
			name: "rw", writable: true, mode: "rw",
			wantPragmas:   []string{"trusted_schema(0)", "busy_timeout(5000)"},
			forbidPragmas: []string{"query_only(1)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

			if len(q["mode"]) != 1 {
				t.Fatalf("mode must be set exactly once: %v", q["mode"])
			}

			pragmas := make(map[string]bool)
			for _, p := range q["_pragma"] {
				pragmas[p] = true
			}

			for _, p := range tc.wantPragmas {
				if !pragmas[p] {
					t.Fatalf("missing pragma %q in %v", p, q["_pragma"])
				}
			}

			for _, p := range tc.forbidPragmas {
				if pragmas[p] {
					t.Fatalf("unexpected pragma %q in %v", p, q["_pragma"])
				}
			}

			for k := range q {
				if k != "mode" && k != "_pragma" {
					t.Fatalf("unexpected DSN parameter %q", k)
				}
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

	if u.Path != dbPath || u.Query().Get("mode") != "ro" {
		t.Fatalf("bad dsn %q (path=%q query=%v)", dsn, u.Path, u.Query())
	}

	d := mustConnect(t, m, name)

	row, qerr := d.Query(context.Background(), "SELECT COUNT(*) AS c FROM items", nil)
	if qerr != nil || row["c"] != "2" {
		t.Fatalf("row=%v err=%v", row, qerr)
	}
}

func TestResolveConnection_SQLiteVariants(t *testing.T) {
	env := newSandboxEnv(t)
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	for _, writeEnabled := range []bool{false, true} {
		m := env.manager(WithWriteOperationsEnabled(writeEnabled))

		for _, drv := range []string{"sqlite3", "sqlite", "SQLITE3", " Sqlite "} {
			t.Run(fmt.Sprintf("%q/write=%v", drv, writeEnabled), func(t *testing.T) {
				cfg := ExternalSQLStoreConnectConfig{Driver: drv, DSN: "a.db"}

				rc, err := m.resolveConnection(cfg)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if rc.Driver != SQLiteDriverName {
					t.Fatalf("driver = %q, want %q", rc.Driver, SQLiteDriverName)
				}

				if !rc.SQLite {
					t.Fatal("SQLite flag must be set")
				}

				if rc.Writable != writeEnabled {
					t.Fatalf("writable = %v, want %v", rc.Writable, writeEnabled)
				}

				if !strings.HasPrefix(rc.DSN, "file:") {
					t.Fatalf("dsn must be built by service, got %q", rc.DSN)
				}

				if cfg.Driver != drv || cfg.DSN != "a.db" {
					t.Fatalf("input cfg was modified: %+v", cfg)
				}
			})
		}
	}
}

func TestResolveConnection_NonSQLitePassThrough(t *testing.T) {
	m := NewExternalStoreManager()

	raw := "user:pass@tcp(127.0.0.1:3306)/db"

	rc, err := m.resolveConnection(ExternalSQLStoreConnectConfig{Driver: "mysql", DSN: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rc.Driver != "mysql" || rc.DSN != raw || rc.SQLite || rc.Writable {
		t.Fatalf("unexpected result: %+v", rc)
	}
}

func TestResolveConnection_SQLiteRejectsBadPath(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	rc, err := m.resolveConnection(sqliteCfg("../outside/secret.db"))
	if err == nil {
		t.Fatalf("expected rejection, got %+v", rc)
	}

	if rc != (resolvedConnection{}) {
		t.Fatalf("result must be zero on error, got %+v", rc)
	}
}

func TestConnect_ReadOnly_WhenWritesDisabled(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db")

	row, err := d.Query(context.Background(), "SELECT id, name FROM items ORDER BY id", nil)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if row["id"] != "1" || row["name"] != "alpha" {
		t.Fatalf("unexpected row: %v", row)
	}

	t.Run("blocked by guard", func(t *testing.T) {
		if _, qerr := d.Query(context.Background(), "INSERT INTO items (id, name) VALUES (3, 'gamma')", nil); qerr == nil {
			t.Fatal("INSERT must be rejected by the statement guard")
		}
	})

	t.Run("blocked by DSN", func(t *testing.T) {
		_, werr := d.db.ExecContext(context.Background(), "INSERT INTO items (id, name) VALUES (3, 'gamma')")
		if werr == nil || !strings.Contains(strings.ToLower(werr.Error()), "readonly") {
			t.Fatalf("write must fail with readonly error, got: %v", werr)
		}
	})
}

func TestConnect_Writable_WhenEnabled(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db")

	row, err := d.Query(context.Background(), "INSERT INTO items (id, name) VALUES (3, 'gamma') RETURNING id", nil)
	if err != nil || row["id"] != "3" {
		t.Fatalf("insert via guard: row=%v err=%v", row, err)
	}

	cnt, qerr := d.Query(context.Background(), "SELECT COUNT(*) AS c FROM items", nil)
	if qerr != nil || cnt["c"] != "3" {
		t.Fatalf("row=%v err=%v", cnt, qerr)
	}
}

func TestConnect_RW_DoesNotCreateMissingFile(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager(WithWriteOperationsEnabled(true))

	d, err := m.Connect(context.Background(), sqliteCfg("new.db"))
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
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	ro := env.manager()
	rw := env.manager(WithWriteOperationsEnabled(true))

	t.Run("same config hits the cache", func(t *testing.T) {
		d1 := mustConnect(t, ro, "a.db")
		d2 := mustConnect(t, ro, "a.db")

		if d1 != d2 {
			t.Fatal("same config must hit the cache")
		}
	})

	t.Run("relative and absolute forms share an entry", func(t *testing.T) {
		d1 := mustConnect(t, ro, "a.db")
		d2 := mustConnect(t, ro, filepath.Join(env.root, "a.db"))

		if d1 != d2 {
			t.Fatal("relative and absolute forms of the same file must share a cache entry")
		}
	})

	t.Run("ro and rw use different keys", func(t *testing.T) {
		roRC, err := ro.resolveConnection(sqliteCfg("a.db"))
		if err != nil {
			t.Fatal(err)
		}

		rwRC, err := rw.resolveConnection(sqliteCfg("a.db"))
		if err != nil {
			t.Fatal(err)
		}

		if roRC.DSN == rwRC.DSN {
			t.Fatalf("ro and rw must produce different DSNs: %q", roRC.DSN)
		}

		d1 := mustConnect(t, ro, "a.db")
		d2 := mustConnect(t, rw, "a.db")

		if d1.writable || !d2.writable {
			t.Fatalf("writable flags: ro=%v rw=%v", d1.writable, d2.writable)
		}
	})
}

func TestConnect_Rejected(t *testing.T) {
	env := newSandboxEnv(t)

	secret := filepath.Join(env.outside, "secret.db")
	createSQLiteDB(t, secret)
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	inputs := []string{
		"", ":memory:", "file:a.db", "file:" + secret, "a.db?mode=rwc",
		"../outside/secret.db", secret, "/etc/passwd",
	}

	for _, writeEnabled := range []bool{false, true} {
		m := env.manager(WithWriteOperationsEnabled(writeEnabled))

		for _, in := range inputs {
			t.Run(fmt.Sprintf("%q/write=%v", in, writeEnabled), func(t *testing.T) {
				d, err := m.Connect(context.Background(), sqliteCfg(in))
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

	d, err := m.Connect(context.Background(), sqliteCfg("a.db"))
	if err == nil {
		_ = d.db.Close()
		t.Fatal("sqlite must be disabled when sandbox is not configured")
	}
}

func TestConnect_UserDSNNeverUsedRaw(t *testing.T) {
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
			d, err := m.Connect(context.Background(), ExternalSQLStoreConnectConfig{
				Driver: SQLiteDriverName,
				DSN:    dsn,
			})
			if err == nil {
				_ = d.db.Close()
				t.Fatalf("user-supplied DSN %q bypasses sandbox validation", dsn)
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

			d, err := m.Connect(context.Background(), sqliteCfg("a.db"))
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

func TestSQLiteGuard(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		write bool
		ok    bool
	}{
		{"select", "SELECT 1", false, true},
		{"select lower", "select 1", false, true},
		{"select trailing semicolon", "SELECT 1;  -- done", false, true},
		{"with cte", "WITH x AS (SELECT 1) SELECT * FROM x", false, true},
		{"values", "VALUES (1), (2)", false, true},
		{"leading comment", "/* hi */ -- yo\n SELECT 1", false, true},
		{"semicolon in string", "SELECT 'a;b'", false, true},
		{"semicolon in quoted identifier", `SELECT 1 AS "a;b"`, false, true},
		{"semicolon in bracket identifier", "SELECT 1 AS [a;b]", false, true},
		{"escaped quote", "SELECT 'it''s; fine'", false, true},

		{"attach", "ATTACH DATABASE '/tmp/x.db' AS x", false, false},
		{"attach lower", "attach database '/tmp/x.db' as x", false, false},
		{"attach bound param", "ATTACH DATABASE ? AS x", false, false},
		{"detach", "DETACH DATABASE x", false, false},
		{"vacuum", "VACUUM", false, false},
		{"vacuum into", "VACUUM INTO '/tmp/out.db'", false, false},
		{"vacuum into after comment", "/**/ vacuum into '/tmp/out.db'", false, false},
		{"pragma", "PRAGMA writable_schema=1", false, false},
		{"create", "CREATE TABLE t(x)", false, false},
		{"drop", "DROP TABLE items", false, false},
		{"multi statement", "SELECT 1; VACUUM INTO '/tmp/out.db'", false, false},
		{"multi statement w/ comment", "SELECT 1; /* x */ ATTACH 'a' AS b", false, false},
		{"unterminated string", "SELECT 'abc", false, false},
		{"unterminated bracket", "SELECT [abc", false, false},
		{"empty", "   ", false, false},
		{"only comment", "-- nothing", false, false},

		{"insert ro", "INSERT INTO t VALUES (1)", false, false},
		{"update ro", "UPDATE t SET x = 1", false, false},
		{"delete ro", "DELETE FROM t", false, false},
		{"insert rw", "INSERT INTO t VALUES (1)", true, true},
		{"update rw", "UPDATE t SET x = 1", true, true},
		{"delete rw", "DELETE FROM t", true, true},
		{"replace rw", "REPLACE INTO t VALUES (1)", true, true},
		{"vacuum into rw", "VACUUM INTO '/tmp/out.db'", true, false},
		{"attach rw", "ATTACH 'a' AS b", true, false},
		{"pragma rw", "PRAGMA writable_schema=1", true, false},
		{"drop rw", "DROP TABLE t", true, false},
		{"multi statement rw", "INSERT INTO t VALUES (1); ATTACH 'a' AS b", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSQLiteStatement(tc.sql, tc.write)
			if (err == nil) != tc.ok {
				t.Fatalf("ok=%v, err=%v", tc.ok, err)
			}
		})
	}
}

func TestGuard_BlocksAttachVacuumPragma(t *testing.T) {
	env := newSandboxEnv(t)

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	secret := filepath.Join(env.outside, "secret.db")
	createSQLiteDBWithSQL(t, secret, `CREATE TABLE secrets (v TEXT); INSERT INTO secrets VALUES ('TOP');`)

	leak := filepath.Join(env.outside, "leak.db")
	before := readFile(t, secret)

	tests := []struct {
		name   string
		sql    string
		params []interface{}
	}{
		{"attach plain", fmt.Sprintf("ATTACH DATABASE '%s' AS x", secret), nil},
		{"attach lower case", fmt.Sprintf("attach '%s' as x", secret), nil},
		{"attach leading comment", fmt.Sprintf("/* c */ ATTACH '%s' AS x", secret), nil},
		{"attach after another statement", fmt.Sprintf("SELECT 1; ATTACH '%s' AS x", secret), nil},
		{"attach bound parameter", "ATTACH DATABASE ? AS x", []interface{}{secret}},
		{"attach uri form", fmt.Sprintf("ATTACH 'file:%s?mode=rw' AS x", secret), nil},
		{"attach memory", "ATTACH ':memory:' AS x", nil},
		{"attach temp db", "ATTACH '' AS x", nil},

		{"vacuum into", fmt.Sprintf("VACUUM INTO '%s'", leak), nil},
		{"vacuum into main", fmt.Sprintf("VACUUM main INTO '%s'", leak), nil},
		{"vacuum into bound parameter", "VACUUM INTO ?", []interface{}{leak}},
		{"vacuum into leading comment", fmt.Sprintf("/**/ vacuum into '%s'", leak), nil},
		{"vacuum into after another statement", fmt.Sprintf("SELECT 1; VACUUM INTO '%s'", leak), nil},

		{"pragma writable_schema", "PRAGMA writable_schema = 1", nil},
		{"pragma query_only off", "PRAGMA query_only = 0", nil},
	}

	for _, writeEnabled := range []bool{false, true} {
		m := env.manager(WithWriteOperationsEnabled(writeEnabled))
		d := mustConnect(t, m, "a.db")

		for _, tc := range tests {
			t.Run(fmt.Sprintf("%s/write=%v", tc.name, writeEnabled), func(t *testing.T) {
				if _, qerr := d.Query(context.Background(), tc.sql, tc.params); qerr == nil {
					t.Fatalf("%q must be denied", tc.sql)
				}

				if _, qerr := d.Query(context.Background(), "SELECT v FROM x.secrets", nil); qerr == nil {
					t.Fatal("attached schema must not be reachable")
				}
			})
		}
	}

	if _, err := os.Stat(leak); !os.IsNotExist(err) {
		t.Fatalf("VACUUM INTO created a file outside the sandbox (stat err=%v)", err)
	}

	if after := readFile(t, secret); before != after {
		t.Fatal("outside file was modified")
	}
}

func TestSQLite_LoadExtensionDisabled(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db")

	if _, err := d.Query(context.Background(), "SELECT load_extension('/nonexistent/lib.so')", nil); err == nil {
		t.Fatal("load_extension must not be available")
	}
}

func TestQuery_Semantics(t *testing.T) {
	env := newSandboxEnv(t)
	m := env.manager()

	createSQLiteDB(t, filepath.Join(env.root, "a.db"))
	d := mustConnect(t, m, "a.db")

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

func TestQuery_NonSQLiteDbSkipsGuard(t *testing.T) {
	env := newSandboxEnv(t)
	createSQLiteDB(t, filepath.Join(env.root, "a.db"))

	raw, err := sql.Open(SQLiteDriverName, filepath.Join(env.root, "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })

	d := &ExternalDb{db: raw, useAllowList: false}

	if _, qerr := d.Query(context.Background(), "PRAGMA user_version", nil); qerr != nil {
		t.Fatalf("guard must not apply to non-sqlite connections: %v", qerr)
	}
}
