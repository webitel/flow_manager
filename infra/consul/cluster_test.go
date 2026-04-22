package consul

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/infra/consul/mocks"
)

func TestCluster_Start(t *testing.T) {
	// Зберігаємо і відновлюємо оригінальні значення
	originalNewConsul := newConsul
	originalReconnectDuration := reconnectDuration

	t.Cleanup(func() {
		newConsul = originalNewConsul
		reconnectDuration = originalReconnectDuration
	})

	// Робимо тести швидкими
	reconnectDuration = 1 * time.Millisecond

	cluster := NewCluster("test-app", "consul:8500", wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false}))

	t.Run("Successful registration on first attempt", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		mockConsulInstance := &Consul{
			id:    "test-id",
			agent: mockAgent,
			log:   wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true}),
			stop:  make(chan struct{}),
			check: func() error { return nil },
		}
		newConsul = func(id, consulAgentAddr string, log *wlog.Logger, check CheckFunction) (*Consul, error) {
			return mockConsulInstance, nil
		}

		mockAgent.On("ServiceRegister", mock.Anything).Return(nil).Once()
		mockAgent.On("PassTTL", mock.Anything, mock.Anything).Return(nil).Once()
		mockAgent.On("ServiceDeregister", mock.Anything).Return(nil).Once() // Для Shutdown

		// Act
		err := cluster.Start("test-id", "localhost", 80)

		// Assert
		require.NoError(t, err)
		cluster.Stop() // Перевіряємо, що Stop викликає дереєстрацію
		mockAgent.AssertExpectations(t)
	})

	t.Run("Successful registration after retries", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		mockConsulInstance := &Consul{
			id:    "test-id",
			agent: mockAgent,
			log:   wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true}),
			stop:  make(chan struct{}),
			check: func() error { return nil },
		}
		newConsul = func(id, consulAgentAddr string, log *wlog.Logger, check CheckFunction) (*Consul, error) {
			return mockConsulInstance, nil
		}

		// Помилка двічі, потім успіх
		mockAgent.On("ServiceRegister", mock.Anything).Return(errors.New("fail")).Twice()
		mockAgent.On("ServiceRegister", mock.Anything).Return(nil).Once()
		mockAgent.On("PassTTL", mock.Anything, mock.Anything).Return(nil).Once()
		mockAgent.On("ServiceDeregister", mock.Anything).Return(nil).Once()

		// Act
		err := cluster.Start("test-id", "localhost", 80)

		// Assert
		require.NoError(t, err)
		cluster.Stop()
		mockAgent.AssertExpectations(t)
	})

	t.Run("Failed registration after all attempts", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		mockConsulInstance := &Consul{
			id:    "test-id",
			agent: mockAgent,
			log:   wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: true}),
			stop:  make(chan struct{}),
			check: func() error { return nil },
		}
		newConsul = func(id, consulAgentAddr string, log *wlog.Logger, check CheckFunction) (*Consul, error) {
			return mockConsulInstance, nil
		}

		// Помилка всі 10 разів
		mockAgent.On("ServiceRegister", mock.Anything).Return(errors.New("fail")).Times(defaultReconnectAttempts)

		// Act
		err := cluster.Start("test-id", "localhost", 80)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeded maximum reconnect attempts")
		mockAgent.AssertExpectations(t)
	})
}
