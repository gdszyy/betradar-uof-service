package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"betradar-uof-service/services"
)

func main() {
	dbConnStr := flag.String("db", "", "Database connection string")
	mode := flag.String("mode", "incremental", "Assignment mode: 'incremental' (default) or 'full'")
	flag.Parse()

	if *dbConnStr == "" {
		fmt.Println("Usage: assign_tab_chip_incremental -db <connection_string> [-mode incremental|full]")
		fmt.Println("\nModes:")
		fmt.Println("  incremental - Only assign tab/chip to markets without tab_id (default, recommended for production)")
		fmt.Println("  full        - Assign tab/chip to all markets, updating existing assignments")
		fmt.Println("\nExample:")
		fmt.Println("  assign_tab_chip_incremental -db 'postgresql://user:pass@localhost/dbname' -mode incremental")
		os.Exit(1)
	}

	// Validate mode
	if *mode != "incremental" && *mode != "full" {
		fmt.Printf("Invalid mode: %s. Must be 'incremental' or 'full'\n", *mode)
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

	// Create service
	service := services.NewMarketTabChipServiceOptimized(db)

	// Run assignment based on mode
	log.Printf("\n=== Running in %s mode ===\n", *mode)

	var assignErr error
	if *mode == "incremental" {
		assignErr = service.AssignTabChipToNewMarkets()
	} else {
		assignErr = service.AssignTabChipToAllMarkets()
	}

	if assignErr != nil {
		log.Fatalf("Failed to assign tab/chip: %v", assignErr)
	}

	// Print summary of unmapped markets
	log.Println("\n=== Unmapped Markets Summary ===")
	summary, err := service.GetUnmappedSummary()
	if err != nil {
		log.Printf("Warning: Failed to get unmapped summary: %v", err)
	} else {
		if len(summary) == 0 {
			log.Println("✓ No unmapped markets found")
		} else {
			log.Printf("Found %d unmapped reasons:\n", len(summary))
			for _, item := range summary {
				log.Printf("  - %s: %d markets (%d events)\n",
					item["reason"], item["count"], item["event_count"])
			}
		}
	}

	log.Println("\n✓ Tab/Chip assignment completed successfully!")
}
