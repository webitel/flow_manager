package postgres

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	pgsql "github.com/webitel/flow_manager/infra/sql/pgsql"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type EmailRepository struct {
	db infraSql.Store
}

func NewEmailRepository(db infraSql.Store) store.EmailStore {
	return &EmailRepository{db: db}
}

const profileTaskFetchSQL = `update call_center.cc_email_profile
set last_activity_at = now(),
    state            = 'active'
where id in (select id
             from call_center.cc_email_profile
             where ((enabled and "listen"))
               and last_activity_at < now() - (fetch_interval || ' sec')::interval
             order by last_activity_at nulls first
             limit 100 for update skip locked)
returning id, (extract(EPOCH from updated_at) * 1000)::int8 updated_at`

func (r *EmailRepository) ProfileTaskFetch(node string) ([]*model.EmailProfileTask, error) {
	var tasks []*model.EmailProfileTask
	if err := r.db.Select(context.Background(), &tasks, profileTaskFetchSQL, pgx.NamedArgs{}); err != nil {
		return nil, err
	}
	return tasks, nil
}

const saveEmailSQL = `insert into call_center.cc_email ("from", "to", profile_id, subject, cc, body, direction, message_id, sender, reply_to,
                      in_reply_to, parent_id, html, attachment_ids, contact_ids, owner_id, cid)
values (@From, @To, @ProfileId, @Subject, @Cc, @Body::text, @Direction, @MessageId, @Sender, @ReplyTo, @InReplyTo,
        (select m.id from call_center.cc_email m where m.in_reply_to = @MessageId limit 1),
        @Html::text, @AttachmentIds::int8[], @ContactIds::int8[], @OwnerId::int8, @Cid::jsonb)
returning id`

type emailIdRow struct {
	Id int64 `db:"id"`
}

func (r *EmailRepository) Save(domainId int64, m *model.Email) error {
	var cidJson []byte
	if v := m.CIDJson(); v != nil {
		cidJson = *v
	}

	var row emailIdRow
	if err := r.db.Get(context.Background(), &row, saveEmailSQL, pgx.NamedArgs{
		"From":          m.From,
		"To":            m.To,
		"ProfileId":     m.ProfileId,
		"Subject":       m.Subject,
		"Cc":            m.CC,
		"Body":          m.Body,
		"Direction":     m.Direction,
		"MessageId":     m.MessageId,
		"Sender":        m.Sender,
		"ReplyTo":       m.ReplyTo,
		"InReplyTo":     m.InReplyTo,
		"Html":          m.HtmlBody,
		"AttachmentIds": m.AttachmentIds(),
		"ContactIds":    m.ContactIds,
		"OwnerId":       m.OwnerId,
		"Cid":           cidJson,
	}); err != nil {
		return err
	}
	m.Id = row.Id
	return nil
}

type emailProfileRow struct {
	Id        int             `db:"id"`
	DomainId  int64           `db:"domain_id"`
	Name      string          `db:"name"`
	FlowId    int             `db:"flow_id"`
	Login     string          `db:"login"`
	Password  string          `db:"password"`
	Mailbox   string          `db:"mailbox"`
	SmtpHost  string          `db:"smtp_host"`
	SmtpPort  int             `db:"smtp_port"`
	ImapHost  string          `db:"imap_host"`
	ImapPort  int             `db:"imap_port"`
	UpdatedAt int64           `db:"updated_at"`
	AuthType  string          `db:"auth_type"`
	Params    json.RawMessage `db:"params"`
	Token     json.RawMessage `db:"token"`
}

const getProfileSQL = `select t.id, t.name, t.login, t.password, t.mailbox,
       coalesce(t.imap_host, '') as imap_host,
       t.imap_port,
       coalesce(t.smtp_host, '') as smtp_host,
       t.smtp_port,
       (extract(EPOCH from t.updated_at) * 1000)::int8 updated_at,
       t.flow_id,
       t.domain_id,
       coalesce(t.auth_type, 'plain') as auth_type,
       t.params,
       t.token
from call_center.cc_email_profile t
where t.id = @Id`

func (r *EmailRepository) GetProfile(id int) (*model.EmailProfile, error) {
	var row emailProfileRow
	if err := r.db.Get(context.Background(), &row, getProfileSQL, pgx.NamedArgs{
		"Id": id,
	}); err != nil {
		return nil, err
	}

	profile := &model.EmailProfile{
		Id:        row.Id,
		DomainId:  row.DomainId,
		Name:      row.Name,
		FlowId:    row.FlowId,
		Login:     row.Login,
		Password:  row.Password,
		Mailbox:   row.Mailbox,
		SmtpHost:  row.SmtpHost,
		SmtpPort:  row.SmtpPort,
		ImapHost:  row.ImapHost,
		ImapPort:  row.ImapPort,
		UpdatedAt: row.UpdatedAt,
		AuthType:  row.AuthType,
	}
	if len(row.Params) > 0 {
		_ = json.Unmarshal(row.Params, &profile.Params)
	}
	if len(row.Token) > 0 {
		_ = json.Unmarshal(row.Token, &profile.Token)
	}
	return profile, nil
}

const setTokenSQL = `update call_center.cc_email_profile
set updated_at = now(),
    token = @Token
where id = @Id`

func (r *EmailRepository) SetToken(id int, token *oauth2.Token) error {
	data, _ := json.Marshal(token)
	return r.db.Exec(context.Background(), setTokenSQL, pgx.NamedArgs{
		"Id":    id,
		"Token": data,
	})
}

