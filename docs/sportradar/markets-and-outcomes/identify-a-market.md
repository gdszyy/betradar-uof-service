# Identify a Market

Betradar Unified Odds 使用**市场**（markets）和**市场线**（market lines）。每个市场都是一种投注类型，由一个唯一的 ID 标识。在一个市场内，通常会提供多条不同的市场线。

每条市场线都通过额外的**说明符**（specifiers）进行唯一标识（例如，总进球数 2.5 与总进球数 1.5 属于同一市场，但它们是两条不同的市场线。它们的市场 ID 相同，但第一条市场线有一个说明符 `((goals=2.5))`，而另一条有一个说明符 `((goals=1.5))`，这使得它们能够被唯一标识）。

## 相同市场的两条不同市场线的 XML 示例

```xml
<market name="Total" id="18" specifiers="total=3.5" status="1">
    <outcome name="over 3.5" active="1" id="12" odds="2.3"/>
    <outcome name="under 3.5" active="1" id="13" odds="1.55"/>
</market>

<market name="Total" id="18" specifiers="total=2.75" status="1">
    <outcome name="under 2.5" active="1" id="13" odds="2.1"/>
    <outcome name="over 2.5" active="1" id="12" odds="1.65"/>
</market>
```

体育赛事的市场生命周期通常始于 Betradar 提供赛前赔率——通常在比赛开始前很久（并且同一市场会持续到滚球阶段）。如果 Betradar 不提供某项体育赛事的滚球覆盖，或者由于某种原因该市场不适合滚球，那么该市场将在比赛开始时关闭。

通常，Unified Odds 模型中的市场涵盖赛前和滚球阶段。如果体育赛事尚未开始、正在进行或已经结束，它们也会使用相同的 ID 和说明符。
