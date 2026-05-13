package call

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/webitel/wlog"

	genpb "github.com/webitel/flow_manager/gen/cc"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// ComplexDeps is the narrow interface required by bridge, joinQueue, and joinAgent ops.
type ComplexDeps interface {
	GetStore() store.Store
	GetMediaFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, error)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, error)
	JoinToInboundQueue(ctx context.Context, in *genpb.CallJoinToQueueRequest) (genpb.MemberService_CallJoinToQueueClient, error)
	JoinToAgent(ctx context.Context, in *genpb.CallJoinToAgentRequest) (genpb.MemberService_CallJoinToAgentClient, error)
}

// RegisterComplex adds call ops that use sub-flows or blocking gRPC streams.
func RegisterComplex(reg *ops.Registry, deps ComplexDeps) {
	reg.Register("bridge", &bridgeOp{deps: deps})
	reg.Register("joinQueue", &joinQueueOp{deps: deps})
	reg.Register("joinAgent", &joinAgentOp{deps: deps})
}

// hooksIndex returns the _hooks_index map from a node's Args.
func hooksIndex(node *tree.Node) map[string]int {
	if node == nil {
		return nil
	}
	idx, _ := node.Args["_hooks_index"].(map[string]int)
	return idx
}

// hookBranch returns the child branch for a named hook, or nil if absent.
func hookBranch(node *tree.Node, name string) *tree.Node {
	idx := hooksIndex(node)
	if idx == nil {
		return nil
	}
	i, ok := idx[name]
	if !ok || i >= len(node.Children) {
		return nil
	}
	return node.Children[i]
}

// snapVars makes a shallow copy of the variables map for use in goroutines.
func snapVars(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// runHook runs a named sub-flow asynchronously if the hook branch exists.
// vars is merged into the snapshot so the sub-flow sees updated call variables.
func runHook(ctx context.Context, in ops.OpInput, name string, extraVars map[string]string) {
	if in.RunBranch == nil {
		return
	}
	branch := hookBranch(in.Node, name)
	if branch == nil {
		return
	}
	v := snapVars(in.Variables)
	for k, val := range extraVars {
		v[k] = val
	}
	in.RunBranch(ctx, branch, v)
}

// ── bridge ────────────────────────────────────────────────────────────────────

type bridgeOp struct{ deps ComplexDeps }

func (bridgeOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *bridgeOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("bridge: no call connection in context")
	}

	props, ok2 := in.Node.RawArgs.(map[string]any)
	if !ok2 {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "bridge: bad arguments")
	}

	if _, ok2 = props["endpoints"]; !ok2 {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "bridge: endpoints required")
	}

	endpoints, appErr := replaceBridgeEndpoints(call, props["endpoints"])
	if appErr != nil {
		return ops.OpOutput{}, appErr
	}
	if len(endpoints) == 0 {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "bridge: no endpoints")
	}

	codecs, _ := arrayStrings(props["codecs"])

	remoteEndpoints, appErr := o.getRemoteEndpoints(call, endpoints)
	if appErr != nil {
		return ops.OpOutput{}, appErr
	}

	// "bridged" hook channel — the ESL Bridge sends on this when bridge is up.
	var br chan struct{}
	if branch := hookBranch(in.Node, "bridged"); branch != nil && in.RunBranch != nil {
		br = make(chan struct{}, 1)
		varSnap := snapVars(in.Variables)
		runBranch := in.RunBranch
		go func() {
			if _, ok := <-br; ok {
				go runBranch(ctx, branch, varSnap)
			}
		}()
	}

	vars := map[string]string{
		model.CallVariableSchemaIds: call.GetVariable(model.CallVariableSchemaIds),
	}
	if sendOnAnswer := mapStr(props, "sendOnAnswer"); sendOnAnswer != "" {
		sendOnAnswer = call.ParseText(sendOnAnswer)
		vars["execute_on_answer"] = "send_dtmf " + replaceQuotes(sendOnAnswer)
	}
	if glob, ok2 := props["parameters"].(map[string]any); ok2 {
		for k, v := range glob {
			vars[k] = fmt.Sprintf("%v", v)
		}
	}

	pickup := mapStr(props, "pickup")
	if pickup != "" {
		pickup = call.ParseText(pickup)
	}

	strategy := mapStr(props, "strategy")

	t := call.GetVariable("variable_transfer_history")
	_, appErr = call.Bridge(ctx, call, strategy, vars, remoteEndpoints, codecs, br, pickup)
	if appErr != nil {
		return ops.OpOutput{}, appErr
	}

	// Cancel flow if transfer happened or hangup_after_bridge is set.
	newT := call.GetVariable("variable_transfer_history")
	if t != newT && (call.GetVariable("variable_hangup_after_bridge") == "" || call.GetVariable("variable_hangup_after_bridge") == "true") {
		return ops.OpOutput{Break: true}, nil
	}
	if (call.GetVariable("variable_bridge_hangup_cause") == "NORMAL_CLEARING" || call.GetVariable("variable_last_bridge_hangup_cause") == "NORMAL_CLEARING") && call.GetVariable("variable_hangup_after_bridge") == "true" {
		return ops.OpOutput{Break: true}, nil
	}
	if call.GetVariable("variable_last_bridge_hangup_cause") == "ORIGINATOR_CANCEL" &&
		call.GetVariable("variable_originate_disposition") == "ORIGINATOR_CANCEL" &&
		call.GetVariable("variable_sip_redirect_dialstring") != "" &&
		call.GetVariable("variable_webitel_detect_redirect") != "false" {
		return ops.OpOutput{Break: true}, nil
	}

	return ops.OpOutput{}, nil
}

