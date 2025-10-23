package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// LDEventHandler LD 事件处理器
type LDEventHandler struct {
	db                   *sql.DB
	larkNotifier         *LarkNotifier
	subscriptionManager  *MatchSubscriptionManager
}

// NewLDEventHandler 创建事件处理器
func NewLDEventHandler(db *sql.DB, notifier *LarkNotifier) *LDEventHandler {
	return &LDEventHandler{
		db:           db,
		larkNotifier: notifier,
	}
}

// SetSubscriptionManager 设置订阅管理器
func (h *LDEventHandler) SetSubscriptionManager(manager *MatchSubscriptionManager) {
	h.subscriptionManager = manager
}

// HandleEvent 处理事件
func (h *LDEventHandler) HandleEvent(event *LDEvent) {
	// 保存到数据库
	// 注意: LDEvent 结构体需要添加 MatchID 和 SportID 字段
	// 这些字段需要从比赛信息中获取
	// 暂时使用空值
	query := `
		INSERT INTO ld_events (
			uuid, event_id, match_id, sport_id, type, type_name,
			info, side, mtime, stime, match_status,
			t1_score, t2_score, player1, player2, extra_info, is_important
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (uuid) DO NOTHING
	`
	
	typeName := GetEventTypeName(event.Type)
	isImportant := IsImportantEvent(event.Type)
	
	_, err := h.db.Exec(query,
		event.UUID, event.ID, event.MatchID, event.SportID, event.Type, typeName,
		event.Info, event.Side, event.MTime, event.STime, event.MatchStatus,
		event.T1Score, event.T2Score, event.Player1, event.Player2, event.ExtraInfo, isImportant,
	)
	
	if err != nil {
		log.Printf("[LD] ❌ Failed to save event: %v", err)
		return
	}
	
	log.Printf("[LD] ✅ Event saved: %s (%s)", typeName, event.UUID)
	
	// 记录事件到订阅管理器
	if h.subscriptionManager != nil && event.MatchID != "" {
		h.subscriptionManager.RecordEvent(event.MatchID)
		
		// 如果是比赛结束事件，更新状态
		if event.MatchStatus == "ended" || event.MatchStatus == "closed" {
			h.subscriptionManager.UpdateMatchStatus(event.MatchID, event.MatchStatus)
		}
	}
	
	// 发送重要事件通知
	if isImportant && h.larkNotifier != nil {
		h.sendEventNotification(event, typeName)
	}
}

