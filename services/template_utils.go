package services

import (
	"fmt"
	"strconv"
	"strings"
)

// toOrdinal 将数字字符串转换为序数词
// "1" -> "1st", "2" -> "2nd", "3" -> "3rd", "4" -> "4th"
func toOrdinal(num string) string {
	// 尝试解析为整数
	n, err := strconv.Atoi(num)
	if err != nil {
		return num // 如果不是数字，返回原值
	}

	// 处理特殊情况: 11, 12, 13 都是 "th"
	if n%100 >= 11 && n%100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}

	// 根据个位数确定后缀
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// formatWithSign 格式化数字并添加正负号
// value: 数字字符串，例如 "2.5" 或 "-1.5"
// negate: 是否取反
// 返回: "+2.5", "-2.5" 等带符号的字符串
func formatWithSign(value string, negate bool) string {
	// 尝试解析为浮点数
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value // 如果不是数字，返回原值
	}

	// 如果需要取反
	if negate {
		f = -f
	}

	// 格式化为带符号的字符串
	if f >= 0 {
		return fmt.Sprintf("+%g", f)
	}
	return fmt.Sprintf("%g", f)
}

// replaceSpecifiers 替换名称模板中的 specifiers 占位符
// 支持以下格式:
// - {X}: 直接替换为 specifier X 的值
// - {!X}: 替换为 specifier X 的序数形式
// - {+X}: 替换为 specifier X 的值并添加 +/- 符号
// - {-X}: 替换为 specifier X 的取反值并添加 +/- 符号
func replaceSpecifiers(name string, specifiers string) string {
	if specifiers == "" {
		return name
	}

	// 解析 specifiers 为 map
	specMap := parseSpecifiers(specifiers)

	// 按优先级替换占位符
	// 1. 先处理特殊前缀 (避免被基本替换覆盖)
	for key, value := range specMap {
		// 序数替换 {!X}
		if strings.Contains(name, "{!"+key+"}") {
			ordinalValue := toOrdinal(value)
			name = strings.ReplaceAll(name, "{!"+key+"}", ordinalValue)
		}

		// 正号替换 {+X}
		if strings.Contains(name, "{+"+key+"}") {
			signedValue := formatWithSign(value, false)
			name = strings.ReplaceAll(name, "{+"+key+"}", signedValue)
		}

		// 负号替换 {-X}
		if strings.Contains(name, "{-"+key+"}") {
			signedValue := formatWithSign(value, true)
			name = strings.ReplaceAll(name, "{-"+key+"}", signedValue)
		}

		// 基本替换 {X}
		name = strings.ReplaceAll(name, "{"+key+"}", value)
	}

	return name
}

// parseSpecifiers 解析 specifiers 字符串为 map
// "pointnr=3|hcp=-1.5" -> {"pointnr": "3", "hcp": "-1.5"}
func parseSpecifiers(specifiers string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(specifiers, "|")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// replaceCompetitors 替换名称模板中的竞争者占位符
// - {$competitor1}: 替换为主队名称
// - {$competitor2}: 替换为客队名称
func replaceCompetitors(name string, homeTeam string, awayTeam string) string {
	name = strings.ReplaceAll(name, "{$competitor1}", homeTeam)
	name = strings.ReplaceAll(name, "{$competitor2}", awayTeam)
	return name
}
