package provider

import "context"

// FaceProvider define a interface para provedores de reconhecimento facial
type FaceProvider interface {
	// DetectFaces detecta faces na imagem e retorna informações sobre cada uma
	DetectFaces(ctx context.Context, image []byte) ([]DetectedFace, error)

	// IndexFace extrai embedding de uma face na imagem
	// Retorna faceID (identificador do provider), embedding e erro
	IndexFace(ctx context.Context, image []byte) (faceID string, embedding []float64, err error)

	// CompareFaces calcula similaridade entre dois embeddings
	// Retorna valor entre 0.0 (diferentes) e 1.0 (idênticos)
	CompareFaces(ctx context.Context, embedding1, embedding2 []float64) (similarity float64, err error)

	// DeleteFace remove face do índice do provider (se aplicável)
	DeleteFace(ctx context.Context, faceID string) error
}

// DetectedFace representa uma face detectada na imagem
type DetectedFace struct {
	BoundingBox  BoundingBox `json:"bounding_box"`
	Confidence   float64     `json:"confidence"`
	QualityScore float64     `json:"quality_score"`
}

// BoundingBox representa a área da face na imagem
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
