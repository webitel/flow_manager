package grpc

import (
	"github.com/webitel/engine/discovery"
	"testing"
)

func TestChatClient(t *testing.T) {
	t.Log("TestChatClient")

	service, _ := discovery.NewServiceDiscovery("Chat-Test", "10.9.8.111:8500", func() (bool, error) {
		return true, nil
	})

	cm := NewChatManager(service)
	if err := cm.Start(); err != nil {
		t.Error(err.Error())
	}

	defer cm.Stop()
}
