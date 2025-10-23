package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	// Betradar配置
	AccessToken   string
	UOFAPIToken   string // UOF API Token (for REST API calls)
	Username      string
	Password      string
	BookmakerID   string
	Products      []string
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
	
	// 通知配置
	LarkWebhook string // 飞书机器人Webhook URL
	
	// The Sports 配置
	TheSportsUsername  string // The Sports Username (for both HTTP API and MQTT)
	TheSportsSecret    string // The Sports Secret (for both HTTP API and MQTT)
	
	// 自动订阅配置
	AutoBookingIntervalMinutes int // 自动订阅间隔(分钟)
}

func Load() *Config {
	log.Println("[Config] Loading configuration from environment variables...")
	
	username := getEnv("UOF_USERNAME", "")
	password := getEnv("UOF_PASSWORD", "")
	
	// 记录凭证状态（隐藏密码）
	if username != "" {
		log.Printf("[Config] ✅ UOF_USERNAME loaded: %s", username)
	} else {
		log.Println("[Config] ⚠️  UOF_USERNAME not set")
	}
	
	if password != "" {
		log.Printf("[Config] ✅ UOF_PASSWORD loaded: %s (length: %d)", maskPassword(password), len(password))
	} else {
		log.Println("[Config] ⚠️  UOF_PASSWORD not set")
	}
	
	// 检查 LARK_WEBHOOK_URL
	larkWebhook := getEnv("LARK_WEBHOOK_URL", "")
	if larkWebhook != "" {
		log.Printf("[Config] ✅ LARK_WEBHOOK_URL loaded: %s (length: %d)", larkWebhook, len(larkWebhook))
	} else {
		log.Println("[Config] ⚠️  LARK_WEBHOOK_URL not set")
	}
	
		return &Config{
		// Betradar配置
		AccessToken:   getEnv("BETRADAR_ACCESS_TOKEN", ""),
		UOFAPIToken:   getEnv("UOF_API_TOKEN", getEnv("BETRADAR_ACCESS_TOKEN", "")), // 默认使用 AccessToken
		Username:      username,
		Password:      password,
		BookmakerID:   getEnv("BOOKMAKER_ID", username),
		Products:      getProducts(),
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
		
			// 通知配置
			LarkWebhook: larkWebhook,
			
			// The Sports 配置
			TheSportsUsername: getEnv("THESPORTS_USERNAME", ""),
			TheSportsSecret:   getEnv("THESPORTS_SECRET", ""),
			
			// 自动订阅配置
			AutoBookingIntervalMinutes: getEnvInt("AUTO_BOOKING_INTERVAL_MINUTES", 30),
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

func getProducts() []string {
	products := getEnv("PRODUCTS", "liveodds,pre")
	return strings.Split(products, ",")
}



// maskPassword 隐藏密码，只显示前2个和后2个字符
func maskPassword(password string) string {
	if password == "" {
		return ""
	}
	length := len(password)
	if length <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[length-2:]
}

