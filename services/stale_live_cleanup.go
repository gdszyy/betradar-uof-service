package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	"uof-service/config"
	"uof-service/logger"
)

// StaleLiveCleanupService 过期live比赛清理服务
// 处理断联期间已结束但仍为live状态的比赛
type StaleLiveCleanupService struct {
	config       *config.Config
	db           *sql.DB
	client       *http.Client
	larkNotifier *LarkNotifier
}

// NewStaleLiveCleanupService 创建过期live比赛清理服务
func NewStaleLiveCleanupService(cfg *config.Config, db *sql.DB, notifier *LarkNotifier) *StaleLiveCleanupService {
	return &StaleLiveCleanupService{
		config:       cfg,
		db:           db,
		client:       &http.Client{Timeout: 30 * time.Second},
		larkNotifier: notifier,
	}
}

// StaleLiveCleanupResult 清理结果
type StaleLiveCleanupResult struct {
	TotalLive       int
	TwoDaysOld      int
	OneDayOld       int
	Deleted         int
	Updated         int
	Failed          int
	DeletedList     []string
	UpdatedList     []string
	FailedList      map[string]string
}

// StaleLiveMatch 过期的live比赛
type StaleLiveMatch struct {
	EventID      string
	ScheduleTime time.Time
	DaysOld      int
}

