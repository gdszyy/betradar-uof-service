package main

import (
	"database/sql"
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

	log.Println("=== Examining Recent Messages ===\n")

	// Get sample messages
	rows, err := db.Query(`
		SELECT message_type, routing_key, xml_content, event_id, product_id, sport_id, timestamp
		FROM uof_messages 
		ORDER BY received_at DESC 
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Failed to query messages: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var msgType, routingKey, xmlContent string
		var eventID, sportID sql.NullString
		var productID sql.NullInt64
		var timestamp sql.NullInt64

		if err := rows.Scan(&msgType, &routingKey, &xmlContent, &eventID, &productID, &sportID, &timestamp); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		count++
		log.Printf("\n--- Message %d ---", count)
		log.Printf("Type: %s", msgType)
		log.Printf("Routing Key: %s", routingKey)
		log.Printf("Event ID: %v (valid: %v)", eventID.String, eventID.Valid)
		log.Printf("Product ID: %v (valid: %v)", productID.Int64, productID.Valid)
		log.Printf("Sport ID: %v (valid: %v)", sportID.String, sportID.Valid)
		log.Printf("Timestamp: %v (valid: %v)", timestamp.Int64, timestamp.Valid)
		
		// Show first 500 chars of XML
		xmlPreview := xmlContent
		if len(xmlPreview) > 500 {
			xmlPreview = xmlPreview[:500] + "..."
		}
		// Remove extra whitespace for readability
		xmlPreview = strings.ReplaceAll(xmlPreview, "\n", " ")
		xmlPreview = strings.ReplaceAll(xmlPreview, "  ", " ")
		log.Printf("XML Preview: %s", xmlPreview)
	}

	if count == 0 {
		log.Println("No messages found")
	}

	// Get message type distribution
	log.Println("\n\n=== Message Type Distribution ===\n")
	rows, err = db.Query(`
		SELECT message_type, COUNT(*) as count
		FROM uof_messages
		GROUP BY message_type
		ORDER BY count DESC
	`)
	if err != nil {
		log.Printf("Failed to get distribution: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var msgType string
			var count int
			if err := rows.Scan(&msgType, &count); err != nil {
				continue
			}
			log.Printf("%s: %d messages", msgType, count)
		}
	}

	log.Println("\n=== Examination Complete ===")
}

