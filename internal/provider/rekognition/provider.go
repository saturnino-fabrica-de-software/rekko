package rekognition

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/audit"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

const (
	// maxImageSize is the maximum image size supported by AWS Rekognition (5MB)
	maxImageSize = 5 * 1024 * 1024
	// minImageSize is the minimum image size for valid processing
	minImageSize = 100
)

// Provider implements the provider.FaceProvider interface using AWS Rekognition
// Each Provider instance is associated with a specific tenant for collection isolation
type Provider struct {
	client      *Client
	tenantID    uuid.UUID
	auditLogger audit.Logger
}

// ProviderOption defines optional configuration for Provider
type ProviderOption func(*Provider)

// WithAuditLogger sets the audit logger for the provider
func WithAuditLogger(logger audit.Logger) ProviderOption {
	return func(p *Provider) {
		p.auditLogger = logger
	}
}

// Ensure Provider implements provider.FaceProvider interface at compile time
var _ provider.FaceProvider = (*Provider)(nil)

// NewProvider creates a new Rekognition provider for a specific tenant
// The provider will use tenant-specific collections for all face operations
func NewProvider(ctx context.Context, cfg Config, tenantID uuid.UUID, opts ...ProviderOption) (*Provider, error) {
	client, err := NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create rekognition client: %w", err)
	}

	// Ensure the tenant's collection exists
	if err := client.EnsureCollection(ctx, tenantID.String()); err != nil {
		return nil, fmt.Errorf("ensure collection for tenant %s: %w", tenantID, err)
	}

	p := &Provider{
		client:   client,
		tenantID: tenantID,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

// logAudit logs an audit event if an audit logger is configured
// Audit failure does not affect the operation (fire-and-forget)
func (p *Provider) logAudit(ctx context.Context, eventType audit.EventType, success bool, err error, metadata map[string]string) {
	if p.auditLogger == nil {
		return
	}

	event := audit.Event{
		TenantID:  p.tenantID,
		EventType: eventType,
		Provider:  "rekognition",
		Success:   success,
		Metadata:  metadata,
	}

	if err != nil {
		event.Error = err.Error()
	}

	_ = p.auditLogger.Log(ctx, event)
}

// validateImage checks if image data is valid for Rekognition processing
func validateImage(image []byte) error {
	if len(image) == 0 {
		return ErrInvalidImage
	}
	if len(image) < minImageSize {
		return fmt.Errorf("%w: image too small (%d bytes, minimum %d)", ErrInvalidImage, len(image), minImageSize)
	}
	if len(image) > maxImageSize {
		return fmt.Errorf("%w: image too large (%d bytes, maximum %d)", ErrInvalidImage, len(image), maxImageSize)
	}
	return nil
}

// DetectFaces detects faces in an image using AWS Rekognition DetectFaces API
// Returns an empty slice if no faces are detected (not an error)
func (p *Provider) DetectFaces(ctx context.Context, image []byte) ([]provider.DetectedFace, error) {
	if err := validateImage(image); err != nil {
		p.logAudit(ctx, audit.EventFaceDetected, false, err, map[string]string{
			"image_size": strconv.Itoa(len(image)),
		})
		return nil, fmt.Errorf("tenant %s: %w", p.tenantID, err)
	}

	input := &rekognition.DetectFacesInput{
		Image: &types.Image{
			Bytes: image,
		},
		Attributes: []types.Attribute{types.AttributeAll},
	}

	output, err := p.client.rekognition.DetectFaces(ctx, input)
	if err != nil {
		p.logAudit(ctx, audit.EventFaceDetected, false, err, map[string]string{
			"image_size": strconv.Itoa(len(image)),
		})
		return nil, fmt.Errorf("tenant %s: detect faces: %w", p.tenantID, err)
	}

	// Convert AWS Rekognition face details to provider.DetectedFace
	faces := make([]provider.DetectedFace, 0, len(output.FaceDetails))
	for _, detail := range output.FaceDetails {
		faces = append(faces, provider.DetectedFace{
			BoundingBox: provider.BoundingBox{
				X:      float64(*detail.BoundingBox.Left),
				Y:      float64(*detail.BoundingBox.Top),
				Width:  float64(*detail.BoundingBox.Width),
				Height: float64(*detail.BoundingBox.Height),
			},
			Confidence:   float64(*detail.Confidence),
			QualityScore: p.calculateQualityScore(detail.Quality),
		})
	}

	p.logAudit(ctx, audit.EventFaceDetected, true, nil, map[string]string{
		"faces_count": strconv.Itoa(len(faces)),
		"image_size":  strconv.Itoa(len(image)),
	})

	return faces, nil
}

// IndexFace indexes a face in the tenant's Rekognition collection
// Returns the Rekognition-generated faceID and nil for embedding (Rekognition does not expose embeddings)
// Only indexes the first face found in the image; returns error if no face or multiple faces
func (p *Provider) IndexFace(ctx context.Context, image []byte) (string, []float64, error) {
	if err := validateImage(image); err != nil {
		p.logAudit(ctx, audit.EventFaceRegistered, false, err, map[string]string{
			"image_size": strconv.Itoa(len(image)),
		})
		return "", nil, fmt.Errorf("tenant %s: %w", p.tenantID, err)
	}

	collectionID := p.client.config.CollectionName(p.tenantID.String())

	input := &rekognition.IndexFacesInput{
		CollectionId: aws.String(collectionID),
		Image: &types.Image{
			Bytes: image,
		},
		MaxFaces:      aws.Int32(1), // Only index the first face
		QualityFilter: types.QualityFilterAuto,
		DetectionAttributes: []types.Attribute{
			types.AttributeDefault, // Minimal attributes for indexing
		},
	}

	output, err := p.client.rekognition.IndexFaces(ctx, input)
	if err != nil {
		p.logAudit(ctx, audit.EventFaceRegistered, false, err, map[string]string{
			"image_size": strconv.Itoa(len(image)),
		})
		return "", nil, fmt.Errorf("tenant %s: index face: %w", p.tenantID, err)
	}

	// Check if face was successfully indexed
	if len(output.FaceRecords) == 0 {
		// Face was not indexed - check why
		if len(output.UnindexedFaces) > 0 {
			indexErr := ParseIndexFacesError(output.UnindexedFaces)
			p.logAudit(ctx, audit.EventFaceRegistered, false, indexErr, map[string]string{
				"image_size": strconv.Itoa(len(image)),
				"reason":     "unindexed_faces",
			})
			return "", nil, indexErr
		}
		noFaceErr := fmt.Errorf("tenant %s: %w", p.tenantID, ErrNoFaceDetected)
		p.logAudit(ctx, audit.EventFaceRegistered, false, noFaceErr, map[string]string{
			"image_size": strconv.Itoa(len(image)),
			"reason":     "no_face_detected",
		})
		return "", nil, noFaceErr
	}

	faceRecord := output.FaceRecords[0]
	faceID := *faceRecord.Face.FaceId

	p.logAudit(ctx, audit.EventFaceRegistered, true, nil, map[string]string{
		"face_id":    faceID,
		"image_size": strconv.Itoa(len(image)),
	})

	// AWS Rekognition does not expose face embeddings
	// Return nil for embedding; the service layer should store face data differently
	return faceID, nil, nil
}

// CompareFaces compares two face images using AWS Rekognition CompareFaces API
// Note: This method receives embeddings as parameters (per interface), but AWS Rekognition
// requires actual images for comparison. The service layer must be adapted to pass images
// instead of embeddings when using Rekognition provider.
//
// For now, this implementation returns an error indicating the operation is not supported
// with embeddings. Use SearchFacesByImage for 1:N comparison operations.
func (p *Provider) CompareFaces(ctx context.Context, embedding1, embedding2 []float64) (float64, error) {
	// AWS Rekognition does not support direct embedding comparison
	// The CompareFaces API requires actual image bytes, not embeddings
	// This is an architectural limitation that needs to be addressed at the service layer
	return 0, fmt.Errorf("tenant %s: compare faces with embeddings not supported by Rekognition (use CompareFaceImages instead)", p.tenantID)
}

// CompareFaceImages compares two face images using AWS Rekognition CompareFaces API
// Returns similarity score between 0.0 (completely different) and 1.0 (identical)
// This is the Rekognition-specific method that should be used instead of CompareFaces
func (p *Provider) CompareFaceImages(ctx context.Context, sourceImage, targetImage []byte, similarityThreshold float64) (float64, error) {
	if err := validateImage(sourceImage); err != nil {
		p.logAudit(ctx, audit.EventFaceCompared, false, err, map[string]string{
			"source_image_size": strconv.Itoa(len(sourceImage)),
			"target_image_size": strconv.Itoa(len(targetImage)),
		})
		return 0, fmt.Errorf("tenant %s: source image: %w", p.tenantID, err)
	}
	if err := validateImage(targetImage); err != nil {
		p.logAudit(ctx, audit.EventFaceCompared, false, err, map[string]string{
			"source_image_size": strconv.Itoa(len(sourceImage)),
			"target_image_size": strconv.Itoa(len(targetImage)),
		})
		return 0, fmt.Errorf("tenant %s: target image: %w", p.tenantID, err)
	}

	input := &rekognition.CompareFacesInput{
		SourceImage: &types.Image{
			Bytes: sourceImage,
		},
		TargetImage: &types.Image{
			Bytes: targetImage,
		},
		SimilarityThreshold: aws.Float32(float32(similarityThreshold * 100)), // Convert 0-1 to 0-100
	}

	output, err := p.client.rekognition.CompareFaces(ctx, input)
	if err != nil {
		// Check if error is due to no face detected
		if parsedErr := ParseNoFaceError(err); parsedErr != nil {
			p.logAudit(ctx, audit.EventFaceCompared, false, parsedErr, map[string]string{
				"source_image_size": strconv.Itoa(len(sourceImage)),
				"target_image_size": strconv.Itoa(len(targetImage)),
			})
			return 0, fmt.Errorf("tenant %s: %w", p.tenantID, parsedErr)
		}
		p.logAudit(ctx, audit.EventFaceCompared, false, err, map[string]string{
			"source_image_size": strconv.Itoa(len(sourceImage)),
			"target_image_size": strconv.Itoa(len(targetImage)),
		})
		return 0, fmt.Errorf("tenant %s: compare faces: %w", p.tenantID, err)
	}

	// If no matches found, return 0 similarity
	if len(output.FaceMatches) == 0 {
		p.logAudit(ctx, audit.EventFaceCompared, true, nil, map[string]string{
			"similarity":        "0",
			"source_image_size": strconv.Itoa(len(sourceImage)),
			"target_image_size": strconv.Itoa(len(targetImage)),
			"matched":           "false",
		})
		return 0, nil
	}

	// Return the similarity of the best match (normalized to 0-1 range)
	bestMatch := output.FaceMatches[0]
	similarity := float64(*bestMatch.Similarity) / 100.0

	p.logAudit(ctx, audit.EventFaceCompared, true, nil, map[string]string{
		"similarity":        fmt.Sprintf("%.4f", similarity),
		"source_image_size": strconv.Itoa(len(sourceImage)),
		"target_image_size": strconv.Itoa(len(targetImage)),
		"matched":           "true",
	})

	return similarity, nil
}

// DeleteFace removes a face from the tenant's Rekognition collection
// Returns ErrFaceNotFound if the face ID does not exist in the collection
func (p *Provider) DeleteFace(ctx context.Context, faceID string) error {
	collectionID := p.client.config.CollectionName(p.tenantID.String())

	input := &rekognition.DeleteFacesInput{
		CollectionId: aws.String(collectionID),
		FaceIds:      []string{faceID},
	}

	output, err := p.client.rekognition.DeleteFaces(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case errCodeResourceNotFound:
				deleteErr := fmt.Errorf("tenant %s: %w", p.tenantID, ErrCollectionNotFound)
				p.logAudit(ctx, audit.EventFaceDeleted, false, deleteErr, map[string]string{
					"face_id": faceID,
					"reason":  "collection_not_found",
				})
				return deleteErr
			case errCodeAccessDenied:
				deleteErr := fmt.Errorf("tenant %s: %w", p.tenantID, ErrInvalidCredentials)
				p.logAudit(ctx, audit.EventFaceDeleted, false, deleteErr, map[string]string{
					"face_id": faceID,
					"reason":  "access_denied",
				})
				return deleteErr
			}
		}
		p.logAudit(ctx, audit.EventFaceDeleted, false, err, map[string]string{
			"face_id": faceID,
		})
		return fmt.Errorf("tenant %s: delete face: %w", p.tenantID, err)
	}

	// Check if the face was actually deleted
	if len(output.DeletedFaces) == 0 {
		notFoundErr := fmt.Errorf("tenant %s: %w", p.tenantID, ErrFaceNotFound)
		p.logAudit(ctx, audit.EventFaceDeleted, false, notFoundErr, map[string]string{
			"face_id": faceID,
			"reason":  "face_not_found",
		})
		return notFoundErr
	}

	p.logAudit(ctx, audit.EventFaceDeleted, true, nil, map[string]string{
		"face_id": faceID,
	})

	return nil
}

