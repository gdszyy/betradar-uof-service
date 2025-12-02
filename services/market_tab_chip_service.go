package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"betradar-uof-service/database"
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

// AssignTabChipToMarket assigns tab_id and chip_id to a market based on its properties
func (s *MarketTabChipService) AssignTabChipToMarket(market *database.Market) error {
	if market == nil {
		return fmt.Errorf("market cannot be nil")
	}

	// Parse groups from market description cache
	groups, err := s.getMarketGroups(market.ID)
	if err != nil {
		log.Printf("Warning: failed to get market groups for market %d: %v", market.ID, err)
		groups = []string{}
	}

	// Parse specifiers
	specifiers := s.parseSpecifiers(market.Specifiers)

	// Determine tab_id based on groups and specifiers
	tabID, err := s.determineTabID(groups, specifiers, market.MarketType)
	if err != nil {
		log.Printf("Warning: failed to determine tab for market %d: %v", market.ID, err)
		return nil // Don't fail, just skip tab assignment
	}

	// Determine chip_id based on tab and specifiers
	chipID := s.determineChipID(tabID, specifiers)

	// Update market with tab_id and chip_id
	query := `
		UPDATE markets 
		SET tab_id = $1, chip_id = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	_, err = s.db.Exec(query, tabID, chipID, market.ID)
	if err != nil {
		return fmt.Errorf("failed to update market %d with tab/chip: %w", market.ID, err)
	}

	// Record the mapping
	if err := s.recordTabChipMapping(market.ID, market.EventID, tabID, chipID, specifiers); err != nil {
		log.Printf("Warning: failed to record tab/chip mapping for market %d: %v", market.ID, err)
	}

	return nil
}

// AssignTabChipToAllMarkets assigns tab_id and chip_id to all markets
func (s *MarketTabChipService) AssignTabChipToAllMarkets() error {
	query := `
		SELECT id, event_id, sr_market_id, market_type, market_name, specifiers, status
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
		var market database.Market
		if err := rows.Scan(&market.ID, &market.EventID, &market.SRMarketID, &market.MarketType, &market.MarketName, &market.Specifiers, &market.Status); err != nil {
			log.Printf("Error scanning market row: %v", err)
			continue
		}

		if err := s.AssignTabChipToMarket(&market); err != nil {
			log.Printf("Error assigning tab/chip to market %d: %v", market.ID, err)
			continue
		}

		count++
		if count%100 == 0 {
			log.Printf("Processed %d markets", count)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating market rows: %w", err)
	}

	log.Printf("Successfully assigned tab/chip to %d markets", count)
	return nil
}

// determineTabID determines the tab_id for a market based on its properties
func (s *MarketTabChipService) determineTabID(groups []string, specifiers []database.SpecifierPair, marketType string) (string, error) {
	// Check if market belongs to a group-based tab
	groupToTabMap := map[string]string{
		"regular_play":  "regular_play",
		"player_props":  "player_props",
		"micro_market":  "micro_market",
		"bookings":      "bookings",
		"corners":       "corners",
		"1st_half":      "1st_half",
		"combo":         "combo",
		"2nd_half":      "2nd_half",
		"scorers":       "scorers",
	}

	for _, group := range groups {
		if tabID, exists := groupToTabMap[group]; exists {
			return tabID, nil
		}
	}

	// Check if market belongs to a specifier-aggregate tab
	specifierToTabMap := map[string]string{
		"inningnr":   "innings",
		"setnr":      "sets",
		"mapnr":      "maps",
		"quarternr":  "quarters",
		"periodnr":   "periods",
		"framenr":    "frames",
		"overnr":     "overs",
		"drivenr":    "drives",
		"cornernr":   "corners",
		"goalnr":     "1st_half", // goalnr can be in 1st_half, 2nd_half, or scorers
	}

	for _, spec := range specifiers {
		if tabID, exists := specifierToTabMap[spec.Name]; exists {
			// Special handling for goalnr: determine if it's 1st_half, 2nd_half, or scorers
			if spec.Name == "goalnr" {
				// Check if market has other specifiers that indicate half
				for _, s := range specifiers {
					if s.Name == "period" && s.Value == "1" {
						return "1st_half", nil
					}
					if s.Name == "period" && s.Value == "2" {
						return "2nd_half", nil
					}
				}
				// Default to 1st_half if no period specifier
				return "1st_half", nil
			}
			return tabID, nil
		}
	}

	// Default to regular_play if no specific tab found
	return "regular_play", nil
}

