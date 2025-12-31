package services

import (
	"fmt"
	"regexp"
	"strings"
)

// OutcomeNameResolver 负责解析 outcome 名称模板并替换占位符
type OutcomeNameResolver struct {
	marketDescService *MarketDescriptionsService
}

// NewOutcomeNameResolver 创建 OutcomeNameResolver
func NewOutcomeNameResolver(marketDescService *MarketDescriptionsService) *OutcomeNameResolver {
	return &OutcomeNameResolver{
		marketDescService: marketDescService,
	}
}

// ResolveOutcomeName 解析 outcome 名称
// 从模板 "over {total}" + specifiers "total=2.5" -> "over 2.5"
func (r *OutcomeNameResolver) ResolveOutcomeName(
	marketID string,
	outcomeID string,
	specifiers string,
) (string, error) {
	// 1. 从 market_descriptions 服务获取模板
	template, err := r.marketDescService.GetOutcomeNameTemplate(marketID, outcomeID)
	if err != nil {
		return "", fmt.Errorf("failed to get outcome template: %w", err)
	}

	// 2. 如果没有 specifiers,直接返回模板
	if specifiers == "" {
		return template, nil
	}

	// 3. 解析 specifiers 字符串 "total=2.5|hcp=1.5" -> map
	specMap := parseSpecifiers(specifiers)

	// 4. 替换模板中的占位符
	resolved := template
	for key, value := range specMap {
		placeholder := "{" + key + "}"
		resolved = strings.ReplaceAll(resolved, placeholder, value)
	}

	// 5. 处理特殊占位符 {$competitor1} {$competitor2} {!periodnr} 等
	// 这些占位符需要额外的上下文信息,暂时保持原样
	// 在实际使用时,调用方需要提供这些信息

	return resolved, nil
}

// ResolveMarketName 解析 market 名称
// 从模板 "{!periodnr} period - {$competitor1} total" + specifiers "periodnr=1" -> "1st period - Home total"
func (r *OutcomeNameResolver) ResolveMarketName(
	marketID string,
	specifiers string,
	competitor1Name string,
	competitor2Name string,
) (string, error) {
	// 1. 从 market_descriptions 服务获取模板
	template, err := r.marketDescService.GetMarketNameTemplate(marketID)
	if err != nil {
		return "", fmt.Errorf("failed to get market template: %w", err)
	}

	// 2. 如果没有 specifiers,直接返回模板
	if specifiers == "" {
		return template, nil
	}

	// 3. 解析 specifiers
	specMap := parseSpecifiers(specifiers)

	// 4. 替换普通占位符
	resolved := template
	for key, value := range specMap {
		placeholder := "{" + key + "}"
		resolved = strings.ReplaceAll(resolved, placeholder, value)
	}

	// 5. 替换特殊占位符
	resolved = strings.ReplaceAll(resolved, "{$competitor1}", competitor1Name)
	resolved = strings.ReplaceAll(resolved, "{$competitor2}", competitor2Name)

	// 6. 处理序数词 {!periodnr} -> "1st", "2nd", "3rd"
	resolved = replaceOrdinalPlaceholders(resolved, specMap)

	return resolved, nil
}



// replaceOrdinalPlaceholders 替换序数词占位符
// {!periodnr} with periodnr=1 -> "1st"
// {!periodnr} with periodnr=2 -> "2nd"
func replaceOrdinalPlaceholders(template string, specMap map[string]string) string {
	// 匹配 {!xxx} 格式的占位符
	re := regexp.MustCompile(`\{!(\w+)\}`)
	
	result := re.ReplaceAllStringFunc(template, func(match string) string {
		// 提取变量名
		varName := strings.Trim(match, "{!}")
		
		// 从 specMap 获取值
		value, exists := specMap[varName]
		if !exists {
			return match // 保持原样
		}

		// 转换为序数词
		return toOrdinal(value)
	})

	return result
}



// BatchResolveOutcomeNames 批量解析 outcome 名称
func (r *OutcomeNameResolver) BatchResolveOutcomeNames(
	marketID string,
	outcomeIDs []string,
	specifiers string,
) (map[string]string, error) {
	result := make(map[string]string)

	for _, outcomeID := range outcomeIDs {
		name, err := r.ResolveOutcomeName(marketID, outcomeID, specifiers)
		if err != nil {
			// 记录错误但继续处理其他 outcomes
			result[outcomeID] = outcomeID // 使用 ID 作为 fallback
			continue
		}
		result[outcomeID] = name
	}

	return result, nil
}
