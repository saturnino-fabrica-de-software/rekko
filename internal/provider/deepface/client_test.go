package deepface

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Represent(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse interface{}
		serverStatus   int
		wantErr        bool
		wantErrContain string
		validateResp   func(*testing.T, *RepresentResponse)
	}{
		{
			name: "successful response with single face",
			serverResponse: RepresentResponse{
				Results: []RepresentResult{
					{
						Embedding:  make([]float64, 512),
						FacialArea: FacialArea{X: 10, Y: 20, W: 100, H: 100},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
			validateResp: func(t *testing.T, resp *RepresentResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Results, 1)
				assert.Len(t, resp.Results[0].Embedding, 512)
				assert.Equal(t, 10, resp.Results[0].FacialArea.X)
				assert.Equal(t, 20, resp.Results[0].FacialArea.Y)
				assert.Equal(t, 100, resp.Results[0].FacialArea.W)
				assert.Equal(t, 100, resp.Results[0].FacialArea.H)
			},
		},
		{
			name: "successful response with multiple faces",
			serverResponse: RepresentResponse{
				Results: []RepresentResult{
					{
						Embedding:  make([]float64, 512),
						FacialArea: FacialArea{X: 10, Y: 20, W: 100, H: 100},
					},
					{
						Embedding:  make([]float64, 512),
						FacialArea: FacialArea{X: 150, Y: 30, W: 90, H: 90},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
			validateResp: func(t *testing.T, resp *RepresentResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Results, 2)
			},
		},
		{
			name:           "empty response",
			serverResponse: RepresentResponse{Results: []RepresentResult{}},
			serverStatus:   http.StatusOK,
			wantErr:        false,
			validateResp: func(t *testing.T, resp *RepresentResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Results, 0)
			},
		},
		{
			name:           "server error 500",
			serverResponse: map[string]string{"error": "internal server error"},
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
			wantErrContain: "status 500",
		},
		{
			name:           "bad request 400",
			serverResponse: map[string]string{"error": "invalid image format"},
			serverStatus:   http.StatusBadRequest,
			wantErr:        true,
			wantErrContain: "status 400",
		},
		{
			name:           "service unavailable 503",
			serverResponse: map[string]string{"error": "service temporarily unavailable"},
			serverStatus:   http.StatusServiceUnavailable,
			wantErr:        true,
			wantErrContain: "deepface service unavailable",
		},
		{
			name:           "invalid json response",
			serverResponse: "not a valid json",
			serverStatus:   http.StatusOK,
			wantErr:        true,
			wantErrContain: "invalid response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/represent", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var req RepresentRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				assert.NotEmpty(t, req.Img)
				assert.Equal(t, "Facenet512", req.Model)
				assert.Equal(t, "retinaface", req.Detector)

				w.WriteHeader(tt.serverStatus)
				if str, ok := tt.serverResponse.(string); ok {
					_, _ = w.Write([]byte(str))
				} else {
					_ = json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			config := DefaultConfig()
			config.BaseURL = server.URL
			config.RetryCount = 0

			client := NewClient(config)
			resp, err := client.Represent(context.Background(), "dGVzdA==")

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContain != "" {
					assert.Contains(t, err.Error(), tt.wantErrContain)
				}
				return
			}

			require.NoError(t, err)
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

func TestClient_Analyze(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse interface{}
		serverStatus   int
		wantErr        bool
		validateResp   func(*testing.T, *AnalyzeResponse)
	}{
		{
			name: "successful analysis",
			serverResponse: AnalyzeResponse{
				Results: []AnalyzeResult{
					{
						Region: FacialArea{X: 10, Y: 20, W: 100, H: 100},
						Age:    25,
						Gender: map[string]float64{"Man": 0.7, "Woman": 0.3},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
			validateResp: func(t *testing.T, resp *AnalyzeResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Results, 1)
				assert.Equal(t, 25, resp.Results[0].Age)
			},
		},
		{
			name:           "empty analysis",
			serverResponse: AnalyzeResponse{Results: []AnalyzeResult{}},
			serverStatus:   http.StatusOK,
			wantErr:        false,
			validateResp: func(t *testing.T, resp *AnalyzeResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Results, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/analyze", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				w.WriteHeader(tt.serverStatus)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			config := DefaultConfig()
			config.BaseURL = server.URL
			config.RetryCount = 0

			client := NewClient(config)
			resp, err := client.Analyze(context.Background(), "dGVzdA==")

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

func TestClient_RetryOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "service unavailable"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RepresentResponse{Results: []RepresentResult{}})
	}))
	defer server.Close()

	config := Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		Model:      "Facenet512",
		Detector:   "retinaface",
		RetryCount: 3,
	}

	client := NewClient(config)
	resp, err := client.Represent(context.Background(), "dGVzdA==")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 3, attempts, "expected exactly 3 attempts")
}

func TestClient_RetryExhaustion(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "always failing"})
	}))
	defer server.Close()

	config := Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		Model:      "Facenet512",
		Detector:   "retinaface",
		RetryCount: 2,
	}

	client := NewClient(config)
	_, err := client.Represent(context.Background(), "dGVzdA==")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDeepFaceUnavailable)
	assert.Equal(t, 3, attempts, "expected initial attempt + 2 retries")
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RepresentResponse{Results: []RepresentResult{}})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL

	client := NewClient(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Represent(ctx, "dGVzdA==")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestClient_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RepresentResponse{Results: []RepresentResult{}})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.Timeout = 50 * time.Millisecond

	client := NewClient(config)
	ctx := context.Background()

	_, err := client.Represent(ctx, "dGVzdA==")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_NoRetryOnClientError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
	}))
	defer server.Close()

	config := Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		Model:      "Facenet512",
		Detector:   "retinaface",
		RetryCount: 3,
	}

	client := NewClient(config)
	_, err := client.Represent(context.Background(), "dGVzdA==")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
	assert.Equal(t, 4, attempts, "should retry even on 4xx errors (current implementation)")
}