// SearchFacesByImage searches for similar faces in the collection using an image
// This is a Rekognition-specific operation not part of the base FaceProvider interface
// Returns up to maxFaces matches with similarity above the threshold
func (p *Provider) SearchFacesByImage(ctx context.Context, image []byte, maxFaces int, similarityThreshold float64) ([]SearchResult, error) {
	if err := validateImage(image); err != nil {
		p.logAudit(ctx, audit.EventFaceSearched, false, err, map[string]string{
			"image_size": strconv.Itoa(len(image)),
			"max_faces":  strconv.Itoa(maxFaces),
			"threshold":  fmt.Sprintf("%.4f", similarityThreshold),
		})
		return nil, fmt.Errorf("tenant %s: %w", p.tenantID, err)
	}

	collectionID := p.client.config.CollectionName(p.tenantID.String())

	// Validate maxFaces to prevent integer overflow
	if maxFaces < 0 || maxFaces > 4096 {
		validationErr := fmt.Errorf("tenant %s: maxFaces must be between 0 and 4096, got %d", p.tenantID, maxFaces)
		p.logAudit(ctx, audit.EventFaceSearched, false, validationErr, map[string]string{
			"image_size": strconv.Itoa(len(image)),
			"max_faces":  strconv.Itoa(maxFaces),
			"threshold":  fmt.Sprintf("%.4f", similarityThreshold),
			"reason":     "invalid_max_faces",
		})
		return nil, validationErr
	}

	input := &rekognition.SearchFacesByImageInput{
		CollectionId: aws.String(collectionID),
		Image: &types.Image{
			Bytes: image,
		},
		MaxFaces:           aws.Int32(int32(maxFaces)),                      // #nosec G115 - validated above
		FaceMatchThreshold: aws.Float32(float32(similarityThreshold * 100)), // Convert 0-1 to 0-100
	}

	output, err := p.client.rekognition.SearchFacesByImage(ctx, input)
	if err != nil {
		// Check if error is due to no face detected
		if parsedErr := ParseNoFaceError(err); parsedErr != nil {
			p.logAudit(ctx, audit.EventFaceSearched, false, parsedErr, map[string]string{
				"image_size": strconv.Itoa(len(image)),
				"max_faces":  strconv.Itoa(maxFaces),
				"threshold":  fmt.Sprintf("%.4f", similarityThreshold),
			})
			return nil, fmt.Errorf("tenant %s: %w", p.tenantID, parsedErr)
		}

		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case errCodeResourceNotFound:
				searchErr := fmt.Errorf("tenant %s: %w", p.tenantID, ErrCollectionNotFound)
				p.logAudit(ctx, audit.EventFaceSearched, false, searchErr, map[string]string{
					"image_size": strconv.Itoa(len(image)),
					"max_faces":  strconv.Itoa(maxFaces),
					"threshold":  fmt.Sprintf("%.4f", similarityThreshold),
					"reason":     "collection_not_found",
				})
				return nil, searchErr
			case errCodeAccessDenied:
				searchErr := fmt.Errorf("tenant %s: %w", p.tenantID, ErrInvalidCredentials)
				p.logAudit(ctx, audit.EventFaceSearched, false, searchErr, map[string]string{
					"image_size": strconv.Itoa(len(image)),
					"max_faces":  strconv.Itoa(maxFaces),
					"threshold":  fmt.Sprintf("%.4f", similarityThreshold),
					"reason":     "access_denied",
				})
				return nil, searchErr
			}
		}
		p.logAudit(ctx, audit.EventFaceSearched, false, err, map[string]string{
			"image_size": strconv.Itoa(len(image)),
			"max_faces":  strconv.Itoa(maxFaces),
			"threshold":  fmt.Sprintf("%.4f", similarityThreshold),
		})
		return nil, fmt.Errorf("tenant %s: search faces by image: %w", p.tenantID, err)
	}

	// Convert AWS Rekognition matches to SearchResult
	results := make([]SearchResult, 0, len(output.FaceMatches))
	for _, match := range output.FaceMatches {
		results = append(results, SearchResult{
			FaceID:     *match.Face.FaceId,
			Similarity: float64(*match.Similarity) / 100.0, // Normalize to 0-1
			ExternalImageID: func() string {
				if match.Face.ExternalImageId != nil {
					return *match.Face.ExternalImageId
				}
				return ""
			}(),
		})
	}

	p.logAudit(ctx, audit.EventFaceSearched, true, nil, map[string]string{
		"image_size":    strconv.Itoa(len(image)),
		"max_faces":     strconv.Itoa(maxFaces),
		"threshold":     fmt.Sprintf("%.4f", similarityThreshold),
		"matches_found": strconv.Itoa(len(results)),
	})

	return results, nil
}

