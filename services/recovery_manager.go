package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"uof-service/config"
)

type RecoveryManager struct {
	config *config.Config
	client *http.Client
}

func NewRecoveryManager(cfg *config.Config) *RecoveryManager {
	return &RecoveryManager{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TriggerFullRecovery 触发全量恢复
func (r *RecoveryManager) TriggerFullRecovery() error {
	log.Println("Starting full recovery for all configured products...")
	
	var errors []error
	for _, product := range r.config.RecoveryProducts {
		if err := r.triggerProductRecovery(product); err != nil {
			log.Printf("Failed to trigger recovery for product %s: %v", product, err)
			errors = append(errors, err)
		} else {
			log.Printf("Successfully triggered recovery for product: %s", product)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("recovery failed for %d products", len(errors))
	}
	
	log.Println("Full recovery triggered successfully for all products")
	return nil
}

// triggerProductRecovery 触发单个产品的恢复
func (r *RecoveryManager) triggerProductRecovery(product string) error {
	// 构建恢复URL
	url := fmt.Sprintf("%s/%s/recovery/initiate_request", r.config.APIBaseURL, product)
	
	// 如果配置了恢复时间范围，添加after参数
	if r.config.RecoveryAfterHours > 0 {
		afterTimestamp := time.Now().Add(-time.Duration(r.config.RecoveryAfterHours) * time.Hour).UnixMilli()
		url = fmt.Sprintf("%s?after=%d", url, afterTimestamp)
		log.Printf("Recovery for %s: requesting data after %s (%d hours ago)", 
			product, 
			time.UnixMilli(afterTimestamp).Format(time.RFC3339),
			r.config.RecoveryAfterHours)
	} else {
		log.Printf("Recovery for %s: requesting default range (72 hours)", product)
	}
	
	// 创建POST请求
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// 添加认证头
	req.Header.Set("x-access-token", r.config.AccessToken)
	
	log.Printf("Sending recovery request to: %s", url)
	
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
		return fmt.Errorf("recovery request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	log.Printf("Recovery response for %s (status %d): %s", product, resp.StatusCode, string(body))
	
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
	
	log.Printf("Sending event recovery request to: %s", url)
	
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
	
	log.Printf("Event recovery response (status %d): %s", resp.StatusCode, string(body))
	
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
	
	log.Printf("Sending stateful messages recovery request to: %s", url)
	
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
	
	log.Printf("Stateful messages recovery response (status %d): %s", resp.StatusCode, string(body))
	
	return nil
}

