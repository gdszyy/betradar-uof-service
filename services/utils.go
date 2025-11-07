package services

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractEventIDFromURN 从 event URN (sr:match:123) 中提取数字 ID (123)
func ExtractEventIDFromURN(urn string) (int64, error) {
	parts := strings.Split(urn, ":")
	// 检查 URN 格式是否为 sr:match:ID
	if len(parts) != 3 || parts[0] != "sr" || parts[1] != "match" {
		return 0, fmt.Errorf("invalid event URN format: %s", urn)
	}
	return strconv.ParseInt(parts[2], 10, 64)
}

// ExtractMarketIDFromURN 从 market URN (sr:market:123) 中提取数字 ID (123)
func ExtractMarketIDFromURN(urn string) (int64, error) {
	parts := strings.Split(urn, ":")
	// 检查 URN 格式是否为 sr:market:ID
	if len(parts) != 3 || parts[0] != "sr" || parts[1] != "market" {
		return 0, fmt.Errorf("invalid market URN format: %s", urn)
	}
	return strconv.ParseInt(parts[2], 10, 64)
}

