package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
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
	log.Println("【步骤 1】从 market_descriptions 同步 groups 到 markets...")

	// 检查 market_descriptions 表是否存在且有数据
	var descCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM market_descriptions").Scan(&descCount)
	if err != nil {
		return 0, fmt.Errorf("检查 market_descriptions 表失败: %w", err)
	}

	log.Printf("  market_descriptions 表中有 %d 条记录\n", descCount)

	if descCount == 0 {
		log.Println("  警告：market_descriptions 表为空，跳过同步")
		return 0, nil
	}

	// 使用 SQL 更新 markets 表中的 groups 字段
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

	log.Printf("  ✓ 成功同步 %d 个市场的 groups\n\n", rowsAffected)
	return int(rowsAffected), nil
}

// AssignTabsBasedOnGroups 基于 groups 为市场分配 tabs
func (s *MarketGroupsService) AssignTabsBasedOnGroups() (int, error) {
	log.Println("【步骤 2】基于 groups 分配 tabs...")

	// 使用 CASE 语句进行高效的批量更新
	result, err := s.db.Exec(`
		UPDATE markets
		SET tab_id = CASE
			WHEN groups LIKE '%regular_play%' THEN 'regular_play'
			WHEN groups LIKE '%player_props%' THEN 'player_props'
			WHEN groups LIKE '%micro_market%' THEN 'micro_market'
			WHEN groups LIKE '%bookings%' THEN 'bookings'
			WHEN groups LIKE '%corners%' THEN 'corners'
			WHEN groups LIKE '%1st_half%' THEN '1st_half'
			WHEN groups LIKE '%combo%' THEN 'combo'
			WHEN groups LIKE '%2nd_half%' THEN '2nd_half'
			WHEN groups LIKE '%scorers%' THEN 'scorers'
			WHEN groups LIKE '%innings%' THEN 'innings'
			WHEN groups LIKE '%sets%' THEN 'sets'
			WHEN groups LIKE '%maps%' THEN 'maps'
			WHEN groups LIKE '%quarters%' THEN 'quarters'
			WHEN groups LIKE '%periods%' THEN 'periods'
			WHEN groups LIKE '%frames%' THEN 'frames'
			WHEN groups LIKE '%overs%' THEN 'overs'
			WHEN groups LIKE '%drives%' THEN 'drives'
		END,
		updated_at = CURRENT_TIMESTAMP
		WHERE (tab_id IS NULL OR tab_id = '')
		AND groups IS NOT NULL AND groups != ''
	`)

	if err != nil {
		return 0, fmt.Errorf("分配 tabs 失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败: %w", err)
	}

	log.Printf("  ✓ 基于 groups 分配了 %d 个市场的 tabs\n\n", rowsAffected)
	return int(rowsAffected), nil
}

// AssignTabsBasedOnSpecifiers 基于 specifiers 为市场分配 tabs
func (s *MarketGroupsService) AssignTabsBasedOnSpecifiers() (int, error) {
	log.Println("【步骤 3】基于 specifiers 分配 tabs...")

	result, err := s.db.Exec(`
		UPDATE markets
		SET tab_id = CASE
			WHEN specifiers LIKE '%inningnr%' THEN 'innings'
			WHEN specifiers LIKE '%setnr%' THEN 'sets'
			WHEN specifiers LIKE '%mapnr%' THEN 'maps'
			WHEN specifiers LIKE '%quarternr%' THEN 'quarters'
			WHEN specifiers LIKE '%periodnr%' THEN 'periods'
			WHEN specifiers LIKE '%framenr%' THEN 'frames'
			WHEN specifiers LIKE '%overnr%' THEN 'overs'
			WHEN specifiers LIKE '%drivenr%' THEN 'drives'
		END,
		updated_at = CURRENT_TIMESTAMP
		WHERE (tab_id IS NULL OR tab_id = '')
		AND specifiers IS NOT NULL AND specifiers != ''
	`)

	if err != nil {
		return 0, fmt.Errorf("分配 tabs 失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取受影响行数失败: %w", err)
	}

	log.Printf("  ✓ 基于 specifiers 分配了 %d 个市场的 tabs\n\n", rowsAffected)
	return int(rowsAffected), nil
}

