package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ReplayClient 重放服务器API客户端
type ReplayClient struct {
	baseURL     string
	accessToken string
	client      *http.Client
}

// ReplayStatus 重放状态
type ReplayStatus struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

// NewReplayClient 创建重放客户端
// accessToken: Betradar access token (from BETRADAR_ACCESS_TOKEN)
// apiBaseURL: API base URL (from BETRADAR_API_BASE_URL)
func NewReplayClient(accessToken, apiBaseURL string) *ReplayClient {
	log.Println("[ReplayClient] Initializing Replay API client...")
	
	if accessToken != "" {
		log.Printf("[ReplayClient] ✅ Access token configured (length: %d)", len(accessToken))
	} else {
		log.Println("[ReplayClient] ⚠️  Access token is empty")
	}
	
	// 默认使用 global API
	if apiBaseURL == "" {
		apiBaseURL = "https://global.api.betradar.com/v1"
	}
	log.Printf("[ReplayClient] Using API: %s", apiBaseURL)
	
	return &ReplayClient{
		baseURL:     apiBaseURL,
		accessToken: accessToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest 执行HTTP请求
func (r *ReplayClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := r.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 添加access token到HTTP header
	if r.accessToken != "" {
		req.Header.Set("x-access-token", r.accessToken)
		log.Printf("[ReplayClient] Making %s request to %s with x-access-token header", method, path)
	} else {
		log.Printf("[ReplayClient] ⚠️  Making %s request to %s WITHOUT token", method, path)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ListEvents 列出重放列表中的赛事
func (r *ReplayClient) ListEvents() (string, error) {
	log.Println("📋 Listing replay events...")
	respBody, err := r.doRequest("GET", "/replay/", nil)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

// AddEvent 添加赛事到重放列表
func (r *ReplayClient) AddEvent(eventID string, startTime int) error {
	path := fmt.Sprintf("/replay/events/%s", eventID)
	if startTime > 0 {
		path += fmt.Sprintf("?start_time=%d", startTime)
	}
	
	log.Printf("➕ Adding event %s to replay list (start_time: %d min)...", eventID, startTime)
	_, err := r.doRequest("PUT", path, nil)
	if err != nil {
		return fmt.Errorf("add event: %w", err)
	}
	
	log.Printf("✅ Event %s added successfully", eventID)
	return nil
}

// RemoveEvent 从重放列表中删除赛事
func (r *ReplayClient) RemoveEvent(eventID string) error {
	path := fmt.Sprintf("/replay/events/%s", eventID)
	log.Printf("➖ Removing event %s from replay list...", eventID)
	_, err := r.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("remove event: %w", err)
	}
	
	log.Printf("✅ Event %s removed successfully", eventID)
	return nil
}

// PlayOptions 重放选项
type PlayOptions struct {
	Speed              int  `json:"speed,omitempty"`               // 重放速度(默认10)
	MaxDelay           int  `json:"max_delay,omitempty"`           // 最大延迟(毫秒,默认10000)
	NodeID             int  `json:"node_id,omitempty"`             // 节点ID
	ProductID          int  `json:"product_id,omitempty"`          // 产品ID
	UseReplayTimestamp bool `json:"use_replay_timestamp,omitempty"` // 使用重放时间戳
}

// Play 开始重放
func (r *ReplayClient) Play(options PlayOptions) error {
	// 设置默认值
	if options.Speed == 0 {
		options.Speed = 10
	}
	if options.MaxDelay == 0 {
		options.MaxDelay = 10000
	}
	
	// 构建查询参数
	path := fmt.Sprintf("/replay/play?speed=%d&max_delay=%d", options.Speed, options.MaxDelay)
	if options.NodeID > 0 {
		path += fmt.Sprintf("&node_id=%d", options.NodeID)
	}
	if options.ProductID > 0 {
		path += fmt.Sprintf("&product=%d", options.ProductID)
	}
	if options.UseReplayTimestamp {
		path += "&use_replay_timestamp=true"
	}
	
	log.Printf("▶️  Starting replay (speed: %dx, max_delay: %dms, node_id: %d)...", 
		options.Speed, options.MaxDelay, options.NodeID)
	
	_, err := r.doRequest("POST", path, nil)
	if err != nil {
		return fmt.Errorf("start replay: %w", err)
	}
	
	log.Println("✅ Replay started successfully")
	return nil
}

// Stop 停止重放
func (r *ReplayClient) Stop() error {
	log.Println("⏸️  Stopping replay...")
	_, err := r.doRequest("POST", "/replay/stop", nil)
	if err != nil {
		return fmt.Errorf("stop replay: %w", err)
	}
	
	log.Println("✅ Replay stopped successfully")
	return nil
}

// Reset 重置(停止并清空列表)
func (r *ReplayClient) Reset() error {
	log.Println("🔄 Resetting replay (stop and clear playlist)...")
	_, err := r.doRequest("POST", "/replay/reset", nil)
	if err != nil {
		return fmt.Errorf("reset replay: %w", err)
	}
	
	log.Println("✅ Replay reset successfully")
	return nil
}

// GetStatus 获取重放状态
func (r *ReplayClient) GetStatus() (string, error) {
	log.Println("📊 Getting replay status...")
	respBody, err := r.doRequest("GET", "/replay/status", nil)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

// WaitUntilReady 等待重放准备就绪
func (r *ReplayClient) WaitUntilReady(maxWait time.Duration) error {
	log.Println("⏳ Waiting for replay to be ready...")
	
	start := time.Now()
	for {
		if time.Since(start) > maxWait {
			return fmt.Errorf("timeout waiting for replay to be ready")
		}
		
		status, err := r.GetStatus()
		if err != nil {
			return err
		}
		
		// 检查是否还在设置中
		if !bytes.Contains([]byte(status), []byte("SETTING_UP")) {
			log.Println("✅ Replay is ready")
			return nil
		}
		
		log.Println("   Still setting up, waiting...")
		time.Sleep(2 * time.Second)
	}
}

// QuickReplay 快速重放单个赛事(便捷方法)
func (r *ReplayClient) QuickReplay(eventID string, speed int, nodeID int) error {
	// 1. 重置以清空之前的列表
	if err := r.Reset(); err != nil {
		log.Printf("⚠️  Reset failed (may be already empty): %v", err)
	}
	
	// 2. 添加赛事
	if err := r.AddEvent(eventID, 0); err != nil {
		return fmt.Errorf("add event: %w", err)
	}
	
	// 3. 等待一下让事件被添加到列表
	log.Println("⏳ Waiting for event to be added to playlist...")
	time.Sleep(3 * time.Second)
	
	// 4. 验证事件已在列表中 (最多重试5次)
	var eventsXML string
	var err error
	for i := 0; i < 5; i++ {
		eventsXML, err = r.ListEvents()
		if err != nil {
			return fmt.Errorf("verify playlist: %w", err)
		}
		
		// 检查是否为空
		if bytes.Contains([]byte(eventsXML), []byte("size=\"0\"")) {
			if i < 4 {
				log.Printf("⚠️  Playlist still empty, waiting 2 more seconds... (attempt %d/5)", i+1)
				time.Sleep(2 * time.Second)
				continue
			}
		}
		
		// 检查XML中是否包含我们的事件
		if bytes.Contains([]byte(eventsXML), []byte(eventID)) {
			log.Printf("✅ Event %s confirmed in playlist (attempt %d)", eventID, i+1)
			break
		}
		
		if i < 4 {
			log.Printf("⚠️  Event not in playlist yet, waiting 2 more seconds... (attempt %d/5)", i+1)
			time.Sleep(2 * time.Second)
		}
	}
	
	// 最终验证
	if !bytes.Contains([]byte(eventsXML), []byte(eventID)) {
		log.Printf("❌ Playlist verification failed after 5 attempts!")
		log.Printf("   Expected event: %s", eventID)
		log.Printf("   Playlist response: %s", eventsXML)
		log.Printf("   ")
		log.Printf("ℹ️  Possible reasons:")
		log.Printf("   1. Event is less than 48 hours old (only older events are replayable)")
		log.Printf("   2. Event ID is invalid or not available in Replay server")
		log.Printf("   3. Event was added but immediately removed by Betradar")
		return fmt.Errorf("event %s not found in playlist - may be too recent (need >48h old) or invalid", eventID)
	}
	
	// 5. 开始重放
	options := PlayOptions{
		Speed:              speed,
		MaxDelay:           10000,
		NodeID:             nodeID,
		UseReplayTimestamp: true,
	}
	
	if err := r.Play(options); err != nil {
		return fmt.Errorf("start replay: %w", err)
	}
	
	// 6. 等待准备就绪
	if err := r.WaitUntilReady(30 * time.Second); err != nil {
		return fmt.Errorf("wait for ready: %w", err)
	}
	
	log.Printf("🎉 Replay of %s is now running!", eventID)
	return nil
}

