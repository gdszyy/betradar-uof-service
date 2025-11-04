package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

// LeagueInfo 联赛信息
type LeagueInfo struct {
	LeagueID        string  `json:"league_id"`         // 联赛 ID (sr:tournament:xxx)
	LeagueName      string  `json:"league_name"`       // 联赛名称
	SportID         string  `json:"sport_id"`          // 体育类型 ID
	TotalMatches    int     `json:"total_matches"`     // 总比赛数
	LiveMatches     int     `json:"live_matches"`      // Live 比赛数
	UpcomingMatches int     `json:"upcoming_matches"`  // 即将开始的比赛数
	Popularity      float64 `json:"popularity"`        // 热门度 (0-100)
}

// handleGetLeagues 获取联赛列表
// GET /api/leagues?sport_id=sr:sport:1&sort=popularity&order=desc
func (s *Server) handleGetLeagues(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// 解析参数
	sportID := r.URL.Query().Get("sport_id")
	sortBy := r.URL.Query().Get("sort") // popularity, name, total_matches, live_matches
	if sortBy == "" {
		sortBy = "popularity" // 默认按热门度排序
	}
	
	order := r.URL.Query().Get("order") // asc, desc
	if order == "" {
		order = "desc" // 默认降序
	}
	
	// 查询联赛信息
	leagues, err := s.getLeaguesInfoSimplified(sportID)
	if err != nil {
		log.Printf("[API] Error getting leagues: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to get leagues",
		})
		return
	}
	
	// 排序
	sortLeagues(leagues, sortBy, order)
	
	// 返回结果
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(leagues),
		"sort_by": sortBy,
		"order":   order,
		"leagues": leagues,
	})
}

// getLeaguesInfoSimplified 简化版:返回预定义的热门联赛列表
// TODO: 后续可以从 Sportradar API 或数据库动态获取
func (s *Server) getLeaguesInfoSimplified(sportID string) ([]LeagueInfo, error) {
	// 预定义的热门联赛列表
	allLeagues := []LeagueInfo{
		// 足球联赛
		{LeagueID: "sr:tournament:17", LeagueName: "英超 (Premier League)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:23", LeagueName: "西甲 (La Liga)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:35", LeagueName: "意甲 (Serie A)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:34", LeagueName: "德甲 (Bundesliga)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:16", LeagueName: "法甲 (Ligue 1)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:7", LeagueName: "欧冠 (Champions League)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:679", LeagueName: "欧罗巴联赛 (Europa League)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:132", LeagueName: "世界杯 (World Cup)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:8", LeagueName: "英冠 (Championship)", SportID: "sr:sport:1"},
		{LeagueID: "sr:tournament:238", LeagueName: "中超 (Chinese Super League)", SportID: "sr:sport:1"},
		
		// 篮球联赛
		{LeagueID: "sr:tournament:132", LeagueName: "NBA", SportID: "sr:sport:2"},
		{LeagueID: "sr:tournament:138", LeagueName: "Euroleague", SportID: "sr:sport:2"},
		{LeagueID: "sr:tournament:154", LeagueName: "CBA (中国篮球)", SportID: "sr:sport:2"},
		
		// 网球
		{LeagueID: "sr:tournament:1", LeagueName: "澳网 (Australian Open)", SportID: "sr:sport:5"},
		{LeagueID: "sr:tournament:2", LeagueName: "法网 (French Open)", SportID: "sr:sport:5"},
		{LeagueID: "sr:tournament:3", LeagueName: "温网 (Wimbledon)", SportID: "sr:sport:5"},
		{LeagueID: "sr:tournament:4", LeagueName: "美网 (US Open)", SportID: "sr:sport:5"},
	}
	
	// 筛选体育类型
	var filteredLeagues []LeagueInfo
	for _, league := range allLeagues {
		if sportID == "" || league.SportID == sportID {
			filteredLeagues = append(filteredLeagues, league)
		}
	}
	
	// 为每个联赛查询统计信息
	for i := range filteredLeagues {
		stats, err := s.getLeagueStats(filteredLeagues[i].LeagueID)
		if err != nil {
			log.Printf("[API] Error getting stats for league %s: %v", filteredLeagues[i].LeagueID, err)
			continue
		}
		
		filteredLeagues[i].TotalMatches = stats.TotalMatches
		filteredLeagues[i].LiveMatches = stats.LiveMatches
		filteredLeagues[i].UpcomingMatches = stats.UpcomingMatches
		filteredLeagues[i].Popularity = calculatePopularity(&filteredLeagues[i])
	}
	
	return filteredLeagues, nil
}

