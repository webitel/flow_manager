// Package meeting provides the "createMeeting" op — creates a meeting via the
// Webitel meeting service and stores the resulting URL in a flow variable.
package meeting

import (
	"context"
	"fmt"

	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type meetingOp struct {
	client domainmeeting.Client
}

// New returns an Op that calls the meeting service to create a meeting URL.
func New(client domainmeeting.Client) ops.Op {
	return &meetingOp{client: client}
}

func (m *meetingOp) Kind() ops.OpKind { return ops.OpKindSync }

type meetingArgs struct {
	SetVar    string            `json:"setVar"`
	Title     string            `json:"title,omitempty"`
	ExpireSec int64             `json:"expireSec,omitempty"`
	BasePath  string            `json:"basePath,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (m *meetingOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv meetingArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.SetVar == "" {
		return ops.OpOutput{}, fmt.Errorf("createMeeting: setVar is required")
	}

	url, err := m.client.Create(ctx, in.DomainID, argv.Title, int(argv.ExpireSec), argv.BasePath, argv.Variables)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("createMeeting: %w", err)
	}

	return ops.OpOutput{SetVars: map[string]string{argv.SetVar: url}}, nil
}
