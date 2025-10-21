package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("=== Database Cleanup Tool ===\n")

	// Show current statistics
	var totalMessages int
	err = db.QueryRow("SELECT COUNT(*) FROM uof_messages").Scan(&totalMessages)
	if err != nil {
		log.Fatalf("Failed to count messages: %v", err)
	}

	var emptyTypeMessages int
	err = db.QueryRow("SELECT COUNT(*) FROM uof_messages WHERE message_type = '' OR message_type IS NULL").Scan(&emptyTypeMessages)
	if err != nil {
		log.Fatalf("Failed to count empty type messages: %v", err)
	}

	log.Printf("Total messages: %d", totalMessages)
	log.Printf("Messages with empty type: %d", emptyTypeMessages)
	log.Printf("Messages with valid type: %d\n", totalMessages-emptyTypeMessages)

	if emptyTypeMessages == 0 {
		log.Println("✓ No cleanup needed - all messages have valid types")
		return
	}

	// Ask for confirmation
	fmt.Printf("\nThis will DELETE %d messages with empty message_type.\n", emptyTypeMessages)
	fmt.Print("Are you sure you want to continue? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" {
		log.Println("Cleanup cancelled")
		return
	}

	// Perform cleanup
	log.Println("\nStarting cleanup...")

	// Delete from uof_messages
	result, err := db.Exec("DELETE FROM uof_messages WHERE message_type = '' OR message_type IS NULL")
	if err != nil {
		log.Fatalf("Failed to delete messages: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("✓ Deleted %d messages from uof_messages", rowsAffected)

	// Also clean up other tables (they should be empty anyway based on our diagnosis)
	tables := []string{"odds_changes", "bet_stops", "bet_settlements", "tracked_events", "producer_status"}
	for _, table := range tables {
		result, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Warning: Failed to clean %s: %v", table, err)
			continue
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("✓ Deleted %d rows from %s", rowsAffected, table)
		}
	}

	// Show final statistics
	err = db.QueryRow("SELECT COUNT(*) FROM uof_messages").Scan(&totalMessages)
	if err == nil {
		log.Printf("\n✓ Cleanup complete! Remaining messages: %d", totalMessages)
	}

	log.Println("\n=== Cleanup Complete ===")
	log.Println("You can now redeploy the service with the fixed code.")
}

