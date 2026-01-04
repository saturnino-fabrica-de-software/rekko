package alert

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockMetricsGetter struct {
	values map[string]float64
}

func (m *mockMetricsGetter) GetMetricValue(ctx context.Context, tenantID uuid.UUID, metricName, aggregation string, start, end time.Time) (float64, error) {
	val, ok := m.values[metricName]
	if !ok {
		return 0, nil
	}
	return val, nil
}

func TestEngine_Evaluate(t *testing.T) {
	tests := []struct {
		name          string
		alert         *Alert
		metricValues  map[string]float64
		wantTriggered bool
	}{
		{
			name: "single condition met",
			alert: &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "cpu_usage", Aggregation: "avg", Operator: "gt", Threshold: 80},
				},
				ConditionLogic: "AND",
				WindowSeconds:  300,
			},
			metricValues:  map[string]float64{"cpu_usage": 85},
			wantTriggered: true,
		},
		{
			name: "single condition not met",
			alert: &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "cpu_usage", Aggregation: "avg", Operator: "gt", Threshold: 80},
				},
				ConditionLogic: "AND",
				WindowSeconds:  300,
			},
			metricValues:  map[string]float64{"cpu_usage": 75},
			wantTriggered: false,
		},
		{
			name: "multiple conditions AND all met",
			alert: &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "cpu_usage", Aggregation: "avg", Operator: "gt", Threshold: 80},
					{MetricName: "memory_usage", Aggregation: "avg", Operator: "gt", Threshold: 70},
				},
				ConditionLogic: "AND",
				WindowSeconds:  300,
			},
			metricValues:  map[string]float64{"cpu_usage": 85, "memory_usage": 75},
			wantTriggered: true,
		},
		{
			name: "multiple conditions AND one not met",
			alert: &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "cpu_usage", Aggregation: "avg", Operator: "gt", Threshold: 80},
					{MetricName: "memory_usage", Aggregation: "avg", Operator: "gt", Threshold: 70},
				},
				ConditionLogic: "AND",
				WindowSeconds:  300,
			},
			metricValues:  map[string]float64{"cpu_usage": 85, "memory_usage": 65},
			wantTriggered: false,
		},
		{
			name: "multiple conditions OR one met",
			alert: &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "cpu_usage", Aggregation: "avg", Operator: "gt", Threshold: 80},
					{MetricName: "memory_usage", Aggregation: "avg", Operator: "gt", Threshold: 70},
				},
				ConditionLogic: "OR",
				WindowSeconds:  300,
			},
			metricValues:  map[string]float64{"cpu_usage": 85, "memory_usage": 65},
			wantTriggered: true,
		},
		{
			name: "multiple conditions OR none met",
			alert: &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "cpu_usage", Aggregation: "avg", Operator: "gt", Threshold: 80},
					{MetricName: "memory_usage", Aggregation: "avg", Operator: "gt", Threshold: 70},
				},
				ConditionLogic: "OR",
				WindowSeconds:  300,
			},
			metricValues:  map[string]float64{"cpu_usage": 75, "memory_usage": 65},
			wantTriggered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetrics := &mockMetricsGetter{values: tt.metricValues}
			engine := NewEngine(mockMetrics)

			triggered, metadata, err := engine.Evaluate(context.Background(), tt.alert)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if triggered != tt.wantTriggered {
				t.Errorf("Evaluate() triggered = %v, want %v", triggered, tt.wantTriggered)
			}
			if metadata == nil {
				t.Error("Evaluate() metadata should not be nil")
			}
		})
	}
}

func TestEvaluateCondition_ThroughEngine(t *testing.T) {
	tests := []struct {
		name          string
		operator      string
		value         float64
		threshold     float64
		wantTriggered bool
	}{
		{"greater than true", "gt", 100, 80, true},
		{"greater than false", "gt", 80, 100, false},
		{"greater than equal true equal", "gte", 100, 100, true},
		{"greater than equal true greater", "gte", 100, 80, true},
		{"greater than equal false", "gte", 80, 100, false},
		{"less than true", "lt", 80, 100, true},
		{"less than false", "lt", 100, 80, false},
		{"less than equal true equal", "lte", 100, 100, true},
		{"less than equal true less", "lte", 80, 100, true},
		{"less than equal false", "lte", 100, 80, false},
		{"equal true", "eq", 100, 100, true},
		{"equal false", "eq", 100, 80, false},
		{"unknown operator", "unknown", 100, 80, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetrics := &mockMetricsGetter{
				values: map[string]float64{"test_metric": tt.value},
			}
			engine := NewEngine(mockMetrics)

			alert := &Alert{
				TenantID: uuid.New(),
				Conditions: []Condition{
					{MetricName: "test_metric", Aggregation: "avg", Operator: tt.operator, Threshold: tt.threshold},
				},
				ConditionLogic: "AND",
				WindowSeconds:  300,
			}

			triggered, _, err := engine.Evaluate(context.Background(), alert)
			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}
			if triggered != tt.wantTriggered {
				t.Errorf("Evaluate() triggered = %v, want %v", triggered, tt.wantTriggered)
			}
		})
	}
}
