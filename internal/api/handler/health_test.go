package handler

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealthHandler_Health(t *testing.T) {
	app := fiber.New()
	handler := NewHealthHandler()
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result HealthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Status = %s, want ok", result.Status)
	}

	if result.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	app := fiber.New()
	handler := NewHealthHandler()
	app.Get("/ready", handler.Ready)

	req := httptest.NewRequest("GET", "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result HealthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Status != "ready" {
		t.Errorf("Status = %s, want ready", result.Status)
	}
}
