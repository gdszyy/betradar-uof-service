package business

import (
	"context"
	"fmt"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
	"uof-service/pkg/processing"
)

// DefaultMatchService 默认赛事管理服务实现
type DefaultMatchService struct {
	logger  common.Logger
	storage processing.DataStorage
}

// NewMatchService 创建赛事管理服务
func NewMatchService(logger common.Logger, storage processing.DataStorage) MatchService {
	return &DefaultMatchService{
		logger:  logger,
		storage: storage,
	}
}

// GetMatch 获取比赛信息
func (s *DefaultMatchService) GetMatch(ctx context.Context, matchID string) (*models.Match, error) {
	s.logger.Debug("Getting match: %s", matchID)

	match, err := s.storage.GetMatch(ctx, matchID)
	if err != nil {
		s.logger.Error("Failed to get match: %v", err)
		return nil, err
	}

	s.logger.Debug("Retrieved match: %s", matchID)
	return match, nil
}

// UpdateMatch 更新比赛信息
func (s *DefaultMatchService) UpdateMatch(ctx context.Context, match *models.Match) error {
	s.logger.Debug("Updating match: %s", match.ID)

	// 设置更新时间
	match.UpdatedAt = time.Now()

	if err := s.storage.SaveMatch(ctx, match); err != nil {
		s.logger.Error("Failed to update match: %v", err)
		return err
	}

	s.logger.Debug("Match updated successfully: %s", match.ID)
	return nil
}

// GetLiveMatches 获取直播比赛列表
func (s *DefaultMatchService) GetLiveMatches(ctx context.Context) ([]*models.Match, error) {
	s.logger.Debug("Getting live matches")

	filter := processing.MatchQueryFilter{
		Status: models.MatchStatusLive,
		Limit:  100,
	}

	matches, err := s.storage.QueryMatches(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get live matches: %v", err)
		return nil, err
	}

	s.logger.Debug("Retrieved %d live matches", len(matches))
	return matches, nil
}

// GetUpcomingMatches 获取即将开始的比赛
func (s *DefaultMatchService) GetUpcomingMatches(ctx context.Context, hours int) ([]*models.Match, error) {
	s.logger.Debug("Getting upcoming matches (next %d hours)", hours)

	now := time.Now()
	endTime := now.Add(time.Duration(hours) * time.Hour)

	filter := processing.MatchQueryFilter{
		Status:    models.MatchStatusNotStarted,
		StartTime: now,
		EndTime:   endTime,
		Limit:     100,
	}

	matches, err := s.storage.QueryMatches(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get upcoming matches: %v", err)
		return nil, err
	}

	s.logger.Debug("Retrieved %d upcoming matches", len(matches))
	return matches, nil
}

// GetMatchesBySport 获取指定运动的比赛
func (s *DefaultMatchService) GetMatchesBySport(ctx context.Context, sportID string) ([]*models.Match, error) {
	s.logger.Debug("Getting matches for sport: %s", sportID)

	filter := processing.MatchQueryFilter{
		SportID: sportID,
		Limit:   100,
	}

	matches, err := s.storage.QueryMatches(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get matches by sport: %v", err)
		return nil, err
	}

	s.logger.Debug("Retrieved %d matches for sport: %s", len(matches), sportID)
	return matches, nil
}

// GetMatchesByDate 获取指定日期的比赛
func (s *DefaultMatchService) GetMatchesByDate(ctx context.Context, date time.Time) ([]*models.Match, error) {
	s.logger.Debug("Getting matches for date: %s", date.Format("2006-01-02"))

	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := startTime.Add(24 * time.Hour)

	filter := processing.MatchQueryFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     500,
	}

	matches, err := s.storage.QueryMatches(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get matches by date: %v", err)
		return nil, err
	}

	s.logger.Debug("Retrieved %d matches for date: %s", len(matches), date.Format("2006-01-02"))
	return matches, nil
}

