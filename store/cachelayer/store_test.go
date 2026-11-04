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

func TestMain(m *testing.M) {
	if wlog.GlobalLogger() == nil {
		wlog.InitGlobalLogger(wlog.NewLogger(&wlog.LoggerConfiguration{}))
	}
	os.Exit(m.Run())
}

const defaultFixtureSQL = `
CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO items (id, name) VALUES (1, 'alpha'), (2, 'beta');
`

type sandbox struct {
	root    string
	outside string
}

func newSandbox(t *testing.T) sandbox {
	t.Helper()

	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval tempdir: %v", err)
	}

	s := sandbox{
		root:    filepath.Join(base, "sandbox"),
		outside: filepath.Join(base, "outside"),
	}
	mustMkdir(t, s.root)
	mustMkdir(t, s.outside)

	return s
}

func (s sandbox) manager(opts ...ExternalStoreManagerOption) *ExternalStoreManager {
	all := append([]ExternalStoreManagerOption{WithSQLiteSandboxDir(s.root)}, opts...)
	return NewExternalStoreManager(all...)
}

func (s sandbox) in(name string) string     { return filepath.Join(s.root, name) }
func (s sandbox) outsideOf(n string) string { return filepath.Join(s.outside, n) }

func fixtureDSN(path string) string {
	u := url.URL{Scheme: "file", OmitHost: true, Path: path}
	return u.String()
}

func createDB(t *testing.T, path string, stmts ...string) {
	t.Helper()

	script := defaultFixtureSQL
	if len(stmts) > 0 {
		script = strings.Join(stmts, ";\n")
	}

	db, err := sql.Open("sqlite", fixtureDSN(path))
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer db.Close()

	if _, err := db.Exec(script); err != nil {
		t.Fatalf("init fixture %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are not supported here: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
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

func forEachMode(t *testing.T, s sandbox, dbName string, body func(t *testing.T, d *ExternalDb)) {
	t.Helper()

	for _, writable := range []bool{false, true} {
		t.Run(fmt.Sprintf("write=%v", writable), func(t *testing.T) {
			d := mustConnect(t, s.manager(WithWriteOperationsEnabled(writable)), dbName)
			body(t, d)
		})
	}
}

func TestResolveSandboxPath(t *testing.T) {
	t.Run("allows", func(t *testing.T) {
		s := newSandbox(t)
		m := s.manager()

		dbPath := s.in("a.db")
		createDB(t, dbPath)

		mustMkdir(t, s.in("sub"))
		nested := s.in("sub/b.db")
		createDB(t, nested)

		mustSymlink(t, dbPath, s.in("link.db"))

		weird := s.in("we?ird #name%.db")
		createDB(t, weird)

		tests := []struct{ name, input, want string }{
			{"relative path", "a.db", dbPath},
			{"absolute path inside", dbPath, dbPath},
			{"nested relative path", "sub/b.db", nested},
			{"dotdot that stays inside", "sub/../a.db", dbPath},
			{"surrounding whitespace", "  a.db  ", dbPath},
			{"symlink inside sandbox (resolved to target)", "link.db", dbPath},
			{"file name with ? # % (is just a name now)", "we?ird #name%.db", weird},
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
	})

	t.Run("denies", func(t *testing.T) {
		s := newSandbox(t)
		m := s.manager()

		createDB(t, s.in("ok.db"))
		secret := s.outsideOf("secret.db")
		createDB(t, secret)

		evilDir := s.root + "-evil"
		mustMkdir(t, evilDir)
		evil := filepath.Join(evilDir, "x.db")
		createDB(t, evil)

		mustMkdir(t, s.in("dir.db"))
		mustSymlink(t, secret, s.in("escape.db"))
		mustSymlink(t, s.outside, s.in("escape_dir"))
		mustSymlink(t, s.outsideOf("nope.db"), s.in("dangling.db"))

		tests := []struct{ name, input string }{
			{"empty", ""},
			{"only spaces", "   "},

			{"memory", ":memory:"},
			{"file uri relative", "file:ok.db"},
			{"file uri absolute", "file:///etc/passwd"},
			{"query string", "ok.db?mode=rwc"},
			{"fragment", "ok.db#x"},
			{"nul byte", "ok.db\x00.txt"},
			{"url-encoded dotdot", "%2e%2e/outside/secret.db"},

			{"dotdot", "../outside/secret.db"},
			{"dotdot after inner dir", "sub/../../outside/secret.db"},
			{"absolute outside", secret},
			{"/etc/passwd", "/etc/passwd"},
			{"sibling with common prefix", evil},

			{"symlink to outside file", "escape.db"},
			{"symlink to outside dir", "escape_dir/secret.db"},
			{"dangling symlink", "dangling.db"},

			{"sandbox root (relative)", "."},
			{"sandbox root (absolute)", s.root},
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
	})

	t.Run("does not leak file existence", func(t *testing.T) {
		s := newSandbox(t)
		m := s.manager()

		existingOutside := s.outsideOf("secret.db")
		createDB(t, existingOutside)
		missingOutside := s.outsideOf("missing.db")

		_, errExisting := m.resolveSandboxPath(existingOutside)
		_, errMissing := m.resolveSandboxPath(missingOutside)

		if errExisting == nil || errMissing == nil {
			t.Fatalf("both must be denied: %v / %v", errExisting, errMissing)
		}
		if errExisting.StatusCode != errMissing.StatusCode || errExisting.Id != errMissing.Id {
			t.Fatalf("different responses leak existence: (%d,%s) vs (%d,%s)",
				errExisting.StatusCode, errExisting.Id, errMissing.StatusCode, errMissing.Id)
		}
		for _, e := range []error{errExisting, errMissing} {
			if strings.Contains(e.Error(), s.outside) {
				t.Fatalf("error leaks real path: %v", e)
			}
		}
	})

	t.Run("sandbox root is a symlink", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))

		linkRoot := filepath.Join(filepath.Dir(s.root), "sandbox-link")
		mustSymlink(t, s.root, linkRoot)

		m := NewExternalStoreManager(WithSQLiteSandboxDir(linkRoot))
		want := s.in("a.db")

		for name, input := range map[string]string{
			"relative":               "a.db",
			"absolute via link":      filepath.Join(linkRoot, "a.db"),
			"absolute via real root": want,
		} {
			t.Run(name, func(t *testing.T) {
				got, err := m.resolveSandboxPath(input)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != want {
					t.Fatalf("got %q, want %q", got, want)
				}
			})
		}
	})

	t.Run("sandbox dir given as relative path", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))

		t.Chdir(filepath.Dir(s.root))
		m := NewExternalStoreManager(WithSQLiteSandboxDir("sandbox"))

		inside, errInside := m.resolveSandboxPath("a.db")
		_, errOutside := m.resolveSandboxPath("../outside")

		if errInside != nil || inside != s.in("a.db") {
			t.Fatalf("got %q, err=%v", inside, errInside)
		}
		if errOutside == nil {
			t.Fatal("escape must be denied")
		}
	})
}

