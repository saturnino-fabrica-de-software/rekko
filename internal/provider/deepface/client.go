package deepface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds the configuration for the DeepFace client
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	Model      string
	Detector   string
	RetryCount int
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		BaseURL:    "http://localhost:5005",
		Timeout:    30 * time.Second,
		Model:      "Facenet512",
		Detector:   "retinaface",
		RetryCount: 3,
	}
}

// Client is the HTTP client for DeepFace API
type Client struct {
	httpClient *http.Client
	config     Config
}

// NewClient creates a new DeepFace client
func NewClient(config Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

// Represent calls POST /represent to generate face embeddings
func (c *Client) Represent(ctx context.Context, imageBase64 string) (*RepresentResponse, error) {
	req := RepresentRequest{
		Img:      imageBase64,
		Model:    c.config.Model,
		Detector: c.config.Detector,
	}

	var resp RepresentResponse
	if err := c.doRequestWithRetry(ctx, "POST", "/represent", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Analyze calls POST /analyze to detect faces in image
func (c *Client) Analyze(ctx context.Context, imageBase64 string) (*AnalyzeResponse, error) {
	req := AnalyzeRequest{
		Img:      imageBase64,
		Actions:  []string{}, // empty = just detect face
		Detector: c.config.Detector,
	}

	var resp AnalyzeResponse
	if err := c.doRequestWithRetry(ctx, "POST", "/analyze", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// maxBackoff is the maximum backoff duration for retries
const maxBackoff = 30 * time.Second

// calculateBackoff calculates exponential backoff duration for a given attempt
// Returns 1s, 2s, 4s, 8s, etc. up to maxBackoff
func calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return time.Second
	}
	// Calculate 2^(attempt-1) seconds safely
	seconds := 1
	for i := 1; i < attempt && i < 6; i++ {
		seconds *= 2
	}
	return time.Duration(seconds) * time.Second
}

// doRequestWithRetry executes HTTP request with retry logic
func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, capped at maxBackoff
			backoff := calculateBackoff(attempt)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		lastErr = c.doRequest(ctx, method, path, body, result)
		if lastErr == nil {
			return nil
		}

		// Don't retry on context errors
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Don't retry on client errors (4xx) - only retry on server errors (5xx)
		if isClientError(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("%w: %v", ErrDeepFaceUnavailable, lastErr)
}

// isClientError checks if the error is a 4xx client error
func isClientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for status 4xx patterns
	for status := 400; status < 500; status++ {
		if strings.Contains(errStr, fmt.Sprintf("status %d", status)) {
			return true
		}
	}
	return false
}

// doRequest executes a single HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	url := c.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("deepface returned status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}
	}

	return nil
}
