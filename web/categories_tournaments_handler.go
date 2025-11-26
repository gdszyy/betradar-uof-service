package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// CategoryInfo 分类信息
type CategoryInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SportID    string `json:"sport_id"`
	MatchCount int    `json:"match_count"`
}

// TournamentInfo 联赛信息
type TournamentInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CategoryID string `json:"category_id"`
	SportID    string `json:"sport_id"`
	MatchCount int    `json:"match_count"`
}

// handleGetCategories 获取分类列表
// GET /api/categories
func (s *Server) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting categories...")

	// 解析查询参数
	sportIDsStr := r.URL.Query().Get("sport_ids")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	sortBy := r.URL.Query().Get("sort")

	// 默认值
	page := 1
	pageSize := 100
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 500 {
			pageSize = ps
		}
	}

	// 默认排序：按名称升序
	if sortBy == "" {
		sortBy = "name_asc"
	}

	// 构建查询
	var args []interface{}
	argIndex := 1

	// 从 tracked_events 表的 srn_id 中提取 category
	// srn_id 格式: sr:sport:1:category:1:tournament:100:match:12345
	// 我们需要从 srn_id 中提取 category 和 tournament 信息

	// 体育类型筛选
	sportIDFilter := ""
	if sportIDsStr != "" {
		sportIDList := strings.Split(sportIDsStr, ",")
		placeholders := make([]string, len(sportIDList))
		for i, id := range sportIDList {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			// 转换为数字 ID (例如: "sr:sport:1" -> "1")
			sportIDParts := strings.Split(strings.TrimSpace(id), ":")
			if len(sportIDParts) >= 3 {
				args = append(args, sportIDParts[2])
			} else {
				args = append(args, strings.TrimSpace(id))
			}
			argIndex++
		}
		sportIDFilter = fmt.Sprintf("AND sport_id IN (%s)", strings.Join(placeholders, ","))
	}

	// 查询分类及比赛数量
	// 从 srn_id 中提取 category_id
	query := fmt.Sprintf(`
		WITH category_data AS (
			SELECT DISTINCT
				CASE 
					WHEN srn_id ~ 'category:([0-9]+)' THEN 
						'sr:category:' || (regexp_match(srn_id, 'category:([0-9]+)'))[1]
					ELSE NULL
				END AS category_id,
				sport_id
			FROM tracked_events
			WHERE srn_id IS NOT NULL
				AND srn_id ~ 'category:([0-9]+)'
				%s
		)
		SELECT 
			cd.category_id,
			cd.category_id AS category_name,
			'sr:sport:' || cd.sport_id AS sport_id,
			COUNT(e.event_id) AS match_count
		FROM category_data cd
		LEFT JOIN tracked_events e ON e.srn_id LIKE '%%' || cd.category_id || '%%'
		WHERE cd.category_id IS NOT NULL
		GROUP BY cd.category_id, cd.sport_id
	`, sportIDFilter)

	// 添加排序
	switch sortBy {
	case "name_asc":
		query += " ORDER BY cd.category_id ASC"
	case "name_desc":
		query += " ORDER BY cd.category_id DESC"
	case "match_count_asc":
		query += " ORDER BY match_count ASC"
	case "match_count_desc":
		query += " ORDER BY match_count DESC"
	default:
		query += " ORDER BY cd.category_id ASC"
	}

	// 添加分页
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	// 执行查询
	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Error querying categories: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query categories: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []CategoryInfo
	for rows.Next() {
		var category CategoryInfo
		if err := rows.Scan(&category.ID, &category.Name, &category.SportID, &category.MatchCount); err != nil {
			log.Printf("[API] Error scanning category: %v", err)
			continue
		}
		categories = append(categories, category)
	}

	if categories == nil {
		categories = []CategoryInfo{}
	}

	// 查询总数
	countQuery := fmt.Sprintf(`
		WITH category_data AS (
			SELECT DISTINCT
				CASE 
					WHEN srn_id ~ 'category:([0-9]+)' THEN 
						'sr:category:' || (regexp_match(srn_id, 'category:([0-9]+)'))[1]
					ELSE NULL
				END AS category_id,
				sport_id
			FROM tracked_events
			WHERE srn_id IS NOT NULL
				AND srn_id ~ 'category:([0-9]+)'
				%s
		)
		SELECT COUNT(DISTINCT cd.category_id)
		FROM category_data cd
		WHERE cd.category_id IS NOT NULL
	`, sportIDFilter)
	var totalCount int
	countArgs := args[:len(args)-2] // 去除 LIMIT 和 OFFSET 参数
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil && err != sql.ErrNoRows {
		log.Printf("[API] Error counting categories: %v", err)
		totalCount = 0
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	// 构建响应
	response := map[string]interface{}{
		"success":     true,
		"count":       len(categories),
		"total":       totalCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"sort":        sortBy,
		"data":        categories,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetTournaments 获取联赛列表
// GET /api/tournaments
func (s *Server) handleGetTournaments(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting tournaments...")

	// 解析查询参数
	categoryID := r.URL.Query().Get("category_id")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	sortBy := r.URL.Query().Get("sort")

	// 检查 category_id 是否存在
	if categoryID == "" {
		http.Error(w, "category_id is required", http.StatusBadRequest)
		return
	}

	// 默认值
	page := 1
	pageSize := 100
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 500 {
			pageSize = ps
		}
	}

	// 默认排序：按名称升序
	if sortBy == "" {
		sortBy = "name_asc"
	}

	// 构建查询
	// 从 srn_id 中提取 tournament_id
	// srn_id 格式: sr:sport:1:category:1:tournament:100:match:12345
	query := `
		WITH tournament_data AS (
			SELECT DISTINCT
				CASE 
					WHEN srn_id ~ 'tournament:([0-9]+)' THEN 
						'sr:tournament:' || (regexp_match(srn_id, 'tournament:([0-9]+)'))[1]
					ELSE NULL
				END AS tournament_id,
				CASE 
					WHEN srn_id ~ 'category:([0-9]+)' THEN 
						'sr:category:' || (regexp_match(srn_id, 'category:([0-9]+)'))[1]
					ELSE NULL
				END AS category_id,
				sport_id
			FROM tracked_events
			WHERE srn_id IS NOT NULL
				AND srn_id ~ 'tournament:([0-9]+)'
				AND srn_id LIKE '%' || $1 || '%'
		)
		SELECT 
			td.tournament_id,
			td.tournament_id AS tournament_name,
			td.category_id,
			'sr:sport:' || td.sport_id AS sport_id,
			COUNT(e.event_id) AS match_count
		FROM tournament_data td
		LEFT JOIN tracked_events e ON e.srn_id LIKE '%' || td.tournament_id || '%'
		WHERE td.tournament_id IS NOT NULL
		GROUP BY td.tournament_id, td.category_id, td.sport_id
	`

	// 添加排序
	switch sortBy {
	case "name_asc":
		query += " ORDER BY td.tournament_id ASC"
	case "name_desc":
		query += " ORDER BY td.tournament_id DESC"
	case "match_count_asc":
		query += " ORDER BY match_count ASC"
	case "match_count_desc":
		query += " ORDER BY match_count DESC"
	default:
		query += " ORDER BY td.tournament_id ASC"
	}

	// 添加分页
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" LIMIT $2 OFFSET $3")

	// 执行查询
	rows, err := s.db.Query(query, categoryID, pageSize, offset)
	if err != nil {
		log.Printf("[API] Error querying tournaments: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query tournaments: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tournaments []TournamentInfo
	for rows.Next() {
		var tournament TournamentInfo
		if err := rows.Scan(&tournament.ID, &tournament.Name, &tournament.CategoryID, &tournament.SportID, &tournament.MatchCount); err != nil {
			log.Printf("[API] Error scanning tournament: %v", err)
			continue
		}
		tournaments = append(tournaments, tournament)
	}

	if tournaments == nil {
		tournaments = []TournamentInfo{}
	}

	// 查询总数
	countQuery := `
		WITH tournament_data AS (
			SELECT DISTINCT
				CASE 
					WHEN srn_id ~ 'tournament:([0-9]+)' THEN 
						'sr:tournament:' || (regexp_match(srn_id, 'tournament:([0-9]+)'))[1]
					ELSE NULL
				END AS tournament_id,
				CASE 
					WHEN srn_id ~ 'category:([0-9]+)' THEN 
						'sr:category:' || (regexp_match(srn_id, 'category:([0-9]+)'))[1]
					ELSE NULL
				END AS category_id
			FROM tracked_events
			WHERE srn_id IS NOT NULL
				AND srn_id ~ 'tournament:([0-9]+)'
				AND srn_id LIKE '%' || $1 || '%'
		)
		SELECT COUNT(DISTINCT td.tournament_id)
		FROM tournament_data td
		WHERE td.tournament_id IS NOT NULL
	`
	var totalCount int
	if err := s.db.QueryRow(countQuery, categoryID).Scan(&totalCount); err != nil && err != sql.ErrNoRows {
		log.Printf("[API] Error counting tournaments: %v", err)
		totalCount = 0
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	// 构建响应
	response := map[string]interface{}{
		"success":     true,
		"count":       len(tournaments),
		"total":       totalCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"sort":        sortBy,
		"data":        tournaments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