func TestBuildSQLiteDSN(t *testing.T) {
	t.Run("is disabled when sandbox is not configured", func(t *testing.T) {
		m := NewExternalStoreManager()

		dsn, err := m.buildSQLiteDSN("a.db")

		if err == nil || dsn != "" {
			t.Fatalf("want error and empty dsn, got dsn=%q err=%v", dsn, err)
		}
	})

	t.Run("sets mode and pragmas", func(t *testing.T) {
		tests := []struct {
			name     string
			writable bool
			mode     string
		}{
			{"read-only", false, "ro"},
			{"writable", true, "rw"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				s := newSandbox(t)
				createDB(t, s.in("a.db"))
				m := s.manager(WithWriteOperationsEnabled(tc.writable))

				dsn, err := m.buildSQLiteDSN("a.db")

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				u, perr := url.Parse(dsn)
				if perr != nil {
					t.Fatalf("parse %q: %v", dsn, perr)
				}
				q := u.Query()

				if u.Scheme != "file" || u.Path != s.in("a.db") {
					t.Fatalf("scheme=%q path=%q", u.Scheme, u.Path)
				}
				if got := q["mode"]; len(got) != 1 || got[0] != tc.mode {
					t.Fatalf("mode = %v, want [%s]", got, tc.mode)
				}
				for k := range q {
					if k != "mode" && k != "_pragma" && k != "_defensive" {
						t.Fatalf("unexpected DSN parameter %q", k)
					}
				}
			})
		}
	})

	t.Run("pragmas from DSN are really applied", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		dsn, err := s.manager().buildSQLiteDSN("a.db")
		if err != nil {
			t.Fatal(err)
		}

		raw, oerr := sql.Open("sqlite", dsn)
		if oerr != nil {
			t.Fatal(oerr)
		}
		t.Cleanup(func() { _ = raw.Close() })

		var trustedSchema, busyTimeout int
		e1 := raw.QueryRow("PRAGMA trusted_schema").Scan(&trustedSchema)
		e2 := raw.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)

		if e1 != nil || e2 != nil {
			t.Fatalf("pragma read: %v / %v", e1, e2)
		}
		if trustedSchema != 0 || busyTimeout != 5000 {
			t.Fatalf("trusted_schema=%d busy_timeout=%d", trustedSchema, busyTimeout)
		}
	})

	t.Run("file name with special chars survives the round trip", func(t *testing.T) {
		s := newSandbox(t)
		name := "my db & co?#%.db"
		createDB(t, s.in(name))
		m := s.manager()

		d := mustConnect(t, m, name)
		row, err := d.Query(context.Background(), "SELECT COUNT(*) AS c FROM items", nil)

		if err != nil || row["c"] != "2" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})
}