// AssignDefaultTab 为未分配的市场分配默认 tab
func (s *MarketGroupsService) AssignDefaultTab(defaultTab string) (int, error) {
	log.Printf("【步骤 4】分配默认 tab: %s...\n", defaultTab)

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

	log.Printf("  ✓ 分配了 %d 个市场的默认 tab\n\n", rowsAffected)
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

	return stats, nil
}

// GetTabDistribution 获取 Tab 分布
func (s *MarketGroupsService) GetTabDistribution() (map[string]int, error) {
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

	distribution := make(map[string]int)
	for rows.Next() {
		var tabID string
		var count int
		if err := rows.Scan(&tabID, &count); err != nil {
			continue
		}
		distribution[tabID] = count
	}

	return distribution, nil
}

func main() {
	// 获取数据库连接字符串
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("错误：未设置 DATABASE_URL 环境变量")
	}

	// 连接数据库
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("错误：无法连接到数据库: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("错误：无法 ping 数据库: %v", err)
	}

	log.Println("=" * 60)
	log.Println("市场 Groups 和 Tabs 分配")
	log.Println("=" * 60)
	log.Printf("开始时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	service := NewMarketGroupsService(db)

	// 获取初始统计
	log.Println("【初始状态】")
	stats, err := service.GetMarketStats()
	if err != nil {
		log.Fatalf("获取统计信息失败: %v", err)
	}
	log.Printf("  总市场数：%d\n", stats["total"])
	log.Printf("  已分配：%d\n", stats["mapped"])
	log.Printf("  未分配：%d\n", stats["unmapped"])
	log.Printf("  有 groups：%d\n\n", stats["with_groups"])

	// 步骤 1：同步 groups
	_, err = service.SyncMarketGroupsFromDescriptions()
	if err != nil {
		log.Printf("警告：同步 groups 失败: %v\n", err)
	}

	// 步骤 2：基于 groups 分配 tabs
	_, err = service.AssignTabsBasedOnGroups()
	if err != nil {
		log.Fatalf("错误：基于 groups 分配 tabs 失败: %v", err)
	}

	// 步骤 3：基于 specifiers 分配 tabs
	_, err = service.AssignTabsBasedOnSpecifiers()
	if err != nil {
		log.Fatalf("错误：基于 specifiers 分配 tabs 失败: %v", err)
	}

	// 步骤 4：分配默认 tab
	_, err = service.AssignDefaultTab("regular_play")
	if err != nil {
		log.Fatalf("错误：分配默认 tab 失败: %v", err)
	}

	// 获取最终统计
	log.Println("【最终统计】")
	stats, err = service.GetMarketStats()
	if err != nil {
		log.Fatalf("获取统计信息失败: %v", err)
	}
	log.Printf("  总市场数：%d\n", stats["total"])
	log.Printf("  已分配：%d\n", stats["mapped"])
	log.Printf("  未分配：%d\n", stats["unmapped"])
	log.Printf("  映射率：%.2f%%\n\n", float64(stats["mapped"].(int))/float64(stats["total"].(int))*100)

	// 显示 Tab 分布
	log.Println("【Tab 分布】")
	distribution, err := service.GetTabDistribution()
	if err != nil {
		log.Fatalf("获取 Tab 分布失败: %v", err)
	}

	log.Println("  Tab ID          | 市场数")
	log.Println("  " + "-"*40)
	for tabID, count := range distribution {
		log.Printf("  %-15s | %d\n", tabID, count)
	}

	log.Printf("\n✓ 分配完成！\n")
	log.Printf("结束时间：%s\n", time.Now().Format("2006-01-02 15:04:05"))
}
