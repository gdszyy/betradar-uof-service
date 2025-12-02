package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// MarketTabChipService handles tab and chip assignment for markets
type MarketTabChipService struct {
	db *sql.DB
}

// NewMarketTabChipService creates a new instance of MarketTabChipService
func NewMarketTabChipService(db *sql.DB) *MarketTabChipService {
	return &MarketTabChipService{
		db: db,
	}
}

// AssignTabChipToAllMarkets assigns tab_id and chip_id to all markets
func (s *MarketTabChipService) AssignTabChipToAllMarkets() error {
	query := `
		SELECT id, event_id, sr_market_id, market_type, market_name, specifiers, status, groups
		FROM markets
		WHERE tab_id IS NULL OR tab_id = ''
		ORDER BY event_id, id
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var marketID int
		var eventID, srMarketID, marketType, marketName, specifiers, status, groupsStr sql.NullString

		if err := rows.Scan(&marketID, &eventID, &srMarketID, &marketType, &marketName, &specifiers, &status, &groupsStr); err != nil {
			log.Printf("Error scanning market row: %v", err)
			continue
		}

		// Parse groups and specifiers
		groups := s.parseGroups(groupsStr.String)
		specs := s.parseSpecifiers(specifiers.String)

		// Determine tab_id based on groups and specifiers
		tabID, err := s.determineTabID(groups, specs, marketType.String)
		if err != nil {
			log.Printf("Warning: failed to determine tab for market %d: %v", marketID, err)
			continue
		}

		// Determine chip_id based on tab and specifiers
		chipID := s.determineChipID(tabID, specs)

		// Update market with tab_id and chip_id
		updateQuery := `
			UPDATE markets 
			SET tab_id = $1, chip_id = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`

		_, err = s.db.Exec(updateQuery, tabID, chipID, marketID)
		if err != nil {
			log.Printf("Error updating market %d: %v", marketID, err)
			continue
		}

		count++
		if count%100 == 0 {
			log.Printf("Processed %d markets", count)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("✓ Assigned tab/chip to %d markets", count)
	return nil
}

// determineTabID determines the tab for a market based on groups and specifiers
func (s *MarketTabChipService) determineTabID(groups []string, specifiers []SpecifierPair, marketType string) (string, error) {
	// Check groups first (group-based tabs)
	groupTabMap := map[string]string{
		"regular_play": "regular_play",
		"player_props": "player_props",
		"micro_market": "micro_market",
		"bookings":     "bookings",
		"corners":      "corners",
		"1st_half":     "1st_half",
		"combo":        "combo",
		"2nd_half":     "2nd_half",
		"scorers":      "scorers",
	}

	for _, group := range groups {
		if tabID, ok := groupTabMap[group]; ok {
			return tabID, nil
		}
	}

	// Check specifiers (specifier-based tabs)
	specifierTabMap := map[string]string{
		"inningnr":    "innings",
		"setnr":       "sets",
		"mapnr":       "maps",
		"quarternr":   "quarters",
		"periodnr":    "periods",
		"framenr":     "frames",
		"overnr":      "overs",
		"drivenr":     "drives",
	}

	for _, spec := range specifiers {
		if tabID, ok := specifierTabMap[spec.Name]; ok {
			return tabID, nil
		}
	}

	// Default to regular_play if no match found
	return "regular_play", fmt.Errorf("no matching tab found for groups=%v, specifiers=%v", groups, specifiers)
}

// determineChipID determines the chip for a market based on tab and specifiers
func (s *MarketTabChipService) determineChipID(tabID string, specifiers []SpecifierPair) string {
	// Map of tab to primary specifier
	primarySpecifierMap := map[string]string{
		"innings": "inningnr",
		"sets":    "setnr",
		"maps":    "mapnr",
		"quarters": "quarternr",
		"periods": "periodnr",
		"frames":  "framenr",
		"overs":   "overnr",
		"drives":  "drivenr",
		"1st_half": "goalnr",
		"2nd_half": "goalnr",
		"corners":  "cornernr",
	}

	primarySpec, ok := primarySpecifierMap[tabID]
	if !ok {
		// No chip for this tab
		return ""
	}

	// Find the specifier value
	for _, spec := range specifiers {
		if spec.Name == primarySpec {
			return fmt.Sprintf("%s_%s_%s", tabID, spec.Name, spec.Value)
		}
	}

	return ""
}

// parseGroups parses groups string into array
func (s *MarketTabChipService) parseGroups(groupsStr string) []string {
	if groupsStr == "" {
		return []string{}
	}

	// Try to parse as JSON array first
	var groups []string
	if err := json.Unmarshal([]byte(groupsStr), &groups); err == nil {
		return groups
	}

	// Fall back to comma-separated parsing
	parts := strings.Split(groupsStr, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// SpecifierPair represents a specifier key-value pair
type SpecifierPair struct {
	Name  string
	Value string
}

// parseSpecifiers parses specifiers string into key-value pairs
func (s *MarketTabChipService) parseSpecifiers(specifiersStr string) []SpecifierPair {
	if specifiersStr == "" {
		return []SpecifierPair{}
	}

	var specs []SpecifierPair
	parts := strings.Split(specifiersStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			specs = append(specs, SpecifierPair{
				Name:  strings.TrimSpace(kv[0]),
				Value: strings.TrimSpace(kv[1]),
			})
		}
	}

	return specs
}

// GetTabsForEvent retrieves all tabs with markets for a specific event
func (s *MarketTabChipService) GetTabsForEvent(eventID string) ([]map[string]interface{}, error) {
	query := `
		SELECT DISTINCT tab_id, COUNT(*) as market_count
		FROM markets
		WHERE event_id = $1 AND tab_id IS NOT NULL AND tab_id != ''
		GROUP BY tab_id
		ORDER BY tab_id
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tabs: %w", err)
	}
	defer rows.Close()

	var tabs []map[string]interface{}
	for rows.Next() {
		var tabID string
		var marketCount int

		if err := rows.Scan(&tabID, &marketCount); err != nil {
			return nil, fmt.Errorf("failed to scan tab: %w", err)
		}

		tabs = append(tabs, map[string]interface{}{
			"tab_id":       tabID,
			"market_count": marketCount,
		})
	}

	return tabs, nil
}

