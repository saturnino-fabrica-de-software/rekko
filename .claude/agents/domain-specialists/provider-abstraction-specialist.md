---
name: provider-abstraction-specialist
description: Provider abstraction pattern specialist. Use EXCLUSIVELY for designing interfaces that allow seamless swapping between DeepFace (dev) and AWS Rekognition (prod), adapter patterns, and provider factory implementation.
tools: Read, Write, Edit, Glob, Grep, Bash
model: sonnet
mcp_integrations:
  - context7: Validate Go interface patterns and adapter design
---

# provider-abstraction-specialist

---

## 🎯 Purpose

The `provider-abstraction-specialist` is responsible for:

1. **Interface Design** - Define contracts for face recognition providers
2. **Adapter Pattern** - Implement adapters for DeepFace and Rekognition
3. **Provider Factory** - Runtime provider selection
4. **Configuration** - Environment-based provider switching
5. **Error Mapping** - Consistent error handling across providers

---

## 🚨 CRITICAL RULES

### Rule 1: Interface First
```go
// Define interface BEFORE any implementation
// All consumers depend on interface, never on concrete types
```

### Rule 2: Provider Transparency
```go
// Caller should NOT know which provider is being used
// Same input → Same output (semantically)
```

### Rule 3: Environment-Based Selection
```go
Dev/Test: DeepFace (free, local, Docker)
Staging:  AWS Rekognition (real infra, test data)
Prod:     AWS Rekognition (full scale)
```

---

## 📋 Provider Architecture

### 1. Core Interface

```go
// internal/provider/provider.go
package provider

import "context"

// FaceProvider defines the contract for face recognition backends
type FaceProvider interface {
    // RegisterFace indexes a face and returns its embedding
    RegisterFace(ctx context.Context, req RegisterFaceRequest) (*RegisterFaceResult, error)

    // VerifyFace compares a face against a registered face (1:1)
    VerifyFace(ctx context.Context, req VerifyFaceRequest) (*VerifyFaceResult, error)

    // SearchFace finds matching faces in the collection (1:N)
    SearchFace(ctx context.Context, req SearchFaceRequest) (*SearchFaceResult, error)

    // DeleteFace removes a face from the collection
    DeleteFace(ctx context.Context, tenantID, externalID string) error

    // CheckLiveness verifies the image is from a live person
    CheckLiveness(ctx context.Context, imageData []byte) (*LivenessResult, error)

    // AssessQuality evaluates image quality for face recognition
    AssessQuality(ctx context.Context, imageData []byte) (*QualityResult, error)

    // SupportsNativeSearch returns true if provider has built-in search
    SupportsNativeSearch() bool

    // ProviderName returns the provider identifier
    ProviderName() string
}

// Request/Result types (provider-agnostic)
type RegisterFaceRequest struct {
    TenantID      string
    ExternalID    string
    ImageData     []byte
    CollectionID  string // Optional, provider-specific
}

type RegisterFaceResult struct {
    FaceID        string
    Embedding     []float64
    Quality       float64
    BoundingBox   BoundingBox
    LivenessCheck *LivenessResult
}

type VerifyFaceRequest struct {
    TenantID     string
    ExternalID   string
    ImageData    []byte
    TargetImage  []byte    // Optional: for direct comparison
    Threshold    float64
}

type VerifyFaceResult struct {
    Matched     bool
    Confidence  float64
    FaceID      string
}

type SearchFaceRequest struct {
    TenantID     string
    ImageData    []byte
    MaxResults   int
    Threshold    float64
    CollectionID string
}

type SearchFaceResult struct {
    Matches       []FaceMatch
    TotalSearched int
}

type FaceMatch struct {
    ExternalID string
    FaceID     string
    Confidence float64
    BoundingBox BoundingBox
}

type LivenessResult struct {
    IsLive      bool
    Confidence  float64
    SpoofType   string // empty if live
}

type QualityResult struct {
    Overall     float64
    Sharpness   float64
    Brightness  float64
    Pose        Pose
    IsAcceptable bool
    Issues      []string
}

type BoundingBox struct {
    Top    float64
    Left   float64
    Width  float64
    Height float64
}

type Pose struct {
    Yaw   float64
    Pitch float64
    Roll  float64
}
```

### 2. DeepFace Adapter