func TestResolveConnection(t *testing.T) {
	t.Run("maps every sqlite alias to the sandboxed driver", func(t *testing.T) {
		for _, writable := range []bool{false, true} {
			for _, alias := range []string{"sqlite3", "sqlite", "SQLITE3", " Sqlite ", SQLiteDriverName} {
				t.Run(fmt.Sprintf("%q/write=%v", alias, writable), func(t *testing.T) {
					s := newSandbox(t)
					createDB(t, s.in("a.db"))
					m := s.manager(WithWriteOperationsEnabled(writable))

					driver, dsn, err := m.resolveConnection(ExternalSQLStoreConnectConfig{Driver: alias, DSN: "a.db"})

					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if driver != SQLiteDriverName {
						t.Fatalf("driver = %q, want %q", driver, SQLiteDriverName)
					}
					if !strings.HasPrefix(dsn, "file:") {
						t.Fatalf("dsn must be built by service, got %q", dsn)
					}
				})
			}
		}
	})

	t.Run("passes non-sqlite config through untouched", func(t *testing.T) {
		m := NewExternalStoreManager()
		raw := "user:pass@tcp(127.0.0.1:3306)/db"

		driver, dsn, err := m.resolveConnection(ExternalSQLStoreConnectConfig{Driver: "mysql", DSN: raw})

		if err != nil || driver != "mysql" || dsn != raw {
			t.Fatalf("driver=%q dsn=%q err=%v", driver, dsn, err)
		}
	})

	t.Run("rejects a path outside the sandbox", func(t *testing.T) {
		s := newSandbox(t)
		m := s.manager()

		driver, dsn, err := m.resolveConnection(sqliteCfg("../outside/secret.db"))

		if err == nil {
			t.Fatalf("expected rejection, got driver=%q dsn=%q", driver, dsn)
		}
		if dsn != "" {
			t.Fatalf("dsn must be empty on error, got %q", dsn)
		}
	})

	t.Run("ro and rw produce different DSNs (so different cache keys)", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))

		_, roDSN, e1 := s.manager().resolveConnection(sqliteCfg("a.db"))
		_, rwDSN, e2 := s.manager(WithWriteOperationsEnabled(true)).resolveConnection(sqliteCfg("a.db"))

		if e1 != nil || e2 != nil {
			t.Fatalf("errors: %v / %v", e1, e2)
		}
		if roDSN == rwDSN {
			t.Fatalf("DSNs must differ, both = %q", roDSN)
		}
	})
}

