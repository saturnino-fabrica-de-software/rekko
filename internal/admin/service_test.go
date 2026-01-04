package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsParams_Defaults(t *testing.T) {
	now := time.Now()
	params := MetricsParams{
		StartDate: now.AddDate(0, 0, -30),
		EndDate:   now,
		Interval:  "day",
		Limit:     100,
		Offset:    0,
	}

	assert.Equal(t, "day", params.Interval)
	assert.Equal(t, 100, params.Limit)
	assert.Equal(t, 0, params.Offset)
	assert.True(t, params.StartDate.Before(params.EndDate))
}

func TestFacesMetrics_Structure(t *testing.T) {
	metrics := FacesMetrics{
		TotalRegistered: 150,
		Active:          145,
		Timeline: []FacesTimeline{
			{Period: "2024-01-01", Registered: 50},
			{Period: "2024-01-02", Registered: 25},
		},
	}

	assert.Equal(t, int64(150), metrics.TotalRegistered)
	assert.Equal(t, int64(145), metrics.Active)
	assert.Len(t, metrics.Timeline, 2)
}

func TestOperationsMetrics_Structure(t *testing.T) {
	metrics := OperationsMetrics{
		TotalOperations: 500,
		ByType: map[string]int64{
			"verification": 500,
		},
		Timeline: []OperationsTimeline{
			{Period: "2024-01-01", Total: 100, Success: 95, Failure: 5},
		},
	}

	assert.Equal(t, int64(500), metrics.TotalOperations)
	assert.Equal(t, int64(500), metrics.ByType["verification"])
	assert.Len(t, metrics.Timeline, 1)
}

func TestRequestsMetrics_Structure(t *testing.T) {
	metrics := RequestsMetrics{
		TotalRequests: 650,
		ByEndpoint: map[string]int64{
			"/v1/faces":        150,
			"/v1/faces/verify": 500,
		},
		Timeline: []RequestsTimeline{
			{Period: "2024-01-01", Total: 110, FacesRegister: 10, FacesVerify: 100},
		},
	}

	assert.Equal(t, int64(650), metrics.TotalRequests)
	assert.Equal(t, int64(150), metrics.ByEndpoint["/v1/faces"])
	assert.Equal(t, int64(500), metrics.ByEndpoint["/v1/faces/verify"])
	assert.Len(t, metrics.Timeline, 1)
}

func TestMetricsResponse_Structure(t *testing.T) {
	response := MetricsResponse{
		Data: FacesMetrics{TotalRegistered: 100, Active: 100},
		Meta: ResponseMeta{
			TenantID:    "tenant-123",
			Period:      Period{Start: "2024-01-01", End: "2024-01-31"},
			GeneratedAt: time.Now(),
		},
		Pagination: &PaginationMeta{
			Total:  10,
			Limit:  100,
			Offset: 0,
		},
	}

	assert.NotNil(t, response.Data)
	assert.Equal(t, "tenant-123", response.Meta.TenantID)
	assert.NotNil(t, response.Pagination)
	assert.Equal(t, 10, response.Pagination.Total)
}
