package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

type FaceRepositoryInterface interface {
	Create(ctx context.Context, face *domain.Face) error
	Update(ctx context.Context, face *domain.Face) error
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error)
	Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error
	SearchByEmbedding(ctx context.Context, tenantID uuid.UUID, embedding []float64, threshold float64, limit int) ([]domain.SearchMatch, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Face, error)
}

type VerificationRepositoryInterface interface {
	Create(ctx context.Context, v *domain.Verification) error
}

type SearchAuditRepositoryInterface interface {
	Create(ctx context.Context, audit *domain.SearchAudit) error
}

type RateLimiterInterface interface {
	CheckSearchLimit(ctx context.Context, tenantID uuid.UUID, limit int) error
}

type FaceService struct {
	faceRepo         FaceRepositoryInterface
	verificationRepo VerificationRepositoryInterface
	searchAuditRepo  SearchAuditRepositoryInterface
	provider         provider.FaceProvider
	rateLimiter      RateLimiterInterface
	threshold        float64
}

func NewFaceService(
	faceRepo FaceRepositoryInterface,
	verificationRepo VerificationRepositoryInterface,
	searchAuditRepo SearchAuditRepositoryInterface,
	faceProvider provider.FaceProvider,
	rateLimiter RateLimiterInterface,
) *FaceService {
	return &FaceService{
		faceRepo:         faceRepo,
		verificationRepo: verificationRepo,
		searchAuditRepo:  searchAuditRepo,
		provider:         faceProvider,
		rateLimiter:      rateLimiter,
		threshold:        0.8,
	}
}

func (s *FaceService) WithThreshold(threshold float64) *FaceService {
	s.threshold = threshold
	return s
}

func (s *FaceService) Register(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte, requireLiveness bool, livenessThreshold float64) (*domain.Face, error) {
	// Use AnalyzeFace for a single HTTP call (3 calls -> 1 call optimization)
	analysis, err := s.provider.AnalyzeFace(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: analyze face: %w", tenantID, err)
	}

	// Validate face count
	if analysis.FaceCount == 0 {
		return nil, domain.ErrNoFaceDetected
	}
	if analysis.FaceCount > 1 {
		return nil, domain.ErrMultipleFaces
	}

	// Validate liveness if required
	if requireLiveness && analysis.LivenessScore < livenessThreshold {
		return nil, domain.ErrLivenessFailed
	}

	// Check if external_id already has a face registered
	// If yes: update (allows re-registration with better photo)
	// If no: create new
	existingFace, err := s.faceRepo.GetByExternalID(ctx, tenantID, externalID)
	if err == nil && existingFace != nil {
		// Update existing face with new embedding/quality
		existingFace.Embedding = analysis.Embedding
		existingFace.QualityScore = analysis.QualityScore
		if err := s.faceRepo.Update(ctx, existingFace); err != nil {
			return nil, fmt.Errorf("tenant %s: update face: %w", tenantID, err)
		}
		// Get the updated face to return complete data
		return s.faceRepo.GetByExternalID(ctx, tenantID, externalID)
	}

	// Create new face
	face := &domain.Face{
		TenantID:     tenantID,
		ExternalID:   externalID,
		Embedding:    analysis.Embedding,
		QualityScore: analysis.QualityScore,
	}

	if err := s.faceRepo.Create(ctx, face); err != nil {
		return nil, err
	}

	return face, nil
}

func (s *FaceService) Verify(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte) (*domain.Verification, error) {
	start := time.Now()

	storedFace, err := s.faceRepo.GetByExternalID(ctx, tenantID, externalID)
	if err != nil {
		return nil, err
	}

	detectedFaces, err := s.provider.DetectFaces(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: detect faces: %w", tenantID, err)
	}

	if len(detectedFaces) == 0 {
		return nil, domain.ErrNoFaceDetected
	}

	if len(detectedFaces) > 1 {
		return nil, domain.ErrMultipleFaces
	}

	_, newEmbedding, err := s.provider.IndexFace(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: index face for verification: %w", tenantID, err)
	}

	similarity, err := s.provider.CompareFaces(ctx, storedFace.Embedding, newEmbedding)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: compare faces: %w", tenantID, err)
	}

	verified := similarity >= s.threshold
	latencyMs := time.Since(start).Milliseconds()

	verification := &domain.Verification{
		TenantID:   tenantID,
		FaceID:     &storedFace.ID,
		ExternalID: externalID,
		Verified:   verified,
		Confidence: similarity,
		LatencyMs:  latencyMs,
	}

	// Audit log - error is intentionally not returned
	// The verification result was already determined successfully
	// In production, this would be logged with proper observability
	_ = s.verificationRepo.Create(ctx, verification)

	return verification, nil
}

func (s *FaceService) Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error {
	// Verify face exists and belongs to tenant before deleting
	if _, err := s.faceRepo.GetByExternalID(ctx, tenantID, externalID); err != nil {
		return err
	}

	// Delete from database
	if err := s.faceRepo.Delete(ctx, tenantID, externalID); err != nil {
		return fmt.Errorf("tenant %s: delete face: %w", tenantID, err)
	}

	return nil
}

