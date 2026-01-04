package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSign(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		payload  []byte
		expected string
	}{
		{
			name:     "simple payload",
			secret:   "my-secret-key",
			payload:  []byte(`{"type":"test","data":"hello"}`),
			expected: "sha256=9e5c1e3e5f7d3c5e5f7d3c5e5f7d3c5e5f7d3c5e5f7d3c5e5f7d3c5e5f7d3c5e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := Sign(tt.secret, tt.payload)
			assert.NotEmpty(t, signature)
			assert.Contains(t, signature, "sha256=")

			isValid := Verify(tt.secret, tt.payload, signature)
			assert.True(t, isValid, "signature should be valid")
		})
	}
}

func TestVerify(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"test":"data"}`)
	validSignature := Sign(secret, payload)

	tests := []struct {
		name      string
		secret    string
		payload   []byte
		signature string
		expected  bool
	}{
		{
			name:      "valid signature",
			secret:    secret,
			payload:   payload,
			signature: validSignature,
			expected:  true,
		},
		{
			name:      "invalid signature",
			secret:    secret,
			payload:   payload,
			signature: "sha256=invalid",
			expected:  false,
		},
		{
			name:      "wrong secret",
			secret:    "wrong-secret",
			payload:   payload,
			signature: validSignature,
			expected:  false,
		},
		{
			name:      "modified payload",
			secret:    secret,
			payload:   []byte(`{"test":"modified"}`),
			signature: validSignature,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Verify(tt.secret, tt.payload, tt.signature)
			assert.Equal(t, tt.expected, result)
		})
	}
}
