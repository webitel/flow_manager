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
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, direction, destination, parent_id, timestamp, state, app_id, from_type, from_name,
                      from_number, from_id, to_type, to_name, to_number, to_id, payload, domain_id, created_at)
values (:Id, :Direction, :Destination, :ParentId, :Timestamp, :State, :AppId, :FromType, :FromName, :FromNumber, :FromId,
        :ToType, :ToName, :ToNumber, :ToId, :Payload, :DomainId, :CreatedAt)
on conflict (id)
    do update set
		created_at = :CreatedAt,
        direction = :Direction,
        destination = :Destination,
        parent_id = :ParentId,
        from_type = :FromType,
        from_name = :FromName,
        from_number = :FromNumber,
        from_id = :FromId,
        to_type = :ToType,
        to_name = :ToName,
        to_number = :ToNumber,
        to_id = :ToId,
        payload = :Payload`, map[string]interface{}{
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

		"ToType":   call.GetTo().GetType(),
		"ToName":   call.GetTo().GetName(),
		"ToNumber": call.GetTo().GetNumber(),
		"ToId":     call.GetTo().GetId(),
		"Payload":  nil,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.Save", "store.sql_call.save.error", nil,
			fmt.Sprintf("Id=%v %v", call.Id, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetState(call *model.CallAction) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, state, timestamp, app_id, domain_id)
values (:Id, :State, :Timestamp, :AppId, :DomainId)
on conflict (id) where timestamp < :Timestamp and cause isnull
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
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, state, timestamp, app_id, domain_id, cause, sip_code)
values (:Id, :State, :Timestamp, :AppId, :DomainId, :Cause, :SipCode)
on conflict (id) where timestamp <= :Timestamp
    do update set
      state = EXCLUDED.state,
      cause = EXCLUDED.cause,
      sip_code = EXCLUDED.sip_code,
      timestamp = EXCLUDED.timestamp`, map[string]interface{}{
		"Id":        call.Id,
		"State":     call.Event,
		"Timestamp": call.Timestamp,
		"AppId":     call.AppId,
		"DomainId":  call.DomainId,
		"Cause":     call.Cause,
		"SipCode":   call.SipCode,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetState", "store.sql_call.set_state.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlCallStore) SetBridged(call *model.CallActionBridge) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, state, timestamp, app_id, domain_id, bridged_id)
values (:Id, :State, :Timestamp, :AppId, :DomainId, :BridgedId)
on conflict (id) where timestamp < :Timestamp
    do update set
      state = :State,
      bridged_id = :BridgedId,
      timestamp = :Timestamp`, map[string]interface{}{
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
	_, err := s.GetMaster().Exec(`with c as (
    delete from cc_calls c
	where c.hangup_at > 0
    returning c.created_at, c.id, c.direction, c.destination, c.parent_id, c.app_id, c.from_type, c.from_name, c.from_number, c.from_id,
       c.to_type, c.to_name, c.to_number, c.to_id, c.payload, c.domain_id,
       c.answered_at, c.bridged_at, c.hangup_at, c.hold_sec, c.cause, c.sip_code, c.bridged_id
)
insert into cc_calls_history (created_at, id, direction, destination, parent_id, app_id, from_type, from_name, from_number, from_id,
                              to_type, to_name, to_number, to_id, payload, domain_id, answered_at, bridged_at, hangup_at, hold_sec, cause, sip_code, bridged_id)
select c.created_at created_at, c.id, c.direction, c.destination, c.parent_id, c.app_id, c.from_type, c.from_name, c.from_number, c.from_id,
       c.to_type, c.to_name, c.to_number, c.to_id, c.payload, c.domain_id,
       c.answered_at, c.bridged_at, c.hangup_at, c.hold_sec, c.cause, c.sip_code, c.bridged_id
from c`)
	if err != nil {
		return model.NewAppError("SqlCallStore.MoveToHistory", "store.sql_call.move_to_store.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}
