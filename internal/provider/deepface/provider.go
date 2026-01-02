package deepface

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
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
		faces = append(faces, provider.DetectedFace{
			BoundingBox: provider.BoundingBox{
				X:      float64(result.FacialArea.X),
				Y:      float64(result.FacialArea.Y),
				Width:  float64(result.FacialArea.W),
				Height: float64(result.FacialArea.H),
			},
			Confidence:   0.99, // DeepFace doesn't return confidence, assume high if detected
			QualityScore: 0.95, // DeepFace doesn't return quality, assume good
		})
	}

	return faces, nil
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

// Ensure Provider implements provider.FaceProvider
var _ provider.FaceProvider = (*Provider)(nil)
