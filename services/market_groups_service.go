package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// MarketGroupsService 处理市场 groups 的获取和更新
type MarketGroupsService struct {
	db *sql.DB
}

// NewMarketGroupsService 创建新的 MarketGroupsService
func NewMarketGroupsService(db *sql.DB) *MarketGroupsService {
	return &MarketGroupsService{db: db}
}

// SyncMarketGroupsFromDescriptions 从 market_descriptions 表同步 groups 到 markets 表
func (s *MarketGroupsService) SyncMarketGroupsFromDescriptions() (int, error) {
	log.Println("开始从 market_descriptions 同步 groups 到 markets...")

	// 检查 market_descriptions 表是否存在且有数据
	var descCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM market_descriptions").Scan(&descCount)
	if err != nil {
		return 0, fmt.Errorf("检查 market_descriptions 表失败: %w", err)
	}

	if descCount == 0 {
		log.Println("警告：market_descriptions 表为空，无法同步 groups")
		return 0, nil
	}

	// 使用 SQL 更新 markets 表中的 groups 字段
	// 基于 sr_market_id 匹配 market_descriptions 表
	result, err := s.db.Exec(`
		UPDATE markets m
		SET groups = md.groups, updated_at = CURRENT_TIMESTAMP
		FROM market_descriptions md
		WHERE m.sr_market_id = md.market_id
		AND (m.groups IS NULL OR m.groups = '')
		AND md.groups IS NOT NULL AND md.groups != ''
	`)

	if err != nil {
		return 0, fmt.Errorf("更新 markets 表失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败: %w", err)
	}

	log.Printf("✓ 成功同步 %d 个市场的 groups\n", rowsAffected)
	return int(rowsAffected), nil
}

// AssignTabsBasedOnGroups 基于 groups 为市场分配 tabs
func (s *MarketGroupsService) AssignTabsBasedOnGroups() (int, error) {
	log.Println("开始基于 groups 分配 tabs...")

	// 定义 group 到 tab 的映射
	groupTabMapping := map[string]string{
		"regular_play":    "regular_play",
		"player_props":    "player_props",
		"micro_market":    "micro_market",
		"bookings":        "bookings",
		"corners":         "corners",
		"1st_half":        "1st_half",
		"combo":           "combo",
		"2nd_half":        "2nd_half",
		"scorers":         "scorers",
		"innings":         "innings",
		"sets":            "sets",
		"maps":            "maps",
		"quarters":        "quarters",
		"periods":         "periods",
		"frames":          "frames",
		"overs":           "overs",
		"drives":          "drives",
	}

	// 获取所有需要分配的市场
	rows, err := s.db.Query(`
		SELECT id, groups
		FROM markets
		WHERE (tab_id IS NULL OR tab_id = '')
		AND groups IS NOT NULL AND groups != ''
	`)
	if err != nil {
		return 0, fmt.Errorf("查询市场失败: %w", err)
	}
	defer rows.Close()

	var totalAssigned int
	var groupsToUpdate []struct {
		marketID int
		tabID    string
	}

	for rows.Next() {
		var marketID int
		var groupsStr string

		if err := rows.Scan(&marketID, &groupsStr); err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		// 解析 groups（可能是 JSON 数组或逗号分隔的字符串）
		var groups []string
		if strings.HasPrefix(groupsStr, "[") {
			// JSON 数组格式
			err := json.Unmarshal([]byte(groupsStr), &groups)
			if err != nil {
				// 如果 JSON 解析失败，尝试作为逗号分隔的字符串
				groups = strings.Split(groupsStr, ",")
			}
		} else {
			// 逗号分隔的字符串
			groups = strings.Split(groupsStr, ",")
		}

		// 查找第一个匹配的 group
		var tabID string
		for _, group := range groups {
			group = strings.TrimSpace(group)
			if mappedTab, exists := groupTabMapping[group]; exists {
				tabID = mappedTab
				break
			}
		}

		// 如果找到了映射，记录更新
		if tabID != "" {
			groupsToUpdate = append(groupsToUpdate, struct {
				marketID int
				tabID    string
			}{marketID, tabID})
		}
	}

	// 批量更新
	for _, item := range groupsToUpdate {
		_, err := s.db.Exec(`
			UPDATE markets
			SET tab_id = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`, item.tabID, item.marketID)

		if err != nil {
			log.Printf("更新市场 %d 失败: %v", item.marketID, err)
			continue
		}
		totalAssigned++
	}

	log.Printf("✓ 基于 groups 分配了 %d 个市场的 tabs\n", totalAssigned)
	return totalAssigned, nil
}

