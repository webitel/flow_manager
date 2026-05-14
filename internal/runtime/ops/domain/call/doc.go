package call

import (
	"context"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// ── ringReady / preAnswer / answer ────────────────────────────────────────────
// Named wrapper types so each can carry its own Doc().

type ringReadyOp struct{}

func (ringReadyOp) Kind() ops.OpKind { return ops.OpKindSync }
func (ringReadyOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	return syncOp(ringReadyFn).Execute(ctx, in)
}
func (ringReadyOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Sends SIP 180 Ringing to the caller. Does NOT answer the call — caller hears standard ringback tone. " +
			"Use before preAnswer or answer to signal the call is being processed.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Notes: []string{
			"Sends SIP 180 — no media channel opened yet.",
			"Prevents PBX/carrier from playing their own ringback tone.",
			"Typical sequence: ringReady → preAnswer → playback → answer.",
		},
		Examples: map[string]ops.Example{
			"basic": {Schema: `{"ringReady": ""}`},
		},
	}
}

type preAnswerOp struct{}

func (preAnswerOp) Kind() ops.OpKind { return ops.OpKindSync }
func (preAnswerOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	return syncOp(preAnswerFn).Execute(ctx, in)
}
func (preAnswerOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Opens early media (SIP 183 Session Progress) without fully answering the call. " +
			"Allows playing audio or collecting DTMF before the call is billed.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Notes: []string{
			"Sends SIP 183 — media channel opens but call is not yet answered (no billing yet).",
			"Follow with answer when you want to fully accept the call.",
			"Common pattern: ringReady → preAnswer → playback → answer.",
		},
		Examples: map[string]ops.Example{
			"basic": {Schema: `{"preAnswer": ""}`},
		},
	}
}

type answerOp struct{}

func (answerOp) Kind() ops.OpKind { return ops.OpKindSync }
func (answerOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	return syncOp(answerFn).Execute(ctx, in)
}
func (answerOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Answers the inbound call. Required before playback, bridge, or any op that needs a media channel.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Examples: map[string]ops.Example{
			"basic": {Schema: `{"answer": ""}`},
		},
	}
}

// Ensure the new named types satisfy ops.Op.
var _ ops.Op = ringReadyOp{}
var _ ops.Op = preAnswerOp{}
var _ ops.Op = answerOp{}

// ── hangup ────────────────────────────────────────────────────────────────────

func (hangupOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Terminates the call with an ITU-T Q.850 cause code string.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"hangup": {
				Type: "string",
				Description: "Cause code. Common values: " +
					"NORMAL_CLEARING (expected end), " +
					"USER_BUSY (SIP 486), " +
					"NO_ANSWER (SIP 480), " +
					"CALL_REJECTED (SIP 603 — intentional refusal), " +
					"NO_ROUTE_DESTINATION (SIP 404). " +
					"Empty string = system default.",
			},
		},
		Notes: []string{
			"Use NORMAL_CLEARING for all expected end-of-flow terminations.",
			"Use CALL_REJECTED when the flow intentionally refuses the call (blacklist, out-of-hours with no callback).",
			"ABANDONED is set automatically when a caller leaves a queue — do not set it manually.",
		},
		Examples: map[string]ops.Example{
			"normal": {
				Description: "End call normally",
				Schema:      `{"hangup": "NORMAL_CLEARING"}`,
			},
			"reject": {
				Description: "Reject call (blacklist / out-of-hours)",
				Schema:      `{"hangup": "CALL_REJECTED"}`,
			},
		},
	}
}

// ── playback ──────────────────────────────────────────────────────────────────

