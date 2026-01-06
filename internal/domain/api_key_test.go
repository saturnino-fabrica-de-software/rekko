package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		keyType string
		env     string
		wantErr bool
	}{
		{
			name:    "generate secret test key",
			keyType: KeyTypeSecret,
			env:     EnvTest,
			wantErr: false,
		},
		{
			name:    "generate secret live key",
			keyType: KeyTypeSecret,
			env:     EnvLive,
			wantErr: false,
		},
		{
			name:    "generate public test key",
			keyType: KeyTypePublic,
			env:     EnvTest,
			wantErr: false,
		},
		{
			name:    "generate public live key",
			keyType: KeyTypePublic,
			env:     EnvLive,
			wantErr: false,
		},
		{
			name:    "invalid environment",
			keyType: KeyTypeSecret,
			env:     "invalid",
			wantErr: true,
		},
		{
			name:    "invalid key type",
			keyType: "invalid",
			env:     EnvTest,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plainKey, hash, prefix, err := GenerateAPIKey(tt.keyType, tt.env)

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

			// Expected format: sk_live_<random32>
			expectedPrefix := tt.keyType + "_" + tt.env + "_"
			if !strings.HasPrefix(plainKey, expectedPrefix) {
				t.Errorf("plainKey prefix = %s, want prefix %s", plainKey[:len(expectedPrefix)], expectedPrefix)
			}

			if len(plainKey) != len(expectedPrefix)+apiKeyLength {
				t.Errorf("plainKey length = %d, want %d", len(plainKey), len(expectedPrefix)+apiKeyLength)
			}

			if hash == "" {
				t.Errorf("hash is empty")
			}

			// Prefix should be first 14 chars (sk_live_A1b2C3)
			if prefix != plainKey[:14] {
				t.Errorf("prefix = %s, want %s", prefix, plainKey[:14])
			}

			if !IsValidFormat(plainKey) {
				t.Errorf("generated key has invalid format: %s", plainKey)
			}
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "sk_test_ABC123XYZ789ABC123XYZ789ABC12345"

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
			name: "valid secret test key",
			key:  "sk_test_" + strings.Repeat("A", apiKeyLength),
			want: true,
		},
		{
			name: "valid secret live key",
			key:  "sk_live_" + strings.Repeat("B", apiKeyLength),
			want: true,
		},
		{
			name: "valid public test key",
			key:  "pk_test_" + strings.Repeat("C", apiKeyLength),
			want: true,
		},
		{
			name: "valid public live key",
			key:  "pk_live_" + strings.Repeat("D", apiKeyLength),
			want: true,
		},
		{
			name: "invalid key type",
			key:  "xx_test_" + strings.Repeat("A", apiKeyLength),
			want: false,
		},
		{
			name: "invalid environment",
			key:  "sk_prod_" + strings.Repeat("A", apiKeyLength),
			want: false,
		},
		{
			name: "too short",
			key:  "sk_test_ABC",
			want: false,
		},
		{
			name: "too long",
			key:  "sk_test_" + strings.Repeat("A", apiKeyLength+10),
			want: false,
		},
		{
			name: "invalid characters",
			key:  "sk_test_" + strings.Repeat("!", apiKeyLength),
			want: false,
		},
		{
			name: "missing parts",
			key:  "sk_test",
			want: false,
		},
		{
			name: "old format with rekko prefix - invalid",
			key:  "rekko_sk_test_" + strings.Repeat("A", apiKeyLength),
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
				KeyPrefix:   "sk_test_ABCD12",
				Environment: EnvTest,
			},
			wantErr: false,
		},
		{
			name: "missing tenant_id",
			apiKey: APIKey{
				Name:        "Test Key",
				KeyHash:     "hash123",
				KeyPrefix:   "sk_test_ABCD12",
				Environment: EnvTest,
			},
			wantErr: true,
		},
		{
			name: "missing name",
			apiKey: APIKey{
				TenantID:    validTenantID,
				KeyHash:     "hash123",
				KeyPrefix:   "sk_test_ABCD12",
				Environment: EnvTest,
			},
			wantErr: true,
		},
		{
			name: "missing key_hash",
			apiKey: APIKey{
				TenantID:    validTenantID,
				Name:        "Test Key",
				KeyPrefix:   "sk_test_ABCD12",
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
				KeyPrefix:   "sk_test_ABCD12",
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
		plainKey, _, _, err := GenerateAPIKey(KeyTypeSecret, EnvTest)
		if err != nil {
			t.Fatalf("GenerateAPIKey() failed: %v", err)
		}

		if keys[plainKey] {
			t.Errorf("duplicate key generated: %s", plainKey)
		}
		keys[plainKey] = true
	}
}
