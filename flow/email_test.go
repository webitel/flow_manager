package flow

import (
	"testing"

	"golang.org/x/oauth2"

	"github.com/webitel/flow_manager/model"
)

func TestSanitizeInlineSmtp(t *testing.T) {
	t.Run("plain keeps connection settings and drops tenant-bound fields", func(t *testing.T) {
		smtp := model.SmtSettings{
			Id:       42,
			AuthType: model.MailAuthTypePlain,
			Auth:     model.SmtpPlainAuth{User: "user", Password: "pass"},
			Server:   "smtp.example.com",
			Port:     587,
			Tls:      true,
			Params:   &model.MailParams{OAuth2: &model.OAuth2Config{ClientId: "id"}},
			Token:    &oauth2.Token{AccessToken: "token"},
		}

		if err := sanitizeInlineSmtp(&smtp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if smtp.Id != 0 || smtp.Params != nil || smtp.Token != nil {
			t.Fatalf("tenant-bound fields not cleared: %+v", smtp)
		}
		if smtp.Server != "smtp.example.com" || smtp.Port != 587 || !smtp.Tls ||
			smtp.Auth.User != "user" || smtp.Auth.Password != "pass" {
			t.Fatalf("connection settings changed: %+v", smtp)
		}
	})

	t.Run("empty auth type treated as plain", func(t *testing.T) {
		smtp := model.SmtSettings{Id: 7}
		if err := sanitizeInlineSmtp(&smtp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if smtp.Id != 0 {
			t.Fatalf("id not cleared: %d", smtp.Id)
		}
	})

	t.Run("oauth2 without profile rejected", func(t *testing.T) {
		smtp := model.SmtSettings{Id: 7, AuthType: model.MailAuthTypeOAuth2}
		if err := sanitizeInlineSmtp(&smtp); err == nil {
			t.Fatal("expected error for inline oauth2")
		}
	})
}
