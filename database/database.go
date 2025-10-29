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
			
			// Live Data 事件表
			`CREATE TABLE IF NOT EXISTS ld_events (
				id BIGSERIAL PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				uuid VARCHAR(36) UNIQUE,
				event_id VARCHAR(50),
				match_id VARCHAR(50) NOT NULL,
				sport_id INTEGER,
				type INTEGER NOT NULL,
				type_name VARCHAR(100),
				info TEXT,
				side VARCHAR(10),
				mtime VARCHAR(10),
				stime BIGINT,
				match_status VARCHAR(20),
				t1_score INTEGER DEFAULT 0,
				t2_score INTEGER DEFAULT 0,
				player1 VARCHAR(100),
				player2 VARCHAR(100),
				extra_info TEXT,
				is_important BOOLEAN DEFAULT false
			)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_events_uuid ON ld_events(uuid)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_events_match_id ON ld_events(match_id)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_events_type ON ld_events(type)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_events_stime ON ld_events(stime)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_events_is_important ON ld_events(is_important)`,
			
			// Live Data 比赛表
			`CREATE TABLE IF NOT EXISTS ld_matches (
				id BIGSERIAL PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				match_id VARCHAR(50) UNIQUE NOT NULL,
				sport_id INTEGER,
				t1_id VARCHAR(50),
				t2_id VARCHAR(50),
				t1_name VARCHAR(200),
				t2_name VARCHAR(200),
				match_status VARCHAR(20),
				match_time VARCHAR(10),
				t1_score INTEGER DEFAULT 0,
				t2_score INTEGER DEFAULT 0,
				match_date VARCHAR(20),
				start_time VARCHAR(20),
				coverage_type VARCHAR(50),
				device_id VARCHAR(50),
				subscribed BOOLEAN DEFAULT false,
				last_event_at TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_matches_match_id ON ld_matches(match_id)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_matches_sport_id ON ld_matches(sport_id)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_matches_status ON ld_matches(match_status)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_matches_subscribed ON ld_matches(subscribed)`,
			
			// Live Data 阵容表
			`CREATE TABLE IF NOT EXISTS ld_lineups (
				id BIGSERIAL PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				match_id VARCHAR(50) NOT NULL,
				team1_players TEXT,
				team2_players TEXT
			)`,
			`CREATE INDEX IF NOT EXISTS idx_ld_lineups_match_id ON ld_lineups(match_id)`,
			
			// 静态数据表 - Sports
			`CREATE TABLE IF NOT EXISTS sports (
				id VARCHAR(50) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			
			// 静态数据表 - Categories
			`CREATE TABLE IF NOT EXISTS categories (
				id VARCHAR(50) PRIMARY KEY,
				sport_id VARCHAR(50) NOT NULL,
				name VARCHAR(255) NOT NULL,
				country_code VARCHAR(10),
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_categories_sport_id ON categories(sport_id)`,
			
			// 静态数据表 - Tournaments
			`CREATE TABLE IF NOT EXISTS tournaments (
				id VARCHAR(50) PRIMARY KEY,
				sport_id VARCHAR(50) NOT NULL,
				category_id VARCHAR(50) NOT NULL,
				name VARCHAR(255) NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_tournaments_sport_id ON tournaments(sport_id)`,
			`CREATE INDEX IF NOT EXISTS idx_tournaments_category_id ON tournaments(category_id)`,
			
			// 静态数据表 - Void Reasons
			`CREATE TABLE IF NOT EXISTS void_reasons (
				id INT PRIMARY KEY,
				description VARCHAR(255) NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			
			// 静态数据表 - Betstop Reasons
			`CREATE TABLE IF NOT EXISTS betstop_reasons (
				id INT PRIMARY KEY,
				description VARCHAR(255) NOT NULL,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			
			// 赛程表 - Scheduled Events
			`CREATE TABLE IF NOT EXISTS scheduled_events (
				id BIGSERIAL PRIMARY KEY,
				event_id VARCHAR(100) UNIQUE NOT NULL,
				sport_id VARCHAR(50),
				sport_name VARCHAR(255),
				category_id VARCHAR(50),
				category_name VARCHAR(255),
				tournament_id VARCHAR(50),
				tournament_name VARCHAR(255),
				home_team_id VARCHAR(100),
				home_team_name VARCHAR(255),
				away_team_id VARCHAR(100),
				away_team_name VARCHAR(255),
				scheduled_time TIMESTAMP,
				status VARCHAR(50),
				live_odds VARCHAR(20),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE INDEX IF NOT EXISTS idx_scheduled_events_event_id ON scheduled_events(event_id)`,
			`CREATE INDEX IF NOT EXISTS idx_scheduled_events_sport_id ON scheduled_events(sport_id)`,
			`CREATE INDEX IF NOT EXISTS idx_scheduled_events_scheduled_time ON scheduled_events(scheduled_time)`,
			`CREATE INDEX IF NOT EXISTS idx_scheduled_events_status ON scheduled_events(status)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// 添加缺失的列（如果不存在）
	alterTableMigrations := []string{
		`ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS sport VARCHAR(100)`,
		`ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS sport_name VARCHAR(100)`,
		`ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_team_id VARCHAR(100)`,
		`ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_team_name VARCHAR(255)`,
		`ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_team_id VARCHAR(100)`,
		`ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_team_name VARCHAR(255)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_id ON tracked_events(sport_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_home_team_id ON tracked_events(home_team_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_away_team_id ON tracked_events(away_team_id)`,
	}

	for _, migration := range alterTableMigrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("alter table migration failed: %w", err)
		}
	}

	return nil
}

