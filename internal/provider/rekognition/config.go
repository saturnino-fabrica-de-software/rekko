package rekognition

import "fmt"

// Config holds configuration for AWS Rekognition provider
type Config struct {
	// Region is the AWS region where Rekognition service will be used (e.g., "us-east-1")
	Region string

	// CollectionPrefix is the prefix used to generate collection names
	// Collections will be named as: {CollectionPrefix}{tenant_id}
	CollectionPrefix string
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		Region:           "us-east-1",
		CollectionPrefix: "rekko-",
	}
}

// CollectionName generates the collection name for a given tenant ID
// Format: {CollectionPrefix}{tenantID}
// Example: "rekko-tenant-123"
func (c Config) CollectionName(tenantID string) string {
	return fmt.Sprintf("%s%s", c.CollectionPrefix, tenantID)
}
