package config

import (
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

