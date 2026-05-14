package contacts

import "github.com/webitel/flow_manager/internal/runtime/ops"

// ── getContact ────────────────────────────────────────────────────────────────

func (o *getContactOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Fetches a contact by ID (etag) and stores the full contact object as JSON in a variable. " +
			"Use ${setVar}.name, ${setVar}.phones, etc. to access fields afterward.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"token": {
				Type:        "string",
				Required:    true,
				Description: "Webitel API access token for the Contacts service.",
			},
			"setVar": {
				Type:        "string",
				Required:    true,
				Description: "Variable name to store the contact JSON object.",
			},
			"id": {
				Type:        "string",
				Required:    true,
				Description: "Contact etag / ID. Supports ${variables} — typically ${wbt_contact_id}.",
			},
			"fields": {
				Type:        "array",
				Description: "Fields to return (e.g. [\"name\",\"phones\",\"emails\",\"variables\"]). Empty = all fields.",
			},
		},
		Notes: []string{
			"${wbt_contact_id} is automatically set when a call/chat is linked to a contact in Webitel.",
			"Access nested fields via JS dot-path: ${contact_data.name.common_name}.",
		},
		Examples: map[string]ops.Example{
			"lookup_by_contact_id": {
				Description: "Fetch contact linked to the current call",
				Schema: `{"getContact": {
  "token": "<CONTACTS_TOKEN>",
  "setVar": "contact_data",
  "id": "${wbt_contact_id}",
  "fields": ["name", "phones", "emails", "variables"]
}}`,
			},
		},
	}
}

// ── findContact ───────────────────────────────────────────────────────────────

func (o *findContactOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Searches contacts by phone, email, name, or other criteria. " +
			"Stores the result list as JSON. Returns an empty array if no contacts match.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"token": {
				Type:        "string",
				Required:    true,
				Description: "Webitel API access token for the Contacts service.",
			},
			"setVar": {
				Type:        "string",
				Required:    true,
				Description: "Variable name to store the found contacts JSON array.",
			},
			"q": {
				Type:        "string",
				Description: "Free-text search query (phone number, name, email). Supports ${variables}.",
			},
			"size": {
				Type:        "integer",
				Default:     16,
				Description: "Max number of results to return.",
			},
			"page": {
				Type:        "integer",
				Default:     1,
				Description: "Page number for pagination.",
			},
			"fields": {
				Type:        "array",
				Description: "Fields to include in results (e.g. [\"id\",\"name\",\"phones\"]).",
			},
		},
		Notes: []string{
			"Use size=1 when searching by a unique identifier (phone, email) — the first result is the contact.",
			"Check result length before accessing fields: an empty array means no match.",
		},
		Examples: map[string]ops.Example{
			"by_phone": {
				Description: "Find contact by caller's phone number",
				Schema: `{"findContact": {
  "token": "<CONTACTS_TOKEN>",
  "setVar": "found_contacts",
  "q": "${caller_id_number}",
  "size": 1,
  "fields": ["id", "name", "phones", "variables"]
}}`,
			},
		},
	}
}

// ── updateContact ─────────────────────────────────────────────────────────────

func (o *updateContactOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Updates fields on an existing contact. " +
			"Use x_json_mask to specify which fields to update (partial update — only listed paths are written).",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"token": {
				Type:        "string",
				Required:    true,
				Description: "Webitel API access token for the Contacts service.",
			},
			"setVar": {
				Type:        "string",
				Description: "Variable to store the updated contact JSON (optional).",
			},
			"x_json_mask": {
				Type:     "array",
				Required: true,
				Description: "JSON field paths to update (partial update). " +
					"Only fields listed here are written. Example: [\"variables\"], [\"name\"], [\"name\",\"variables\"].",
			},
			"input": {
				Type:     "object",
				Required: true,
				Description: "Contact data to write. Must include 'etag' (contact ID). " +
					"name: {common_name: string}. " +
					"variables: [{key: string, value: string}] — custom key-value pairs on the contact.",
			},
		},
		Notes: []string{
			"x_json_mask controls what is updated — omitting a field from the mask leaves it unchanged even if provided in input.",
			"variables is a key-value list, not a map: [{\"key\": \"order_id\", \"value\": \"A-1042\"}].",
		},
		Examples: map[string]ops.Example{
			"save_call_result": {
				Description: "Store call outcome and VIP flag on the contact",
				Schema: `{"updateContact": {
  "token": "<CONTACTS_TOKEN>",
  "x_json_mask": ["variables"],
  "input": {
    "etag": "${wbt_contact_id}",
    "variables": [
      {"key": "last_call_result", "value": "${call_result}"},
      {"key": "vip",              "value": "true"}
    ]
  }
}}`,
			},
			"update_name": {
				Description: "Update the contact's display name",
				Schema: `{"updateContact": {
  "token": "<CONTACTS_TOKEN>",
  "x_json_mask": ["name"],
  "input": {
    "etag": "${wbt_contact_id}",
    "name": {"common_name": "${client_name}"}
  }
}}`,
			},
		},
	}
}
