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
}

// DefaultTenantSettings retorna configurações padrão
func DefaultTenantSettings() TenantSettings {
	return TenantSettings{
		VerificationThreshold: 0.8,
		MaxFacesPerUser:       1,
		RequireLiveness:       false,
	}
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
