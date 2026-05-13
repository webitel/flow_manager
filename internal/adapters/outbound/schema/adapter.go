package schema

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/webitel/engine/pkg/presign"
	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

var (
	requestGroup  singleflight.Group
	systemGroup   singleflight.Group
	hookGroup     singleflight.Group
	variableGroup singleflight.Group

	systemCache   = model.NewLruWithParams(300, "system_settings", 60, "")
	variableCache = model.NewLruWithParams(10000, "variable", 10, "")
)

// Adapter implements schema lookup, system settings, hook lookup,
// and schema variable read/write.
type SchemaAdapter struct {
	store        store.Store
	schemaCache  model.ObjectCache
	cert         presign.PreSign // optional; set via SetCert after Start()
	timezoneList map[int]*time.Location
}

func NewSchemaAdapter(st store.Store, schemaCache model.ObjectCache) *SchemaAdapter {
	return &SchemaAdapter{store: st, schemaCache: schemaCache}
}

// SetCert wires the signing key after Start() creates it.
func (a *SchemaAdapter) SetCert(c presign.PreSign) { a.cert = c }

// ── timezones ─────────────────────────────────────────────────────────────────

func (a *SchemaAdapter) InitCacheTimezones() error {
	list, storeErr := a.store.Calendar().GetTimezones()
	if storeErr != nil {
		return fmt.Errorf("InitCacheTimezones: store.calendar.get_timezones: %w", storeErr)
	}

	a.timezoneList = make(map[int]*time.Location, len(list))

	for _, v := range list {
		if loc, err := time.LoadLocation(v.SysName); err != nil {
			wlog.Warn(fmt.Sprintf("bad database timezone name %s, skip cache", v.SysName))
		} else {
			a.timezoneList[v.Id] = loc
		}
	}

	return nil
}

func (a *SchemaAdapter) GetLocation(id int) *time.Location {
	loc, _ := a.timezoneList[id]
	return loc
}

// ── schema ────────────────────────────────────────────────────────────────────

func (a *SchemaAdapter) GetSchema(domainId int64, id int, updatedAt int64) (*model.Schema, error) {
	if v, ok := a.schemaCache.Get(id); ok {
		s := v.(*model.Schema)
		if s.UpdatedAt == updatedAt {
			return s, nil
		}
	}

	v, doErr, _ := requestGroup.Do(fmt.Sprintf("GetSchema-%d-%d-%d", domainId, id, updatedAt), func() (interface{}, error) {
		return a.store.Schema().Get(domainId, id)
	})
	if doErr != nil {
		return nil, toError("GetSchema", doErr)
	}

	s := v.(*model.Schema)
	wlog.Debug(fmt.Sprintf("add schema \"%s\" [%d] to cache", s.Name, s.Id))
	a.schemaCache.AddWithDefaultExpires(id, s)
	return s, nil
}

func (a *SchemaAdapter) GetSchemaById(domainId int64, id int) (*model.Schema, error) {
	v, err, _ := requestGroup.Do(fmt.Sprintf("GetSchemaById-%d-%d", domainId, id), func() (interface{}, error) {
		return a.store.Schema().GetUpdatedAt(domainId, id)
	})
	if err != nil {
		return nil, toError("GetSchemaById", err)
	}
	return a.GetSchema(domainId, id, v.(int64))
}

func (a *SchemaAdapter) SearchTransferredRouting(domainId int64, schemaId int) (*model.Routing, error) {
	routing, rErr := a.store.Schema().GetTransferredRouting(domainId, schemaId)
	if rErr != nil {
		return nil, toError("SearchTransferredRouting", rErr)
	}
	var schemaErr error
	routing.Schema, schemaErr = a.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if schemaErr != nil {
		return nil, schemaErr
	}
	return routing, nil
}

// ── system settings ───────────────────────────────────────────────────────────

