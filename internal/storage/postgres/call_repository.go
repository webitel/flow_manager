package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/flow"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	pgsql "github.com/webitel/flow_manager/internal/infrastructure/sql/pgsql"
	"github.com/webitel/flow_manager/internal/storage"
)

type CallRepository struct {
	db infraSql.Store
}

func NewCallRepository(db infraSql.Store) storage.CallStore {
	return &CallRepository{db: db}
}

const saveCallSQL = `insert into call_center.cc_calls (id, direction, destination, parent_id, "timestamp", state, app_id, from_type, from_name,
                  from_number, from_id, to_type, to_name, to_number, to_id, payload, domain_id, created_at, gateway_id, user_id, queue_id, agent_id, team_id,
				  attempt_id, member_id, grantee_id, params, heartbeat, destination_name, contact_id)
values (@Id, @Direction, @Destination, @ParentId, to_timestamp(@Timestamp::double precision /1000), @State, @AppId, @FromType, @FromName, @FromNumber, @FromId,
        @ToType, @ToName, @ToNumber, @ToId, @Payload, @DomainId, to_timestamp(@CreatedAt::double precision /1000), @GatewayId, @UserId, @QueueId, @AgentId, @TeamId,
		@AttemptId, @MemberId, @GranteeId, @Params::jsonb, case when @Hb::int > 0 then now() end, @DestinationName, @ContactId)
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
		params = EXCLUDED.params,
        heartbeat = EXCLUDED.heartbeat,
        destination_name = EXCLUDED.destination_name,
        contact_id = EXCLUDED.contact_id`

func (r *CallRepository) Save(c *call.CallActionRinging) error {
	return r.db.Exec(context.Background(), saveCallSQL, pgx.NamedArgs{
		"DomainId":        c.DomainId,
		"Id":              c.Id,
		"Direction":       c.Direction,
		"Destination":     c.Destination,
		"ParentId":        c.ParentId,
		"Timestamp":       c.Timestamp,
		"State":           c.Event,
		"AppId":           c.AppId,
		"CreatedAt":       c.Timestamp,
		"FromType":        c.GetFrom().GetType(),
		"FromName":        c.GetFrom().GetName(),
		"FromNumber":      c.GetFrom().GetNumber(),
		"FromId":          c.GetFrom().GetId(),
		"ToType":          c.GetTo().GetType(),
		"ToName":          c.GetTo().GetName(),
		"ToNumber":        c.GetTo().GetNumber(),
		"ToId":            c.GetTo().GetId(),
		"GatewayId":       c.GatewayId,
		"UserId":          c.UserId,
		"QueueId":         c.GetQueueId(),
		"AgentId":         c.GetAgentId(),
		"TeamId":          c.GetTeamId(),
		"AttemptId":       c.GetAttemptId(),
		"MemberId":        c.GetMemberIdId(),
		"GranteeId":       c.GranteeId,
		"Payload":         nil,
		"Params":          c.GetParams(),
		"Hb":              c.Heartbeat,
		"DestinationName": c.DestinationName,
		"ContactId":       c.ContactId,
	})
}

const setStateSQL = `insert into call_center.cc_calls(id, state, timestamp, app_id, domain_id)
values (@Id::uuid, @State, to_timestamp(@Timestamp::double precision /1000), @AppId, @DomainId)
on conflict (id) where timestamp < to_timestamp(@Timestamp::double precision /1000) and cause isnull
    do update set
      state = EXCLUDED.state,
      timestamp = EXCLUDED.timestamp`

func (r *CallRepository) SetState(c *call.CallAction) error {
	return r.db.Exec(context.Background(), setStateSQL, pgx.NamedArgs{
		"Id":        c.Id,
		"State":     c.Event,
		"Timestamp": c.Timestamp,
		"AppId":     c.AppId,
		"DomainId":  c.DomainId,
	})
}

