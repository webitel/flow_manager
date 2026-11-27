package sql

import (
	"errors"
	"testing"
)

func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, want string
		err        Error
	}{
		{
			name: "Without parameter name (empty param)",
			err:  Error{err: ErrParameterRequired, param: ""},
			want: "sql: parameter required",
		},
		{
			name: "With parameter name (quoted via %q)",
			err:  Error{err: ErrParameterRequired, param: "user_id"},
			want: `sql: "user_id" parameter required`,
		},
		{
			name: "With another base error and parameter name",
			err:  Error{err: errors.New("must be positive"), param: "limit"},
			want: `sql: "limit" must be positive`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.err.Error(); tt.want != err {
				t.Errorf("Error() want: %q got: %q", tt.want, err)
			}
		})
	}
}
