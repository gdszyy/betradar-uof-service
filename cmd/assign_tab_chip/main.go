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
	flag.Parse()

	if *dbConnStr == "" {
		fmt.Println("Usage: assign_tab_chip -db <connection_string>")
		fmt.Println("\nExample:")
		fmt.Println("  assign_tab_chip -db 'postgresql://user:pass@localhost/dbname'")
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
	service := services.NewMarketTabChipService(db)

	// Assign tab/chip to all markets
	log.Println("\n=== Assigning Tab/Chip to Markets ===")
	if err := service.AssignTabChipToAllMarkets(); err != nil {
		log.Fatalf("Failed to assign tab/chip to markets: %v", err)
	}

	log.Println("\n✓ Tab/Chip assignment completed successfully!")
}