const setHangupSQL = `insert into call_center.cc_calls (id, state, timestamp, app_id, domain_id, cause,
		sip_code, payload, hangup_by, tags, amd_result, params, talk_sec, amd_ai_result, amd_ai_logs, amd_ai_positive,
	    schema_ids, hangup_phrase, transfer_from)
values (@Id, @State, to_timestamp(@Timestamp::double precision /1000), @AppId, @DomainId, @Cause,
	@SipCode, @Variables::json, @HangupBy, @Tags, @AmdResult, @Params::jsonb, coalesce(@TalkSec::int, 0), @AmdAiResult,
	@AmdAiResultLog, @AmdAiPositive, @SchemaIds::int[], @HangupPhrase::varchar, @TransferFrom::uuid)
on conflict (id) where timestamp <= to_timestamp(@Timestamp::double precision / 1000)
    do update set
      state = EXCLUDED.state,
      cause = EXCLUDED.cause,
      sip_code = EXCLUDED.sip_code,
      payload = coalesce(call_center.cc_calls.payload, '{}') || EXCLUDED.payload,
      hangup_by = EXCLUDED.hangup_by,
	  tags = EXCLUDED.tags,
	  amd_result = EXCLUDED.amd_result,
	  params = coalesce(call_center.cc_calls.params, '{}') || EXCLUDED.params,
	  talk_sec = EXCLUDED.talk_sec::int,
      timestamp = EXCLUDED.timestamp,
      amd_ai_result = EXCLUDED.amd_ai_result,
      amd_ai_logs = EXCLUDED.amd_ai_logs,
      amd_ai_positive = EXCLUDED.amd_ai_positive,
      schema_ids = EXCLUDED.schema_ids,
      hangup_phrase = EXCLUDED.hangup_phrase,
      transfer_from = coalesce(call_center.cc_calls.transfer_from, EXCLUDED.transfer_from)`

func (r *CallRepository) SetHangup(c *call.CallActionHangup) error {
	return r.db.Exec(context.Background(), setHangupSQL, pgx.NamedArgs{
		"Id":             c.Id,
		"State":          c.Event,
		"Timestamp":      c.Timestamp,
		"AppId":          c.AppId,
		"DomainId":       c.DomainId,
		"Cause":          c.Cause,
		"SipCode":        c.SipCode,
		"HangupBy":       c.HangupBy,
		"AmdResult":      c.AmdResult,
		"TalkSec":        c.TalkSec,
		"Tags":           c.Tags,
		"Variables":      c.VariablesToJson(),
		"Params":         c.Parameters(),
		"AmdAiResult":    c.AmdAiResult,
		"AmdAiResultLog": c.AmdAiResultLog,
		"AmdAiPositive":  c.AmdAiPositive,
		"SchemaIds":      c.SchemaIds,
		"HangupPhrase":   c.HangupPhrase,
		"TransferFrom":   c.TransferFrom,
	})
}

const setBridgedSQL = `call call_center.cc_call_set_bridged(@Id::uuid, @State::varchar, to_timestamp(@Timestamp::double precision /1000), @AppId::varchar,
    @DomainId::int8, @BridgedId::uuid, @ToName::varchar)`

func (r *CallRepository) SetBridged(c *call.CallActionBridge) error {
	return r.db.Exec(context.Background(), setBridgedSQL, pgx.NamedArgs{
		"Id":        c.Id,
		"State":     c.Event,
		"Timestamp": c.Timestamp,
		"AppId":     c.AppId,
		"DomainId":  c.DomainId,
		"BridgedId": c.BridgedId,
		"ToName":    c.To.Name,
	})
}

const deleteCallSQL = `delete from call_center.cc_calls where id = @Id`

func (r *CallRepository) Delete(id string) error {
	return r.db.Exec(context.Background(), deleteCallSQL, pgx.NamedArgs{"Id": id})
}

