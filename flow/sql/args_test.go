package sql

import (
	"errors"
	"reflect"
	"testing"
)

func TestArgs_Normalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *Args
		expected *Args
	}{{
		name:     "Nil pointer does not panic",
		input:    nil,
		expected: nil,
	}, {
		name: "Driver trimmed and lowercased, valid positive timeout unchanged",
		input: &Args{
			Driver:  "  PostgreSQL  ",
			DSN:     "postgres://user:pass@localhost:5432/db",
			Query:   "SELECT 1",
			Params:  []any{123},
			Timeout: 10,
		},
		expected: &Args{
			Driver:  "postgresql",
			DSN:     "postgres://user:pass@localhost:5432/db",
			Query:   "SELECT 1",
			Params:  []any{123},
			Timeout: 10,
		},
	}, {
		name: "Negative timeout replaced with defaultExternalSQLTimeout",
		input: &Args{
			Driver:  "MySQL",
			Timeout: -5,
		},
		expected: &Args{
			Driver:  "mysql",
			Timeout: defaultExternalSQLTimeout,
		},
	}, {
		name: "Zero timeout remains zero",
		input: &Args{
			Driver:  "sqlite3",
			Timeout: 0,
		},
		expected: &Args{
			Driver:  "sqlite3",
			Timeout: 0,
		},
	}, {
		name: "Driver already normalized, valid timeout",
		input: &Args{
			Driver:  "clickhouse",
			Timeout: 30,
		},
		expected: &Args{
			Driver:  "clickhouse",
			Timeout: 30,
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.input.Normalize()

			if tt.input == nil {
				return
			}

			if tt.input.Driver != tt.expected.Driver {
				t.Errorf("Driver = %q, want %q", tt.input.Driver, tt.expected.Driver)
			}

			if tt.input.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %d, want %d", tt.input.Timeout, tt.expected.Timeout)
			}

			if tt.input.DSN != tt.expected.DSN {
				t.Errorf("DSN unexpectedly modified: got %q, want %q", tt.input.DSN, tt.expected.DSN)
			}
			if tt.input.Query != tt.expected.Query {
				t.Errorf("Query unexpectedly modified: got %q, want %q", tt.input.Query, tt.expected.Query)
			}
			if !reflect.DeepEqual(tt.input.Params, tt.expected.Params) {
				t.Errorf("Params unexpectedly modified: got %v, want %v", tt.input.Params, tt.expected.Params)
			}
		})
	}
}

func TestArgs_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *Args
		wantErr error
	}{{
		name:    "Error: nil pointer receiver returns required args error",
		input:   nil,
		wantErr: ErrParameterRequired,
	}, {
		name: "Error: empty query string returns required query error",
		input: &Args{
			Driver: Postgres,
			DSN:    "postgres://localhost",
			Query:  "",
		},
		wantErr: ErrParameterRequired,
	}, {
		name: "Error: whitespace-only query string returns required query error",
		input: &Args{
			Driver: Postgres,
			DSN:    "postgres://localhost",
			Query:  "   \t  ",
		},
		wantErr: ErrParameterRequired,
	}, {
		name: "Error: empty driver returns required driver error",
		input: &Args{
			Driver: "",
			DSN:    "postgres://localhost",
			Query:  "SELECT 1",
		},
		wantErr: ErrParameterRequired,
	}, {
		name: "Error: unsupported driver returns ErrUnsupportedDriver",
		input: &Args{
			Driver: "oracle",
			DSN:    "postgres://localhost",
			Query:  "SELECT 1",
		},
		wantErr: ErrUnsupportedDriver,
	}, {
		name: "Error: empty DSN returns required dsn error",
		input: &Args{
			Driver: Postgres,
			DSN:    "",
			Query:  "SELECT 1",
		},
		wantErr: ErrParameterRequired,
	}, {
		name: "Error: whitespace-only DSN returns required dsn error",
		input: &Args{
			Driver: Postgres,
			DSN:    "   ",
			Query:  "SELECT 1",
		},
		wantErr: ErrParameterRequired,
	}, {
		name: "Success: valid sqlite3 configuration",
		input: &Args{
			Driver: SQLite3,
			DSN:    "file:test.db",
			Query:  "SELECT 1",
		},
		wantErr: nil,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() expected error wrapping %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, expected error to unwrap to %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %+v", err)
				}
			}
		})
	}
}

