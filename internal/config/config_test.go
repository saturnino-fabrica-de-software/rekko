package config

import (
	"os"
	"testing"
)

const (
	envProduction  = "production"
	envDevelopment = "development"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(*Config) bool
	}{
		{
			name: "loads with all required vars",
			envVars: map[string]string{
				"PORT":           "8080",
				"ENV":            envProduction,
				"DATABASE_URL":   "postgres://localhost/test",
				"API_KEY_SECRET": "secret123",
			},
			wantErr: false,
			check: func(c *Config) bool {
				return c.Port == 8080 &&
					c.Environment == envProduction &&
					c.DatabaseURL == "postgres://localhost/test" &&
					c.APIKeySecret == "secret123"
			},
		},
		{
			name: "uses defaults when optional vars missing",
			envVars: map[string]string{
				"DATABASE_URL":   "postgres://localhost/test",
				"API_KEY_SECRET": "secret123",
			},
			wantErr: false,
			check: func(c *Config) bool {
				return c.Port == 3000 &&
					c.Environment == envDevelopment &&
					c.ProviderType == "deepface"
			},
		},
		{
			name: "fails when DATABASE_URL missing",
			envVars: map[string]string{
				"API_KEY_SECRET": "secret123",
			},
			wantErr: true,
			check:   nil,
		},
		{
			name: "fails when API_KEY_SECRET missing",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://localhost/test",
			},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("failed to set env var %s: %v", k, err)
				}
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if tt.check != nil && !tt.check(cfg) {
				t.Errorf("Load() config check failed, got: %+v", cfg)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{envDevelopment, envDevelopment, true},
		{envProduction, envProduction, false},
		{"staging", "staging", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Environment: tt.env}
			if got := c.IsDevelopment(); got != tt.want {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{envProduction, envProduction, true},
		{envDevelopment, envDevelopment, false},
		{"staging", "staging", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Environment: tt.env}
			if got := c.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}
