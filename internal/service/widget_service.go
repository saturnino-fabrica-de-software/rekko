package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

const (
	widgetSessionDuration = 10 * time.Minute
)

type WidgetSessionRepositoryInterface interface {
	Create(ctx context.Context, session *domain.WidgetSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WidgetSession, error)
	DeleteExpired(ctx context.Context) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type TenantRepositoryInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetByPublicKey(ctx context.Context, publicKey string) (*domain.Tenant, error)
	GetAllowedDomains(ctx context.Context, tenantID uuid.UUID) ([]string, error)
}

type WidgetService struct {
	sessionRepo WidgetSessionRepositoryInterface
	tenantRepo  TenantRepositoryInterface
	faceService *FaceService
}

func NewWidgetService(
	sessionRepo WidgetSessionRepositoryInterface,
	tenantRepo TenantRepositoryInterface,
	faceService *FaceService,
) *WidgetService {
	return &WidgetService{
		sessionRepo: sessionRepo,
		tenantRepo:  tenantRepo,
		faceService: faceService,
	}
}

// CreateSession creates a new widget session after validating public key and origin
func (s *WidgetService) CreateSession(ctx context.Context, publicKey, origin string) (*domain.WidgetSession, error) {
	// 1. Validate input
	if publicKey == "" {
		return nil, domain.ErrValidationFailed.WithError(fmt.Errorf("public_key is required"))
	}

	if origin == "" {
		return nil, domain.ErrValidationFailed.WithError(fmt.Errorf("origin is required"))
	}

	// 2. Parse and validate origin URL
	parsedOrigin, err := parseOrigin(origin)
	if err != nil {
		return nil, domain.ErrInvalidOrigin.WithError(err)
	}

	// 3. Get tenant by public key
	tenant, err := s.tenantRepo.GetByPublicKey(ctx, publicKey)
	if err != nil {
		return nil, domain.ErrInvalidPublicKey.WithError(err)
	}

	// 4. Get allowed domains for tenant
	allowedDomains, err := s.tenantRepo.GetAllowedDomains(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: get allowed domains: %w", tenant.ID, err)
	}

	// 5. Validate origin is in allowed domains
	if !isOriginAllowed(parsedOrigin, allowedDomains) {
		return nil, domain.ErrOriginNotAllowed.WithError(
			fmt.Errorf("origin %s not in allowed domains", parsedOrigin),
		)
	}

	// 6. Create session
	session := &domain.WidgetSession{
		TenantID:  tenant.ID,
		Origin:    parsedOrigin,
		ExpiresAt: time.Now().Add(widgetSessionDuration),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("tenant %s: create widget session: %w", tenant.ID, err)
	}

	return session, nil
}

// ValidateSession validates a session ID and returns the session if valid
func (s *WidgetService) ValidateSession(ctx context.Context, sessionID uuid.UUID) (*domain.WidgetSession, error) {
	// 1. Get session
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 2. Check if expired
	if session.IsExpired() {
		return nil, domain.ErrWidgetSessionExpired
	}

	return session, nil
}

// Register registers a new face using a widget session
func (s *WidgetService) Register(ctx context.Context, sessionID uuid.UUID, externalID string, imageBytes []byte) (*domain.Face, error) {
	// 1. Validate session
	session, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 2. Get tenant settings
	tenant, err := s.tenantRepo.GetByID(ctx, session.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: get tenant: %w", session.TenantID, err)
	}
	settings := tenant.GetSettings()

	// 3. Call face service to register using tenant settings
	face, err := s.faceService.Register(ctx, session.TenantID, externalID, imageBytes, settings.RequireLiveness, settings.LivenessThreshold)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: widget register: %w", session.TenantID, err)
	}

	return face, nil
}

// ValidateLiveness validates liveness for a widget session
// This is used by the widget to validate active liveness challenges before registration
//
// Threshold is now tenant-configurable via settings.liveness_threshold
// Default: 0.90 (see domain.DefaultTenantSettings)
// For DeepFace (dev), recommend setting to 0.40 as it estimates confidence from face area
// For Rekognition (prod), typical values are 0.80-0.95
func (s *WidgetService) ValidateLiveness(ctx context.Context, sessionID uuid.UUID, imageBytes []byte) (*domain.LivenessResult, error) {
	// 1. Validate session
	session, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 2. Get tenant settings for liveness threshold
	tenant, err := s.tenantRepo.GetByID(ctx, session.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: get tenant: %w", session.TenantID, err)
	}
	settings := tenant.GetSettings()

	// 3. Call face service to check liveness with tenant's threshold
	result, err := s.faceService.CheckLiveness(ctx, imageBytes, settings.LivenessThreshold)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: widget validate liveness: %w", session.TenantID, err)
	}

	return result, nil
}

// Search performs a 1:N face search using a widget session
// This enables "walk-in" identification at event entrances without requiring an external ID
// Returns the best match if confidence >= threshold
func (s *WidgetService) Search(ctx context.Context, sessionID uuid.UUID, imageBytes []byte, clientIP string) (*domain.SearchResult, error) {
	// 1. Validate session
	session, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 2. Get tenant for settings
	tenant, err := s.tenantRepo.GetByID(ctx, session.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: get tenant: %w", session.TenantID, err)
	}

	// 3. Get tenant settings for search configuration
	settings := tenant.GetSettings()

	// 4. Perform search using face service
	// Widget search returns top 1 match only for fast identification
	result, err := s.faceService.Search(ctx, tenant, imageBytes, settings.SearchThreshold, 1, clientIP)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: widget search: %w", session.TenantID, err)
	}

	return result, nil
}

// CleanupExpiredSessions removes all expired sessions
// This should be called periodically (e.g., via cron job)
func (s *WidgetService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	count, err := s.sessionRepo.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}
	return count, nil
}

// parseOrigin parses and normalizes an origin URL
// Returns the origin in the format "https://example.com"
func parseOrigin(origin string) (string, error) {
	// Parse URL
	u, err := url.Parse(origin)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("scheme must be http or https")
	}

	// Validate host
	if u.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	// Normalize: scheme://host (no path, query, or fragment)
	normalized := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	return normalized, nil
}

// isOriginAllowed checks if an origin is in the list of allowed domains
func isOriginAllowed(origin string, allowedDomains []string) bool {
	// If no domains configured, deny all (secure by default)
	if len(allowedDomains) == 0 {
		return false
	}

	// Parse origin to get host
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originHost := u.Host

	// Check each allowed domain
	for _, domain := range allowedDomains {
		// Exact match
		if originHost == domain {
			return true
		}

		// Wildcard subdomain match (e.g., *.example.com)
		if strings.HasPrefix(domain, "*.") {
			baseDomain := strings.TrimPrefix(domain, "*.")
			if strings.HasSuffix(originHost, baseDomain) {
				return true
			}
		}
	}

	return false
}
