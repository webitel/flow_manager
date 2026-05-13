// Package queue provides native ops for CC queue data retrieval:
// getQueueMetrics, getQueueInfo, getQueueAgents.
package queue

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/domain/flow"
	queuedomain "github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/store"
)

// Register adds getQueueMetrics, getQueueInfo, and getQueueAgents to reg.
func Register(reg *ops.Registry, s store.QueueStore) {
	reg.Register("getQueueMetrics", &getQueueMetricsOp{store: s})
	reg.Register("getQueueInfo", &getQueueInfoOp{store: s})
	reg.Register("getQueueAgents", &getQueueAgentsOp{store: s})
}

// ── getQueueMetrics ───────────────────────────────────────────────────────────

type getQueueMetricsOp struct{ store store.QueueStore }

func (o *getQueueMetricsOp) Kind() ops.OpKind { return ops.OpKindSync }

type getQueueMetricsArgs struct {
	Queue       *queuedomain.SearchEntity `json:"queue"`
	Bucket      *queuedomain.SearchEntity `json:"bucket"`
	Set         string              `json:"set"`
	Metric      string              `json:"metric"`
	Field       string              `json:"field"`
	Calls       string              `json:"calls"`
	LastMinutes int                 `json:"lastMinutes"`
	SlSec       int                 `json:"slSec"`
}

func (o *getQueueMetricsOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getQueueMetricsArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueMetrics: %w", err)
	}
	if argv.Queue == nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueMetrics: queue is required")
	}
	if argv.Set == "" {
		return ops.OpOutput{}, fmt.Errorf("getQueueMetrics: set is required")
	}

	var res float64
	if argv.Calls == "complete" && argv.Metric != "count" {
		req := &queuedomain.SearchQueueCompleteStatistics{
			QueueId:     argv.Queue.Id,
			QueueName:   argv.Queue.Name,
			LastMinutes: argv.LastMinutes,
			Metric:      argv.Metric,
			Field:       argv.Field,
			SlSec:       argv.SlSec,
		}
		if argv.Bucket != nil {
			req.BucketId = argv.Bucket.Id
			req.BucketName = argv.Bucket.Name
		}
		var err error
		if res, err = o.store.HistoryStatistics(in.DomainID, req); err != nil {
			return ops.OpOutput{}, fmt.Errorf("getQueueMetrics: %w", err)
		}
	}

	return ops.OpOutput{SetVars: map[string]string{argv.Set: fmt.Sprintf("%v", res)}}, nil
}

// ── getQueueInfo ──────────────────────────────────────────────────────────────

type getQueueInfoOp struct{ store store.QueueStore }

func (o *getQueueInfoOp) Kind() ops.OpKind { return ops.OpKindSync }

type getQueueInfoArgs struct {
	Queue *queuedomain.SearchEntity `json:"queue"`
	Set   flow.Variables     `json:"set"`
}

func (o *getQueueInfoOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getQueueInfoArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueInfo: %w", err)
	}
	if argv.Queue == nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueInfo: queue is required")
	}
	if len(argv.Set) == 0 {
		return ops.OpOutput{}, fmt.Errorf("getQueueInfo: set is required")
	}

	res, err := o.store.GetQueueData(in.DomainID, argv.Queue, argv.Set)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueInfo: %w", err)
	}

	setVars := make(map[string]string, len(res))
	for k, v := range res {
		setVars[k] = fmt.Sprintf("%v", v)
	}
	return ops.OpOutput{SetVars: setVars}, nil
}

// ── getQueueAgents ────────────────────────────────────────────────────────────

type getQueueAgentsOp struct{ store store.QueueStore }

func (o *getQueueAgentsOp) Kind() ops.OpKind { return ops.OpKindSync }

type getQueueAgentsArgs struct {
	Queue   *queuedomain.SearchEntity `json:"queue"`
	Channel string              `json:"channel"`
	Set     flow.Variables     `json:"set"`
}

func (o *getQueueAgentsOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getQueueAgentsArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueAgents: %w", err)
	}
	if argv.Queue == nil || argv.Queue.Id == nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueAgents: queue.id is required")
	}

	res, err := o.store.GetQueueAgents(in.DomainID, *argv.Queue.Id, argv.Channel, argv.Set)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("getQueueAgents: %w", err)
	}

	setVars := make(map[string]string, len(res))
	for k, v := range res {
		setVars[k] = fmt.Sprintf("%v", v)
	}
	return ops.OpOutput{SetVars: setVars}, nil
}
