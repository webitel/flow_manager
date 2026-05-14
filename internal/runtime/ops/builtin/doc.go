package builtin

import "github.com/webitel/flow_manager/internal/runtime/ops"

func (ifOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Conditional branch. Evaluates a JS expression and runs 'then' if truthy, 'else' otherwise. " +
			"Supports ${var} (session), $${var} (global), and &func(args) (date/time helpers) in expressions.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"expression": {
				Type:     "string",
				Required: true,
				Description: "Boolean expression. Syntax: " +
					"${name} → session variable; " +
					"$${name} → global/domain variable; " +
					"&func(arg) → date/time helper (&wday, &hour, &time_of_day, &mon, &mday, &date_time, …); " +
					"standard JS operators (==, !=, >, <, >=, <=, &&, ||, !, +, -, *, /, %, ?:). " +
					"Pattern arg forms: exact '5', range '9-18', list '1,3,5', mixed '1,3-5,7'. " +
					"Reserved keywords stripped before eval: function, case, if, return, new, switch, var, this, typeof, for, while, break, do, continue.",
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
			"&wday numbering: 1=Sun, 2=Mon, 3=Tue, 4=Wed, 5=Thu, 6=Fri, 7=Sat — 'Mon-Fri' is &wday(2-6).",
			"&time_of_day accepts 'HH:MM-HH:MM', comma-separated for multiple windows: '09:00-13:00,14:00-18:00'.",
			"&date_time range format: 'YYYY-MM-DD HH:MM:SS~YYYY-MM-DD HH:MM:SS' (evaluated in flow timezone).",
			"Helper args are always strings — write &hour(9-18), not &hour('9-18').",
			"Write &func(args), not sys.func(args). The parser rewrites '&' helpers and '${}'/'$${}' vars before JS eval.",
			"Both 'then' and 'else' are optional arrays.",
			"Do not use reserved JS keywords in expressions; keep them as pure inline logic (no statements, no control-flow keywords).",
		},
		Examples: map[string]ops.Example{
			"business_hours": {
				Description: "Open queue 09:00-18:00 on weekdays (Mon-Fri)",
				Schema: `{"if": {
  "expression": "&wday(2-6) && &time_of_day(09:00-18:00)",
  "then": [{"joinQueue": {"queue": {"id": 42}}}],
  "else": [{"hangup": "NORMAL_CLEARING"}]
}}`,
			},
			"multi_window": {
				Description: "Split workday (09-13 and 14-18), Mon-Fri",
				Schema: `{"if": {
  "expression": "&wday(2-6) && &time_of_day(09:00-13:00,14:00-18:00)",
  "then": [{"joinQueue": {"queue": {"id": 42}}}],
  "else": [{"hangup": "NORMAL_CLEARING"}]
}}`,
			},
			"variable_check": {
				Description: "Route based on a session variable",
				Schema: `{"if": {
  "expression": "${language} == 'uk'",
  "then": [{"playback": {"files": [{"id": "${greet_uk_id}"}]}}],
  "else": [{"playback": {"files": [{"id": "${greet_en_id}"}]}}]
}}`,
			},
			"global_and_session": {
				Description: "Combine global and session variables",
				Schema: `{"if": {
  "expression": "$${vip_level} >= 3 && ${caller_id_number} != ''",
  "then": [{"goto": "vip_queue_branch"}],
  "else": [{"goto": "regular_queue_branch"}]
}}`,
			},
			"holiday_window": {
				Description: "Match a specific datetime range (Christmas)",
				Schema: `{"if": {
  "expression": "&date_time(2026-12-24 00:00:00~2026-12-26 23:59:59)",
  "then": [{"hangup": "NORMAL_CLEARING"}]
}}`,
			},
			"numeric_threshold": {
				Description: "Numeric comparison on a session variable",
				Schema: `{"if": {
  "expression": "${queue_waiting} > 10",
  "then": [{"goto": "overflow_branch"}],
  "else": [{"joinQueue": {"queue": {"id": 42}}}]
}}`,
			},
		},
	}
}

