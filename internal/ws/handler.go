package ws

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

func Handler(hub *Hub) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		tenantIDValue := c.Locals("tenant_id")
		if tenantIDValue == nil {
			_ = c.Close()
			return
		}

		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			_ = c.Close()
			return
		}

		client := &Client{
			hub:      hub,
			conn:     c,
			tenantID: tenantID,
			send:     make(chan []byte, 256),
		}

		hub.register <- client

		go client.WritePump()
		client.ReadPump()
	})
}

func UpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}
