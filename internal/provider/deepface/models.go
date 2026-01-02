package deepface

// RepresentRequest for POST /represent
type RepresentRequest struct {
	Img      string `json:"img"`      // base64 encoded image
	Model    string `json:"model"`    // "Facenet512", "VGG-Face", etc
	Detector string `json:"detector"` // "retinaface", "mtcnn", etc
}

// RepresentResponse from POST /represent
type RepresentResponse struct {
	Results []RepresentResult `json:"results"`
}

type RepresentResult struct {
	Embedding  []float64  `json:"embedding"`
	FacialArea FacialArea `json:"facial_area"`
}

type FacialArea struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// AnalyzeRequest for POST /analyze
type AnalyzeRequest struct {
	Img      string   `json:"img"`
	Actions  []string `json:"actions"` // ["age", "gender", "emotion", "race"]
	Detector string   `json:"detector"`
}

// AnalyzeResponse from POST /analyze
type AnalyzeResponse struct {
	Results []AnalyzeResult `json:"results"`
}

type AnalyzeResult struct {
	Region  FacialArea         `json:"region"`
	Age     int                `json:"age"`
	Gender  map[string]float64 `json:"gender"`
	Emotion map[string]float64 `json:"emotion"`
	Race    map[string]float64 `json:"race"`
}
