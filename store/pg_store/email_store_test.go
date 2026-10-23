package sqlstore

import (
	"context"
	"reflect"
	"testing"

	"github.com/webitel/flow_manager/model"
)

func TestEmailLastBridgedEmptyAddress(t *testing.T) {
	s := &SqlEmailStore{}
	for _, address := range []string{"", "  ", "invalid", "name <a@example.com>"} {
		got, err := s.LastBridged(context.Background(), 1, address, "24", nil,
			model.Variables{"owner": "agent_id", "ext": "extension", "queue": "queue_id"})
		want := model.Variables{"owner": "", "ext": "", "queue": ""}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("address %q: got %v, err %v", address, got, err)
		}
	}
}

func TestEmailLastBridgedInvalidField(t *testing.T) {
	s := &SqlEmailStore{}
	for _, field := range []any{"variables.agent_id", "agent_id; drop table x", 1} {
		_, err := s.LastBridged(context.Background(), 1, "a@example.com", "24", nil, model.Variables{"owner": field})
		if err == nil || err.Id != "store.sql_email.get_last_bridged.invalid_field" || err.StatusCode != 400 {
			t.Fatalf("field %v: err %v", field, err)
		}
	}
}
