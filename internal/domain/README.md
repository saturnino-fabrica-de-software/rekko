# Domain Layer

This package contains the core domain entities and business rules for the Rekko FRaaS platform.

## Entities

### Tenant

Represents a B2B client in the system with multi-tenancy support.

```go
tenant := &domain.Tenant{
    ID:       uuid.New(),
    Name:     "Acme Corp",
    Slug:     "acme-corp",
    IsActive: true,
    Plan:     domain.PlanPro,
    Settings: map[string]interface{}{
        "verification_threshold": 0.85,
    },
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
}

// Validate tenant
if err := tenant.Validate(); err != nil {
    log.Fatal(err)
}
```

#### Plans

Available plans:
- `PlanStarter` - Basic tier
- `PlanPro` - Professional tier
- `PlanEnterprise` - Enterprise tier

#### Slug Validation

Tenant slugs must:
- Contain only lowercase letters, numbers, and hyphens
- Not start or end with a hyphen
- Not contain consecutive hyphens

Valid examples: `acme`, `acme-corp`, `acme-corp-123`
Invalid examples: `Acme`, `acme_corp`, `-acme`, `acme-`, `acme--corp`

### API Key

Represents an API key for authentication with support for test and live environments.

```go
// Generate a new API key
plainKey, hash, prefix, err := domain.GenerateAPIKey(domain.EnvTest)
if err != nil {
    log.Fatal(err)
}

// plainKey: "rekko_test_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6" (store this securely, show only once)
// hash: "sha256_hash..." (store in database)
// prefix: "rekko_test_A1B2" (store for display purposes)

// Create API key entity
apiKey := &domain.APIKey{
    ID:          uuid.New(),
    TenantID:    tenant.ID,
    Name:        "Production API Key",
    KeyHash:     hash,
    KeyPrefix:   prefix,
    Environment: domain.EnvLive,
    IsActive:    true,
    CreatedAt:   time.Now(),
}

// Validate API key
if err := apiKey.Validate(); err != nil {
    log.Fatal(err)
}
```

#### API Key Format

API keys follow the format: `rekko_{env}_{32_random_chars}`

- `env`: Either `test` or `live`
- Random part: 32 characters using base62 encoding (0-9, A-Z, a-z)

Examples:
- Test: `rekko_test_A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6`
- Live: `rekko_live_X9Y8Z7W6V5U4T3S2R1Q0P9O8N7M6L5K4`

#### Security

- API keys are generated using `crypto/rand` for cryptographic security
- Keys are hashed with SHA-256 before storage
- Only the hash is stored in the database
- The plain key is shown only once during generation
- Key prefix (first 16 chars) is stored for display purposes

#### Validation

```go
// Validate format
if !domain.IsValidFormat(inputKey) {
    return domain.ErrInvalidAPIKeyFormat
}

// Hash for comparison
hash := domain.HashAPIKey(inputKey)

// Compare with stored hash
if hash != storedHash {
    return domain.ErrUnauthorized
}
```

## Error Handling

All domain errors are pre-defined for consistency:

```go
// Tenant errors
domain.ErrTenantNotFound    // 404
domain.ErrTenantInactive    // 403

// API Key errors
domain.ErrAPIKeyNotFound       // 404
domain.ErrAPIKeyRevoked        // 401
domain.ErrInvalidAPIKeyFormat  // 401

// Face recognition errors
domain.ErrFaceNotFound      // 404
domain.ErrFaceExists        // 409
domain.ErrNoFaceDetected    // 422
domain.ErrMultipleFaces     // 422
domain.ErrLowQualityImage   // 422
domain.ErrLivenessFailed    // 422

// Generic errors
domain.ErrUnauthorized      // 401
domain.ErrForbidden         // 403
domain.ErrBadRequest        // 400
domain.ErrValidationFailed  // 422
domain.ErrRateLimitExceeded // 429
```

### Error Usage

```go
// Wrap errors with context
if err != nil {
    return domain.ErrTenantNotFound.WithError(err)
}

// Check error type
if errors.Is(err, domain.ErrAPIKeyRevoked) {
    // Handle revoked key
}

// Get HTTP status code
statusCode := appError.StatusCode
```

## Default Settings

```go
settings := domain.DefaultTenantSettings()
// {
//     VerificationThreshold: 0.8,
//     MaxFacesPerUser: 1,
//     RequireLiveness: false,
// }
```

## Best Practices

1. **Always validate** entities before persisting:
   ```go
   if err := tenant.Validate(); err != nil {
       return err
   }
   ```

2. **Never store plain API keys** - only store the hash:
   ```go
   plainKey, hash, prefix, _ := domain.GenerateAPIKey(env)
   // Store hash and prefix in DB, return plainKey to user once
   ```

3. **Use pre-defined errors** for consistency:
   ```go
   return domain.ErrTenantNotFound
   ```

4. **Validate API key format** before processing:
   ```go
   if !domain.IsValidFormat(key) {
       return domain.ErrInvalidAPIKeyFormat
   }
   ```

5. **Check tenant status** before operations:
   ```go
   if !tenant.IsActive {
       return domain.ErrTenantInactive
   }
   ```