```go
// internal/provider/deepface/deepface.go
package deepface

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

// Config for DeepFace provider
type Config struct {
    BaseURL     string        `env:"DEEPFACE_URL" envDefault:"http://localhost:5000"`
    Timeout     time.Duration `env:"DEEPFACE_TIMEOUT" envDefault:"30s"`
    Model       string        `env:"DEEPFACE_MODEL" envDefault:"ArcFace"`
    Detector    string        `env:"DEEPFACE_DETECTOR" envDefault:"retinaface"`
}

// DeepFaceProvider implements FaceProvider using DeepFace API
type DeepFaceProvider struct {
    config     Config
    httpClient *http.Client
}

// Ensure interface compliance
var _ provider.FaceProvider = (*DeepFaceProvider)(nil)

// New creates a DeepFace provider
func New(cfg Config) *DeepFaceProvider {
    return &DeepFaceProvider{
        config: cfg,
        httpClient: &http.Client{
            Timeout: cfg.Timeout,
        },
    }
}

func (d *DeepFaceProvider) ProviderName() string {
    return "deepface"
}

func (d *DeepFaceProvider) SupportsNativeSearch() bool {
    return false // DeepFace doesn't have built-in collection search
}

// RegisterFace generates embedding using DeepFace represent endpoint
func (d *DeepFaceProvider) RegisterFace(ctx context.Context, req provider.RegisterFaceRequest) (*provider.RegisterFaceResult, error) {
    // Encode image to base64
    imgBase64 := base64.StdEncoding.EncodeToString(req.ImageData)

    // Call DeepFace /represent endpoint
    payload := map[string]interface{}{
        "img":           "data:image/jpeg;base64," + imgBase64,
        "model_name":    d.config.Model,
        "detector_backend": d.config.Detector,
    }

    respBody, err := d.doRequest(ctx, "/represent", payload)
    if err != nil {
        return nil, fmt.Errorf("deepface represent: %w", err)
    }

    // Parse response
    var result struct {
        Results []struct {
            Embedding   []float64 `json:"embedding"`
            FacialArea  struct {
                X int `json:"x"`
                Y int `json:"y"`
                W int `json:"w"`
                H int `json:"h"`
            } `json:"facial_area"`
        } `json:"results"`
    }

    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }

    if len(result.Results) == 0 {
        return nil, provider.ErrNoFaceDetected
    }

    if len(result.Results) > 1 {
        return nil, provider.ErrMultipleFaces
    }

    face := result.Results[0]

    return &provider.RegisterFaceResult{
        FaceID:    req.ExternalID, // DeepFace doesn't generate IDs
        Embedding: face.Embedding,
        Quality:   0.95, // DeepFace doesn't return quality, assume good
        BoundingBox: provider.BoundingBox{
            Left:   float64(face.FacialArea.X),
            Top:    float64(face.FacialArea.Y),
            Width:  float64(face.FacialArea.W),
            Height: float64(face.FacialArea.H),
        },
    }, nil
}

// VerifyFace compares two faces using DeepFace verify endpoint
func (d *DeepFaceProvider) VerifyFace(ctx context.Context, req provider.VerifyFaceRequest) (*provider.VerifyFaceResult, error) {
    if req.TargetImage == nil {
        return nil, fmt.Errorf("target image required for verification")
    }

    img1Base64 := base64.StdEncoding.EncodeToString(req.ImageData)
    img2Base64 := base64.StdEncoding.EncodeToString(req.TargetImage)

    payload := map[string]interface{}{
        "img1_path":        "data:image/jpeg;base64," + img1Base64,
        "img2_path":        "data:image/jpeg;base64," + img2Base64,
        "model_name":       d.config.Model,
        "detector_backend": d.config.Detector,
    }

    respBody, err := d.doRequest(ctx, "/verify", payload)
    if err != nil {
        return nil, fmt.Errorf("deepface verify: %w", err)
    }

    var result struct {
        Verified bool    `json:"verified"`
        Distance float64 `json:"distance"`
        Threshold float64 `json:"threshold"`
    }

    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }

    // Convert distance to confidence (inverse relationship)
    confidence := 1.0 - result.Distance

    return &provider.VerifyFaceResult{
        Matched:    result.Verified,
        Confidence: confidence,
    }, nil
}

// CheckLiveness - DeepFace doesn't support liveness, return best-effort
func (d *DeepFaceProvider) CheckLiveness(ctx context.Context, imageData []byte) (*provider.LivenessResult, error) {
    // DeepFace doesn't have built-in liveness
    // For dev, we can use a basic check or skip
    // In prod, use a dedicated liveness provider

    // For now, return "live" with warning
    return &provider.LivenessResult{
        IsLive:     true,
        Confidence: 0.5, // Low confidence = not really checked
    }, nil
}

// AssessQuality - Basic quality check using face detection
func (d *DeepFaceProvider) AssessQuality(ctx context.Context, imageData []byte) (*provider.QualityResult, error) {
    // Try to detect face - if detected, quality is acceptable
    imgBase64 := base64.StdEncoding.EncodeToString(imageData)

    payload := map[string]interface{}{
        "img":              "data:image/jpeg;base64," + imgBase64,
        "detector_backend": d.config.Detector,
        "enforce_detection": true,
    }

    respBody, err := d.doRequest(ctx, "/represent", payload)
    if err != nil {
        return &provider.QualityResult{
            Overall:      0,
            IsAcceptable: false,
            Issues:       []string{"face detection failed"},
        }, nil
    }

    var result struct {
        Results []struct {
            FacialArea struct {
                W int `json:"w"`
                H int `json:"h"`
            } `json:"facial_area"`
        } `json:"results"`
    }

    json.Unmarshal(respBody, &result)

    if len(result.Results) == 0 {
        return &provider.QualityResult{
            Overall:      0,
            IsAcceptable: false,
            Issues:       []string{"no face detected"},
        }, nil
    }

    // Basic quality assessment based on face size
    faceArea := result.Results[0].FacialArea.W * result.Results[0].FacialArea.H
    quality := min(1.0, float64(faceArea)/50000) // Normalize to 0-1

    return &provider.QualityResult{
        Overall:      quality,
        IsAcceptable: quality >= 0.6,
    }, nil
}

func (d *DeepFaceProvider) SearchFace(ctx context.Context, req provider.SearchFaceRequest) (*provider.SearchFaceResult, error) {
    // DeepFace doesn't support collection search
    // This should be handled at service layer
    return nil, provider.ErrOperationNotSupported
}

func (d *DeepFaceProvider) DeleteFace(ctx context.Context, tenantID, externalID string) error {
    // DeepFace is stateless - deletion handled at DB level
    return nil
}

// doRequest makes HTTP request to DeepFace API
func (d *DeepFaceProvider) doRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
    body, _ := json.Marshal(payload)

    req, err := http.NewRequestWithContext(ctx, "POST", d.config.BaseURL+endpoint, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := d.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respBody := new(bytes.Buffer)
    respBody.ReadFrom(resp.Body)

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("deepface error: %s", respBody.String())
    }

    return respBody.Bytes(), nil
}
```

