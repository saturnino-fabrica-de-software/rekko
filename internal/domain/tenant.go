package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tenant representa um cliente B2B do sistema
type Tenant struct {
	ID         uuid.UUID              `json:"id"`
	Name       string                 `json:"name"`
	APIKeyHash string                 `json:"-"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
	IsActive   bool                   `json:"is_active"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// TenantSettings contém configurações específicas do tenant
type TenantSettings struct {
	VerificationThreshold float64 `json:"verification_threshold"`
	MaxFacesPerUser       int     `json:"max_faces_per_user"`
	RequireLiveness       bool    `json:"require_liveness"`
}

// DefaultTenantSettings retorna configurações padrão
func DefaultTenantSettings() TenantSettings {
	return TenantSettings{
		VerificationThreshold: 0.8,
		MaxFacesPerUser:       1,
		RequireLiveness:       false,
	}
}
