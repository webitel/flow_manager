package grpc

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/gen/chat/messages"
	"sync"
	"time"

	proto "github.com/webitel/flow_manager/gen/chat"
	"github.com/webitel/flow_manager/model"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/wlog"
)

type ChatManager struct {
	serviceDiscovery discovery.ServiceDiscovery
	poolConnections  discovery.Pool

	watcher   *discovery.Watcher
	startOnce sync.Once
	stop      chan struct{}
	stopped   chan struct{}
	log       *wlog.Logger
}

func NewChatManager() *ChatManager {
	return &ChatManager{
		stop:            make(chan struct{}),
		stopped:         make(chan struct{}),
		poolConnections: discovery.NewPoolConnections(),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "chat manager"),
		),
	}
}

func (cm *ChatManager) Start(sd discovery.ServiceDiscovery) error {
	cm.log.Debug("starting chat client")
	cm.serviceDiscovery = sd

	if services, err := cm.serviceDiscovery.GetByName(ChatClientService); err != nil {
		return err
	} else {
		for _, v := range services {
			cm.registerConnection(v)
		}
	}

	cm.startOnce.Do(func() {
		cm.watcher = discovery.MakeWatcher("chat client", WatcherInterval, cm.wakeUp)
		go cm.watcher.Start()
		go func() {
			defer func() {
				cm.log.Debug("stopped")
				close(cm.stopped)
			}()

			for {
				select {
				case <-cm.stop:
					cm.log.Debug("received stop signal")
					return
				}
			}
		}()
	})
	return nil
}

func (cm *ChatManager) Stop() {
	if cm.watcher != nil {
		cm.watcher.Stop()
	}

	if cm.poolConnections != nil {
		cm.poolConnections.CloseAllConnections()
	}

	close(cm.stop)
	<-cm.stopped
}

func (cm *ChatManager) registerConnection(v *discovery.ServiceConnection) {
	addr := fmt.Sprintf("%s:%d", v.Host, v.Port)
	c, err := NewChatClientConnection(v.Id, addr)
	if err != nil {
		cm.log.Error(err.Error(),
			wlog.String("connection_id", v.Id),
			wlog.String("connection_addr", addr),
		)
		return
	}
	cm.poolConnections.Append(c)
	cm.log.Debug("register",
		wlog.String("connection_id", v.Id),
		wlog.String("connection_name", c.Name()),
		wlog.String("connection_addr", addr),
	)
}

func (cm *ChatManager) getClient(name string) (*ChatClientConnection, error) {
	conn, err := cm.poolConnections.GetById(name)
	if err != nil {
		return nil, err
	}
	return conn.(*ChatClientConnection), nil
}

func (cm *ChatManager) getRandCli() (*ChatClientConnection, error) {
	conn, err := cm.poolConnections.Get(discovery.StrategyRoundRobin)
	if err != nil {
		return nil, err
	}
	return conn.(*ChatClientConnection), nil
}

func (cm *ChatManager) wakeUp() {
	list, err := cm.serviceDiscovery.GetByName(ChatClientService)
	if err != nil {
		cm.log.Err(err)
		return
	}

	for _, v := range list {
		if _, err := cm.poolConnections.GetById(v.Id); err == discovery.ErrNotFoundConnection {
			cm.registerConnection(v)
		}
	}
	cm.poolConnections.RecheckConnections(list.Ids())
}

func inputFile(f *model.File) *messages.InputFile {
	if f == nil {
		return nil
	}

	if len(f.Url) != 0 {
		return &messages.InputFile{
			FileSource: &messages.InputFile_Url{
				Url: f.Url,
			},
		}
	} else {
		return &messages.InputFile{
			FileSource: &messages.InputFile_Id{
				Id: fmt.Sprintf("%d", f.Id),
			},
		}
	}
}

func inputKeyboard(btns [][]model.ChatButton) *messages.InputKeyboard {
	l := len(btns)
	if l == 0 {
		return nil
	}

	keyboard := &messages.InputKeyboard{
		Rows: make([]*messages.InputButtonRow, 0, l),
	}

	for _, row := range btns {
		l = len(row)
		if l == 0 {
			continue
		}

		buttons := make([]*messages.InputButton, 0, l)

		for _, btn := range row {
			buttons = append(buttons, &messages.InputButton{
				Caption: btn.Caption,
				Text:    btn.Text,
				Type:    btn.Type,
				Url:     btn.Url,
				Code:    btn.Code,
			})
		}

		keyboard.Rows = append(keyboard.Rows, &messages.InputButtonRow{
			Buttons: buttons,
		})
	}

	return keyboard
}

func inputPeers(ps []model.BroadcastPeer) []*messages.InputPeer {
	peers := make([]*messages.InputPeer, 0, len(ps))

	for _, v := range ps {
		peers = append(peers, &messages.InputPeer{
			Id:   v.Id,
			Type: v.Type,
			Via:  v.Via,
		})
	}

	return peers
}

func (cc *ChatManager) BroadcastMessage(ctx context.Context, domainId int64, req model.BroadcastChat, peers []model.BroadcastPeer) (*model.BroadcastChatResponse, error) {
	c, e := cc.getRandCli()
	if e != nil {
		return nil, e
	}

	msg := &messages.InputMessage{
		Text:     req.Text,
		File:     inputFile(req.File),
		Keyboard: inputKeyboard(req.Buttons),
	}

	var newContext context.Context
	if req.Timeout != 0 {
		newContext, _ = context.WithTimeout(context.Background(), time.Duration(req.Timeout+5)*time.Millisecond)
	} else {
		newContext = ctx
	}

	broadcastResponse, e := c.messages.BroadcastMessageNA(newContext, &messages.BroadcastMessageRequest{
		Message:   msg,
		Peers:     inputPeers(peers),
		Timeout:   req.Timeout,
		Variables: req.Variables,
	})

	if e != nil {
		return nil, e
	}

	res := model.BroadcastChatResponse{
		Failed: make([]*model.FailedReceiver, 0),
	}

	for _, peer := range broadcastResponse.GetFailure() {
		res.Failed = append(res.Failed, &model.FailedReceiver{Id: peer.PeerId, Error: peer.Error.Message})
	}
	res.Variables = broadcastResponse.Variables
	return &res, nil
}

func (cc *ChatManager) LinkContact(ctx context.Context, contactId string, conversationId string) error {
	c, e := cc.getRandCli()
	if e != nil {
		return e
	}
	_, e = c.contacts.LinkContactToClientNA(ctx, &messages.LinkContactToClientNARequest{
		ConversationId: conversationId,
		ContactId:      contactId,
	})
	if e != nil {
		return e
	}
	return nil
}

func (cc *ChatManager) SendAction(ctx context.Context, channelId string, action model.ChatAction) error {
	c, e := cc.getRandCli()
	if e != nil {
		return e
	}

	var a proto.UserAction = proto.UserAction_Typing

	switch action {
	case model.ChatActionCancel:
		a = proto.UserAction_Cancel

	}

	msg := &proto.SendUserActionRequest{
		ChannelId: channelId,
		Action:    a,
	}

	if _, e = c.messages.SendUserAction(ctx, msg); e != nil {
		return e
	}

	return nil
}
