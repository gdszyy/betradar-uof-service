# 盘口分组配置更新说明

**更新日期**: 2024-12-02  
**版本**: v2.1

## 1. 概述

基于最终的盘口分组方案,更新了 `config/market_tabs_config.json` 配置文件,以支持前端的 Tab 和 Chip 展示逻辑。

## 2. 配置文件结构

### 2.1 源文件

- `config/final_tab_chip_config.csv` - Tab 配置源文件
- `config/final_chip_enumeration.csv` - Chip 枚举源文件
- `config/market_tabs_config.json` - 生成的 JSON 配置文件

### 2.2 配置统计

- **Tab 总数**: 17 个
- **Chip 总数**: 147 个 (不包括 dynamic 值)
- **市场总数**: 1,000+ 个

## 3. Tab 配置详情

| Tab ID | Tab 名称 | 类型 | 市场数量 | Chip Specifiers |
|--------|---------|------|----------|-----------------|
| regular_play | 常规玩法 | group | 198 | 无 |
| innings | 分局 | specifier_aggregate | 176 | inningnr, dismissalnr, deliverynr |
| player_props | 球员道具 | group | 135 | count, appearancenr |
| micro_market | 微盘口 | group | 78 | pitchnr, playnr, pointnr |
| sets | 分盘 | specifier_aggregate | 58 | setnr, gamenr, legnr, endnr |
| maps | 分地图 | specifier_aggregate | 48 | mapnr, roundnr |
| bookings | 罚牌 | group | 47 | 无 |
| corners | 角球 | group | 46 | cornernr |
| 1st_half | 上半场 | group | 43 | goalnr, pointnr |
| quarters | 分节 | specifier_aggregate | 22 | quarternr |
| combo | 组合玩法 | group | 20 | 无 |
| 2nd_half | 下半场 | group | 20 | goalnr |
| periods | 分时段 | specifier_aggregate | 20 | periodnr |
| frames | 分Frame | specifier_aggregate | 20 | framenr |
| scorers | 射手 | group | 13 | goalnr |
| overs | 分Over | specifier_aggregate | 7 | overnr |
| drives | 分Drive | specifier_aggregate | 2 | drivenr |

## 4. Chip 配置详情

### 4.1 主要 Chip 类型

#### 4.1.1 分节 (Quarters)
- 第1节 (quarternr=1)
- 第2节 (quarternr=2)
- 第3节 (quarternr=3)
- 第4节 (quarternr=4)

#### 4.1.2 分盘 (Sets)
- 第1盘 ~ 第5盘 (setnr=1~5)
- Game 1 ~ Game 13 (gamenr=1~13)

#### 4.1.3 分地图 (Maps)
- Map 1 ~ Map 5 (mapnr=1~5)
- Round 1 ~ Round 30 (roundnr=1~30)

#### 4.1.4 分局 (Innings)
- 第1局 ~ 第9局 (inningnr=1~9)

#### 4.1.5 分Over (Overs)
- 1 ~ 50 (overnr=1~50)

#### 4.1.6 分Frame (Frames)
- 1 ~ 19 (framenr=1~19)

#### 4.1.7 分Drive (Drives)
- 1 ~ 20 (drivenr=1~20)

#### 4.1.8 角球 (Corners)
- 第1个角球 ~ 第5个角球 (cornernr=1~5)

#### 4.1.9 射手 (Scorers)
- 第1球 ~ 第5球 (goalnr=1~5)

#### 4.1.10 上半场/下半场
- 第1球 ~ 第5球 (goalnr=1~5)

### 4.2 Dynamic Chip

以下 Chip 使用动态值,需要根据实际数据生成:
- `pointnr` (微盘口、上半场)
- `dismissalnr` (分局)
- `deliverynr` (分局)
- `pitchnr` (微盘口)
- `playnr` (微盘口)
- `count` (球员道具)
- `legnr` (分盘)
- `endnr` (分盘)

## 5. API 使用

### 5.1 获取配置

**端点**: `GET /api/config/market-tabs`

**响应示例**:
```json
{
  "tabs": [
    {
      "id": "regular_play",
      "label": "常规玩法",
      "type": "group",
      "groups": ["regular_play"],
      "chipSpecifiers": [],
      "marketCount": 198,
      "primarySpecifier": null
    },
    ...
  ],
  "chips": {
    "quarters_quarternr_1": {
      "tabId": "quarters",
      "label": "第1节",
      "specifier": "quarternr",
      "value": "1"
    },
    ...
  }
}
```

### 5.2 前端使用流程

1. **初始化**: 调用 `/api/config/market-tabs` 获取配置
2. **Tab 渲染**: 根据 `tabs` 数组渲染 Tab 导航
3. **Chip 渲染**: 根据选中的 Tab 和 `chipSpecifiers` 渲染 Chip 筛选器
4. **动态 Chip**: 对于 dynamic 类型的 Chip,从实际 market 数据中提取 specifier 值

## 6. 配置生成脚本

配置文件通过 Python 脚本自动生成:

```python
# 位置: /tmp/generate_market_tabs_config.py
# 输入: final_tab_chip_config.csv, final_chip_enumeration.csv
# 输出: config/market_tabs_config.json
```

**重新生成配置**:
```bash
cd /home/ubuntu/betradar-uof-service
python3 /tmp/generate_market_tabs_config.py
```

## 7. 数据验证

### 7.1 Tab 类型分布

- **group**: 9 个 (常规玩法、球员道具、微盘口、罚牌、角球、上半场、组合玩法、下半场、射手)
- **specifier_aggregate**: 8 个 (分局、分盘、分地图、分节、分时段、分Frame、分Over、分Drive)

### 7.2 市场覆盖率

配置覆盖了 Sportradar UOF 中的主要市场类型,包括:
- 传统体育 (足球、篮球、网球、棒球等)
- 电竞 (地图、回合)
- 特殊玩法 (微盘口、组合玩法)

## 8. 后续优化建议

### 8.1 Dynamic Chip 实现

需要在后端实现 dynamic chip 的动态生成逻辑:

```go
// 伪代码
func GenerateDynamicChips(tabId string, markets []Market) []Chip {
    chips := []Chip{}
    specifierValues := make(map[string]bool)
    
    for _, market := range markets {
        for key, value := range market.Specifiers {
            if isDynamicSpecifier(tabId, key) {
                specifierValues[value] = true
            }
        }
    }
    
    for value := range specifierValues {
        chips = append(chips, Chip{
            Label: formatChipLabel(tabId, value),
            Specifier: getDynamicSpecifier(tabId),
            Value: value,
        })
    }
    
    return chips
}
```

### 8.2 Market 到 Tab 的映射

需要在 `odds_change_parser.go` 中实现 market 到 tabId 的映射:

```go
func CalculateTabID(market Market) string {
    // 1. 检查 market.Groups
    for _, group := range market.Groups {
        if tabId := groupToTabMap[group]; tabId != "" {
            return tabId
        }
    }
    
    // 2. 检查 specifiers
    if hasSpecifier(market, "quarternr") {
        return "quarters"
    }
    if hasSpecifier(market, "setnr") {
        return "sets"
    }
    // ... 其他映射逻辑
    
    return "regular_play" // 默认
}
```

## 9. 版本历史

- **v2.1** (2024-12-02): 基于最终 CSV 配置更新
- **v2.0** (2024-12-02): 初始版本,包含 17 个基础 Tab

---

**维护者**: Manus AI  
**最后更新**: 2024-12-02
