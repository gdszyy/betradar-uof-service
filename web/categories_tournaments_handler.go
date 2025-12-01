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
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	SportID      string  `json:"sport_id"`
	MatchCount   int     `json:"match_count"`
	Popularity   float64 `json:"popularity,omitempty"` // 热度分数
}

// TournamentResponse 联赛响应
type TournamentResponse struct {
	TournamentID   string  `json:"tournament_id"`
	TournamentName string  `json:"tournament_name"`
	CategoryID     string  `json:"category_id"`
	SportID        string  `json:"sport_id"`
	MatchCount     int     `json:"match_count"`
	Popularity     float64 `json:"popularity,omitempty"` // 热度分数
}

// handleGetCategories 获取分类列表
// GET /api/categories?sport_ids=sr:sport:1,sr:sport:2&page=1&page_size=10&sort=name|match_count_asc|match_count_desc|popularity_desc
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
			sportFilter = fmt.Sprintf("AND c.sport_id IN (%s)", strings.Join(placeholders, ","))
	}

	// 排序
	orderBy := "ORDER BY category_name"
	if sort == "match_count_asc" {
		orderBy = "ORDER BY match_count ASC"
	} else if sort == "match_count_desc" {
		orderBy = "ORDER BY match_count DESC"
	}
	// 注意：popularity_desc 排序已被移除，因为 tournament_popularity_scores 表为空

	// 查询 - 从 categories 表获取所有 category，左连接 tracked_events 获取赛事数量
	// 修复：移除对 tournament_popularity_scores 的依赖（该表为空），使用 DISTINCT 防止重复
	query := fmt.Sprintf(`
		SELECT DISTINCT
			c.id as category_id,
			c.name as category_name,
			c.sport_id,
			COALESCE(COUNT(DISTINCT te.event_id), 0) AS match_count
		FROM categories c
		LEFT JOIN tracked_events te ON c.id = te.category_id
		WHERE 1=1 %s
		GROUP BY c.id, c.name, c.sport_id
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
		SELECT COUNT(DISTINCT c.id)
		FROM categories c
		WHERE 1=1 %s
	`, sportFilter)

	var totalCount int
	countArgs := args[:len(args)-2] // 去除 LIMIT 和 OFFSET
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		log.Printf("[API] Error counting categories: %v", err)
		totalCount = 0
	}

	// 查询热度前五名的联赛
	// 修复：直接从 tracked_events 表计算，按赛事数量排序（作为热度指标）
	topTournamentsQuery := `
		SELECT 
			te.tournament_id,
			MAX(te.tournament_name) AS tournament_name,
			MAX(te.category_id) AS category_id,
			COUNT(DISTINCT te.event_id) AS match_count
		FROM tracked_events te
		WHERE te.tournament_id IS NOT NULL 
			AND te.tournament_id != ''
			AND te.tournament_name IS NOT NULL
			AND te.tournament_name != ''
		GROUP BY te.tournament_id
		ORDER BY match_count DESC
		LIMIT 5
	`
	
	type TopTournament struct {
		TournamentID   string `json:"tournament_id"`
		TournamentName string `json:"tournament_name"`
		CategoryID     string `json:"category_id"`
		MatchCount     int    `json:"match_count"`
	}
	
	var topTournaments []TopTournament
	topRows, err := s.db.Query(topTournamentsQuery)
	if err != nil {
		log.Printf("[API] Error querying top tournaments: %v", err)
	} else {
		defer topRows.Close()
			for topRows.Next() {
				var t TopTournament
				if err := topRows.Scan(&t.TournamentID, &t.TournamentName, &t.CategoryID, &t.MatchCount); err != nil {
					log.Printf("[API] Error scanning top tournament: %v", err)
					continue
				}
				topTournaments = append(topTournaments, t)
			}
	}
	
	if topTournaments == nil {
		topTournaments = []TopTournament{}
	}

	response := map[string]interface{}{
		"success":        true,
		"data":           categories,
		"page":           page,
		"page_size":      pageSize,
		"count":          len(categories),
		"total":          totalCount,
		"total_pages":    (totalCount + pageSize - 1) / pageSize,
		"top_tournaments": topTournaments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetTournaments 获取联赛列表
// GET /api/tournaments?category_id=sr:category:1&page=1&page_size=10&sort=name|match_count_asc|match_count_desc|popularity_desc
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

	// 排序
	orderBy := "ORDER BY tournament_name"
	if sort == "match_count_asc" {
		orderBy = "ORDER BY match_count ASC"
	} else if sort == "match_count_desc" {
		orderBy = "ORDER BY match_count DESC"
	} else if sort == "popularity_desc" {
		orderBy = "ORDER BY popularity DESC"
	}

	// 查询 - 从 tracked_events 表获取 tournament 信息，左连接 tournament_popularity_scores 获取热度
	// 使用 MAX() 选择非空的 sport_id，避免同一 tournament 因 sport_id 不同而重复
	query := fmt.Sprintf(`
		SELECT 
			te.tournament_id,
			MAX(COALESCE(te.tournament_name, te.tournament_id)) AS tournament_name,
			te.category_id,
			MAX(CASE WHEN te.sport_id != '' THEN te.sport_id ELSE NULL END) AS sport_id,
			COUNT(DISTINCT te.event_id) AS match_count,
			MAX(COALESCE(tps.final_popularity_score, 0)) AS popularity
		FROM tracked_events te
		LEFT JOIN tournament_popularity_scores tps ON te.tournament_id = tps.tournament_id
		WHERE te.category_id = $1 
			AND te.tournament_id IS NOT NULL 
			AND te.tournament_id != ''
		GROUP BY te.tournament_id, te.category_id
		%s
		LIMIT $2 OFFSET $3
	`, orderBy)

	offset := (page - 1) * pageSize
	rows, err := s.db.Query(query, categoryID, pageSize, offset)
	if err != nil {
		log.Printf("[API] Error querying tournaments: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query tournaments: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tournaments []TournamentResponse
	for rows.Next() {
		var tournament TournamentResponse
		if err := rows.Scan(&tournament.TournamentID, &tournament.TournamentName, &tournament.CategoryID, &tournament.SportID, &tournament.MatchCount, &tournament.Popularity); err != nil {
			log.Printf("[API] Error scanning tournament: %v", err)
			continue
		}
		tournaments = append(tournaments, tournament)
	}

	if tournaments == nil {
		tournaments = []TournamentResponse{}
	}

	// 查询总数
	countQuery := `
		SELECT COUNT(DISTINCT te.tournament_id)
		FROM tracked_events te
		WHERE te.category_id = $1 
			AND te.tournament_id IS NOT NULL 
			AND te.tournament_id != ''
	`

	var totalCount int
	if err := s.db.QueryRow(countQuery, categoryID).Scan(&totalCount); err != nil {
		log.Printf("[API] Error counting tournaments: %v", err)
		totalCount = 0
	}

	response := map[string]interface{}{
		"success":     true,
		"data":        tournaments,
		"page":        page,
		"page_size":   pageSize,
		"count":       len(tournaments),
		"total":       totalCount,
		"total_pages": (totalCount + pageSize - 1) / pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
