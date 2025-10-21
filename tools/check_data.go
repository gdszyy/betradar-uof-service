package main

import (
	"database/sql"
	"log"
	"os"
	"time"

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

	log.Println("=== Database Statistics ===\n")

	// Check each table
	tables := []string{
		"uof_messages",
		"tracked_events",
		"odds_changes",
		"bet_stops",
		"bet_settlements",
		"producer_status",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			log.Printf("✗ %s: Error - %v", table, err)
			continue
		}
		log.Printf("✓ %s: %d rows", table, count)

		// Show latest record timestamp if available
		if table == "uof_messages" && count > 0 {
			var latestTime time.Time
			err := db.QueryRow("SELECT MAX(received_at) FROM uof_messages").Scan(&latestTime)
			if err == nil {
				log.Printf("  └─ Latest message: %s", latestTime.Format("2006-01-02 15:04:05"))
			}
		}

		if table == "tracked_events" && count > 0 {
			var latestTime sql.NullTime
			err := db.QueryRow("SELECT MAX(last_message_at) FROM tracked_events").Scan(&latestTime)
			if err == nil && latestTime.Valid {
				log.Printf("  └─ Latest event update: %s", latestTime.Time.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// Show sample of recent messages
	log.Println("\n=== Recent Messages (Last 5) ===\n")
	rows, err := db.Query(`
		SELECT message_type, event_id, routing_key, received_at 
		FROM uof_messages 
		ORDER BY received_at DESC 
		LIMIT 5
	`)
	if err != nil {
		log.Printf("Error querying messages: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var msgType, eventID, routingKey string
			var receivedAt time.Time
			if err := rows.Scan(&msgType, &eventID, &routingKey, &receivedAt); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			count++
			log.Printf("%d. [%s] %s - Event: %s - %s",
				count, receivedAt.Format("15:04:05"), msgType, eventID, routingKey)
		}
		if count == 0 {
			log.Println("(No messages found)")
		}
	}

	// Show tracked events
	log.Println("\n=== Tracked Events (Top 5 by message count) ===\n")
	rows, err = db.Query(`
		SELECT event_id, sport_id, status, message_count, last_message_at 
		FROM tracked_events 
		ORDER BY message_count DESC 
		LIMIT 5
	`)
	if err != nil {
		log.Printf("Error querying tracked events: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var eventID, sportID, status string
			var messageCount int
			var lastMessageAt sql.NullTime
			if err := rows.Scan(&eventID, &sportID, &status, &messageCount, &lastMessageAt); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			count++
			lastMsg := "never"
			if lastMessageAt.Valid {
				lastMsg = lastMessageAt.Time.Format("15:04:05")
			}
			log.Printf("%d. Event %s (Sport: %s) - %s - %d messages - Last: %s",
				count, eventID, sportID, status, messageCount, lastMsg)
		}
		if count == 0 {
			log.Println("(No tracked events found)")
		}
	}

	// Show producer status
	log.Println("\n=== Producer Status ===\n")
	rows, err = db.Query(`
		SELECT product_id, status, last_alive, subscribed, updated_at 
		FROM producer_status 
		ORDER BY product_id
	`)
	if err != nil {
		log.Printf("Error querying producer status: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var productID int
			var status string
			var lastAlive int64
			var subscribed int
			var updatedAt time.Time
			if err := rows.Scan(&productID, &status, &lastAlive, &subscribed, &updatedAt); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			count++
			log.Printf("Product %d: %s - Subscribed: %d - Updated: %s",
				productID, status, subscribed, updatedAt.Format("15:04:05"))
		}
		if count == 0 {
			log.Println("(No producer status found)")
		}
	}

	log.Println("\n=== Check Complete ===")
}

