package database

import (
	"database/sql"
	"fmt"
	"log"

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
// 此函数由统一 schema 自动生成，基于 SportRader UOF API 规范
func Migrate(db *sql.DB) error {
	tables := []string{
		// 运动类型
		`CREATE TABLE IF NOT EXISTS sports (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 类别/国家
		`CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(50) PRIMARY KEY,
    sport_id VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    country_code VARCHAR(10),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 锦标赛
		`CREATE TABLE IF NOT EXISTS tournaments (
    id VARCHAR(50) PRIMARY KEY,
    sport_id VARCHAR(50) NOT NULL,
    category_id VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 作废原因
		`CREATE TABLE IF NOT EXISTS void_reasons (
    id INT PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 停止投注原因
		`CREATE TABLE IF NOT EXISTS betstop_reasons (
    id INT PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 球员信息
		`CREATE TABLE IF NOT EXISTS players (
    player_id VARCHAR(50) PRIMARY KEY,
    player_name VARCHAR(200) NOT NULL,
    nationality VARCHAR(10),
    date_of_birth DATE,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 跟踪的体育赛事
		`CREATE TABLE IF NOT EXISTS tracked_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL UNIQUE,
    srn_id VARCHAR(200),
    sport_id VARCHAR(50),
    sport VARCHAR(50),
    sport_name VARCHAR(100),
    category_id VARCHAR(50),
    category_name VARCHAR(200),
    tournament_id VARCHAR(50),
    tournament_name VARCHAR(200),
    home_team_id VARCHAR(100),
    home_team_name VARCHAR(255),
    away_team_id VARCHAR(100),
    away_team_name VARCHAR(255),
    home_score INTEGER,
    away_score INTEGER,
    schedule_time TIMESTAMP,
    match_status VARCHAR(50),
    match_time VARCHAR(50),
    status VARCHAR(50) DEFAULT 'active',
    subscribed BOOLEAN DEFAULT FALSE,
    message_count INTEGER DEFAULT 0,
    last_message_at TIMESTAMP,
    attendance INTEGER,
    sellout BOOLEAN DEFAULT FALSE,
    feature_match BOOLEAN DEFAULT FALSE,
    live_video_available BOOLEAN DEFAULT FALSE,
    live_data_available BOOLEAN DEFAULT FALSE,
    broadcasts_count INTEGER DEFAULT 0,
    broadcasts_data JSONB,
    social_links JSONB,
    ticket_url TEXT,
    popularity_score DECIMAL(5,2) DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 赛程表
		`CREATE TABLE IF NOT EXISTS scheduled_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL UNIQUE,
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
);`,
		
		// 盘口表
		`CREATE TABLE IF NOT EXISTS markets (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    sr_market_id VARCHAR(200) NOT NULL,
    market_type VARCHAR(100),
    market_name VARCHAR(200),
    specifiers TEXT,
    status VARCHAR(50),
    producer_id INTEGER,
    favourite BOOLEAN,
    home_team_name VARCHAR(200),
    away_team_name VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers)
);`,
		
		// 赔率表 (当前最新赔率)
		`CREATE TABLE IF NOT EXISTS odds (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL,
    event_id VARCHAR(100) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name VARCHAR(200),
    odds_value DECIMAL(10, 2),
    probability DECIMAL(5, 4),
    active BOOLEAN DEFAULT TRUE,
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (market_id, outcome_id)
);`,
		
		// 赔率历史表
		`CREATE TABLE IF NOT EXISTS odds_history (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL,
    event_id VARCHAR(100) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name VARCHAR(200),
    odds_value DECIMAL(10, 2),
    probability DECIMAL(5, 4),
    change_type VARCHAR(20),
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 结果表 (静态定义)
		`CREATE TABLE IF NOT EXISTS outcomes (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name VARCHAR(200),
    specifiers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// UOF 原始消息存储
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
);`,
		
		// 赔率变化记录
		`CREATE TABLE IF NOT EXISTS odds_changes (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    odds_change_reason VARCHAR(50),
    markets_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 投注停止记录
		`CREATE TABLE IF NOT EXISTS bet_stops (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    groups VARCHAR(200),
    reason VARCHAR(200),
    market_status VARCHAR(50),
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 投注结算记录
		`CREATE TABLE IF NOT EXISTS bet_settlements (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER,
    product_id INTEGER,
    timestamp BIGINT,
    certainty INTEGER,
    sr_market_id VARCHAR(200),
    specifiers TEXT,
    outcome_id VARCHAR(200),
    void_factor DECIMAL(3, 2),
    dead_heat_factor DECIMAL(10, 8),
    result VARCHAR(20),
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 投注取消记录
		`CREATE TABLE IF NOT EXISTS bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER,
    product_id INTEGER,
    timestamp BIGINT,
    sr_market_id VARCHAR(200),
    specifiers TEXT,
    void_reason VARCHAR(200),
    start_time BIGINT,
    end_time BIGINT,
    superceded_by VARCHAR(100),
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 结算回滚记录
		`CREATE TABLE IF NOT EXISTS rollback_bet_settlements (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER,
    product_id INTEGER,
    timestamp BIGINT,
    sr_market_id VARCHAR(200),
    specifiers TEXT,
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 取消回滚记录
		`CREATE TABLE IF NOT EXISTS rollback_bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER,
    product_id INTEGER,
    timestamp BIGINT,
    sr_market_id VARCHAR(200),
    specifiers TEXT,
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 生产者状态监控
		`CREATE TABLE IF NOT EXISTS producer_status (
    id BIGSERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'unknown',
    last_alive BIGINT NOT NULL,
    subscribed INTEGER DEFAULT 0,
    recovery_id BIGINT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 恢复状态记录
		`CREATE TABLE IF NOT EXISTS recovery_status (
    id BIGSERIAL PRIMARY KEY,
    request_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'initiated',
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);`,
		
		// 盘口描述缓存
		`CREATE TABLE IF NOT EXISTS market_descriptions (
    market_id VARCHAR(50) PRIMARY KEY,
    market_name TEXT NOT NULL,
    groups TEXT,
    specifiers TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`,
		
		// 结果描述缓存
		`CREATE TABLE IF NOT EXISTS outcome_descriptions (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(50) NOT NULL,
    outcome_id VARCHAR(50) NOT NULL,
    outcome_name TEXT NOT NULL,
    UNIQUE (market_id, outcome_id)
);`,
		
		// 结果映射表
		`CREATE TABLE IF NOT EXISTS mapping_outcomes (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(50) NOT NULL,
    outcome_id VARCHAR(50) NOT NULL,
    product_outcome_name TEXT,
    product_id INTEGER,
    sport_id VARCHAR(50),
    UNIQUE (market_id, outcome_id)
);`,
		
	}
	
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}
	
	// 创建索引以提高查询性能
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_event_id ON tracked_events(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_id ON tracked_events(sport_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_schedule_time ON tracked_events(schedule_time)`,
		`CREATE INDEX IF NOT EXISTS idx_tracked_events_subscribed ON tracked_events(subscribed)`,
		
		`CREATE INDEX IF NOT EXISTS idx_markets_event_id ON markets(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_markets_sr_market_id ON markets(sr_market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status)`,
		
		`CREATE INDEX IF NOT EXISTS idx_odds_market_id ON odds(market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_odds_event_id ON odds(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_odds_active ON odds(active)`,
		
		`CREATE INDEX IF NOT EXISTS idx_odds_history_market_id ON odds_history(market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_odds_history_event_id ON odds_history(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_odds_history_timestamp ON odds_history(timestamp)`,
		
		`CREATE INDEX IF NOT EXISTS idx_uof_messages_event_id ON uof_messages(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_uof_messages_message_type ON uof_messages(message_type)`,
		`CREATE INDEX IF NOT EXISTS idx_uof_messages_received_at ON uof_messages(received_at)`,
		
		`CREATE INDEX IF NOT EXISTS idx_bet_settlements_event_id ON bet_settlements(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bet_cancels_event_id ON bet_cancels(event_id)`,
		
		`CREATE INDEX IF NOT EXISTS idx_categories_sport_id ON categories(sport_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tournaments_sport_id ON tournaments(sport_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tournaments_category_id ON tournaments(category_id)`,
		
		`CREATE INDEX IF NOT EXISTS idx_scheduled_events_event_id ON scheduled_events(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_events_sport_id ON scheduled_events(sport_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_events_scheduled_time ON scheduled_events(scheduled_time)`,
		
		`CREATE INDEX IF NOT EXISTS idx_recovery_status_request_id ON recovery_status(request_id)`,
		`CREATE INDEX IF NOT EXISTS idx_recovery_status_product_id ON recovery_status(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_recovery_status_status ON recovery_status(status)`,
	}
	
	for _, sql := range indexes {
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	
	log.Println("✅ Database migration completed successfully - All tables and indexes created")
	return nil
}