// ExecuteCleanup 执行清理
func (s *StaleLiveCleanupService) ExecuteCleanup() (*StaleLiveCleanupResult, error) {
	logger.Println("[StaleLiveCleanup] 🧹 Starting stale live matches cleanup...")
	
	result := &StaleLiveCleanupResult{
		DeletedList: []string{},
		UpdatedList: []string{},
		FailedList:  make(map[string]string),
	}
	
	// 1. 查询所有live状态的比赛
	liveMatches, err := s.queryLiveMatches()
	if err != nil {
		return nil, fmt.Errorf("failed to query live matches: %w", err)
	}
	
	result.TotalLive = len(liveMatches)
	logger.Printf("[StaleLiveCleanup] 📊 Found %d live matches", result.TotalLive)
	
	if result.TotalLive == 0 {
		logger.Println("[StaleLiveCleanup] ℹ️  No live matches to check")
		return result, nil
	}
	
	// 2. 分类处理
	now := time.Now()
	for _, match := range liveMatches {
		daysSince := int(now.Sub(match.ScheduleTime).Hours() / 24)
		
		if daysSince >= 2 {
			// 开赛时间与今天相差两天或以上：直接删除
			result.TwoDaysOld++
			if err := s.deleteMatch(match.EventID); err != nil {
				logger.Printf("[StaleLiveCleanup] ❌ Failed to delete %s: %v", match.EventID, err)
				result.Failed++
				result.FailedList[match.EventID] = fmt.Sprintf("delete failed: %v", err)
			} else {
				logger.Printf("[StaleLiveCleanup] ✅ Deleted %s (scheduled %d days ago)", match.EventID, daysSince)
				result.Deleted++
				result.DeletedList = append(result.DeletedList, match.EventID)
			}
			time.Sleep(200 * time.Millisecond)
			
		} else if daysSince >= 1 {
			// 开赛时间与今天相差一天：使用summary接口查询最新状态
			result.OneDayOld++
			if err := s.updateMatchStatus(match.EventID); err != nil {
				logger.Printf("[StaleLiveCleanup] ❌ Failed to update %s: %v", match.EventID, err)
				result.Failed++
				result.FailedList[match.EventID] = fmt.Sprintf("update failed: %v", err)
			} else {
				logger.Printf("[StaleLiveCleanup] ✅ Updated %s (scheduled %d days ago)", match.EventID, daysSince)
				result.Updated++
				result.UpdatedList = append(result.UpdatedList, match.EventID)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	logger.Printf("[StaleLiveCleanup] 📈 Cleanup completed: %d deleted, %d updated, %d failed", 
		result.Deleted, result.Updated, result.Failed)
	
	// 3. 发送飞书通知
	s.sendCleanupReport(result)
	
	return result, nil
}

// queryLiveMatches 查询所有live状态的比赛
func (s *StaleLiveCleanupService) queryLiveMatches() ([]StaleLiveMatch, error) {
	// 查询status为live或active，且有schedule_time的比赛
	query := `
		SELECT event_id, schedule_time
		FROM tracked_events
		WHERE status IN ('live', 'active')
		  AND schedule_time IS NOT NULL
		ORDER BY schedule_time ASC
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()
	
	var matches []StaleLiveMatch
	for rows.Next() {
		var match StaleLiveMatch
		if err := rows.Scan(&match.EventID, &match.ScheduleTime); err != nil {
			logger.Printf("[StaleLiveCleanup] ⚠️  Failed to scan row: %v", err)
			continue
		}
		matches = append(matches, match)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return matches, nil
}

// deleteMatch 删除比赛记录
func (s *StaleLiveCleanupService) deleteMatch(eventID string) error {
	// 从tracked_events删除
	_, err := s.db.Exec("DELETE FROM tracked_events WHERE event_id = $1", eventID)
	if err != nil {
		return fmt.Errorf("failed to delete from tracked_events: %w", err)
	}
	
	// 同时删除相关的markets和outcomes
	_, err = s.db.Exec("DELETE FROM markets WHERE event_id = $1", eventID)
	if err != nil {
		logger.Printf("[StaleLiveCleanup] ⚠️  Failed to delete markets for %s: %v", eventID, err)
	}
	
	_, err = s.db.Exec("DELETE FROM outcomes WHERE event_id = $1", eventID)
	if err != nil {
		logger.Printf("[StaleLiveCleanup] ⚠️  Failed to delete outcomes for %s: %v", eventID, err)
	}
	
	return nil
}

// updateMatchStatus 使用summary接口更新比赛状态
func (s *StaleLiveCleanupService) updateMatchStatus(eventID string) error {
	// API: /sports/{language}/sport_events/{urn_type}:{id}/summary.xml
	url := fmt.Sprintf("%s/sports/en/sport_events/%s/summary.xml", s.config.APIBaseURL, eventID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		// 比赛不存在，直接删除
		logger.Printf("[StaleLiveCleanup] ℹ️  Match %s not found (404), deleting...", eventID)
		return s.deleteMatch(eventID)
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析XML响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	var summary SportEventSummary
	if err := xml.Unmarshal(body, &summary); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}
	
	// 更新数据库
	return s.updateMatchInDB(eventID, &summary)
}

// SportEventSummary summary接口响应结构
type SportEventSummary struct {
	XMLName xml.Name `xml:"match_summary"`
	SportEvent struct {
		ID     string `xml:"id,attr"`
		Status string `xml:"status,attr"` // not_started, live, ended, closed, cancelled, postponed
		Tournament struct {
			Sport struct {
				ID   string `xml:"id,attr"` // sport_id
				Name string `xml:"name,attr"`
			} `xml:"sport"`
		} `xml:"tournament"`
	} `xml:"sport_event"`
	SportEventStatus struct {
		Status      string `xml:"status,attr"`
		MatchStatus string `xml:"match_status,attr"`
		HomeScore   *int   `xml:"home_score,attr"`
		AwayScore   *int   `xml:"away_score,attr"`
	} `xml:"sport_event_status"`
}

// updateMatchInDB 更新数据库中的比赛状态
func (s *StaleLiveCleanupService) updateMatchInDB(eventID string, summary *SportEventSummary) error {
	status := summary.SportEvent.Status
	matchStatus := summary.SportEventStatus.MatchStatus
	homeScore := summary.SportEventStatus.HomeScore
	awayScore := summary.SportEventStatus.AwayScore
	
	logger.Printf("[StaleLiveCleanup] 📝 Updating %s: status=%s, match_status=%s", 
		eventID, status, matchStatus)
	
	// 如果比赛已结束，可以选择删除或更新状态
	if status == "ended" || status == "closed" || status == "cancelled" {
// 更新为ended状态而不是删除，保留历史记录
			sportID := summary.SportEvent.Tournament.Sport.ID
			
			query := `
				UPDATE tracked_events 
				SET status = $1, 
				    match_status = $2,
				    home_score = $3,
				    away_score = $4,
				    sport_id = COALESCE(NULLIF($5, ''), sport_id), -- 仅在 sport_id 不为空时更新
				    updated_at = $6
				WHERE event_id = $7
			`
			_, err := s.db.Exec(query, status, matchStatus, homeScore, awayScore, sportID, time.Now(), eventID)
			return err
		}
	
	// 比赛仍在进行，更新状态信息
	query := `
		UPDATE tracked_events 
		SET status = $1, 
		    match_status = $2,
		    home_score = $3,
		    away_score = $4,
		    updated_at = $5
		WHERE event_id = $6
	`
	_, err := s.db.Exec(query, status, matchStatus, homeScore, awayScore, time.Now(), eventID)
	return err
}

// sendCleanupReport 发送清理报告
func (s *StaleLiveCleanupService) sendCleanupReport(result *StaleLiveCleanupResult) {
	if s.larkNotifier == nil {
		return
	}
	
	if result.TwoDaysOld == 0 && result.OneDayOld == 0 {
		// 没有需要清理的，不发送通知
		return
	}
	
	var msg string
	msg += "🧹 **过期Live比赛清理报告**\n\n"
	msg += fmt.Sprintf("📊 Live比赛总数: **%d** 场\n", result.TotalLive)
	msg += fmt.Sprintf("🗑️  2天以上比赛: **%d** 场\n", result.TwoDaysOld)
	msg += fmt.Sprintf("📅 1天以上比赛: **%d** 场\n", result.OneDayOld)
	msg += fmt.Sprintf("✅ 已删除: **%d** 场\n", result.Deleted)
	msg += fmt.Sprintf("🔄 已更新: **%d** 场\n", result.Updated)
	
	if result.Failed > 0 {
		msg += fmt.Sprintf("❌ 失败: **%d** 场\n", result.Failed)
	}
	
	// 列出删除的比赛（最多10个）
	if len(result.DeletedList) > 0 && len(result.DeletedList) <= 10 {
		msg += "\n**已删除比赛:**\n"
		for _, matchID := range result.DeletedList {
			msg += fmt.Sprintf("- %s\n", matchID)
		}
	} else if len(result.DeletedList) > 10 {
		msg += fmt.Sprintf("\n**已删除比赛:** %d 场（仅显示前10个）\n", len(result.DeletedList))
		for i := 0; i < 10; i++ {
			msg += fmt.Sprintf("- %s\n", result.DeletedList[i])
		}
	}
	
	// 列出更新的比赛（最多10个）
	if len(result.UpdatedList) > 0 && len(result.UpdatedList) <= 10 {
		msg += "\n**已更新比赛:**\n"
		for _, matchID := range result.UpdatedList {
			msg += fmt.Sprintf("- %s\n", matchID)
		}
	} else if len(result.UpdatedList) > 10 {
		msg += fmt.Sprintf("\n**已更新比赛:** %d 场（仅显示前10个）\n", len(result.UpdatedList))
		for i := 0; i < 10; i++ {
			msg += fmt.Sprintf("- %s\n", result.UpdatedList[i])
		}
	}
	
	// 列出失败的比赛（最多5个）
	if len(result.FailedList) > 0 && len(result.FailedList) <= 5 {
		msg += "\n**失败比赛:**\n"
		for matchID, errMsg := range result.FailedList {
			msg += fmt.Sprintf("- %s: %s\n", matchID, errMsg)
		}
	}
	
	msg += fmt.Sprintf("\n⏰ 时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	
	s.larkNotifier.SendText(msg)
}
