package processing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
)

// PostgreSQLStorage PostgreSQL 存储实现
type PostgreSQLStorage struct {
	db     *sql.DB
	logger common.Logger
}

// NewPostgreSQLStorage 创建 PostgreSQL 存储
func NewPostgreSQLStorage(db *sql.DB, logger common.Logger) DataStorage {
	return &PostgreSQLStorage{
		db:     db,
		logger: logger,
	}
}

// SaveEvent 保存事件
func (s *PostgreSQLStorage) SaveEvent(ctx context.Context, event *models.Event) error {
	s.logger.Debug("Saving event: %s", event.ID)

	// 将 Data 转换为 JSON
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO events (id, type, source, timestamp, match_id, sport_id, data, raw_data, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			source = EXCLUDED.source,
			timestamp = EXCLUDED.timestamp,
			match_id = EXCLUDED.match_id,
			sport_id = EXCLUDED.sport_id,
			data = EXCLUDED.data,
			raw_data = EXCLUDED.raw_data,
			processed_at = EXCLUDED.processed_at
	`

	_, err = s.db.ExecContext(ctx, query,
		event.ID,
		event.Type,
		event.Source,
		event.Timestamp,
		event.MatchID,
		event.SportID,
		dataJSON,
		event.RawData,
		event.ProcessedAt,
	)

	if err != nil {
		s.logger.Error("Failed to save event: %v", err)
		return common.NewAppError("STORAGE_FAILED", "Failed to save event", err)
	}

	s.logger.Debug("Event saved successfully: %s", event.ID)
	return nil
}

// SaveMatch 保存比赛
func (s *PostgreSQLStorage) SaveMatch(ctx context.Context, match *models.Match) error {
	s.logger.Debug("Saving match: %s", match.ID)

	// 将 Statistics 转换为 JSON
	statsJSON, err := json.Marshal(match.Statistics)
	if err != nil {
		return fmt.Errorf("failed to marshal match statistics: %w", err)
	}

	query := `
		INSERT INTO matches (id, sport_id, sport_name, home_team_id, home_team_name, away_team_id, away_team_name, 
			status, start_time, home_score, away_score, statistics, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			sport_id = EXCLUDED.sport_id,
			sport_name = EXCLUDED.sport_name,
			home_team_id = EXCLUDED.home_team_id,
			home_team_name = EXCLUDED.home_team_name,
			away_team_id = EXCLUDED.away_team_id,
			away_team_name = EXCLUDED.away_team_name,
			status = EXCLUDED.status,
			start_time = EXCLUDED.start_time,
			home_score = EXCLUDED.home_score,
			away_score = EXCLUDED.away_score,
			statistics = EXCLUDED.statistics,
			source = EXCLUDED.source,
			updated_at = EXCLUDED.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		match.ID,
		match.SportID,
		match.SportName,
		match.HomeTeam.ID,
		match.HomeTeam.Name,
		match.AwayTeam.ID,
		match.AwayTeam.Name,
		match.Status,
		match.StartTime,
		match.Score.Home,
		match.Score.Away,
		statsJSON,
		match.Source,
		match.CreatedAt,
		match.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to save match: %v", err)
		return common.NewAppError("STORAGE_FAILED", "Failed to save match", err)
	}

	s.logger.Debug("Match saved successfully: %s", match.ID)
	return nil
}

// SaveOdds 保存赔率
func (s *PostgreSQLStorage) SaveOdds(ctx context.Context, odds *models.Odds) error {
	s.logger.Debug("Saving odds: %s", odds.ID)

	// 将 Outcomes 转换为 JSON
	outcomesJSON, err := json.Marshal(odds.Outcomes)
	if err != nil {
		return fmt.Errorf("failed to marshal odds outcomes: %w", err)
	}

	query := `
		INSERT INTO odds (id, match_id, market_id, market_name, outcomes, status, source, timestamp, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			match_id = EXCLUDED.match_id,
			market_id = EXCLUDED.market_id,
			market_name = EXCLUDED.market_name,
			outcomes = EXCLUDED.outcomes,
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			timestamp = EXCLUDED.timestamp,
			updated_at = EXCLUDED.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		odds.ID,
		odds.MatchID,
		odds.MarketID,
		odds.MarketName,
		outcomesJSON,
		odds.Status,
		odds.Source,
		odds.Timestamp,
		odds.CreatedAt,
		odds.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to save odds: %v", err)
		return common.NewAppError("STORAGE_FAILED", "Failed to save odds", err)
	}

	s.logger.Debug("Odds saved successfully: %s", odds.ID)
	return nil
}

// GetEvent 获取事件
func (s *PostgreSQLStorage) GetEvent(ctx context.Context, eventID string) (*models.Event, error) {
	s.logger.Debug("Getting event: %s", eventID)

	query := `
		SELECT id, type, source, timestamp, match_id, sport_id, data, raw_data, processed_at
		FROM events
		WHERE id = $1
	`

	var event models.Event
	var dataJSON []byte

	err := s.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID,
		&event.Type,
		&event.Source,
		&event.Timestamp,
		&event.MatchID,
		&event.SportID,
		&dataJSON,
		&event.RawData,
		&event.ProcessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, common.ErrNotFound
	}

	if err != nil {
		s.logger.Error("Failed to get event: %v", err)
		return nil, common.NewAppError("STORAGE_FAILED", "Failed to get event", err)
	}

	// 解析 JSON 数据
	if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return &event, nil
}

// GetMatch 获取比赛
func (s *PostgreSQLStorage) GetMatch(ctx context.Context, matchID string) (*models.Match, error) {
	s.logger.Debug("Getting match: %s", matchID)

	query := `
		SELECT id, sport_id, sport_name, home_team_id, home_team_name, away_team_id, away_team_name,
			status, start_time, home_score, away_score, statistics, source, created_at, updated_at
		FROM matches
		WHERE id = $1
	`

	var match models.Match
	var statsJSON []byte

	err := s.db.QueryRowContext(ctx, query, matchID).Scan(
		&match.ID,
		&match.SportID,
		&match.SportName,
		&match.HomeTeam.ID,
		&match.HomeTeam.Name,
		&match.AwayTeam.ID,
		&match.AwayTeam.Name,
		&match.Status,
		&match.StartTime,
		&match.Score.Home,
		&match.Score.Away,
		&statsJSON,
		&match.Source,
		&match.CreatedAt,
		&match.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, common.ErrNotFound
	}

	if err != nil {
		s.logger.Error("Failed to get match: %v", err)
		return nil, common.NewAppError("STORAGE_FAILED", "Failed to get match", err)
	}

	// 解析 JSON 数据
	if err := json.Unmarshal(statsJSON, &match.Statistics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal match statistics: %w", err)
	}

	return &match, nil
}

// GetOdds 获取赔率
func (s *PostgreSQLStorage) GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error) {
	s.logger.Debug("Getting odds for match: %s", matchID)

	query := `
		SELECT id, match_id, market_id, market_name, outcomes, status, source, timestamp, created_at, updated_at
		FROM odds
		WHERE match_id = $1
		ORDER BY timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, matchID)
	if err != nil {
		s.logger.Error("Failed to get odds: %v", err)
		return nil, common.NewAppError("STORAGE_FAILED", "Failed to get odds", err)
	}
	defer rows.Close()

	var oddsList []*models.Odds
	for rows.Next() {
		var odds models.Odds
		var outcomesJSON []byte

		err := rows.Scan(
			&odds.ID,
			&odds.MatchID,
			&odds.MarketID,
			&odds.MarketName,
			&outcomesJSON,
			&odds.Status,
			&odds.Source,
			&odds.Timestamp,
			&odds.CreatedAt,
			&odds.UpdatedAt,
		)

		if err != nil {
			s.logger.Error("Failed to scan odds: %v", err)
			continue
		}

		// 解析 JSON 数据
		if err := json.Unmarshal(outcomesJSON, &odds.Outcomes); err != nil {
			s.logger.Error("Failed to unmarshal odds outcomes: %v", err)
			continue
		}

		oddsList = append(oddsList, &odds)
	}

	return oddsList, nil
}

