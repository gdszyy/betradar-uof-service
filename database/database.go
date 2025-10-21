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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_uof_messages_event_id ON uof_messages(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_uof_messages_message_type ON uof_messages(message_type)`,
		`CREATE INDEX IF NOT EXISTS idx_uof_messages_received_at ON uof_messages(received_at)`,

		// 跟踪的赛事表
		`CREATE TABLE IF NOT EXISTS tracked_events (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) UNIQUE NOT NULL,
			sport_id VARCHAR(50),
			status VARCHAR(20) DEFAULT 'active',
			message_count INTEGER DEFAULT 0,
			last_message_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_event_id ON tracked_events(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_status ON tracked_events(status)`,

		// 赔率变化表
		`CREATE TABLE IF NOT EXISTS odds_changes (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) NOT NULL,
			product_id INTEGER NOT NULL,
			timestamp BIGINT NOT NULL,
			odds_change_reason VARCHAR(50),
			markets_count INTEGER DEFAULT 0,
			xml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_odds_changes_event_id ON odds_changes(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_odds_changes_timestamp ON odds_changes(timestamp)`,

		// 投注停止表
		`CREATE TABLE IF NOT EXISTS bet_stops (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) NOT NULL,
			product_id INTEGER NOT NULL,
			timestamp BIGINT NOT NULL,
			market_status VARCHAR(50),
			xml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bet_stops_event_id ON bet_stops(event_id)`,

		// 投注结算表
		`CREATE TABLE IF NOT EXISTS bet_settlements (
			id BIGSERIAL PRIMARY KEY,
			event_id VARCHAR(100) NOT NULL,
			product_id INTEGER NOT NULL,
			timestamp BIGINT NOT NULL,
			certainty INTEGER,
			xml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bet_settlements_event_id ON bet_settlements(event_id)`,

			// 生产者状态表
			`CREATE TABLE IF NOT EXISTS producer_status (
				id BIGSERIAL PRIMARY KEY,
				product_id INTEGER UNIQUE NOT NULL,
				status VARCHAR(20) DEFAULT 'unknown',
				last_alive BIGINT NOT NULL,
				subscribed INTEGER DEFAULT 0,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,

			// 恢复状态表
			`CREATE TABLE IF NOT EXISTS recovery_status (
				id BIGSERIAL PRIMARY KEY,
				request_id INTEGER NOT NULL,
				product_id INTEGER NOT NULL,
				node_id INTEGER NOT NULL,
				status VARCHAR(20) DEFAULT 'initiated',
				timestamp BIGINT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				completed_at TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_recovery_status_request_id ON recovery_status(request_id)`,
			`CREATE INDEX IF NOT EXISTS idx_recovery_status_product_id ON recovery_status(product_id)`,
			`CREATE INDEX IF NOT EXISTS idx_recovery_status_status ON recovery_status(status)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