func (a *SchemaAdapter) GetSystemSettings(ctx context.Context, domainId int64, name string) (model.SysValue, error) {
	key := fmt.Sprintf("%d-%s", domainId, name)
	if c, ok := systemCache.Get(key); ok {
		return c.(model.SysValue), nil
	}

	v, err, share := systemGroup.Do(key, func() (interface{}, error) {
		res, err := a.store.SystemcSettings().Get(ctx, domainId, name)
		if err != nil {
			return model.SysValue{}, err
		}
		var val interface{}
		var s model.SysValue
		json.Unmarshal(res, &val) //nolint:errcheck
		switch b := val.(type) {
		case bool:
			s.BoolValue = b
		case string:
			s.StringValue = b
		}
		return s, nil
	})
	if err != nil {
		return model.SysValue{}, toError("GetSystemSettings", err)
	}
	if !share {
		systemCache.AddWithDefaultExpires(key, v.(model.SysValue))
	}
	return v.(model.SysValue), nil
}

// ── hook ──────────────────────────────────────────────────────────────────────

func (a *SchemaAdapter) GetHookById(key string) (model.WebHook, error) {
	v, err, _ := hookGroup.Do(key, func() (interface{}, error) {
		return a.store.WebHook().Get(key)
	})
	if err != nil {
		return model.WebHook{}, toError("GetHookById", err)
	}
	return v.(model.WebHook), nil
}

// ── schema variables ──────────────────────────────────────────────────────────

func (a *SchemaAdapter) SchemaVariable(ctx context.Context, domainId int64, name string) string {
	key := fmt.Sprintf("%d-%s", domainId, name)
	if v, ok := variableCache.Get(key); ok {
		return v.(string)
	}
	v, _, _ := variableGroup.Do(key, func() (interface{}, error) {
		return a.schemaVariable(key, domainId, name), nil
	})
	return v.(string)
}

func (a *SchemaAdapter) SetGlobalVar(ctx context.Context, domainId int64, name, value string, encrypt bool) error {
	return a.SetSchemaVariable(ctx, domainId, map[string]*model.SchemaVariable{
		name: {Value: []byte(value), Encrypt: encrypt},
	})
}

func (a *SchemaAdapter) SetSchemaVariable(ctx context.Context, domainId int64, vars map[string]*model.SchemaVariable) error {
	if len(vars) == 0 {
		return nil
	}
	for k, v := range vars {
		if v.Encrypt && a.cert != nil {
			enc, err := a.cert.EncryptBytes(v.Value)
			if err != nil {
				return fmt.Errorf("SetSchemaVariable: app.encrypt: %w", err)
			}
			v.Value = enc
		}
		var buf bytes.Buffer
		buf.WriteString(`"`)
		buf.Write(v.Value)
		buf.WriteString(`"`)
		v.Value = buf.Bytes()
		if setErr := a.store.Schema().SetVariable(domainId, k, v); setErr != nil {
			wlog.Error(setErr.Error())
		}
	}
	return nil
}

func (a *SchemaAdapter) schemaVariable(key string, domainId int64, name string) string {
	sb, err := a.store.Schema().GetVariable(domainId, name)
	if err != nil {
		wlog.Error(fmt.Sprintf("get schema variable error: %s", err.Error()))
		return ""
	}
	if sb.Encrypt && a.cert != nil {
		b, err := a.cert.DecryptBytes(sb.Value)
		if err != nil {
			wlog.Error(fmt.Sprintf("decrypt schema variable error: %s", err.Error()))
			return ""
		}
		val := removeQuote(b)
		variableCache.AddWithDefaultExpires(key, val)
		return val
	}
	val := removeQuote(sb.Value)
	variableCache.AddWithDefaultExpires(key, val)
	return val
}

// ── helpers ───────────────────────────────────────────────────────────────────

