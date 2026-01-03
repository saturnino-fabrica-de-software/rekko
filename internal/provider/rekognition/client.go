package rekognition

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/smithy-go"
)

const (
	errCodeAccessDenied     = "AccessDeniedException"
	errCodeResourceNotFound = "ResourceNotFoundException"
	errCodeResourceExists   = "ResourceAlreadyExistsException"
	errCodeInvalidParameter = "InvalidParameterException"
)

// Client wraps the AWS Rekognition client and provides collection management operations
type Client struct {
	rekognition *rekognition.Client
	config      Config
}

// NewClient creates a new Rekognition client with the provided configuration
// It uses the AWS default credential chain to authenticate
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	// Load AWS SDK config using default credential chain
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		rekognition: rekognition.NewFromConfig(awsCfg),
		config:      cfg,
	}, nil
}

// CreateCollection creates a new Rekognition collection for the specified tenant
// Returns ErrCollectionAlreadyExists if a collection with the same name already exists
func (c *Client) CreateCollection(ctx context.Context, tenantID string) error {
	collectionID := c.config.CollectionName(tenantID)

	input := &rekognition.CreateCollectionInput{
		CollectionId: aws.String(collectionID),
	}

	_, err := c.rekognition.CreateCollection(ctx, input)
	if err != nil {
		// Check if collection already exists
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case errCodeResourceExists:
				return fmt.Errorf("tenant %s: %w", tenantID, ErrCollectionAlreadyExists)
			case errCodeInvalidParameter:
				return fmt.Errorf("tenant %s: invalid collection parameters: %w", tenantID, err)
			case errCodeAccessDenied:
				return fmt.Errorf("tenant %s: %w", tenantID, ErrInvalidCredentials)
			}
		}
		return fmt.Errorf("failed to create collection for tenant %s: %w", tenantID, err)
	}

	return nil
}

// DeleteCollection deletes a Rekognition collection for the specified tenant
// Returns ErrCollectionNotFound if the collection does not exist
func (c *Client) DeleteCollection(ctx context.Context, tenantID string) error {
	collectionID := c.config.CollectionName(tenantID)

	input := &rekognition.DeleteCollectionInput{
		CollectionId: aws.String(collectionID),
	}

	_, err := c.rekognition.DeleteCollection(ctx, input)
	if err != nil {
		// Check if collection doesn't exist
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case errCodeResourceNotFound:
				return fmt.Errorf("tenant %s: %w", tenantID, ErrCollectionNotFound)
			case errCodeAccessDenied:
				return fmt.Errorf("tenant %s: %w", tenantID, ErrInvalidCredentials)
			}
		}
		return fmt.Errorf("failed to delete collection for tenant %s: %w", tenantID, err)
	}

	return nil
}

// ListCollections returns a list of all collection IDs in the configured AWS region
func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
	input := &rekognition.ListCollectionsInput{
		MaxResults: aws.Int32(100), // Fetch up to 100 collections at a time
	}

	var collections []string
	paginator := rekognition.NewListCollectionsPaginator(c.rekognition, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == errCodeAccessDenied {
				return nil, fmt.Errorf("list collections: %w", ErrInvalidCredentials)
			}
			return nil, fmt.Errorf("failed to list collections: %w", err)
		}

		collections = append(collections, output.CollectionIds...)
	}

	return collections, nil
}

// CollectionExists checks if a collection exists for the specified tenant
func (c *Client) CollectionExists(ctx context.Context, tenantID string) (bool, error) {
	collectionID := c.config.CollectionName(tenantID)

	input := &rekognition.DescribeCollectionInput{
		CollectionId: aws.String(collectionID),
	}

	_, err := c.rekognition.DescribeCollection(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case errCodeResourceNotFound:
				return false, nil
			case errCodeAccessDenied:
				return false, fmt.Errorf("tenant %s: %w", tenantID, ErrInvalidCredentials)
			}
		}
		return false, fmt.Errorf("failed to check collection for tenant %s: %w", tenantID, err)
	}

	return true, nil
}

// GetCollectionFaceCount returns the number of faces indexed in a tenant's collection
func (c *Client) GetCollectionFaceCount(ctx context.Context, tenantID string) (int64, error) {
	collectionID := c.config.CollectionName(tenantID)

	input := &rekognition.DescribeCollectionInput{
		CollectionId: aws.String(collectionID),
	}

	output, err := c.rekognition.DescribeCollection(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case errCodeResourceNotFound:
				return 0, fmt.Errorf("tenant %s: %w", tenantID, ErrCollectionNotFound)
			case errCodeAccessDenied:
				return 0, fmt.Errorf("tenant %s: %w", tenantID, ErrInvalidCredentials)
			}
		}
		return 0, fmt.Errorf("failed to describe collection for tenant %s: %w", tenantID, err)
	}

	return *output.FaceCount, nil
}

// EnsureCollection creates a collection if it doesn't exist, or does nothing if it already exists
func (c *Client) EnsureCollection(ctx context.Context, tenantID string) error {
	exists, err := c.CollectionExists(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	if err := c.CreateCollection(ctx, tenantID); err != nil {
		// Ignore if collection was created concurrently
		if errors.Is(err, ErrCollectionAlreadyExists) {
			return nil
		}
		return err
	}

	return nil
}

// ParseNoFaceError checks if an AWS error indicates no face was detected
func ParseNoFaceError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case errCodeInvalidParameter:
			// Check if the error message indicates no face detected
			if msg := apiErr.ErrorMessage(); msg != "" {
				return fmt.Errorf("%w: %s", ErrNoFaceDetected, msg)
			}
			return ErrNoFaceDetected
		}
	}

	return err
}

// ParseIndexFacesError interprets errors from IndexFaces operation
func ParseIndexFacesError(unindexedFaces []types.UnindexedFace) error {
	if len(unindexedFaces) == 0 {
		return nil
	}

	// Check the first unindexed face for the reason
	face := unindexedFaces[0]
	if len(face.Reasons) > 0 {
		switch face.Reasons[0] {
		case types.ReasonExceedsMaxFaces:
			return ErrMultipleFaces
		case types.ReasonExtremePose, types.ReasonLowBrightness,
			types.ReasonLowSharpness, types.ReasonLowConfidence,
			types.ReasonSmallBoundingBox, types.ReasonLowFaceQuality:
			return fmt.Errorf("%w: %s", ErrNoFaceDetected, face.Reasons[0])
		}
	}

	return ErrNoFaceDetected
}
