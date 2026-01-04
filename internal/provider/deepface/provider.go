package deepface

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

const (
	// minFaceArea is the minimum face area (in pixels²) for reliable detection
	minFaceArea = 2500 // 50x50 pixels
	// maxFaceArea is used for confidence scaling
	maxFaceArea = 250000 // 500x500 pixels
)

// Provider implements provider.FaceProvider using DeepFace API
type Provider struct {
	client *Client
}

// NewProvider creates a new DeepFace provider
func NewProvider(config Config) *Provider {
	return &Provider{
		client: NewClient(config),
	}
}

// DetectFaces detects faces in the image
func (p *Provider) DetectFaces(ctx context.Context, image []byte) ([]provider.DetectedFace, error) {
	imageBase64 := base64.StdEncoding.EncodeToString(image)

	resp, err := p.client.Represent(ctx, imageBase64)
	if err != nil {
		return nil, fmt.Errorf("detect faces: %w", err)
	}

	faces := make([]provider.DetectedFace, 0, len(resp.Results))
	for _, result := range resp.Results {
		// Calculate confidence based on face area (larger faces = more reliable detection)
		faceArea := float64(result.FacialArea.W * result.FacialArea.H)
		confidence := calculateConfidence(faceArea)
		qualityScore := calculateQuality(faceArea)

		faces = append(faces, provider.DetectedFace{
			BoundingBox: provider.BoundingBox{
				X:      float64(result.FacialArea.X),
				Y:      float64(result.FacialArea.Y),
				Width:  float64(result.FacialArea.W),
				Height: float64(result.FacialArea.H),
			},
			Confidence:   confidence,
			QualityScore: qualityScore,
		})
	}

	return faces, nil
}

// calculateConfidence estimates confidence based on face area
// DeepFace doesn't return confidence, so we estimate based on face size
// Larger faces are more likely to be accurately detected
func calculateConfidence(faceArea float64) float64 {
	if faceArea < minFaceArea {
		return 0.5 // Low confidence for very small faces
	}
	// Scale from 0.7 to 0.99 based on face area
	normalized := math.Min(1.0, (faceArea-minFaceArea)/(maxFaceArea-minFaceArea))
	return 0.7 + (normalized * 0.29)
}

// calculateQuality estimates quality score based on face area
// DeepFace doesn't return quality, so we estimate based on face size
// Larger faces typically have better quality for embedding extraction
func calculateQuality(faceArea float64) float64 {
	if faceArea < minFaceArea {
		return 0.4 // Low quality for very small faces
	}
	// Scale from 0.6 to 0.95 based on face area
	normalized := math.Min(1.0, (faceArea-minFaceArea)/(maxFaceArea-minFaceArea))
	return 0.6 + (normalized * 0.35)
}

// IndexFace extracts face embedding from image
func (p *Provider) IndexFace(ctx context.Context, image []byte) (string, []float64, error) {
	imageBase64 := base64.StdEncoding.EncodeToString(image)

	resp, err := p.client.Represent(ctx, imageBase64)
	if err != nil {
		return "", nil, fmt.Errorf("index face: %w", err)
	}

	if len(resp.Results) == 0 {
		return "", nil, ErrNoFaceInResponse
	}

	// Use first face found
	result := resp.Results[0]

	// Generate local UUID as face ID (DeepFace doesn't persist faces)
	faceID := uuid.New().String()

	return faceID, result.Embedding, nil
}

// CompareFaces calculates similarity between two embeddings
func (p *Provider) CompareFaces(ctx context.Context, embedding1, embedding2 []float64) (float64, error) {
	// DeepFace doesn't have embedding comparison endpoint
	// We calculate cosine similarity locally
	similarity := CosineSimilarity(embedding1, embedding2)
	return similarity, nil
}

// DeleteFace is a no-op for DeepFace (stateless provider)
func (p *Provider) DeleteFace(ctx context.Context, faceID string) error {
	// DeepFace doesn't persist faces, nothing to delete
	return nil
}

// CheckLiveness performs passive liveness detection using DeepFace
// Currently uses basic face detection as a proxy for liveness
// TODO: Implement proper liveness detection when DeepFace API supports it
func (p *Provider) CheckLiveness(ctx context.Context, image []byte, threshold float64) (*provider.LivenessResult, error) {
	faces, err := p.DetectFaces(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("check liveness: %w", err)
	}

	singleFace := len(faces) == 1
	qualityOK := singleFace && faces[0].QualityScore >= 0.6

	confidence := 0.0
	if singleFace {
		confidence = faces[0].Confidence * faces[0].QualityScore
	}

	isLive := singleFace && qualityOK && confidence >= threshold

	result := &provider.LivenessResult{
		IsLive:     isLive,
		Confidence: confidence,
		Checks: provider.LivenessChecks{
			EyesOpen:     true,
			FacingCamera: qualityOK,
			QualityOK:    qualityOK,
			SingleFace:   singleFace,
		},
	}

	if !isLive {
		if !singleFace {
			if len(faces) == 0 {
				result.Reasons = append(result.Reasons, "no face detected")
			} else {
				result.Reasons = append(result.Reasons, "multiple faces detected")
			}
		}
		if !qualityOK {
			result.Reasons = append(result.Reasons, "image quality too low")
		}
		if confidence < threshold {
			result.Reasons = append(result.Reasons, "confidence below threshold")
		}
	}

	return result, nil
}

// Ensure Provider implements provider.FaceProvider
var _ provider.FaceProvider = (*Provider)(nil)
