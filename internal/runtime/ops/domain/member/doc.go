package member

import "github.com/webitel/flow_manager/internal/runtime/ops"

// Documenter interface is implemented per op struct; collected by cmd/docgen.

func (o *ccPositionOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Returns the current position of the active call in the queue.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"set": {Type: "string", Required: true, Description: "Variable name to store the queue position."},
		},
		Examples: map[string]ops.Example{
			"get_position": {
				Description: "Store call queue position",
				Schema:      `{"ccPosition": {"set": "pos"}}`,
			},
		},
	}
}

func (o *memberInfoOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Retrieves CC member properties and maps them to flow variables.",
		AvailableIn: []string{"voice", "chat", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"member": {Type: "object", Required: true, Description: "Search criteria to identify the member."},
			"set":    {Type: "object", Required: true, Description: "Map of member field → variable name."},
		},
		Examples: map[string]ops.Example{
			"get_member_info": {
				Description: "Get member name and store it",
				Schema:      `{"memberInfo": {"member": {"id": "${member_id}"}, "set": {"name": "memberName"}}}`,
			},
		},
	}
}

func (o *patchMembersOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Updates one or more CC members matching the search criteria.",
		AvailableIn: []string{"voice", "chat", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"member": {Type: "object", Required: true, Description: "Search criteria to identify target members."},
			"patch":  {Type: "object", Required: true, Description: "Fields to update on the matched members."},
		},
		Examples: map[string]ops.Example{
			"patch_member": {
				Description: "Set stop cause on a member",
				Schema:      `{"patchMembers": {"member": {"id": "${member_id}"}, "patch": {"stop_cause": "abandoned"}}}`,
			},
		},
	}
}

func (o *callbackQueueOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Creates a callback member in the specified queue.",
		AvailableIn: []string{"voice", "chat", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"queue_id":      {Type: "integer", Description: "Target queue ID (deprecated; prefer queue.id)."},
			"holdSec":       {Type: "integer", Description: "Hold time in seconds before the callback attempt."},
			"name":          {Type: "string", Description: "Member display name."},
			"communication": {Type: "object", Required: true, Description: "Destination and communication type."},
			"queue":         {Type: "object", Description: "Queue reference {id} — takes precedence over queue_id."},
			"variables":     {Type: "object", Description: "Extra key/value variables attached to the member."},
			"expire_at":     {Type: "integer", Description: "Unix timestamp (ms) after which the member expires."},
			"stop_cause":    {Type: "string", Description: "Pre-set stop cause for the callback attempt."},
		},
		Examples: map[string]ops.Example{
			"create_callback": {
				Description: "Create a callback member in queue 7",
				Schema:      `{"callbackQueue": {"queue": {"id": 7}, "holdSec": 0, "communication": {"destination": "${caller_id_number}"}}}`,
			},
		},
	}
}

func (o *ewtOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Calculates the Estimated Wait Time (EWT) for the given queues and buckets.",
		AvailableIn: []string{"voice"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"setVar":     {Type: "string", Required: true, Description: "Variable name to store the EWT value (in seconds)."},
			"queue_ids":  {Type: "array", Description: "Queue IDs to include in the EWT calculation."},
			"bucket_ids": {Type: "array", Description: "Bucket IDs to include in the EWT calculation."},
			"min":        {Type: "integer", Default: 60, Description: "Minimum sample window in seconds."},
			"strategy":   {Type: "string", Description: "EWT strategy (reserved for future use)."},
		},
		Examples: map[string]ops.Example{
			"ewt": {
				Description: "Get EWT for queue 5",
				Schema:      `{"ewt": {"setVar": "waitTime", "queue_ids": [5]}}`,
			},
		},
	}
}
