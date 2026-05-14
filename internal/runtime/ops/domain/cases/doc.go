package cases

import "github.com/webitel/flow_manager/internal/runtime/ops"

// ── createCase ────────────────────────────────────────────────────────────────

func (o *createCaseOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Creates a new support case in the Webitel Cases module and optionally stores the result.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"token": {
				Type:        "string",
				Required:    true,
				Description: "Webitel API access token for the Cases service.",
			},
			"setVar": {
				Type:        "string",
				Description: "Variable to store the created case JSON (optional). Use ${setVar}.id for the case ID.",
			},
			"fields": {
				Type:        "array",
				Description: "Fields to return in the response (e.g. [\"id\",\"subject\",\"status\"]). Empty = default set.",
			},
			"input": {
				Type:     "object",
				Required: true,
				Description: "Case data to create. " +
					"subject (required, string): case title — supports ${variables}. " +
					"description (string): detailed description. " +
					"source ({id|name}): origin channel — use {name: \"phone\"}, {name: \"chat\"}, {name: \"email\"}, etc. " +
					"service ({id|name}): service catalog item. " +
					"status ({id|name}): initial status. " +
					"priority ({id|name}): case priority. " +
					"assignee ({id|name}): agent to assign. " +
					"reporter ({id|name}): contact who reported. " +
					"contact_info (string): free-text contact info.",
			},
		},
		Notes: []string{
			"source.name is the most common lookup field — use the source name configured in Webitel (e.g. 'phone', 'chat', 'email').",
			"Use fields: [\"id\"] if you only need the case ID to avoid unnecessary data transfer.",
		},
		Examples: map[string]ops.Example{
			"minimal": {
				Description: "Create a case from an inbound call",
				Schema: `{"createCase": {
  "token": "<CASES_TOKEN>",
  "setVar": "new_case",
  "input": {
    "subject": "Call from ${caller_id_number}",
    "source": {"name": "phone"}
  }
}}`,
			},
			"full": {
				Description: "Create a case with service, priority, and contact info",
				Schema: `{"createCase": {
  "token": "<CASES_TOKEN>",
  "setVar": "new_case",
  "fields": ["id", "subject", "status"],
  "input": {
    "subject": "${case_subject}",
    "description": "${case_description}",
    "service":  {"id": "<SERVICE_ID>"},
    "priority": {"name": "Medium"},
    "source":   {"name": "phone"},
    "contact_info": "${caller_id_number}"
  }
}}`,
			},
			"chat_case": {
				Description: "Create a case from a chat conversation",
				Schema: `{"createCase": {
  "token": "<CASES_TOKEN>",
  "setVar": "new_case",
  "input": {
    "subject": "Chat from ${client_name}",
    "source": {"name": "chat"},
    "description": "${first_message}"
  }
}}`,
			},
		},
	}
}

// ── updateCase ────────────────────────────────────────────────────────────────

func (o *updateCaseOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Updates an existing case by etag (case ID). " +
			"Use x_json_mask for partial update — only listed paths are written.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"token": {
				Type:        "string",
				Required:    true,
				Description: "Webitel API access token for the Cases service.",
			},
			"setVar": {
				Type:        "string",
				Description: "Variable to store the updated case JSON (optional).",
			},
			"x_json_mask": {
				Type:     "array",
				Required: true,
				Description: "JSON field paths to update. Only listed fields are written. " +
					"Example: [\"status\"], [\"status\",\"close_result\"], [\"subject\",\"description\"].",
			},
			"input": {
				Type:     "object",
				Required: true,
				Description: "Updated case data. Must include 'etag' (case ID — supports ${variables}). " +
					"subject (string), description (string), " +
					"status ({id|name}): new case status. " +
					"close_result (string): required when transitioning to a closed/final status. " +
					"assignee ({id|name}): reassign agent.",
			},
		},
		Notes: []string{
			"x_json_mask controls what is written — fields not listed are left unchanged.",
			"close_result is mandatory when moving a case to a final/closed status in most Webitel configurations.",
		},
		Examples: map[string]ops.Example{
			"close_case": {
				Description: "Move case to Resolved status with a close note",
				Schema: `{"updateCase": {
  "token": "<CASES_TOKEN>",
  "x_json_mask": ["status", "close_result"],
  "input": {
    "etag": "${case_id}",
    "status": {"name": "Resolved"},
    "close_result": "Issue resolved during call"
  }
}}`,
			},
			"reassign": {
				Description: "Reassign case to a different agent",
				Schema: `{"updateCase": {
  "token": "<CASES_TOKEN>",
  "x_json_mask": ["assignee"],
  "input": {
    "etag": "${case_id}",
    "assignee": {"id": "<AGENT_ID>"}
  }
}}`,
			},
			"update_subject_and_description": {
				Description: "Edit case subject and description",
				Schema: `{"updateCase": {
  "token": "<CASES_TOKEN>",
  "x_json_mask": ["subject", "description"],
  "input": {
    "etag": "${case_id}",
    "subject": "${updated_subject}",
    "description": "${updated_description}"
  }
}}`,
			},
		},
	}
}