func (o *bridgeOp) getRemoteEndpoints(call model.Call, endpoints model.Applications) ([]*model.Endpoint, error) {
	length := len(endpoints)
	endp, storeErr := o.deps.GetStore().Endpoint().Get(int64(call.DomainId()), "NAME", "NUMBER", endpoints)
	if storeErr != nil {
		return nil, fmt.Errorf("getRemoteEndpoints: store.endpoint.get: %w", storeErr)
	}
	for key, e := range endp {
		if key > length {
			break
		}
		switch e.TypeName {
		case "gateway":
			if e.Destination != nil {
				e.Number = model.NewString(mapStr(endpoints[key], "dialString"))
				e.Name = model.NewString(mapStr(endpoints[key], "displayName"))
				if *e.Name == "" {
					e.Name = e.Number
				}
				e.Destination = model.NewString(fmt.Sprintf("%s@%s", *e.Number, *e.Destination))
			}
			e.Variables = endpointVars(endpoints[key], e.Variables)
		case "user":
			e.Variables = endpointVars(endpoints[key], e.Variables)
			if e.HasPush != nil && *e.HasPush {
				e.Variables = append(e.Variables, "execute_on_originate=wbt_send_hook")
			}
			if tmp := call.GetVariable("variable_wbt_contact_id"); tmp != "" {
				if !(call.UserId() > 0 && call.Direction() == model.CallDirectionOutbound) {
					e.Variables = append(e.Variables, fmt.Sprintf("wbt_contact_id=%s", tmp))
				}
			}
			if tmp := call.GetVariable("variable_wbt_hide_contact"); tmp != "" {
				e.Variables = append(e.Variables, fmt.Sprintf("wbt_hide_contact=%s", tmp))
			}
		default:
			wlog.Warn(fmt.Sprintf("call %s skip bridge endpoint %v - unknown type", call.Id(), e))
		}
	}
	return endp, nil
}

// replaceBridgeEndpoints marshals endpoints to JSON, expands FS vars via
// call.ParseText, then unmarshals back. Mirrors legacy replaceBridgeRequest.
func replaceBridgeEndpoints(call model.Call, arr any) (model.Applications, error) {
	data, err := json.Marshal(arr)
	if err != nil {
		return nil, apperrs.Newf(http.StatusBadRequest, "bridge: call.bridge.valid.endpoints: %s", err.Error())
	}
	var res model.Applications
	if err = json.Unmarshal([]byte(call.ParseText(string(data))), &res); err != nil {
		return nil, apperrs.Newf(http.StatusBadRequest, "bridge: call.bridge.valid.endpoints: %s", err.Error())
	}
	return res, nil
}

