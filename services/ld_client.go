package services

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
	
	"uof-service/config"
)

// LDClient Live Data Socket 客户端
type LDClient struct {
	config       *config.Config
	conn         net.Conn
	connected    bool
	mu           sync.RWMutex
	done         chan struct{}
	reconnecting bool
	
	// 消息处理器
	eventHandler    func(*LDEvent)
	matchInfoHandler func(*LDMatchInfo)
	lineupHandler   func(*LDLineup)
	
	// 已订阅的比赛
	subscribedMatches map[string]bool
	matchesMu         sync.RWMutex
}

// NewLDClient 创建 LD 客户端
func NewLDClient(cfg *config.Config) *LDClient {
	return &LDClient{
		config:            cfg,
		done:              make(chan struct{}),
		subscribedMatches: make(map[string]bool),
	}
}

	// Connect 连接到 LD 服务器
	func (c *LDClient) Connect() error {
		// 使用生产环境服务器
		host := "livedata.betradar.com:2017"
		
		log.Printf("[LD] Connecting to Live Data server: %s...", host)
	
	// 创建 TLS 配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}
	
	// 建立 TLS 连接
	conn, err := tls.Dial("tcp", host, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()
	
	log.Println("[LD] ✅ Connected to Live Data server")
	
	// 发送登录消息
	if err := c.login(); err != nil {
		c.conn.Close()
		return fmt.Errorf("login failed: %w", err)
	}
	
	// 启动消息接收
	go c.receiveMessages()
	
	// 启动心跳
	go c.sendHeartbeat()
	
	return nil
}

	// login 发送登录消息
	func (c *LDClient) login() error {
		loginMsg := fmt.Sprintf(`<login><credential><loginname value="%s"/><password value="%s"/></credential></login>%c`,
			c.config.Username, c.config.Password, 0x00)
	
	log.Println("[LD] Sending login message...")
	
	if err := c.sendMessage(loginMsg); err != nil {
		return err
	}
	
	log.Println("[LD] ✅ Login message sent")
	return nil
}

// sendMessage 发送消息
func (c *LDClient) sendMessage(msg string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}
	
	// 添加结束符
	if !strings.HasSuffix(msg, "\x00") {
		msg += "\x00"
	}
	
	_, err := c.conn.Write([]byte(msg))
	return err
}

// receiveMessages 接收消息
func (c *LDClient) receiveMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[LD] ❌ Panic in receiveMessages: %v", r)
		}
		c.reconnect()
	}()
	
	buffer := make([]byte, 0, 65536)
	tempBuf := make([]byte, 4096)
	
	for {
		select {
		case <-c.done:
			return
		default:
		}
		
		// 设置读取超时
		c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		
		n, err := c.conn.Read(tempBuf)
		if err != nil {
			if err == io.EOF {
				log.Println("[LD] Connection closed by server")
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Println("[LD] ⚠️  Read timeout, connection may be dead")
			} else {
				log.Printf("[LD] ❌ Read error: %v", err)
			}
			c.reconnect()
			return
		}
		
		buffer = append(buffer, tempBuf[:n]...)
		
		// 处理所有完整的消息 (以 0x00 分隔)
		for {
			idx := -1
			for i, b := range buffer {
				if b == 0x00 {
					idx = i
					break
				}
			}
			
			if idx == -1 {
				break
			}
			
			// 提取消息
			msgBytes := buffer[:idx]
			buffer = buffer[idx+1:]
			
			// 处理消息
			if len(msgBytes) > 0 {
				c.handleMessage(msgBytes)
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *LDClient) handleMessage(data []byte) {
	msgStr := string(data)
	
	// 跳过空消息
	if strings.TrimSpace(msgStr) == "" {
		return
	}
	
	// 判断消息类型
	if strings.Contains(msgStr, "<alive") {
		// Alive 消息
		log.Println("[LD] ❤️  Received alive message")
		return
	}
	
	if strings.Contains(msgStr, "<login") {
		// 登录响应
		log.Println("[LD] ✅ Login successful")
		return
	}
	
	if strings.Contains(msgStr, "<bookmatch") {
		// 订阅响应
		log.Println("[LD] ✅ Match subscription confirmed")
		return
	}
	
	if strings.Contains(msgStr, "<event") {
		// 事件消息
		var event LDEvent
		if err := xml.Unmarshal(data, &event); err != nil {
			log.Printf("[LD] ❌ Failed to parse event: %v", err)
			return
		}
		
		log.Printf("[LD] 📊 Event: type=%d, info=%s, side=%s, mtime=%s",
			event.Type, event.Info, event.Side, event.MTime)
		
		if c.eventHandler != nil {
			c.eventHandler(&event)
		}
		return
	}
	
	if strings.Contains(msgStr, "<match") {
		// 比赛信息
		var matchInfo LDMatchInfo
		if err := xml.Unmarshal(data, &matchInfo); err != nil {
			log.Printf("[LD] ❌ Failed to parse match info: %v", err)
			return
		}
		
		log.Printf("[LD] 🏟️  Match info: %s vs %s, status=%s",
			matchInfo.T1Name, matchInfo.T2Name, matchInfo.MatchStatus)
		
		if c.matchInfoHandler != nil {
			c.matchInfoHandler(&matchInfo)
		}
		return
	}
	
	if strings.Contains(msgStr, "<lineup") {
		// 阵容信息
		var lineup LDLineup
		if err := xml.Unmarshal(data, &lineup); err != nil {
			log.Printf("[LD] ❌ Failed to parse lineup: %v", err)
			return
		}
		
		log.Printf("[LD] 👥 Lineup received for match %s", lineup.MatchID)
		
		if c.lineupHandler != nil {
			c.lineupHandler(&lineup)
		}
		return
	}
	
	// 其他消息类型
	if len(msgStr) < 200 {
		log.Printf("[LD] 📨 Received: %s", msgStr)
	} else {
		log.Printf("[LD] 📨 Received long message (%d bytes)", len(msgStr))
	}
}

// sendHeartbeat 发送心跳
func (c *LDClient) sendHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.RLock()
			connected := c.connected
			c.mu.RUnlock()
			
			if !connected {
				continue
			}
			
			aliveMsg := fmt.Sprintf(`<alive />%c`, 0x00)
			if err := c.sendMessage(aliveMsg); err != nil {
				log.Printf("[LD] ❌ Failed to send heartbeat: %v", err)
			} else {
				log.Println("[LD] 💓 Heartbeat sent")
			}
		}
	}
}

