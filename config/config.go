package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// Betradar配置
	AccessToken   string
	MessagingHost string
	APIBaseURL    string
	RoutingKeys   []string

	// 数据库配置
	DatabaseURL string

	// 服务器配置
	Port string

	// 其他配置
	Environment string
	
	// 恢复配置
	AutoRecovery        bool   // 启动时自动触发恢复
	RecoveryAfterHours  int    // 恢复多少小时内的数据（0=默认72小时）
	RecoveryProducts    []string // 需要恢复的产品列表
}

func Load() *Config {
	return &Config{
		// Betradar配置
		AccessToken:   getEnv("BETRADAR_ACCESS_TOKEN", ""),
		MessagingHost: getEnv("BETRADAR_MESSAGING_HOST", "stgmq.betradar.com:5671"),
		APIBaseURL:    getEnv("BETRADAR_API_BASE_URL", "https://stgapi.betradar.com/v1"),
		RoutingKeys:   getRoutingKeys(),

		// 数据库配置
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/uof?sslmode=disable"),

		// 服务器配置
		Port: getEnv("PORT", "8080"),

		// 其他配置
		Environment: getEnv("ENVIRONMENT", "development"),
		
		// 恢复配置
		AutoRecovery:       getEnv("AUTO_RECOVERY", "true") == "true",
		RecoveryAfterHours: getEnvInt("RECOVERY_AFTER_HOURS", 10),  // Betradar最多允许10小时
		RecoveryProducts:   getRecoveryProducts(),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getRoutingKeys() []string {
	keys := getEnv("ROUTING_KEYS", "#")
	return strings.Split(keys, ",")
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	fmt.Sscanf(value, "%d", &result)
	if result == 0 {
		return defaultValue
	}
	return result
}

func getRecoveryProducts() []string {
	products := getEnv("RECOVERY_PRODUCTS", "liveodds,pre")
	return strings.Split(products, ",")
}

