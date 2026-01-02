package config

import (
	"os"
	"testing"
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
				"ENV":            "production",
				"DATABASE_URL":   "postgres://localhost/test",
				"API_KEY_SECRET": "secret123",
			},
			wantErr: false,
			check: func(c *Config) bool {
				return c.Port == 8080 &&
					c.Environment == "production" &&
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
					c.Environment == "development" &&
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
				os.Setenv(k, v)
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
		{"development", "development", true},
		{"production", "production", false},
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
		{"production", "production", true},
		{"development", "development", false},
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
