package email

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
)

const (
	MailGmail   = "gmail"
	MailOutlook = "outlook"
	MailCidKey  = "__cid"

	MailAuthTypeOAuth2 = "oauth2"
)

type OAuth2Config struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

type MailParams struct {
	OAuth2   *OAuth2Config `json:"oauth2"`
	Insecure bool          `json:"insecure"`
}

type EmailCid int64

type Email struct {
	Id          int64            `json:"id" db:"id"`
	Direction   string           `json:"direction" db:"direction"`
	MessageId   string           `json:"message_id" db:"message_id"`
	Subject     string           `json:"subject" db:"subject"`
	ProfileId   int              `json:"profile_id" db:"profile_id"`
	From        []string         `json:"from" db:"from"`
	To          []string         `json:"to" db:"to"`
	Sender      []string         `json:"sender" db:"sender"`
	ReplyTo     []string         `json:"reply_to" db:"reply_to"`
	InReplyTo   string           `json:"in_reply_to" db:"in_reply_to"`
	CC          []string         `json:"cc" db:"cc"`
	Body        []byte           `json:"body" db:"body"`
	HtmlBody    []byte           `json:"html_body" db:"html_body"`
	AttemptId   *int64           `json:"attempt_id" db:"attempt_id"`
	ContactIds  []int64          `json:"contactIds" db:"contact_ids"`
	OwnerId     *int64           `json:"ownerId" db:"owner_id"`
	Attachments []files.File
	Cid         map[string]EmailCid
}

func (e *Email) CIDJson() *[]byte {
	if len(e.Cid) == 0 {
		return nil
	}
	j, _ := json.Marshal(e.Cid)
	return &j
}

func (e *Email) AttachmentIds() []int64 {
	l := len(e.Attachments)
	if l == 0 {
		return nil
	}
	ids := make([]int64, 0, l)
	for _, v := range e.Attachments {
		ids = append(ids, int64(v.Id))
	}
	return ids
}

type EmailAction struct {
	FlowId    int   `json:"flow_id"`
	AttemptId int64 `json:"attempt_id"`
}

// EmailConnection is a flow Connection specialised for email channels.
type EmailConnection interface {
	flow.Connection
	SchemaId() int
	Reply(text string) (*Email, error)
	Email() *Email
}

type ReplyEmail struct{}

type EmailProfileTask struct {
	Id        int   `json:"id" db:"id"`
	UpdatedAt int64 `json:"updated_at" db:"updated_at"`
}

type EmailProfile struct {
	Id        int           `json:"id" db:"id"`
	DomainId  int64         `json:"domain_id" db:"domain_id"`
	Name      string        `json:"name" db:"name"`
	FlowId    int           `json:"flow_id" db:"flow_id"`
	Login     string        `json:"login" db:"login"`
	Password  string        `json:"password" db:"password"`
	Mailbox   string        `json:"mailbox" db:"mailbox"`
	SmtpHost  string        `json:"smtp_host" db:"smtp_host"`
	SmtpPort  int           `json:"smtp_port" db:"smtp_port"`
	ImapHost  string        `json:"imap_host" db:"imap_host"`
	ImapPort  int           `json:"imap_port" db:"imap_port"`
	UpdatedAt int64         `json:"updated_at" db:"updated_at"`
	Params    *MailParams   `json:"params" db:"params"`
	Token     *oauth2.Token `json:"token" db:"token"`
	AuthType  string        `json:"auth_type" db:"auth_type"`
}

func (p *EmailProfile) OAuthConfig() oauth2.Config {
	if p.Params != nil && p.Params.OAuth2 != nil {
		return OAuthConfig(p.ImapHost, p.Params.OAuth2)
	}
	return oauth2.Config{}
}

func (p *EmailProfile) Tls() bool {
	return p.Params != nil && p.Params.Insecure
}

// OAuthConfig builds an oauth2.Config for Gmail or Office365 from a profile.
func OAuthConfig(host string, cfg *OAuth2Config) oauth2.Config {
	if strings.Contains(host, MailGmail+".com") {
		return oauth2.Config{
			ClientID:     cfg.ClientId,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://www.googleapis.com/oauth2/v4/token",
			},
			RedirectURL: cfg.RedirectURL,
			Scopes:      []string{"https://mail.google.com/"},
		}
	} else if strings.Contains(host, "office365") && cfg != nil {
		return oauth2.Config{
			ClientID:     cfg.ClientId,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/organizations/oauth2/v2.0/token",
			},
			RedirectURL: cfg.RedirectURL,
			Scopes: []string{
				"https://outlook.office.com/User.Read",
				"https://outlook.office.com/IMAP.AccessAsUser.All",
				"https://outlook.office.com/SMTP.Send",
				"offline_access",
			},
		}
	}
	return oauth2.Config{}
}

// ── SMTP ──────────────────────────────────────────────────────────────────────

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
}

// ── mail ID generation ────────────────────────────────────────────────────────

var maxBigInt = big.NewInt(math.MaxInt64)

func GenerateMailID() (string, error) {
	t := time.Now().UnixNano()
	pid := os.Getpid()
	rint, err := rand.Int(rand.Reader, maxBigInt)
	if err != nil {
		return "", err
	}
	h, err := os.Hostname()
	if err != nil {
		h = "localhost.localdomain"
	}
	return fmt.Sprintf("<%d.%d.%d@%s>", t, pid, rint, h), nil
}
