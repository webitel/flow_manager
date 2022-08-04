package grpc

import (
	"testing"

	"github.com/webitel/engine/discovery"
)

func TestChatClient(t *testing.T) {
	t.Log("TestChatClient")

	service, _ := discovery.NewServiceDiscovery("Chat-Test", "10.9.8.111:8500", func() (bool, error) {
		return true, nil
	})

	cm := NewChatManager()
	if err := cm.Start(service); err != nil {
		t.Error(err.Error())
	}

	defer cm.Stop()
}
