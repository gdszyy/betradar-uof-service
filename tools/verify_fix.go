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

	log.Println("=== Verifying Bug Fix ===\n")

	// Check message type distribution
	log.Println("--- Message Type Distribution ---")
	rows, err := db.Query(`
		SELECT 
			CASE 
				WHEN message_type = '' OR message_type IS NULL THEN '(empty)'
				ELSE message_type 
			END as type,
			COUNT(*) as count
		FROM uof_messages
		GROUP BY type
		ORDER BY count DESC
		LIMIT 20
	`)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	totalEmpty := 0
	totalValid := 0
	for rows.Next() {
		var msgType string
		var count int
		if err := rows.Scan(&msgType, &count); err != nil {
			continue
		}
		if msgType == "(empty)" {
			totalEmpty = count
			log.Printf("  ❌ %s: %d messages (OLD - before fix)", msgType, count)
		} else {
			totalValid += count
			log.Printf("  ✅ %s: %d messages", msgType, count)
		}
	}

	log.Printf("\nSummary:")
	log.Printf("  Old messages (empty type): %d", totalEmpty)
	log.Printf("  New messages (valid type): %d", totalValid)

	if totalValid > 0 {
		log.Printf("\n✅ FIX IS WORKING! New messages have valid types.")
	} else {
		log.Printf("\n⚠️  No new messages yet. Wait a few seconds and try again.")
	}

	// Check recent messages (last 5 minutes)
	log.Println("\n--- Recent Messages (Last 5 Minutes) ---")
	rows, err = db.Query(`
		SELECT message_type, event_id, routing_key, received_at
		FROM uof_messages
		WHERE received_at > NOW() - INTERVAL '5 minutes'
		  AND (message_type != '' AND message_type IS NOT NULL)
		ORDER BY received_at DESC
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var msgType, routingKey string
			var eventID sql.NullString
			var receivedAt time.Time
			if err := rows.Scan(&msgType, &eventID, &routingKey, &receivedAt); err != nil {
				continue
			}
			count++
			evtStr := "N/A"
			if eventID.Valid {
				evtStr = eventID.String
			}
			log.Printf("  %d. [%s] %s - Event: %s",
				count, receivedAt.Format("15:04:05"), msgType, evtStr)
		}
		if count == 0 {
			log.Println("  (No recent messages with valid types)")
		}
	}

	// Check specialized tables
	log.Println("\n--- Specialized Tables Status ---")
	tables := map[string]string{
		"odds_changes":    "Odds changes",
		"bet_stops":       "Bet stops",
		"bet_settlements": "Bet settlements",
		"tracked_events":  "Tracked events",
		"producer_status": "Producer status",
	}

	for table, description := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			log.Printf("  ❌ %s: Error - %v", description, err)
			continue
		}
		
		if count > 0 {
			log.Printf("  ✅ %s: %d rows", description, count)
		} else {
			log.Printf("  ⚠️  %s: 0 rows (waiting for data...)", description)
		}
	}

	// Check producer status specifically
	log.Println("\n--- Producer Status Detail ---")
	rows, err = db.Query(`
		SELECT product_id, status, last_alive, subscribed, updated_at
		FROM producer_status
		ORDER BY product_id
	`)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var productID, subscribed int
			var status string
			var lastAlive int64
			var updatedAt time.Time
			if err := rows.Scan(&productID, &status, &lastAlive, &subscribed, &updatedAt); err != nil {
				continue
			}
			count++
			
			// Check if alive is recent (within last minute)
			aliveTime := time.Unix(lastAlive/1000, 0)
			isRecent := time.Since(aliveTime) < time.Minute
			recentMark := "✅"
			if !isRecent {
				recentMark = "⚠️"
			}
			
			log.Printf("  %s Product %d: %s - Subscribed: %d - Last alive: %s",
				recentMark, productID, status, subscribed, aliveTime.Format("15:04:05"))
		}
		if count == 0 {
			log.Println("  (No producer status yet - waiting for alive messages)")
		}
	}

	// Overall assessment
	log.Println("\n=== Assessment ===")
	if totalValid > 0 {
		log.Println("✅ Bug fix is WORKING - new messages are being parsed correctly")
		log.Println("✅ Service is receiving and processing messages")
		
		if totalEmpty > 0 {
			log.Printf("\n⚠️  You have %d old messages with empty type", totalEmpty)
			log.Println("   Recommendation: Run cleanup_database.go to remove them")
			log.Println("   Command: go run tools/cleanup_database.go")
		}
	} else {
		log.Println("⚠️  No new messages detected yet")
		log.Println("   Possible reasons:")
		log.Println("   1. Service just deployed - wait 10-20 seconds")
		log.Println("   2. Service not running - check Railway logs")
		log.Println("   3. AMQP connection issue - check Railway logs")
	}

	log.Println("\n=== Verification Complete ===")
}

