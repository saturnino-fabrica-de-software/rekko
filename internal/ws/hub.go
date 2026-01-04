package ws

import (
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

func (h *Hub) Run() {
	for {
		select {
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
	defer h.mu.RUnlock()

	clients := h.tenants[event.TenantID]
	if clients == nil {
		return
	}

	message, err := json.Marshal(event)
	if err != nil {
		return
	}

	for client := range clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
			delete(h.tenants[event.TenantID], client)
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
