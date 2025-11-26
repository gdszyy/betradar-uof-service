package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// CategoryResponse 分类响应
type CategoryResponse struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	SportID      string `json:"sport_id"`
	MatchCount   int    `json:"match_count"`
}

// TournamentResponse 联赛响应
type TournamentResponse struct {
	TournamentID   string `json:"tournament_id"`
	TournamentName string `json:"tournament_name"`
	CategoryID     string `json:"category_id"`
	SportID        string `json:"sport_id"`
	MatchCount     int    `json:"match_count"`
}

// handleGetCategories 获取分类列表
// GET /api/categories?sport_ids=sr:sport:1,sr:sport:2&page=1&page_size=10&sort=name|match_count_asc|match_count_desc
func (s *Server) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting categories...")

	// 解析参数
	sportIDsStr := r.URL.Query().Get("sport_ids")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "name"
	}

	// 构建查询
	var args []interface{}
	argIndex := 1

	// 体育类型筛选
	sportFilter := ""
	if sportIDsStr != "" {
		sportIDs := strings.Split(sportIDsStr, ",")
		placeholders := make([]string, len(sportIDs))
		for i, id := range sportIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, strings.TrimSpace(id))
			argIndex++
		}
		sportFilter = fmt.Sprintf("AND t.sport_id IN (%s)", strings.Join(placeholders, ","))
	}

	// 排序
	orderBy := "ORDER BY t.category_id"
	if sort == "match_count_asc" {
		orderBy = "ORDER BY match_count ASC"
	} else if sort == "match_count_desc" {
		orderBy = "ORDER BY match_count DESC"
	}

	// 查询
	query := fmt.Sprintf(`
		SELECT 
			t.category_id,
			COALESCE(t.category_name, t.category_id) AS category_name,
			t.sport_id,
			COUNT(DISTINCT e.event_id) AS match_count
		FROM teams t
		INNER JOIN tracked_events e ON (e.home_team_id = t.team_id OR e.away_team_id = t.team_id)
		WHERE t.category_id IS NOT NULL
			%s
		GROUP BY t.category_id, t.category_name, t.sport_id
		%s
		LIMIT $%d OFFSET $%d
	`, sportFilter, orderBy, argIndex, argIndex+1)

	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Error querying categories: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query categories: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []CategoryResponse
	for rows.Next() {
		var cat CategoryResponse
		if err := rows.Scan(&cat.CategoryID, &cat.CategoryName, &cat.SportID, &cat.MatchCount); err != nil {
			log.Printf("[API] Error scanning category: %v", err)
			continue
		}
		categories = append(categories, cat)
	}

	if categories == nil {
		categories = []CategoryResponse{}
	}

	// 查询总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT t.category_id)
		FROM teams t
		INNER JOIN tracked_events e ON (e.home_team_id = t.team_id OR e.away_team_id = t.team_id)
		WHERE t.category_id IS NOT NULL
			%s
	`, sportFilter)

	var totalCount int
	countArgs := args[:len(args)-2] // 去除 LIMIT 和 OFFSET
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		log.Printf("[API] Error counting categories: %v", err)
		totalCount = 0
	}

	response := map[string]interface{}{
		"success":     true,
		"data":        categories,
		"page":        page,
		"page_size":   pageSize,
		"count":       len(categories),
		"total":       totalCount,
		"total_pages": (totalCount + pageSize - 1) / pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetTournaments 获取联赛列表
// GET /api/tournaments?category_id=sr:category:1&page=1&page_size=10&sort=name|match_count_asc|match_count_desc
func (s *Server) handleGetTournaments(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting tournaments...")

	// 解析参数
	categoryID := r.URL.Query().Get("category_id")
	if categoryID == "" {
		http.Error(w, "category_id is required", http.StatusBadRequest)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "name"
	}

	// 注意：目前 teams 表中没有 tournament_id 字段
	// sort 参数暂时未使用
	_ = sort
	// 我们需要从 event_id 或其他来源获取 tournament 信息
	// 暂时返回空结果，需要进一步讨论数据来源

	response := map[string]interface{}{
		"success":   true,
		"data":      []TournamentResponse{},
		"page":      page,
		"page_size": pageSize,
		"count":     0,
		"total":     0,
		"message":   "Tournament data not available in current schema. Need to add tournament_id to teams table or fetch from SportRader API.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
