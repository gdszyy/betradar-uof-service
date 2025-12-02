package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dbConnStr := flag.String("db", "", "Database connection string")
	tabConfigFile := flag.String("tabs", "", "Path to tab configuration CSV file")
	chipConfigFile := flag.String("chips", "", "Path to chip configuration CSV file")
	flag.Parse()

	if *dbConnStr == "" || *tabConfigFile == "" || *chipConfigFile == "" {
		fmt.Println("Usage: import_tab_chip -db <connection_string> -tabs <tab_csv> -chips <chip_csv>")
		fmt.Println("\nExample:")
		fmt.Println("  import_tab_chip -db 'postgresql://user:pass@localhost/dbname' \\")
		fmt.Println("    -tabs final_tab_chip_config.csv -chips final_chip_enumeration.csv")
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", *dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✓ Connected to database successfully")

	// Import tab configurations
	log.Println("\n=== Importing Tab Configurations ===")
	if err := importTabConfigs(db, *tabConfigFile); err != nil {
		log.Fatalf("Failed to import tab configurations: %v", err)
	}

	// Import chip configurations
	log.Println("\n=== Importing Chip Configurations ===")
	if err := importChipConfigs(db, *chipConfigFile); err != nil {
		log.Fatalf("Failed to import chip configurations: %v", err)
	}

	log.Println("\n✓ Tab and chip configurations imported successfully!")
}

func importTabConfigs(db *sql.DB, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open tab config file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read tab config CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("tab config CSV must have header and at least one data row")
	}

	log.Printf("Found %d tab configurations to import", len(records)-1)

	// Skip header row
	successCount := 0
	for i, record := range records[1:] {
		if len(record) < 8 {
			log.Printf("⚠ Warning: skipping row %d with insufficient columns", i+2)
			continue
		}

		marketCount, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			log.Printf("⚠ Warning: invalid market count in row %d: %v", i+2, err)
			marketCount = 0
		}

		tabID := strings.TrimSpace(record[0])
		tabLabel := strings.TrimSpace(record[1])
		tabType := strings.TrimSpace(record[2])
		chipSpecifiers := strings.TrimSpace(record[4])
		group := strings.TrimSpace(record[5])
		primarySpecifier := strings.TrimSpace(record[6])

		// Handle "无" (none) value
		if chipSpecifiers == "无" {
			chipSpecifiers = ""
		}

		// Insert or update tab configuration
		query := `
			INSERT INTO market_tabs (id, label, type, market_count, chip_specifiers, group_name, primary_specifier, display_order, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $9)
			ON CONFLICT (id) DO UPDATE SET
				label = EXCLUDED.label,
				type = EXCLUDED.type,
				market_count = EXCLUDED.market_count,
				chip_specifiers = EXCLUDED.chip_specifiers,
				group_name = EXCLUDED.group_name,
				primary_specifier = EXCLUDED.primary_specifier,
				updated_at = EXCLUDED.updated_at
		`

		displayOrder := i
		now := time.Now()

		_, err = db.Exec(query, tabID, tabLabel, tabType, marketCount, chipSpecifiers, group, primarySpecifier, displayOrder, now)
		if err != nil {
			return fmt.Errorf("failed to insert tab config row %d: %w", i+2, err)
		}

		successCount++
		log.Printf("  ✓ Tab %d/%d: %s (%s)", successCount, len(records)-1, tabID, tabLabel)
	}

	log.Printf("✓ Successfully imported %d tabs", successCount)
	return nil
}

func importChipConfigs(db *sql.DB, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open chip config file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read chip config CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("chip config CSV must have header and at least one data row")
	}

	log.Printf("Found %d chip configurations to import", len(records)-1)

	// Skip header row
	chipIndex := 0
	successCount := 0
	currentTabID := ""

	for i, record := range records[1:] {
		if len(record) < 5 {
			log.Printf("⚠ Warning: skipping row %d with insufficient columns", i+2)
			continue
		}

		tabID := strings.TrimSpace(record[0])
		chipSpecifier := strings.TrimSpace(record[2])
		chipValue := strings.TrimSpace(record[3])
		chipLabel := strings.TrimSpace(record[4])

		// Reset display order when tab changes
		if tabID != currentTabID {
			chipIndex = 0
			currentTabID = tabID
		}

		// Generate chip ID: tab_id_specifier_value
		chipID := fmt.Sprintf("%s_%s_%s", tabID, chipSpecifier, chipValue)

		// Handle "dynamic" values
		var specifierPtr *string
		var valuePtr *string

		if chipSpecifier != "dynamic" && chipSpecifier != "" {
			specifierPtr = &chipSpecifier
		}
		if chipValue != "dynamic" && chipValue != "" {
			valuePtr = &chipValue
		}

		// Insert or update chip configuration
		query := `
			INSERT INTO market_chips (id, tab_id, specifier, value, label, display_order, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
			ON CONFLICT (id) DO UPDATE SET
				label = EXCLUDED.label,
				display_order = EXCLUDED.display_order,
				updated_at = EXCLUDED.updated_at
		`

		now := time.Now()

		_, err = db.Exec(query, chipID, tabID, specifierPtr, valuePtr, chipLabel, chipIndex, now)
		if err != nil {
			return fmt.Errorf("failed to insert chip config row %d: %w", i+2, err)
		}

		chipIndex++
		successCount++
		if successCount%10 == 0 {
			log.Printf("  ✓ Imported %d chips...", successCount)
		}
	}

	log.Printf("✓ Successfully imported %d chips", successCount)
	return nil
}