func (whileOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Loops the 'do' body while the condition is truthy. " +
			"Uses the same expression engine as 'if' — supports ${var}, $${var}, and &func(args).",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"condition": {
				Type:        "string",
				Required:    true,
				Description: "Boolean expression (same syntax as 'if'). Loop continues while truthy.",
			},
			"do": {
				Type:        "array",
				Description: "Apps executed on each iteration.",
			},
		},
		Notes: []string{
			"No built-in iteration limit — use a counter variable and break inside 'do' to avoid infinite loops.",
		},
		Examples: map[string]ops.Example{
			"counter_loop": {
				Description: "Loop up to 3 retries using a counter variable",
				Schema: `{"set": {"tries": "0"}},
{"while": {
  "condition": "${tries} < 3",
  "do": [
    {"playback": {"files": [{"id": "${prompt_id}"}]}},
    {"set": {"tries": "${tries} + 1"}}
  ]
}}`,
			},
		},
	}
}

func (switchOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Multi-branch routing based on a variable value. Use '_' as the default/fallthrough case.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"variable": {
				Type:        "string",
				Required:    true,
				Description: "Variable reference whose value selects the branch, e.g. '${ivr_choice}'.",
			},
			"case": {
				Type:        "object",
				Required:    true,
				Description: "Map of value → app array. '_' runs when no other case matches.",
			},
		},
		Notes: []string{
			"Use '_' (underscore) as the default case key — not 'default'.",
		},
		Examples: map[string]ops.Example{
			"ivr_routing": {
				Description: "Route IVR digit input to queue, bridge, or default",
				Schema: `{"switch": {
  "variable": "${ivr_choice}",
  "case": {
    "1": [{"joinQueue": {"queue": {"id": 10}}}],
    "2": [{"bridge": {"endpoints": [{"type": "user", "extension": "200"}]}}],
    "_": [{"goto": "ivr-menu-tag"}]
  }
}}`,
			},
		},
	}
}

func (setOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Sets one or more session variables. Values are always strings and support ${var} interpolation.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"<varName>": {
				Type:        "string",
				Description: "Arbitrary key→value pairs. All values are stored as strings.",
			},
		},
		Notes: []string{
			"Values are always strings — numeric comparisons must compare against string literals: ${x} > '10'.",
			"Variables are accessible anywhere in the flow via ${variable_name}.",
			"Special channel variables: hangup_after_bridge, continue_on_fail (set BEFORE bridge).",
		},
		Examples: map[string]ops.Example{
			"basic": {
				Description: "Set two variables",
				Schema:      `{"set": {"welcome_lang": "uk", "max_tries": "3"}}`,
			},
			"interpolated": {
				Description: "Build a value from existing variables",
				Schema:      `{"set": {"greeting": "Hello ${client_name}!"}}`,
			},
			"bridge_flags": {
				Description: "Set channel vars before bridge",
				Schema:      `{"set": {"continue_on_fail": "true", "hangup_after_bridge": "true"}}`,
			},
		},
	}
}

func (gotoOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Jumps execution to the node with the specified 'tag'. Used for menu repeats or loops.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"goto": {
				Type:        "string",
				Required:    true,
				Description: "Tag name of the target node (set via tag: on the destination app object).",
			},
		},
		Notes: []string{
			"Maximum 100 consecutive goto jumps without an intermediate op — prevents infinite tight-loops.",
			"The target node must have a matching 'tag' field in the schema.",
		},
		Examples: map[string]ops.Example{
			"ivr_repeat": {
				Description: "Return to IVR menu on invalid input",
				Schema:      `[{"tag": "ivr-menu", ...}, ..., {"goto": "ivr-menu"}]`,
			},
		},
	}
}

func (breakOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Stops flow execution immediately. Triggers the disconnect handler.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
	}
}

func (logOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Writes a debug message to the schema execution log. Has no effect on flow logic.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Args: map[string]ops.ArgDoc{
			"log": {
				Type:        "string",
				Description: "Message text. Supports ${var} interpolation.",
			},
		},
		Examples: map[string]ops.Example{
			"debug": {
				Description: "Log key variables for debugging",
				Schema:      `{"log": "choice=${ivr_choice} caller=${caller_id_number}"}`,
			},
		},
	}
}