func removeQuote(text []byte) string {
	l := len(text)
	if l < 2 {
		return string(text)
	}
	if text[0] == '"' {
		text = text[1:]
		l--
	}
	if text[l-1] == '"' {
		text = text[:l-1]
	}
	return string(text)
}

// ── call routing ──────────────────────────────────────────────────────────────

func (a *SchemaAdapter) GetRoutingFromDestToGateway(domainId int64, gatewayId int) (*model.Routing, error) {
	routing, err := a.store.CallRouting().FromGateway(domainId, gatewayId)
	if err != nil {
		return nil, toError("GetRoutingFromDestToGateway", err)
	}
	var appErr error
	routing.Schema, appErr = a.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	return routing, appErr
}

func (a *SchemaAdapter) SearchOutboundToDestinationRouting(domainId int64, dest string) (*model.Routing, error) {
	routing, err := a.store.CallRouting().SearchToDestination(domainId, dest)
	if err != nil {
		return nil, toError("SearchOutboundToDestinationRouting", err)
	}
	var appErr error
	routing.Schema, appErr = a.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	return routing, appErr
}

func (a *SchemaAdapter) SearchOutboundFromQueueRouting(domainId int64, queueId int) (*model.Routing, error) {
	routing, err := a.store.CallRouting().FromQueue(domainId, queueId)
	if err != nil {
		return nil, toError("SearchOutboundFromQueueRouting", err)
	}
	var appErr error
	routing.Schema, appErr = a.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	return routing, appErr
}

func (a *SchemaAdapter) TransferQueueRouting(domainId int64, queueId int) (*model.Routing, error) {
	return &model.Routing{
		DomainId: domainId,
		Schema: &model.Schema{
			DomainId: domainId, Name: "transfer queue",
			Schema: model.Applications{
				{"sleep": 500},
				{"unSet": []any{"wbt_bt_queue_id", "wbt_bt_queue"}},
				{"joinQueue": map[string]any{"queue": map[string]any{"id": queueId}}},
				{"hangup": nil},
			},
		},
	}, nil
}

func (a *SchemaAdapter) TransferAgentRouting(domainId int64, agentId int) (*model.Routing, error) {
	return &model.Routing{
		DomainId: domainId,
		Schema: &model.Schema{
			DomainId: domainId, Name: "transfer agent",
			Schema: model.Applications{
				{"unSet": []any{"wbt_bt_agent_id"}},
				{"set": map[string]any{"ignore_display_updates": true}},
				{"joinAgent": map[string]any{"agent": map[string]any{"id": agentId}, "queue_name": "transfer"}},
				{"hangup": nil},
			},
		},
	}, nil
}

// ── chat routing ──────────────────────────────────────────────────────────────

func (a *SchemaAdapter) GetChatRouteFromProfile(domainId, profileId int64) (*model.Routing, error) {
	routing, err := a.store.Chat().RoutingFromProfile(domainId, profileId)
	if err != nil {
		return nil, fmt.Errorf("GetChatRouteFromProfile: store.chat.routing_from_profile: %w", err)
	}
	var appErr error
	routing.Schema, appErr = a.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	return routing, appErr
}

func (a *SchemaAdapter) GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, error) {
	routing, err := a.store.Chat().RoutingFromSchemaId(domainId, schemaId)
	if err != nil {
		return nil, fmt.Errorf("GetChatRouteFromSchemaId: store.chat.routing_from_schema: %w", err)
	}
	var appErr error
	routing.Schema, appErr = a.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	return routing, appErr
}

func (a *SchemaAdapter) GetChatRouteFromUserId(domainId int64, userId int64) (*model.Routing, error) {
	return &model.Routing{
		DomainId: domainId,
		Schema: &model.Schema{
			DomainId: domainId, Name: "transfer to user",
			Schema: model.Applications{
				{"bridge": map[string]interface{}{"userId": userId}},
			},
		},
	}, nil
}

// ── helpers ── (continued) ────────────────────────────────────────────────────

func toError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
