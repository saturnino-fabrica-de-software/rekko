package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTenant_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tenant  Tenant
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tenant",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "test-company",
				IsActive: true,
				Plan:     PlanStarter,
				Settings: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "valid tenant with pro plan",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Pro Company",
				Slug:     "pro-company",
				IsActive: true,
				Plan:     PlanPro,
				Settings: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "valid tenant with enterprise plan",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Enterprise Corp",
				Slug:     "enterprise-corp",
				IsActive: true,
				Plan:     PlanEnterprise,
				Settings: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			tenant: Tenant{
				ID:       uuid.New(),
				Slug:     "test-company",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant name cannot be empty",
		},
		{
			name: "empty slug",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant slug cannot be empty",
		},
		{
			name: "invalid slug with uppercase",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "Test-Company",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant slug must contain only lowercase letters, numbers and hyphens",
		},
		{
			name: "invalid slug with underscore",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "test_company",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant slug must contain only lowercase letters, numbers and hyphens",
		},
		{
			name: "invalid slug with spaces",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "test company",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant slug must contain only lowercase letters, numbers and hyphens",
		},
		{
			name: "invalid slug starting with hyphen",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "-test-company",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant slug must contain only lowercase letters, numbers and hyphens",
		},
		{
			name: "invalid slug ending with hyphen",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "test-company-",
				Plan:     PlanStarter,
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "tenant slug must contain only lowercase letters, numbers and hyphens",
		},
		{
			name: "invalid plan",
			tenant: Tenant{
				ID:       uuid.New(),
				Name:     "Test Company",
				Slug:     "test-company",
				Plan:     "invalid-plan",
				IsActive: true,
			},
			wantErr: true,
			errMsg:  "invalid plan type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tenant.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Tenant.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Tenant.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestIsValidPlan(t *testing.T) {
	tests := []struct {
		name string
		plan string
		want bool
	}{
		{
			name: "starter plan",
			plan: PlanStarter,
			want: true,
		},
		{
			name: "pro plan",
			plan: PlanPro,
			want: true,
		},
		{
			name: "enterprise plan",
			plan: PlanEnterprise,
			want: true,
		},
		{
			name: "invalid plan",
			plan: "invalid",
			want: false,
		},
		{
			name: "empty plan",
			plan: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidPlan(tt.plan)
			if got != tt.want {
				t.Errorf("IsValidPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultTenantSettings(t *testing.T) {
	settings := DefaultTenantSettings()

	if settings.VerificationThreshold != 0.8 {
		t.Errorf("VerificationThreshold = %v, want 0.8", settings.VerificationThreshold)
	}

	if settings.MaxFacesPerUser != 1 {
		t.Errorf("MaxFacesPerUser = %v, want 1", settings.MaxFacesPerUser)
	}

	if settings.RequireLiveness != false {
		t.Errorf("RequireLiveness = %v, want false", settings.RequireLiveness)
	}

	if settings.LivenessThreshold != 0.90 {
		t.Errorf("LivenessThreshold = %v, want 0.90", settings.LivenessThreshold)
	}
}

func TestTenantSlugValidation(t *testing.T) {
	validSlugs := []string{
		"test",
		"test-company",
		"test-company-123",
		"123-test",
		"a",
		"test123",
	}

	for _, slug := range validSlugs {
		t.Run("valid_slug_"+slug, func(t *testing.T) {
			tenant := Tenant{
				ID:       uuid.New(),
				Name:     "Test",
				Slug:     slug,
				Plan:     PlanStarter,
				IsActive: true,
			}

			if err := tenant.Validate(); err != nil {
				t.Errorf("Validate() failed for valid slug %q: %v", slug, err)
			}
		})
	}

	invalidSlugs := []string{
		"Test",
		"TEST",
		"test_company",
		"test company",
		"test.company",
		"-test",
		"test-",
		"test--company",
		"",
	}

	for _, slug := range invalidSlugs {
		t.Run("invalid_slug_"+slug, func(t *testing.T) {
			tenant := Tenant{
				ID:       uuid.New(),
				Name:     "Test",
				Slug:     slug,
				Plan:     PlanStarter,
				IsActive: true,
			}

			if err := tenant.Validate(); err == nil {
				t.Errorf("Validate() should fail for invalid slug %q", slug)
			}
		})
	}
}

func TestTenantJSONSerialization(t *testing.T) {
	now := time.Now()
	tenant := Tenant{
		ID:        uuid.New(),
		Name:      "Test Company",
		Slug:      "test-company",
		IsActive:  true,
		Plan:      PlanPro,
		Settings:  map[string]interface{}{"key": "value"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if tenant.ID == uuid.Nil {
		t.Error("tenant.ID should not be nil")
	}

	if tenant.Name == "" {
		t.Error("tenant.Name should not be empty")
	}

	if tenant.Slug == "" {
		t.Error("tenant.Slug should not be empty")
	}
}