func (o *playbackOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Plays audio file(s). Optionally waits for DTMF input (IVR) or captures voice via STT.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"files": {
				Type:     "array",
				Required: true,
				Description: "Audio files to play. Each item: {type, id, name}. " +
					"type: mp3, wav, ogg, silence (name=duration ms), tone, local_stream. " +
					"id: file ID in Webitel media storage. name: filename or silence duration in ms.",
			},
			"terminator": {
				Type:        "string",
				Default:     "#",
				Description: "DTMF key that interrupts playback.",
			},
			"getDigits": {
				Type: "object",
				Description: "If set — waits for DTMF input after playback (IVR menu). " +
					"Fields: setVar (required, variable to store pressed digits), " +
					"min (default 1), max (default 1), tries (default 3), " +
					"timeout (ms), regexp (validation pattern, e.g. ^[1-5]$), flushDTMF (bool).",
			},
			"getSpeech": {
				Type: "object",
				Description: "If set — captures caller speech (STT) while/after playback. " +
					"Primary STT mechanism for voice bots. " +
					"Key fields: setVar (required, stores transcript), lang (required, BCP-47 e.g. uk-UA), " +
					"recognizer (Google STT path, e.g. 'projects/${p}/locations/eu'), " +
					"uri (Google STT endpoint), model (short|long|telephony|medical_dictation, default short), " +
					"timeout (ms, default 5000), vadTimeout (silence timeout ms), " +
					"singleUtterance (bool), interim (bool), alternativeLang (array), " +
					"version ('v2'=Google v2, 'v3'=Scribe/AI bridge). " +
					"v3 requires: profile.id instead of recognizer/uri.",
			},
		},
		Notes: []string{
			"getSpeech and getDigits are mutually exclusive — use one per playback step.",
			"For STT: use files: [{type: silence, name: '10'}] to trigger the listener before the caller speaks.",
			"version=v2 uses Google STT directly (recognizer + uri required); version=v3 uses AI bridge (profile.id required).",
			"Transcript is stored in the variable specified by getSpeech.setVar (commonly google_transcript).",
			"After getSpeech: use classifier or switch on the transcript variable to route by intent.",
		},
		Examples: map[string]ops.Example{
			"simple": {
				Description: "Play a file",
				Schema:      `{"playback": {"files": [{"type": "mp3", "id": "<FILE_ID>"}]}}`,
			},
			"silence_pause": {
				Description: "2-second silent pause",
				Schema:      `{"playback": {"files": [{"type": "silence", "name": "2000"}]}}`,
			},
			"ivr_menu": {
				Description: "IVR digit capture (1-5)",
				Schema: `{"playback": {
  "files": [{"type": "mp3", "id": "<MENU_FILE_ID>"}],
  "getDigits": {
    "setVar": "ivr_choice",
    "min": 1, "max": 1,
    "tries": 3, "timeout": 5000,
    "regexp": "^[1-5]$"
  }
}}`,
			},
			"stt_capture": {
				Description: "Capture voice response after a question",
				Schema: `{"playback": {"files": [{"name": "<QUESTION_FILE>.wav"}]}},
{"playback": {
  "files": [{"type": "silence", "name": "10"}],
  "getSpeech": {
    "setVar": "google_transcript",
    "lang": "uk-UA",
    "recognizer": "projects/${stt_project}/locations/eu",
    "uri": "eu-speech.googleapis.com",
    "model": "short",
    "timeout": 9000,
    "vadTimeout": "${vadTimeout}",
    "interim": true,
    "breakFinalOnTimeout": true
  }
}}`,
			},
		},
	}
}

// ── bridge ────────────────────────────────────────────────────────────────────

func (o *bridgeOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Connects the current call (A-leg) to another party (B-leg) via SIP endpoint or gateway. " +
			"After the bridge ends, execution continues with the next op in the flat sequence.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"endpoints": {
				Type:     "array",
				Required: true,
				Description: "B-leg endpoints to dial. Each endpoint: " +
					"type (required: 'user' or 'gateway'), " +
					"extension (for type=user), " +
					"dialString (phone number for type=gateway), " +
					"gateway: {id|name} (gateway selector — always nested, never on root), " +
					"parameters: {leg_timeout, leg_delay_start, origination_caller_id_number, …} (per-endpoint SIP vars), " +
					"idle (bool, type=user only — only bridge if agent is idle).",
			},
			"strategy": {
				Type:        "string",
				Description: "failover — try endpoints in order until one answers. multiply — dial all simultaneously, first to answer wins. Omit for single endpoint.",
			},
			"parameters": {
				Type:        "object",
				Description: "Global SIP channel vars applied to the entire bridge. NOT for continue_on_fail or hangup_after_bridge — set those via {set} BEFORE bridge.",
			},
		},
		Notes: []string{
			"Set continue_on_fail and hangup_after_bridge via {\"set\": {\"continue_on_fail\": \"true\"}} BEFORE bridge — never inside bridge args.",
			"Ring timeout is endpoint.parameters.leg_timeout (string, e.g. '15') — NOT a top-level timeout field.",
			"gateway.id must be NESTED: {\"gateway\": {\"id\": 5}} — never {\"gatewayId\": 5} on the endpoint root.",
			"bridge does NOT set ${_result} — to detect failure use continue_on_fail=true and add fallback steps after bridge.",
			"There is NO 'timers' field in bridge — periodic actions while waiting belong in joinQueue.timers.",
			"Valid endpoint types: user, gateway. Never type='sip'.",
		},
		Examples: map[string]ops.Example{
			"internal_user": {
				Description: "Bridge to internal operator by extension",
				Schema: `{"set": {"continue_on_fail": "true", "hangup_after_bridge": "true"}},
{"bridge": {
  "endpoints": [{"type": "user", "extension": "<EXTENSION>"}]
}}`,
			},
			"gateway_dialstring": {
				Description: "Bridge to mobile via gateway name (dialString format)",
				Schema: `{"set": {"continue_on_fail": "true", "hangup_after_bridge": "true"}},
{"bridge": {
  "endpoints": [{
    "type": "gateway",
    "dialString": "<MOBILE_NUMBER>",
    "name": "<GATEWAY_NAME>",
    "parameters": {
      "origination_caller_id_number": "<CALLER_ID>",
      "leg_timeout": "15"
    }
  }]
}},
{"playback": {"files": [{"id": "<NO_ANSWER_FILE_ID>"}]}},
{"joinQueue": {"queue": {"id": "<QUEUE_ID>"}}}`,
			},
			"gateway_by_id": {
				Description: "Bridge to mobile via gateway ID (nested gateway.id format)",
				Schema: `{"set": {"continue_on_fail": "true", "hangup_after_bridge": "true"}},
{"bridge": {
  "endpoints": [{
    "type": "gateway",
    "dialString": "<MOBILE_NUMBER>",
    "gateway": {"id": "<GATEWAY_ID>"},
    "parameters": {
      "origination_caller_id_number": "<CALLER_ID>",
      "leg_timeout": "15"
    }
  }]
}}`,
			},
			"failover_users": {
				Description: "Try two operators in order (failover strategy)",
				Schema: `{"bridge": {
  "strategy": "failover",
  "endpoints": [
    {"type": "user", "extension": "101", "parameters": {"leg_timeout": "20"}},
    {"type": "user", "extension": "102", "parameters": {"leg_timeout": "20"}}
  ]
}}`,
			},
		},
	}
}

