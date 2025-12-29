package services

import (
	"database/sql"
	"fmt"
	"time"
	"uof-service/logger"
)

// BusinessMonitor 业务监控服务
type BusinessMonitor struct {
	db           *sql.DB
	larkNotifier *LarkNotifier
}

// NewBusinessMonitor 创建业务监控服务
func NewBusinessMonitor(db *sql.DB, larkNotifier *LarkNotifier) *BusinessMonitor {
	return &BusinessMonitor{
		db:           db,
		larkNotifier: larkNotifier,
	}
}

// RecordException 记录异常到数据库并报送到飞书
func (m *BusinessMonitor) RecordException(excType, eventID, message, severity string) {
	// 1. 记录到数据库
	query := "INSERT INTO exceptions (type, event_id, message, severity) VALUES ($1, $2, $3, $4)"
	_, err := m.db.Exec(query, excType, eventID, message, severity)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to record exception to DB: %v", err)
	}

	// 2. 报送到飞书 (仅当 severity 为 high 或 critical 时，或者根据需求全部报送)
	if m.larkNotifier != nil {
		alertMsg := fmt.Sprintf("🚨 **业务异常告警**\n类型: %s\n赛事ID: %s\n详情: %s\n级别: %s\n时间: %s",
			excType, eventID, message, severity, time.Now().Format("2006-01-02 15:04:05"))
		m.larkNotifier.NotifyError("BusinessMonitor", alertMsg)
	}
}

// CheckMatchStartStatus 检查比赛是否如期开赛
func (m *BusinessMonitor) CheckMatchStartStatus() {
	logger.Println("[BusinessMonitor] Checking match start status...")

	// 查找已经过了开赛时间 15 分钟，但状态仍为 'not_started' 的比赛
	// 排除已经取消或结束的比赛
	query := `
		SELECT event_id, home_team_name, away_team_name, schedule_time 
		FROM tracked_events 
		WHERE status = 'not_started' 
		AND schedule_time < $1 
		AND schedule_time > $2
	`
	
	// 检查过去 2 小时内应该开始但没开始的比赛
	now := time.Now()
	threshold := now.Add(-15 * time.Minute)
	lookback := now.Add(-2 * time.Hour)

	rows, err := m.db.Query(query, threshold, lookback)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to query late start matches: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var eventID, home, away string
		var scheduleTime time.Time
		if err := rows.Scan(&eventID, &home, &away, &scheduleTime); err != nil {
			continue
		}

		msg := fmt.Sprintf("比赛未如期开赛: %s vs %s (预计开赛时间: %s)", 
			home, away, scheduleTime.Format("2006-01-02 15:04:05"))
		
		// 检查是否已经记录过该异常，避免重复告警
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exceptions WHERE event_id = $1 AND type = 'LATE_START')", eventID).Scan(&exists)
		
		if !exists {
			m.RecordException("LATE_START", eventID, msg, "high")
		}
	}
}

// CheckOddsStagnation 检查赔率是否停滞
func (m *BusinessMonitor) CheckOddsStagnation() {
	logger.Println("[BusinessMonitor] Checking odds stagnation...")

	// 查找状态为 'live' 且超过 30 分钟没有收到消息的比赛
	query := `
		SELECT event_id, home_team_name, away_team_name, last_message_at 
		FROM tracked_events 
		WHERE status = 'live' 
		AND last_message_at < $1
	`
	
	threshold := time.Now().Add(-30 * time.Minute)
	rows, err := m.db.Query(query, threshold)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to query stagnant odds: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var eventID, home, away string
		var lastMsgAt time.Time
		if err := rows.Scan(&eventID, &home, &away, &lastMsgAt); err != nil {
			continue
		}

		msg := fmt.Sprintf("滚球赔率停滞: %s vs %s (最后收到消息时间: %s)", 
			home, away, lastMsgAt.Format("2006-01-02 15:04:05"))
		
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exceptions WHERE event_id = $1 AND type = 'ODDS_STAGNATION' AND created_at > $2)", 
			eventID, time.Now().Add(-1 * time.Hour)).Scan(&exists)
		
		if !exists {
			m.RecordException("ODDS_STAGNATION", eventID, msg, "medium")
		}
	}
}

// CheckMissingSettlements 检查是否缺失结算消息
func (m *BusinessMonitor) CheckMissingSettlements() {
	logger.Println("[BusinessMonitor] Checking missing settlements...")

	// 查找状态为 'ended' 超过 2 小时，但在 bet_settlements 表中没有记录的比赛
	query := `
		SELECT e.event_id, e.home_team_name, e.away_team_name, e.updated_at 
		FROM tracked_events e
		LEFT JOIN bet_settlements s ON e.event_id = s.event_id
		WHERE e.status = 'ended' 
		AND e.updated_at < $1 
		AND e.updated_at > $2
		AND s.id IS NULL
	`
	
	now := time.Now()
	threshold := now.Add(-2 * time.Hour)
	lookback := now.Add(-24 * time.Hour)

	rows, err := m.db.Query(query, threshold, lookback)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to query missing settlements: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var eventID, home, away string
		var updatedAt time.Time
		if err := rows.Scan(&eventID, &home, &away, &updatedAt); err != nil {
			continue
		}

		msg := fmt.Sprintf("赛事结束但缺失结算: %s vs %s (结束时间: %s)", 
			home, away, updatedAt.Format("2006-01-02 15:04:05"))
		
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exceptions WHERE event_id = $1 AND type = 'MISSING_SETTLEMENT')", eventID).Scan(&exists)
		
		if !exists {
			m.RecordException("MISSING_SETTLEMENT", eventID, msg, "high")
		}
	}
}

// Start 启动监控任务
func (m *BusinessMonitor) Start() {
	// 每 10 分钟检查一次
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			m.CheckMatchStartStatus()
			m.CheckOddsStagnation()
			m.CheckMissingSettlements()
		}
	}()
	
	// 启动时立即执行一次
	go func() {
		m.CheckMatchStartStatus()
		m.CheckOddsStagnation()
		m.CheckMissingSettlements()
	}()
}
