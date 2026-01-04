package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.tenants)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHub_AddAndRemoveClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	tenantID := uuid.New()
	client := &Client{
		hub:      hub,
		tenantID: tenantID,
		send:     make(chan []byte, 1),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.GetConnectedClients(tenantID))

	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.GetConnectedClients(tenantID))
}

func TestHub_BroadcastToTenant(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	tenantID := uuid.New()
	client := &Client{
		hub:      hub,
		tenantID: tenantID,
		send:     make(chan []byte, 10),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	testData := map[string]string{"message": "test"}
	hub.BroadcastToTenant(tenantID, EventFaceRegistered, testData)

	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.send:
		var event Event
		err := json.Unmarshal(msg, &event)
		assert.NoError(t, err)
		assert.Equal(t, EventFaceRegistered, event.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHub_TenantIsolation(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	client1 := &Client{
		hub:      hub,
		tenantID: tenant1,
		send:     make(chan []byte, 10),
	}

	client2 := &Client{
		hub:      hub,
		tenantID: tenant2,
		send:     make(chan []byte, 10),
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	testData := map[string]string{"message": "only for tenant1"}
	hub.BroadcastToTenant(tenant1, EventAlert, testData)

	time.Sleep(50 * time.Millisecond)

	select {
	case <-client1.send:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("client1 should receive message")
	}

	select {
	case <-client2.send:
		t.Fatal("client2 should not receive message from tenant1")
	case <-time.After(100 * time.Millisecond):
	}
}
