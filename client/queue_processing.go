package client

import (
	"context"
	"sync"

	"github.com/webitel/protos/workflow"
)

type QueueProcessing struct {
	cli  *fConnection
	form *workflow.Form
	sync.RWMutex
}

func (q *queueApi) NewProcessing(ctx context.Context, domainId int64, schemaId int, vars map[string]string) (*QueueProcessing, error) {
	cli, err := q.cli.getRandomClient()
	if err != nil {
		return nil, err
	}

	qp := &QueueProcessing{
		cli: cli,
	}
	qp.form, err = cli.processing.StartProcessing(ctx, &workflow.StartProcessingRequest{
		SchemaId:  uint32(schemaId),
		DomainId:  domainId,
		Variables: vars,
	})
	if err != nil {
		return nil, err
	}

	return qp, nil
}

func (p *QueueProcessing) Form() []byte {
	p.RLock()
	defer p.RUnlock()

	return p.form.Form
}

func (p *QueueProcessing) Id() string {
	p.RLock()
	defer p.RUnlock()

	return p.form.Id
}

func (p *QueueProcessing) ActionForm(ctx context.Context, action string, vars map[string]string) ([]byte, error) {
	f, err := p.cli.processing.FormAction(ctx, &workflow.FormActionRequest{
		Id:        p.Id(),
		Action:    action,
		Variables: vars,
	})
	if err != nil {
		return nil, err
	}
	p.Lock()
	p.form = f
	p.Unlock()

	return p.form.Form, nil
}

func (p *QueueProcessing) Close() error {
	_, err := p.cli.processing.CancelProcessing(context.Background(), &workflow.CancelProcessingRequest{
		Id: p.Id(),
	})

	return err
}
