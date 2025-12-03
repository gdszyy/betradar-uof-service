package services

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// TabChipAssignmentService 负责 Tab/Chip 的分类和分配
type TabChipAssignmentService struct {
	db             *sql.DB
	playersService *PlayersService
}

// NewTabChipAssignmentService 创建一个新的 TabChipAssignmentService
func NewTabChipAssignmentService(db *sql.DB, playersService *PlayersService) *TabChipAssignmentService {
	return &TabChipAssignmentService{
		db:             db,
		playersService: playersService,
	}
}

// Start 启动服务并开始定期分配
func (s *TabChipAssignmentService) Start() {
	log.Println("[TabChipService] Starting Tab/Chip Assignment Service...")

	// 启动后立即运行一次
	go s.RunAssignment()

	// 每天定期运行
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			s.RunAssignment()
		}
	}()
}

// RunAssignment 运行完整的 Tab/Chip 分配逻辑
func (s *TabChipAssignmentService) RunAssignment() {
	log.Println("[TabChipService] Running full Tab/Chip assignment...")

	if s.db == nil {
		log.Println("[TabChipService] ⚠️  Database not available, skipping assignment")
		return
	}

	// 1. 分配 Tab ID
	if err := s.assignTabs(); err != nil {
		log.Printf("[TabChipService] ⚠️  Failed to assign tabs: %v", err)
		return
	}

	// 2. 分配 Chip ID
	if err := s.assignChips(); err != nil {
		log.Printf("[TabChipService] ⚠️  Failed to assign chips: %v", err)
		return
	}

	log.Println("[TabChipService] ✅ Full Tab/Chip assignment completed")
}

// assignTabs 分配 Tab ID
func (s *TabChipAssignmentService) assignTabs() error {
	log.Println("[TabChipService] Assigning Tab IDs...")

	// 使用 CASE 语句进行高效分配
	query := `
	UPDATE markets
	SET tab_id = CASE
		-- 球员属性
		WHEN market_type = 'player_props' THEN 'player_props'
		-- 特殊玩法
		WHEN market_type IN ('combo', 'bookings', 'scorers', 'micro_market') THEN 'special_markets'
		-- 时序相关的市场 (不分配 Tab)
		WHEN specifiers LIKE '%quarternr%' OR specifiers LIKE '%setnr%' OR specifiers LIKE '%periodnr%' OR specifiers LIKE '%inningnr%' OR specifiers LIKE '%framenr%' OR specifiers LIKE '%overnr%' OR specifiers LIKE '%drivenr%' THEN NULL
		-- 其他都归为全场
		ELSE 'full_match'
	END,
	updated_at = CURRENT_TIMESTAMP
	`

	result, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to execute tab assignment query: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[TabChipService] ✅ Assigned Tab IDs for %d markets", rowsAffected)
	return nil
}

// assignChips 分配 Chip ID
func (s *TabChipAssignmentService) assignChips() error {
	log.Println("[TabChipService] Assigning Chip IDs...")

	// 1. 分配全场 Chip
	if err := s.assignFullMatchChips(); err != nil {
		return err
	}

	// 2. 分配球员属性 Chip
	if err := s.assignPlayerPropsChips(); err != nil {
		return err
	}

	// 3. 分配特殊玩法 Chip
	if err := s.assignSpecialMarketsChips(); err != nil {
		return err
	}

	return nil
}

// assignFullMatchChips 分配全场 Chip
func (s *TabChipAssignmentService) assignFullMatchChips() error {
	query := `
	UPDATE markets
	SET chip_id = CASE
		WHEN market_type = '1x2' THEN 'chip_1x2'
		WHEN market_type = 'handicap' THEN 'chip_handicap'
		WHEN market_type = 'totals' THEN 'chip_totals'
		WHEN market_type = 'both_teams_to_score' THEN 'chip_both_teams_to_score'
		WHEN market_type = 'asian_handicap' THEN 'chip_asian_handicap'
		WHEN market_type = 'draw_no_bet' THEN 'chip_draw_no_bet'
		ELSE 'chip_other'
	END
	WHERE tab_id = 'full_match'
	`

	_, err := s.db.Exec(query)
	return err
}

// assignPlayerPropsChips 分配球员属性 Chip
func (s *TabChipAssignmentService) assignPlayerPropsChips() error {
	// 查询所有球员属性市场
	rows, err := s.db.Query(`
		SELECT id, specifiers
		FROM markets
		WHERE tab_id = 'player_props'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 准备更新语句
	stmt, err := s.db.Prepare("UPDATE markets SET chip_id = $1 WHERE id = $2")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 遍历并更新
	for rows.Next() {
		var marketID int
		var specifiers string
		if err := rows.Scan(&marketID, &specifiers); err != nil {
			continue
		}

		// 从 specifiers 中提取球员信息
		playerID, playerName := s.extractPlayerInfo(specifiers)
		if playerID != "" {
			chipID := fmt.Sprintf("chip_player_%s_%s", playerID, s.sanitizePlayerName(playerName))
			stmt.Exec(chipID, marketID)
		}
	}

	return nil
}

// assignSpecialMarketsChips 分配特殊玩法 Chip
func (s *TabChipAssignmentService) assignSpecialMarketsChips() error {
	query := `
	UPDATE markets
	SET chip_id = 'chip_' || market_type
	WHERE tab_id = 'special_markets'
	`

	_, err := s.db.Exec(query)
	return err
}

// extractPlayerInfo 从 specifiers 中提取球员信息
func (s *TabChipAssignmentService) extractPlayerInfo(specifiers string) (string, string) {
	// specifiers 格式: "player=sr:competitor:123456"
	pairs := strings.Split(specifiers, "|")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 && parts[0] == "player" {
			playerID := parts[1]
		// 使用 PlayersService 获取球员名称
		if s.playersService != nil {
			playerName := s.playersService.GetPlayerName(playerID)
			if playerName != "" {
				return playerID, playerName
			}
		}
			return playerID, ""
		}
	}
	return "", ""
}

// sanitizePlayerName 清理球员名称以用于 Chip ID
func (s *TabChipAssignmentService) sanitizePlayerName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	// 移除所有非字母数字字符
	reg := regexp.MustCompile("[^a-z0-9_]")
	return reg.ReplaceAllString(name, "")
}
