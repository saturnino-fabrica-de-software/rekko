package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{
			name:    "generate test key",
			env:     EnvTest,
			wantErr: false,
		},
		{
			name:    "generate live key",
			env:     EnvLive,
			wantErr: false,
		},
		{
			name:    "invalid environment",
			env:     "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainKey, hash, prefix, err := GenerateAPIKey(tt.env)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateAPIKey() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateAPIKey() unexpected error: %v", err)
				return
			}

			expectedPrefix := "rekko_" + tt.env + "_"
			if !strings.HasPrefix(plainKey, expectedPrefix) {
				t.Errorf("plainKey prefix = %s, want prefix %s", plainKey[:len(expectedPrefix)], expectedPrefix)
			}

			if len(plainKey) != len(expectedPrefix)+apiKeyLength {
				t.Errorf("plainKey length = %d, want %d", len(plainKey), len(expectedPrefix)+apiKeyLength)
			}

			if hash == "" {
				t.Errorf("hash is empty")
			}

			if prefix != plainKey[:16] {
				t.Errorf("prefix = %s, want %s", prefix, plainKey[:16])
			}

			if !IsValidFormat(plainKey) {
				t.Errorf("generated key has invalid format: %s", plainKey)
			}
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "rekko_test_ABC123XYZ789"

	hash1 := HashAPIKey(key)
	hash2 := HashAPIKey(key)

	if hash1 != hash2 {
		t.Errorf("hash not deterministic: hash1=%s, hash2=%s", hash1, hash2)
	}

	if len(hash1) != 64 {
		t.Errorf("hash length = %d, want 64 (SHA256 hex)", len(hash1))
	}
}

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "valid test key",
			key:  "rekko_test_" + strings.Repeat("A", apiKeyLength),
			want: true,
		},
		{
			name: "valid live key",
			key:  "rekko_live_" + strings.Repeat("B", apiKeyLength),
			want: true,
		},
		{
			name: "invalid prefix",
			key:  "invalid_test_" + strings.Repeat("A", apiKeyLength),
			want: false,
		},
		{
			name: "invalid environment",
			key:  "rekko_prod_" + strings.Repeat("A", apiKeyLength),
			want: false,
		},
		{
			name: "too short",
			key:  "rekko_test_ABC",
			want: false,
		},
		{
			name: "too long",
			key:  "rekko_test_" + strings.Repeat("A", apiKeyLength+10),
			want: false,
		},
		{
			name: "invalid characters",
			key:  "rekko_test_" + strings.Repeat("!", apiKeyLength),
			want: false,
		},
		{
			name: "missing parts",
			key:  "rekko_test",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidFormat(tt.key)
			if got != tt.want {
				t.Errorf("IsValidFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKey_Validate(t *testing.T) {
	validTenantID := uuid.New()

	tests := []struct {
		name    string
		apiKey  APIKey
		wantErr bool
	}{
		{
			name: "valid api key",
			apiKey: APIKey{
				TenantID:    validTenantID,
				Name:        "Test Key",
				KeyHash:     "hash123",
				KeyPrefix:   "rekko_test_ABCD",
				Environment: EnvTest,
			},
			wantErr: false,
		},
		{
			name: "missing tenant_id",
			apiKey: APIKey{
				Name:        "Test Key",
				KeyHash:     "hash123",
				KeyPrefix:   "rekko_test_ABCD",
				Environment: EnvTest,
			},
			wantErr: true,
		},
		{
			name: "missing name",
			apiKey: APIKey{
				TenantID:    validTenantID,
				KeyHash:     "hash123",
				KeyPrefix:   "rekko_test_ABCD",
				Environment: EnvTest,
			},
			wantErr: true,
		},
		{
			name: "missing key_hash",
			apiKey: APIKey{
				TenantID:    validTenantID,
				Name:        "Test Key",
				KeyPrefix:   "rekko_test_ABCD",
				Environment: EnvTest,
			},
			wantErr: true,
		},
		{
			name: "missing key_prefix",
			apiKey: APIKey{
				TenantID:    validTenantID,
				Name:        "Test Key",
				KeyHash:     "hash123",
				Environment: EnvTest,
			},
			wantErr: true,
		},
		{
			name: "invalid environment",
			apiKey: APIKey{
				TenantID:    validTenantID,
				Name:        "Test Key",
				KeyHash:     "hash123",
				KeyPrefix:   "rekko_test_ABCD",
				Environment: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.apiKey.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("APIKey.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		plainKey, _, _, err := GenerateAPIKey(EnvTest)
		if err != nil {
			t.Fatalf("GenerateAPIKey() failed: %v", err)
		}

		if keys[plainKey] {
			t.Errorf("duplicate key generated: %s", plainKey)
		}
		keys[plainKey] = true
	}
}
