package builtin

import "github.com/webitel/flow_manager/internal/runtime/ops"

func (ifOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Conditional branch. Evaluates an expression and runs 'then' if truthy, 'else' otherwise.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"expression": {
				Type:     "string",
				Required: true,
				Description: "JS-compatible boolean expression. " +
					"${var} → session variable, $${var} → global variable, &func(args) → date/time helper.",
			},
			"then": {
				Type:        "array",
				Description: "Apps to execute when expression is truthy.",
			},
			"else": {
				Type:        "array",
				Description: "Apps to execute when expression is falsy.",
			},
		},
		Notes: []string{
			"Both 'then' and 'else' are optional.",
			"Reserved JS keywords are stripped before evaluation — keep expressions pure inline logic.",
		},
		Examples: map[string]ops.Example{
			"business_hours": {
				Description: "Open queue 09:00-18:00 on weekdays",
				Schema: `{"if": {"expression": "&wday(2-6) && &time_of_day(09:00-18:00)",` +
					` "then": [{"joinQueue": {"queue": {"id": 42}}}],` +
					` "else": [{"hangup": "NORMAL_CLEARING"}]}}`,
			},
			"variable_check": {
				Description: "Route based on a session variable",
				Schema:      `{"if": {"expression": "${language} == 'uk'", "then": [...], "else": [...]}}`,
			},
		},
	}
}

func (whileOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Loops the 'do' body while the condition is truthy.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"condition": {
				Type:        "string",
				Required:    true,
				Description: "Same expression syntax as 'if'. Loop runs while truthy.",
			},
			"do": {
				Type:        "array",
				Description: "Apps executed on each iteration.",
			},
		},
		Notes: []string{
			"No built-in iteration limit — use a counter variable + break to avoid infinite loops.",
		},
	}
}

func (switchOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Dispatches to a named branch based on a variable value. Use '_' as the default case.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"variable": {
				Type:        "string",
				Required:    true,
				Description: "Variable reference whose value selects the branch, e.g. '${lang}'.",
			},
			"case": {
				Type:        "object",
				Required:    true,
				Description: "Map of case-value → app array. '_' is the default/fallthrough branch.",
			},
		},
		Examples: map[string]ops.Example{
			"language_routing": {
				Description: "Route by language variable",
				Schema: `{"switch": {"variable": "${lang}",` +
					` "case": {"uk": [{"goto": "uk_branch"}], "en": [{"goto": "en_branch"}], "_": [{"goto": "default_branch"}]}}}`,
			},
		},
	}
}

func (setOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Sets one or more session variables. Values support ${var} interpolation.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"<varName>": {
				Type:        "string",
				Description: "Key-value pairs to set. Values are interpolated.",
			},
		},
		Examples: map[string]ops.Example{
			"basic": {
				Description: "Set two variables",
				Schema:      `{"set": {"greeting": "hello ${name}", "lang": "uk"}}`,
			},
		},
	}
}

func (gotoOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Jumps execution to a node marked with the matching 'tag'. Supports loops.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"goto": {
				Type:        "string",
				Required:    true,
				Description: "Tag name of the target node.",
			},
		},
		Notes: []string{
			"Maximum 100 consecutive goto jumps without an intermediate op — prevents infinite tight-loops.",
		},
		Examples: map[string]ops.Example{
			"retry_loop": {
				Description: "Loop back to a tagged node",
				Schema:      `[{"tag": "retry", "set": {"attempts": "0"}}, ..., {"goto": "retry"}]`,
			},
		},
	}
}

func (breakOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Stops flow execution immediately.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
	}
}

func (logOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Emits a debug log line. No effect on flow logic.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Args: map[string]ops.ArgDoc{
			"log": {
				Type:        "string",
				Description: "Message to log. Supports ${var} interpolation.",
			},
		},
	}
}
