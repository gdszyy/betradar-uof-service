package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

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
	var conditions []string
	var args []interface{}
	argIndex := 1

	if sportIDsStr != "" {
		sportIDs := strings.Split(sportIDsStr, ",")
		placeholders := make([]string, len(sportIDs))
		for i, id := range sportIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, "%"+id+"%")
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("("+strings.Join(placeholders, " OR ")+")", strings.Repeat("srn_id LIKE %s", len(sportIDs))))
		
		// 修正条件构建
		orConditions := make([]string, len(sportIDs))
		for i := range sportIDs {
			orConditions[i] = fmt.Sprintf("srn_id LIKE $%d", i+1)
		}
		conditions = []string{fmt.Sprintf("(%s)", strings.Join(orConditions, " OR "))}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 排序
	orderBy := "ORDER BY category_id"
	if sort == "match_count_asc" {
		orderBy = "ORDER BY match_count ASC"
	} else if sort == "match_count_desc" {
		orderBy = "ORDER BY match_count DESC"
	}

	// 查询
	query := fmt.Sprintf(`
		WITH category_data AS (
			SELECT 
				SUBSTRING(srn_id FROM 'sr:sport:[0-9]+:category:([0-9]+)') AS category_id,
				COUNT(*) AS match_count
			FROM tracked_events
			%s
			AND srn_id ~ 'sr:sport:[0-9]+:category:[0-9]+'
			GROUP BY category_id
		)
		SELECT 
			'sr:category:' || category_id AS category_id,
			'Category ' || category_id AS category_name,
			match_count
		FROM category_data
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Error querying categories: %v", err)
		http.Error(w, "Failed to query categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type CategoryResponse struct {
		CategoryID   string `json:"category_id"`
		CategoryName string `json:"category_name"`
		MatchCount   int    `json:"match_count"`
	}

	var categories []CategoryResponse
	for rows.Next() {
		var cat CategoryResponse
		if err := rows.Scan(&cat.CategoryID, &cat.CategoryName, &cat.MatchCount); err != nil {
			log.Printf("[API] Error scanning category: %v", err)
			continue
		}
		categories = append(categories, cat)
	}

	if categories == nil {
		categories = []CategoryResponse{}
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      categories,
		"page":      page,
		"page_size": pageSize,
		"count":     len(categories),
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

	// 排序
	orderBy := "ORDER BY tournament_id"
	if sort == "match_count_asc" {
		orderBy = "ORDER BY match_count ASC"
	} else if sort == "match_count_desc" {
		orderBy = "ORDER BY match_count DESC"
	}

	// 查询
	query := fmt.Sprintf(`
		WITH tournament_data AS (
			SELECT 
				SUBSTRING(srn_id FROM 'sr:sport:[0-9]+:category:[0-9]+:tournament:([0-9]+)') AS tournament_id,
				COUNT(*) AS match_count
			FROM tracked_events
			WHERE srn_id LIKE $1
			AND srn_id ~ 'sr:sport:[0-9]+:category:[0-9]+:tournament:[0-9]+'
			GROUP BY tournament_id
		)
		SELECT 
			'sr:tournament:' || tournament_id AS tournament_id,
			'Tournament ' || tournament_id AS tournament_name,
			match_count
		FROM tournament_data
		%s
		LIMIT $2 OFFSET $3
	`, orderBy)

	rows, err := s.db.Query(query, "%"+categoryID+"%", pageSize, (page-1)*pageSize)
	if err != nil {
		log.Printf("[API] Error querying tournaments: %v", err)
		http.Error(w, "Failed to query tournaments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type TournamentResponse struct {
		TournamentID   string `json:"tournament_id"`
		TournamentName string `json:"tournament_name"`
		MatchCount     int    `json:"match_count"`
	}

	var tournaments []TournamentResponse
	for rows.Next() {
		var t TournamentResponse
		if err := rows.Scan(&t.TournamentID, &t.TournamentName, &t.MatchCount); err != nil {
			log.Printf("[API] Error scanning tournament: %v", err)
			continue
		}
		tournaments = append(tournaments, t)
	}

	if tournaments == nil {
		tournaments = []TournamentResponse{}
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      tournaments,
		"page":      page,
		"page_size": pageSize,
		"count":     len(tournaments),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
