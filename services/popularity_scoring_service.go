package services

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"uof-service/logger"
)

// PopularityScoringService 热度评分服务
type PopularityScoringService struct {
	db              *sql.DB
	larkNotifier    *LarkNotifier
	scriptDir       string
	updateInterval  time.Duration
	executionHour   int // 每天执行的小时（0-23）
}

// PopularityScoringResult 热度评分执行结果
type PopularityScoringResult struct {
	EventsUpdated      int64
	TournamentsUpdated int64
	ExecutionTime      time.Duration
	Error              error
}

// NewPopularityScoringService 创建热度评分服务
func NewPopularityScoringService(db *sql.DB, larkNotifier *LarkNotifier) *PopularityScoringService {
	// 获取脚本目录路径
	scriptDir := os.Getenv("POPULARITY_SCRIPT_DIR")
	if scriptDir == "" {
		// 尝试多个可能的路径
		possiblePaths := []string{
			"./scripts",           // 本地开发环境
			"/app/scripts",        // Railway 部署环境
			"../scripts",          // 备用路径
		}
		
		for _, path := range possiblePaths {
			if _, err := os.Stat(filepath.Join(path, "calculate_event_popularity.sql")); err == nil {
				scriptDir = path
				break
			}
		}
		
		if scriptDir == "" {
			logger.Println("[PopularityScoring] ⚠️  Warning: Could not find scripts directory, using default ./scripts")
			scriptDir = "./scripts"
		}
	}
	
	// 获取执行时间配置（默认凌晨 2 点）
	executionHour := 2
	
	return &PopularityScoringService{
		db:             db,
		larkNotifier:   larkNotifier,
		scriptDir:      scriptDir,
		updateInterval: 24 * time.Hour,
		executionHour:  executionHour,
	}
}

// Start 启动热度评分服务
func (s *PopularityScoringService) Start() error {
	logger.Println("[PopularityScoring] Starting popularity scoring service...")
	
	// 启动时立即执行一次（延迟 30 秒，等待其他服务初始化）
	go func() {
		time.Sleep(30 * time.Second)
		logger.Println("[PopularityScoring] 🚀 Executing initial popularity scoring...")
		
		if result := s.ExecuteScoring(); result.Error != nil {
			logger.Errorf("[PopularityScoring] ❌ Initial scoring failed: %v", result.Error)
			if s.larkNotifier != nil {
				s.larkNotifier.NotifyError("Popularity Scoring", fmt.Sprintf("Initial scoring failed: %v", result.Error))
			}
		} else {
			logger.Printf("[PopularityScoring] ✅ Initial scoring completed: %d events, %d tournaments updated in %s",
				result.EventsUpdated, result.TournamentsUpdated, result.ExecutionTime)
			if s.larkNotifier != nil {
				s.notifySuccess(result)
			}
		}
	}()
	
	// 启动定时任务
	go s.scheduleDaily()
	
	logger.Printf("[PopularityScoring] ✅ Popularity scoring service started (daily at %d:00)", s.executionHour)
	return nil
}

// scheduleDaily 每天定时执行
func (s *PopularityScoringService) scheduleDaily() {
	for {
		// 计算到下一个执行时间的延迟
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), s.executionHour, 0, 0, 0, now.Location())
		
		// 如果今天的执行时间已过，设置为明天
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}
		
		delay := time.Until(nextRun)
		logger.Printf("[PopularityScoring] Next scoring scheduled at %s (in %s)",
			nextRun.Format("2006-01-02 15:04:05"), delay.Round(time.Minute))
		
		// 等待到执行时间
		time.Sleep(delay)
		
		// 执行评分
		logger.Println("[PopularityScoring] 🔄 Daily popularity scoring triggered")
		
		if result := s.ExecuteScoring(); result.Error != nil {
			logger.Errorf("[PopularityScoring] ❌ Daily scoring failed: %v", result.Error)
			if s.larkNotifier != nil {
				s.larkNotifier.NotifyError("Popularity Scoring", fmt.Sprintf("Daily scoring failed: %v", result.Error))
			}
		} else {
			logger.Printf("[PopularityScoring] ✅ Daily scoring completed: %d events, %d tournaments updated in %s",
				result.EventsUpdated, result.TournamentsUpdated, result.ExecutionTime)
			if s.larkNotifier != nil {
				s.notifySuccess(result)
			}
		}
	}
}

// ExecuteScoring 执行热度评分计算
func (s *PopularityScoringService) ExecuteScoring() PopularityScoringResult {
	startTime := time.Now()
	result := PopularityScoringResult{}
	
	// 1. 执行比赛热度评分
	eventsUpdated, err := s.executeEventScoring()
	if err != nil {
		result.Error = fmt.Errorf("event scoring failed: %w", err)
		return result
	}
	result.EventsUpdated = eventsUpdated
	
	// 2. 执行联赛热度评分
	tournamentsUpdated, err := s.executeTournamentScoring()
	if err != nil {
		result.Error = fmt.Errorf("tournament scoring failed: %w", err)
		return result
	}
	result.TournamentsUpdated = tournamentsUpdated
	
	result.ExecutionTime = time.Since(startTime)
	return result
}

// executeEventScoring 执行比赛热度评分
func (s *PopularityScoringService) executeEventScoring() (int64, error) {
	scriptPath := filepath.Join(s.scriptDir, "calculate_event_popularity.sql")
	
	// 读取 SQL 脚本
	sqlContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read event scoring script: %w", err)
	}
	
	// 执行 SQL
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	result, err := tx.Exec(string(sqlContent))
	if err != nil {
		return 0, fmt.Errorf("failed to execute event scoring SQL: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	logger.Printf("[PopularityScoring] Event scoring completed: %d events updated", rowsAffected)
	
	return rowsAffected, nil
}

// executeTournamentScoring 执行联赛热度评分
func (s *PopularityScoringService) executeTournamentScoring() (int64, error) {
	scriptPath := filepath.Join(s.scriptDir, "calculate_tournament_popularity.sql")
	
	// 读取 SQL 脚本
	sqlContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read tournament scoring script: %w", err)
	}
	
	// 执行 SQL
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	result, err := tx.Exec(string(sqlContent))
	if err != nil {
		return 0, fmt.Errorf("failed to execute tournament scoring SQL: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	logger.Printf("[PopularityScoring] Tournament scoring completed: %d tournaments updated", rowsAffected)
	
	return rowsAffected, nil
}

// notifySuccess 发送成功通知
func (s *PopularityScoringService) notifySuccess(result PopularityScoringResult) {
	message := fmt.Sprintf(
		"热度评分计算完成\n\n"+
			"✅ 比赛评分更新: %d 场\n"+
			"✅ 联赛评分更新: %d 个\n"+
			"⏱️  执行时间: %s\n"+
			"🕐 执行时间: %s",
		result.EventsUpdated,
		result.TournamentsUpdated,
		result.ExecutionTime.Round(time.Second),
		time.Now().Format("2006-01-02 15:04:05"),
	)
	
	// 这里可以调用 larkNotifier 的通用通知方法
	// 由于 LarkNotifier 可能没有通用的 Notify 方法，我们使用 NotifyError 的格式
	logger.Println("[PopularityScoring] " + message)
}
