package queue

import "github.com/webitel/flow_manager/internal/runtime/ops"

func (o *getQueueMetricsOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Returns a statistical metric for a queue (e.g. service level, handle time).",
		AvailableIn: []string{"voice", "chat", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"queue":       {Type: "object", Required: true, Description: "Queue to query {id, name}."},
			"bucket":      {Type: "object", Description: "Optional bucket filter {id, name}."},
			"set":         {Type: "string", Required: true, Description: "Variable name to store the metric value."},
			"calls":       {Type: "string", Description: `"complete" to query historical completed calls.`},
			"metric":      {Type: "string", Description: "Metric name (e.g. sl, aht, abandonment_rate)."},
			"field":       {Type: "string", Description: "Sub-field for composite metrics."},
			"lastMinutes": {Type: "integer", Description: "Rolling window in minutes for historical queries."},
			"slSec":       {Type: "integer", Description: "Service-level threshold in seconds."},
		},
		Examples: map[string]ops.Example{
			"service_level": {
				Description: "Get service level for queue 3 over the last 60 minutes",
				Schema:      `{"getQueueMetrics": {"queue": {"id": 3}, "calls": "complete", "metric": "sl", "slSec": 20, "lastMinutes": 60, "set": "queue_sl"}}`,
			},
		},
	}
}

func (o *getQueueInfoOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Retrieves queue properties and maps them to flow variables.",
		AvailableIn: []string{"voice", "chat", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"queue": {Type: "object", Required: true, Description: "Queue to query {id, name}."},
			"set":   {Type: "object", Required: true, Description: "Map of queue column → variable name."},
		},
		Examples: map[string]ops.Example{
			"get_queue_info": {
				Description: "Get queue type and store it",
				Schema:      `{"getQueueInfo": {"queue": {"id": 5}, "set": {"type": "queueType"}}}`,
			},
		},
	}
}

func (o *getQueueAgentsOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Retrieves agent information for a queue and maps it to flow variables.",
		AvailableIn: []string{"voice", "chat", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"queue":   {Type: "object", Required: true, Description: "Queue to query {id} (id is required)."},
			"channel": {Type: "string", Description: "Channel filter (e.g. voice, chat)."},
			"set":     {Type: "object", Description: "Map of agent field → variable name."},
		},
		Examples: map[string]ops.Example{
			"get_agents": {
				Description: "Get agents for queue 4",
				Schema:      `{"getQueueAgents": {"queue": {"id": 4}, "channel": "voice", "set": {"agent": "agentId"}}}`,
			},
		},
	}
}
