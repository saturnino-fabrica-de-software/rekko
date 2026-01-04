package admin

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEmail = "admin@rekko.com"
	testRole  = "super_admin"
)

func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "rekko-test", 1*time.Hour)
	userID := uuid.New()
	email := testEmail
	role := testRole

	token, err := service.GenerateToken(userID, email, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "rekko-test", 1*time.Hour)
	userID := uuid.New()
	email := testEmail
	role := testRole

	token, err := service.GenerateToken(userID, email, role)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, "rekko-test", claims.Issuer)
}

func TestJWTService_ValidateToken_InvalidToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "rekko-test", 1*time.Hour)

	tests := []struct {
		name        string
		token       string
		expectedErr error
	}{
		{
			name:        "invalid token format",
			token:       "invalid.token.format",
			expectedErr: ErrInvalidToken,
		},
		{
			name:        "empty token",
			token:       "",
			expectedErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateToken(tt.token)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestJWTService_ValidateToken_ExpiredToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "rekko-test", -1*time.Hour)
	userID := uuid.New()
	email := testEmail
	role := testRole

	token, err := service.GenerateToken(userID, email, role)
	require.NoError(t, err)

	_, err = service.ValidateToken(token)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestJWTService_ValidateToken_DifferentSecret(t *testing.T) {
	service1 := NewJWTService("secret-1", "rekko-test", 1*time.Hour)
	service2 := NewJWTService("secret-2", "rekko-test", 1*time.Hour)

	userID := uuid.New()
	email := testEmail
	role := testRole

	token, err := service1.GenerateToken(userID, email, role)
	require.NoError(t, err)

	_, err = service2.ValidateToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTService_RefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "rekko-test", 1*time.Hour)
	userID := uuid.New()
	email := testEmail
	role := testRole

	oldToken, err := service.GenerateToken(userID, email, role)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	newToken, err := service.RefreshToken(oldToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, oldToken, newToken)

	claims, err := service.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestJWTService_RefreshToken_InvalidToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "rekko-test", 1*time.Hour)

	_, err := service.RefreshToken("invalid.token")
	assert.ErrorIs(t, err, ErrInvalidToken)
}
