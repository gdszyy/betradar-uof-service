package services

import (
	"log"
	"time"
)

// AutoBookingScheduler 自动订阅调度器
type AutoBookingScheduler struct {
	autoBooking *AutoBookingService
	interval    time.Duration
	stopChan    chan struct{}
	running     bool
}

// NewAutoBookingScheduler 创建自动订阅调度器
func NewAutoBookingScheduler(autoBooking *AutoBookingService, interval time.Duration) *AutoBookingScheduler {
	return &AutoBookingScheduler{
		autoBooking: autoBooking,
		interval:    interval,
		stopChan:    make(chan struct{}),
		running:     false,
	}
}

// Start 启动调度器
func (s *AutoBookingScheduler) Start() {
	if s.running {
		log.Println("[AutoBookingScheduler] Already running")
		return
	}
	
	s.running = true
	log.Printf("[AutoBookingScheduler] 🚀 Started with interval: %v", s.interval)
	
	// 立即执行一次
	go func() {
		log.Println("[AutoBookingScheduler] 🔄 Running initial auto-booking...")
		bookable, success, err := s.autoBooking.BookAllBookableMatches()
		if err != nil {
			log.Printf("[AutoBookingScheduler] ❌ Initial auto-booking failed: %v", err)
		} else {
			log.Printf("[AutoBookingScheduler] ✅ Initial auto-booking completed: %d bookable, %d success", bookable, success)
		}
	}()
	
	// 定期执行
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				log.Println("[AutoBookingScheduler] 🔄 Running scheduled auto-booking...")
				bookable, success, err := s.autoBooking.BookAllBookableMatches()
				if err != nil {
					log.Printf("[AutoBookingScheduler] ❌ Scheduled auto-booking failed: %v", err)
				} else {
					log.Printf("[AutoBookingScheduler] ✅ Scheduled auto-booking completed: %d bookable, %d success", bookable, success)
				}
			case <-s.stopChan:
				log.Println("[AutoBookingScheduler] 🛑 Stopped")
				return
			}
		}
	}()
}

// Stop 停止调度器
func (s *AutoBookingScheduler) Stop() {
	if !s.running {
		return
	}
	
	s.running = false
	close(s.stopChan)
	log.Println("[AutoBookingScheduler] 🛑 Stopping...")
}

// IsRunning 检查是否正在运行
func (s *AutoBookingScheduler) IsRunning() bool {
	return s.running
}

