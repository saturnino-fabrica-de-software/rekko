package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Environment constants
const (
	EnvTest = "test"
	EnvLive = "live"
)

const (
	apiKeyLength = 32
	base62Chars  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
	validEnvironments = map[string]bool{
		EnvTest: true,
		EnvLive: true,
	}
)

// APIKey representa uma chave de API para autenticação
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	Environment string     `json:"environment"`
	IsActive    bool       `json:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GenerateAPIKey gera uma nova API key com hash e prefix
// Retorna: (plainKey, hash, prefix)
func GenerateAPIKey(env string) (string, string, string, error) {
	if !validEnvironments[env] {
		return "", "", "", errors.New("invalid environment")
	}

	randomPart, err := generateSecureRandomString(apiKeyLength)
	if err != nil {
		return "", "", "", err
	}

	prefix := "rekko_" + env + "_"
	plainKey := prefix + randomPart

	hash := HashAPIKey(plainKey)

	keyPrefix := plainKey[:16]

	return plainKey, hash, keyPrefix, nil
}

// HashAPIKey gera o hash SHA256 de uma API key
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// IsValidFormat verifica se a API key tem o formato correto
func IsValidFormat(key string) bool {
	if !strings.HasPrefix(key, "rekko_") {
		return false
	}

	parts := strings.SplitN(key, "_", 3)
	if len(parts) != 3 {
		return false
	}

	env := parts[1]
	if !validEnvironments[env] {
		return false
	}

	randomPart := parts[2]
	if len(randomPart) != apiKeyLength {
		return false
	}

	for _, char := range randomPart {
		if !strings.ContainsRune(base62Chars, char) {
			return false
		}
	}

	return true
}

// Validate verifica se a API key é válida
func (a *APIKey) Validate() error {
	if a.TenantID == uuid.Nil {
		return errors.New("tenant_id cannot be empty")
	}

	if a.Name == "" {
		return errors.New("name cannot be empty")
	}

	if a.KeyHash == "" {
		return errors.New("key_hash cannot be empty")
	}

	if a.KeyPrefix == "" {
		return errors.New("key_prefix cannot be empty")
	}

	if !validEnvironments[a.Environment] {
		return errors.New("invalid environment")
	}

	return nil
}

// generateSecureRandomString gera uma string aleatória segura usando crypto/rand
func generateSecureRandomString(length int) (string, error) {
	result := make([]byte, length)
	base62Len := big.NewInt(int64(len(base62Chars)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, base62Len)
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[num.Int64()]
	}

	return string(result), nil
}