// determineChipID determines the chip_id for a market based on tab and specifiers
func (s *MarketTabChipService) determineChipID(tabID string, specifiers []database.SpecifierPair) string {
	// Map of tab to primary specifier
	tabToPrimarySpecifier := map[string]string{
		"innings":      "inningnr",
		"sets":         "setnr",
		"maps":         "mapnr",
		"quarters":     "quarternr",
		"periods":      "periodnr",
		"frames":       "framenr",
		"overs":        "overnr",
		"drives":       "drivenr",
		"corners":      "cornernr",
		"1st_half":     "goalnr",
		"2nd_half":     "goalnr",
		"scorers":      "goalnr",
		"player_props": "count",
		"micro_market": "pitchnr",
	}

	primarySpec, exists := tabToPrimarySpecifier[tabID]
	if !exists {
		return "" // No chip for this tab
	}

	// Find the primary specifier value
	for _, spec := range specifiers {
		if spec.Name == primarySpec {
			// Generate chip ID: tab_id_specifier_value
			chipID := fmt.Sprintf("%s_%s_%s", tabID, primarySpec, spec.Value)
			return chipID
		}
	}

	return "" // No chip if primary specifier not found
}

// parseSpecifiers parses the specifiers string into a structured format
func (s *MarketTabChipService) parseSpecifiers(specifiersStr string) []database.SpecifierPair {
	var specifiers []database.SpecifierPair

	if specifiersStr == "" {
		return specifiers
	}

	// Parse specifiers in format "key1=value1,key2=value2"
	parts := strings.Split(specifiersStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			specifiers = append(specifiers, database.SpecifierPair{
				Name:  strings.TrimSpace(kv[0]),
				Value: strings.TrimSpace(kv[1]),
			})
		}
	}

	return specifiers
}

// getMarketGroups retrieves the groups for a market from the cache
func (s *MarketTabChipService) getMarketGroups(marketID int) ([]string, error) {
	var groupsJSON string

	query := `
		SELECT groups FROM market_groups_cache WHERE market_id = $1
	`

	err := s.db.QueryRow(query, marketID).Scan(&groupsJSON)
	if err == sql.ErrNoRows {
		// Try to get from market description cache
		return s.getGroupsFromMarketDescription(marketID)
	}
	if err != nil {
		return []string{}, err
	}

	var groups []string
	if err := json.Unmarshal([]byte(groupsJSON), &groups); err != nil {
		return []string{}, err
	}

	return groups, nil
}

// getGroupsFromMarketDescription retrieves groups from market description
func (s *MarketTabChipService) getGroupsFromMarketDescription(marketID int) ([]string, error) {
	// This would need to query from market_descriptions_cache table
	// For now, return empty slice
	return []string{}, nil
}

// recordTabChipMapping records the tab/chip assignment in the mapping table
func (s *MarketTabChipService) recordTabChipMapping(marketID int, eventID string, tabID string, chipID string, specifiers []database.SpecifierPair) error {
	query := `
		INSERT INTO market_tab_chip_mapping (market_id, event_id, tab_id, chip_id, specifier_name, specifier_value, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (market_id, tab_id, chip_id) DO NOTHING
	`

	// Record primary specifier used for chip determination
	for _, spec := range specifiers {
		_, err := s.db.Exec(query, marketID, eventID, tabID, chipID, spec.Name, spec.Value)
		if err != nil {
			return fmt.Errorf("failed to insert mapping: %w", err)
		}
	}

	return nil
}

