package meeting

import "github.com/webitel/flow_manager/internal/runtime/ops"

func (m *meetingOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Creates a Webitel meeting and stores the meeting URL in a flow variable.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"setVar": {
				Type:        "string",
				Required:    true,
				Description: "Variable name to store the resulting meeting URL.",
			},
			"title": {
				Type:        "string",
				Description: "Meeting title.",
			},
			"expireSec": {
				Type:        "integer",
				Default:     0,
				Description: "Meeting expiry in seconds (0 = service default).",
			},
			"basePath": {
				Type:        "string",
				Description: "Base URL path for the meeting link.",
			},
			"variables": {
				Type:        "object",
				Description: "Additional key-value pairs passed to the meeting service.",
			},
		},
		Examples: map[string]ops.Example{
			"create_meeting": {
				Description: "Create a meeting that expires in 1 hour",
				Schema:      `{"createMeeting": {"setVar": "meetUrl", "title": "Support call", "expireSec": 3600}}`,
			},
		},
	}
}
