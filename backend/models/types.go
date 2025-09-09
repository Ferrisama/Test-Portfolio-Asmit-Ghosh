package models

import "time"

// Trade represents a single trade in the journal.
// All timestamps are stored in UTC.
type Trade struct {
	ID         string     `json:"id"`
	Symbol     string     `json:"symbol"`
	Side       string     `json:"side"` // "long" or "short"
	EntryTime  time.Time  `json:"entry_time"`
	ExitTime   *time.Time `json:"exit_time,omitempty"`
	EntryPrice float64    `json:"entry_price"`
	ExitPrice  *float64   `json:"exit_price,omitempty"`
	Quantity   float64    `json:"quantity"`
	Fees       float64    `json:"fees"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AnalyticsSummary represents summary metrics calculated over a set of trades.
type AnalyticsSummary struct {
	WinRate      float64 `json:"winRate"`
	ProfitFactor float64 `json:"profitFactor"`
	MaxDD        float64 `json:"maxDD"`
	Sharpe       float64 `json:"sharpe"`
	Sortino      float64 `json:"sortino"`
	Expectancy   float64 `json:"expectancy"`
}

// ImportReport summarizes the result of an import operation.
type ImportReport struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// Query represents the filters from the UI.
type Query struct {
	Symbol    string     `json:"symbol"`
	Side      string     `json:"side"`
	StartTime *time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// EquityPoint represents a point in the equity curve for charting.
type EquityPoint struct {
	Time  string  `json:"t"`
	Value float64 `json:"v"`
}