// HandleMatchInfo 处理比赛信息
func (h *LDEventHandler) HandleMatchInfo(matchInfo *LDMatchInfo) {
	// 先尝试更新
	updateQuery := `
		UPDATE ld_matches SET
			match_status = $1,
			match_time = $2,
			t1_score = $3,
			t2_score = $4,
			coverage_type = $5,
			device_id = $6,
			last_event_at = NOW(),
			updated_at = NOW()
		WHERE match_id = $7
	`
	
	result, err := h.db.Exec(updateQuery,
		matchInfo.MatchStatus, matchInfo.MatchTime,
		matchInfo.T1Score, matchInfo.T2Score,
		matchInfo.CoverageType, matchInfo.DeviceID,
		matchInfo.MatchID,
	)
	
	if err != nil {
		log.Printf("[LD] ❌ Failed to update match info: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	
	if rowsAffected == 0 {
		// 如果没有更新任何行,说明是新比赛,插入
		insertQuery := `
			INSERT INTO ld_matches (
				match_id, sport_id, t1_id, t2_id, t1_name, t2_name,
				match_status, match_time, t1_score, t2_score,
				match_date, start_time, coverage_type, device_id,
				subscribed, last_event_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW())
		`
		
		_, err = h.db.Exec(insertQuery,
			matchInfo.MatchID, matchInfo.SportID,
			matchInfo.T1ID, matchInfo.T2ID,
			matchInfo.T1Name, matchInfo.T2Name,
			matchInfo.MatchStatus, matchInfo.MatchTime,
			matchInfo.T1Score, matchInfo.T2Score,
			matchInfo.MatchDate, matchInfo.StartTime,
			matchInfo.CoverageType, matchInfo.DeviceID,
			true,
		)
		
		if err != nil {
			log.Printf("[LD] ❌ Failed to insert match info: %v", err)
			return
		}
		
		log.Printf("[LD] ✅ Match info saved: %s vs %s", matchInfo.T1Name, matchInfo.T2Name)
	} else {
		log.Printf("[LD] ✅ Match info updated: %s vs %s (%d-%d)",
			matchInfo.T1Name, matchInfo.T2Name, matchInfo.T1Score, matchInfo.T2Score)
	}
	
	// 更新订阅管理器中的比赛状态
	if h.subscriptionManager != nil && matchInfo.MatchID != "" {
		h.subscriptionManager.UpdateMatchStatus(matchInfo.MatchID, matchInfo.MatchStatus)
	}
}

// HandleLineup 处理阵容
func (h *LDEventHandler) HandleLineup(lineup *LDLineup) {
	// 将球员列表转换为 JSON
	team1JSON, _ := json.Marshal(lineup.Team1.Players)
	team2JSON, _ := json.Marshal(lineup.Team2.Players)
	
	// 先尝试更新
	updateQuery := `
		UPDATE ld_lineups SET
			team1_players = $1,
			team2_players = $2,
			updated_at = NOW()
		WHERE match_id = $3
	`
	
	result, err := h.db.Exec(updateQuery, string(team1JSON), string(team2JSON), lineup.MatchID)
	if err != nil {
		log.Printf("[LD] ❌ Failed to update lineup: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	
	if rowsAffected == 0 {
		// 如果没有更新任何行,插入新记录
		insertQuery := `
			INSERT INTO ld_lineups (match_id, team1_players, team2_players)
			VALUES ($1, $2, $3)
		`
		
		_, err = h.db.Exec(insertQuery, lineup.MatchID, string(team1JSON), string(team2JSON))
		if err != nil {
			log.Printf("[LD] ❌ Failed to insert lineup: %v", err)
			return
		}
		
		log.Printf("[LD] ✅ Lineup saved for match %s", lineup.MatchID)
	} else {
		log.Printf("[LD] ✅ Lineup updated for match %s", lineup.MatchID)
	}
}

// sendEventNotification 发送事件通知到飞书
func (h *LDEventHandler) sendEventNotification(event *LDEvent, typeName string) {
	// 获取比赛信息
	var t1Name, t2Name string
	query := `SELECT t1_name, t2_name FROM ld_matches WHERE match_id = $1`
	err := h.db.QueryRow(query, event.MatchID).Scan(&t1Name, &t2Name)
	
	if err != nil {
		log.Printf("[LD] ⚠️  Match not found for event notification: %s", event.MatchID)
		return
	}
	
	// 构建通知消息
	var emoji string
	switch event.Type {
	case EventTypeGoal, EventTypePenaltyGoal:
		emoji = "⚽"
	case EventTypeOwnGoal:
		emoji = "🥅"
	case EventTypeRedCard:
		emoji = "🟥"
	case EventTypeYellowRedCard:
		emoji = "🟨🟥"
	case EventTypeMatchStart:
		emoji = "🏁"
	case EventTypeMatchEnd:
		emoji = "🏁"
	default:
		emoji = "📊"
	}
	
	sideText := ""
	switch event.Side {
	case "home":
		sideText = t1Name
	case "away":
		sideText = t2Name
	default:
		sideText = ""
	}
	
	message := fmt.Sprintf("%s **%s**\n\n", emoji, typeName)
	message += fmt.Sprintf("**比赛**: %s vs %s\n", t1Name, t2Name)
	if sideText != "" {
		message += fmt.Sprintf("**球队**: %s\n", sideText)
	}
	if event.Player1 != "" {
		message += fmt.Sprintf("**球员**: %s\n", event.Player1)
	}
	message += fmt.Sprintf("**时间**: %s\n", event.MTime)
	message += fmt.Sprintf("**比分**: %d - %d\n", event.T1Score, event.T2Score)
	if event.Info != "" {
		message += fmt.Sprintf("**详情**: %s\n", event.Info)
	}
	
	if err := h.larkNotifier.SendText(message); err != nil {
		log.Printf("[LD] ❌ Failed to send event notification: %v", err)
	}
}

