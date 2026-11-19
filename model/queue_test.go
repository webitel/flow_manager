package model_test

import (
	"testing"

	"github.com/webitel/flow_manager/model"
)

func TestSearchEntityIsEmpty(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	tests := []struct {
		name   string
		entity *model.SearchEntity
		want   bool
	}{
		{"nil", nil, true},
		{"no fields", &model.SearchEntity{}, true},
		{"zero id", &model.SearchEntity{Id: intPtr(0)}, true},
		{"negative id", &model.SearchEntity{Id: intPtr(-1)}, true},
		{"empty name", &model.SearchEntity{Name: strPtr("")}, true},
		{"zero id and empty name", &model.SearchEntity{Id: intPtr(0), Name: strPtr("")}, true},
		{"positive id", &model.SearchEntity{Id: intPtr(5)}, false},
		{"name", &model.SearchEntity{Name: strPtr("support")}, false},
		{"zero id with name", &model.SearchEntity{Id: intPtr(0), Name: strPtr("support")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entity.IsEmpty(); got != tt.want {
				t.Fatalf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
