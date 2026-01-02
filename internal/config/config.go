package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Server
	Port        int    `envconfig:"PORT" default:"3000"`
	Environment string `envconfig:"ENV" default:"development"`

	// Database
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// Provider
	ProviderType string `envconfig:"PROVIDER_TYPE" default:"deepface"`
	DeepFaceURL  string `envconfig:"DEEPFACE_URL" default:"http://localhost:5000"`

	// Security
	APIKeySecret string `envconfig:"API_KEY_SECRET" required:"true"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
