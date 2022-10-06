package model

type Email struct {
	Id          int64    `json:"id" db:"id"`
	Direction   string   `json:"direction" db:"direction" `
	MessageId   string   `json:"message_id" db:"message_id"`
	Subject     string   `json:"subject" db:"subject"`
	ProfileId   int      `json:"profile_id" db:"profile_id"`
	From        []string `json:"from" db:"from"`
	To          []string `json:"to" db:"to"`
	Sender      []string `json:"sender" db:"sender"`
	ReplyTo     []string `json:"reply_to" db:"reply_to"`
	InReplyTo   string   `json:"in_reply_to" db:"in_reply_to"`
	CC          []string `json:"cc" db:"cc"`
	Body        []byte   `json:"body" db:"body"`
	HtmlBody    []byte   `json:"html_body" db:"html_body"`
	AttemptId   *int64   `json:"attempt_id" db:"attempt_id"`
	Attachments []File
}

type EmailAction struct {
	FlowId    int   `json:"flow_id"`
	AttemptId int64 `json:"attempt_id"`
}

type EmailConnection interface {
	Connection
	SchemaId() int
	Reply(text string) (*Email, *AppError)
	Email() *Email
}

type ReplyEmail struct {
}

type EmailProfileTask struct {
	Id        int   `json:"id" db:"id"`
	UpdatedAt int64 `json:"updated_at" db:"updated_at"`
}

type EmailProfile struct {
	Id        int    `json:"id" db:"id"`
	DomainId  int64  `json:"domain_id" db:"domain_id"`
	Name      string `json:"name" db:"name"`
	FlowId    int    `json:"flow_id" db:"flow_id"`
	Login     string `json:"login" db:"login"`
	Password  string `json:"password" db:"password"`
	Mailbox   string `json:"mailbox" db:"mailbox"`
	SmtpHost  string `json:"smtp_host" db:"smtp_host"`
	SmtpPort  int    `json:"smtp_port" db:"smtp_port"`
	ImapHost  string `json:"imap_host" db:"imap_host"`
	ImapPort  int    `json:"imap_port" db:"imap_port"`
	UpdatedAt int64  `json:"updated_at" db:"updated_at"`
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
