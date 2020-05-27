package sqlstore

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlCallStore struct {
	SqlStore
}

func NewSqlCallStore(sqlStore SqlStore) store.CallStore {
	st := &SqlCallStore{sqlStore}
	return st
}

func (s SqlCallStore) Save(call *model.CallActionRinging) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, direction, destination, parent_id, "timestamp", state, app_id, from_type, from_name,
                      from_number, from_id, to_type, to_name, to_number, to_id, payload, domain_id, created_at, gateway_id, user_id, queue_id, agent_id, team_id, attempt_id, member_id)
values (:Id, :Direction, :Destination, :ParentId, to_timestamp(:Timestamp::double precision /1000), :State, :AppId, :FromType, :FromName, :FromNumber, :FromId,
        :ToType, :ToName, :ToNumber, :ToId, :Payload, :DomainId, to_timestamp(:CreatedAt::double precision /1000), :GatewayId, :UserId, :QueueId, :AgentId, :TeamId, :AttemptId, :MemberId)
on conflict (id)
    do update set
		created_at = EXCLUDED.created_at,
		direction = EXCLUDED.direction,
		destination = EXCLUDED.destination,
		parent_id = EXCLUDED.parent_id,
		from_type = EXCLUDED.from_type,
		from_name = EXCLUDED.from_name,
		from_number = EXCLUDED.from_number,
		from_id = EXCLUDED.from_id,
		to_type = EXCLUDED.to_type,
		to_name = EXCLUDED.to_name,
		to_number = EXCLUDED.to_number,
		to_id = EXCLUDED.to_id,
		gateway_id = EXCLUDED.gateway_id,		
		user_id = EXCLUDED.user_id,		
		payload = EXCLUDED.payload,
		queue_id = EXCLUDED.queue_id,
		agent_id = EXCLUDED.agent_id,
		team_id = EXCLUDED.team_id,
		attempt_id = EXCLUDED.attempt_id,
		member_id = EXCLUDED.member_id
		`, map[string]interface{}{
		"DomainId":    call.DomainId,
		"Id":          call.Id,
		"Direction":   call.Direction,
		"Destination": call.Destination,
		"ParentId":    call.ParentId,
		"Timestamp":   call.Timestamp,
		"State":       call.Event,
		"AppId":       call.AppId,
		"CreatedAt":   call.Timestamp,
		"FromType":    call.GetFrom().GetType(),
		"FromName":    call.GetFrom().GetName(),
		"FromNumber":  call.GetFrom().GetNumber(),
		"FromId":      call.GetFrom().GetId(),

		"ToType":    call.GetTo().GetType(),
		"ToName":    call.GetTo().GetName(),
		"ToNumber":  call.GetTo().GetNumber(),
		"ToId":      call.GetTo().GetId(),
		"GatewayId": call.GatewayId,
		"UserId":    call.UserId,
		"QueueId":   call.GetQueueId(),
		"AgentId":   call.GetAgentId(),
		"TeamId":    call.GetTeamId(),
		"AttemptId": call.GetAttemptId(),
		"MemberId":  call.GetMemberIdId(),
		"Payload":   nil,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.Save", "store.sql_call.save.error", nil,
			fmt.Sprintf("Id=%v %v", call.Id, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

// TODO race... fix remove
func (s SqlCallStore) SetState(call *model.CallAction) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_calls(id, state, timestamp, app_id, domain_id)
values (:Id, :State, to_timestamp(:Timestamp::double precision /1000), :AppId, :DomainId)
on conflict (id) where timestamp < to_timestamp(:Timestamp::double precision /1000) and cause isnull
    do update set 
      state = EXCLUDED.state,
      timestamp = EXCLUDED.timestamp`, map[string]interface{}{
		"Id":        call.Id,
		"State":     call.Event,
		"Timestamp": call.Timestamp,
		"AppId":     call.AppId,
		"DomainId":  call.DomainId,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetState", "store.sql_call.set_state.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetHangup(call *model.CallActionHangup) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, state, timestamp, app_id, domain_id, cause, sip_code, payload)
