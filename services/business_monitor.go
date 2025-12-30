package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"uof-service/logger"
)

// ExceptionInfo 异常信息结构
type ExceptionInfo struct {
	Type     string
	EventID  string
	Message  string
	Severity string
}

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

// recordToDB 仅记录异常到数据库
func (m *BusinessMonitor) recordToDB(excType, eventID, message, severity string) bool {
	query := "INSERT INTO exceptions (type, event_id, message, severity) VALUES ($1, $2, $3, $4)"
	_, err := m.db.Exec(query, excType, eventID, message, severity)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to record exception to DB: %v", err)
		return false
	}
	return true
}

// CheckMatchStartStatus 检查比赛是否如期开赛
func (m *BusinessMonitor) CheckMatchStartStatus() []ExceptionInfo {
	logger.Println("[BusinessMonitor] Checking match start status...")
	var newExceptions []ExceptionInfo

	query := `
		SELECT event_id, home_team_name, away_team_name, schedule_time 
		FROM tracked_events 
		WHERE status = 'not_started' 
		AND schedule_time < $1 
		AND schedule_time > $2
	`
	
	now := time.Now()
	threshold := now.Add(-15 * time.Minute)
	lookback := now.Add(-2 * time.Hour)

	rows, err := m.db.Query(query, threshold, lookback)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to query late start matches: %v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var eventID, home, away string
		var scheduleTime time.Time
		if err := rows.Scan(&eventID, &home, &away, &scheduleTime); err != nil {
			continue
		}

		msg := fmt.Sprintf("%s vs %s (预计: %s)", 
			home, away, scheduleTime.Format("15:04:05"))
		
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exceptions WHERE event_id = $1 AND type = 'LATE_START')", eventID).Scan(&exists)
		
		if !exists {
			if m.recordToDB("LATE_START", eventID, msg, "high") {
				newExceptions = append(newExceptions, ExceptionInfo{
					Type:     "LATE_START",
					EventID:  eventID,
					Message:  msg,
					Severity: "high",
				})
			}
		}
	}
	return newExceptions
}

// CheckOddsStagnation 检查赔率是否停滞
func (m *BusinessMonitor) CheckOddsStagnation() []ExceptionInfo {
	logger.Println("[BusinessMonitor] Checking odds stagnation...")
	var newExceptions []ExceptionInfo

	query := `
		SELECT event_id, home_team_name, away_team_name, last_message_at 
		FROM tracked_events 
		WHERE status = 'live' 
		AND (live_odds = 'booked' OR live_odds = 'bookable')
		AND last_message_at < $1
	`
	
	threshold := time.Now().Add(-30 * time.Minute)
	logger.Printf("[BusinessMonitor] Querying odds stagnation with threshold: %v", threshold)
	rows, err := m.db.Query(query, threshold)
	if err != nil {
		logger.Errorf("[BusinessMonitor] Failed to query odds stagnation: %v", err)
		return nil
	}

	// 调试：检查 live_odds 字段的分布情况
	var totalLive int
	m.db.QueryRow("SELECT COUNT(*) FROM tracked_events WHERE status = 'live'").Scan(&totalLive)
	var bookedLive int
	m.db.QueryRow("SELECT COUNT(*) FROM tracked_events WHERE status = 'live' AND live_odds = 'booked'").Scan(&bookedLive)
	var bookableLive int
	m.db.QueryRow("SELECT COUNT(*) FROM tracked_events WHERE status = 'live' AND live_odds = 'bookable'").Scan(&bookableLive)
	var nullLive int
	m.db.QueryRow("SELECT COUNT(*) FROM tracked_events WHERE status = 'live' AND (live_odds IS NULL OR live_odds = '')").Scan(&nullLive)
	logger.Printf("[BusinessMonitor] Debug: total live: %d, booked: %d, bookable: %d, null/empty: %d", totalLive, bookedLive, bookableLive, nullLive)

	defer rows.Close()

	for rows.Next() {
		var eventID, home, away string
		var lastMsgAt time.Time
		if err := rows.Scan(&eventID, &home, &away, &lastMsgAt); err != nil {
			continue
		}

		msg := fmt.Sprintf("%s vs %s (最后消息: %s)", 
			home, away, lastMsgAt.Format("15:04:05"))
		
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exceptions WHERE event_id = $1 AND type = 'ODDS_STAGNATION' AND created_at > $2)", 
			eventID, time.Now().Add(-1 * time.Hour)).Scan(&exists)
		
		if !exists {
			if m.recordToDB("ODDS_STAGNATION", eventID, msg, "medium") {
				newExceptions = append(newExceptions, ExceptionInfo{
					Type:     "ODDS_STAGNATION",
					EventID:  eventID,
					Message:  msg,
					Severity: "medium",
				})
			}
		}
	}
	return newExceptions
}

