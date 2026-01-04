package mock

import (
	"context"
	"crypto/sha256"
	"math"

	"github.com/google/uuid"
	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

const embeddingDimension = 512

// Provider implementa provider.FaceProvider para testes e desenvolvimento
type Provider struct{}

// New cria uma nova instância do MockProvider
func New() *Provider {
	return &Provider{}
}

// DetectFaces simula detecção de faces
func (p *Provider) DetectFaces(ctx context.Context, image []byte) ([]provider.DetectedFace, error) {
	if len(image) < 1000 {
		return nil, domain.ErrInvalidImage
	}

	return []provider.DetectedFace{
		{
			BoundingBox: provider.BoundingBox{
				X:      0.1,
				Y:      0.1,
				Width:  0.8,
				Height: 0.8,
			},
			Confidence:   0.99,
			QualityScore: 0.95,
		},
	}, nil
}

// IndexFace gera embedding determinístico baseado no hash da imagem
func (p *Provider) IndexFace(ctx context.Context, image []byte) (string, []float64, error) {
	if len(image) < 1000 {
		return "", nil, domain.ErrInvalidImage
	}

	faceID := uuid.New().String()
	embedding := generateEmbedding(image)

	return faceID, embedding, nil
}

// CompareFaces calcula similaridade coseno entre embeddings
func (p *Provider) CompareFaces(ctx context.Context, emb1, emb2 []float64) (float64, error) {
	if len(emb1) != embeddingDimension || len(emb2) != embeddingDimension {
		return 0, domain.ErrInvalidImage.WithError(nil)
	}

	return cosineSimilarity(emb1, emb2), nil
}

// DeleteFace simula remoção (no-op para mock)
func (p *Provider) DeleteFace(ctx context.Context, faceID string) error {
	return nil
}

// CheckLiveness performs passive liveness detection (mock returns live)
func (p *Provider) CheckLiveness(ctx context.Context, image []byte, threshold float64) (*provider.LivenessResult, error) {
	if len(image) < 1000 {
		return nil, domain.ErrInvalidImage
	}

	return &provider.LivenessResult{
		IsLive:     true,
		Confidence: 0.95,
		Checks: provider.LivenessChecks{
			EyesOpen:     true,
			FacingCamera: true,
			QualityOK:    true,
			SingleFace:   true,
		},
	}, nil
}

// generateEmbedding gera embedding determinístico baseado no hash da imagem
func generateEmbedding(image []byte) []float64 {
	hash := sha256.Sum256(image)
	embedding := make([]float64, embeddingDimension)
	hashLen := len(hash)

	for i := 0; i < embeddingDimension; i++ {
		idx := i % hashLen
		//nolint:gosec // idx is always < hashLen due to modulo operation
		embedding[i] = (float64(hash[idx])/255.0)*2 - 1
	}

	norm := 0.0
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	for i := range embedding {
		embedding[i] /= norm
	}

	return embedding
}

// cosineSimilarity calcula similaridade coseno entre dois vetores
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

var _ provider.FaceProvider = (*Provider)(nil)
