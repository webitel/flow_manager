package email

import "github.com/webitel/flow_manager/internal/runtime/ops"

func (o *sendEmailOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Send an SMTP email with optional attachments, OAuth2 auth, retry, and DB persistence.",
		AvailableIn: []string{"voice", "chat", "email", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"to":               {Type: "[]string", Required: true, Description: "Recipient addresses."},
			"message":          {Type: "string", Required: true, Description: "Email body."},
			"subject":          {Type: "string", Description: "Subject line."},
			"from":             {Type: "string", Description: "Sender address (defaults to SMTP user)."},
			"cc":               {Type: "[]string", Description: "CC addresses."},
			"replyToId":        {Type: "string", Description: "In-Reply-To message ID (with angle brackets)."},
			"type":             {Type: "string", Description: "MIME type, default text/html."},
			"profile":          {Type: "object", Description: "SMTP profile lookup {id|name}."},
			"smtp":             {Type: "object", Description: "Inline SMTP settings (overridden by profile)."},
			"attachment.files": {Type: "[]File", Description: "Files to attach by file ID."},
			"retryCount":       {Type: "int", Description: "Number of send retries on dial failure."},
			"async":            {Type: "bool", Description: "Send in background goroutine; returns immediately."},
			"store":            {Type: "bool", Description: "Persist sent email to DB (requires smtp profile with Id)."},
			"contactIds":       {Type: "[]int64", Description: "Contact IDs to link (requires store=true)."},
			"ownerId":          {Type: "int64", Description: "Owner user ID to link (requires store=true)."},
			"set":              {Type: "object", Description: `Variable mapping keys: "message_id", "id" (DB row), "error".`},
		},
		Examples: map[string]ops.Example{
			"send_html": {
				Description: "Send a simple HTML email via stored SMTP profile",
				Schema: `{"sendEmail": {"profile": {"id": 1}, "to": ["user@example.com"],` +
					` "subject": "Hello", "message": "<b>Hi!</b>",` +
					` "set": {"message_id": "email_mid", "error": "email_err"}}}`,
			},
		},
	}
}
