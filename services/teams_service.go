package services

import (
	"database/sql"
	"fmt"
	"time"
	"uof-service/database"
	"uof-service/logger"
)

// TeamsService 队伍管理服务
type TeamsService struct {
	db *sql.DB
}

// NewTeamsService 创建新的队伍服务实例
func NewTeamsService(db *sql.DB) *TeamsService {
	return &TeamsService{
		db: db,
	}
}

// TeamInfo 队伍信息结构
type TeamInfo struct {
	TeamID       string
	TeamName     string
	SportID      string
	SportName    string
	CategoryID   string
	CategoryName string
}

// GetOrCreateTeam 获取或创建队伍记录
// 如果队伍不存在，则创建新记录并返回 true 表示是新队伍
func (s *TeamsService) GetOrCreateTeam(teamInfo TeamInfo) (team *database.Team, isNew bool, err error) {
	// 首先尝试查询队伍是否存在
	team, err = s.GetTeamByID(teamInfo.TeamID)
	if err == nil {
		// 队伍已存在
		return team, false, nil
	}
	
	if err != sql.ErrNoRows {
		// 查询出错
		return nil, false, fmt.Errorf("failed to query team: %w", err)
	}
	
	// 队伍不存在，创建新记录
	team, err = s.CreateTeam(teamInfo)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create team: %w", err)
	}
	
	logger.Printf("[TeamsService] ✅ Created new team: %s (ID: %s)", teamInfo.TeamName, teamInfo.TeamID)
	return team, true, nil
}

// GetTeamByID 根据 team_id 查询队伍
func (s *TeamsService) GetTeamByID(teamID string) (*database.Team, error) {
	query := `
		SELECT id, team_id, team_name, sport_id, sport_name, category_id, category_name,
		       logo_url, logo_fetched, logo_fetch_attempted_at, logo_fetch_retry_count,
		       created_at, updated_at
		FROM teams
		WHERE team_id = $1
	`
	
	var team database.Team
	err := s.db.QueryRow(query, teamID).Scan(
		&team.ID,
		&team.TeamID,
		&team.TeamName,
		&team.SportID,
		&team.SportName,
		&team.CategoryID,
		&team.CategoryName,
		&team.LogoURL,
		&team.LogoFetched,
		&team.LogoFetchAttemptedAt,
		&team.LogoFetchRetryCount,
		&team.CreatedAt,
		&team.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &team, nil
}

// CreateTeam 创建新的队伍记录
func (s *TeamsService) CreateTeam(teamInfo TeamInfo) (*database.Team, error) {
	query := `
		INSERT INTO teams (team_id, team_name, sport_id, sport_name, category_id, category_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, team_id, team_name, sport_id, sport_name, category_id, category_name,
		          logo_url, logo_fetched, logo_fetch_attempted_at, logo_fetch_retry_count,
		          created_at, updated_at
	`
	
	now := time.Now()
	var team database.Team
	
	err := s.db.QueryRow(
		query,
		teamInfo.TeamID,
		teamInfo.TeamName,
		teamInfo.SportID,
		teamInfo.SportName,
		teamInfo.CategoryID,
		teamInfo.CategoryName,
		now,
		now,
	).Scan(
		&team.ID,
		&team.TeamID,
		&team.TeamName,
		&team.SportID,
		&team.SportName,
		&team.CategoryID,
		&team.CategoryName,
		&team.LogoURL,
		&team.LogoFetched,
		&team.LogoFetchAttemptedAt,
		&team.LogoFetchRetryCount,
		&team.CreatedAt,
		&team.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &team, nil
}

// UpdateTeamLogo 更新队伍的 Logo URL
func (s *TeamsService) UpdateTeamLogo(teamID string, logoURL string, success bool) error {
	query := `
		UPDATE teams
		SET logo_url = $1,
		    logo_fetched = $2,
		    logo_fetch_attempted_at = $3,
		    logo_fetch_retry_count = CASE WHEN $2 = true THEN 0 ELSE logo_fetch_retry_count + 1 END,
		    updated_at = $4
		WHERE team_id = $5
	`
	
	now := time.Now()
	_, err := s.db.Exec(query, logoURL, success, now, now, teamID)
	if err != nil {
		return fmt.Errorf("failed to update team logo: %w", err)
	}
	
	if success {
		logger.Printf("[TeamsService] ✅ Updated logo for team %s: %s", teamID, logoURL)
	} else {
		logger.Printf("[TeamsService] ⚠️  Failed to fetch logo for team %s", teamID)
	}
	
	return nil
}

// GetTeamsNeedingLogoFetch 获取需要获取 Logo 的队伍列表
// 条件：logo_fetched = false 且 (logo_fetch_attempted_at 为空 或 重试次数 < 最大重试次数)
func (s *TeamsService) GetTeamsNeedingLogoFetch(maxRetries int, limit int) ([]*database.Team, error) {
	query := `
		SELECT id, team_id, team_name, sport_id, sport_name, category_id, category_name,
		       logo_url, logo_fetched, logo_fetch_attempted_at, logo_fetch_retry_count,
		       created_at, updated_at
		FROM teams
		WHERE logo_fetched = false
		  AND (logo_fetch_attempted_at IS NULL OR logo_fetch_retry_count < $1)
		ORDER BY created_at ASC
		LIMIT $2
	`
	
	rows, err := s.db.Query(query, maxRetries, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query teams needing logo fetch: %w", err)
	}
	defer rows.Close()
	
	var teams []*database.Team
	for rows.Next() {
		var team database.Team
		err := rows.Scan(
			&team.ID,
			&team.TeamID,
			&team.TeamName,
			&team.SportID,
			&team.SportName,
			&team.CategoryID,
			&team.CategoryName,
			&team.LogoURL,
			&team.LogoFetched,
			&team.LogoFetchAttemptedAt,
			&team.LogoFetchRetryCount,
			&team.CreatedAt,
			&team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, &team)
	}
	
	return teams, nil
}
