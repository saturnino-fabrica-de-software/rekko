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

	// CheckLiveness performs passive liveness detection on an image
	// Returns liveness result with confidence and individual checks
	CheckLiveness(ctx context.Context, image []byte, threshold float64) (*LivenessResult, error)
}

// DetectedFace represents a detected face in the image
type DetectedFace struct {
	BoundingBox  BoundingBox `json:"bounding_box"`
	Confidence   float64     `json:"confidence"`
	QualityScore float64     `json:"quality_score"`
	EyesOpen     *bool       `json:"eyes_open,omitempty"`
	Pose         *Pose       `json:"pose,omitempty"`
}

// Pose represents face orientation angles
type Pose struct {
	Pitch float64 `json:"pitch"` // up/down rotation
	Roll  float64 `json:"roll"`  // tilted rotation
	Yaw   float64 `json:"yaw"`   // left/right rotation
}

// BoundingBox represents the face area in the image
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LivenessResult represents the result of a liveness check
type LivenessResult struct {
	IsLive     bool           `json:"is_live"`
	Confidence float64        `json:"confidence"`
	Reasons    []string       `json:"reasons,omitempty"`
	Checks     LivenessChecks `json:"checks"`
}

// LivenessChecks contains individual liveness check results
type LivenessChecks struct {
	EyesOpen     bool `json:"eyes_open"`
	FacingCamera bool `json:"facing_camera"`
	QualityOK    bool `json:"quality_ok"`
	SingleFace   bool `json:"single_face"`
}
