package call

import (
	"testing"

	"github.com/webitel/flow_manager/model"
)

func TestSchemaRecordsSession(t *testing.T) {
	cases := []struct {
		name   string
		schema *model.Schema
		want   bool
	}{
		{
			name:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			name:   "no recording",
			schema: &model.Schema{Schema: model.Applications{{"answer": ""}, {"bridge": map[string]any{}}}},
			want:   false,
		},
		{
			name:   "top level recordSession",
			schema: &model.Schema{Schema: model.Applications{{"recordSession": map[string]any{"minSec": 4}}}},
			want:   true,
		},
		{
			name: "nested recordSession",
			schema: &model.Schema{Schema: model.Applications{{"if": map[string]any{
				"then": []any{map[string]any{"recordSession": map[string]any{"minSec": 4}}},
			}}}},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemaRecordsSession(tc.schema); got != tc.want {
				t.Fatalf("schemaRecordsSession() = %v, want %v", got, tc.want)
			}
		})
	}
}
