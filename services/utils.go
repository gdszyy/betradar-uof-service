package services

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractEventIDFromURN 从 event URN (sr:match:123) 中提取数字 ID (123)
func ExtractEventIDFromURN(urn string) (int64, error) {
	parts := strings.Split(urn, ":")
	if len(parts) != 3 || parts[0] != "sr" || parts[1] != "match" {
		return 0, fmt.Errorf("invalid event URN format: %s", urn)
	}
	return strconv.ParseInt(parts[2], 10, 64)
}


// ExtractMarketIDFromURN 从 market ID 字符串中提取数字 ID
// 注意：SportRader UOF 中 market id 是纯整数字符串（如 "98", "777"），不是 URN 格式
// 此函数兼容两种格式以保证向后兼容性：
// 1. 标准格式（纯整数）: "98" -> 98
// 2. URN 格式（历史兼容）: "sr:market:98" -> 98
func ExtractMarketIDFromURN(marketIDStr string) (int64, error) {
	// 先尝试直接解析为整数（这是 SportRader UOF 的标准格式）
	if marketID, err := strconv.ParseInt(marketIDStr, 10, 64); err == nil {
		return marketID, nil
	}
	
	// 如果直接解析失败，尝试 URN 格式（向后兼容，以防有历史数据）
	parts := strings.Split(marketIDStr, ":")
	if len(parts) == 3 && parts[0] == "sr" && parts[1] == "market" {
		marketID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid market URN format: %s (ID part is not a valid integer)", marketIDStr)
		}
		return marketID, nil
	}
	
	// 两种格式都不匹配
	return 0, fmt.Errorf("invalid market ID format: %s (expected integer like '98' or URN like 'sr:market:98')", marketIDStr)
}

// CleanSQLQuery 清理 SQL 语句中的换行符和多余空格，使其成为单行
func CleanSQLQuery(query string) string {
	// 替换所有换行符和制表符为空格
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")

	// 替换多个连续空格为一个空格
	for strings.Contains(query, "  ") {
		query = strings.ReplaceAll(query, "  ", " ")
	}

	// 去除首尾空格
	return strings.TrimSpace(query)
}