func (softSleepOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Pauses schema execution for a given duration without blocking a goroutine. " +
			"The runtime suspends the flow and resumes via a timer worker after the delay expires.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"softSleep": {
				Type:        "integer",
				Required:    true,
				Description: "Pause duration in milliseconds.",
			},
		},
		Examples: map[string]ops.Example{
			"two_seconds": {
				Description: "Pause for 2 seconds",
				Schema:      `{"softSleep": 2000}`,
			},
		},
	}
}

func (stringOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Applies a string transformation function to data and stores the result in a variable.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"setVar": {
				Type:        "string",
				Required:    true,
				Description: "Variable to store the result.",
			},
			"fn": {
				Type:     "string",
				Required: true,
				Description: "Function name. " +
					"Go-native: reverse, charAt (args: [index]), length, " +
					"base64 (args: ['encoder'|'decoder']), MD5, SHA-256, SHA-512, " +
					"gomatch (args: ['regexp'] — returns captured groups). " +
					"JS String.prototype: toUpperCase, toLowerCase, trim, " +
					"split (args: [delimiter] — result joined by ','), " +
					"replace (args: [search, replacement]), includes, indexOf, slice (args: [start, end]).",
			},
			"data": {
				Type:        "string",
				Required:    true,
				Description: "Input string. Supports ${variables}.",
			},
			"args": {
				Type:        "array",
				Description: "Extra arguments for the chosen fn (see fn description for per-function args).",
			},
		},
		Notes: []string{
			"For JS functions, a /regex/flags string in args is automatically converted to a RegExp object.",
			"split returns all parts joined by ',' into a single string.",
		},
		Examples: map[string]ops.Example{
			"uppercase": {
				Description: "Convert a variable to upper case",
				Schema:      `{"string": {"setVar": "name_upper", "fn": "toUpperCase", "data": "${client_name}"}}`,
			},
			"md5": {
				Description: "MD5-hash a phone number",
				Schema:      `{"string": {"setVar": "phone_hash", "fn": "MD5", "data": "${caller_id_number}"}}`,
			},
			"base64_encode": {
				Description: "Base64-encode a token",
				Schema:      `{"string": {"setVar": "encoded", "fn": "base64", "data": "${token}", "args": ["encoder"]}}`,
			},
			"regexp_match": {
				Description: "Validate input with a Go regexp",
				Schema:      `{"string": {"setVar": "match_result", "fn": "gomatch", "data": "${user_input}", "args": ["^[0-9]{10}$"]}}`,
			},
			"trim_and_lowercase": {
				Description: "Normalize user input",
				Schema: `{"string": {"setVar": "input_clean", "fn": "trim", "data": "${raw_input}"}},
{"string": {"setVar": "input_clean", "fn": "toLowerCase", "data": "${input_clean}"}}`,
			},
		},
	}
}

func (mathOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Applies a Math function or picks a random value from a list.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"setVar": {
				Type:        "string",
				Required:    true,
				Description: "Variable to store the result.",
			},
			"fn": {
				Type:    "string",
				Default: "random",
				Description: "Function name. " +
					"Go-native: random (picks a random element from data array). " +
					"JS Math.*: round, floor, ceil, abs, " +
					"max (max of data array), min (min of data array), " +
					"pow (args: [base, exponent]), sqrt, PI (returns Math.PI).",
			},
			"data": {
				Type:        "array",
				Description: "Input values for the function. For 'random' — the pool to pick from.",
			},
		},
		Examples: map[string]ops.Example{
			"random_pick": {
				Description: "Pick a random greeting",
				Schema:      `{"math": {"setVar": "greeting", "fn": "random", "data": ["Hello!", "Hi there!", "Welcome!"]}}`,
			},
			"round": {
				Description: "Round a numeric variable",
				Schema:      `{"math": {"setVar": "rounded", "fn": "round", "data": ["${raw_score}"]}}`,
			},
			"max_of_list": {
				Description: "Return the largest of three values",
				Schema:      `{"math": {"setVar": "biggest", "fn": "max", "data": [1, 5, 3]}}`,
			},
		},
	}
}