### 3. AWS Rekognition Adapter

```go
// internal/provider/rekognition/rekognition.go
package rekognition

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/rekognition"
    "github.com/aws/aws-sdk-go-v2/service/rekognition/types"

    "github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

// Config for Rekognition provider
type Config struct {
    Region              string `env:"AWS_REGION" envDefault:"us-east-1"`
    CollectionIDPrefix  string `env:"REKOGNITION_COLLECTION_PREFIX" envDefault:"rekko-"`
    QualityThreshold    float64 `env:"REKOGNITION_QUALITY_THRESHOLD" envDefault:"0.9"`
}

// RekognitionProvider implements FaceProvider using AWS Rekognition
type RekognitionProvider struct {
    client *rekognition.Client
    config Config
}

var _ provider.FaceProvider = (*RekognitionProvider)(nil)

// New creates a Rekognition provider
func New(ctx context.Context, cfg Config) (*RekognitionProvider, error) {
    awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
    if err != nil {
        return nil, fmt.Errorf("load aws config: %w", err)
    }

    return &RekognitionProvider{
        client: rekognition.NewFromConfig(awsCfg),
        config: cfg,
    }, nil
}

func (r *RekognitionProvider) ProviderName() string {
    return "rekognition"
}

func (r *RekognitionProvider) SupportsNativeSearch() bool {
    return true // Rekognition has built-in face search
}

func (r *RekognitionProvider) collectionID(tenantID string) string {
    return r.config.CollectionIDPrefix + tenantID
}

// RegisterFace indexes a face in Rekognition collection
func (r *RekognitionProvider) RegisterFace(ctx context.Context, req provider.RegisterFaceRequest) (*provider.RegisterFaceResult, error) {
    // Ensure collection exists
    if err := r.ensureCollection(ctx, req.TenantID); err != nil {
        return nil, err
    }

    input := &rekognition.IndexFacesInput{
        CollectionId:        aws.String(r.collectionID(req.TenantID)),
        Image:               &types.Image{Bytes: req.ImageData},
        ExternalImageId:     aws.String(req.ExternalID),
        MaxFaces:            aws.Int32(1),
        QualityFilter:       types.QualityFilterAuto,
        DetectionAttributes: []types.Attribute{types.AttributeAll},
    }

    result, err := r.client.IndexFaces(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("index face: %w", err)
    }

    if len(result.FaceRecords) == 0 {
        if len(result.UnindexedFaces) > 0 {
            reason := result.UnindexedFaces[0].Reasons
            return nil, fmt.Errorf("face not indexed: %v", reason)
        }
        return nil, provider.ErrNoFaceDetected
    }

    face := result.FaceRecords[0]

    return &provider.RegisterFaceResult{
        FaceID:  *face.Face.FaceId,
        Quality: float64(*face.Face.Confidence) / 100,
        BoundingBox: provider.BoundingBox{
            Top:    float64(*face.Face.BoundingBox.Top),
            Left:   float64(*face.Face.BoundingBox.Left),
            Width:  float64(*face.Face.BoundingBox.Width),
            Height: float64(*face.Face.BoundingBox.Height),
        },
    }, nil
}

// VerifyFace uses CompareFaces for 1:1 verification
func (r *RekognitionProvider) VerifyFace(ctx context.Context, req provider.VerifyFaceRequest) (*provider.VerifyFaceResult, error) {
    if req.TargetImage == nil {
        return nil, fmt.Errorf("target image required")
    }

    input := &rekognition.CompareFacesInput{
        SourceImage:         &types.Image{Bytes: req.ImageData},
        TargetImage:         &types.Image{Bytes: req.TargetImage},
        SimilarityThreshold: aws.Float32(float32(req.Threshold * 100)),
    }

    result, err := r.client.CompareFaces(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("compare faces: %w", err)
    }

    if len(result.FaceMatches) == 0 {
        return &provider.VerifyFaceResult{
            Matched:    false,
            Confidence: 0,
        }, nil
    }

    match := result.FaceMatches[0]

    return &provider.VerifyFaceResult{
        Matched:    true,
        Confidence: float64(*match.Similarity) / 100,
    }, nil
}

// SearchFace uses SearchFacesByImage for 1:N search
func (r *RekognitionProvider) SearchFace(ctx context.Context, req provider.SearchFaceRequest) (*provider.SearchFaceResult, error) {
    input := &rekognition.SearchFacesByImageInput{
        CollectionId:       aws.String(r.collectionID(req.TenantID)),
        Image:              &types.Image{Bytes: req.ImageData},
        MaxFaces:           aws.Int32(int32(req.MaxResults)),
        FaceMatchThreshold: aws.Float32(float32(req.Threshold * 100)),
    }

    result, err := r.client.SearchFacesByImage(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("search faces: %w", err)
    }

    matches := make([]provider.FaceMatch, 0, len(result.FaceMatches))
    for _, match := range result.FaceMatches {
        matches = append(matches, provider.FaceMatch{
            FaceID:     *match.Face.FaceId,
            ExternalID: *match.Face.ExternalImageId,
            Confidence: float64(*match.Similarity) / 100,
        })
    }

    return &provider.SearchFaceResult{
        Matches:       matches,
        TotalSearched: len(result.FaceMatches),
    }, nil
}

// CheckLiveness uses DetectFaces with attributes
func (r *RekognitionProvider) CheckLiveness(ctx context.Context, imageData []byte) (*provider.LivenessResult, error) {
    // Note: For production, use CreateFaceLivenessSession API
    // This is a simplified version using face attributes

    input := &rekognition.DetectFacesInput{
        Image:      &types.Image{Bytes: imageData},
        Attributes: []types.Attribute{types.AttributeAll},
    }

    result, err := r.client.DetectFaces(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("detect faces: %w", err)
    }

    if len(result.FaceDetails) == 0 {
        return nil, provider.ErrNoFaceDetected
    }

    face := result.FaceDetails[0]

    // Check for sunglasses, open eyes (basic liveness indicators)
    eyesOpen := face.EyesOpen != nil && *face.EyesOpen.Value
    confidence := float64(*face.Confidence) / 100

    return &provider.LivenessResult{
        IsLive:     eyesOpen,
        Confidence: confidence,
    }, nil
}

// DeleteFace removes face from collection
func (r *RekognitionProvider) DeleteFace(ctx context.Context, tenantID, externalID string) error {
    // First, find the face by external ID
    listInput := &rekognition.ListFacesInput{
        CollectionId: aws.String(r.collectionID(tenantID)),
        MaxResults:   aws.Int32(1000),
    }

    listResult, err := r.client.ListFaces(ctx, listInput)
    if err != nil {
        return fmt.Errorf("list faces: %w", err)
    }

    var faceID string
    for _, face := range listResult.Faces {
        if face.ExternalImageId != nil && *face.ExternalImageId == externalID {
            faceID = *face.FaceId
            break
        }
    }

    if faceID == "" {
        return provider.ErrFaceNotFound
    }

    // Delete the face
    deleteInput := &rekognition.DeleteFacesInput{
        CollectionId: aws.String(r.collectionID(tenantID)),
        FaceIds:      []string{faceID},
    }

    _, err = r.client.DeleteFaces(ctx, deleteInput)
    return err
}

func (r *RekognitionProvider) ensureCollection(ctx context.Context, tenantID string) error {
    collID := r.collectionID(tenantID)

    _, err := r.client.CreateCollection(ctx, &rekognition.CreateCollectionInput{
        CollectionId: aws.String(collID),
    })

    if err != nil {
        // Ignore if already exists
        var resourceExists *types.ResourceAlreadyExistsException
        if !errors.As(err, &resourceExists) {
            return fmt.Errorf("create collection: %w", err)
        }
    }

    return nil
}
```