// GetMarketsByTabChip retrieves markets for a specific tab and chip
func (s *MarketTabChipService) GetMarketsByTabChip(eventID string, tabID string, chipID string) ([]*database.MarketTabChipView, error) {
	var query string
	var args []interface{}

	if chipID == "" || chipID == "all" {
		query = `
			SELECT 
				m.id, m.event_id, m.sr_market_id, m.market_type, m.market_name, m.specifiers, m.status,
				m.tab_id, m.chip_id,
				mt.label, mt.type,
				mc.label, mc.specifier, mc.value,
				m.created_at, m.updated_at
			FROM markets m
			LEFT JOIN market_tabs mt ON m.tab_id = mt.id
			LEFT JOIN market_chips mc ON m.chip_id = mc.id
			WHERE m.event_id = $1 AND m.tab_id = $2
			ORDER BY m.id
		`
		args = []interface{}{eventID, tabID}
	} else {
		query = `
			SELECT 
				m.id, m.event_id, m.sr_market_id, m.market_type, m.market_name, m.specifiers, m.status,
				m.tab_id, m.chip_id,
				mt.label, mt.type,
				mc.label, mc.specifier, mc.value,
				m.created_at, m.updated_at
			FROM markets m
			LEFT JOIN market_tabs mt ON m.tab_id = mt.id
			LEFT JOIN market_chips mc ON m.chip_id = mc.id
			WHERE m.event_id = $1 AND m.tab_id = $2 AND m.chip_id = $3
			ORDER BY m.id
		`
		args = []interface{}{eventID, tabID, chipID}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	var markets []*database.MarketTabChipView
	for rows.Next() {
		var m database.MarketTabChipView
		if err := rows.Scan(&m.MarketID, &m.EventID, &m.SRMarketID, &m.MarketType, &m.MarketName, &m.Specifiers, &m.Status,
			&m.TabID, &m.ChipID,
			&m.TabLabel, &m.TabType,
			&m.ChipLabel, &m.ChipSpecifier, &m.ChipValue,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan market row: %w", err)
		}
		markets = append(markets, &m)
	}

	return markets, rows.Err()
}

// GetTabsForEvent retrieves all tabs available for an event
func (s *MarketTabChipService) GetTabsForEvent(eventID string) ([]*database.MarketTab, error) {
	query := `
		SELECT DISTINCT mt.id, mt.label, mt.type, COUNT(m.id) as market_count, 
			   mt.chip_specifiers, mt.group_name, mt.primary_specifier, mt.display_order,
			   mt.created_at, mt.updated_at
		FROM market_tabs mt
		LEFT JOIN markets m ON m.tab_id = mt.id AND m.event_id = $1
		GROUP BY mt.id, mt.label, mt.type, mt.chip_specifiers, mt.group_name, mt.primary_specifier, mt.display_order, mt.created_at, mt.updated_at
		HAVING COUNT(m.id) > 0
		ORDER BY mt.display_order
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tabs: %w", err)
	}
	defer rows.Close()

	var tabs []*database.MarketTab
	for rows.Next() {
		var t database.MarketTab
		if err := rows.Scan(&t.ID, &t.Label, &t.Type, &t.MarketCount, &t.ChipSpecifiers, &t.GroupName, &t.PrimarySpecifier, &t.DisplayOrder, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan tab row: %w", err)
		}
		tabs = append(tabs, &t)
	}

	return tabs, rows.Err()
}

// GetChipsForTab retrieves all chips for a specific tab
func (s *MarketTabChipService) GetChipsForTab(tabID string) ([]*database.MarketChip, error) {
	query := `
		SELECT id, tab_id, specifier, value, label, display_order, created_at, updated_at
		FROM market_chips
		WHERE tab_id = $1
		ORDER BY display_order
	`

	rows, err := s.db.Query(query, tabID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chips: %w", err)
	}
	defer rows.Close()

	var chips []*database.MarketChip
	for rows.Next() {
		var c database.MarketChip
		if err := rows.Scan(&c.ID, &c.TabID, &c.Specifier, &c.Value, &c.Label, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chip row: %w", err)
		}
		chips = append(chips, &c)
	}

	return chips, rows.Err()
}
