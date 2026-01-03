# Face Provider Factory

This package provides a factory function for creating face recognition providers based on runtime configuration.

## Overview

The `face` package acts as an orchestration layer that selects the appropriate face recognition provider (DeepFace or AWS Rekognition) based on environment variables.

## Supported Providers

| Provider | Environment | Description |
|----------|-------------|-------------|
| **DeepFace** | Dev/Test | Local, free, Docker-based provider |
| **AWS Rekognition** | Staging/Prod | Cloud-based, scalable AWS service |

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/google/uuid"
    "github.com/saturnino-fabrica-de-software/rekko/internal/config"
    "github.com/saturnino-fabrica-de-software/rekko/internal/face"
)

func main() {
    ctx := context.Background()

    // Load configuration from environment
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }

    tenantID := uuid.New()

    // Create provider (automatically selects based on FACE_PROVIDER env var)
    provider, err := face.NewFaceProvider(ctx, cfg, tenantID)
    if err != nil {
        panic(err)
    }

    // Use provider transparently
    imageData := loadImage()
    faces, err := provider.DetectFaces(ctx, imageData)
    if err != nil {
        panic(err)
    }
}
```

### Environment Configuration

#### DeepFace (Development)

```bash
# .env.development
FACE_PROVIDER=deepface
DEEPFACE_URL=http://localhost:5000
```

Start DeepFace server:
```bash
docker-compose up deepface
```

#### AWS Rekognition (Production)

```bash
# .env.production
FACE_PROVIDER=rekognition
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
```

Or use AWS credential chain (IAM roles, etc.).

## Provider Selection Logic

1. **FACE_PROVIDER=rekognition** → AWS Rekognition
   - Requires: `AWS_REGION`, AWS credentials
   - Creates tenant-specific collections (`rekko-{tenant_id}`)

2. **FACE_PROVIDER=deepface** → DeepFace
   - Requires: `DEEPFACE_URL` (default: `http://localhost:5000`)
   - Stateless provider (no persistent collections)

3. **FACE_PROVIDER not set** → Defaults to DeepFace

## Multi-Tenancy

- **Rekognition**: Each tenant gets a separate collection (`rekko-{tenant_id}`)
  - Ensures data isolation at AWS level
  - Collection created automatically on first use

- **DeepFace**: Stateless provider
  - Tenant isolation handled at database level
  - No provider-side collections

## Error Handling

```go
provider, err := face.NewFaceProvider(ctx, cfg, tenantID)
if err != nil {
    // Possible errors:
    // - Unknown provider type
    // - AWS configuration error (Rekognition)
    // - Collection creation error (Rekognition)
    return fmt.Errorf("create provider: %w", err)
}
```

## Architecture Benefits

1. **Provider Transparency**: Application code doesn't know which provider is being used
2. **Environment-Based**: Different providers in dev/staging/prod
3. **Type Safety**: Interface ensures all providers implement same contract
4. **No Import Cycles**: Factory in separate package from providers
5. **Testability**: Easy to swap providers in tests

## See Also

- [Provider Interface](../provider/provider.go)
- [DeepFace Provider](../provider/deepface/)
- [Rekognition Provider](../provider/rekognition/)
- [Configuration](../config/config.go)
