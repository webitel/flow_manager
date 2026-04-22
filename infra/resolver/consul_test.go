package resolver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/webitel/flow_manager/infra/resolver/mocks"
)

func TestWatchConsulService(t *testing.T) {
	t.Run("Successfully fetches and sends services", func(t *testing.T) {
		// --- Arrange ---
		mockServicer := new(mocks.Servicer)
		out := make(chan []serviceMeta, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		tgt := target{Service: "test-service", Healthy: true}
		serviceEntries := []*api.ServiceEntry{
			{
				Service: &api.AgentService{ID: "id1", Address: "10.0.0.1", Port: 8080},
				Node:    &api.Node{Address: "10.0.0.1"},
			},
			{
				Service: &api.AgentService{ID: "id2", Address: "10.0.0.2", Port: 8080},
				Node:    &api.Node{Address: "10.0.0.2"},
			},
		}
		queryMeta := &api.QueryMeta{LastIndex: 123}

		// 1. Налаштовуємо мок для ПЕРШОГО виклику. Ми очікуємо його лише один раз.
		mockServicer.On("Service", tgt.Service, tgt.Tag, tgt.Healthy, mock.AnythingOfType("*api.QueryOptions")).
			Return(serviceEntries, queryMeta, nil).
			Once()

		mockServicer.On("Service", tgt.Service, tgt.Tag, tgt.Healthy, mock.AnythingOfType("*api.QueryOptions")).
			Return(nil, &api.QueryMeta{}, nil)

		// --- Act ---
		go watchConsulService(ctx, mockServicer, tgt, out)

		// --- Assert ---
		select {
		case result := <-out:
			assert.Len(t, result, 2)
			assert.Equal(t, "10.0.0.1:8080", result[0].addr)
			assert.Equal(t, "id1", result[0].id)
			assert.Equal(t, "10.0.0.2:8080", result[1].addr)
			assert.Equal(t, "id2", result[1].id)
		case <-ctx.Done():
			t.Fatal("Test timed out, did not receive services")
		}

		// Перевіряємо, що очікування, позначені як .Once(), були виконані.
		mockServicer.AssertExpectations(t)
	})

	t.Run("Handles error from consul", func(t *testing.T) {
		// --- Arrange ---
		mockServicer := new(mocks.Servicer)
		out := make(chan []serviceMeta, 1)
		// Короткий таймаут, бо ми не очікуємо нічого в каналі
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		tgt := target{Service: "test-service", Healthy: true, MaxBackoff: 10 * time.Millisecond}
		consulErr := errors.New("consul is down")

		// Налаштовуємо мок: повертаємо помилку для всіх викликів
		mockServicer.On("Service", tgt.Service, tgt.Tag, tgt.Healthy, mock.Anything).Return(nil, nil, consulErr)

		// --- Act ---
		go watchConsulService(ctx, mockServicer, tgt, out)

		// --- Assert ---
		// Переконуємось, що нічого не було відправлено в канал
		select {
		case <-out:
			t.Fatal("Received services when an error was expected")
		case <-time.After(40 * time.Millisecond):
			// Все добре, нічого не прийшло
		}

		mockServicer.AssertExpectations(t)
	})
}
