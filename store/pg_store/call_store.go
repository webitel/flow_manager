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
                      from_number, from_id, to_type, to_name, to_number, to_id, payload)
values (:Id, :Direction, :Destination, :ParentId, :Timestamp, :State, :AppId, :FromType, :FromName, :FromNumber, :FromId,
        :ToType, :ToName, :ToNumber, :ToId, :Payload)
on conflict (id)
    do update set
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
		"Id":          call.Id,
		"Direction":   call.Direction,
		"Destination": call.Destination,
		"ParentId":    call.ParentId,
		"Timestamp":   call.Timestamp,
		"State":       call.Event,
		"AppId":       call.AppId,
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
	_, err := s.GetMaster().Exec(`insert into cc_calls (id, state, timestamp, app_id)
values (:Id, :State, :Timestamp, :AppId)
on conflict (id) where timestamp < :Timestamp
    do update set 
      state = :State,
      timestamp = :Timestamp`, map[string]interface{}{
		"Id":        call.Id,
		"State":     call.Event,
		"Timestamp": call.Timestamp,
		"AppId":     call.AppId,
	})

	if err != nil {
		return model.NewAppError("SqlCallStore.SetState", "store.sql_call.set_state.error", nil,
			fmt.Sprintf("Id=%v, State=%v %v", call.Id, call.Event, err.Error()), extractCodeFromErr(err))
	}

	return nil
}
