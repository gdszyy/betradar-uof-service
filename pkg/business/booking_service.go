package business

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"uof-service/pkg/common"
)

// DefaultBookingService 默认预订管理服务实现
type DefaultBookingService struct {
	logger     common.Logger
	apiBaseURL string
	apiToken   string
	httpClient *http.Client
}

// NewBookingService 创建预订管理服务
func NewBookingService(logger common.Logger, apiBaseURL, apiToken string) BookingService {
	return &DefaultBookingService{
		logger:     logger,
		apiBaseURL: apiBaseURL,
		apiToken:   apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// BookMatch 预订比赛
func (s *DefaultBookingService) BookMatch(ctx context.Context, matchID string) error {
	s.logger.Info("Booking match: %s", matchID)

	url := fmt.Sprintf("%s/v1/liveodds/booking-calendar/events/%s/book", s.apiBaseURL, matchID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		s.logger.Error("Failed to create booking request: %v", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to book match: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("Booking failed with status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("booking failed with status %d", resp.StatusCode)
	}

	s.logger.Info("Successfully booked match: %s", matchID)
	return nil
}

// UnbookMatch 取消预订
func (s *DefaultBookingService) UnbookMatch(ctx context.Context, matchID string) error {
	s.logger.Info("Unbooking match: %s", matchID)

	url := fmt.Sprintf("%s/v1/liveodds/booking-calendar/events/%s/unbook", s.apiBaseURL, matchID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		s.logger.Error("Failed to create unbooking request: %v", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to unbook match: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("Unbooking failed with status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("unbooking failed with status %d", resp.StatusCode)
	}

	s.logger.Info("Successfully unbooked match: %s", matchID)
	return nil
}

// GetBookableMatches 获取可预订的比赛
func (s *DefaultBookingService) GetBookableMatches(ctx context.Context, date time.Time) ([]*BookableMatch, error) {
	s.logger.Debug("Getting bookable matches for date: %s", date.Format("2006-01-02"))

	url := fmt.Sprintf("%s/v1/liveodds/booking-calendar/events/%s/schedule.json",
		s.apiBaseURL,
		date.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.logger.Error("Failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to get bookable matches: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("Get bookable matches failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed with status %d", resp.StatusCode)
	}

	var response struct {
		Events []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Scheduled string `json:"scheduled"`
			Status    string `json:"status"`
			Sport     struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"sport"`
		} `json:"events"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		s.logger.Error("Failed to decode response: %v", err)
		return nil, err
	}

	matches := make([]*BookableMatch, 0, len(response.Events))
	for _, event := range response.Events {
		scheduled, _ := time.Parse(time.RFC3339, event.Scheduled)

		matches = append(matches, &BookableMatch{
			ID:        event.ID,
			Name:      event.Name,
			Scheduled: scheduled,
			Status:    event.Status,
			SportID:   event.Sport.ID,
			SportName: event.Sport.Name,
		})
	}

	s.logger.Debug("Retrieved %d bookable matches", len(matches))
	return matches, nil
}

// BookAllBookableMatches 预订所有可预订的比赛
func (s *DefaultBookingService) BookAllBookableMatches(ctx context.Context, date time.Time) (int, error) {
	s.logger.Info("Booking all bookable matches for date: %s", date.Format("2006-01-02"))

	matches, err := s.GetBookableMatches(ctx, date)
	if err != nil {
		return 0, err
	}

	bookedCount := 0
	for _, match := range matches {
		// 只预订状态为 "bookable" 的比赛
		if match.Status != "bookable" {
			continue
		}

		if err := s.BookMatch(ctx, match.ID); err != nil {
			s.logger.Error("Failed to book match %s: %v", match.ID, err)
			continue
		}

		bookedCount++
		time.Sleep(100 * time.Millisecond) // 避免请求过快
	}

	s.logger.Info("Successfully booked %d/%d matches", bookedCount, len(matches))
	return bookedCount, nil
}

// GetBookedMatches 获取已预订的比赛
func (s *DefaultBookingService) GetBookedMatches(ctx context.Context) ([]*BookableMatch, error) {
	s.logger.Debug("Getting booked matches")

	// 获取今天和明天的比赛
	today := time.Now()
	tomorrow := today.Add(24 * time.Hour)

	todayMatches, err := s.GetBookableMatches(ctx, today)
	if err != nil {
		return nil, err
	}

	tomorrowMatches, err := s.GetBookableMatches(ctx, tomorrow)
	if err != nil {
		return nil, err
	}

	// 合并并过滤已预订的比赛
	allMatches := append(todayMatches, tomorrowMatches...)
	bookedMatches := make([]*BookableMatch, 0)

	for _, match := range allMatches {
		if match.Status == "booked" {
			bookedMatches = append(bookedMatches, match)
		}
	}

	s.logger.Debug("Found %d booked matches", len(bookedMatches))
	return bookedMatches, nil
}

// BookableMatch 可预订的比赛
type BookableMatch struct {
	ID        string
	Name      string
	Scheduled time.Time
	Status    string // "bookable", "booked", "not_bookable"
	SportID   string
	SportName string
}

