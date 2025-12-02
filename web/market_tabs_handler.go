package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// MarketTabsConfig 盘口分组配置结构
type MarketTabsConfig struct {
	Tabs  []TabConfig           `json:"tabs"`
	Chips map[string]ChipConfig `json:"chips"`
}

// TabConfig Tab配置结构
type TabConfig struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Type            string   `json:"type"`
	Groups          []string `json:"groups"`
	ChipSpecifiers  []string `json:"chipSpecifiers"`
}

// ChipConfig Chip配置结构
type ChipConfig struct {
	Label     string `json:"label"`
	Specifier string `json:"specifier"`
	Value     string `json:"value"`
}

// MarketTabsHandler 盘口分组配置处理器
type MarketTabsHandler struct {
	configPath string
}

// NewMarketTabsHandler 创建盘口分组配置处理器
func NewMarketTabsHandler() *MarketTabsHandler {
	// 获取配置文件路径
	configPath := filepath.Join("config", "market_tabs_config.json")
	return &MarketTabsHandler{
		configPath: configPath,
	}
}

// HandleGetMarketTabs 处理获取盘口分组配置请求
func (h *MarketTabsHandler) HandleGetMarketTabs(w http.ResponseWriter, r *http.Request) {
	// 读取配置文件
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load market tabs configuration", http.StatusInternalServerError)
		return
	}

	// 解析配置
	var config MarketTabsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		http.Error(w, "Failed to parse market tabs configuration", http.StatusInternalServerError)
		return
	}

	// 返回配置
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
