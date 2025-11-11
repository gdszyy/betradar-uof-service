package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

	// CategoryInfo 类别信息
	type CategoryInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	// LeagueInfo 联赛信息
	type LeagueInfo struct {
	LeagueID        string  `json:"league_id"`         // 联赛 ID (sr:tournament:xxx)
	LeagueName      string  `json:"league_name"`       // 联赛名称
	SportID         string  `json:"sport_id"`          // 体育类型 ID
		Category CategoryInfo `json:"category"` // 类别信息 (地域/分组)
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
	leagues, err := s.getLeaguesInfo(sportID)
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

// getLeaguesInfo 从 Sportradar API 获取联赛信息
func (s *Server) getLeaguesInfo(sportID string) ([]LeagueInfo, error) {
	var leagues []LeagueInfo
	
	// 如果指定了 sport_id,只获取该体育类型的联赛
	if sportID != "" {
		tournaments, err := s.sportradarAPIClient.GetTournamentsBySport(sportID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tournaments for sport %s: %w", sportID, err)
		}
		
			for _, tournament := range tournaments.Tournaments {
					league := LeagueInfo{
							LeagueID:     tournament.ID,
							LeagueName:   extractEnglishName(tournament.Name),
							SportID:      sportID,
							Category: CategoryInfo{
								ID:   tournament.Category.ID,
								Name: extractEnglishName(tournament.Category.Name),
							},
					}
			
			// 获取统计信息
			stats, err := s.getLeagueStats(tournament.ID)
			if err != nil {
				log.Printf("[API] Failed to get stats for league %s: %v", tournament.ID, err)
			} else {
				league.TotalMatches = stats.TotalMatches
				league.LiveMatches = stats.LiveMatches
				league.UpcomingMatches = stats.UpcomingMatches
				league.Popularity = calculatePopularity(&league)
			}
			
			leagues = append(leagues, league)
		}
	} else {
		// 获取所有体育类型的联赛
		allTournaments, err := s.sportradarAPIClient.GetAllTournaments()
		if err != nil {
			return nil, fmt.Errorf("failed to get all tournaments: %w", err)
		}
		
		for sportID, tournaments := range allTournaments {
			for _, tournament := range tournaments.Tournaments {
				league := LeagueInfo{
					LeagueID:     tournament.ID,
					LeagueName:   extractEnglishName(tournament.Name),
					SportID:      sportID,
						Category: CategoryInfo{
							ID:   tournament.Category.ID,
							Name: extractEnglishName(tournament.Category.Name),
						},
				}
				
				// 获取统计信息
				stats, err := s.getLeagueStats(tournament.ID)
				if err != nil {
					log.Printf("[API] Failed to get stats for league %s: %v", tournament.ID, err)
				} else {
					league.TotalMatches = stats.TotalMatches
					league.LiveMatches = stats.LiveMatches
					league.UpcomingMatches = stats.UpcomingMatches
					league.Popularity = calculatePopularity(&league)
				}
				
				leagues = append(leagues, league)
			}
		}
	}
	
	return leagues, nil
}

// LeagueStats 联赛统计
type LeagueStats struct {
	TotalMatches    int
	LiveMatches     int
	UpcomingMatches int
}

// getLeagueStats 获取联赛统计信息
	// 通过查询 tracked_events 表,匹配 event_id 来统计 (原 srn_id 字段存储了 event_id)
	func (s *Server) getLeagueStats(leagueID string) (*LeagueStats, error) {
		// 根据用户反馈，tracked_events 的 srn_id 字段实际存储了 event_id，
		// 且 event_id 包含 leagueID (sr:tournament:xxx)
		
		query := `
			SELECT 
				COUNT(*) as total_matches,
				COUNT(CASE WHEN status = 'active' AND match_status IS NOT NULL THEN 1 END) as live_matches,
				COUNT(CASE WHEN status = 'active' AND match_status IS NULL AND schedule_time > NOW() THEN 1 END) as upcoming_matches
			FROM tracked_events
			WHERE event_id LIKE $1
		`
		
		var stats LeagueStats
		err := s.db.QueryRow(query, "%"+leagueID+"%").Scan(
			&stats.TotalMatches,
			&stats.LiveMatches,
			&stats.UpcomingMatches,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to query league stats: %w", err)
		}
		
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




// extractEnglishName 从联赛名称中提取英文部分
// 例如: "中超 (Chinese Super League)" -> "Chinese Super League"
//      "Premier League" -> "Premier League"
func extractEnglishName(name string) string {
	// 1. 尝试提取括号中的内容
	re := regexp.MustCompile(`\((.*?)\)`)
	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		englishName := strings.TrimSpace(matches[1])
		// 验证提取的内容是否主要是英文
		if isEnglishText(englishName) {
			return englishName
		}
	}
	
	// 2. 如果没有括号，检查名称是否包含中文
	if containsChinese(name) {
		// 尝试移除中文部分，保留英文部分
		cleanedName := removeChinese(name)
		if cleanedName != "" {
			return cleanedName
		}
	}
	
	// 3. 返回原名称
	return name
}

// isEnglishText 检查文本是否主要是英文
func isEnglishText(s string) bool {
	if s == "" {
		return false
	}
	
	englishChars := 0
	totalChars := 0
	
	for _, r := range s {
		if unicode.IsLetter(r) {
			totalChars++
			if r < 128 { // ASCII 范围内的字母
				englishChars++
			}
		}
	}
	
	if totalChars == 0 {
		return false
	}
	
	// 如果超过 80% 是英文字母，认为是英文文本
	return float64(englishChars)/float64(totalChars) > 0.8
}

// containsChinese 检查字符串是否包含中文字符
func containsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// removeChinese 移除字符串中的中文字符
func removeChinese(s string) string {
	var result strings.Builder
	
	for _, r := range s {
		if !unicode.Is(unicode.Han, r) {
			result.WriteRune(r)
		}
	}
	
	return strings.TrimSpace(result.String())
}