func endpointVars(src model.ApplicationObject, res []string) []string {
	if v, ok := src["parameters"].(map[string]any); ok {
		for k, vv := range v {
			res = append(res, fmt.Sprintf("'%s'='%s'", k, vv))
		}
	}
	return res
}

func mapStr(m any, key string) string {
	switch mm := m.(type) {
	case map[string]any:
		if v, ok := mm[key]; ok {
			switch s := v.(type) {
			case string:
				return s
			case map[string]any, []any:
				return ""
			default:
				return fmt.Sprint(s)
			}
		}
	case model.ApplicationObject:
		return mapStr(map[string]any(mm), key)
	}
	return ""
}

func arrayStrings(raw any) ([]string, bool) {
	arr, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, el := range arr {
		if s, ok := el.(string); ok {
			out = append(out, s)
		}
	}
	return out, true
}

func replaceQuotes(s string) string {
	out := make([]byte, 0, len(s))
	for i := range s {
		if s[i] != '\'' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

// ── joinQueue (call) ──────────────────────────────────────────────────────────

type joinQueueOp struct{ deps ComplexDeps }

func (joinQueueOp) Kind() ops.OpKind { return ops.OpKindSync }

type joinQueueArgs struct {
	Name     string `json:"name"`
	Number   string `json:"number"`
	Priority int32  `json:"priority"`
	Queue    struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"queue"`
	BucketId int32 `json:"bucket_id"` // deprecated
	Bucket   struct {
		Id int32 `json:"id"`
	} `json:"bucket"`
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	} `json:"agent"`
	StickyAgentId       int32               `json:"stickyAgentId"`
	Ringtone            model.PlaybackFile  `json:"ringtone"`
	Timers              []callTimerArg      `json:"timers"`
	TransferAfterBridge *model.SearchEntity `json:"transferAfterBridge"`
}

type callTimerArg struct {
	Interval    int `json:"interval"`
	Tries       int `json:"tries"`
	Offset      int `json:"offset"`
	ChildrenIdx int `json:"_children_idx"`
}

func (o *joinQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("joinQueue: no call connection in context")
	}
	if call.InQueue() {
		return ops.OpOutput{}, fmt.Errorf("call.queue.in_queue: call is in queue")
	}

	var q joinQueueArgs
	if err := ops.DecodeArgs(in, &q); err != nil {
		return ops.OpOutput{}, err
	}

	if q.BucketId > 0 && q.Bucket.Id == 0 {
		q.Bucket.Id = q.BucketId
	}

	// Waiting sub-flow runs concurrently during queue wait.
	wCtx, wCancel := context.WithCancel(ctx)
	defer func() { wCancel() }()
	if branch := hookBranch(in.Node, "waiting"); branch != nil && in.RunBranch != nil {
		in.RunBranch(wCtx, branch, snapVars(in.Variables))
	}

	// Timers.
	callStartTimers(wCtx, q.Timers, in)

	if q.TransferAfterBridge != nil && q.TransferAfterBridge.Id != nil {
		if _, appErr := call.SetTransferAfterBridge(ctx, *q.TransferAfterBridge.Id); appErr != nil {
			return ops.OpOutput{}, appErr
		}
	}

	t := call.GetVariable("variable_transfer_history")

	// Ringtone resolution.
	var ringtone *genpb.CallJoinToQueueRequest_WaitingMusic
	if q.Ringtone.Name != nil || q.Ringtone.Id != nil {
		req := []*model.PlaybackFile{{Id: q.Ringtone.Id, Name: q.Ringtone.Name}}
		if res, appErr := o.deps.GetMediaFiles(call.DomainId(), &req); appErr == nil && len(res) > 0 && res[0] != nil && res[0].Type != nil {
			ringtone = &genpb.CallJoinToQueueRequest_WaitingMusic{
				Id:   int32(*res[0].Id),
				Type: *res[0].Type,
			}
		}
	}

	// Sticky agent resolution.
	var stickyAgentId int32
	if q.Agent != nil {
		if q.Agent.Extension != nil && q.Agent.Id == nil {
			q.Agent.Id, _ = o.deps.GetAgentIdByExtension(call.DomainId(), *q.Agent.Extension)
		}
		if q.Agent.Id != nil {
			stickyAgentId = *q.Agent.Id
		}
	} else {
		stickyAgentId = q.StickyAgentId
	}

	vars := call.DumpExportVariables()
	if cid := call.GetContactId(); cid != 0 {
		vars["wbt_contact_id"] = fmt.Sprintf("%d", cid)
	}
	if call.MeetingId() != "" {
		vars["sip_h_X-Webitel-Meeting"] = call.MeetingId()
	}

	if call.Stopped() || call.HangupCause() != "" {
		return ops.OpOutput{}, nil
	}

	qCtx, cancelQueue := context.WithCancel(context.Background())
	stream, appErr := o.deps.JoinToInboundQueue(qCtx, &genpb.CallJoinToQueueRequest{
		MemberCallId: call.Id(),
		Queue: &genpb.CallJoinToQueueRequest_Queue{
			Id:   int32(q.Queue.Id),
			Name: q.Queue.Name,
		},
		WaitingMusic:  ringtone,
		Priority:      q.Priority,
		BucketId:      q.Bucket.Id,
		Variables:     vars,
		DomainId:      call.DomainId(),
		StickyAgentId: stickyAgentId,
		IsTransfer:    call.TransferQueueId() > 0 && !call.IsBlindTransferQueue(),
	})
	if appErr != nil {
		call.Log().Err(appErr)
		return ops.OpOutput{}, nil
	}

	call.SetQueueCancel(cancelQueue)
	defer call.SetQueueCancel(nil)

	for {
		var msg genpb.QueueEvent
		err := stream.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			wlog.Error(err.Error())
			return ops.OpOutput{}, nil
		}

		switch e := msg.Data.(type) {
		case *genpb.QueueEvent_Offering:
			runHook(context.Background(), in, "offering", map[string]string{
				"cc_agent_name":    e.Offering.AgentName,
				"cc_agent_call_id": e.Offering.AgentCallId,
				"cc_agent_id":      fmt.Sprintf("%d", e.Offering.AgentId),
			})

		case *genpb.QueueEvent_Bridged:
			call.SetQueueCancel(nil)
			wCancel()
			runHook(context.Background(), in, "bridged", nil)

		case *genpb.QueueEvent_Leaving:
			runHook(context.Background(), in, "reporting", map[string]string{
				"cc_result": e.Leaving.Result,
			})
		}
	}

	if t != call.GetVariable("variable_transfer_history") {
		return ops.OpOutput{Break: true}, nil
	}
	return ops.OpOutput{}, nil
}

// callStartTimers launches timer goroutines for joinQueue (call channel).
// Timer child nodes are already extracted by parseJoinQueueTimers.
func callStartTimers(ctx context.Context, timers []callTimerArg, in ops.OpInput) {
	if in.RunBranch == nil || in.Node == nil || len(timers) == 0 {
		return
	}
	varSnap := snapVars(in.Variables)
	for _, t := range timers {
		t := t
		if t.Interval <= 0 {
			continue
		}
		if t.ChildrenIdx < 0 || t.ChildrenIdx >= len(in.Node.Children) {
			continue
		}
		branch := in.Node.Children[t.ChildrenIdx]
		go callRunTimer(ctx, t, branch, varSnap, in.RunBranch)
	}
}

func callRunTimer(ctx context.Context, t callTimerArg, branch *tree.Node, varSnap map[string]string, runBranch func(context.Context, *tree.Node, map[string]string)) {
	tries := t.Tries
	if tries <= 0 {
		tries = 999
	}
	interval := time.Duration(t.Interval) * time.Second
	timer := time.NewTimer(interval)
	defer timer.Stop()
	fired := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runBranch(ctx, branch, varSnap)
			fired++
			if fired >= tries {
				return
			}
			interval += time.Duration(t.Offset) * time.Second
			if interval < time.Second {
				return
			}
			timer.Reset(interval)
		}
	}
}

// ── joinAgent ─────────────────────────────────────────────────────────────────

type joinAgentOp struct{ deps ComplexDeps }

func (joinAgentOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *joinAgentOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("joinAgent: no call connection in context")
	}

	var argv struct {
		Agent *struct {
			Id        *int32  `json:"id"`
			Extension *string `json:"extension"`
		} `json:"agent"`
		Processing       *model.Processing  `json:"processing"`
		Ringtone         model.PlaybackFile `json:"ringtone"`
		Timeout          int32              `json:"timeout"`
		QueueName        string             `json:"queue_name"`
		CancelDistribute bool               `json:"cancel_distribute"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Agent == nil {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "joinAgent: agent required")
	}

	var agentId *int32
	if argv.Agent.Id == nil && argv.Agent.Extension != nil {
		agentId, _ = o.deps.GetAgentIdByExtension(call.DomainId(), *argv.Agent.Extension)
	} else {
		agentId = argv.Agent.Id
	}
	if agentId == nil {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "joinAgent: agent not found")
	}

	t := call.GetVariable("variable_transfer_history")

	// Ringtone resolution.
	var ringtone *genpb.CallJoinToAgentRequest_WaitingMusic
	if argv.Ringtone.Name != nil || argv.Ringtone.Id != nil {
		req := []*model.PlaybackFile{{Id: argv.Ringtone.Id, Name: argv.Ringtone.Name}}
		if res, appErr := o.deps.GetMediaFiles(call.DomainId(), &req); appErr == nil && len(res) > 0 && res[0] != nil && res[0].Type != nil {
			ringtone = &genpb.CallJoinToAgentRequest_WaitingMusic{
				Id:   int32(*res[0].Id),
				Type: *res[0].Type,
			}
		}
	}

	req := &genpb.CallJoinToAgentRequest{
		DomainId:         call.DomainId(),
		MemberCallId:     call.Id(),
		AgentId:          *agentId,
		WaitingMusic:     ringtone,
		Timeout:          argv.Timeout,
		Variables:        call.DumpExportVariables(),
		QueueName:        argv.QueueName,
		CancelDistribute: argv.CancelDistribute,
		IsTransfer:       call.TransferAgentId() > 0,
	}

	if argv.Processing != nil && argv.Processing.Enabled {
		req.Processing = &genpb.CallJoinToAgentRequest_Processing{
			Enabled:    true,
			RenewalSec: argv.Processing.RenewalSec,
			Sec:        argv.Processing.Sec,
		}
		if argv.Processing.Form.Id > 0 {
			req.Processing.Form = &genpb.QueueFormSchema{Id: argv.Processing.Form.Id}
		}
		if argv.Processing.Prolongation != nil && argv.Processing.Prolongation.Enabled {
			req.Processing.ProcessingProlongation = &genpb.ProcessingProlongation{
				Enabled:             true,
				RepeatsNumber:       argv.Processing.Prolongation.RepeatsNumber,
				ProlongationTimeSec: argv.Processing.Prolongation.ProlongationTimeSec,
				IsTimeoutRetry:      argv.Processing.Prolongation.IsTimeoutRetry,
			}
		}
	}

	stream, appErr := o.deps.JoinToAgent(ctx, req)
	if appErr != nil {
		call.Log().Err(appErr)
		return ops.OpOutput{}, nil
	}

	for {
		var msg genpb.QueueEvent
		err := stream.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			wlog.Error(err.Error())
			return ops.OpOutput{}, nil
		}

		switch e := msg.Data.(type) {
		case *genpb.QueueEvent_Joined:
			call.Set(ctx, model.Variables{"attempt_id": e.Joined.AttemptId}) //nolint:errcheck

		case *genpb.QueueEvent_Bridged:
			agentExt := call.GetVariable("Caller-Caller-ID-Number")
			runHook(context.Background(), in, "bridged", map[string]string{
				"agent_id":        fmt.Sprintf("%d", e.Bridged.AgentId),
				"agent_extension": agentExt,
			})

		case *genpb.QueueEvent_Leaving:
			call.Set(ctx, model.Variables{"cc_result": e.Leaving.Result}) //nolint:errcheck
		}
	}

	if t != call.GetVariable("variable_transfer_history") {
		return ops.OpOutput{Break: true}, nil
	}
	return ops.OpOutput{}, nil
}
