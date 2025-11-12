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
		VirtualHost   string // 新增: AMQP Virtual Host
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
	AutoBookingEnabled         bool // 是否启用自动订阅
	AutoBookingIntervalMinutes int  // 自动订阅间隔(分钟)
	
	// 数据清理配置
	CleanupRetainDaysMessages    int // uof_messages 保留天数
	CleanupRetainDaysOdds        int // odds_changes, markets, odds 保留天数
	CleanupRetainDaysBets        int // bet_stops, bet_settlements 保留天数
	CleanupRetainDaysLiveData    int // ld_events, ld_lineups 保留天数
	CleanupRetainDaysEvents      int // tracked_events, ld_matches 保留天数
	
	// Producer 监控配置
	ProducerCheckIntervalSeconds int // 检查间隔（秒）
	ProducerDownThresholdSeconds int // 下线阈值（秒）
	
	// 订阅同步配置
	SubscriptionSyncIntervalMinutes int // 订阅同步间隔(分钟)
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
		APIBaseURL:    getEnv("BETRADAR_API_BASE_URL", "https://stgapi.betradar.com"),
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
		AutoBookingEnabled:         getEnv("AUTO_BOOKING_ENABLED", "false") == "true", // 默认关闭
		AutoBookingIntervalMinutes: getEnvInt("AUTO_BOOKING_INTERVAL_MINUTES", 30),
		
		// 数据清理配置（所有数据默认保留 2 天）
		CleanupRetainDaysMessages:  getEnvInt("CLEANUP_RETAIN_DAYS_MESSAGES", 2),   // 原始消息默认保留 2 天
		CleanupRetainDaysOdds:      getEnvInt("CLEANUP_RETAIN_DAYS_ODDS", 2),       // 赔率数据默认保留 2 天
		CleanupRetainDaysBets:      getEnvInt("CLEANUP_RETAIN_DAYS_BETS", 2),       // 投注记录默认保留 2 天
		CleanupRetainDaysLiveData:  getEnvInt("CLEANUP_RETAIN_DAYS_LIVEDATA", 2),   // Live Data 默认保留 2 天
		CleanupRetainDaysEvents:    getEnvInt("CLEANUP_RETAIN_DAYS_EVENTS", 2),    // 赛事信息默认保留 2 天
		
		// Producer 监控配置
		ProducerCheckIntervalSeconds: getEnvInt("PRODUCER_CHECK_INTERVAL_SECONDS", 60),   // 默认每 60 秒检查一次
		ProducerDownThresholdSeconds: getEnvInt("PRODUCER_DOWN_THRESHOLD_SECONDS", 10),  // 默认 10 秒不响应才告警 (alive 消息间隔)
		
		// 订阅同步配置
		SubscriptionSyncIntervalMinutes: getEnvInt("SUBSCRIPTION_SYNC_INTERVAL_MINUTES", 5), // 默认每 5 分钟同步一次
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
	keys := getEnv("ROUTING_KEYS", "")
	if keys != "" {
		return strings.Split(keys, ",")
	}
	
	// 默认订阅 live 和 pre-match 的所有消息
	return []string{
		"liveodds.-.odds_change.#",
		"liveodds.-.bet_stop.#",
		"liveodds.-.bet_settlement.#",
		"liveodds.-.bet_cancel.#",
		"liveodds.-.fixture_change.#",
		"pre.-.odds_change.#",        // Pre-match odds changes
		"pre.-.bet_stop.#",            // Pre-match bet stops
		"pre.-.bet_settlement.#",      // Pre-match bet settlements
		"pre.-.bet_cancel.#",          // Pre-match bet cancels
		"pre.-.fixture_change.#",      // Pre-match fixture changes
	}
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

