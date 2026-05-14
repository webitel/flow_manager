package processing

import "github.com/webitel/flow_manager/internal/runtime/ops"

// ── formComponent ─────────────────────────────────────────────────────────────

func (o *formComponentOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Defines a UI component to appear inside an operator form. " +
			"Must be called BEFORE generateForm. The component ID is referenced in generateForm.body.",
		AvailableIn: []string{"form"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"id": {
				Type:        "string",
				Required:    true,
				Description: "Unique component ID within the form. Must match the ID listed in generateForm.body.",
			},
			"view": {
				Type:     "object",
				Required: true,
				Description: "Visual configuration. " +
					"component (required): form-text (read-only text block), wt-input (text input), " +
					"wt-select (dropdown), wt-textarea (multi-line), wt-datetimepicker (date/time picker), " +
					"form-i-frame (embedded iframe), rich-text-editor (HTML editor). " +
					"label: field label shown to operator. " +
					"hint: helper text below the field. " +
					"initialValue: pre-filled value (supports ${variables}). " +
					"readonly (bool): prevent editing. " +
					"collapsible (bool): allow collapsing (form-text). " +
					"enableCopying (bool): show copy button (form-text). " +
					"color: form-text accent (primary, secondary, accent, warning, danger). " +
					"options: [{name, value}] for wt-select. " +
					"multiple (bool): multi-select for wt-select. " +
					"output: 'html' or 'text' for rich-text-editor.",
			},
		},
		Notes: []string{
			"formComponent only registers the component — it does not render the form. Call generateForm to render.",
			"Component IDs must be unique within one generateForm call.",
			"initialValue supports ${variables} — use it to pre-fill with caller data.",
		},
		Examples: map[string]ops.Example{
			"read_only_text": {
				Description: "Display caller info (read-only)",
				Schema: `{"formComponent": {
  "id": "client-info",
  "view": {
    "component": "form-text",
    "label": "Client",
    "initialValue": "Phone: ${caller_id_number}\nName: ${client_name}",
    "collapsible": false,
    "enableCopying": true
  }
}}`,
			},
			"text_input": {
				Description: "Editable text field for operator notes",
				Schema: `{"formComponent": {
  "id": "operator-note",
  "view": {
    "component": "wt-input",
    "label": "Note",
    "hint": "Brief note about this call",
    "initialValue": ""
  }
}}`,
			},
			"dropdown": {
				Description: "Dropdown with predefined options",
				Schema: `{"formComponent": {
  "id": "call-reason",
  "view": {
    "component": "wt-select",
    "label": "Call Reason",
    "options": [
      {"name": "Technical Issue", "value": "technical"},
      {"name": "Billing",         "value": "billing"},
      {"name": "General Info",    "value": "info"}
    ]
  }
}}`,
			},
			"rich_text_editor": {
				Description: "HTML editor for email reply body",
				Schema: `{"formComponent": {
  "id": "reply-body",
  "view": {
    "component": "rich-text-editor",
    "label": "Reply Text",
    "initialValue": "Dear ${client_name},<br/><br/>",
    "output": "html"
  }
}}`,
			},
		},
	}
}

// ── generateForm ──────────────────────────────────────────────────────────────

