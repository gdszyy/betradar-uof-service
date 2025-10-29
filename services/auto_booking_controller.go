package services

import (
	"sync"
	"time"
	"uof-service/config"
	"uof-service/logger"
)

// AutoBookingController 自动订阅控制器
type AutoBookingController struct {
	config          *config.Config
	service         *AutoBookingService
	enabled         bool
	intervalMinutes int
	ticker          *time.Ticker
	stopChan        chan struct{}
	mu              sync.RWMutex
	running         bool
}

// NewAutoBookingController 创建自动订阅控制器
func NewAutoBookingController(cfg *config.Config, service *AutoBookingService) *AutoBookingController {
	return &AutoBookingController{
		config:          cfg,
		service:         service,
		enabled:         cfg.AutoBookingEnabled,
		intervalMinutes: cfg.AutoBookingIntervalMinutes,
		stopChan:        make(chan struct{}),
	}
}

// Start 启动自动订阅服务
func (c *AutoBookingController) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		logger.Println("[AutoBookingController] Already running")
		return
	}

	if !c.enabled {
		logger.Println("[AutoBookingController] Auto-booking is disabled")
		return
	}

	logger.Printf("[AutoBookingController] Starting auto-booking service (interval: %d minutes)", c.intervalMinutes)

	c.running = true
	c.ticker = time.NewTicker(time.Duration(c.intervalMinutes) * time.Minute)

	go c.run()
}

// Stop 停止自动订阅服务
func (c *AutoBookingController) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		logger.Println("[AutoBookingController] Not running")
		return
	}

	logger.Println("[AutoBookingController] Stopping auto-booking service")

	c.running = false
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.stopChan)

	logger.Println("[AutoBookingController] Auto-booking service stopped")
}

// run 运行自动订阅循环
func (c *AutoBookingController) run() {
	// 立即执行一次
	c.executeBooking()

	// 定期执行
	for {
		select {
		case <-c.ticker.C:
			c.executeBooking()
		case <-c.stopChan:
			return
		}
	}
}

// executeBooking 执行一次自动订阅
func (c *AutoBookingController) executeBooking() {
	logger.Println("[AutoBookingController] Executing auto-booking...")

	bookable, success, err := c.service.ScanAndBookLiveMatches()
	if err != nil {
		logger.Errorf("[AutoBookingController] ❌ Failed: %v", err)
		return
	}

	logger.Printf("[AutoBookingController] ✅ Completed: %d bookable, %d success", bookable, success)
}

// Enable 启用自动订阅
func (c *AutoBookingController) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.enabled && c.running {
		logger.Println("[AutoBookingController] Already enabled and running")
		return
	}

	c.enabled = true
	logger.Println("[AutoBookingController] Auto-booking enabled")

	// 如果之前没有运行,现在启动
	if !c.running {
		c.mu.Unlock() // 解锁以避免死锁
		c.Start()
		c.mu.Lock()
	}
}

// Disable 禁用自动订阅
func (c *AutoBookingController) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		logger.Println("[AutoBookingController] Already disabled")
		return
	}

	c.enabled = false
	logger.Println("[AutoBookingController] Auto-booking disabled")

	// 停止运行中的服务
	if c.running {
		c.mu.Unlock() // 解锁以避免死锁
		c.Stop()
		c.mu.Lock()
	}
}

// IsEnabled 检查是否启用
func (c *AutoBookingController) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// IsRunning 检查是否运行中
func (c *AutoBookingController) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetStatus 获取状态信息
func (c *AutoBookingController) GetStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"enabled":          c.enabled,
		"running":          c.running,
		"interval_minutes": c.intervalMinutes,
	}
}

// SetInterval 设置执行间隔(分钟)
func (c *AutoBookingController) SetInterval(minutes int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if minutes < 1 {
		minutes = 1
	}

	c.intervalMinutes = minutes
	logger.Printf("[AutoBookingController] Interval updated to %d minutes", minutes)

	// 如果正在运行,重启以应用新的间隔
	if c.running {
		c.mu.Unlock() // 解锁以避免死锁
		c.Stop()
		c.Start()
		c.mu.Lock()
	}
}