// AssignTabsBasedOnSpecifiers 基于 specifiers 为市场分配 tabs
func (s *MarketGroupsService) AssignTabsBasedOnSpecifiers() (int, error) {
	log.Println("开始基于 specifiers 分配 tabs...")

	// 定义 specifier 到 tab 的映射
	specifierTabMapping := map[string]string{
		"inningnr":   "innings",
		"setnr":      "sets",
		"mapnr":      "maps",
		"quarternr":  "quarters",
		"periodnr":   "periods",
		"framenr":    "frames",
		"overnr":     "overs",
		"drivenr":    "drives",
	}

	totalAssigned := 0

	for specifier, tabID := range specifierTabMapping {
		result, err := s.db.Exec(`
			UPDATE markets
			SET tab_id = $1, updated_at = CURRENT_TIMESTAMP
			WHERE (tab_id IS NULL OR tab_id = '')
			AND specifiers LIKE $2
		`, tabID, "%"+specifier+"%")

		if err != nil {
			log.Printf("更新 specifier %s 失败: %v", specifier, err)
			continue
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("获取受影响行数失败: %v", err)
			continue
		}

		if rowsAffected > 0 {
			log.Printf("  %s → %s : %d 个市场", specifier, tabID, rowsAffected)
			totalAssigned += int(rowsAffected)
		}
	}

	log.Printf("✓ 基于 specifiers 分配了 %d 个市场的 tabs\n", totalAssigned)
	return totalAssigned, nil
}

// AssignDefaultTab 为未分配的市场分配默认 tab
func (s *MarketGroupsService) AssignDefaultTab(defaultTab string) (int, error) {
	log.Printf("开始分配默认 tab: %s...\n", defaultTab)

	result, err := s.db.Exec(`
		UPDATE markets
		SET tab_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE tab_id IS NULL OR tab_id = ''
	`, defaultTab)

	if err != nil {
		return 0, fmt.Errorf("分配默认 tab 失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败: %w", err)
	}

	log.Printf("✓ 分配了 %d 个市场的默认 tab\n", rowsAffected)
	return int(rowsAffected), nil
}

// GetMarketStats 获取市场统计信息
func (s *MarketGroupsService) GetMarketStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总市场数
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM markets").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	// 已分配的市场数
	var mapped int
	err = s.db.QueryRow("SELECT COUNT(*) FROM markets WHERE tab_id IS NOT NULL AND tab_id != ''").Scan(&mapped)
	if err != nil {
		return nil, err
	}
	stats["mapped"] = mapped

	// 未分配的市场数
	var unmapped int
	err = s.db.QueryRow("SELECT COUNT(*) FROM markets WHERE tab_id IS NULL OR tab_id = ''").Scan(&unmapped)
	if err != nil {
		return nil, err
	}
	stats["unmapped"] = unmapped

	// 有 groups 的市场数
	var withGroups int
	err = s.db.QueryRow("SELECT COUNT(*) FROM markets WHERE groups IS NOT NULL AND groups != ''").Scan(&withGroups)
	if err != nil {
		return nil, err
	}
	stats["with_groups"] = withGroups

	// Tab 分布
	rows, err := s.db.Query(`
		SELECT tab_id, COUNT(*) as count
		FROM markets
		WHERE tab_id IS NOT NULL
		GROUP BY tab_id
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tabDistribution := make(map[string]int)
	for rows.Next() {
		var tabID string
		var count int
		if err := rows.Scan(&tabID, &count); err != nil {
			continue
		}
		tabDistribution[tabID] = count
	}
	stats["tab_distribution"] = tabDistribution

	return stats, nil
}

// LogAssignment 记录分配操作
func (s *MarketGroupsService) LogAssignment(marketID int, eventID, oldTabID, newTabID, assignmentType, assignedBy string) error {
	_, err := s.db.Exec(`
		INSERT INTO market_tab_chip_assignment_log 
		(market_id, event_id, old_tab_id, new_tab_id, assignment_type, assigned_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, marketID, eventID, oldTabID, newTabID, assignmentType, assignedBy, time.Now())

	return err
}