func TestConnect(t *testing.T) {
	t.Run("read-only mode", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		d := mustConnect(t, s.manager(), "a.db")
		ctx := context.Background()

		t.Run("allows SELECT", func(t *testing.T) {
			row, err := d.Query(ctx, "SELECT id, name FROM items ORDER BY id", nil)

			if err != nil || row["id"] != "1" || row["name"] != "alpha" {
				t.Fatalf("row=%v err=%v", row, err)
			}
		})

		t.Run("rejects INSERT via Query", func(t *testing.T) {
			_, err := d.Query(ctx, "INSERT INTO items (id, name) VALUES (3, 'gamma')", nil)

			if err == nil {
				t.Fatal("INSERT must fail on a read-only connection")
			}
		})

		t.Run("rejects INSERT via Exec with readonly error", func(t *testing.T) {
			_, err := d.db.ExecContext(ctx, "INSERT INTO items (id, name) VALUES (3, 'gamma')")

			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "readonly") {
				t.Fatalf("want readonly error, got: %v", err)
			}
		})
	})

	t.Run("writable mode", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		d := mustConnect(t, s.manager(WithWriteOperationsEnabled(true)), "a.db")
		ctx := context.Background()

		inserted, err := d.Query(ctx, "INSERT INTO items (id, name) VALUES (3, 'gamma') RETURNING id", nil)
		count, cerr := d.Query(ctx, "SELECT COUNT(*) AS c FROM items", nil)

		if err != nil || inserted["id"] != "3" {
			t.Fatalf("insert: row=%v err=%v", inserted, err)
		}
		if cerr != nil || count["c"] != "3" {
			t.Fatalf("count: row=%v err=%v", count, cerr)
		}
	})

	t.Run("does not create a missing file even in rw mode", func(t *testing.T) {
		s := newSandbox(t)
		m := s.manager(WithWriteOperationsEnabled(true))

		d, err := m.Connect(context.Background(), sqliteCfg("new.db"))

		if err == nil {
			_ = d.db.Close()
			t.Fatal("must not connect to a non-existing file")
		}
		if _, serr := os.Stat(s.in("new.db")); serr == nil {
			t.Fatal("file must not be created")
		}
	})

	t.Run("is disabled without sandbox", func(t *testing.T) {
		m := NewExternalStoreManager()

		d, err := m.Connect(context.Background(), sqliteCfg("a.db"))

		if err == nil {
			_ = d.db.Close()
			t.Fatal("sqlite must be disabled when sandbox is not configured")
		}
	})

	t.Run("rejects bad input in both modes", func(t *testing.T) {
		s := newSandbox(t)
		secret := s.outsideOf("secret.db")
		createDB(t, secret)
		createDB(t, s.in("a.db"))

		inputs := []string{
			"", ":memory:", "file:a.db", "file:" + secret, "a.db?mode=rwc",
			"../outside/secret.db", secret, "/etc/passwd",
		}

		for _, writable := range []bool{false, true} {
			m := s.manager(WithWriteOperationsEnabled(writable))

			for _, in := range inputs {
				t.Run(fmt.Sprintf("%q/write=%v", in, writable), func(t *testing.T) {
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
	})

	t.Run("never uses a user-supplied DSN raw", func(t *testing.T) {
		s := newSandbox(t)
		m := s.manager(WithWriteOperationsEnabled(true))
		secret := s.outsideOf("secret.db")
		createDB(t, secret)

		for _, dsn := range []string{secret, "file:" + secret + "?mode=rw", ":memory:"} {
			t.Run(dsn, func(t *testing.T) {
				d, err := m.Connect(context.Background(), ExternalSQLStoreConnectConfig{
					Driver: SQLiteDriverName,
					DSN:    dsn,
				})

				if err == nil {
					_ = d.db.Close()
					t.Fatalf("DSN %q bypassed sandbox validation", dsn)
				}
			})
		}
	})
}

func TestConnect_Cache(t *testing.T) {
	t.Run("same config returns the same instance", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		m := s.manager()

		d1 := mustConnect(t, m, "a.db")
		d2 := mustConnect(t, m, "a.db")

		if d1 != d2 {
			t.Fatal("same config must hit the cache")
		}
	})

	t.Run("relative and absolute forms share an entry", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		m := s.manager()

		d1 := mustConnect(t, m, "a.db")
		d2 := mustConnect(t, m, s.in("a.db"))

		if d1 != d2 {
			t.Fatal("both forms must resolve to one cache entry")
		}
	})

	t.Run("concurrent calls create a single instance", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		m := s.manager()

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
	})
}