const moveToHistorySQL = `with hb as materialized (
	update call_center.cc_calls
	set state = 'hangup',
		sip_code = 604,
		hangup_at = now() - interval '2s',
		hangup_by = 'service',
		cause = 'MEDIA_TIMEOUT'
	where hangup_at isnull and cause isnull and heartbeat < now() - (((params->>'heartbeat')::int * 3) || ' sec')::interval),
del_calls as materialized (
    select *
    from call_center.cc_calls c
        where c.hangup_at < now() - '1 sec'::interval
            and c.direction notnull
            and not exists(select 1 from call_center.cc_calls cc where case when c.parent_id notnull then cc.id = c.parent_id else cc.parent_id = c.id and cc.hangup_at isnull and c.direction notnull  end )
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
),
ins as (
    insert
into call_center.cc_calls_history (created_at, id, direction, destination, parent_id, app_id, from_type, from_name,
                                   from_number, from_id,
                                   to_type, to_name, to_number, to_id, payload, domain_id, answered_at, bridged_at,
                                   hangup_at, hold_sec, cause, sip_code, bridged_id,
                                   gateway_id, user_id, queue_id, team_id, agent_id, attempt_id, member_id, hangup_by,
                                   transfer_from, transfer_to, amd_result, amd_duration,
                                   tags, grantee_id, "hold", user_ids, agent_ids, gateway_ids, queue_ids, team_ids, params,
								   blind_transfer, talk_sec, amd_ai_result, amd_ai_logs, amd_ai_positive, contact_id, search_number,
  								   schema_ids, hangup_phrase, blind_transfers, attempt_ids)
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
	   c.search_number,
       c.schema_ids,
	   c.hangup_phrase,
	   c.blind_transfers,
	   c.rattempt_ids
from (
         select (t.r).*,
                case when (t.r).agent_id isnull then t.agent_ids else (t.r).agent_id || t.agent_ids end agent_ids,
                case when (t.r).user_id isnull then t.user_ids else (t.r).user_id || t.user_ids end     user_ids,
                case
                    when (t.r).gateway_id isnull then t.gateway_ids
                    else (t.r).gateway_id || t.gateway_ids end                                          gateway_ids,
                case when (t.r).queue_id isnull then t.queue_ids else (t.r).queue_id || t.queue_ids end queue_ids,
                case when (t.r).team_id isnull then t.team_ids else (t.r).team_id || t.team_ids end team_ids,
                case when (t.r).attempt_id isnull then t.attempt_ids else (t.r).attempt_id || t.attempt_ids end rattempt_ids,
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
                         array_agg(distinct ch.attempt_id)
                         filter ( where c.parent_id isnull and ch.attempt_id notnull ) attempt_ids,
						 call_center.jsonb_concat_agg(ch.payload) p_vars,
						 string_agg(distinct nums.from_number, '|') filter ( where  nums.from_number != '' and nums.from_number notnull ) search_number
                  from del_calls c
                           left join call_center.cc_calls ch on (ch.parent_id = c.id or (ch.id = c.bridged_id) or ch.transfer_to = c.id or ch.transfer_from = c.id)
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
     ) c
    returning  id, parent_id::uuid, bridged_id, user_id, domain_id
)
select id::text, user_id, domain_id
from ins
where bridged_id isnull and parent_id notnull and user_id notnull`

func (r *CallRepository) MoveToHistory() ([]call.MissedCall, error) {
	var out []call.MissedCall
	if err := r.db.Select(context.Background(), &out, moveToHistorySQL, pgx.NamedArgs{}); err != nil {
		return nil, err
	}
	return out, nil
}

const updateFromSQL = `update call_center.cc_calls
set from_number = coalesce(@Number, from_number),
    from_name = coalesce(@Name, from_name),
    destination = coalesce(@Destination, destination)
where id = @Id`

func (r *CallRepository) UpdateFrom(id string, name, number, destination *string) error {
	return r.db.Exec(context.Background(), updateFromSQL, pgx.NamedArgs{
		"Number":      number,
		"Name":        name,
		"Destination": destination,
		"Id":          id,
	})
}

const saveTranscriptSQL = `insert into call_center.cc_calls_transcribe (call_id, confidence, transcribe, response)
select @CallId, (x.j->'alternatives'->0->'confidence')::text::numeric,
        x.j->'alternatives'->0->>'transcript' as transcript,
        x.j
from (
    select @R::jsonb as j
) x`

func (r *CallRepository) SaveTranscript(transcribe *call.CallActionTranscript) error {
	raw, _ := json.Marshal(transcribe.Transcript)
	return r.db.Exec(context.Background(), saveTranscriptSQL, pgx.NamedArgs{
		"CallId": transcribe.Id,
		"R":      raw,
	})
}

type lastBridgedRow struct {
	Variables json.RawMessage `db:"variables"`
}

