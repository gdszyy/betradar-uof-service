package services

import (
	"strings"
	"uof-service/logger"
)

// GetMarketNameImproved 获取市场名称 (改进版)
// 支持完整的模板替换，包括序数、正负号等特殊前缀
func (s *MarketDescriptionsService) GetMarketNameImproved(marketID string, specifiers string, ctx *ReplacementContext) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if market, ok := s.markets[marketID]; ok {
		name := market.Name

		// 1. 替换 specifiers (包括特殊前缀)
		name = replaceSpecifiers(name, specifiers)

		// 2. 替换竞争者占位符
		if ctx != nil {
			name = replaceCompetitors(name, ctx.HomeTeamName, ctx.AwayTeamName)
		}

		return name
	}

	logger.Printf("[MarketDescService] ⚠️  Market not found: %s", marketID)
	return marketID
}

// GetOutcomeNameImproved 获取结果名称 (改进版)
// 支持完整的模板替换和 Variant Market 的同步查询
func (s *MarketDescriptionsService) GetOutcomeNameImproved(marketID, outcomeID, specifiers string, ctx *ReplacementContext) string {
	s.mu.RLock()

	// 第一优先级: 从 outcomes 中查找
	if outcomes, ok := s.outcomes[marketID]; ok {
		if outcome, ok := outcomes[outcomeID]; ok {
			s.mu.RUnlock()
			name := outcome.Name

			// 1. 替换 specifiers (包括特殊前缀)
			name = replaceSpecifiers(name, specifiers)

			// 2. 替换竞争者占位符
			if ctx != nil {
				name = replaceCompetitors(name, ctx.HomeTeamName, ctx.AwayTeamName)
			}

			return name
		}
	}

	s.mu.RUnlock()

	// 第二优先级: 检查是否是 Variant Market
	if strings.Contains(specifiers, "variant=") {
		variantURN := s.extractVariantURN(specifiers)
		if variantURN != "" {
			// 再次检查缓存 (可能在上面的 RUnlock 之后被其他协程填充)
			s.mu.RLock()
			if outcomes, ok := s.outcomes[marketID]; ok {
				if outcome, ok := outcomes[outcomeID]; ok {
					s.mu.RUnlock()
					return outcome.Name
				}
			}
			s.mu.RUnlock()

			// 缓存中仍然没有，同步调用 API
			logger.Printf("[MarketDescService] ⚡️ Variant outcome not cached, fetching synchronously: marketID=%s, outcomeID=%s, variant=%s", marketID, outcomeID, variantURN)
			name, err := s.fetchAndCacheVariant(marketID, outcomeID, variantURN)
			if err != nil {
				logger.Printf("[MarketDescService] ⚠️  Failed to fetch variant synchronously: %v", err)
				// 降级: 返回 outcomeID
				return outcomeID
			}
			return name
		}
	}

	// 第三优先级: 检查是否是球员市场
	if strings.HasPrefix(outcomeID, "sr:player:") {
		if s.playersService != nil {
			playerName := s.playersService.GetPlayerName(outcomeID)
			return playerName
		}
		return outcomeID
	}

	// 第四优先级: 从 mappings 中查询 (仅用于特殊情况的降级)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if mappings, ok := s.mappings[marketID]; ok {
		if productOutcomeName, ok := mappings[outcomeID]; ok {
			logger.Printf("[MarketDescService] ℹ️  Using mapping fallback for marketID=%s, outcomeID=%s, name=%s", marketID, outcomeID, productOutcomeName)
			return productOutcomeName
		}
	}

	// 最终降级: 返回 outcomeID
	logger.Printf("[MarketDescService] ⚠️  Outcome name not found: marketID=%s, outcomeID=%s, specifiers=%s", marketID, outcomeID, specifiers)
	return outcomeID
}
