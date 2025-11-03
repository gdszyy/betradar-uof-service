package web

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	
	"uof-service/services"
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

// getLeaguesInfo 从数据库获取联赛信息
func (s *Server) getLeaguesInfo(sportID string) ([]LeagueInfo, error) {
	// 构建查询
	query := `
		SELECT 
			e.srn_id,
			e.sport_id,
			COUNT(*) as total_matches,
			COUNT(CASE WHEN e.status = 'active' AND e.match_status IS NOT NULL THEN 1 END) as live_matches,
			COUNT(CASE WHEN e.status = 'active' AND e.match_status IS NULL AND e.schedule_time > NOW() THEN 1 END) as upcoming_matches
		FROM tracked_events e
		WHERE e.srn_id IS NOT NULL AND e.srn_id != ''
	`
	
	args := []interface{}{}
	argIndex := 1
	
	// 体育类型筛选
	if sportID != "" {
		query += ` AND e.sport_id = $` + string(rune(argIndex+'0'))
		args = append(args, sportID)
		argIndex++
	}
	
	query += `
		GROUP BY e.srn_id, e.sport_id
		HAVING COUNT(*) > 0
		ORDER BY live_matches DESC, total_matches DESC
	`
	
	// 执行查询
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// 解析结果
	leagueMap := make(map[string]*LeagueInfo)
	
	for rows.Next() {
		var srnID, sportIDVal string
		var totalMatches, liveMatches, upcomingMatches int
		
		if err := rows.Scan(&srnID, &sportIDVal, &totalMatches, &liveMatches, &upcomingMatches); err != nil {
			log.Printf("[API] Error scanning league row: %v", err)
			continue
		}
		
		// 从 srn_id 提取联赛 ID
		// 格式: sr:sport:1:season:12345:tournament:678:match:9999
		leagueID := extractLeagueID(srnID)
		if leagueID == "" {
			continue
		}
		
		// 聚合相同联赛的数据
		if existing, ok := leagueMap[leagueID]; ok {
			existing.TotalMatches += totalMatches
			existing.LiveMatches += liveMatches
			existing.UpcomingMatches += upcomingMatches
		} else {
			leagueMap[leagueID] = &LeagueInfo{
				LeagueID:        leagueID,
				LeagueName:      "", // 稍后填充
				SportID:         sportIDVal,
				TotalMatches:    totalMatches,
				LiveMatches:     liveMatches,
				UpcomingMatches: upcomingMatches,
			}
		}
	}
	
	// 转换为切片
	leagues := make([]LeagueInfo, 0, len(leagueMap))
	for _, league := range leagueMap {
		// 计算热门度
		league.Popularity = calculatePopularity(league)
		
		// 获取联赛名称 (如果有)
		league.LeagueName = getLeagueName(league.LeagueID)
		
		leagues = append(leagues, *league)
	}
	
	return leagues, nil
}

// extractLeagueID 从 srn_id 提取联赛 ID
// 输入: sr:sport:1:season:12345:tournament:678:match:9999
// 输出: sr:tournament:678
func extractLeagueID(srnID string) string {
	// 使用正则表达式提取 tournament ID
	re := regexp.MustCompile(`tournament:(\d+)`)
	matches := re.FindStringSubmatch(srnID)
	if len(matches) > 1 {
		return "sr:tournament:" + matches[1]
	}
	return ""
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

// getLeagueName 获取联赛名称
// TODO: 从数据库或缓存中获取联赛名称
// 目前返回空字符串,可以后续集成 Sportradar API 或维护联赛名称映射表
func getLeagueName(leagueID string) string {
	// 常见联赛名称映射 (可以扩展)
	knownLeagues := map[string]string{
		"sr:tournament:17":  "英超 (Premier League)",
		"sr:tournament:23":  "西甲 (La Liga)",
		"sr:tournament:35":  "意甲 (Serie A)",
		"sr:tournament:34":  "德甲 (Bundesliga)",
		"sr:tournament:34":  "法甲 (Ligue 1)",
		"sr:tournament:7":   "欧冠 (Champions League)",
		"sr:tournament:679": "欧罗巴联赛 (Europa League)",
		"sr:tournament:132": "世界杯 (World Cup)",
	}
	
	if name, ok := knownLeagues[leagueID]; ok {
		return name
	}
	
	// 如果没有映射,返回 ID
	return leagueID
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

