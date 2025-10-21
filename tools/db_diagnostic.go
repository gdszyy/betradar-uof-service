package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Printf("Connecting to database...")
	log.Printf("DATABASE_URL: %s", maskPassword(databaseURL))

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✓ Database connection successful")

	// Check current schema
	log.Println("\n--- Checking current schema ---")
	var currentSchema string
	err = db.QueryRow("SELECT current_schema()").Scan(&currentSchema)
	if err != nil {
		log.Fatalf("Failed to get current schema: %v", err)
	}
	log.Printf("Current schema: %s", currentSchema)

	// Check search path
	var searchPath string
	err = db.QueryRow("SHOW search_path").Scan(&searchPath)
	if err != nil {
		log.Fatalf("Failed to get search path: %v", err)
	}
	log.Printf("Search path: %s", searchPath)

	// List all schemas
	log.Println("\n--- Available schemas ---")
	rows, err := db.Query("SELECT schema_name FROM information_schema.schemata ORDER BY schema_name")
	if err != nil {
		log.Fatalf("Failed to list schemas: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			log.Printf("Error scanning schema: %v", err)
			continue
		}
		log.Printf("  - %s", schemaName)
	}

	// List all tables in public schema
	log.Println("\n--- Tables in public schema ---")
	rows, err = db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatalf("Failed to list tables: %v", err)
	}
	defer rows.Close()

	tableCount := 0
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Printf("Error scanning table: %v", err)
			continue
		}
		log.Printf("  - %s", tableName)
		tableCount++
	}
	
	if tableCount == 0 {
		log.Println("  (no tables found)")
	}

	// Try to create a test table
	log.Println("\n--- Testing table creation ---")
	testTableSQL := `CREATE TABLE IF NOT EXISTS test_diagnostic_table (
		id SERIAL PRIMARY KEY,
		test_value TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	
	_, err = db.Exec(testTableSQL)
	if err != nil {
		log.Printf("✗ Failed to create test table: %v", err)
	} else {
		log.Println("✓ Test table created successfully")
		
		// Verify it exists
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'test_diagnostic_table'
			)
		`).Scan(&exists)
		
		if err != nil {
			log.Printf("✗ Failed to verify test table: %v", err)
		} else if exists {
			log.Println("✓ Test table verified in public schema")
			
			// Clean up
			_, err = db.Exec("DROP TABLE IF EXISTS test_diagnostic_table")
			if err != nil {
				log.Printf("Warning: Failed to drop test table: %v", err)
			} else {
				log.Println("✓ Test table cleaned up")
			}
		} else {
			log.Println("✗ Test table not found after creation!")
		}
	}

	// Check database permissions
	log.Println("\n--- Checking permissions ---")
	var hasCreatePrivilege bool
	err = db.QueryRow(`
		SELECT has_schema_privilege(current_user, 'public', 'CREATE')
	`).Scan(&hasCreatePrivilege)
	
	if err != nil {
		log.Printf("Failed to check CREATE privilege: %v", err)
	} else {
		log.Printf("CREATE privilege on public schema: %v", hasCreatePrivilege)
	}

	log.Println("\n--- Diagnostic complete ---")
}

func maskPassword(dbURL string) string {
	// Simple password masking for logging
	// Format: postgres://user:password@host:port/database
	if len(dbURL) > 20 {
		return dbURL[:20] + "..."
	}
	return "***"
}