values (:Id, :State, to_timestamp(:Timestamp::double precision /1000), :AppId, :DomainId, :Cause, :SipCode, :Variables::json)
on conflict (id) where timestamp <= to_timestamp(:Timestamp::double precision /1000)
    do update set
      state = EXCLUDED.state,
      cause = EXCLUDED.cause,
      sip_code = EXCLUDED.sip_code,
      payload = EXCLUDED.payload,
      timestamp = EXCLUDED.timestamp`, map[string]interface{}{
		"Id":        call.Id,
		"State":     call.Event,
		"Timestamp": call.Timestamp,
		"AppId":     call.AppId,
		"DomainId":  call.DomainId,
		"Cause":     call.Cause,
		"SipCode":   call.SipCode,
		"Variables": call.VariablesToJson(),
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetHangup", "store.sql_call.set_state.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetBridged(call *model.CallActionBridge) *model.AppError {
	_, err := s.GetMaster().Exec(`call cc_call_set_bridged(:Id::varchar, :State::varchar, to_timestamp(:Timestamp::double precision /1000), :AppId::varchar,
    :DomainId::int8, :BridgedId::varchar)`, map[string]interface{}{
		"Id":        call.Id,
		"State":     call.Event,
		"Timestamp": call.Timestamp,
		"AppId":     call.AppId,
		"DomainId":  call.DomainId,
		"BridgedId": call.BridgedId,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetBridged", "store.sql_call.set_bridged.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) MoveToHistory() *model.AppError {
	_, err := s.GetMaster().Exec(`
BEGIN;
with c as (
    delete from cc_calls c
	where c.hangup_at < now() - '1 sec'::interval and c.direction notnull
    returning c.created_at, c.id, c.direction, c.destination, c.parent_id, c.app_id, c.from_type, c.from_name, c.from_number, c.from_id,
       c.to_type, c.to_name, c.to_number, c.to_id, c.payload, c.domain_id,
       c.answered_at, c.bridged_at, c.hangup_at, c.hold_sec, c.cause, c.sip_code, c.bridged_id, c.gateway_id, c.user_id, c.queue_id, c.team_id, c.agent_id, c.attempt_id, c.member_id
)
insert into cc_calls_history (created_at, id, direction, destination, parent_id, app_id, from_type, from_name, from_number, from_id,
                              to_type, to_name, to_number, to_id, payload, domain_id, answered_at, bridged_at, hangup_at, hold_sec, cause, sip_code, bridged_id,
							gateway_id, user_id, queue_id, team_id, agent_id, attempt_id, member_id)
select c.created_at created_at, c.id, c.direction, c.destination, c.parent_id, c.app_id, c.from_type, c.from_name, c.from_number, c.from_id,
       c.to_type, c.to_name, c.to_number, c.to_id, c.payload, c.domain_id,
       c.answered_at, c.bridged_at, c.hangup_at, c.hold_sec, c.cause, c.sip_code, c.bridged_id, c.gateway_id, c.user_id, c.queue_id, c.team_id, c.agent_id, c.attempt_id, c.member_id
from c;
COMMIT;`)
	if err != nil {
		return model.NewAppError("SqlCallStore.MoveToHistory", "store.sql_call.move_to_store.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) AddMemberToQueueQueue(domainId int64, queueId int, number, name string, typeId, holdSec int, variables map[string]string) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_member(queue_id, communications, name, variables, last_hangup_at, domain_id)
select q.id queue_id, json_build_array(jsonb_build_object('destination', :Number::varchar) ||
                      jsonb_build_object('type', jsonb_build_object('id', :TypeId::int4))),
       :Name::varchar,
       :Variables::jsonb vars,
       (extract(epoch from now() + (:HoldSec::int4 || ' sec')::interval) * 1000)::int8 lh,
       q.domain_id
from cc_queue q
where q.id = :QueueId::int4 and q.domain_id = :DomainId::int8`, map[string]interface{}{
		"DomainId":  domainId,
		"QueueId":   queueId,
		"Number":    number,
		"TypeId":    typeId,
		"Name":      name,
		"HoldSec":   holdSec,
		"Variables": model.MapStringToJson(variables),
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.AddMemberToQueueQueue", "store.sql_call.callback_queue.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}
