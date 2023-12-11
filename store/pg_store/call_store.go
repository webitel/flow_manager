package sqlstore

import (
	"fmt"
	"strings"

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
					  attempt_id, member_id, grantee_id, params)
values (:Id, :Direction, :Destination, :ParentId, to_timestamp(:Timestamp::double precision /1000), :State, :AppId, :FromType, :FromName, :FromNumber, :FromId,
        :ToType, :ToName, :ToNumber, :ToId, :Payload, :DomainId, to_timestamp(:CreatedAt::double precision /1000), :GatewayId, :UserId, :QueueId, :AgentId, :TeamId, 
		:AttemptId, :MemberId, :GranteeId, jsonb_build_object('sip_id', :SipId::varchar))
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
		grantee_Id = EXCLUDED.grantee_Id,
		params = EXCLUDED.params
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
		"SipId":     call.SipId,
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
values (:Id::uuid, :State, to_timestamp(:Timestamp::double precision /1000), :AppId, :DomainId)
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
	_, err := s.GetMaster().Exec(`insert into call_center.cc_calls (id, state, timestamp, app_id, domain_id, cause, 
			sip_code, payload, hangup_by, tags, amd_result, params, talk_sec, amd_ai_result, amd_ai_logs, amd_ai_positive)
values (:Id, :State, to_timestamp(:Timestamp::double precision /1000), :AppId, :DomainId, :Cause, 
	:SipCode, :Variables::json, :HangupBy, :Tags, :AmdResult, :Params::jsonb, coalesce(:TalkSec::int, 0), :AmdAiResult, :AmdAiResultLog, :AmdAiPositive)
on conflict (id) where timestamp <= to_timestamp(:Timestamp::double precision / 1000)
    do update set
      state = EXCLUDED.state,
      cause = EXCLUDED.cause,
      sip_code = EXCLUDED.sip_code,
      payload = coalesce(call_center.cc_calls.payload, '{}') || EXCLUDED.payload,
      hangup_by = EXCLUDED.hangup_by,
	  tags = EXCLUDED.tags,
	  amd_result = EXCLUDED.amd_result,
	  params = EXCLUDED.params || call_center.cc_calls.params,
	  talk_sec = EXCLUDED.talk_sec::int,
      timestamp = EXCLUDED.timestamp,
      amd_ai_result = EXCLUDED.amd_ai_result,
      amd_ai_logs = EXCLUDED.amd_ai_logs,
      amd_ai_positive = EXCLUDED.amd_ai_positive
     `, map[string]interface{}{
		"Id":             call.Id,
		"State":          call.Event,
		"Timestamp":      call.Timestamp,
		"AppId":          call.AppId,
		"DomainId":       call.DomainId,
		"Cause":          call.Cause,
		"SipCode":        call.SipCode,
		"HangupBy":       call.HangupBy,
		"AmdResult":      call.AmdResult,
		"TalkSec":        call.TalkSec,
		"Tags":           pq.Array(call.Tags),
		"Variables":      call.VariablesToJson(),
		"Params":         call.Parameters(),
		"AmdAiResult":    call.AmdAiResult,
		"AmdAiResultLog": pq.Array(call.AmdAiResultLog),
		"AmdAiPositive":  call.AmdAiPositive,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetHangup", "store.sql_call.set_state.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetBridged(call *model.CallActionBridge) *model.AppError {
	_, err := s.GetMaster().Exec(`call call_center.cc_call_set_bridged(:Id::uuid, :State::varchar, to_timestamp(:Timestamp::double precision /1000), :AppId::varchar,
    :DomainId::int8, :BridgedId::uuid)`, map[string]interface{}{
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

func (s SqlCallStore) Delete(id string) *model.AppError {
	_, err := s.GetMaster().Exec(`delete from call_center.cc_calls
where id = :Id;`, map[string]interface{}{
		"Id": id,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.Delete", "store.sql_call.delete.error", nil,
			fmt.Sprintf("Id=%v, %v", id, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) MoveToHistory() *model.AppError {
	_, err := s.GetMaster().Exec(`
with del_calls as materialized (
    select *
    from call_center.cc_calls c
        where c.hangup_at < now() - '1 sec'::interval
            and c.direction notnull
            and not exists(select 1 from call_center.cc_calls cc where case when c.parent_id notnull then cc.id = c.parent_id else cc.parent_id = c.id and cc.hangup_at isnull end )
            and not exists(select 1 from call_center.cc_member_attempt att where att.id = c.attempt_id )
    order by c.hangup_at asc
    for update skip locked
    limit 1000
),
dd as (
    delete
    from call_center.cc_calls m
    where m.id in (
        select del_calls.id
        from del_calls
    )
)
insert
into call_center.cc_calls_history (created_at, id, direction, destination, parent_id, app_id, from_type, from_name,
                                   from_number, from_id,
                                   to_type, to_name, to_number, to_id, payload, domain_id, answered_at, bridged_at,
                                   hangup_at, hold_sec, cause, sip_code, bridged_id,
                                   gateway_id, user_id, queue_id, team_id, agent_id, attempt_id, member_id, hangup_by,
                                   transfer_from, transfer_to, amd_result, amd_duration,
                                   tags, grantee_id, "hold", user_ids, agent_ids, gateway_ids, queue_ids, team_ids, params, 
								   blind_transfer, talk_sec, amd_ai_result, amd_ai_logs, amd_ai_positive, contact_id, search_number)
select c.created_at created_at,
       c.id::uuid,
       c.direction,
       c.destination,
       c.parent_id::uuid,
       c.app_id,
       c.from_type,
       c.from_name,
       c.from_number,
       c.from_id,
       c.to_type,
       c.to_name,
       c.to_number,
       c.to_id,
       coalesce(p_vars, '{}'),
       c.domain_id,
       c.answered_at,
       c.bridged_at,
       c.hangup_at,
       c.hold_sec,
       c.cause,
       c.sip_code,
       c.bridged_id::uuid,
       c.gateway_id,
       c.user_id,
       c.queue_id,
       c.team_id,
       c.agent_id,
       c.attempt_id,
       c.member_id,
       c.hangup_by,
       c.transfer_from::uuid,
       c.transfer_to::uuid,
       c.amd_result,
       c.amd_duration,
       c.tags,
       c.grantee_id,
       c.hold,
       c.user_ids,
       c.agent_ids,
       c.gateway_ids,
       c.queue_ids,
       c.team_ids,
	   c.params,
	   c.blind_transfer,
	   c.talk_sec,
	   c.amd_ai_result, 
	   c.amd_ai_logs,
	   c.amd_ai_positive,
	   c.contact_id,
	   c.search_number
from (
         select (t.r).*,
                case when (t.r).agent_id isnull then t.agent_ids else (t.r).agent_id || t.agent_ids end agent_ids,
                case when (t.r).user_id isnull then t.user_ids else (t.r).user_id || t.user_ids end     user_ids,
                case
                    when (t.r).gateway_id isnull then t.gateway_ids
                    else (t.r).gateway_id || t.gateway_ids end                                          gateway_ids,
                case when (t.r).queue_id isnull then t.queue_ids else (t.r).queue_id || t.queue_ids end queue_ids,
                case when (t.r).team_id isnull then t.team_ids else (t.r).team_id || t.team_ids end team_ids,
				coalesce(t.p_vars, '{}') || coalesce((t.r).payload, '{}')  as p_vars,
				t.search_number
         from (
                  select c                                                             r,
                         array_agg(distinct ch.user_id)
                         filter ( where c.parent_id isnull and ch.user_id notnull )    user_ids,
                         array_agg(distinct ch.agent_id)
                         filter ( where c.parent_id isnull and ch.agent_id notnull )   agent_ids,
                         array_agg(distinct ch.queue_id)
                         filter ( where c.parent_id isnull and ch.queue_id notnull )   queue_ids,
                         array_agg(distinct ch.gateway_id)
                         filter ( where c.parent_id isnull and ch.gateway_id notnull ) gateway_ids,
                         array_agg(distinct ch.team_id)
                         filter ( where c.parent_id isnull and ch.team_id notnull ) team_ids,
						 call_center.jsonb_concat_agg(ch.payload) p_vars,
						 string_agg(distinct nums.from_number, '|') filter ( where  nums.from_number != '' and nums.from_number notnull ) search_number
                  from del_calls c
                           left join call_center.cc_calls ch on (ch.parent_id = c.id or (ch.id = c.bridged_id))
						   left join lateral (
										select c.from_number
										union distinct
										select c.to_number
										union distinct
										select c.destination
										union distinct
										select ch.from_number
										union distinct
										select ch.to_number
										union distinct
										select ch.destination
									) nums on true
							  group by 1
						  ) t
     ) c`)
	if err != nil {
		return model.NewAppError("SqlCallStore.MoveToHistory", "store.sql_call.move_to_store.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) UpdateFrom(id string, name, number *string) *model.AppError {
	_, err := s.GetMaster().Exec(`update call_center.cc_calls
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
	_, err := s.GetMaster().Exec(`insert into call_center.cc_calls_transcribe (call_id, transcribe)
values (:CallId::varchar, :Transcribe::varchar)`, map[string]interface{}{
		"CallId":     callId,
		"Transcribe": transcribe,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SaveTranscribe", "store.sql_call.save_transcribe.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) LastBridged(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	f := make([]string, 0)

	for k, v := range mapRes {
		var val = ""
		switch v {
		case "extension":
			val = "extension::varchar as " + pq.QuoteIdentifier(k)
		case "id":
			val = "id::varchar as " + pq.QuoteIdentifier(k)
		case "queue_id":
			val = "queue_id::varchar as " + pq.QuoteIdentifier(k)
		case "agent_id":
			val = "agent_id::varchar as " + pq.QuoteIdentifier(k)
		case "description":
			val = "description::varchar as " + pq.QuoteIdentifier(k)
		case "created_at":
			val = "created_at::varchar as " + pq.QuoteIdentifier(k)
		case "gateway_id":
			val = "gateway_id::varchar as " + pq.QuoteIdentifier(k)
		case "destination":
			val = "destination::varchar as " + pq.QuoteIdentifier(k)
		default:

			if !strings.HasPrefix(fmt.Sprintf("%s", v), "variables.") {
				continue
			}

			val = fmt.Sprintf("coalesce(regexp_replace((h.variables->%s)::text, '\n|\t', ' ', 'g'), '') as %s", pq.QuoteLiteral(fmt.Sprintf("%s", v)[10:]), pq.QuoteIdentifier(k))
		}

		f = append(f, val)
	}

	var t *properties

	// fixme extension dialer logic
	err := s.GetReplica().SelectOne(&t, `select row_to_json(t) variables
from (select `+strings.Join(f, ", ")+`
      from (select to_char(h.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') created_at,
                   coalesce(case
                                when h.direction = 'inbound' or q.type = any (array [4,5]::smallint[]) then h.to_number
                                else h.from_number end, '') as         extension,
                   h.queue_id,
                   ah.agent_id,
                   coalesce(regexp_replace(ah.description, '\n|\t',  ' ', 'g'), '')             as         description,
                   h.gateway_id,
                   h.payload                                as         variables,
				   h.id,
			       h.destination
            from call_center.cc_calls_history h
                     left join call_center.cc_queue q on q.id = h.queue_id
                     left join call_center.cc_member_attempt_history ah
                               on ah.domain_id = h.domain_id and ah.member_call_id = h.id::text
            where (h.domain_id = :DomainId and h.created_at > now() - (:Hours::varchar || ' hours')::interval)
              and (:QueueIds::int[] isnull or (h.queue_id = any (:QueueIds) or h.queue_id isnull))
              and (
                    (h.domain_id = :DomainId and h.destination ~~* :Number::varchar)
                    or (h.domain_id = :DomainId and h.to_number ~~* :Number::varchar)
                    or (h.domain_id = :DomainId and h.from_number ~~* :Number::varchar)
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
            order by h.created_at desc) h
      order by h.created_at desc
      limit 1) t`, map[string]interface{}{
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

	return t.Variables, nil
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

func (s SqlCallStore) SetUserId(domainId int64, id string, userId int64) *model.AppError {
	_, err := s.GetMaster().Exec(`update call_center.cc_calls
set user_id = :UserId
where domain_id = :DomainId and id = :Id;`, map[string]interface{}{
		"DomainId": domainId,
		"UserId":   userId,
		"Id":       id,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetUserId", "store.sql_call.set_user_id.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetBlindTransfer(domainId int64, id string, destination string) *model.AppError {
	_, err := s.GetMaster().Exec(`update call_center.cc_calls
set blind_transfer = :Destination 
where id = :Id and domain_id = :DomainId`, map[string]interface{}{
		"Id":          id,
		"DomainId":    domainId,
		"Destination": destination,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetBlindTransfer", "store.sql_call.set_blind_transfer.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetContactId(domainId int64, id string, contactId int64) *model.AppError {
	_, err := s.GetMaster().Exec(`with ua as (
    update call_center.cc_calls
        set contact_id  = :ContactId
    where id = :Id and domain_id = :DomainId
    returning id
), uh as (
    update call_center.cc_calls_history
        set contact_id  = :ContactId
    where id = :Id and domain_id = :DomainId
        and not exists(select 1 from ua)
    returning id
)
select ua.id as id
from ua
union all
select uh.id as id
from uh`, map[string]interface{}{
		"DomainId":  domainId,
		"ContactId": contactId,
		"Id":        id,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetContactId", "store.sql_call.set_contact.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}