type profileUpdatedAtRow struct {
	UpdatedAt int64 `db:"updated_at"`
}

const getProfileUpdatedAtSQL = `select (extract(EPOCH from updated_at) * 1000)::int8 updated_at
from call_center.cc_email_profile
where id = @Id and domain_id = @DomainId`

func (r *EmailRepository) GetProfileUpdatedAt(domainId int64, id int) (int64, error) {
	var row profileUpdatedAtRow
	if err := r.db.Get(context.Background(), &row, getProfileUpdatedAtSQL, pgx.NamedArgs{
		"Id":       id,
		"DomainId": domainId,
	}); err != nil {
		return 0, err
	}
	return row.UpdatedAt, nil
}

const setErrorSQL = `update call_center.cc_email_profile
set fetch_err = @Err
where id = @Id`

func (r *EmailRepository) SetError(profileId int, appErr error) error {
	errMsg := ""
	if appErr != nil {
		errMsg = appErr.Error()
	}
	return r.db.Exec(context.Background(), setErrorSQL, pgx.NamedArgs{
		"Id":  profileId,
		"Err": errMsg,
	})
}

func (r *EmailRepository) GerProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, error) {
	f := make([]string, 0, len(mapRes))
	for k, vi := range mapRes {
		v, _ := vi.(string)
		var val string
		switch v {
		case "html":
			f = append(f, "cid as "+pgsql.QuoteIdentifier(model.MailCidKey))
			val = `coalesce("html"::text, '') as ` + pgsql.QuoteIdentifier(k)
		case "from", "to", "subject", "contact_ids", "owner_id",
			"cc", "sender", "reply_to", "in_reply_to", "body", "attachments", "message_id", "id":
			val = `coalesce("` + v + `"::text, '') as ` + pgsql.QuoteIdentifier(k)
		default:
			continue
		}
		f = append(f, val)
	}

	q := `select row_to_json(t) variables
from (
    select ` + strings.Join(f, ", ") + `
    from (
        select
            id::text as id,
            message_id,
            array_to_string("from", ',') as from,
            array_to_string("to", ',') as to,
            subject,
            array_to_string("cc", ',') as cc,
            array_to_string("sender", ',') as sender,
            array_to_string("reply_to", ',') as reply_to,
            in_reply_to,
            body,
            html,
            (select jsonb_agg(row_to_json(t))
            from (
                select f.id, f.name, f.size, f.mime_type as mime
                from storage.files f
                where f.uuid = e.message_id
                    and f.domain_id = @DomainId
                limit 40
            ) t)::text as attachments,
            coalesce(array_to_json(contact_ids)::text, '') as contact_ids,
            coalesce(e.owner_id::text, '') as owner_id,
            e.cid
        from call_center.cc_email e
        where (id = @Id or message_id = @MessageId)
            and exists(select 1 from call_center.cc_email_profile p where p.domain_id = @DomainId and p.id = e.profile_id)
        order by e.created_at desc
        limit 1
    ) t
) t`

	var row lastBridgedRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"DomainId":  domainId,
		"Id":        id,
		"MessageId": messageId,
	}); err != nil {
		return nil, err
	}

	var vars model.Variables
	if err := json.Unmarshal(row.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

type smtpSettingsRow struct {
	Id       int             `db:"id"`
	AuthType string          `db:"auth_type"`
	Port     int             `db:"port"`
	Server   string          `db:"server"`
	Tls      bool            `db:"tls"`
	Auth     json.RawMessage `db:"auth"`
	Params   json.RawMessage `db:"params"`
	Token    json.RawMessage `db:"token"`
}

const smtpSettingsSQL = `select jsonb_build_object('user', p.login, 'password', p.password) as auth,
       p.smtp_port as port,
       p.smtp_host as server,
       coalesce((p.params ->> 'insecure')::bool, false) as tls,
       coalesce(p.auth_type, 'plain') as auth_type,
       p.params,
       p.token,
       p.id
from call_center.cc_email_profile p
where p.domain_id = @DomainId::int8
  and (p.id = @Id::int or p.name = @Name::varchar)
limit 1`

func (r *EmailRepository) SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, error) {
	var row smtpSettingsRow
	if err := r.db.Get(context.Background(), &row, smtpSettingsSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"Id":       search.Id,
		"Name":     search.Name,
	}); err != nil {
		return nil, err
	}

	settings := &model.SmtSettings{
		Id:       row.Id,
		AuthType: row.AuthType,
		Port:     row.Port,
		Server:   row.Server,
		Tls:      row.Tls,
	}
	if len(row.Auth) > 0 {
		_ = json.Unmarshal(row.Auth, &settings.Auth)
	}
	if len(row.Params) > 0 {
		_ = json.Unmarshal(row.Params, &settings.Params)
	}
	if len(row.Token) > 0 {
		_ = json.Unmarshal(row.Token, &settings.Token)
	}
	return settings, nil
}

const setContactSQL = `update call_center.cc_email
set contact_ids = @ContactIds
where message_id = @Id`

func (r *EmailRepository) SetContact(ctx context.Context, domainId int64, id string, contactIds []int64) error {
	return r.db.Exec(ctx, setContactSQL, pgx.NamedArgs{
		"ContactIds": contactIds,
		"Id":         id,
	})
}