// GetMatchStatistics 获取比赛统计
func (s *DefaultMatchService) GetMatchStatistics(ctx context.Context, matchID string) (map[string]interface{}, error) {
	s.logger.Debug("Getting match statistics: %s", matchID)

	match, err := s.storage.GetMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	return match.Statistics, nil
}

// UpdateMatchStatus 更新比赛状态
func (s *DefaultMatchService) UpdateMatchStatus(ctx context.Context, matchID string, status string) error {
	s.logger.Debug("Updating match status: %s -> %s", matchID, status)

	match, err := s.storage.GetMatch(ctx, matchID)
	if err != nil {
		return err
	}

	match.Status = status
	match.UpdatedAt = time.Now()

	if err := s.storage.SaveMatch(ctx, match); err != nil {
		s.logger.Error("Failed to update match status: %v", err)
		return err
	}

	s.logger.Debug("Match status updated successfully: %s", matchID)
	return nil
}

// UpdateMatchScore 更新比赛比分
func (s *DefaultMatchService) UpdateMatchScore(ctx context.Context, matchID string, homeScore, awayScore int) error {
	s.logger.Debug("Updating match score: %s -> %d:%d", matchID, homeScore, awayScore)

	match, err := s.storage.GetMatch(ctx, matchID)
	if err != nil {
		return err
	}

	match.Score.Home = homeScore
	match.Score.Away = awayScore
	match.UpdatedAt = time.Now()

	if err := s.storage.SaveMatch(ctx, match); err != nil {
		s.logger.Error("Failed to update match score: %v", err)
		return err
	}

	s.logger.Debug("Match score updated successfully: %s", matchID)
	return nil
}

// SearchMatches 搜索比赛
func (s *DefaultMatchService) SearchMatches(ctx context.Context, query MatchSearchQuery) ([]*models.Match, error) {
	s.logger.Debug("Searching matches with query: %+v", query)

	filter := processing.MatchQueryFilter{
		SportID:   query.SportID,
		Status:    query.Status,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Limit:     query.Limit,
	}

	if filter.Limit == 0 {
		filter.Limit = 50
	}

	matches, err := s.storage.QueryMatches(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to search matches: %v", err)
		return nil, err
	}

	// 过滤球队名称
	if query.TeamName != "" {
		filteredMatches := make([]*models.Match, 0)
		for _, match := range matches {
			if containsTeam(match, query.TeamName) {
				filteredMatches = append(filteredMatches, match)
			}
		}
		matches = filteredMatches
	}

	s.logger.Debug("Found %d matches", len(matches))
	return matches, nil
}

// GetMatchSummary 获取比赛摘要
func (s *DefaultMatchService) GetMatchSummary(ctx context.Context, matchID string) (*MatchSummary, error) {
	s.logger.Debug("Getting match summary: %s", matchID)

	match, err := s.storage.GetMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	summary := &MatchSummary{
		MatchID:      match.ID,
		SportID:      match.SportID,
		SportName:    match.SportName,
		HomeTeam:     match.HomeTeam.Name,
		AwayTeam:     match.AwayTeam.Name,
		Status:       match.Status,
		StartTime:    match.StartTime,
		Score:        fmt.Sprintf("%d:%d", match.Score.Home, match.Score.Away),
		Source:       match.Source,
	}

	return summary, nil
}

// containsTeam 检查比赛是否包含指定球队
func containsTeam(match *models.Match, teamName string) bool {
	return match.HomeTeam.Name == teamName || match.AwayTeam.Name == teamName
}

// MatchSearchQuery 比赛搜索查询
type MatchSearchQuery struct {
	SportID   string
	Status    string
	TeamName  string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// MatchSummary 比赛摘要
type MatchSummary struct {
	MatchID   string
	SportID   string
	SportName string
	HomeTeam  string
	AwayTeam  string
	Status    string
	StartTime time.Time
	Score     string
	Source    string
}