func (httpRequestOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Makes an HTTP request to an external API and stores response fields in variables.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"url": {
				Type:        "string",
				Required:    true,
				Description: "Request URL. Supports ${variables}.",
			},
			"method": {
				Type:        "string",
				Default:     "GET",
				Description: "HTTP method: GET, POST, PUT, PATCH, DELETE.",
			},
			"headers": {
				Type:        "object",
				Description: "HTTP headers as key→value map.",
			},
			"data": {
				Type:        "string",
				Description: "Request body (JSON string). Supports ${variables}.",
			},
			"timeout": {
				Type:        "integer",
				Default:     1000,
				Description: "Request timeout in milliseconds.",
			},
			"responseCode": {
				Type:        "string",
				Description: "Variable name to store the HTTP status code.",
			},
			"parser": {
				Type:        "string",
				Description: "Force response content-type: application/json, application/xml, text/plain. Auto-detected if omitted.",
			},
			"exportVariables": {
				Type:        "object",
				Description: "Map of flow_variable → JSON path in the response body. E.g. {\"client_name\": \"data.name\"}.",
			},
			"insecureSkipVerify": {
				Type:        "boolean",
				Default:     false,
				Description: "Skip TLS certificate verification.",
			},
			"exportCookie": {
				Type:        "string",
				Description: "Variable name to store the response Set-Cookie header.",
			},
		},
		Examples: map[string]ops.Example{
			"crm_lookup": {
				Description: "Look up a client by caller ID and store name + ID",
				Schema: `{"httpRequest": {
  "url": "https://api.example.com/clients",
  "method": "POST",
  "headers": {"Content-Type": "application/json", "Authorization": "Bearer ${crm_token}"},
  "data": "{\"phone\": \"${caller_id_number}\"}",
  "timeout": 5000,
  "responseCode": "crm_status",
  "parser": "application/json",
  "exportVariables": {
    "client_name": "data.name",
    "client_id":   "data.id",
    "vip_status":  "data.is_vip"
  }
}}`,
			},
			"simple_get": {
				Description: "Simple GET with status code capture",
				Schema: `{"httpRequest": {
  "url": "https://api.example.com/status/${session_id}",
  "responseCode": "api_code",
  "exportVariables": {"result_status": "status"}
}}`,
			},
		},
	}
}

func (cacheOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Read from or write to Redis cache. Useful for sharing data between flows " +
			"or storing pre-computed values (e.g. AI suggestions) keyed by caller/session ID.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      false,
		Args: map[string]ops.ArgDoc{
			"type": {
				Type:        "string",
				Required:    true,
				Description: "Cache backend. Currently only 'redis' is supported.",
			},
			"action": {
				Type:        "string",
				Required:    true,
				Description: "Operation: get, set, delete.",
			},
			"get": {
				Type:        "object",
				Description: "Used when action=get. Map of {flow_variable: cache_key}. Reads cache_key and stores result in flow_variable.",
			},
			"set": {
				Type:        "object",
				Description: "Used when action=set. Contains 'data' ({cache_key: value} map) and 'ttl' (seconds as string).",
			},
			"delete": {
				Type:        "object",
				Description: "Used when action=delete. Contains 'keys' array of cache keys to remove.",
			},
		},
		Notes: []string{
			"Cache is scoped per Webitel domain — keys are isolated per tenant.",
			"Always set a TTL to avoid stale data; omitting TTL may result in indefinite storage.",
		},
		Examples: map[string]ops.Example{
			"get": {
				Description: "Read a cached AI suggestion into a flow variable",
				Schema: `{"cache": {
  "action": "get",
  "type": "redis",
  "get": {"ai_suggestion": "${caller_id_number}"}
}}`,
			},
			"set": {
				Description: "Store a value in cache for 1 hour",
				Schema: `{"cache": {
  "action": "set",
  "type": "redis",
  "set": {
    "data": {"customer_status": "${crm_status}"},
    "ttl": "3600"
  }
}}`,
			},
			"delete": {
				Description: "Remove a cached key",
				Schema: `{"cache": {
  "action": "delete",
  "type": "redis",
  "delete": {"keys": ["customer_status"]}
}}`,
			},
		},
	}
}

