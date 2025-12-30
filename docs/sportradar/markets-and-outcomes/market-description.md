# 市场描述 (Market Description)

市场描述和结果描述使用模板语言来描述实际的市场或结果。这通过一个示例最容易理解。

## 基础模板

市场名称模板中，任何位于花括号 `{}` 之间的内容都可以被 `odds_change` 消息中的实际市场信息替换。通常，花括号中的内容是一个**指示符（specifier）**的名称，应被该指示符的特定值替换。

**XML 示例：市场模板**

```xml
<market id="300" name="Race to {pointnr} points">
```

如果您收到一个包含以下指示符的 `odds_change` 消息：

**XML 示例：指示符**

```xml
<market id="300" specifiers="pointnr=3">
```

您应该将市场名称显示为：“Race to 3 points”。

> **注意：** SDK 会自动处理这些市场描述，因此在使用 SDK 时，您无需自行管理上述任何内容。

## 特殊前缀

模板语言支持在指示符名称前使用特殊前缀，以实现特定的格式化效果。

### 1. 序数（Ordinal）前缀 `!`

如果开花括号后的第一个字符是 `!`，后跟一个指示符名称，则应将花括号表达式替换为该指示符的**序数**形式。

**XML 示例：序数模板**

```xml
<market id="446" name="{!periodnr} period - total">
```

如果您收到一个包含以下指示符的 `odds_change` 消息：

**XML 示例：序数指示符**

```xml
<market id="446" specifiers="periodnr=2">
```

您应该将市场名称显示为：“**Second** period – total”。

### 2. 正负号（Sign）前缀 `+` 和 `-`

| 前缀 | 描述 |
| :--- | :--- |
| `+` | 指示符必须是数字类型。您应该将花括号表达式替换为指示符的值，并在前面添加正确的 `+/-` 符号。 |
| `-` | 指示符必须是数字类型。您应该将花括号表达式替换为**取反**的指示符值，并在前面添加正确的 `+/-` 符号。 |

## 特殊关键字

有几个特殊的关键字也可以出现在花括号内，它们具有特殊的含义：

| 关键字 | 描述 | 示例 |
| :--- | :--- | :--- |
| `{$competitor1}` | 替换为体育赛事中第一个参赛者的名称。这通常出现在结果描述中。 | N/A |
| `{$competitor2}` | 替换为体育赛事中第二个参赛者的名称。这通常出现在结果描述中。 | N/A |
| `{$event}` | 替换为赛事的名称。这通常用于独赢（Outright）市场。 | N/A |
| `{%player}` | 替换为指示符的名称（通常是玩家或参赛者的 ID）。 | `{%player} total dismissals` -> `John Rodriquez total dismissals` |

## 市场描述模板示例

下表总结了市场描述模板的各种替换规则和示例：

| 模板表达式 | 替换规则 | 示例 |
| :--- | :--- | :--- |
| `{X}` | 替换为指示符 X 的值。 | `Race to {pointnr} points` |
| `{!X}` | 替换为指示符 X 的**序数**值。 | N/A |
| `{+X}` | 替换为指示符 X 的值，并在前面添加 `+/-` 符号。 | N/A |
| `{-X}` | 替换为指示符 X 的**取反**值，并在前面添加 `+/-` 符号。 | N/A |
| `{X+c}` 或 `{X-c}` | 替换为指示符 X 的值 `+` 或 `-` 数字 `c`。 | N/A |
| `{$competitorN}` | 替换为赛事中的第 N 个参赛者（或根据情况替换为“TeamN”或“PlayerN”）。 | N/A |
| `{$event}` | 替换为赛事的名称。 | N/A |
| `{%player}` | 替换为指示符的名称（通常是玩家或参赛者的 ID）。 | `{%player} total dismissals` |

## product\_ids 属性

在某些情况下，除了正常的 `product_id` 属性外，还会显示 `product_ids` 属性。

这表明某些映射可用于多个赔率提供商。当映射同时包含 Live Odds 生产商和 BetPal 生产商时，通常会出现这种情况。在这种情况下，属性将显示为：`product_ids="1|4"`（`1` 代表 Live Odds，`4` 代表 BetPal）。

Sportradar 正在逐步淘汰使用 `product_id` 属性，因此客户端应开始使用 `product_ids`。

**XML 示例：包含 product\_ids 的映射**

```xml
<mappings>
    <mapping product_id="1" product_ids="1|4" sport_id="sr:sport:6" market_id="8:232" sov_template="{total}">
      <mapping_outcome outcome_id="13" product_outcome_id="2528" product_outcome_name="under"/>
      <mapping_outcome outcome_id="12" product_outcome_id="2530" product_outcome_name="over"/>
</mapping>
```
