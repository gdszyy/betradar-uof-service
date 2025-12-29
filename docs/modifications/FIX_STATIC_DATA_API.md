# 修复静态数据 API 404 错误

## 问题描述

在 Railway 日志中出现多个静态数据 API 404 错误:

```
2025/11/10 06:51:06 [StaticData] ⚠️  Failed to load sports: failed to fetch sports: API returned status 404
2025/11/10 06:51:06 [StaticData] ⚠️  Failed to load categories: failed to fetch categories: API returned status 404
2025/11/10 06:51:06 [StaticData] ⚠️  Failed to load tournaments: failed to fetch tournaments: API returned status 404
2025/11/10 06:51:06 [StaticData] ⚠️  Failed to load void reasons: failed to fetch void reasons: API returned status 404
```

## 根本原因

代码中使用的 API 端点路径不正确,与 Sportradar 官方文档不符。

### 错误的 API 路径

| 功能 | 错误路径 | 状态 |
|------|---------|------|
| Sports | `/descriptions/en/sports.xml` | ❌ 404 |
| Categories | `/descriptions/en/categories.xml` | ❌ 404 |
| Tournaments | `/descriptions/en/tournaments.xml` | ❌ 404 |
| Void Reasons | `/descriptions/en/void_reasons.xml` | ❌ 404 |
| Betstop Reasons | `/descriptions/en/betstop_reasons.xml` | ❌ 404 |

## 解决方案