// reconnect 重新连接
func (c *LDClient) reconnect() {
	c.mu.Lock()
	if c.reconnecting {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
	
	log.Println("[LD] 🔄 Reconnecting in 5 seconds...")
	time.Sleep(5 * time.Second)
	
	c.mu.Lock()
	c.reconnecting = false
	c.mu.Unlock()
	
	if err := c.Connect(); err != nil {
		log.Printf("[LD] ❌ Reconnect failed: %v", err)
		go c.reconnect()
		return
	}
	
	// 重新订阅比赛
	c.resubscribeMatches()
}

// SubscribeMatch 订阅比赛
func (c *LDClient) SubscribeMatch(matchID string) error {
	msg := fmt.Sprintf(`<match matchid="%s" />%c`, matchID, 0x00)
	
	log.Printf("[LD] 📝 Subscribing to match: %s", matchID)
	
	if err := c.sendMessage(msg); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	
	c.matchesMu.Lock()
	c.subscribedMatches[matchID] = true
	c.matchesMu.Unlock()
	
	return nil
}

// UnsubscribeMatch 取消订阅比赛
func (c *LDClient) UnsubscribeMatch(matchID string) error {
	msg := fmt.Sprintf(`<unsubscribe matchid="%s" />%c`, matchID, 0x00)
	
	log.Printf("[LD] ❌ Unsubscribing from match: %s", matchID)
	
	if err := c.sendMessage(msg); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	
	c.matchesMu.Lock()
	delete(c.subscribedMatches, matchID)
	c.matchesMu.Unlock()
	
	return nil
}

// resubscribeMatches 重新订阅所有比赛
func (c *LDClient) resubscribeMatches() {
	c.matchesMu.RLock()
	matches := make([]string, 0, len(c.subscribedMatches))
	for matchID := range c.subscribedMatches {
		matches = append(matches, matchID)
	}
	c.matchesMu.RUnlock()
	
	log.Printf("[LD] 🔄 Resubscribing to %d matches...", len(matches))
	
	for _, matchID := range matches {
		if err := c.SubscribeMatch(matchID); err != nil {
			log.Printf("[LD] ❌ Failed to resubscribe to %s: %v", matchID, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// SetEventHandler 设置事件处理器
func (c *LDClient) SetEventHandler(handler func(*LDEvent)) {
	c.eventHandler = handler
}

// SetMatchInfoHandler 设置比赛信息处理器
func (c *LDClient) SetMatchInfoHandler(handler func(*LDMatchInfo)) {
	c.matchInfoHandler = handler
}

// SetLineupHandler 设置阵容处理器
func (c *LDClient) SetLineupHandler(handler func(*LDLineup)) {
	c.lineupHandler = handler
}

// Close 关闭连接
func (c *LDClient) Close() error {
	close(c.done)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn != nil {
		return c.conn.Close()
	}
	
	return nil
}

