package admin

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken is returned when token validation fails
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when token is expired
	ErrExpiredToken = errors.New("token expired")
	// ErrInvalidClaims is returned when claims are invalid
	ErrInvalidClaims = errors.New("invalid claims")
)

// AdminClaims represents JWT claims for super admin
type AdminClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// JWTService handles JWT operations for super admin authentication
type JWTService struct {
	secretKey []byte
	issuer    string
	expiresIn time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey, issuer string, expiresIn time.Duration) *JWTService {
	return &JWTService{
		secretKey: []byte(secretKey),
		issuer:    issuer,
		expiresIn: expiresIn,
	}
}

// GenerateToken generates a new JWT token for admin user
func (s *JWTService) GenerateToken(userID uuid.UUID, email, role string) (string, error) {
	now := time.Now()
	claims := AdminClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiresIn)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken validates and parses a JWT token
func (s *JWTService) ValidateToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshToken generates a new token with extended expiration
func (s *JWTService) RefreshToken(oldToken string) (string, error) {
	claims, err := s.ValidateToken(oldToken)
	if err != nil {
		return "", err
	}

	return s.GenerateToken(claims.UserID, claims.Email, claims.Role)
}
