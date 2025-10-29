package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	"uof-service/config"
)

// PrematchService Pre-match 赛事订阅服务
type PrematchService struct {
	config *config.Config
	db     *sql.DB
	client *http.Client
}

// NewPrematchService 创建 Pre-match 服务
func NewPrematchService(cfg *config.Config, db *sql.DB) *PrematchService {
	return &PrematchService{
		config: cfg,
		db:     db,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// PrematchEvent Pre-match 赛事结构
type PrematchEvent struct {
	ID        string `xml:"id,attr"`
	Scheduled string `xml:"scheduled,attr"`
	Status    string `xml:"status,attr"`
	LiveOdds  string `xml:"liveodds,attr"`
}

// PrematchResult Pre-match 订阅结果
type PrematchResult struct {
	TotalEvents   int
	Bookable      int
	AlreadyBooked int
	Success       int
	Failed        int
}

// FetchPrematchEvents 获取所有 pre-match 赛事
func (s *PrematchService) FetchPrematchEvents() ([]PrematchEvent, error) {
	logger.Println("[PrematchService] 🔍 Fetching pre-match events...")
	
	var allEvents []PrematchEvent
	start := 0
	limit := 1000
	maxPages := 10 // 最多获取 10 页,避免无限循环
	
	for page := 0; page < maxPages; page++ {
		url := fmt.Sprintf("%s/sports/en/schedules/pre/schedule.xml?start=%d&limit=%d", 
			s.config.APIBaseURL, start, limit)
		
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		req.Header.Set("x-access-token", s.config.AccessToken)
		
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to query pre-match schedule: %w", err)
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
		}
		
		type Schedule struct {
			SportEvents []PrematchEvent `xml:"sport_event"`
		}
		
		var schedule Schedule
		if err := xml.Unmarshal(body, &schedule); err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}
		
		if len(schedule.SportEvents) == 0 {
			break
		}
		
		allEvents = append(allEvents, schedule.SportEvents...)
		
		logger.Printf("[PrematchService] 📄 Fetched page %d: %d events (start=%d)", 
			page+1, len(schedule.SportEvents), start)
		
		if len(schedule.SportEvents) < limit {
			break
		}
		
		start += limit
	}
	
	logger.Printf("[PrematchService] ✅ Total fetched: %d pre-match events", len(allEvents))
	
	return allEvents, nil
}

// StorePrematchEvents 存储 pre-match 赛事到数据库
func (s *PrematchService) StorePrematchEvents(events []PrematchEvent) (int, error) {
	stored := 0
	
	for _, event := range events {
		// 解析 scheduled 时间
		var scheduleTime *time.Time
		if event.Scheduled != "" {
			t, err := time.Parse(time.RFC3339, event.Scheduled)
			if err == nil {
				scheduleTime = &t
			}
		}
		
		// 插入或更新数据库
		query := `
			INSERT INTO tracked_events (
				event_id, sport_id, status, schedule_time, 
				subscribed, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (event_id) DO UPDATE SET
				status = EXCLUDED.status,
				schedule_time = EXCLUDED.schedule_time,
				updated_at = EXCLUDED.updated_at
		`
		
		_, err := s.db.Exec(
			query,
			event.ID,
			"unknown", // sport_id 需要从 fixture 获取
			event.Status,
			scheduleTime,
			false, // 默认未订阅
			time.Now(),
			time.Now(),
		)
		
		if err != nil {
			logger.Printf("[PrematchService] ⚠️  Failed to store %s: %v", event.ID, err)
			continue
		}
		
		stored++
	}
	
	logger.Printf("[PrematchService] 💾 Stored %d events to database", stored)
	
	return stored, nil
}

// BookPrematchEvents 订阅所有可订阅的 pre-match 赛事
func (s *PrematchService) BookPrematchEvents(events []PrematchEvent) (*PrematchResult, error) {
	result := &PrematchResult{
		TotalEvents: len(events),
	}
	
	for _, event := range events {
		if event.LiveOdds == "bookable" {
			result.Bookable++
			
			// 订阅赛事
			if err := s.bookEvent(event.ID); err != nil {
				logger.Printf("[PrematchService] ⚠️  Failed to book %s: %v", event.ID, err)
				result.Failed++
			} else {
				result.Success++
			}
		} else if event.LiveOdds == "booked" {
			result.AlreadyBooked++
			
			// 更新数据库状态
			_, err := s.db.Exec(
				"UPDATE tracked_events SET subscribed = true, updated_at = $1 WHERE event_id = $2",
				time.Now(), event.ID,
			)
			if err != nil {
				logger.Printf("[PrematchService] ⚠️  Failed to update database for %s: %v", event.ID, err)
			}
		}
	}
	
	logger.Printf("[PrematchService] 📊 Booking completed: %d bookable, %d already booked, %d success, %d failed",
		result.Bookable, result.AlreadyBooked, result.Success, result.Failed)
	
	return result, nil
}

// bookEvent 订阅单个赛事
func (s *PrematchService) bookEvent(eventID string) error {
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/%s/book", 
		s.config.APIBaseURL, eventID)
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 更新数据库
	_, err = s.db.Exec(
		"UPDATE tracked_events SET subscribed = true, updated_at = $1 WHERE event_id = $2",
		time.Now(), eventID,
	)
	if err != nil {
		logger.Printf("[PrematchService] ⚠️  Failed to update database for %s: %v", eventID, err)
	}
	
	logger.Printf("[PrematchService] ✅ Successfully booked %s", eventID)
	
	return nil
}

// ExecutePrematchBooking 执行完整的 pre-match 订阅流程
func (s *PrematchService) ExecutePrematchBooking() (*PrematchResult, error) {
	logger.Println("[PrematchService] 🚀 Starting pre-match booking...")
	
	// 1. 获取 pre-match 赛事
	events, err := s.FetchPrematchEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	
	// 2. 存储到数据库
	stored, err := s.StorePrematchEvents(events)
	if err != nil {
		logger.Printf("[PrematchService] ⚠️  Failed to store events: %v", err)
	} else {
		logger.Printf("[PrematchService] ✅ Stored %d events", stored)
	}
	
	// 3. 订阅赛事
	result, err := s.BookPrematchEvents(events)
	if err != nil {
		return nil, fmt.Errorf("failed to book events: %w", err)
	}
	
	logger.Printf("[PrematchService] 🎉 Pre-match booking completed: %d/%d successful",
		result.Success, result.Bookable)
	
	return result, nil
}

