package services

import (
	"math"
)

// PopularityCalculator 热门度计算器
type PopularityCalculator struct{}

// NewPopularityCalculator 创建热门度计算器
func NewPopularityCalculator() *PopularityCalculator {
	return &PopularityCalculator{}
}

// PopularityFactors 热门度因素
type PopularityFactors struct {
	Attendance         *int  // 到场人数
	Sellout            *bool // 是否售罄
	FeatureMatch       *bool // 是否焦点赛
	LiveVideoAvailable *bool // 是否提供直播
	LiveDataAvailable  *bool // 是否提供实时数据
	BroadcastsCount    *int  // 转播平台数量
}

// CalculatePopularityScore 计算热门度评分 (0-100)
//
// 评分规则:
// - 焦点赛: +30 分
// - 售罄: +25 分
// - 到场人数: 最高 +20 分 (根据人数比例计算)
// - 转播平台数量: 每个平台 +5 分，最高 +15 分
// - 提供直播视频: +5 分
// - 提供实时数据: +5 分
//
// 总分最高 100 分
func (c *PopularityCalculator) CalculatePopularityScore(factors PopularityFactors) float64 {
	score := 0.0
	
	// 1. 焦点赛 (+30 分)
	if factors.FeatureMatch != nil && *factors.FeatureMatch {
		score += 30.0
	}
	
	// 2. 售罄 (+25 分)
	if factors.Sellout != nil && *factors.Sellout {
		score += 25.0
	}
	
	// 3. 到场人数 (最高 +20 分)
	if factors.Attendance != nil && *factors.Attendance > 0 {
		attendance := float64(*factors.Attendance)
		
		// 根据人数分段计算分数
		if attendance >= 50000 {
			// 5万人以上: 满分 20 分
			score += 20.0
		} else if attendance >= 30000 {
			// 3-5万人: 15-20 分
			score += 15.0 + (attendance-30000)/20000*5.0
		} else if attendance >= 10000 {
			// 1-3万人: 10-15 分
			score += 10.0 + (attendance-10000)/20000*5.0
		} else if attendance >= 5000 {
			// 5千-1万人: 5-10 分
			score += 5.0 + (attendance-5000)/5000*5.0
		} else {
			// 5千人以下: 0-5 分
			score += attendance / 5000 * 5.0
		}
	}
	
	// 4. 转播平台数量 (每个 +5 分，最高 +15 分)
	if factors.BroadcastsCount != nil && *factors.BroadcastsCount > 0 {
		broadcastScore := float64(*factors.BroadcastsCount) * 5.0
		if broadcastScore > 15.0 {
			broadcastScore = 15.0
		}
		score += broadcastScore
	}
	
	// 5. 提供直播视频 (+5 分)
	if factors.LiveVideoAvailable != nil && *factors.LiveVideoAvailable {
		score += 5.0
	}
	
	// 6. 提供实时数据 (+5 分)
	if factors.LiveDataAvailable != nil && *factors.LiveDataAvailable {
		score += 5.0
	}
	
	// 限制在 0-100 范围内
	score = math.Min(100.0, math.Max(0.0, score))
	
	// 保留两位小数
	return math.Round(score*100) / 100
}

// IsPopularMatch 判断是否为热门比赛
// 热门比赛定义: 焦点赛 OR 售罄 OR 转播数 > 0 OR 热门度评分 > 50
func (c *PopularityCalculator) IsPopularMatch(factors PopularityFactors) bool {
	// 焦点赛
	if factors.FeatureMatch != nil && *factors.FeatureMatch {
		return true
	}
	
	// 售罄
	if factors.Sellout != nil && *factors.Sellout {
		return true
	}
	
	// 有转播
	if factors.BroadcastsCount != nil && *factors.BroadcastsCount > 0 {
		return true
	}
	
	// 热门度评分高
	score := c.CalculatePopularityScore(factors)
	return score > 50.0
}

// GetPopularityLevel 获取热门度等级
func (c *PopularityCalculator) GetPopularityLevel(score float64) string {
	if score >= 80 {
		return "very_hot" // 超级热门
	} else if score >= 60 {
		return "hot" // 热门
	} else if score >= 40 {
		return "warm" // 较热门
	} else if score >= 20 {
		return "normal" // 普通
	} else {
		return "cold" // 冷门
	}
}

