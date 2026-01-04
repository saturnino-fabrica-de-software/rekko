package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Hub struct {
	clients    map[*Client]bool
	tenants    map[uuid.UUID]map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		tenants:    make(map[uuid.UUID]map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case event := <-h.broadcast:
			h.broadcastToTenant(event)
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	if h.tenants[client.tenantID] == nil {
		h.tenants[client.tenantID] = make(map[*Client]bool)
	}
	h.tenants[client.tenantID][client] = true
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		delete(h.tenants[client.tenantID], client)

		if len(h.tenants[client.tenantID]) == 0 {
			delete(h.tenants, client.tenantID)
		}

		close(client.send)
	}
}

func (h *Hub) broadcastToTenant(event Event) {
	h.mu.RLock()
	clients, ok := h.tenants[event.TenantID]
	if !ok {
		h.mu.RUnlock()
		return
	}

	clientList := make([]*Client, 0, len(clients))
	for client := range clients {
		clientList = append(clientList, client)
	}
	h.mu.RUnlock()

	message, err := json.Marshal(event)
	if err != nil {
		return
	}

	for _, client := range clientList {
		select {
		case client.send <- message:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) BroadcastToTenant(tenantID uuid.UUID, eventType EventType, data interface{}) {
	event := Event{
		TenantID:  tenantID,
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- event:
	default:
	}
}

func (h *Hub) GetConnectedClients(tenantID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.tenants[tenantID])
}