// SearchResult represents a face match result from Rekognition search
type SearchResult struct {
	FaceID          string  // Rekognition-generated face ID
	Similarity      float64 // Similarity score (0-1)
	ExternalImageID string  // External image ID if set during IndexFaces
}

// calculateQualityScore computes an overall quality score from Rekognition quality metrics
// Returns a score between 0.0 (poor quality) and 1.0 (excellent quality)
func (p *Provider) calculateQualityScore(quality *types.ImageQuality) float64 {
	if quality == nil {
		return 0.0
	}

	// AWS Rekognition provides brightness and sharpness scores (0-100)
	// We normalize and average them to get an overall quality score
	brightness := 0.0
	sharpness := 0.0

	if quality.Brightness != nil {
		brightness = float64(*quality.Brightness) / 100.0
	}

	if quality.Sharpness != nil {
		sharpness = float64(*quality.Sharpness) / 100.0
	}

	// Weight sharpness more heavily as it's critical for face recognition
	qualityScore := (brightness*0.3 + sharpness*0.7)

	return qualityScore
}

// CreateCollection creates the Rekognition collection for this provider's tenant
// This is typically called automatically during provider initialization
func (p *Provider) CreateCollection(ctx context.Context) error {
	return p.client.CreateCollection(ctx, p.tenantID.String())
}

// DeleteCollection deletes the Rekognition collection for this provider's tenant
// WARNING: This will permanently delete all indexed faces for the tenant
func (p *Provider) DeleteCollection(ctx context.Context) error {
	return p.client.DeleteCollection(ctx, p.tenantID.String())
}

// GetFaceCount returns the number of faces indexed in this tenant's collection
func (p *Provider) GetFaceCount(ctx context.Context) (int64, error) {
	return p.client.GetCollectionFaceCount(ctx, p.tenantID.String())
}
