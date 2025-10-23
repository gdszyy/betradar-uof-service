package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func main() {
	// 从环境变量获取数据库 URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// 连接数据库
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// 读取迁移文件
	migrationsDir := "database/migrations"
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	if len(files) == 0 {
		log.Println("No migration files found")
		return
	}

	// 执行每个迁移文件
	for _, file := range files {
		log.Printf("Running migration: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			log.Fatalf("Failed to execute migration %s: %v", file, err)
		}

		log.Printf("✅ Migration %s completed successfully", filepath.Base(file))
	}

	log.Println("All migrations completed successfully!")
}