func (r *CallRepository) LastBridged(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes flow.Variables) (flow.Variables, error) {
	fields := make([]string, 0, len(mapRes))
	for k, v := range mapRes {
		var col string
		switch v {
		case "extension":
			col = "extension::varchar as " + pgsql.QuoteIdentifier(k)
		case "id":
			col = "id::varchar as " + pgsql.QuoteIdentifier(k)
		case "queue_id":
			col = "queue_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "agent_id":
			col = "agent_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "description":
			col = "description::varchar as " + pgsql.QuoteIdentifier(k)
		case "created_at":
			col = "created_at::varchar as " + pgsql.QuoteIdentifier(k)
		case "gateway_id":
			col = "gateway_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "destination":
			col = "destination::varchar as " + pgsql.QuoteIdentifier(k)
		default:
			sv := fmt.Sprintf("%s", v)
			if !strings.HasPrefix(sv, "variables.") {
				continue
			}
			col = fmt.Sprintf("coalesce(regexp_replace((h.variables->%s)::text, '\n|\t', ' ', 'g'), '') as %s",
				pgsql.QuoteLiteral(sv[10:]), pgsql.QuoteIdentifier(k))
		}
		fields = append(fields, col)
	}

	q := `select row_to_json(t) variables
from (select ` + strings.Join(fields, ", ") + `
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
            where (h.domain_id = @DomainId and h.created_at > now() - (@Hours::varchar || ' hours')::interval)
              and (@QueueIds::int[] isnull or (h.queue_id = any (@QueueIds) or h.queue_id isnull))
              and (
                    (h.domain_id = @DomainId and h.destination ~~* @Number::varchar)
                    or (h.domain_id = @DomainId and h.to_number ~~* @Number::varchar)
                    or (h.domain_id = @DomainId and h.from_number ~~* @Number::varchar)
                )
              and h.parent_id isnull
              and (
                    ((@Dialer::varchar isnull or @Dialer::varchar = 'false') and
                     (@Inbound::varchar isnull or @Inbound::varchar = 'false') and
                     (@Outbound::varchar isnull or @Outbound::varchar = 'false')) or
                    (
                            case
                                when @Dialer::varchar notnull and @Dialer::varchar != 'false' then
                                        h.attempt_id notnull and case @Dialer
                                                                     when 'bridged' then h.bridged_at notnull
                                                                     when 'attempt' then h.bridged_at isnull
                                                                     else true end
                                else false end
                            or case
                                   when @Inbound::varchar notnull and @Inbound::varchar != 'false' then
                                               h.direction = 'inbound' and case @Inbound
                                                                               when 'bridged' then h.bridged_at notnull
                                                                               when 'attempt' then h.bridged_at isnull
                                                                               else true end
                                   else false end
                            or case
                                   when @Outbound::varchar notnull and @Outbound::varchar != 'false' then
                                               h.direction = 'outbound' and case @Outbound
                                                                                when 'bridged' then h.bridged_at notnull
                                                                                when 'attempt' then h.bridged_at isnull
                                                                                else true end
                                   else false end
                        )
                )
            order by h.created_at desc) h
      order by h.created_at desc
      limit 1) t`

	var row lastBridgedRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"DomainId": domainId,
		"Hours":    hours,
		"Number":   number,
		"Dialer":   dialer,
		"Inbound":  inbound,
		"Outbound": outbound,
		"QueueIds": queueIds,
	}); err != nil {
		return nil, err
	}

	var vars flow.Variables
	if err := json.Unmarshal(row.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

const setGranteeIdSQL = `update call_center.cc_calls
set grantee_id = @GranteeId
where domain_id = @DomainId and id = @Id`

func (r *CallRepository) SetGranteeId(domainId int64, id string, granteeId int64) error {
	return r.db.Exec(context.Background(), setGranteeIdSQL, pgx.NamedArgs{
		"DomainId":  domainId,
		"GranteeId": granteeId,
		"Id":        id,
	})
}

const setUserIdSQL = `update call_center.cc_calls
set user_id = @UserId
where domain_id = @DomainId and id = @Id`

func (r *CallRepository) SetUserId(domainId int64, id string, userId int64) error {
	return r.db.Exec(context.Background(), setUserIdSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"UserId":   userId,
		"Id":       id,
	})
}

const setBlindTransferSQL = `update call_center.cc_calls
set blind_transfer = @Destination::varchar,
    blind_transfers = coalesce(blind_transfers, '[]')::jsonb || jsonb_build_object('number', @Destination::varchar, 'time', (extract(epoch from now())::numeric * 1000)::int8)
where id = @Id and domain_id = @DomainId`

func (r *CallRepository) SetBlindTransfer(domainId int64, id, destination string) error {
	return r.db.Exec(context.Background(), setBlindTransferSQL, pgx.NamedArgs{
		"Id":          id,
		"DomainId":    domainId,
		"Destination": destination,
	})
}

