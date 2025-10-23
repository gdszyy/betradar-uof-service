package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"uof-service/config"
	"uof-service/thesports"
)

// TheSportsClient The Sports API 客户端
type TheSportsClient struct {
	config       *config.Config
	restClient   *thesports.Client
	mqttClient   *thesports.MQTTClient
	db           *sql.DB
	larkNotifier *LarkNotifier
	
	subscribedMatches map[string]bool
	mu                sync.RWMutex
	connected         bool
	connMu            sync.RWMutex
}

// NewTheSportsClient 创建 The Sports 客户端
func NewTheSportsClient(cfg *config.Config, db *sql.DB, notifier *LarkNotifier) *TheSportsClient {
	return &TheSportsClient{
		config:            cfg,
		db:                db,
		larkNotifier:      notifier,
		subscribedMatches: make(map[string]bool),
		connected:         false,
	}
}

// Connect 连接到 The Sports MQTT
func (c *TheSportsClient) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	
	if c.connected {
		return fmt.Errorf("already connected")
	}
	
	log.Println("[TheSports] 🔌 Connecting to The Sports MQTT...")
	
	// 创建 REST 客户端
	c.restClient = thesports.NewClient(c.config.TheSportsUsername, c.config.TheSportsSecret)
	
	// 创建 MQTT 客户端
	c.mqttClient = thesports.NewMQTTClient(
		c.config.TheSportsUsername,
		c.config.TheSportsSecret,
	)
	
	// 连接 MQTT
	if err := c.mqttClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}
	
	// 注册消息处理器
	c.mqttClient.OnMessage("*", c.handleMessage)
	
	// 订阅所有足球实时数据
	log.Println("[TheSports] 📡 Subscribing to football/live/#...")
	if err := c.mqttClient.SubscribeFootball("football/live/#"); err != nil {
		log.Printf("[TheSports] ❌ Failed to subscribe to football: %v", err)
		return fmt.Errorf("failed to subscribe to live football: %w", err)
	}
	log.Println("[TheSports] ✅ Successfully subscribed to football/live/#")
	
	// 订阅所有篮球实时数据
	log.Println("[TheSports] 📡 Subscribing to basketball/live/#...")
	if err := c.mqttClient.SubscribeBasketball("basketball/live/#"); err != nil {
		log.Printf("[TheSports] ❌ Failed to subscribe to basketball: %v", err)
		log.Println("[TheSports] ℹ️  Basketball MQTT may not be available, continuing...")
	} else {
		log.Println("[TheSports] ✅ Successfully subscribed to basketball/live/#")
	}
	
	// 订阅所有电竞实时数据 (实验性)
	log.Println("[TheSports] 📡 Subscribing to esports/live/# (experimental)...")
	if err := c.mqttClient.SubscribeEsports("esports/live/#"); err != nil {
		log.Printf("[TheSports] ❌ Failed to subscribe to esports: %v", err)
		log.Println("[TheSports] ℹ️  Esports MQTT may not be available, will use REST API only")
	} else {
		log.Println("[TheSports] ✅ Successfully subscribed to esports/live/#")
	}
	
	c.connected = true
	
	log.Println("[TheSports] ✅ Connected to The Sports MQTT successfully")
	
	// 发送飞书通知
	if c.larkNotifier != nil {
		c.larkNotifier.SendText("✅ **The Sports 连接成功**\n\n已连接到 The Sports MQTT 服务器\n订阅: \n- football/live/#\n- basketball/live/#\n- esports/live/#")
	}
	
	return nil
}