// ── joinQueue ─────────────────────────────────────────────────────────────────

func (o *joinQueueOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Puts a call into an inbound queue. Waits until an agent picks it up or the caller abandons.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"queue": {
				Type:        "object",
				Required:    true,
				Description: "Queue to join. {id: int} or {name: string}.",
			},
			"priority": {
				Type:        "integer",
				Default:     0,
				Description: "Member priority — higher value = higher priority.",
			},
			"bucket": {
				Type:        "object",
				Description: "Member segmentation bucket. {id: int}.",
			},
			"agent": {
				Type:        "object",
				Description: "Preferred (sticky) agent. {id: int} or {extension: string}.",
			},
			"ringtone": {
				Type:        "object",
				Description: "Hold music played while waiting. {id: int} or {name: string}.",
			},
			"timers": {
				Type: "array",
				Description: "Actions to execute periodically while waiting. " +
					"Each timer: {interval (seconds), tries (repetitions), offset (interval growth per fire in seconds), actions (app array)}.",
			},
			"transferAfterBridge": {
				Type:        "object",
				Description: "Transfer to another schema after agent bridge ends. {id: int} — schema ID.",
			},
		},
		Examples: map[string]ops.Example{
			"simple": {
				Description: "Basic queue with hold music",
				Schema: `{"joinQueue": {
  "queue": {"id": "<QUEUE_ID>"},
  "ringtone": {"id": "<MOH_FILE_ID>"}
}}`,
			},
			"with_timers": {
				Description: "Queue with periodic promo playback every 60s (up to 20 times)",
				Schema: `{"joinQueue": {
  "queue": {"id": "<QUEUE_ID>"},
  "ringtone": {"id": "<MOH_FILE_ID>"},
  "priority": 100,
  "timers": [{
    "interval": 60,
    "tries": 20,
    "actions": [{"playback": {"files": [{"name": "<PROMO_FILE>"}]}}]
  }]
}}`,
			},
		},
	}
}

// ── recordSession ─────────────────────────────────────────────────────────────

func (recordSessionOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Records the call audio to a file. Typically placed immediately after answer.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"action": {
				Type:        "string",
				Default:     "start",
				Description: "start — begin recording. stop — end recording.",
			},
			"name": {
				Type:        "string",
				Default:     "${strepoch()}_${caller_id_number}_${destination_number}",
				Description: "Recording filename (without extension). Supports ${variables}.",
			},
			"type": {
				Type:        "string",
				Default:     "mp3",
				Description: "Audio format: mp3, wav, ogg.",
			},
			"stereo": {
				Type:        "boolean",
				Default:     false,
				Description: "Separate channels for caller and agent.",
			},
			"bridged": {
				Type:        "boolean",
				Default:     false,
				Description: "Record only during active bridge (not hold/IVR).",
			},
			"minSec": {
				Type:        "integer",
				Default:     2,
				Description: "Minimum recording duration in seconds to save. Shorter recordings are discarded.",
			},
			"followTransfer": {
				Type:        "boolean",
				Default:     true,
				Description: "Continue recording after call transfer.",
			},
		},
		Examples: map[string]ops.Example{
			"standard": {
				Description: "Start stereo MP3 recording after answer",
				Schema: `{"answer": ""},
{"recordSession": {
  "action": "start",
  "type": "mp3",
  "stereo": false,
  "minSec": 2,
  "followTransfer": true
}}`,
			},
		},
	}
}

