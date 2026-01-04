package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/webhook"
)

const (
	EventQuotaWarning  = "quota.warning"
	EventQuotaCritical = "quota.critical"
	EventQuotaExceeded = "quota.exceeded"

	cacheKeyUsage = "usage:%s:%s"
)

type CacheService interface {
	Get(ctx context.Context, key string, value interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type WebhookService interface {
	GetWebhooksByEvent(ctx context.Context, tenantID uuid.UUID, eventType string) ([]*webhook.Webhook, error)
	Send(ctx context.Context, webhook *webhook.Webhook, event webhook.EventPayload) error
}

type Service struct {
	repo           *Repository
	webhookService WebhookService
	cache          CacheService
}

func NewService(repo *Repository, webhookService WebhookService, cache CacheService) *Service {
	return &Service{
		repo:           repo,
		webhookService: webhookService,
		cache:          cache,
	}
}

func (s *Service) GetCurrentUsage(ctx context.Context, tenantID uuid.UUID, planID string) (*UsageSummary, error) {
	now := time.Now().UTC()
	period := now.Format("2006-01")

	cacheKey := fmt.Sprintf(cacheKeyUsage, tenantID, period)
	var cached UsageSummary
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	return s.getUsageForPeriod(ctx, tenantID, planID, period, startDate, endDate)
}

func (s *Service) GetUsageForPeriod(ctx context.Context, tenantID uuid.UUID, planID, period string) (*UsageSummary, error) {
	parsedTime, err := time.Parse("2006-01", period)
	if err != nil {
		return nil, fmt.Errorf("invalid period format, use YYYY-MM: %w", err)
	}

	startDate := time.Date(parsedTime.Year(), parsedTime.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(parsedTime.Year(), parsedTime.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	return s.getUsageForPeriod(ctx, tenantID, planID, period, startDate, endDate)
}

func (s *Service) getUsageForPeriod(ctx context.Context, tenantID uuid.UUID, planID, period string, startDate, endDate time.Time) (*UsageSummary, error) {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: get plan: %w", tenantID, err)
	}

	usage, err := s.repo.AggregatePeriod(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: aggregate usage: %w", tenantID, err)
	}

	summary := s.calculateSummary(plan, usage, period)

	cacheKey := fmt.Sprintf(cacheKeyUsage, tenantID, period)
	_ = s.cache.Set(ctx, cacheKey, summary, 5*time.Minute)

	return summary, nil
}

func (s *Service) calculateSummary(plan *Plan, usage *UsageRecord, period string) *UsageSummary {
	summary := &UsageSummary{
		Period: period,
		Plan:   *plan,
		Registrations: UsageDetail{
			Used:  usage.Registrations,
			Quota: plan.QuotaRegistrations,
		},
		Verifications: UsageDetail{
			Used:  usage.Verifications,
			Quota: plan.QuotaVerifications,
		},
		LivenessChecks: UsageDetail{
			Used:  usage.LivenessChecks,
			Quota: 0,
		},
		Billing: BillingSummary{
			BaseFee: plan.MonthlyPrice,
		},
	}

	summary.Registrations.Percentage = calculatePercentage(usage.Registrations, plan.QuotaRegistrations)
	summary.Registrations.Overage = calculateOverage(usage.Registrations, plan.QuotaRegistrations)

	if plan.QuotaVerifications >= 0 {
		summary.Verifications.Percentage = calculatePercentage(usage.Verifications, plan.QuotaVerifications)
		summary.Verifications.Overage = calculateOverage(usage.Verifications, plan.QuotaVerifications)
	}

	summary.Billing.OverageBreakdown.Registrations = float64(summary.Registrations.Overage) * plan.OveragePrice
	summary.Billing.OverageBreakdown.Verifications = float64(summary.Verifications.Overage) * plan.OveragePrice
	summary.Billing.OverageFee = summary.Billing.OverageBreakdown.Registrations + summary.Billing.OverageBreakdown.Verifications
	summary.Billing.Total = summary.Billing.BaseFee + summary.Billing.OverageFee

	summary.Alerts = s.generateAlerts(summary)

	return summary
}

func (s *Service) CheckQuota(ctx context.Context, tenantID uuid.UUID, planID string) error {
	summary, err := s.GetCurrentUsage(ctx, tenantID, planID)
	if err != nil {
		return fmt.Errorf("tenant %s: check quota: %w", tenantID, err)
	}

	for _, alert := range summary.Alerts {
		if err := s.sendAlert(ctx, tenantID, alert, summary); err != nil {
			return fmt.Errorf("tenant %s: send alert: %w", tenantID, err)
		}
	}

	return nil
}

func (s *Service) sendAlert(ctx context.Context, tenantID uuid.UUID, alert UsageAlert, summary *UsageSummary) error {
	webhooks, err := s.webhookService.GetWebhooksByEvent(ctx, tenantID, alert.Type)
	if err != nil {
		return fmt.Errorf("get webhooks: %w", err)
	}

	for _, wh := range webhooks {
		event := webhook.EventPayload{
			Type:      alert.Type,
			TenantID:  tenantID,
			Timestamp: time.Now().UTC(),
			Data: map[string]interface{}{
				"alert":   alert,
				"summary": summary,
			},
		}

		if err := s.webhookService.Send(ctx, wh, event); err != nil {
			return fmt.Errorf("send webhook: %w", err)
		}
	}

	return nil
}

func (s *Service) generateAlerts(summary *UsageSummary) []UsageAlert {
	var alerts []UsageAlert

	checkAndAddAlert := func(detail UsageDetail, resourceType string) {
		if detail.Quota <= 0 {
			return
		}

		if detail.Percentage >= 100 {
			alerts = append(alerts, UsageAlert{
				Type:       EventQuotaExceeded,
				Percentage: detail.Percentage,
				Message:    fmt.Sprintf("%s quota exceeded: %d%% used (%d/%d)", resourceType, int(detail.Percentage), detail.Used, detail.Quota),
			})
		} else if detail.Percentage >= 90 {
			alerts = append(alerts, UsageAlert{
				Type:       EventQuotaCritical,
				Percentage: detail.Percentage,
				Message:    fmt.Sprintf("%s quota critical: %d%% used (%d/%d)", resourceType, int(detail.Percentage), detail.Used, detail.Quota),
			})
		} else if detail.Percentage >= 80 {
			alerts = append(alerts, UsageAlert{
				Type:       EventQuotaWarning,
				Percentage: detail.Percentage,
				Message:    fmt.Sprintf("%s quota warning: %d%% used (%d/%d)", resourceType, int(detail.Percentage), detail.Used, detail.Quota),
			})
		}
	}

	checkAndAddAlert(summary.Registrations, "Registrations")
	checkAndAddAlert(summary.Verifications, "Verifications")

	return alerts
}

func calculatePercentage(used, quota int) float64 {
	if quota <= 0 {
		return 0
	}
	return (float64(used) / float64(quota)) * 100
}

func calculateOverage(used, quota int) int {
	if quota <= 0 {
		return 0
	}
	overage := used - quota
	if overage < 0 {
		return 0
	}
	return overage
}