func (jsOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Executes inline JavaScript (max 1 second timeout). Stores the last expression value in setVar.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"data": {
				Type:     "string",
				Required: true,
				Description: "JS code. ${variables} are pre-substituted as string values before execution. " +
					"Must end with a value expression (not a statement). " +
					"Has access to LocalDate() helper (returns current date as JS Date object).",
			},
			"setVar": {
				Type:        "string",
				Required:    true,
				Description: "Variable to store the script's return value.",
			},
		},
		Notes: []string{
			"${var} substitution happens before JS parsing — wrap numeric vars in parseFloat/parseInt if doing arithmetic.",
			"Execution timeout is 1 second — avoid heavy computation.",
		},
		Examples: map[string]ops.Example{
			"concatenate": {
				Description: "Build a greeting string",
				Schema: `{"js": {
  "data": "'Hello, ' + '${client_name}' + '! Your number is ${caller_id_number}'",
  "setVar": "welcome_msg"
}}`,
			},
			"date_format": {
				Description: "Get today's date as YYYY-MM-DD",
				Schema: `{"js": {
  "data": "var d = LocalDate(); d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0')",
  "setVar": "today"
}}`,
			},
			"conditional_value": {
				Description: "Ternary based on session variable",
				Schema: `{"js": {
  "data": "'${vip_status}' === 'true' ? 'VIP' : 'Standard'",
  "setVar": "client_tier"
}}`,
			},
		},
	}
}

func (o *classifierOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Keyword/phrase classifier. Matches text input against category clusters and stores the matched " +
			"category name in a variable. Fast, deterministic, no LLM call. " +
			"Typically used after STT to route by intent. Returns empty string if no cluster matches.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"input": {
				Type:        "string",
				Required:    true,
				Description: "Text to classify. Usually an STT result variable, e.g. ${google_transcript}.",
			},
			"set": {
				Type:        "string",
				Required:    true,
				Description: "Variable name to store the matched category name. Empty string if no match.",
			},
			"cluster": {
				Type:        "object",
				Required:    true,
				Description: "Category definitions: {category_name: [phrases]}.",
			},
			"matchType": {
				Type:        "string",
				Default:     "full",
				Description: "full — whole-word / exact phrase match. part — substring match (phrase appears anywhere in input).",
			},
			"phraseSearch": {
				Type:        "boolean",
				Default:     false,
				Description: "When true, matches multi-word phrases as a whole rather than individual tokens.",
			},
		},
		Notes: []string{
			"Result is empty string when no cluster matches — always handle the no-match case in switch/if.",
			"Prefer matchType=part for open-ended STT transcripts; matchType=full for tightly controlled yes/no prompts.",
			"Classifier is lightweight (no LLM). Combine with httpRequest→GPT as a fallback for ambiguous input.",
		},
		Examples: map[string]ops.Example{
			"intent_routing": {
				Description: "Classify STT transcript into intent buckets (substring match)",
				Schema: `{"classifier": {
  "input": "${google_transcript}",
  "matchType": "part",
  "phraseSearch": true,
  "set": "intent",
  "cluster": {
    "billing":      ["invoice", "payment", "bill"],
    "tech_support": ["not working", "problem", "error"],
    "general":      ["question", "info", "help"]
  }
}}`,
			},
			"yes_no": {
				Description: "Simple yes/no detection (exact match)",
				Schema: `{"classifier": {
  "input": "${google_transcript}",
  "matchType": "full",
  "phraseSearch": true,
  "set": "answer",
  "cluster": {
    "yes": ["yes", "sure", "correct", "ok"],
    "no":  ["no", "wrong number", "not me"]
  }
}}`,
			},
		},
	}
}
