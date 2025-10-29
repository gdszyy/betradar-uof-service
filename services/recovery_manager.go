package services

import (
"uof-service/logger"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"uof-service/config"
)

type RecoveryManager struct {
	config           *config.Config
	client           *http.Client
	messageStore     *MessageStore // 用于保存恢复状态
	nodeID           int // 用于区分会话的节点ID
	requestIDCounter int // 用于生成唯一的request_id
}

func NewRecoveryManager(cfg *config.Config, store *MessageStore) *RecoveryManager {
	return &RecoveryManager{
		config:           cfg,
		client:           &http.Client{
			Timeout: 30 * time.Second,
		},
		messageStore:     store,
		nodeID:           1, // 默认节点ID为1，可以通过环境变量配置
		requestIDCounter: int(time.Now().Unix()), // 使用当前时间戳作为起始ID
	}
}

// TriggerFullRecovery 触发全量恢复
func (r *RecoveryManager) TriggerFullRecovery() error {
	logger.Println("Starting full recovery for all configured products...")
	
	var errors []error
	var rateLimitErrors int
	
	for _, product := range r.config.RecoveryProducts {
		if err := r.triggerProductRecovery(product); err != nil {
			if bytes.Contains([]byte(err.Error()), []byte("rate limit exceeded")) {
				logger.Printf("⚠️  Recovery for product %s: rate limited, retry scheduled", product)
				rateLimitErrors++
			} else {
				logger.Printf("❌ Failed to trigger recovery for product %s: %v", product, err)
				errors = append(errors, err)
			}
		} else {
			logger.Printf("✅ Successfully triggered recovery for product: %s", product)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("recovery failed for %d products", len(errors))
	}
	
	if rateLimitErrors > 0 {
		logger.Printf("ℹ️  %d product(s) rate limited, retries scheduled in background", rateLimitErrors)
	}
	
	logger.Println("Full recovery triggered successfully for all products")
	return nil
}

// triggerProductRecovery 触发单个产品的恢复
func (r *RecoveryManager) triggerProductRecovery(product string) error {
	// 生成唯一的request_id
	r.requestIDCounter++
	requestID := r.requestIDCounter
	
	// 构建恢复URL
	url := fmt.Sprintf("%s/%s/recovery/initiate_request", r.config.APIBaseURL, product)
	
	// 注意：liveodds对after参数很敏感，建议不使用after参数，让Betradar使用默认范围
	// 如果配置了RECOVERY_AFTER_HOURS且大于0，且产品不是liveodds，才使用after参数
	if r.config.RecoveryAfterHours > 0 && product != "liveodds" {
		// Betradar限制：最多恢复10小时内的数据（Live Odds producers） 
		// 调用频率限制 https://docs.sportradar.com/uof/api-and-structure/api/odds-recovery/restrictions-for-odds-recovery
		hours := r.config.RecoveryAfterHours
		if hours > 10 {
			logger.Printf("WARNING: RECOVERY_AFTER_HOURS=%d exceeds Betradar limit (10 hours), using 10 hours instead", hours)
			hours = 10
		}
		afterTimestamp := time.Now().Add(-time.Duration(hours) * time.Hour).UnixMilli()
		url = fmt.Sprintf("%s?after=%d&request_id=%d&node_id=%d", url, afterTimestamp, requestID, r.nodeID)
		logger.Printf("Recovery for %s: requesting data after %s (%d hours ago) [request_id=%d, node_id=%d]", 
			product, 
			time.UnixMilli(afterTimestamp).Format(time.RFC3339),
			hours,
			requestID,
			r.nodeID)
	} else {
		// 即使不使用after参数，也添加request_id和node_id用于追踪
		url = fmt.Sprintf("%s?request_id=%d&node_id=%d", url, requestID, r.nodeID)
		if product == "liveodds" {
			logger.Printf("Recovery for %s: using default range (no 'after' parameter) [request_id=%d, node_id=%d]", product, requestID, r.nodeID)
		} else {
			logger.Printf("Recovery for %s: using default range (Betradar default) [request_id=%d, node_id=%d]", product, requestID, r.nodeID)
		}
	}
	
	// 创建POST请求
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// 添加认证头
	req.Header.Set("x-access-token", r.config.AccessToken)
	
	logger.Printf("Sending recovery request to: %s", url)
	
	// 发送请求
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		// 检查是否是频率限制错误
		if resp.StatusCode == http.StatusForbidden && bytes.Contains(body, []byte("Too many requests")) {
			logger.Printf("⚠️  Recovery rate limit exceeded for product %s", product)
			logger.Printf("   Will schedule retry in background...")
			
			// 异步重试，不阻塞启动
			go r.scheduleRecoveryRetry(product, requestID, 15*time.Minute)
			
			// 返回特殊错误，让调用者知道已计划重试
			return fmt.Errorf("rate limit exceeded, retry scheduled")
		}
		return fmt.Errorf("recovery request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	logger.Printf("Recovery response for %s (status %d): %s", product, resp.StatusCode, string(body))
	
	// 保存恢复初始化状态
	if r.messageStore != nil {
		// 获取product ID
		productID := 1 // liveodds
		if product == "pre" {
			productID = 3
		}
		
		if err := r.messageStore.SaveRecoveryInitiated(requestID, productID, r.nodeID); err != nil {
			logger.Printf("Warning: Failed to save recovery status: %v", err)
		}
	}
	
	return nil
}

// TriggerEventRecovery 触发单个赛事的恢复
func (r *RecoveryManager) TriggerEventRecovery(product, eventID string) error {
	url := fmt.Sprintf("%s/%s/odds/events/%s/initiate_request", 
		r.config.APIBaseURL, product, eventID)
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", r.config.AccessToken)
	
	logger.Printf("Sending event recovery request to: %s", url)
	
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("event recovery failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	logger.Printf("Event recovery response (status %d): %s", resp.StatusCode, string(body))
	
	return nil
}

// TriggerStatefulMessagesRecovery 触发状态消息恢复（bet_settlement, bet_cancel等）
func (r *RecoveryManager) TriggerStatefulMessagesRecovery(product, eventID string) error {
	url := fmt.Sprintf("%s/%s/stateful_messages/events/%s/initiate_request", 
		r.config.APIBaseURL, product, eventID)
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", r.config.AccessToken)
	
	logger.Printf("Sending stateful messages recovery request to: %s", url)
	
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("stateful messages recovery failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	logger.Printf("Stateful messages recovery response (status %d): %s", resp.StatusCode, string(body))
	
	return nil
}



// scheduleRecoveryRetry 计划在指定延迟后重试恢复
func (r *RecoveryManager) scheduleRecoveryRetry(product string, requestID int, delay time.Duration) {
	logger.Printf("📅 Scheduling recovery retry for product %s in %v", product, delay)
	
	time.Sleep(delay)
	
	logger.Printf("🔄 Retrying recovery for product %s (after rate limit delay)", product)
	
	if err := r.triggerProductRecovery(product); err != nil {
		// 如果再次失败，检查是否还是频率限制
		if bytes.Contains([]byte(err.Error()), []byte("rate limit exceeded")) {
			// 如果还是频率限制，再等更长时间重试
			logger.Printf("⚠️  Recovery retry still rate limited, will try again in 30 minutes")
			go r.scheduleRecoveryRetry(product, requestID, 30*time.Minute)
		} else {
			logger.Printf("❌ Recovery retry failed for product %s: %v", product, err)
		}
	} else {
		logger.Printf("✅ Recovery retry successful for product %s", product)
	}
}