// LeagueStats 联赛统计
type LeagueStats struct {
	TotalMatches    int
	LiveMatches     int
	UpcomingMatches int
}

// getLeagueStats 获取联赛统计信息
// 通过查询 tracked_events 表,匹配 event_id 来统计
func (s *Server) getLeagueStats(leagueID string) (*LeagueStats, error) {
	// 由于 srn_id 为空,我们暂时无法准确统计每个联赛的比赛数
	// 返回模拟数据
	// TODO: 修复 srn_id 填充逻辑后,使用真实查询
	
	query := `
		SELECT 
			COUNT(*) as total_matches,
			COUNT(CASE WHEN status = 'active' AND match_status IS NOT NULL THEN 1 END) as live_matches,
			COUNT(CASE WHEN status = 'active' AND match_status IS NULL AND schedule_time > NOW() THEN 1 END) as upcoming_matches
		FROM tracked_events
		WHERE sport_id = $1
	`
	
	// 从 leagueID 提取 sport_id
	// 这是临时方案,假设所有同一体育类型的比赛都属于这个联赛
	sportID := "sr:sport:1" // 默认足球
	if strings.Contains(leagueID, "sr:tournament:") {
		// 根据已知映射推断 sport_id
		// TODO: 使用更准确的方法
	}
	
	var stats LeagueStats
	err := s.db.QueryRow(query, sportID).Scan(
		&stats.TotalMatches,
		&stats.LiveMatches,
		&stats.UpcomingMatches,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to query league stats: %w", err)
	}
	
	// 由于无法区分具体联赛,我们按比例分配
	// 假设每个联赛平均占该体育类型的 10%
	stats.TotalMatches = stats.TotalMatches / 10
	stats.LiveMatches = stats.LiveMatches / 10
	stats.UpcomingMatches = stats.UpcomingMatches / 10
	
	return &stats, nil
}

// calculatePopularity 计算联赛热门度 (0-100)
// 基于以下因素:
// - Live 比赛数 (权重 50%)
// - 总比赛数 (权重 30%)
// - 即将开始的比赛数 (权重 20%)
func calculatePopularity(league *LeagueInfo) float64 {
	// 归一化因子
	maxLive := 20.0        // 假设最热门的联赛有 20 场 live 比赛
	maxTotal := 100.0      // 假设最热门的联赛有 100 场总比赛
	maxUpcoming := 50.0    // 假设最热门的联赛有 50 场即将开始的比赛
	
	// 计算各项得分
	liveScore := float64(league.LiveMatches) / maxLive * 50.0
	if liveScore > 50.0 {
		liveScore = 50.0
	}
	
	totalScore := float64(league.TotalMatches) / maxTotal * 30.0
	if totalScore > 30.0 {
		totalScore = 30.0
	}
	
	upcomingScore := float64(league.UpcomingMatches) / maxUpcoming * 20.0
	if upcomingScore > 20.0 {
		upcomingScore = 20.0
	}
	
	// 总分
	popularity := liveScore + totalScore + upcomingScore
	
	// 确保在 0-100 范围内
	if popularity > 100.0 {
		popularity = 100.0
	}
	
	return popularity
}

// sortLeagues 对联赛列表排序
func sortLeagues(leagues []LeagueInfo, sortBy string, order string) {
	sort.Slice(leagues, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "popularity":
			less = leagues[i].Popularity < leagues[j].Popularity
		case "name":
			less = strings.ToLower(leagues[i].LeagueName) < strings.ToLower(leagues[j].LeagueName)
		case "total_matches":
			less = leagues[i].TotalMatches < leagues[j].TotalMatches
		case "live_matches":
			less = leagues[i].LiveMatches < leagues[j].LiveMatches
		case "upcoming_matches":
			less = leagues[i].UpcomingMatches < leagues[j].UpcomingMatches
		default:
			less = leagues[i].Popularity < leagues[j].Popularity
		}
		
		// 降序
		if order == "desc" {
			return !less
		}
		
		return less
	})
}