func TestSQLiteAuthorizer_Denies(t *testing.T) {
	s := newSandbox(t)
	createDB(t, s.in("a.db"))

	secret := s.outsideOf("secret.db")
	createDB(t, secret, `CREATE TABLE secrets (v TEXT); INSERT INTO secrets VALUES ('TOP')`)
	leak := s.outsideOf("leak.db")
	secretBefore := readFile(t, secret)

	tests := []struct {
		name   string
		sql    string
		params []any
	}{
		{"ATTACH", fmt.Sprintf("ATTACH DATABASE '%s' AS x", secret), nil},
		{"ATTACH lower case", fmt.Sprintf("attach '%s' as x", secret), nil},
		{"ATTACH after leading comment", fmt.Sprintf("/* c */ ATTACH '%s' AS x", secret), nil},
		{"ATTACH as second statement", fmt.Sprintf("SELECT 1; ATTACH '%s' AS x", secret), nil},
		{"ATTACH bound parameter", "ATTACH DATABASE ? AS x", []any{secret}},
		{"ATTACH URI form", fmt.Sprintf("ATTACH 'file:%s?mode=rw' AS x", secret), nil},
		{"ATTACH :memory:", "ATTACH ':memory:' AS x", nil},
		{"ATTACH temp db", "ATTACH '' AS x", nil},

		{"VACUUM INTO", fmt.Sprintf("VACUUM INTO '%s'", leak), nil},
		{"VACUUM main INTO", fmt.Sprintf("VACUUM main INTO '%s'", leak), nil},
		{"VACUUM INTO bound parameter", "VACUUM INTO ?", []any{leak}},
		{"VACUUM INTO as second statement", fmt.Sprintf("SELECT 1; VACUUM INTO '%s'", leak), nil},

		{"PRAGMA writable_schema", "PRAGMA writable_schema = 1", nil},
		{"PRAGMA query_only off", "PRAGMA query_only = 0", nil},
		{"PRAGMA read", "PRAGMA user_version", nil},

		{"CREATE TABLE", "CREATE TABLE t (id INTEGER)", nil},
		{"CREATE INDEX", "CREATE INDEX idx ON items(name)", nil},
		{"DROP TABLE", "DROP TABLE items", nil},
		{"ALTER TABLE", "ALTER TABLE items ADD COLUMN extra TEXT", nil},
		{"BEGIN", "BEGIN", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			forEachMode(t, s, "a.db", func(t *testing.T, d *ExternalDb) {
				_, err := d.Query(context.Background(), tc.sql, tc.params)
				_, attachedErr := d.Query(context.Background(), "SELECT v FROM x.secrets", nil)

				if err == nil {
					t.Fatalf("%q must be denied", tc.sql)
				}
				if attachedErr == nil {
					t.Fatal("attached schema must not be reachable")
				}
			})
		})
	}

	if _, err := os.Stat(leak); !os.IsNotExist(err) {
		t.Fatalf("VACUUM INTO created a file outside the sandbox (stat err=%v)", err)
	}
	if readFile(t, secret) != secretBefore {
		t.Fatal("outside file was modified")
	}
}

func TestSQLiteAuthorizer_Allows(t *testing.T) {
	t.Run("read queries in both modes", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))

		tests := []struct{ name, sql, col, want string }{
			{"plain select", "SELECT name AS v FROM items WHERE id = 1", "v", "alpha"},
			{"scalar function", "SELECT upper(name) AS v FROM items WHERE id = 1", "v", "ALPHA"},
			{"subquery", "SELECT (SELECT COUNT(*) FROM items) AS v", "v", "2"},
			{"recursive CTE", "WITH RECURSIVE c(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM c WHERE n < 5) SELECT MAX(n) AS v FROM c", "v", "5"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				forEachMode(t, s, "a.db", func(t *testing.T, d *ExternalDb) {
					row, err := d.Query(context.Background(), tc.sql, nil)

					if err != nil || row[tc.col] != tc.want {
						t.Fatalf("row=%v err=%v", row, err)
					}
				})
			})
		}
	})

	t.Run("INSERT, UPDATE, DELETE in rw mode", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))
		d := mustConnect(t, s.manager(WithWriteOperationsEnabled(true)), "a.db")
		ctx := context.Background()

		_, insErr := d.Query(ctx, "INSERT INTO items (id, name) VALUES (3, 'gamma') RETURNING id", nil)
		_, updErr := d.Query(ctx, "UPDATE items SET name = 'delta' WHERE id = 3 RETURNING id", nil)
		afterUpdate, _ := d.Query(ctx, "SELECT name FROM items WHERE id = 3", nil)
		_, delErr := d.Query(ctx, "DELETE FROM items WHERE id = 3 RETURNING id", nil)
		afterDelete, _ := d.Query(ctx, "SELECT name FROM items WHERE id = 3", nil)

		if insErr != nil || updErr != nil || delErr != nil {
			t.Fatalf("insert=%v update=%v delete=%v", insErr, updErr, delErr)
		}
		if afterUpdate["name"] != "delta" {
			t.Fatalf("update not applied: %v", afterUpdate)
		}
		if len(afterDelete) != 0 {
			t.Fatalf("delete not applied: %v", afterDelete)
		}
	})
}

