package web

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	
	"uof-service/services"
)

// handleGetBookedMatches 获取已订阅的比赛列表
func (s *Server) handleGetBookedMatches(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting booked matches...")
	
	// 调用 Betradar API 查询已订阅的比赛
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/booked.xml", s.config.APIBaseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query booked matches: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusInternalServerError)
		return
	}
	
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}
	
	// 解析 XML
	type SportEvent struct {
		ID        string `xml:"id,attr" json:"id"`
		Scheduled string `xml:"scheduled,attr" json:"scheduled"`
		Status    string `xml:"status,attr" json:"status"`
		LiveOdds  string `xml:"liveodds,attr" json:"liveodds"`
	}
	
	type BookingCalendar struct {
		SportEvents []SportEvent `xml:"sport_event" json:"sport_events"`
	}
	
	var calendar BookingCalendar
	if err := xml.Unmarshal(body, &calendar); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse XML: %v", err), http.StatusInternalServerError)
		return
	}
	
	// 返回 JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(calendar.SportEvents),
		"matches": calendar.SportEvents,
	})
}

// handleGetBookableMatches 获取可订阅的比赛列表
func (s *Server) handleGetBookableMatches(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting bookable matches...")
	
	// 查询当前直播赛程
	url := fmt.Sprintf("%s/sports/en/schedules/live/schedule.xml", s.config.APIBaseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query live schedule: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusInternalServerError)
		return
	}
	
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}
	
	// 解析 XML
	type SportEvent struct {
		ID        string `xml:"id,attr" json:"id"`
		Scheduled string `xml:"scheduled,attr" json:"scheduled"`
		Status    string `xml:"status,attr" json:"status"`
		LiveOdds  string `xml:"liveodds,attr" json:"liveodds"`
	}
	
	type Schedule struct {
		SportEvents []SportEvent `xml:"sport_event" json:"sport_events"`
	}
	
	var schedule Schedule
	if err := xml.Unmarshal(body, &schedule); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse XML: %v", err), http.StatusInternalServerError)
		return
	}
	
	// 筛选 bookable 的比赛
	var bookableMatches []SportEvent
	for _, event := range schedule.SportEvents {
		if event.LiveOdds == "bookable" {
			bookableMatches = append(bookableMatches, event)
		}
	}
	
	// 返回 JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"total_live":     len(schedule.SportEvents),
		"bookable_count": len(bookableMatches),
		"matches":        bookableMatches,
	})
}

// handleTriggerAutoBooking 触发自动订阅
func (s *Server) handleTriggerAutoBooking(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Triggering auto booking...")
	
	startupBooking := services.NewStartupBookingService(s.config, s.db, s.larkNotifier)
	
	result, err := startupBooking.ExecuteStartupBooking()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute auto booking: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"total_live":     result.TotalLive,
		"bookable":       result.Bookable,
		"success_count":  result.Success,
		"failed_count":   result.Failed,
		"verified_count": result.AlreadyBooked,
		"booked_matches": result.BookedMatches,
		"failed_matches": result.FailedMatches,
	})
}

