package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
)

type APIKeysHandler struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewAPIKeysHandler(db *pgxpool.Pool, logger *slog.Logger) *APIKeysHandler {
	return &APIKeysHandler{
		db:     db,
		logger: logger,
	}
}

type APIKeyResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	KeyPrefix   string  `json:"key_prefix"`
	Environment string  `json:"environment"`
	IsActive    bool    `json:"is_active"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type APIKeysResponse struct {
	PublicKey string           `json:"public_key"`
	Keys      []APIKeyResponse `json:"keys"`
}

func (h *APIKeysHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals(middleware.LocalTenantID).(uuid.UUID)
	if !ok {
		h.logger.Warn("tenant ID not found in context")
		return fiber.ErrUnauthorized
	}

	// Get tenant's public_key
	var publicKey *string
	err := h.db.QueryRow(c.Context(), `
		SELECT public_key FROM tenants WHERE id = $1
	`, tenantID).Scan(&publicKey)
	if err != nil {
		h.logger.Error("failed to get tenant public key", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}

	// Get all API keys for tenant
	rows, err := h.db.Query(c.Context(), `
		SELECT id, name, key_prefix, environment, is_active, last_used_at, created_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		h.logger.Error("failed to list api keys", "error", err, "tenant_id", tenantID)
		return fiber.ErrInternalServerError
	}
	defer rows.Close()

	keys := make([]APIKeyResponse, 0)
	for rows.Next() {
		var key APIKeyResponse
		var id uuid.UUID
		var lastUsedAt *time.Time
		var createdAt time.Time

		err := rows.Scan(&id, &key.Name, &key.KeyPrefix, &key.Environment, &key.IsActive, &lastUsedAt, &createdAt)
		if err != nil {
			h.logger.Error("failed to scan api key", "error", err)
			continue
		}

		key.ID = id.String()
		key.CreatedAt = createdAt.Format(time.RFC3339)
		if lastUsedAt != nil {
			formatted := lastUsedAt.Format(time.RFC3339)
			key.LastUsedAt = &formatted
		}

		keys = append(keys, key)
	}

	pk := ""
	if publicKey != nil {
		pk = *publicKey
	}

	return c.JSON(APIKeysResponse{
		PublicKey: pk,
		Keys:      keys,
	})
}
