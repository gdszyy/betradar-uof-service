package database

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// MarketTab represents a tab configuration for market card display
type MarketTab struct {
	ID                string    `db:"id"`
	Label             string    `db:"label"`
	Type              string    `db:"type"` // 'group' or 'specifier_aggregate'
	MarketCount       int       `db:"market_count"`
	ChipSpecifiers    string    `db:"chip_specifiers"` // comma-separated specifier names
	GroupName         *string   `db:"group_name"`      // for group-based tabs
	PrimarySpecifier  *string   `db:"primary_specifier"` // for specifier_aggregate tabs
	DisplayOrder      int       `db:"display_order"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// MarketChip represents a chip configuration for market card display
type MarketChip struct {
	ID            string    `db:"id"`
	TabID         string    `db:"tab_id"`
	Specifier     *string   `db:"specifier"`
	Value         *string   `db:"value"`
	Label         string    `db:"label"`
	DisplayOrder  int       `db:"display_order"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// MarketTabChipMapping represents the relationship between market, tab, and chip
type MarketTabChipMapping struct {
	ID              int       `db:"id"`
	MarketID        int       `db:"market_id"`
	EventID         string    `db:"event_id"`
	TabID           string    `db:"tab_id"`
	ChipID          *string   `db:"chip_id"`
	SpecifierName   *string   `db:"specifier_name"`
	SpecifierValue  *string   `db:"specifier_value"`
	CreatedAt       time.Time `db:"created_at"`
}

// MarketGroupsCache represents cached market groups
type MarketGroupsCache struct {
	ID        int       `db:"id"`
	MarketID  int       `db:"market_id"`
	Groups    string    `db:"groups"` // JSON array
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// GroupsArray represents a JSON array of group names
type GroupsArray []string

// Value implements the driver.Valuer interface
func (g GroupsArray) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// Scan implements the sql.Scanner interface
func (g *GroupsArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &g)
}

// MarketSpecifiersCache represents cached parsed specifiers
type MarketSpecifiersCache struct {
	ID               int       `db:"id"`
	MarketID         int       `db:"market_id"`
	SpecifiersJSON   string    `db:"specifiers_json"` // JSONB
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// SpecifierPair represents a key-value pair of specifier
type SpecifierPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SpecifiersArray represents an array of specifiers
type SpecifiersArray []SpecifierPair

// Value implements the driver.Valuer interface for JSONB
func (s SpecifiersArray) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for JSONB
func (s *SpecifiersArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &s)
}

// MarketTabChipView represents the combined view of market with tab and chip info
type MarketTabChipView struct {
	MarketID        int       `db:"market_id"`
	EventID         string    `db:"event_id"`
	SRMarketID      string    `db:"sr_market_id"`
	MarketType      string    `db:"market_type"`
	MarketName      string    `db:"market_name"`
	Groups          string    `db:"groups"`
	Specifiers      string    `db:"specifiers"`
	Status          string    `db:"status"`
	TabID           *string   `db:"tab_id"`
	ChipID          *string   `db:"chip_id"`
	TabLabel        *string   `db:"tab_label"`
	TabType         *string   `db:"tab_type"`
	ChipLabel       *string   `db:"chip_label"`
	ChipSpecifier   *string   `db:"chip_specifier"`
	ChipValue       *string   `db:"chip_value"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// TabChipConfig represents the complete configuration for a tab with its chips
type TabChipConfig struct {
	Tab   *MarketTab   `json:"tab"`
	Chips []*MarketChip `json:"chips"`
}

// MarketCardData represents the data structure for market card display
type MarketCardData struct {
	EventID    string                  `json:"event_id"`
	Tabs       []*MarketTab            `json:"tabs"`
	Markets    map[string][]*MarketTabChipView `json:"markets"` // key: tab_id
	ChipsByTab map[string][]*MarketChip `json:"chips_by_tab"` // key: tab_id
}
