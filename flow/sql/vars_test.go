package sql

import (
	"errors"
	"testing"
)

func TestFormatGlobalVarName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input, want string
	}{{
		name:  "Standard variable without spaces",
		input: "$${var_name}",
		want:  "var_name",
	}, {
		name:  "Variable with leading and trailing spaces",
		input: "$${  my_variable  }",
		want:  "my_variable",
	}, {
		name:  "Variable with tabs and spaces",
		input: "$${	tabbed_var	}",
		want:  "tabbed_var",
	}, {
		name:  "Variable with complex name",
		input: "$${DB_HOST_123}",
		want:  "DB_HOST_123",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := formatGlobalVarName(tt.input); got != tt.want {
				t.Errorf("formatGlobalVarName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveGlobalVars(t *testing.T) {
	t.Parallel()

	mockLookup := func(name string) string {
		vars := map[string]string{
			"host": "localhost",
			"port": "5432",
		}
		return vars[name]
	}

	tests := []struct {
		name, raw, want string
		lookup          globalVarLookup
		wantErr         error
	}{{
		name:    "Success: single global variable replacement",
		raw:     "Server running at $${host}",
		lookup:  mockLookup,
		want:    "Server running at localhost",
		wantErr: nil,
	}, {
		name:    "Success: multiple global variables replacement",
		raw:     "Connection to $${host}:$${port}",
		lookup:  mockLookup,
		want:    "Connection to localhost:5432",
		wantErr: nil,
	}, {
		name:    "Success: string without global variables remains unchanged",
		raw:     "Static text without vars",
		lookup:  mockLookup,
		want:    "Static text without vars",
		wantErr: nil,
	}, {
		name:    "Error: triggers ErrOnlyGlobalVars when validation matches",
		raw:     "${host}",
		lookup:  mockLookup,
		want:    "",
		wantErr: ErrOnlyGlobalVars,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveGlobalVars(tt.raw, tt.lookup)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("resolveGlobalVars() expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("resolveGlobalVars() error = %v, expected %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("resolveGlobalVars() unexpected error: %+v", err)
				}
				if got != tt.want {
					t.Errorf("resolveGlobalVars() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}
