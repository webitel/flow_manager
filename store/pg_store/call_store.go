package sqlstore

import (
	"fmt"
	"github.com/lib/pq"
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
	_, err := s.GetMaster().Exec(`insert into call_center.cc_calls (id, direction, destination, parent_id, "timestamp", state, app_id, from_type, from_name,
                      from_number, from_id, to_type, to_name, to_number, to_id, payload, domain_id, created_at, gateway_id, user_id, queue_id, agent_id, team_id, 
					  attempt_id, member_id, grantee_id)
values (:Id, :Direction, :Destination, :ParentId, to_timestamp(:Timestamp::double precision /1000), :State, :AppId, :FromType, :FromName, :FromNumber, :FromId,
        :ToType, :ToName, :ToNumber, :ToId, :Payload, :DomainId, to_timestamp(:CreatedAt::double precision /1000), :GatewayId, :UserId, :QueueId, :AgentId, :TeamId, 
		:AttemptId, :MemberId, :GranteeId)
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
		member_id = EXCLUDED.member_id,
		grantee_Id = EXCLUDED.grantee_Id
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
		"GranteeId": call.GranteeId,
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
	_, err := s.GetMaster().Exec(`insert into call_center.cc_calls(id, state, timestamp, app_id, domain_id)
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
	_, err := s.GetMaster().Exec(`insert into call_center.cc_calls (id, state, timestamp, app_id, domain_id, cause, sip_code, payload, hangup_by, tags, amd_result)
values (:Id, :State, to_timestamp(:Timestamp::double precision /1000), :AppId, :DomainId, :Cause, :SipCode, :Variables::json, :HangupBy, :Tags, :AmdResult)
on conflict (id) where timestamp <= to_timestamp(:Timestamp::double precision / 1000)
    do update set
      state = EXCLUDED.state,
      cause = EXCLUDED.cause,
      sip_code = EXCLUDED.sip_code,
      payload = EXCLUDED.payload,
      hangup_by = EXCLUDED.hangup_by,
	  tags = EXCLUDED.tags,
	  amd_result = EXCLUDED.amd_result,
      timestamp = EXCLUDED.timestamp`, map[string]interface{}{
		"Id":        call.Id,
		"State":     call.Event,
		"Timestamp": call.Timestamp,
		"AppId":     call.AppId,
		"DomainId":  call.DomainId,
		"Cause":     call.Cause,
		"SipCode":   call.SipCode,
		"HangupBy":  call.HangupBy,
		"AmdResult": call.AmdResult,
		"Tags":      pq.Array(call.Tags),
		"Variables": call.VariablesToJson(),
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetHangup", "store.sql_call.set_state.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetBridged(call *model.CallActionBridge) *model.AppError {
	_, err := s.GetMaster().Exec(`call call_center.cc_call_set_bridged(:Id::varchar, :State::varchar, to_timestamp(:Timestamp::double precision /1000), :AppId::varchar,
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
with c as (
    delete from call_center.cc_calls c
	where c.hangup_at < now() - '1 sec'::interval and c.direction notnull
        and not exists(select 1 from cc_member_attempt att where att.id = c.attempt_id)
    returning c.created_at, c.id, c.direction, c.destination, c.parent_id, c.app_id, c.from_type, c.from_name, c.from_number, c.from_id,
       c.to_type, c.to_name, c.to_number, c.to_id, c.payload, c.domain_id,
       c.answered_at, c.bridged_at, c.hangup_at, c.hold_sec, c.cause, c.sip_code, c.bridged_id, c.gateway_id, c.user_id,
	   c.queue_id, c.team_id, c.agent_id, c.attempt_id, c.member_id, c.hangup_by, c.transfer_from, c.transfer_to, c.amd_result, c.amd_duration, c.tags, c.grantee_id
)
insert into call_center.cc_calls_history (created_at, id, direction, destination, parent_id, app_id, from_type, from_name, from_number, from_id,
                              to_type, to_name, to_number, to_id, payload, domain_id, answered_at, bridged_at, hangup_at, hold_sec, cause, sip_code, bridged_id,
							gateway_id, user_id, queue_id, team_id, agent_id, attempt_id, member_id, hangup_by, transfer_from, transfer_to, amd_result, amd_duration, tags, grantee_id)
select c.created_at created_at, c.id, c.direction, c.destination, c.parent_id, c.app_id, c.from_type, c.from_name, c.from_number, c.from_id,
       c.to_type, c.to_name, c.to_number, c.to_id, c.payload, c.domain_id,
       c.answered_at, c.bridged_at, c.hangup_at, c.hold_sec, c.cause, c.sip_code, c.bridged_id, c.gateway_id, c.user_id, c.queue_id, 
		c.team_id, c.agent_id, c.attempt_id, c.member_id, c.hangup_by, c.transfer_from, c.transfer_to, c.amd_result, c.amd_duration, c.tags, c.grantee_id
from c;`)
	if err != nil {
		return model.NewAppError("SqlCallStore.MoveToHistory", "store.sql_call.move_to_store.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) AddMemberToQueueQueue(domainId int64, queueId int, number, name string, typeId, holdSec int, variables map[string]string) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into call_center.cc_member(queue_id, communications, name, variables, last_hangup_at, domain_id)
select q.id queue_id, json_build_array(jsonb_build_object('destination', :Number::varchar) ||
                      jsonb_build_object('type', jsonb_build_object('id', :TypeId::int4))),
       :Name::varchar,
       :Variables::jsonb vars,
       (extract(epoch from now() + (:HoldSec::int4 || ' sec')::interval) * 1000)::int8 lh,
       q.domain_id
from call_center.cc_queue q
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

func (s SqlCallStore) UpdateFrom(id string, name, number *string) *model.AppError {
	_, err := s.GetMaster().Exec(`update cc_calls
set from_number = coalesce(:Number, from_number),
    from_name = coalesce(:Name, from_name)
where id = :Id`, map[string]interface{}{
		"Number": number,
		"Name":   name,
		"Id":     id,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.UpdateFrom", "store.sql_call.update_from.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SaveTranscribe(callId, transcribe string) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_calls_transcribe (call_id, transcribe)
values (:CallId::varchar, :Transcribe::varchar)`, map[string]interface{}{
		"CallId":     callId,
		"Transcribe": transcribe,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SaveTranscribe", "store.sql_call.save_transcribe.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) LastBridgedExtension(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int) (*model.LastBridged, *model.AppError) {
	var res *model.LastBridged
	// fixme extension dialer logic
	err := s.GetReplica().SelectOne(&res, `select coalesce(extension, '') as extension, queue_id, agent_id
from (
         select h.created_at, case when h.direction = 'inbound' or q.type = any(array[4,5]::smallint[]) then h.to_number else h.from_number end as extension, h.queue_id, h.agent_id
         from cc_calls_history h
 			left join cc_queue q on q.id = h.queue_id
         where (h.domain_id = :DomainId and h.created_at > now() - (:Hours::varchar || ' hours')::interval)
		   and (:QueueIds::int[] isnull or (h.queue_id = any(:QueueIds) or h.queue_id isnull))	
           and (
                 (h.domain_id = :DomainId and destination ~~* :Number::varchar)
                 or (h.domain_id = :DomainId and to_number ~~* :Number::varchar)
                 or (h.domain_id = :DomainId and from_number ~~* :Number::varchar)
             )
           and h.parent_id isnull
           and (
                 ((:Dialer::varchar isnull or :Dialer::varchar = 'false') and
                  (:Inbound::varchar isnull or :Inbound::varchar = 'false') and
                  (:Outbound::varchar isnull or :Outbound::varchar = 'false')) or
                 (
                         case
                             when :Dialer::varchar notnull and :Dialer::varchar != 'false' then
                                     h.attempt_id notnull and case :Dialer
                                                                  when 'bridged' then h.bridged_at notnull
                                                                  when 'attempt' then h.bridged_at isnull
                                                                  else true end
                             else false end
                         or case
                                when :Inbound::varchar notnull and :Inbound::varchar != 'false' then
                                        h.direction = 'inbound' and case :Inbound
                                                                        when 'bridged' then h.bridged_at notnull
                                                                        when 'attempt' then h.bridged_at isnull
                                                                        else true end
                                else false end
                         or case
                                when :Outbound::varchar notnull and :Outbound::varchar != 'false' then
                                        h.direction = 'outbound' and case :Outbound
                                                                         when 'bridged' then h.bridged_at notnull
                                                                         when 'attempt' then h.bridged_at isnull
                                                                         else true end
                                else false end
                     )
             )
         order by h.created_at desc
     ) h
order by h.created_at desc
limit 1`, map[string]interface{}{
		"DomainId": domainId,
		"Hours":    hours,
		"Number":   number,
		"Dialer":   dialer,
		"Inbound":  inbound,
		"Outbound": outbound,
		"QueueIds": pq.Array(queueIds),
	})

	if err != nil {
		return nil, model.NewAppError("SqlCallStore.LastBridgedExtension", "store.sql_call.get_last_bridged.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}

func (s SqlCallStore) SetGranteeId(domainId int64, id string, granteeId int64) *model.AppError {
	_, err := s.GetMaster().Exec(`update call_center.cc_calls
set grantee_id = :GranteeId
where domain_id = :DomainId and id = :Id;`, map[string]interface{}{
		"DomainId":  domainId,
		"GranteeId": granteeId,
		"Id":        id,
	})

	if err != nil {
		model.NewAppError("SqlCallStore.SetGranteeId", "store.sql_call.set_grantee.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}