// Disconnect 断开连接
func (c *TheSportsClient) Disconnect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	
	if !c.connected {
		return nil
	}
	
	log.Println("[TheSports] 🔌 Disconnecting from The Sports MQTT...")
	
	if c.mqttClient != nil {
		if err := c.mqttClient.Disconnect(); err != nil {
			log.Printf("[TheSports] ⚠️  Disconnect error: %v", err)
		}
	}
	
	c.connected = false
	
	log.Println("[TheSports] ✅ Disconnected from The Sports MQTT")
	
	// 发送飞书通知
	if c.larkNotifier != nil {
		c.larkNotifier.SendText("🔌 **The Sports 已断开连接**")
	}
	
	return nil
}

// IsConnected 检查是否已连接
func (c *TheSportsClient) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.connected
}

// SubscribeMatch 订阅比赛
func (c *TheSportsClient) SubscribeMatch(matchID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.subscribedMatches[matchID] {
		return nil // 已订阅
	}
	
	topic := fmt.Sprintf("football/match/%s", matchID)
	if err := c.mqttClient.SubscribeFootball(topic); err != nil {
		return fmt.Errorf("failed to subscribe match %s: %w", matchID, err)
	}
	
	c.subscribedMatches[matchID] = true
	log.Printf("[TheSports] 📝 Subscribed to match: %s", matchID)
	
	return nil
}

// UnsubscribeMatch 取消订阅比赛
func (c *TheSportsClient) UnsubscribeMatch(matchID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.subscribedMatches[matchID] {
		return nil // 未订阅
	}
	
	topic := fmt.Sprintf("football/match/%s", matchID)
	if err := c.mqttClient.Unsubscribe(topic); err != nil {
		return fmt.Errorf("failed to unsubscribe match %s: %w", matchID, err)
	}
	
	delete(c.subscribedMatches, matchID)
	log.Printf("[TheSports] 🗑️  Unsubscribed from match: %s", matchID)
	
	return nil
}

// handleMessage 处理 MQTT 消息
func (c *TheSportsClient) handleMessage(topic string, payload []byte) {
	log.Printf("[TheSports] 📨 Received message on topic: %s (%d bytes)", topic, len(payload))
	
	// 记录消息类型统计
	if len(topic) > 0 {
		var sportType string
		if len(topic) >= 8 && topic[:8] == "football" {
			sportType = "football"
		} else if len(topic) >= 10 && topic[:10] == "basketball" {
			sportType = "basketball"
		} else if len(topic) >= 7 && topic[:7] == "esports" {
			sportType = "esports"
			log.Printf("[TheSports] 🎮 ESPORTS MESSAGE RECEIVED! Topic: %s", topic)
		} else {
			sportType = "unknown"
		}
		log.Printf("[TheSports] 🏆 Sport type: %s", sportType)
	}
	
	// 解析消息
	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("[TheSports] ❌ Failed to parse message: %v", err)
		return
	}
	
	// 提取消息类型
	msgType, ok := msg["type"].(string)
	if !ok {
		log.Printf("[TheSports] ⚠️  Message without type field")
		return
	}
	
	// 根据消息类型处理
	switch msgType {
	case "match_update":
		c.handleMatchUpdate(msg)
	case "incident":
		c.handleIncident(msg)
	case "statistics":
		c.handleStatistics(msg)
	default:
		log.Printf("[TheSports] ℹ️  Unknown message type: %s", msgType)
	}
}

// handleMatchUpdate 处理比赛更新
func (c *TheSportsClient) handleMatchUpdate(msg map[string]interface{}) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}
	
	matchID := fmt.Sprintf("%v", msg["match_id"])
	
	// 提取比赛信息
	homeScore := int(data["home_score"].(float64))
	awayScore := int(data["away_score"].(float64))
	status := data["status"].(string)
	minute := int(data["minute"].(float64))
	
	log.Printf("[TheSports] ⚽ Match Update: %s | Score: %d-%d | Status: %s | Minute: %d",
		matchID, homeScore, awayScore, status, minute)
	
	// 存储到数据库
	c.storeMatchUpdate(matchID, homeScore, awayScore, status, minute)
}