// GetMarketsByTabChip retrieves markets for a specific tab and chip
func (s *MarketTabChipService) GetMarketsByTabChip(eventID string, tabID string, chipID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, event_id, market_type, market_name, tab_id, chip_id, status
		FROM markets
		WHERE event_id = $1 AND tab_id = $2
	`
	args := []interface{}{eventID, tabID}

	if chipID != "" {
		query += " AND chip_id = $3"
		args = append(args, chipID)
	}

	query += " ORDER BY id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	var markets []map[string]interface{}
	for rows.Next() {
		var id int
		var eventID, marketType, marketName, tabID, chipID, status string

		if err := rows.Scan(&id, &eventID, &marketType, &marketName, &tabID, &chipID, &status); err != nil {
			return nil, fmt.Errorf("failed to scan market: %w", err)
		}

		markets = append(markets, map[string]interface{}{
			"id":            id,
			"event_id":      eventID,
			"market_type":   marketType,
			"market_name":   marketName,
			"tab_id":        tabID,
			"chip_id":       chipID,
			"status":        status,
		})
	}

	return markets, nil
}

// GetChipsForTab retrieves all chips for a specific tab
func (s *MarketTabChipService) GetChipsForTab(tabID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, label, specifier, value, display_order
		FROM market_chips
		WHERE tab_id = $1
		ORDER BY display_order, id
	`

	rows, err := s.db.Query(query, tabID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chips: %w", err)
	}
	defer rows.Close()

	var chips []map[string]interface{}
	for rows.Next() {
		var id, label string
		var specifier, value sql.NullString
		var displayOrder int

		if err := rows.Scan(&id, &label, &specifier, &value, &displayOrder); err != nil {
			return nil, fmt.Errorf("failed to scan chip: %w", err)
		}

		chips = append(chips, map[string]interface{}{
			"id":              id,
			"label":           label,
			"specifier":       specifier.String,
			"value":           value.String,
			"display_order":   displayOrder,
		})
	}

	return chips, nil
}
