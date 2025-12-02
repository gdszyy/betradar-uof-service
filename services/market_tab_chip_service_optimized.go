package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// SpecifierPair represents a specifier key-value pair
type SpecifierPair struct {
	Name  string
	Value string
}

// MarketTabChipServiceOptimized provides optimized operations for market tab/chip assignment
// Key features:
// 1. Incremental updates - only processes markets without tab_id
// 2. Exception handling - tracks unmapped markets for maintenance
// 3. Assignment logging - maintains audit trail of all assignments
type MarketTabChipServiceOptimized struct {
	db *sql.DB
}

// NewMarketTabChipServiceOptimized creates a new optimized service instance
func NewMarketTabChipServiceOptimized(db *sql.DB) *MarketTabChipServiceOptimized {
	return &MarketTabChipServiceOptimized{db: db}
}

// AssignTabChipToNewMarkets assigns tab/chip only to markets that don't have tab_id set
// This is the incremental update approach - only processes new markets
func (s *MarketTabChipServiceOptimized) AssignTabChipToNewMarkets() error {
	log.Println("=== Starting Incremental Tab/Chip Assignment ===")

	// Get all markets without tab_id
	query := `
		SELECT id, event_id, market_type, market_name, groups, specifiers, tab_id, chip_id
		FROM markets
		WHERE tab_id IS NULL OR tab_id = ''
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	totalCount := 0
	successCount := 0
	failureCount := 0

	for rows.Next() {
		totalCount++
		var marketID int
		var eventID, marketType, marketName, groupsStr, specifiersStr string
		var oldTabID, oldChipID sql.NullString

		if err := rows.Scan(&marketID, &eventID, &marketType, &marketName, &groupsStr, &specifiersStr, &oldTabID, &oldChipID); err != nil {
			log.Printf("Error scanning market row: %v", err)
			failureCount++
			continue
		}

		// Parse groups and specifiers
		groups := s.parseGroups(groupsStr)
		specifiers := s.parseSpecifiers(specifiersStr)

		// Determine tab and chip
		tabID, err := s.determineTabID(groups, specifiers)
		if err != nil {
			// Log unmapped market
			if err := s.logUnmappedMarket(marketID, eventID, marketType, marketName, groupsStr, specifiersStr, err.Error()); err != nil {
				log.Printf("Error logging unmapped market %d: %v", marketID, err)
			}
			failureCount++
			log.Printf("Market %d: Failed to determine tab - %v", marketID, err)
			continue
		}

		chipID := s.determineChipID(tabID, specifiers)

		// Update market with tab_id and chip_id
		updateQuery := `
			UPDATE markets
			SET tab_id = $1, chip_id = $2, updated_at = NOW()
			WHERE id = $3
		`

		if _, err := s.db.Exec(updateQuery, tabID, chipID, marketID); err != nil {
			log.Printf("Error updating market %d: %v", marketID, err)
			failureCount++
			continue
		}

		// Log assignment
		if err := s.logAssignment(marketID, eventID, oldTabID.String, tabID, oldChipID.String, chipID, "initial", "import_tool"); err != nil {
			log.Printf("Error logging assignment for market %d: %v", marketID, err)
		}

		successCount++
		if successCount%100 == 0 {
			log.Printf("Processed %d markets, assigned %d successfully", totalCount, successCount)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("✓ Incremental assignment completed: %d total, %d success, %d failed", totalCount, successCount, failureCount)
	return nil
}

// AssignTabChipToAllMarkets performs a full assignment (for initial setup)
// This processes all markets, updating existing ones if needed
func (s *MarketTabChipServiceOptimized) AssignTabChipToAllMarkets() error {
	log.Println("=== Starting Full Tab/Chip Assignment ===")

	query := `
		SELECT id, event_id, market_type, market_name, groups, specifiers, tab_id, chip_id
		FROM markets
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	totalCount := 0
	successCount := 0
	failureCount := 0
	updateCount := 0

	for rows.Next() {
		totalCount++
		var marketID int
		var eventID, marketType, marketName, groupsStr, specifiersStr string
		var oldTabID, oldChipID sql.NullString

		if err := rows.Scan(&marketID, &eventID, &marketType, &marketName, &groupsStr, &specifiersStr, &oldTabID, &oldChipID); err != nil {
			log.Printf("Error scanning market row: %v", err)
			failureCount++
			continue
		}

		// Parse groups and specifiers
		groups := s.parseGroups(groupsStr)
		specifiers := s.parseSpecifiers(specifiersStr)

		// Determine tab and chip
		tabID, err := s.determineTabID(groups, specifiers)
		if err != nil {
			// Log unmapped market
			if err := s.logUnmappedMarket(marketID, eventID, marketType, marketName, groupsStr, specifiersStr, err.Error()); err != nil {
				log.Printf("Error logging unmapped market %d: %v", marketID, err)
			}
			failureCount++
			continue
		}

		chipID := s.determineChipID(tabID, specifiers)

		// Check if assignment changed
		assignmentType := "initial"
		if oldTabID.Valid && oldTabID.String != "" {
			assignmentType = "update"
			if oldTabID.String == tabID && oldChipID.String == chipID {
				// No change needed
				successCount++
				continue
			}
			updateCount++
		}

		// Update market
		updateQuery := `
			UPDATE markets
			SET tab_id = $1, chip_id = $2, updated_at = NOW()
			WHERE id = $3
		`

		if _, err := s.db.Exec(updateQuery, tabID, chipID, marketID); err != nil {
			log.Printf("Error updating market %d: %v", marketID, err)
			failureCount++
			continue
		}

		// Log assignment
		if err := s.logAssignment(marketID, eventID, oldTabID.String, tabID, oldChipID.String, chipID, assignmentType, "import_tool"); err != nil {
			log.Printf("Error logging assignment for market %d: %v", marketID, err)
		}

		successCount++
		if successCount%100 == 0 {
			log.Printf("Processed %d markets, assigned %d successfully, updated %d", totalCount, successCount, updateCount)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("✓ Full assignment completed: %d total, %d success, %d failed, %d updated", totalCount, successCount, failureCount, updateCount)
	return nil
}

// GetUnmappedMarkets retrieves all markets that couldn't be mapped
func (s *MarketTabChipServiceOptimized) GetUnmappedMarkets(limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, market_id, event_id, market_type, market_name, reason, created_at
		FROM market_tab_chip_unmapped
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unmapped markets: %w", err)
	}
	defer rows.Close()

	var unmapped []map[string]interface{}
	for rows.Next() {
		var id, marketID int
		var eventID, marketType, marketName, reason string
		var createdAt time.Time

		if err := rows.Scan(&id, &marketID, &eventID, &marketType, &marketName, &reason, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan unmapped market: %w", err)
		}

		unmapped = append(unmapped, map[string]interface{}{
			"id":            id,
			"market_id":     marketID,
			"event_id":      eventID,
			"market_type":   marketType,
			"market_name":   marketName,
			"reason":        reason,
			"created_at":    createdAt,
		})
	}

	return unmapped, nil
}

// GetUnmappedSummary retrieves summary of unmapped markets by reason
func (s *MarketTabChipServiceOptimized) GetUnmappedSummary() ([]map[string]interface{}, error) {
	query := `
		SELECT reason, COUNT(*) as count, COUNT(DISTINCT event_id) as event_count,
		       MIN(created_at) as first_occurrence, MAX(updated_at) as last_occurrence
		FROM market_tab_chip_unmapped
		GROUP BY reason
		ORDER BY count DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unmapped summary: %w", err)
	}
	defer rows.Close()

	var summary []map[string]interface{}
	for rows.Next() {
		var reason string
		var count, eventCount int
		var firstOccurrence, lastOccurrence time.Time

		if err := rows.Scan(&reason, &count, &eventCount, &firstOccurrence, &lastOccurrence); err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}

		summary = append(summary, map[string]interface{}{
			"reason":             reason,
			"count":              count,
			"event_count":        eventCount,
			"first_occurrence":   firstOccurrence,
			"last_occurrence":    lastOccurrence,
		})
	}

	return summary, nil
}

// logUnmappedMarket logs a market that couldn't be mapped
func (s *MarketTabChipServiceOptimized) logUnmappedMarket(marketID int, eventID, marketType, marketName, groups, specifiers, reason string) error {
	query := `
		INSERT INTO market_tab_chip_unmapped (market_id, event_id, market_type, market_name, groups, specifiers, reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (market_id) DO UPDATE SET
			reason = EXCLUDED.reason,
			updated_at = NOW()
	`

	_, err := s.db.Exec(query, marketID, eventID, marketType, marketName, groups, specifiers, reason)
	return err
}

// logAssignment logs an assignment operation
func (s *MarketTabChipServiceOptimized) logAssignment(marketID int, eventID, oldTabID, newTabID, oldChipID, newChipID, assignmentType, assignedBy string) error {
	query := `
		INSERT INTO market_tab_chip_assignment_log (market_id, event_id, old_tab_id, new_tab_id, old_chip_id, new_chip_id, assignment_type, assigned_by, created_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, $7, $8, NOW())
	`

	_, err := s.db.Exec(query, marketID, eventID, oldTabID, newTabID, oldChipID, newChipID, assignmentType, assignedBy)
	return err
}

// parseGroups parses groups string into array
func (s *MarketTabChipServiceOptimized) parseGroups(groupsStr string) []string {
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

// parseSpecifiers parses specifiers string into key-value pairs
func (s *MarketTabChipServiceOptimized) parseSpecifiers(specifiersStr string) []SpecifierPair {
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

// determineTabID determines the tab for a market based on groups and specifiers
func (s *MarketTabChipServiceOptimized) determineTabID(groups []string, specs []SpecifierPair) (string, error) {
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

	for _, spec := range specs {
		if tabID, ok := specifierTabMap[spec.Name]; ok {
			return tabID, nil
		}
	}

	// Default to regular_play if no match found
	return "regular_play", fmt.Errorf("no matching tab found for groups=%v, specifiers=%v", groups, specs)
}

// determineChipID determines the chip for a market based on tab and specifiers
func (s *MarketTabChipServiceOptimized) determineChipID(tabID string, specs []SpecifierPair) string {
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
	for _, spec := range specs {
		if spec.Name == primarySpec {
			return fmt.Sprintf("%s_%s_%s", tabID, spec.Name, spec.Value)
		}
	}

	return ""
}
