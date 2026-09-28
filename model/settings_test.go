package model_test

import (
	"testing"

	"github.com/webitel/flow_manager/model"
)

func TestNewSystemSettingEventFromRoutingKey(t *testing.T) {
	tests := []struct {
		name    string
		rk      string
		want    *model.SystemSettingEvent
		wantErr bool
	}{
		{"update", "system_settings.autolink_call_to_contact.update.1.10", &model.SystemSettingEvent{Name: "autolink_call_to_contact", DomainID: 1}, false},
		{"delete", "system_settings.chat_ai_connection.delete.25.3", &model.SystemSettingEvent{Name: "chat_ai_connection", DomainID: 25}, false},
		{"without user", "system_settings.autolink_mail_to_contact.create.7", &model.SystemSettingEvent{Name: "autolink_mail_to_contact", DomainID: 7}, false},
		{"too short", "system_settings.autolink_call_to_contact.update", nil, true},
		{"wrong object", "domains.autolink_call_to_contact.update.1.1", nil, true},
		{"empty name", "system_settings..update.1.1", nil, true},
		{"domain not a number", "system_settings.autolink_call_to_contact.update.abc.1", nil, true},
		{"domain zero", "system_settings.autolink_call_to_contact.update.0.1", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := model.NewSystemSettingEventFromRoutingKey(tc.rk)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if *got != *tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestSystemSettingCacheKey(t *testing.T) {
	if got := model.SystemSettingCacheKey(12, "autolink_call_to_contact"); got != "12-autolink_call_to_contact" {
		t.Fatalf("got %q", got)
	}
}
