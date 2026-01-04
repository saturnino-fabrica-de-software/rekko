package usage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		name  string
		used  int
		quota int
		want  float64
	}{
		{
			name:  "0% used",
			used:  0,
			quota: 1000,
			want:  0.0,
		},
		{
			name:  "50% used",
			used:  500,
			quota: 1000,
			want:  50.0,
		},
		{
			name:  "100% used",
			used:  1000,
			quota: 1000,
			want:  100.0,
		},
		{
			name:  "150% used (over quota)",
			used:  1500,
			quota: 1000,
			want:  150.0,
		},
		{
			name:  "unlimited quota",
			used:  5000,
			quota: -1,
			want:  0.0,
		},
		{
			name:  "zero quota",
			used:  100,
			quota: 0,
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePercentage(tt.used, tt.quota)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateOverage(t *testing.T) {
	tests := []struct {
		name  string
		used  int
		quota int
		want  int
	}{
		{
			name:  "no overage",
			used:  500,
			quota: 1000,
			want:  0,
		},
		{
			name:  "exactly at quota",
			used:  1000,
			quota: 1000,
			want:  0,
		},
		{
			name:  "500 units over",
			used:  1500,
			quota: 1000,
			want:  500,
		},
		{
			name:  "unlimited quota",
			used:  5000,
			quota: -1,
			want:  0,
		},
		{
			name:  "zero quota",
			used:  100,
			quota: 0,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateOverage(tt.used, tt.quota)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_generateAlerts(t *testing.T) {
	tests := []struct {
		name    string
		summary *UsageSummary
		want    int
	}{
		{
			name: "no alerts - under 80%",
			summary: &UsageSummary{
				Registrations: UsageDetail{
					Used:       700,
					Quota:      1000,
					Percentage: 70.0,
				},
				Verifications: UsageDetail{
					Used:       300,
					Quota:      500,
					Percentage: 60.0,
				},
			},
			want: 0,
		},
		{
			name: "warning alert - 80-89%",
			summary: &UsageSummary{
				Registrations: UsageDetail{
					Used:       850,
					Quota:      1000,
					Percentage: 85.0,
				},
				Verifications: UsageDetail{
					Used:       300,
					Quota:      500,
					Percentage: 60.0,
				},
			},
			want: 1,
		},
		{
			name: "critical alert - 90-99%",
			summary: &UsageSummary{
				Registrations: UsageDetail{
					Used:       950,
					Quota:      1000,
					Percentage: 95.0,
				},
				Verifications: UsageDetail{
					Used:       300,
					Quota:      500,
					Percentage: 60.0,
				},
			},
			want: 1,
		},
		{
			name: "exceeded alert - 100%+",
			summary: &UsageSummary{
				Registrations: UsageDetail{
					Used:       1100,
					Quota:      1000,
					Percentage: 110.0,
				},
				Verifications: UsageDetail{
					Used:       550,
					Quota:      500,
					Percentage: 110.0,
				},
			},
			want: 2,
		},
		{
			name: "multiple alerts",
			summary: &UsageSummary{
				Registrations: UsageDetail{
					Used:       850,
					Quota:      1000,
					Percentage: 85.0,
				},
				Verifications: UsageDetail{
					Used:       460,
					Quota:      500,
					Percentage: 92.0,
				},
			},
			want: 2,
		},
		{
			name: "unlimited quota - no alerts",
			summary: &UsageSummary{
				Registrations: UsageDetail{
					Used:       5000,
					Quota:      1000,
					Percentage: 500.0,
				},
				Verifications: UsageDetail{
					Used:       10000,
					Quota:      -1,
					Percentage: 0.0,
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			alerts := s.generateAlerts(tt.summary)
			assert.Equal(t, tt.want, len(alerts))
		})
	}
}

func TestService_calculateSummary(t *testing.T) {
	tests := []struct {
		name   string
		plan   *Plan
		usage  *UsageRecord
		period string
	}{
		{
			name: "starter plan with normal usage",
			plan: &Plan{
				ID:                 "starter",
				Name:               "Starter",
				MonthlyPrice:       99.00,
				QuotaRegistrations: 1000,
				QuotaVerifications: 500,
				OveragePrice:       0.05,
			},
			usage: &UsageRecord{
				Registrations:  500,
				Verifications:  250,
				LivenessChecks: 100,
			},
			period: "2025-01",
		},
		{
			name: "starter plan with overage",
			plan: &Plan{
				ID:                 "starter",
				Name:               "Starter",
				MonthlyPrice:       99.00,
				QuotaRegistrations: 1000,
				QuotaVerifications: 500,
				OveragePrice:       0.05,
			},
			usage: &UsageRecord{
				Registrations:  1200,
				Verifications:  600,
				LivenessChecks: 200,
			},
			period: "2025-01",
		},
		{
			name: "enterprise plan unlimited verifications",
			plan: &Plan{
				ID:                 "enterprise",
				Name:               "Enterprise",
				MonthlyPrice:       1999.00,
				QuotaRegistrations: 50000,
				QuotaVerifications: -1,
				OveragePrice:       0.01,
			},
			usage: &UsageRecord{
				Registrations:  25000,
				Verifications:  100000,
				LivenessChecks: 50000,
			},
			period: "2025-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			summary := s.calculateSummary(tt.plan, tt.usage, tt.period)

			assert.Equal(t, tt.period, summary.Period)
			assert.Equal(t, tt.plan.ID, summary.Plan.ID)
			assert.Equal(t, tt.usage.Registrations, summary.Registrations.Used)
			assert.Equal(t, tt.usage.Verifications, summary.Verifications.Used)
			assert.Equal(t, tt.usage.LivenessChecks, summary.LivenessChecks.Used)

			expectedBaseFee := tt.plan.MonthlyPrice
			assert.Equal(t, expectedBaseFee, summary.Billing.BaseFee)

			regOverage := calculateOverage(tt.usage.Registrations, tt.plan.QuotaRegistrations)
			verOverage := calculateOverage(tt.usage.Verifications, tt.plan.QuotaVerifications)

			expectedRegOverageFee := float64(regOverage) * tt.plan.OveragePrice
			expectedVerOverageFee := float64(verOverage) * tt.plan.OveragePrice
			expectedTotal := expectedBaseFee + expectedRegOverageFee + expectedVerOverageFee

			assert.Equal(t, expectedRegOverageFee, summary.Billing.OverageBreakdown.Registrations)
			assert.Equal(t, expectedVerOverageFee, summary.Billing.OverageBreakdown.Verifications)
			assert.Equal(t, expectedTotal, summary.Billing.Total)
		})
	}
}