const setContactIdSQL = `with ua as (
    update call_center.cc_calls
        set contact_id = @ContactId
        where id = @Id and domain_id = @DomainId
        returning id),
     uh as (
         update call_center.cc_calls_history h
             set contact_id = @ContactId
             from (select *
                   from (select ua.id
                         from ua
                         union
                         select null) x
                   order by x.id nulls last
                   limit 1) x
             where h.id = @Id and h.domain_id = @DomainId
                 and x.id isnull
             returning h.id)
select ua.id as id
from ua
union all
select uh.id as id
from uh`

func (r *CallRepository) SetContactId(domainId int64, id string, contactId int64) error {
	return r.db.Exec(context.Background(), setContactIdSQL, pgx.NamedArgs{
		"DomainId":  domainId,
		"ContactId": contactId,
		"Id":        id,
	})
}

const setVariablesSQL = `with a as (
    update call_center.cc_calls c
        set payload = coalesce(payload, '{}') || @Vars
    where c.id = @Id::uuid
    returning c.id
), h as (
    update call_center.cc_calls_history c
        set payload = coalesce(payload, '{}') || @Vars
    where c.id = @Id::uuid
    returning c.id
)
select *
from (
    select id
    from a
    union all
    select id
    from h
 ) as t
where t.id notnull
limit 1`

func (r *CallRepository) SetVariables(id string, vars *call.CallVariables) error {
	return r.db.Exec(context.Background(), setVariablesSQL, pgx.NamedArgs{
		"Id":   id,
		"Vars": vars.ToMapJson(),
	})
}

const setHeartbeatSQL = `update call_center.cc_calls
set heartbeat = now() + (params->>'heartbeat' || ' sec')::interval
where id = @Id`

func (r *CallRepository) SetHeartbeat(id string) error {
	return r.db.Exec(context.Background(), setHeartbeatSQL, pgx.NamedArgs{"Id": id})
}

const saveMediaStatsSQL = `insert into call_center.cc_calls_media_stats (created_at, sip_id, domain_id, user_id, mos_avg, mos_min, mos_min_at, mos_max,
                                          mos_max_at, jitter_avg, jitter_min, jitter_min_at, jitter_max, jitter_max_at,
                                          packetloss_avg, packetloss_min, packetloss_min_at, packetloss_max, packetloss_max_at,
                                          roundtrip_avg, roundtrip_max, roundtrip_max_at, roundtrip_min, roundtrip_min_at)
values (now(), @SipId, @DomainId, @UserId, @MosAvg, @MosMin, @MosMinAt, @MosMax, @MosMaxAt, @JitterAvg, @JitterMin,
        @JitterMinAt, @JitterMax, @JitterMaxAt, @PacketlossAvg, @PacketlossMin, @PacketlossMinAt, @PacketlossMax, @PacketlossMaxAt,
        @RoundtripAvg, @RoundtripMax, @RoundtripMaxAt, @RoundtripMin, @RoundtripMinAt)`

func (r *CallRepository) SaveMediaStats(stats *call.CallActionMediaStats) error {
	return r.db.Exec(context.Background(), saveMediaStatsSQL, pgx.NamedArgs{
		"DomainId":        stats.DomainId,
		"SipId":           stats.SipId,
		"UserId":          stats.UserId,
		"MosAvg":          stats.RTP.Mos.Average,
		"MosMin":          stats.RTP.Mos.Min,
		"MosMinAt":        stats.RTP.Mos.MinAt,
		"MosMax":          stats.RTP.Mos.Max,
		"MosMaxAt":        stats.RTP.Mos.MaxAt,
		"JitterAvg":       stats.RTP.Mos.Average,
		"JitterMin":       stats.RTP.Mos.Min,
		"JitterMinAt":     stats.RTP.Mos.MinAt,
		"JitterMax":       stats.RTP.Mos.Max,
		"JitterMaxAt":     stats.RTP.Mos.MaxAt,
		"PacketlossAvg":   stats.RTP.PacketLoss.Average,
		"PacketlossMin":   stats.RTP.PacketLoss.Min,
		"PacketlossMinAt": stats.RTP.PacketLoss.MinAt,
		"PacketlossMax":   stats.RTP.PacketLoss.Max,
		"PacketlossMaxAt": stats.RTP.PacketLoss.MaxAt,
		"RoundtripAvg":    stats.RTP.RoundTrip.Average,
		"RoundtripMax":    stats.RTP.RoundTrip.Max,
		"RoundtripMaxAt":  stats.RTP.RoundTrip.MaxAt,
		"RoundtripMin":    stats.RTP.RoundTrip.Min,
		"RoundtripMinAt":  stats.RTP.RoundTrip.MinAt,
	})
}
