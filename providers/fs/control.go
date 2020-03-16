package fs

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"strings"
)

const (
	HANGUP_NORMAL_TEMPORARY_FAILURE = "NORMAL_TEMPORARY_FAILURE"
	HANGUP_NO_ROUTE_DESTINATION     = "NO_ROUTE_DESTINATION"
)

func (c *Connection) Answer() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "answer", "")
}

func (c *Connection) PreAnswer() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "pre_answer", "")
}

func (c *Connection) RingReady() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "ring_ready", "")
}

func (c *Connection) Hangup(cause string) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", cause)
}

func (c *Connection) HangupNoRoute() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", HANGUP_NO_ROUTE_DESTINATION)
}

func (c *Connection) HangupAppErr() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", HANGUP_NORMAL_TEMPORARY_FAILURE)
}

func (c *Connection) Sleep(timeout int) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "sleep", fmt.Sprintf("%d", timeout))
}

func (c *Connection) Bridge(call model.Call, strategy string, vars map[string]string, endpoints []*model.Endpoint) (model.Response, *model.AppError) {
	var dialString, separator string

	if strategy == "failover" {
		separator = "|"
	} else if strategy != "" && strategy != "multiple" {
		separator = ":_:"
	} else {
		separator = ","
	}

	var from string

	from = fmt.Sprintf("sip_h_X-Webitel-Origin=flow,wbt_parent_id=%s,wbt_from_type=%s,wbt_from_id=%s,wbt_destination=%s",
		call.Id(), call.From().Type, call.From().Id, call.Destination())

	from += fmt.Sprintf(",effective_caller_id_name='%s',effective_caller_id_number='%s'", call.From().Name, call.From().Number)

	dialString += "<sip_route_uri=sip:$${outbound_sip_proxy}," + from
	for key, val := range vars {
		dialString += fmt.Sprintf(",'%s'='%s'", key, val)
	}
	dialString += ">"

	end := make([]string, 0, len(endpoints))

	for _, e := range endpoints {
		switch e.TypeName {
		case "sipGateway":
			if e == nil || e.Destination == nil {
				end = append(end, "error/UNALLOCATED_NUMBER")
			} else if e.Dnd != nil && *e.Dnd {
				end = append(end, "error/GATEWAY_DOWN")
			} else {
				end = append(end, fmt.Sprintf("[%s]sofia/sip/%s", e.ToStringVariables(), *e.Destination))
			}
		case "user":
			if e == nil || e.Destination == nil {
				end = append(end, "error/UNALLOCATED_NUMBER")
			} else if e.Dnd != nil && *e.Dnd {
				end = append(end, "error/USER_BUSY")
			} else {
				end = append(end, fmt.Sprintf("[%s]sofia/sip/%s@%s", e.ToStringVariables(), *e.Destination, call.DomainName()))
			}
		}
	}

	dialString += strings.Join(end, separator)

	return c.Execute(context.Background(), "bridge", dialString)
}

func (c *Connection) Echo(delay int) (model.Response, *model.AppError) {
	if delay == 0 {
		return c.Execute(context.Background(), "echo", "")
	} else {
		return c.Execute(context.Background(), "delay_echo", delay)
	}
}