func TestArgs_Prepare(t *testing.T) {
	t.Parallel()

	mockLookup := func(name string) string {
		vars := map[string]string{
			"driver_val": "postgres",
			"host":       "localhost",
			"port":       "5432",
			"dbname":     "test_db",
		}
		return vars[name]
	}

	tests := []struct {
		name           string
		initial        Args
		lookup         globalVarLookup
		expected       Args
		wantErr        error
		wantErrorParam string
	}{{
		name: "Success: no global variables anywhere, values remain untouched",
		initial: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		wantErr: nil,
	}, {
		name: "Success: global variable in Driver only",
		initial: Args{
			Driver: "$${driver_val}",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		wantErr: nil,
	}, {
		name: "Success: global variables in DSN only",
		initial: Args{
			Driver: "postgres",
			DSN:    "postgres://$${host}:$${port}/$${dbname}",
			Query:  "SELECT 1",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/test_db",
			Query:  "SELECT 1",
		},
		wantErr: nil,
	}, {
		name: "Success: global variables in both Driver and DSN",
		initial: Args{
			Driver: "$${driver_val}",
			DSN:    "postgres://$${host}:$${port}/db",
			Query:  "SELECT 1",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		wantErr: nil,
	}, {
		name: "Error: Query contains global variables triggers ErrQueryContainsVars",
		initial: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT * FROM users WHERE id = $${user_id}",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT * FROM users WHERE id = $${user_id}",
		},
		wantErr: ErrQueryContainsVars,
	}, {
		name: "Error: Driver resolution fails with ErrOnlyGlobalVars, wraps with param 'driver'",
		initial: Args{
			Driver: "${driver_val}",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "${driver_val}",
			DSN:    "postgres://localhost:5432/db",
			Query:  "SELECT 1",
		},
		wantErr:        ErrOnlyGlobalVars,
		wantErrorParam: "driver",
	}, {
		name: "Error: DSN resolution fails with ErrOnlyGlobalVars, wraps with param 'dsn'",
		initial: Args{
			Driver: "postgres",
			DSN:    "${local_dsn}",
			Query:  "select 1 from users;",
		},
		lookup: mockLookup,
		expected: Args{
			Driver: "postgres",
			DSN:    "${local_dsn}",
			Query:  "select 1 from users;",
		},
		wantErr:        ErrOnlyGlobalVars,
		wantErrorParam: "dsn",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := tt.initial
			err := args.Prepare(tt.lookup)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Prepare() expected error wrapping %v, got nil", tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Prepare() error = %v, expected to unwrap to %v", err, tt.wantErr)
				}

				if tt.wantErrorParam != "" {
					var sqlErr *Error
					if errors.As(err, &sqlErr) {
						if sqlErr.param != tt.wantErrorParam {
							t.Errorf("Error.param = %q, want %q", sqlErr.param, tt.wantErrorParam)
						}
					} else {
						t.Errorf("expected error to be of type *Error, got %T", err)
					}
				}

			} else {
				if err != nil {
					t.Fatalf("Prepare() unexpected error: %+v", err)
				}

				if args.Driver != tt.expected.Driver {
					t.Errorf("Driver = %q, want %q", args.Driver, tt.expected.Driver)
				}
				if args.DSN != tt.expected.DSN {
					t.Errorf("DSN = %q, want %q", args.DSN, tt.expected.DSN)
				}
				if args.Query != tt.expected.Query {
					t.Errorf("Query = %q, want %q", args.Query, tt.expected.Query)
				}
			}
		})
	}
}
