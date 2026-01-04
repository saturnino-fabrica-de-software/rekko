package ws

import (
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	tenantID uuid.UUID
	send     chan []byte
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		_ = c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}
