package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MetricsGetter interface {
	GetMetricValue(ctx context.Context, tenantID uuid.UUID, metricName, aggregation string, windowStart, windowEnd time.Time) (float64, error)
}

type Engine struct {
	metrics MetricsGetter
}

func NewEngine(metrics MetricsGetter) *Engine {
	return &Engine{metrics: metrics}
}

func (e *Engine) Evaluate(ctx context.Context, alert *Alert) (bool, map[string]interface{}, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(alert.WindowSeconds) * time.Second)

	results := make(map[string]interface{})
	conditionsMet := make([]bool, len(alert.Conditions))

	for i, cond := range alert.Conditions {
		value, err := e.metrics.GetMetricValue(ctx, alert.TenantID, cond.MetricName, cond.Aggregation, windowStart, now)
		if err != nil {
			return false, nil, fmt.Errorf("get metric %s: %w", cond.MetricName, err)
		}

		met := e.evaluateCondition(cond.Operator, value, cond.Threshold)
		conditionsMet[i] = met

		results[cond.MetricName] = map[string]interface{}{
			"value":       value,
			"threshold":   cond.Threshold,
			"operator":    cond.Operator,
			"met":         met,
			"aggregation": cond.Aggregation,
		}
	}

	var triggered bool
	if alert.ConditionLogic == "OR" {
		for _, met := range conditionsMet {
			if met {
				triggered = true
				break
			}
		}
	} else {
		triggered = true
		for _, met := range conditionsMet {
			if !met {
				triggered = false
				break
			}
		}
	}

	results["triggered"] = triggered
	results["window_start"] = windowStart
	results["window_end"] = now

	return triggered, results, nil
}

func (e *Engine) evaluateCondition(operator string, value, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	default:
		return false
	}
}

func (e *Engine) ShouldTrigger(alert *Alert, now time.Time) bool {
	if alert.LastTriggeredAt == nil {
		return true
	}

	cooldownDuration := time.Duration(alert.CooldownSeconds) * time.Second
	nextTriggerTime := alert.LastTriggeredAt.Add(cooldownDuration)

	return now.After(nextTriggerTime)
}
