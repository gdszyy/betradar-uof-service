package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Connect 连接到数据库
func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

// Migrate 运行数据库迁移
func Migrate(db *sql.DB) error {
	migrations := []string{
		// UOF消息表
		`CREATE TABLE IF NOT EXISTS uof_messages (
			id BIGSERIAL PRIMARY KEY,
			message_type VARCHAR(50) NOT NULL,
			event_id VARCHAR(100),
			product_id INTEGER,
			sport_id VARCHAR(50),
			routing_key VARCHAR(255) NOT NULL,
			xml_content TEXT NOT NULL,
			timestamp BIGINT,
			received_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_event_id (event_id),
			INDEX idx_message_type (message_type),
			INDEX idx_received_at (received_at)
		)`,

		// 跟踪的赛事表
		`CREATE TABLE IF NOT EXISTS tracked_events (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) UNIQUE NOT NULL,
			sport_id VARCHAR(50),
			status VARCHAR(20) DEFAULT 'active',
			message_count INTEGER DEFAULT 0,
			last_message_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_event_id (event_id),
			INDEX idx_status (status)
		)`,

		// 赔率变化表
		`CREATE TABLE IF NOT EXISTS odds_changes (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) NOT NULL,
			product_id INTEGER NOT NULL,
			timestamp BIGINT NOT NULL,
			odds_change_reason VARCHAR(50),
			markets_count INTEGER DEFAULT 0,
			xml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_event_id (event_id),
			INDEX idx_timestamp (timestamp)
		)`,

		// 投注停止表
		`CREATE TABLE IF NOT EXISTS bet_stops (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) NOT NULL,
			product_id INTEGER NOT NULL,
			timestamp BIGINT NOT NULL,
			market_status VARCHAR(50),
			xml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_event_id (event_id)
		)`,

		// 投注结算表
		`CREATE TABLE IF NOT EXISTS bet_settlements (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) NOT NULL,
			product_id INTEGER NOT NULL,
			timestamp BIGINT NOT NULL,
			certainty INTEGER,
			xml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_event_id (event_id)
		)`,

		// 生产者状态表
		`CREATE TABLE IF NOT EXISTS producer_status (
			id BIGSERIAL PRIMARY KEY,
			product_id INTEGER UNIQUE NOT NULL,
			status VARCHAR(20) DEFAULT 'unknown',
			last_alive BIGINT NOT NULL,
			subscribed INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