// QueryEvents 查询事件
func (s *PostgreSQLStorage) QueryEvents(ctx context.Context, filter EventQueryFilter) ([]*models.Event, error) {
	s.logger.Debug("Querying events with filter: %+v", filter)

	query := `
		SELECT id, type, source, timestamp, match_id, sport_id, data, raw_data, processed_at
		FROM events
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.MatchID != "" {
		query += fmt.Sprintf(" AND match_id = $%d", argIndex)
		args = append(args, filter.MatchID)
		argIndex++
	}

	if filter.EventType != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, filter.EventType)
		argIndex++
	}

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIndex)
		args = append(args, filter.Source)
		argIndex++
	}

	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filter.StartTime)
		argIndex++
	}

	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filter.EndTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to query events: %v", err)
		return nil, common.NewAppError("STORAGE_FAILED", "Failed to query events", err)
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		var event models.Event
		var dataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Source,
			&event.Timestamp,
			&event.MatchID,
			&event.SportID,
			&dataJSON,
			&event.RawData,
			&event.ProcessedAt,
		)

		if err != nil {
			s.logger.Error("Failed to scan event: %v", err)
			continue
		}

		// 解析 JSON 数据
		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			s.logger.Error("Failed to unmarshal event data: %v", err)
			continue
		}

		events = append(events, &event)
	}

	return events, nil
}

// QueryMatches 查询比赛
func (s *PostgreSQLStorage) QueryMatches(ctx context.Context, filter MatchQueryFilter) ([]*models.Match, error) {
	s.logger.Debug("Querying matches with filter: %+v", filter)

	query := `
		SELECT id, sport_id, sport_name, home_team_id, home_team_name, away_team_id, away_team_name,
			status, start_time, home_score, away_score, statistics, source, created_at, updated_at
		FROM matches
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.SportID != "" {
		query += fmt.Sprintf(" AND sport_id = $%d", argIndex)
		args = append(args, filter.SportID)
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIndex)
		args = append(args, filter.Source)
		argIndex++
	}

	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND start_time >= $%d", argIndex)
		args = append(args, filter.StartTime)
		argIndex++
	}

	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND start_time <= $%d", argIndex)
		args = append(args, filter.EndTime)
		argIndex++
	}

	query += " ORDER BY start_time DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to query matches: %v", err)
		return nil, common.NewAppError("STORAGE_FAILED", "Failed to query matches", err)
	}
	defer rows.Close()

	var matches []*models.Match
	for rows.Next() {
		var match models.Match
		var statsJSON []byte

		err := rows.Scan(
			&match.ID,
			&match.SportID,
			&match.SportName,
			&match.HomeTeam.ID,
			&match.HomeTeam.Name,
			&match.AwayTeam.ID,
			&match.AwayTeam.Name,
			&match.Status,
			&match.StartTime,
			&match.Score.Home,
			&match.Score.Away,
			&statsJSON,
			&match.Source,
			&match.CreatedAt,
			&match.UpdatedAt,
		)

		if err != nil {
			s.logger.Error("Failed to scan match: %v", err)
			continue
		}

		// 解析 JSON 数据
		if err := json.Unmarshal(statsJSON, &match.Statistics); err != nil {
			s.logger.Error("Failed to unmarshal match statistics: %v", err)
			continue
		}

		matches = append(matches, &match)
	}

	return matches, nil
}

// EventQueryFilter 事件查询过滤器
type EventQueryFilter struct {
	MatchID   string
	EventType string
	Source    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// MatchQueryFilter 比赛查询过滤器
type MatchQueryFilter struct {
	SportID   string
	Status    string
	Source    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

