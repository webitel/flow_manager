package client

import (
	workflow "buf.build/gen/go/webitel/workflow/protocolbuffers/go"
	"context"
	"sync"
)

type QueueProcessing struct {
	cli  *fConnection
	form *workflow.Form
	sync.RWMutex
	fields map[string]string
}

func (q *queueApi) NewProcessing(ctx context.Context, domainId int64, schemaId int, vars map[string]string) (*QueueProcessing, error) {
	cli, err := q.cli.getRandomClient()
	if err != nil {
		return nil, err
	}

	qp := &QueueProcessing{
		cli:    cli,
		fields: make(map[string]string),
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
	if p == nil {
		return nil
	}
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
	for k, v := range vars {
		p.fields[k] = v
	}
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

func (p *QueueProcessing) Fields() map[string]string {
	p.RLock()
	defer p.RUnlock()

	return p.fields
}

func (p *QueueProcessing) Update(f []byte, fields map[string]string) error {
	if p == nil {
		return nil
	}
	p.Lock()
	for k, v := range fields {
		p.fields[k] = v
	}
	p.form.Form = f
	p.Unlock()
	return nil
}
