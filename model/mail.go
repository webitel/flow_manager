package model

import "golang.org/x/oauth2"

type SmtpPlainAuth struct {
	Password string `json:"password" db:"password"`
	User     string `json:"user" db:"login"`
}

type SmtpParams struct {
	OAuth2 *oauth2.Token
}

type SmtSettings struct {
	Id       int           `json:"id" db:"id"`
	AuthType string        `json:"authType" db:"auth_type"`
	Auth     SmtpPlainAuth `json:"auth"`
	Port     int           `json:"port" db:"port"`
	Server   string        `json:"server" db:"server"`
	Tls      bool          `json:"tls" db:"tls"`
	Params   *MailParams   `json:"params" db:"params"`
	Token    *oauth2.Token `json:"token" db:"token"`
	//Insecure bool   `json:"insecure"`
}
