package model

import (
	"golang.org/x/oauth2"

	"github.com/webitel/flow_manager/internal/domain/email"
)

// Re-exports for backward compatibility.
type OAuth2Config = email.OAuth2Config
type MailParams = email.MailParams
type EmailCid = email.EmailCid
type Email = email.Email
type EmailAction = email.EmailAction
type EmailConnection = email.EmailConnection
type ReplyEmail = email.ReplyEmail
type EmailProfileTask = email.EmailProfileTask
type EmailProfile = email.EmailProfile

const (
	MailGmail          = email.MailGmail
	MailOutlook        = email.MailOutlook
	MailCidKey         = email.MailCidKey
	MailAuthTypeOAuth2 = email.MailAuthTypeOAuth2
)

func OAuthConfig(host string, cfg *OAuth2Config) oauth2.Config {
	return email.OAuthConfig(host, cfg)
}

func GenerateMailID() (string, error) {
	return email.GenerateMailID()
}
