package handler

import (
	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status:  "ok",
		Version: "0.1.0",
	})
}

func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	// TODO: Add database connectivity check
	return c.JSON(HealthResponse{
		Status: "ready",
	})
}
