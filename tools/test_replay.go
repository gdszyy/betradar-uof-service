package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"uof-service/services"
)

func main() {
	// 命令行参数
	eventID := flag.String("event", "", "Event ID to replay (e.g., sr:match:12345)")
	speed := flag.Int("speed", 10, "Replay speed multiplier (default: 10x)")
	nodeID := flag.Int("node", 1, "Node ID for routing (default: 1)")
	duration := flag.Int("duration", 60, "How long to run the replay in seconds (default: 60)")
	stopAfter := flag.Bool("stop", true, "Stop replay after duration (default: true)")
	flag.Parse()

	if *eventID == "" {
		log.Fatal("❌ Event ID is required. Use -event=sr:match:12345")
	}

	// 从环境变量获取access token
	accessToken := os.Getenv("BETRADAR_ACCESS_TOKEN")
	dbURL := os.Getenv("DATABASE_URL")

	if accessToken == "" {
		log.Fatal("❌ BETRADAR_ACCESS_TOKEN environment variable is required")
	}

	log.Println("🎬 Betradar UOF Replay Test")
	log.Println("=" + string(make([]byte, 50)))
	log.Printf("Event ID: %s", *eventID)
	log.Printf("Speed: %dx", *speed)
	log.Printf("Node ID: %d", *nodeID)
	log.Printf("Duration: %d seconds", *duration)
	log.Println()

	// 创建重放客户端
	apiBaseURL := os.Getenv("BETRADAR_API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "https://stgapi.betradar.com/v1"
	}
	client := services.NewReplayClient(accessToken, apiBaseURL)

	// 连接数据库(如果提供)
	var db *sql.DB
	var err error
	if dbURL != "" {
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("⚠️  Database connection failed: %v", err)
		} else {
			defer db.Close()
			log.Println("✅ Connected to database")
		}
	}

	// 获取初始统计
	var initialCount int64
	if db != nil {
		err = db.QueryRow("SELECT COUNT(*) FROM uof_messages").Scan(&initialCount)
		if err != nil {
			log.Printf("⚠️  Failed to get initial count: %v", err)
		} else {
			log.Printf("📊 Initial message count: %d", initialCount)
		}
	}
	log.Println()

	// 1. 快速重放
	log.Println("🚀 Starting replay...")
	if err := client.QuickReplay(*eventID, *speed, *nodeID); err != nil {
		log.Fatalf("❌ Failed to start replay: %v", err)
	}

	log.Println()
	log.Printf("⏱️  Replay is running. Waiting %d seconds...", *duration)
	log.Println("   (Check your service logs to see incoming messages)")
	log.Println()

	// 2. 监控一段时间
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if db != nil {
				var currentCount int64
				err = db.QueryRow("SELECT COUNT(*) FROM uof_messages").Scan(&currentCount)
				if err == nil {
					newMessages := currentCount - initialCount
					log.Printf("📈 [%ds] New messages: %d (Total: %d)", 
						int(elapsed.Seconds()), newMessages, currentCount)
					
					// 显示最近的消息类型
					rows, err := db.Query(`
						SELECT message_type, COUNT(*) as count
						FROM uof_messages
						WHERE created_at > NOW() - INTERVAL '10 seconds'
						GROUP BY message_type
						ORDER BY count DESC
					`)
					if err == nil {
						log.Println("   Recent message types:")
						for rows.Next() {
							var msgType string
							var count int
							if err := rows.Scan(&msgType, &count); err == nil {
								if msgType == "" {
									msgType = "(empty)"
								}
								log.Printf("     - %s: %d", msgType, count)
							}
						}
						rows.Close()
					}
				}
			}

			if elapsed >= time.Duration(*duration)*time.Second {
				goto done
			}
		}
	}

done:
	log.Println()
	log.Println("⏰ Duration completed")

	// 3. 获取最终统计
	if db != nil {
		var finalCount int64
		err = db.QueryRow("SELECT COUNT(*) FROM uof_messages").Scan(&finalCount)
		if err == nil {
			newMessages := finalCount - initialCount
			log.Printf("📊 Final statistics:")
			log.Printf("   Initial count: %d", initialCount)
			log.Printf("   Final count: %d", finalCount)
			log.Printf("   New messages: %d", newMessages)
			log.Printf("   Messages per second: %.2f", float64(newMessages)/float64(*duration))
		}

		// 显示消息类型分布
		log.Println()
		log.Println("📋 Message type distribution (last minute):")
		rows, err := db.Query(`
			SELECT message_type, COUNT(*) as count
			FROM uof_messages
			WHERE created_at > NOW() - INTERVAL '1 minute'
			GROUP BY message_type
			ORDER BY count DESC
		`)
		if err == nil {
			for rows.Next() {
				var msgType string
				var count int
				if err := rows.Scan(&msgType, &count); err == nil {
					if msgType == "" {
						msgType = "(empty)"
					}
					log.Printf("   %s: %d", msgType, count)
				}
			}
			rows.Close()
		}

		// 显示专门表的统计
		log.Println()
		log.Println("📊 Specialized tables:")
		tables := []string{"odds_changes", "bet_stops", "bet_settlements", "tracked_events"}
		for _, table := range tables {
			var count int64
			err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
			if err == nil {
				log.Printf("   %s: %d rows", table, count)
			}
		}
	}

	// 4. 停止重放(如果需要)
	if *stopAfter {
		log.Println()
		log.Println("🛑 Stopping replay...")
		if err := client.Stop(); err != nil {
			log.Printf("⚠️  Failed to stop replay: %v", err)
		} else {
			log.Println("✅ Replay stopped")
		}
	}

	log.Println()
	log.Println("✅ Test completed!")
	log.Println()
	log.Println("Next steps:")
	log.Println("1. Check your service logs for detailed message processing")
	log.Println("2. Query the database to see stored odds_changes, bet_stops, etc.")
	log.Println("3. Open the WebSocket UI to see real-time messages")
	log.Println("4. Use /api/messages to see the latest messages via API")
}

