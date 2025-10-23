package models

import "time"

// Odds 统一的赔率模型
type Odds struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"match_id"`
	MarketID    string    `json:"market_id"`
	MarketName  string    `json:"market_name"`
	Outcomes    []Outcome `json:"outcomes"`
	Status      string    `json:"status"` // active, suspended, deactivated
	Source      string    `json:"source"` // uof
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Outcome 赔率结果
type Outcome struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Odds        float64 `json:"odds"`
	Probability float64 `json:"probability,omitempty"`
	Active      bool    `json:"active"`
}

// Market 市场信息
type Market struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Specifiers  map[string]string `json:"specifiers,omitempty"`
}

// OddsStatus 赔率状态
type OddsStatus string

const (
	OddsStatusActive      OddsStatus = "active"
	OddsStatusSuspended   OddsStatus = "suspended"
	OddsStatusDeactivated OddsStatus = "deactivated"
)

