package resolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

// mockSubConn - це проста реалізація balancer.SubConn для тестів
type mockSubConn struct {
	balancer.SubConn

	id string
}

func TestRrPicker_Pick(t *testing.T) {
	// --- Arrange ---
	sc1 := &mockSubConn{id: "sc1"}
	sc2 := &mockSubConn{id: "sc2"}
	sc3 := &mockSubConn{id: "sc3"}

	builder := &rrPickerBuilder{}
	picker := builder.Build(base.PickerBuildInfo{
		ReadySCs: map[balancer.SubConn]base.SubConnInfo{
			sc1: {Address: resolver.Address{Addr: "addr1", ServerName: "id1"}},
			sc2: {Address: resolver.Address{Addr: "addr2", ServerName: "id2"}},
			sc3: {Address: resolver.Address{Addr: "addr3", ServerName: "id3"}},
		},
	})

	t.Run("Round robin picking", func(t *testing.T) {
		// --- Act & Assert ---
		// Виклики мають іти по колу. Оскільки початковий індекс випадковий,
		// ми просто перевіряємо, що за N викликів кожен сервер буде обрано.
		counts := make(map[string]int)

		for range 30 {
			res, err := picker.Pick(balancer.PickInfo{Ctx: context.Background()})
			assert.NoError(t, err)

			counts[res.SubConn.(*mockSubConn).id]++
		}

		assert.Equal(t, 10, counts["sc1"])
		assert.Equal(t, 10, counts["sc2"])
		assert.Equal(t, 10, counts["sc3"])
	})

	t.Run("Static host picking", func(t *testing.T) {
		// --- Arrange ---
		ctx := context.WithValue(context.Background(), StaticHostKey{}, StaticHost{Name: "id2"})

		// --- Act & Assert ---
		// Всі виклики мають іти до sc2
		for range 10 {
			res, err := picker.Pick(balancer.PickInfo{Ctx: ctx})
			assert.NoError(t, err)
			assert.Same(t, sc2, res.SubConn)
		}
	})

	t.Run("Static host not found", func(t *testing.T) {
		// --- Arrange ---
		ctx := context.WithValue(context.Background(), StaticHostKey{}, StaticHost{Name: "non-existent-id"})

		// --- Act & Assert ---
		_, err := picker.Pick(balancer.PickInfo{Ctx: ctx})
		assert.Error(t, err)
		assert.Equal(t, "no such host", err.Error())
	})
}