// CheckMissingSettlements 检查是否缺失结算消息
func (m *BusinessMonitor) CheckMissingSettlements() []ExceptionInfo {
	logger.Println("[BusinessMonitor] Checking missing settlements...")
	var newExceptions []ExceptionInfo

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
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var eventID, home, away string
		var updatedAt time.Time
		if err := rows.Scan(&eventID, &home, &away, &updatedAt); err != nil {
			continue
		}

		msg := fmt.Sprintf("%s vs %s (结束: %s)", 
			home, away, updatedAt.Format("15:04:05"))
		
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exceptions WHERE event_id = $1 AND type = 'MISSING_SETTLEMENT')", eventID).Scan(&exists)
		
		if !exists {
			if m.recordToDB("MISSING_SETTLEMENT", eventID, msg, "high") {
				newExceptions = append(newExceptions, ExceptionInfo{
					Type:     "MISSING_SETTLEMENT",
					EventID:  eventID,
					Message:  msg,
					Severity: "high",
				})
			}
		}
	}
	return newExceptions
}

// SendConsolidatedReport 发送整合后的异常报告
func (m *BusinessMonitor) SendConsolidatedReport(newExceptions []ExceptionInfo) {
	if m.larkNotifier == nil {
		return
	}

	// 如果没有新异常，发送“运行正常”心跳报告
	if len(newExceptions) == 0 {
		var totalLive, bookedLive int
		m.db.QueryRow("SELECT COUNT(*) FROM tracked_events WHERE status = 'live'").Scan(&totalLive)
		m.db.QueryRow("SELECT COUNT(*) FROM tracked_events WHERE status = 'live' AND (live_odds = 'booked' OR live_odds = 'bookable')").Scan(&bookedLive)
		
		msg := fmt.Sprintf("✅ **UOF 业务监控运行正常**\n\n当前未发现业务异常。\n- 正在进行的比赛: %d\n- 监控中的滚球比赛: %d\n\n⏰ 时间: %s", 
			totalLive, bookedLive, time.Now().Format("2006-01-02 15:04:05"))
		m.larkNotifier.SendText(msg)
		return
	}

	// 按类型分组
	grouped := make(map[string][]string)
	for _, ex := range newExceptions {
		grouped[ex.Type] = append(grouped[ex.Type], ex.Message)
	}

	var reportBuilder strings.Builder
	reportBuilder.WriteString("🚨 **业务异常整合告警**\n\n")

	for excType, messages := range grouped {
		// 查询该类型的累积总数 (过去 24 小时)
		var total int
		m.db.QueryRow("SELECT COUNT(*) FROM exceptions WHERE type = $1 AND created_at > $2", 
			excType, time.Now().Add(-24*time.Hour)).Scan(&total)

		typeName := m.getFriendlyTypeName(excType)
		reportBuilder.WriteString(fmt.Sprintf("【%s】累积 %d 条，新增:\n", typeName, total))
		for _, msg := range messages {
			reportBuilder.WriteString(fmt.Sprintf("• %s\n", msg))
		}
		reportBuilder.WriteString("\n")
	}

	reportBuilder.WriteString(fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05")))
	
	m.larkNotifier.SendText(reportBuilder.String())
}

func (m *BusinessMonitor) getFriendlyTypeName(excType string) string {
	switch excType {
	case "LATE_START":
		return "未如期开赛"
	case "ODDS_STAGNATION":
		return "赔率停滞"
	case "MISSING_SETTLEMENT":
		return "缺失结算"
	default:
		return excType
	}
}

// RunOnce 执行一次完整的监控检查
func (m *BusinessMonitor) RunOnce() {
	var allNew []ExceptionInfo
	
	allNew = append(allNew, m.CheckMatchStartStatus()...)
	allNew = append(allNew, m.CheckOddsStagnation()...)
	allNew = append(allNew, m.CheckMissingSettlements()...)
	
	m.SendConsolidatedReport(allNew)
}

// Start 启动监控任务
func (m *BusinessMonitor) Start() {
	// 每 10 分钟检查一次
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			m.RunOnce()
		}
	}()
	
	// 启动时立即执行一次
	go m.RunOnce()
}
