package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Plan types
const (
	PlanStarter    = "starter"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

var (
	validPlans = map[string]bool{
		PlanStarter:    true,
		PlanPro:        true,
		PlanEnterprise: true,
	}

	slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// Tenant representa um cliente B2B do sistema
type Tenant struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	IsActive  bool                   `json:"is_active"`
	Plan      string                 `json:"plan"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TenantSettings contém configurações específicas do tenant
type TenantSettings struct {
	VerificationThreshold float64 `json:"verification_threshold"`
	MaxFacesPerUser       int     `json:"max_faces_per_user"`
	RequireLiveness       bool    `json:"require_liveness"`
	LivenessThreshold     float64 `json:"liveness_threshold"`
	SearchEnabled         bool    `json:"search_enabled"`
	SearchRequireLiveness bool    `json:"search_require_liveness"`
	SearchThreshold       float64 `json:"search_threshold"`
	SearchMaxResults      int     `json:"search_max_results"`
	SearchRateLimit       int     `json:"search_rate_limit"`
}

// DefaultTenantSettings retorna configurações padrão
func DefaultTenantSettings() TenantSettings {
	return TenantSettings{
		VerificationThreshold: 0.8,
		MaxFacesPerUser:       1,
		RequireLiveness:       false,
		LivenessThreshold:     0.90,
		SearchEnabled:         false,
		SearchRequireLiveness: false,
		SearchThreshold:       0.85,
		SearchMaxResults:      10,
		SearchRateLimit:       30,
	}
}

// GetSettings returns typed tenant settings with defaults for missing values
func (t *Tenant) GetSettings() TenantSettings {
	defaults := DefaultTenantSettings()

	if t.Settings == nil {
		return defaults
	}

	// Parse each setting with type assertion and fallback to default
	if v, ok := t.Settings["verification_threshold"].(float64); ok {
		defaults.VerificationThreshold = v
	}
	if v, ok := t.Settings["max_faces_per_user"].(float64); ok {
		defaults.MaxFacesPerUser = int(v)
	}
	if v, ok := t.Settings["require_liveness"].(bool); ok {
		defaults.RequireLiveness = v
	}
	if v, ok := t.Settings["liveness_threshold"].(float64); ok {
		defaults.LivenessThreshold = v
	}
	if v, ok := t.Settings["search_enabled"].(bool); ok {
		defaults.SearchEnabled = v
	}
	if v, ok := t.Settings["search_require_liveness"].(bool); ok {
		defaults.SearchRequireLiveness = v
	}
	if v, ok := t.Settings["search_threshold"].(float64); ok {
		defaults.SearchThreshold = v
	}
	if v, ok := t.Settings["search_max_results"].(float64); ok {
		defaults.SearchMaxResults = int(v)
	}
	if v, ok := t.Settings["search_rate_limit"].(float64); ok {
		defaults.SearchRateLimit = int(v)
	}

	return defaults
}

// Validate verifica se o tenant é válido
func (t *Tenant) Validate() error {
	if t.Name == "" {
		return errors.New("tenant name cannot be empty")
	}

	if t.Slug == "" {
		return errors.New("tenant slug cannot be empty")
	}

	if !slugRegex.MatchString(t.Slug) {
		return errors.New("tenant slug must contain only lowercase letters, numbers and hyphens")
	}

	if !validPlans[t.Plan] {
		return errors.New("invalid plan type")
	}

	return nil
}

// IsValidPlan verifica se o plano é válido
func IsValidPlan(plan string) bool {
	return validPlans[plan]
}
