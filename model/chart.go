package model

import "context"

type ChartEntry struct {
	PlayCount    int       `json:"playCount"`
	PreviousRank *int      `json:"previousRank"`
	Rank         int       `json:"rank"`
	Song         MediaFile `json:"song"`
}

type ChartScope string

const (
	ChartScopeCommunity ChartScope = "community"
	ChartScopePersonal  ChartScope = "personal"
)

// ChartOptions carries the validated query parameters for a chart request.
// Exactly one of Month or Week must be set (Week == 0 means "month period",
// Month == 0 means "week period" — enforced by parseChartOptions).
type ChartOptions struct {
	Limit  int
	Scope  ChartScope
	UserID string // only used when Scope == ChartScopePersonal

	Year  int
	Month int // 1-12; set for a month-period request
	Week  int // 1-53 (ISO week); set for a week-period request
}

type ChartRepository interface {
	GetChart(ctx context.Context, options ChartOptions) ([]ChartEntry, error)
}
