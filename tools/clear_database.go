package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// 从环境变量获取数据库URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Printf("Connecting to database...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to database")
	log.Println("")
	log.Println("⚠️  WARNING: This will DELETE ALL DATA from the following tables:")
	log.Println("   - messages")
	log.Println("   - recovery_status")
	log.Println("   - ld_events")
	log.Println("   - ld_matches")
	log.Println("   - ld_lineups")
	log.Println("")
	log.Println("Press Ctrl+C to cancel, or press Enter to continue...")
	
	// 等待用户确认
	fmt.Scanln()

	log.Println("🗑️  Starting database cleanup...")

	// 清空所有表
	tables := []string{
		"messages",
		"recovery_status",
		"ld_events",
		"ld_matches",
		"ld_lineups",
	}

	for _, table := range tables {
		log.Printf("Truncating table: %s", table)
		
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)
		if _, err := db.Exec(query); err != nil {
			log.Printf("⚠️  Failed to truncate %s: %v (table may not exist)", table, err)
		} else {
			log.Printf("✅ Truncated: %s", table)
		}
	}

	log.Println("")
	log.Println("✅ Database cleanup completed!")
	log.Println("")
	
	// 显示统计信息
	log.Println("📊 Verifying cleanup:")
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.QueryRow(query).Scan(&count); err != nil {
			log.Printf("   %s: (table may not exist)", table)
		} else {
			log.Printf("   %s: %d rows", table, count)
		}
	}
}