// handleIncident 处理比赛事件
func (c *TheSportsClient) handleIncident(msg map[string]interface{}) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}
	
	matchID := fmt.Sprintf("%v", msg["match_id"])
	incidentType := data["type"].(string)
	minute := int(data["time"].(float64))
	team := data["team"].(string)
	
	log.Printf("[TheSports] 🎯 Incident: %s | Type: %s | Team: %s | Minute: %d",
		matchID, incidentType, team, minute)
	
	// 存储到数据库
	c.storeIncident(matchID, incidentType, team, minute, data)
}

// handleStatistics 处理统计数据
func (c *TheSportsClient) handleStatistics(msg map[string]interface{}) {
	matchID := fmt.Sprintf("%v", msg["match_id"])
	log.Printf("[TheSports] 📊 Statistics update for match: %s", matchID)
	
	// 可以选择存储统计数据
}

// storeMatchUpdate 存储比赛更新到数据库
func (c *TheSportsClient) storeMatchUpdate(matchID string, homeScore, awayScore int, status string, minute int) {
	query := `
		INSERT INTO ld_matches (
			match_id, t1_score, t2_score, match_status, match_time, last_event_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (match_id) DO UPDATE SET
			t1_score = EXCLUDED.t1_score,
			t2_score = EXCLUDED.t2_score,
			match_status = EXCLUDED.match_status,
			match_time = EXCLUDED.match_time,
			last_event_at = EXCLUDED.last_event_at,
			updated_at = EXCLUDED.updated_at
	`
	
	matchTime := fmt.Sprintf("%d'", minute)
	now := time.Now()
	
	_, err := c.db.Exec(query, matchID, homeScore, awayScore, status, matchTime, now, now)
	if err != nil {
		log.Printf("[TheSports] ❌ Failed to store match update: %v", err)
	}
}

// storeIncident 存储事件到数据库
func (c *TheSportsClient) storeIncident(matchID, incidentType, team string, minute int, data map[string]interface{}) {
	query := `
		INSERT INTO ld_events (
			uuid, event_id, match_id, type_name, info, side, mtime, stime, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	uuid := fmt.Sprintf("ts_%s_%d_%d", matchID, minute, time.Now().UnixNano())
	eventID := fmt.Sprintf("ts_%d", time.Now().UnixNano())
	info := fmt.Sprintf("%s - %s", incidentType, team)
	matchTime := fmt.Sprintf("%d'", minute)
	stime := time.Now().Unix()
	
	_, err := c.db.Exec(query, uuid, eventID, matchID, incidentType, info, team, matchTime, stime, time.Now())
	if err != nil {
		log.Printf("[TheSports] ❌ Failed to store incident: %v", err)
	}
}

// GetTodayMatches 获取今日比赛
func (c *TheSportsClient) GetTodayMatches() ([]thesports.Match, error) {
	if c.restClient == nil {
		return nil, fmt.Errorf("REST client not initialized")
	}
	
	return c.restClient.GetTodayMatches()
}

// GetLiveMatches 获取直播比赛
func (c *TheSportsClient) GetLiveMatches() ([]thesports.Match, error) {
	if c.restClient == nil {
		return nil, fmt.Errorf("REST client not initialized")
	}
	
	return c.restClient.GetLiveMatches()
}

// GetBasketballTodayMatches 获取今日篮球比赛
func (c *TheSportsClient) GetBasketballTodayMatches() ([]thesports.BasketballMatch, error) {
	if c.restClient == nil {
		return nil, fmt.Errorf("REST client not initialized")
	}
	
	return c.restClient.GetBasketballTodayMatches()
}

// GetBasketballLiveMatches 获取直播篮球比赛
func (c *TheSportsClient) GetBasketballLiveMatches() ([]thesports.BasketballMatch, error) {
	if c.restClient == nil {
		return nil, fmt.Errorf("REST client not initialized")
	}
	
	return c.restClient.GetBasketballLiveMatches()
}