func TestSQLite_UseDefensivePragma(t *testing.T) {
	t.Run("defensive pragma blocks direct sqlite_schema modifications", func(t *testing.T) {
		s := newSandbox(t)
		createDB(t, s.in("a.db"))

		d := mustConnect(t, s.manager(WithWriteOperationsEnabled(true)), "a.db")
		ctx := context.Background()

		_, err := d.Query(ctx, "UPDATE sqlite_dbpage SET data = 'test' WHERE pgno = 1;", nil)

		if err == nil {
			t.Fatal("SECURITY VULNERABILITY: PRAGMA defensive is disabled or bypassed! Direct UPDATE to sqlite_schema succeeded.")
		}

		if !strings.EqualFold("SQL logic error: read-only (1)", err.DetailedError) {
			t.Fatalf("SECURITY VULNERABILITY: PRAGMA defensive is disabled or bypassed! Direct UPDATE to sqlite_schema succeeded. Err: %+v", err)
		}
	})
}

func TestSQLite_LoadExtensionIsUnavailable(t *testing.T) {
	s := newSandbox(t)
	createDB(t, s.in("a.db"))
	d := mustConnect(t, s.manager(), "a.db")

	_, err := d.Query(context.Background(), "SELECT load_extension('/nonexistent/lib.so')", nil)

	if err == nil {
		t.Fatal("load_extension must not be available")
	}
}

func TestQuery(t *testing.T) {
	s := newSandbox(t)
	createDB(t, s.in("a.db"))
	d := mustConnect(t, s.manager(), "a.db")
	ctx := context.Background()

	t.Run("returns empty non-nil map when there are no rows", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT id FROM items WHERE id = 999", nil)
		if err != nil || row == nil || len(row) != 0 {
			t.Fatalf("row=%#v err=%v", row, err)
		}
	})

	t.Run("returns only the first row", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT name FROM items ORDER BY id", nil)

		if err != nil || row["name"] != "alpha" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("binds parameters", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT name FROM items WHERE id = ?", []any{2})

		if err != nil || row["name"] != "beta" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("treats injection in a parameter as plain data", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT name FROM items WHERE name = ?", []any{"x' OR '1'='1"})

		if err != nil || len(row) != 0 {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("converts every value to string", func(t *testing.T) {
		tests := []struct{ name, sql, want string }{
			{"integer", "SELECT 42 AS v", "42"},
			{"real", "SELECT 1.5 AS v", "1.5"},
			{"text", "SELECT 'hi' AS v", "hi"},
			{"blob", "SELECT x'414243' AS v", "ABC"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				row, err := d.Query(ctx, tc.sql, nil)

				if err != nil || row["v"] != tc.want {
					t.Fatalf("row=%v err=%v", row, err)
				}
			})
		}
	})

	t.Run("NULL is currently represented as \"<nil>\"", func(t *testing.T) {
		row, err := d.Query(ctx, "SELECT NULL AS n", nil)
		if err != nil || row["n"] != "<nil>" {
			t.Fatalf("row=%v err=%v", row, err)
		}
	})

	t.Run("returns error for invalid SQL", func(t *testing.T) {
		_, err := d.Query(ctx, "SELEC nonsense", nil)

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns error for canceled context", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := d.Query(cctx, "SELECT 1", nil)

		if err == nil {
			t.Fatal("expected error on canceled context")
		}
	})
}