func (o *generateFormOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Renders the operator form and suspends the flow until the operator clicks an action button. " +
			"Must be called AFTER all formComponent definitions. " +
			"The chosen action ID is stored in the variable named by 'id'.",
		AvailableIn: []string{"form"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"id": {
				Type:     "string",
				Required: true,
				Description: "Form identifier. The action chosen by the operator is saved as a variable with this name. " +
					"Use ${id} after generateForm to branch on the result.",
			},
			"title": {
				Type:        "string",
				Description: "Form title displayed at the top.",
			},
			"body": {
				Type:     "array",
				Required: true,
				Description: "Ordered list of component IDs to display. " +
					"Each ID must match a formComponent defined earlier in the flow.",
			},
			"actions": {
				Type:     "array",
				Required: true,
				Description: "Action buttons shown to the operator. " +
					"Each action: {id: string, view: {text: string, color: primary|success|danger|warning|accent}}. " +
					"The clicked action's id is stored in the form variable.",
			},
		},
		Notes: []string{
			"generateForm is Suspendable — the flow pauses until the operator acts.",
			"All formComponent ops must appear BEFORE generateForm in the flow.",
			"After generateForm, switch or if on ${<form-id>} to handle each action.",
			"Component IDs in body must exactly match the IDs registered by formComponent.",
		},
		Examples: map[string]ops.Example{
			"basic_form": {
				Description: "Full operator form: define components → render → branch on action",
				Schema: `{"formComponent": {
  "id": "client-info",
  "view": {"component": "form-text", "label": "Caller", "initialValue": "${caller_id_number}"}
}},
{"formComponent": {
  "id": "note",
  "view": {"component": "wt-input", "label": "Note"}
}},
{"formComponent": {
  "id": "reason",
  "view": {
    "component": "wt-select",
    "label": "Result",
    "options": [
      {"name": "Resolved",  "value": "resolved"},
      {"name": "Callback",  "value": "callback"}
    ]
  }
}},
{"generateForm": {
  "id": "process-form",
  "title": "Process Call",
  "body": ["client-info", "note", "reason"],
  "actions": [
    {"id": "submit", "view": {"text": "Submit",   "color": "success"}},
    {"id": "cancel", "view": {"text": "Return",   "color": "danger"}}
  ]
}},
{"switch": {
  "variable": "${process-form}",
  "case": {
    "submit": [{"attemptResult": {"status": "success", "description": "${note}"}}],
    "cancel": [{"attemptResult": {"status": "abandoned"}}]
  }
}}`,
			},
		},
	}
}

// ── attemptResult ─────────────────────────────────────────────────────────────

func (o *attemptResultOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Closes the current operator attempt with a result status. " +
			"Typically the last step in a form flow. " +
			"Automatically merges exported session variables into the attempt record.",
		AvailableIn: []string{"form"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"status": {
				Type:     "string",
				Required: true,
				Description: "Outcome: " +
					"'success' — attempt completed; " +
					"'abandoned' — returned to queue for retry; " +
					"'missed' — skipped without processing.",
			},
			"description": {
				Type:        "string",
				Description: "Free-text note attached to the attempt.",
			},
			"variables": {
				Type: "object",
				Description: "Key-value pairs (string → string) persisted to the subscriber/contact record. " +
					"Use to store form answers, tags, external IDs. Keys and values must be strings.",
			},
			"stickyDisplay": {
				Type:        "boolean",
				Default:     false,
				Description: "Stick the outbound caller ID used on this call to the subscriber. " +
					"Subsequent dials to the same subscriber reuse the same display number.",
			},
			"waitBetweenRetries": {
				Type:        "integer",
				Description: "Delay in seconds before the next retry to this subscriber " +
					"(applies to abandoned/missed only). Overrides the queue's default retry interval.",
			},
		},
		Notes: []string{
			"Exported variables (via export op) are automatically merged into variables before saving.",
			"waitBetweenRetries has no effect when status=success.",
		},
		Examples: map[string]ops.Example{
			"success": {
				Description: "Mark attempt as successfully handled",
				Schema:      `{"attemptResult": {"status": "success", "description": "Issue resolved"}}`,
			},
			"abandoned_with_retry": {
				Description: "Return to queue, retry in 5 minutes",
				Schema:      `{"attemptResult": {"status": "abandoned", "description": "No answer", "waitBetweenRetries": 300}}`,
			},
			"success_with_variables": {
				Description: "Save form data to subscriber record",
				Schema: `{"attemptResult": {
  "status": "success",
  "variables": {"order_id": "${order_id}", "segment": "vip"},
  "stickyDisplay": true
}}`,
			},
		},
	}
}