// GetByExternalID retrieves a face by external ID for a tenant
func (s *FaceService) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error) {
	face, err := s.faceRepo.GetByExternalID(ctx, tenantID, externalID)
	if err != nil {
		return nil, err
	}
	return face, nil
}

func (s *FaceService) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Face, error) {
	return s.faceRepo.List(ctx, tenantID, limit, offset)
}

func (s *FaceService) CheckLiveness(ctx context.Context, imageBytes []byte, threshold float64) (*domain.LivenessResult, error) {
	// Call provider to check liveness
	providerResult, err := s.provider.CheckLiveness(ctx, imageBytes, threshold)
	if err != nil {
		return nil, fmt.Errorf("check liveness: %w", err)
	}

	// Convert provider result to domain result
	result := &domain.LivenessResult{
		IsLive:     providerResult.IsLive,
		Confidence: providerResult.Confidence,
		Reasons:    providerResult.Reasons,
		Checks: domain.LivenessChecks{
			EyesOpen:     providerResult.Checks.EyesOpen,
			FacingCamera: providerResult.Checks.FacingCamera,
			QualityOK:    providerResult.Checks.QualityOK,
			SingleFace:   providerResult.Checks.SingleFace,
		},
	}

	return result, nil
}

// Search performs a 1:N face search against all faces in the tenant
// Returns matches above threshold, ordered by similarity
func (s *FaceService) Search(ctx context.Context, tenant *domain.Tenant, imageBytes []byte, threshold float64, maxResults int, clientIP string) (*domain.SearchResult, error) {
	start := time.Now()

	// 1. Extract settings from tenant
	settings := tenant.GetSettings()

	// 2. Verify if search is enabled
	if !settings.SearchEnabled {
		return nil, domain.ErrSearchNotEnabled
	}

	// 3. Apply defaults if not provided
	if threshold <= 0 {
		threshold = settings.SearchThreshold
	}
	if maxResults <= 0 {
		maxResults = settings.SearchMaxResults
	}

	// 4. Validate parameters
	if threshold < 0 || threshold > 1 {
		return nil, domain.ErrInvalidThreshold
	}
	if maxResults < 1 || maxResults > 50 {
		return nil, domain.ErrInvalidMaxResults
	}

	// 5. Check rate limit
	if err := s.rateLimiter.CheckSearchLimit(ctx, tenant.ID, settings.SearchRateLimit); err != nil {
		return nil, domain.ErrSearchRateLimitExceeded
	}

	// 6. Analyze face with single HTTP call (optimized from IndexFace + CheckLiveness)
	analysis, err := s.provider.AnalyzeFace(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: analyze face: %w", tenant.ID, err)
	}

	// Validate face count
	if analysis.FaceCount == 0 {
		return nil, domain.ErrNoFaceDetected
	}

	// 7. Apply liveness check based on SecurityLevel
	switch settings.SecurityLevel {
	case domain.SecurityEnhanced:
		// Basic passive liveness - minimum 0.5 confidence
		if analysis.LivenessScore < 0.5 {
			return nil, domain.ErrLowLivenessConfidence
		}
	case domain.SecurityMaximum:
		// Full liveness check with tenant's configured threshold
		if analysis.LivenessScore < settings.LivenessThreshold {
			return nil, domain.ErrLivenessFailed
		}
		// SecurityStandard: no liveness check (fastest path)
	}

	// 8. Search similar faces in database using embedding from analysis
	matches, err := s.faceRepo.SearchByEmbedding(ctx, tenant.ID, analysis.Embedding, threshold, maxResults)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: search faces: %w", tenant.ID, err)
	}

	// 9. Calculate latency
	latencyMs := time.Since(start).Milliseconds()
	searchID := uuid.New()

	// 10. Create audit log (async, best-effort with panic recovery)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in search audit", "panic", r, "tenant_id", tenant.ID, "search_id", searchID)
			}
		}()
		s.createSearchAudit(tenant.ID, searchID, matches, threshold, maxResults, latencyMs, clientIP)
	}()

	// 11. Return result (TotalFaces removed from hot path - can be added back async if needed)
	return &domain.SearchResult{
		Matches:    matches,
		TotalFaces: 0, // Removed CountByTenant from hot path for performance
		LatencyMs:  latencyMs,
		SearchID:   searchID,
	}, nil
}

// createSearchAudit creates an audit log entry asynchronously
func (s *FaceService) createSearchAudit(tenantID, searchID uuid.UUID, matches []domain.SearchMatch, threshold float64, maxResults int, latencyMs int64, clientIP string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	audit := &domain.SearchAudit{
		ID:           searchID,
		TenantID:     tenantID,
		ResultsCount: len(matches),
		Threshold:    threshold,
		MaxResults:   maxResults,
		LatencyMs:    latencyMs,
		ClientIP:     clientIP,
	}

	// Add top match if exists
	if len(matches) > 0 {
		audit.TopMatchExternalID = &matches[0].ExternalID
		audit.TopMatchSimilarity = &matches[0].Similarity
	}

	// Best-effort audit log creation (errors are intentionally ignored)
	_ = s.searchAuditRepo.Create(ctx, audit)
}
