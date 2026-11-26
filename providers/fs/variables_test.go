package fs

import (
	"slices"
	"testing"
)

func TestResolveExportVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  []string
		input map[string]any
	}{
		{
			name:  "nil_map",
			want:  nil,
			input: nil,
		},
		{
			name:  "empty_map",
			want:  nil,
			input: map[string]any{},
		},
		{
			name:  "no_usr_vars",
			want:  nil,
			input: map[string]any{"variable_wbt_contact_id": "10", "variable_sip_X-Webitel-Header": "test"},
		},
		{
			name: "with_usr_vars",
			want: []string{"test", "IPN"},
			input: map[string]any{
				"variable_wbt_contact_id":       "10",
				"variable_sip_X-Webitel-Header": "test",
				"variable_usr_test":             "test",
				"variable_usr_IPN":              "12671267",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveExportVariables(tt.input)
			slices.Sort(got)
			slices.Sort(tt.want)

			if !slices.Equal(got, tt.want) {
				t.Errorf("resolveExportVariables: got %#v want %#v", got, tt.want)
			}
		})
	}
}