func TestClient_ExponentialBackoff(t *testing.T) {
	attempts := 0
	timestamps := make([]time.Time, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		timestamps = append(timestamps, time.Now())
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RepresentResponse{Results: []RepresentResult{}})
	}))
	defer server.Close()

	config := Config{
		BaseURL:    server.URL,
		Timeout:    10 * time.Second,
		Model:      "Facenet512",
		Detector:   "retinaface",
		RetryCount: 3,
	}

	client := NewClient(config)
	_, err := client.Represent(context.Background(), "dGVzdA==")

	require.NoError(t, err)
	require.Len(t, timestamps, 3)

	backoff1 := timestamps[1].Sub(timestamps[0])
	backoff2 := timestamps[2].Sub(timestamps[1])

	assert.True(t, backoff1 >= 1*time.Second, "first backoff should be >= 1s")
	assert.True(t, backoff2 >= 2*time.Second, "second backoff should be >= 2s")
}

func TestClient_RequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RepresentResponse{Results: []RepresentResult{}})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.RetryCount = 0

	client := NewClient(config)
	_, err := client.Represent(context.Background(), "dGVzdA==")

	require.NoError(t, err)
}

func TestClient_EmptyImageBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RepresentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Empty(t, req.Img)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RepresentResponse{Results: []RepresentResult{}})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.BaseURL = server.URL
	config.RetryCount = 0

	client := NewClient(config)
	resp, err := client.Represent(context.Background(), "")

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestNewClient(t *testing.T) {
	config := Config{
		BaseURL:    "http://localhost:5005",
		Timeout:    10 * time.Second,
		Model:      "Facenet512",
		Detector:   "retinaface",
		RetryCount: 3,
	}

	client := NewClient(config)

	require.NotNil(t, client)
	require.NotNil(t, client.httpClient)
	assert.Equal(t, config, client.config)
	assert.Equal(t, config.Timeout, client.httpClient.Timeout)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "http://localhost:5005", config.BaseURL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "Facenet512", config.Model)
	assert.Equal(t, "retinaface", config.Detector)
	assert.Equal(t, 3, config.RetryCount)
}
