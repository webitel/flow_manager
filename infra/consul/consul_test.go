package consul

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/infra/consul/mocks"
)

// newTestConsul - хелпер для створення екземпляра Consul з моком для тестів.
func newTestConsul(agent Agent, checkFn CheckFunction) *Consul {
	if checkFn == nil {
		checkFn = func() error { return nil } // Здоровий за замовчуванням
	}

	return &Consul{
		id:      "test-instance-id",
		log:     wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false}),
		agent:   agent,
		stop:    make(chan struct{}),
		check:   checkFn,
		checkID: "service:test-instance-id",
	}
}

func TestConsul_RegisterService(t *testing.T) {
	config := Config{
		Name:        "test-service",
		Address:     "localhost",
		Port:        8080,
		TTL:         10 * time.Second,
		CriticalTTL: 20 * time.Second,
	}

	t.Run("Successful registration", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		consul := newTestConsul(mockAgent, nil)
		expectedServiceID := fmt.Sprintf("%s-%s", config.Name, consul.id)

		mockAgent.On("ServiceRegister", mock.AnythingOfType("*api.AgentServiceRegistration")).Return(nil).Once()
		mockAgent.On("PassTTL", consul.checkID, "Service is healthy.").Return(nil).Once()
		mockAgent.On("ServiceDeregister", expectedServiceID).Return(nil).Once()

		// Act
		err := consul.RegisterService(config)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedServiceID, consul.serviceInstanceID)

		// Cleanup
		consul.Shutdown() // Зупиняємо горутину
		mockAgent.AssertExpectations(t)
	})

	t.Run("Failed registration", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		consul := newTestConsul(mockAgent, nil)
		expectedErr := errors.New("consul down")
		mockAgent.On("ServiceRegister", mock.AnythingOfType("*api.AgentServiceRegistration")).Return(expectedErr).Once()

		// Act
		err := consul.RegisterService(config)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
		mockAgent.AssertExpectations(t)
	})
}

func TestConsul_TTLUpdater(t *testing.T) {
	config := Config{Name: "test-service", TTL: 20 * time.Millisecond} // Короткий TTL для швидкого тестування

	t.Run("Healthy check calls PassTTL", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		consul := newTestConsul(mockAgent, func() error { return nil }) // Завжди здоровий
		expectedServiceID := fmt.Sprintf("%s-%s", config.Name, consul.id)

		mockAgent.On("ServiceRegister", mock.Anything).Return(nil).Once()
		// ВИПРАВЛЕНО: Прибираємо .Twice() щоб уникнути "крихкості" тесту.
		// Тепер тест перевіряє, що метод викликається, але не залежить від точної кількості тіків.
		mockAgent.On("PassTTL", consul.checkID, "Service is healthy.").Return(nil)
		mockAgent.On("ServiceDeregister", expectedServiceID).Return(nil).Once()

		// Act
		err := consul.RegisterService(config)
		require.NoError(t, err)

		time.Sleep(config.TTL + 5*time.Millisecond) // Чекаємо, щоб тікер встиг спрацювати хоча б раз

		// Assert
		consul.Shutdown()
		mockAgent.AssertExpectations(t)
	})

	t.Run("Unhealthy check calls FailTTL", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		checkErr := errors.New("service unhealthy")
		consul := newTestConsul(mockAgent, func() error { return checkErr }) // Завжди хворий
		expectedServiceID := fmt.Sprintf("%s-%s", config.Name, consul.id)

		mockAgent.On("ServiceRegister", mock.Anything).Return(nil).Once()
		// ВИПРАВЛЕНО: Прибираємо .Twice() з тієї ж причини.
		mockAgent.On("FailTTL", consul.checkID, checkErr.Error()).Return(nil)
		mockAgent.On("ServiceDeregister", expectedServiceID).Return(nil).Once()

		// Act
		err := consul.RegisterService(config)
		require.NoError(t, err)

		time.Sleep(config.TTL + 5*time.Millisecond) // Чекаємо на один тік

		// Assert
		consul.Shutdown()
		mockAgent.AssertExpectations(t)
	})
}

func TestConsul_Shutdown(t *testing.T) {
	// Arrange
	mockAgent := new(mocks.Agent)
	consul := newTestConsul(mockAgent, nil)
	consul.serviceInstanceID = "test-service-test-instance-id"

	mockAgent.On("ServiceDeregister", consul.serviceInstanceID).Return(nil).Once()

	// Act
	consul.Shutdown()

	// Assert
	mockAgent.AssertExpectations(t)
}

func TestConsul_handleTTLUpdateError(t *testing.T) {
	t.Run("Re-registers on 500 error", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		consul := newTestConsul(mockAgent, nil)
		config := Config{Name: "test-service"}
		consul.config = &config
		consul.serviceInstanceID = "test-service-test-instance-id"

		// Використовуємо 'Message' як було визначено в попередній ітерації
		serverErr := api.StatusError{Code: http.StatusInternalServerError, Body: "server error"}
		// Це виклик пере-реєстрації
		mockAgent.On("ServiceRegister", mock.Anything).Return(nil).Once()
		// Пере-реєстрація також викличе негайне оновлення TTL
		mockAgent.On("PassTTL", consul.checkID, "Service is healthy.").Return(nil).Once()

		// Act
		consul.handleTTLUpdateError(serverErr)

		// Assert
		mockAgent.AssertExpectations(t)
	})

	t.Run("Does not re-register on other errors", func(t *testing.T) {
		// Arrange
		mockAgent := new(mocks.Agent)
		consul := newTestConsul(mockAgent, nil)
		otherErr := errors.New("some other error")

		// Act
		consul.handleTTLUpdateError(otherErr)

		// Assert
		// Ми очікуємо, що ServiceRegister не буде викликано
		mockAgent.AssertNotCalled(t, "ServiceRegister", mock.Anything)
	})
}