根据 [Sportradar UOF 官方文档](https://docs.sportradar.com/uof/api-and-structure/api/),正确的 API 端点如下:

### 修复后的 API 路径

| 功能 | 正确路径 | 状态 |
|------|---------|------|
| Sports | `/sports/en/sports.xml` | ✅ 已修复 |
| Void Reasons | `/descriptions/void_reasons.xml` | ✅ 已修复 |
| Betstop Reasons | `/descriptions/betstop_reasons.xml` | ✅ 已修复 |
| Categories | `/sports/en/sports/sr:sport:{id}/categories.xml` | ⚠️ 需要 sport_id 参数 |
| Tournaments | `/sports/en/sports/{sport_id}/tournaments.xml` | ⚠️ 需要 sport_id 参数 |

### Categories 和 Tournaments 的特殊说明

根据 Sportradar API 文档:

1. **Categories** 端点需要指定 sport ID:
   ```
   GET /sports/{language}/sports/sr:sport:{id}/categories.xml
   ```
   例如: `/sports/en/sports/sr:sport:1/categories.xml` (足球的分类)

2. **Tournaments** 端点需要指定 sport ID:
   ```
   GET /sports/{language}/sports/{sport_id}/tournaments.xml
   ```
   例如: `/sports/en/sports/sr:sport:1/tournaments.xml` (足球的锦标赛)

由于这两个端点需要遍历所有 sports 才能获取完整数据,当前版本暂时禁用了这两个功能。

## 修改内容

### 文件: `services/static_data_service.go`

#### 1. 修复 Sports API 路径
```go
// 修复前
url := fmt.Sprintf("%s/descriptions/en/sports.xml", s.apiBaseURL)

// 修复后
url := fmt.Sprintf("%s/sports/en/sports.xml", s.apiBaseURL)
```

#### 2. 修复 Void Reasons API 路径
```go
// 修复前
url := fmt.Sprintf("%s/descriptions/en/void_reasons.xml", s.apiBaseURL)

// 修复后
url := fmt.Sprintf("%s/descriptions/void_reasons.xml", s.apiBaseURL)
```

#### 3. 修复 Betstop Reasons API 路径
```go
// 修复前
url := fmt.Sprintf("%s/descriptions/en/betstop_reasons.xml", s.apiBaseURL)

// 修复后
url := fmt.Sprintf("%s/descriptions/betstop_reasons.xml", s.apiBaseURL)
```

#### 4. 暂时禁用 Categories 和 Tournaments
```go
// 加载 Categories (需要按 sport 查询,暂时禁用)
// if err := s.LoadCategories(); err != nil {
//     logger.Errorf("[StaticData] ⚠️  Failed to load categories: %v", err)
// }

// 加载 Tournaments (需要按 sport/category 查询,暂时禁用)
// if err := s.LoadTournaments(); err != nil {
//     logger.Errorf("[StaticData] ⚠️  Failed to load tournaments: %v", err)
// }
```

## 验证修复

修复后,重启服务应该看到:

### ✅ 成功的日志
```
[StaticData] 📥 Loading sports from: https://stgapi.betradar.com/v1/sports/en/sports.xml
[StaticData] ✅ Loaded XX sports
[StaticData] 📥 Loading void reasons from: https://stgapi.betradar.com/v1/descriptions/void_reasons.xml
[StaticData] ✅ Loaded XX void reasons
[StaticData] 📥 Loading betstop reasons from: https://stgapi.betradar.com/v1/descriptions/betstop_reasons.xml
[StaticData] ✅ Loaded XX betstop reasons
[StaticData] ✅ All static data loaded
```

### ❌ 不应再出现的错误
```
[StaticData] ⚠️  Failed to load sports: failed to fetch sports: API returned status 404
[StaticData] ⚠️  Failed to load void reasons: failed to fetch void reasons: API returned status 404
[StaticData] ⚠️  Failed to load betstop reasons: failed to fetch betstop reasons: API returned status 404
```

## 未来改进

如果需要加载 Categories 和 Tournaments 数据,可以:

1. 首先加载所有 Sports
2. 遍历每个 Sport ID
3. 为每个 Sport 调用对应的 Categories 和 Tournaments 端点

示例代码:
```go
func (s *StaticDataService) LoadAllCategories() error {
    // 1. 获取所有 sports
    sports, err := s.GetAllSports()
    if err != nil {
        return err
    }
    
    // 2. 遍历每个 sport 加载 categories
    for _, sport := range sports {
        url := fmt.Sprintf("%s/sports/en/sports/%s/categories.xml", 
            s.apiBaseURL, sport.ID)
        // ... 加载和保存逻辑
    }
    
    return nil
}
```

## 参考文档

- [Sportradar UOF API - All Available Sports](https://docs.sportradar.com/uof/api-and-structure/api/sport-event-information/all-available-sports-endpoint/endpoint)
- [Sportradar UOF API - Categories for a Sport](https://docs.sportradar.com/uof/api-and-structure/api/sport-event-information/categories-for-a-sport-endpoint/endpoint)
- [Sportradar UOF API - Betstop Descriptions](https://docs.sportradar.com/uof/api-and-structure/api/betting-descriptions/betstop-descriptions/endpoint)
- [Sportradar UOF API - Void Reasons](https://docs.sportradar.com/uof/api-and-structure/api/betting-descriptions/void-descriptions/endpoint)

---

## 关于 Producer 14 警告

日志中的这个警告是**正常的**:
```
2025/11/10 07:02:28 [AliveMessage] ⚠️  Producer 14 subscription cancelled! All markets should be suspended.
```

### 说明

- Producer 14 的订阅被取消 (`subscribed=0`)
- 如果你只订阅了 Producer 1 (Live Odds) 和 Producer 3 (Live Betting),那么收到其他 Producer 的取消通知是正常的
- 这是 UOF 系统的标准行为,用于通知客户端哪些生产者的订阅状态发生了变化
- 代码会自动更新数据库中的 producer_status 表

### Producer 列表

常见的 UOF Producers:

| Producer ID | 名称 | 说明 |
|------------|------|------|
| 1 | Live Odds | 实时赔率 |
| 3 | Live Betting | 实时投注 |
| 4 | Prematch | 赛前数据 |
| 5 | Virtual Sports | 虚拟体育 |
| 14 | Statistics | 统计数据 |

如果不需要 Producer 14 的告警通知,可以在代码中添加过滤逻辑:

```go
// 只对订阅的 producers 发送告警
subscribedProducers := []int{1, 3} // 你订阅的 producers
if alive.Subscribed == 0 && contains(subscribedProducers, alive.ProductID) {
    logger.Printf("[AliveMessage] ⚠️  Producer %d subscription cancelled!", alive.ProductID)
    // 发送告警...
}
```

---

**修复完成时间**: 2025-11-10  
**影响范围**: 静态数据加载功能  
**风险等级**: 低 (只影响静态数据缓存,不影响核心功能)