### 4. Provider Factory

```go
// internal/provider/factory.go
package provider

import (
    "context"
    "fmt"
    "os"

    "github.com/saturnino-fabrica-de-software/rekko/internal/provider/deepface"
    "github.com/saturnino-fabrica-de-software/rekko/internal/provider/rekognition"
)

// ProviderType identifies the face recognition provider
type ProviderType string

const (
    ProviderDeepFace    ProviderType = "deepface"
    ProviderRekognition ProviderType = "rekognition"
)

// NewProvider creates a face provider based on configuration
func NewProvider(ctx context.Context) (FaceProvider, error) {
    providerType := ProviderType(os.Getenv("FACE_PROVIDER"))

    switch providerType {
    case ProviderDeepFace, "":
        cfg := deepface.Config{
            BaseURL: os.Getenv("DEEPFACE_URL"),
            Model:   os.Getenv("DEEPFACE_MODEL"),
        }
        if cfg.BaseURL == "" {
            cfg.BaseURL = "http://localhost:5000"
        }
        if cfg.Model == "" {
            cfg.Model = "ArcFace"
        }
        return deepface.New(cfg), nil

    case ProviderRekognition:
        cfg := rekognition.Config{
            Region: os.Getenv("AWS_REGION"),
        }
        if cfg.Region == "" {
            cfg.Region = "us-east-1"
        }
        return rekognition.New(ctx, cfg)

    default:
        return nil, fmt.Errorf("unknown provider: %s", providerType)
    }
}
```

---

## 📊 Error Mapping

```go
// internal/provider/errors.go
package provider

import "errors"

// Common provider errors (provider-agnostic)
var (
    ErrNoFaceDetected        = errors.New("no face detected in image")
    ErrMultipleFaces         = errors.New("multiple faces detected")
    ErrFaceNotFound          = errors.New("face not found")
    ErrLivenessFailed        = errors.New("liveness check failed")
    ErrQualityTooLow         = errors.New("image quality too low")
    ErrOperationNotSupported = errors.New("operation not supported by provider")
    ErrProviderUnavailable   = errors.New("provider temporarily unavailable")
    ErrRateLimited           = errors.New("provider rate limit exceeded")
)
```

---

## ✅ Checklist Before Completing

- [ ] Interface defined before implementations
- [ ] All provider methods return provider-agnostic types
- [ ] Error mapping consistent across providers
- [ ] Factory supports environment-based selection
- [ ] Compile-time interface compliance verified
- [ ] No provider-specific types leak to callers
- [ ] Tests mock the interface, not concrete types
