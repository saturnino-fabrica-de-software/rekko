package usage

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	MonthlyPrice       float64   `json:"monthly_price"`
	QuotaRegistrations int       `json:"quota_registrations"`
	QuotaVerifications int       `json:"quota_verifications"`
	OveragePrice       float64   `json:"overage_price"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UsageRecord struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Date           time.Time `json:"date"`
	Registrations  int       `json:"registrations"`
	Verifications  int       `json:"verifications"`
	LivenessChecks int       `json:"liveness_checks"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UsageAlert struct {
	Type       string  `json:"type"`
	Percentage float64 `json:"percentage"`
	Message    string  `json:"message"`
}

type UsageSummary struct {
	Period         string         `json:"period"`
	Plan           Plan           `json:"plan"`
	Registrations  UsageDetail    `json:"registrations"`
	Verifications  UsageDetail    `json:"verifications"`
	LivenessChecks UsageDetail    `json:"liveness_checks"`
	Billing        BillingSummary `json:"billing"`
	Alerts         []UsageAlert   `json:"alerts,omitempty"`
}

type UsageDetail struct {
	Used       int     `json:"used"`
	Quota      int     `json:"quota"`
	Percentage float64 `json:"percentage"`
	Overage    int     `json:"overage"`
}

type BillingSummary struct {
	BaseFee          float64          `json:"base_fee"`
	OverageFee       float64          `json:"overage_fee"`
	Total            float64          `json:"total"`
	OverageBreakdown OverageBreakdown `json:"overage_breakdown"`
}

type OverageBreakdown struct {
	Registrations float64 `json:"registrations"`
	Verifications float64 `json:"verifications"`
}
