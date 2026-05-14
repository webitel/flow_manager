package chat

import "github.com/webitel/flow_manager/internal/runtime/ops"

// ── sendMessage ───────────────────────────────────────────────────────────────

func (o *sendMessageOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Sends a rich message to the chat: text, optional file attachment, and reply/inline keyboard buttons.",
		AvailableIn: []string{"chat"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"text": {
				Type:        "string",
				Description: "Message text. Supports ${variables}.",
			},
			"file": {
				Type: "object",
				Description: "File attachment. " +
					"Provide {url: string} for a public URL, or {id: int, name: string} to look up a file from Webitel media storage.",
			},
			"buttons": {
				Type: "array",
				Description: "Reply keyboard — array of button rows, each row is an array of button objects. " +
					"Button fields: caption (label shown to user), text (text sent when pressed), " +
					"type ('text' or 'url'), url (for type=url).",
			},
			"inline": {
				Type:        "array",
				Description: "Inline keyboard buttons (appear below the message, not as a full keyboard). Same structure as buttons.",
			},
			"noInput": {
				Type:        "boolean",
				Default:     false,
				Description: "Hide the text input field after sending (forces button selection).",
			},
			"kind": {
				Type:        "string",
				Description: "Message kind hint for the chat client (e.g. 'text'). Usually omitted.",
			},
		},
		Examples: map[string]ops.Example{
			"simple_text": {
				Description: "Plain text message",
				Schema:      `{"sendMessage": {"text": "Hello! How can I help you?"}}`,
			},
			"with_buttons": {
				Description: "Text with reply keyboard",
				Schema: `{"sendMessage": {
  "text": "Please choose an option:",
  "buttons": [
    [{"caption": "Support", "text": "support"}],
    [{"caption": "Sales",   "text": "sales"}]
  ]
}}`,
			},
			"with_inline": {
				Description: "Text with inline keyboard and URL button",
				Schema: `{"sendMessage": {
  "text": "Open our portal:",
  "inline": [[
    {"caption": "Portal", "type": "url", "url": "https://example.com"}
  ]]
}}`,
			},
			"with_file": {
				Description: "Send a file from media storage",
				Schema:      `{"sendMessage": {"file": {"id": "<FILE_ID>", "name": "document.pdf"}}}`,
			},
		},
	}
}

// ── sendText ──────────────────────────────────────────────────────────────────

func (sendTextOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Sends a plain text message to the chat. " +
			"Deprecated — use sendMessage instead (sendText supports text only, no buttons or files).",
		AvailableIn: []string{"chat"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"sendText": {
				Type:        "string",
				Description: "Message text. Supports ${variables}.",
			},
		},
		Notes: []string{
			"Deprecated — prefer sendMessage which supports buttons, files, and inline keyboards.",
		},
		Examples: map[string]ops.Example{
			"basic": {
				Schema: `{"sendText": "Your request has been received."}`,
			},
		},
	}
}

// ── menu ──────────────────────────────────────────────────────────────────────

func (menuOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Sends a message with a button keyboard. " +
			"Does NOT wait for a response — use recvMessage after to capture the selection.",
		AvailableIn: []string{"chat"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"text": {
				Type:        "string",
				Description: "Prompt text shown above the buttons.",
			},
			"type": {
				Type:        "string",
				Default:     "buttons",
				Description: "'buttons' — reply keyboard (full-width at bottom of chat). 'inline' — inline keyboard below the message.",
			},
			"buttons": {
				Type:        "array",
				Description: "Button rows for type=buttons. Each row is an array of {caption, text} objects.",
			},
			"inline": {
				Type:        "array",
				Description: "Button rows for type=inline. Same structure as buttons.",
			},
			"noInput": {
				Type:        "boolean",
				Default:     false,
				Description: "Hide text input field, forcing the user to pick a button.",
			},
		},
		Notes: []string{
			"menu only sends the keyboard — it does not wait for input. Follow with recvMessage to capture the selection.",
		},
		Examples: map[string]ops.Example{
			"department_select": {
				Description: "Present department selection, then capture reply",
				Schema: `{"menu": {
  "text": "Choose a department:",
  "type": "buttons",
  "buttons": [
    [{"caption": "Tech Support", "text": "support"}],
    [{"caption": "Sales",        "text": "sales"}],
    [{"caption": "Billing",      "text": "billing"}]
  ]
}},
{"recvMessage": {"set": "department", "timeout": 120}}`,
			},
		},
	}
}

// ── recvMessage ───────────────────────────────────────────────────────────────

func (chatRecvMessageOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Suspends the flow and waits for the next message from the user. " +
			"Stores the received text in a variable. " +
			"Supports trigger commands (prefix 'commands-<text>') for inline branching.",
		AvailableIn: []string{"chat"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"set": {
				Type:        "string",
				Required:    true,
				Description: "Variable name to store the received message text.",
			},
			"timeout": {
				Type:        "integer",
				Description: "Wait timeout in seconds before the timeout path fires.",
			},
			"messageTimeout": {
				Type:        "integer",
				Description: "Inactivity timeout in seconds (resets on each new message).",
			},
			"timeoutSet": {
				Type:        "string",
				Description: "Variable name to set to 'true' when the timeout fires (for detecting no-response).",
			},
		},
		Notes: []string{
			"The flow is suspended (not blocked) while waiting — no goroutine is held.",
			"On timeout: sets timeoutSet='true' if configured, then continues to the next op.",
			"Trigger commands: if the received message matches a 'commands-<text>' trigger key, " +
				"that branch runs and recvMessage re-enters (ReenterOnResume) to keep waiting.",
		},
		Examples: map[string]ops.Example{
			"simple": {
				Description: "Wait up to 2 minutes for user input",
				Schema:      `{"recvMessage": {"set": "user_input", "timeout": 120}}`,
			},
			"with_timeout_detection": {
				Description: "Detect no-response and branch accordingly",
				Schema: `{"recvMessage": {
  "set": "user_reply",
  "timeout": 60,
  "timeoutSet": "chat_timed_out"
}},
{"if": {
  "expression": "${chat_timed_out} == 'true'",
  "then": [{"sendMessage": {"text": "Session expired. Goodbye!"}}, {"break": ""}],
  "else": [{"sendMessage": {"text": "You said: ${user_reply}"}}]
}}`,
			},
			"menu_and_recv": {
				Description: "Send a keyboard then wait for selection",
				Schema: `{"menu": {
  "text": "How can we help?",
  "type": "buttons",
  "buttons": [
    [{"caption": "Tech Support", "text": "support"}],
    [{"caption": "Billing",      "text": "billing"}]
  ]
}},
{"recvMessage": {"set": "selected_dept", "timeout": 120}},
{"switch": {
  "variable": "${selected_dept}",
  "case": {
    "support": [{"joinQueue": {"queue": {"id": "<SUPPORT_QUEUE_ID>"}}}],
    "billing":  [{"joinQueue": {"queue": {"id": "<BILLING_QUEUE_ID>"}}}],
    "_":         [{"sendMessage": {"text": "Please use the buttons above."}}, {"goto": "menu-tag"}]
  }
}}`,
			},
		},
	}
}
