## 架构重构文档

**版本**: v2.0.0  
**更新日期**: 2025-10-23  
**状态**: 🚧 进行中

---

## 📋 重构目标

按照分层架构图重构代码,建立清晰的分层结构,提高代码的可维护性、可扩展性和可测试性。

---

## 🏗️ 架构设计

### 分层结构

```
┌─────────────────────────────────────────┐
│         外部数据源层                      │
│  UOF | Live Data | The Sports | MTS     │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       统一数据接入层 (Ingestion)          │
│  数据接入管理器 | 连接管理器 | 恢复管理器  │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       核心数据处理层 (Processing)         │
│  处理中心 | 验证中心 | 存储中心 | 事件中心 │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       业务服务层 (Business)              │
│  赔率服务 | 赛事服务 | 结算服务 | 通知服务 │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       业务接口层 (Interfaces)            │
│  API 服务 | WebSocket | Webhook         │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       数据存储层 (Storage)               │
│  PostgreSQL | Redis (可选)              │
└─────────────────────────────────────────┘
```

---

## 📁 目录结构

### 新的包结构

```
uof-service/
├── pkg/
│   ├── models/              # 统一数据模型
│   │   ├── event.go         # 事件模型
│   │   ├── match.go         # 比赛模型
│   │   └── odds.go          # 赔率模型
│   │
│   ├── ingestion/           # 数据接入层
│   │   ├── interfaces.go    # 接口定义
│   │   ├── uof/            # UOF 数据源实现
│   │   ├── livedata/       # Live Data 数据源实现
│   │   ├── thesports/      # The Sports 数据源实现
│   │   ├── manager.go      # 连接管理器
│   │   └── recovery.go     # 恢复管理器
│   │
│   ├── processing/          # 数据处理层
│   │   ├── interfaces.go    # 接口定义
│   │   ├── processor.go     # 数据处理器
│   │   ├── validator.go     # 数据验证器
│   │   ├── storage.go       # 数据存储
│   │   └── dispatcher.go    # 事件分发器
│   │
│   ├── business/            # 业务服务层
│   │   ├── interfaces.go    # 接口定义
│   │   ├── odds.go          # 赔率服务
│   │   ├── match.go         # 赛事服务
│   │   ├── settlement.go    # 结算服务
│   │   ├── notification.go  # 通知服务
│   │   ├── subscription.go  # 订阅服务
│   │   └── booking.go       # 预订服务
│   │
│   ├── interfaces/          # 接口层
│   │   ├── interfaces.go    # 接口定义
│   │   ├── api/            # API 服务
│   │   ├── websocket/      # WebSocket 服务
│   │   └── webhook/        # Webhook 服务
│   │
│   └── common/              # 公共包
│       ├── logger.go        # 日志
│       ├── errors.go        # 错误定义
│       └── utils.go         # 工具函数
│
├── internal/                # 内部实现(不对外暴露)
│   ├── adapters/           # 适配器
│   └── repositories/       # 数据仓库
│
├── cmd/                     # 命令行入口
│   └── server/
│       └── main.go
│
├── config/                  # 配置
├── docs/                    # 文档
└── tests/                   # 测试
```

---

## 🎯 重构阶段

### ✅ 阶段 1: 创建分层架构基础 (已完成)

**完成内容**:
- ✅ 定义统一数据模型 (Event, Match, Odds)
- ✅ 定义数据接入层接口
- ✅ 定义数据处理层接口
- ✅ 定义业务服务层接口
- ✅ 定义接口层接口
- ✅ 创建公共工具包 (Logger, Errors)

**新增文件**:
- `pkg/models/event.go` - 事件模型
- `pkg/models/match.go` - 比赛模型
- `pkg/models/odds.go` - 赔率模型
- `pkg/ingestion/interfaces.go` - 数据接入层接口
- `pkg/processing/interfaces.go` - 数据处理层接口
- `pkg/business/interfaces.go` - 业务服务层接口
- `pkg/interfaces/interfaces.go` - 接口层接口
- `pkg/common/logger.go` - 日志工具
- `pkg/common/errors.go` - 错误定义

---

### 🚧 阶段 2: 重构数据接入层 (进行中)

**目标**:
- 实现 UOF 数据源适配器
- 实现连接管理器
- 实现数据恢复管理器
- 迁移现有 UOF 客户端代码

**待完成**:
- [ ] 创建 UOF 数据源实现
- [ ] 创建连接管理器实现
- [ ] 创建恢复管理器实现
- [ ] 迁移现有代码到新架构

---

### ⏳ 阶段 3: 重构核心处理层

**目标**:
- 实现数据处理中心
- 实现数据验证中心
- 实现数据存储中心
- 实现事件分发中心

---

### ⏳ 阶段 4: 重构业务服务层

**目标**:
- 实现赔率管理服务
- 实现赛事管理服务
- 实现结算管理服务
- 实现通知管理服务
- 实现订阅管理服务
- 实现预订管理服务

---

### ⏳ 阶段 5: 重构接口层

**目标**:
- 实现 API 服务
- 实现 WebSocket 服务
- 实现 Webhook 服务
- 迁移现有 API 端点

---

### ⏳ 阶段 6: 测试和优化

**目标**:
- 编写单元测试
- 编写集成测试
- 性能测试和优化
- 文档完善

---

## 🎨 设计原则

### 1. 依赖倒置原则 (DIP)
- 高层模块不依赖低层模块,都依赖抽象
- 抽象不依赖细节,细节依赖抽象

### 2. 单一职责原则 (SRP)
- 每个模块只负责一个功能
- 每个接口只定义一类操作

### 3. 开闭原则 (OCP)
- 对扩展开放,对修改关闭
- 通过接口和抽象实现扩展

### 4. 接口隔离原则 (ISP)
- 接口应该小而专注
- 客户端不应依赖不需要的接口

### 5. 里氏替换原则 (LSP)
- 子类可以替换父类
- 实现可以替换接口

---

## 📊 接口设计

### 数据接入层接口

```go
// DataSource 数据源基础接口
type DataSource interface {
    Connect(ctx context.Context) error
    Disconnect() error
    IsConnected() bool
    GetName() string
    GetType() SourceType
}

// EventDataSource 事件数据源
type EventDataSource interface {
    DataSource
    Subscribe(ctx context.Context, filter EventFilter) error
    Unsubscribe(ctx context.Context, filter EventFilter) error
    GetEventChannel() <-chan *models.Event
}

// MatchDataSource 比赛数据源
type MatchDataSource interface {
    DataSource
    GetMatches(ctx context.Context, filter MatchFilter) ([]*models.Match, error)
    GetMatch(ctx context.Context, matchID string) (*models.Match, error)
    SubscribeMatch(ctx context.Context, matchID string) error
    UnsubscribeMatch(ctx context.Context, matchID string) error
}
```

### 数据处理层接口

```go
// DataProcessor 数据处理器
type DataProcessor interface {
    Process(ctx context.Context, event *models.Event) error
    GetName() string
}

// DataValidator 数据验证器
type DataValidator interface {
    Validate(ctx context.Context, event *models.Event) error
    GetName() string
}

// DataStorage 数据存储
type DataStorage interface {
    SaveEvent(ctx context.Context, event *models.Event) error
    SaveMatch(ctx context.Context, match *models.Match) error
    SaveOdds(ctx context.Context, odds *models.Odds) error
    GetEvent(ctx context.Context, eventID string) (*models.Event, error)
    GetMatch(ctx context.Context, matchID string) (*models.Match, error)
    GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error)
}
```

### 业务服务层接口

```go
// OddsService 赔率服务
type OddsService interface {
    GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error)
    GetOddsHistory(ctx context.Context, matchID string, marketID string) ([]*models.Odds, error)
    SubscribeOdds(ctx context.Context, matchID string) error
    UnsubscribeOdds(ctx context.Context, matchID string) error
}

// MatchService 赛事服务
type MatchService interface {
    GetMatch(ctx context.Context, matchID string) (*models.Match, error)
    GetMatches(ctx context.Context, filter MatchFilter) ([]*models.Match, error)
    GetLiveMatches(ctx context.Context) ([]*models.Match, error)
    GetTodayMatches(ctx context.Context) ([]*models.Match, error)
}
```

---

## 🔄 迁移策略

### 1. 渐进式迁移
- 新功能使用新架构
- 旧功能逐步迁移
- 保持 API 兼容性

### 2. 适配器模式
- 为旧代码创建适配器
- 实现新接口
- 逐步替换实现

### 3. 并行运行
- 新旧代码并行运行
- 逐步切换流量
- 验证功能正确性

---

## ✅ 重构检查清单

### 代码质量
- [ ] 所有接口都有清晰的文档
- [ ] 所有实现都遵循接口契约
- [ ] 错误处理统一且完善
- [ ] 日志记录统一且详细

### 测试覆盖
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试覆盖核心流程
- [ ] 性能测试验证性能指标

### 文档完善
- [ ] API 文档完整
- [ ] 架构文档更新
- [ ] 部署文档更新
- [ ] 迁移指南完整

---

## 📈 预期收益

### 1. 可维护性提升
- 清晰的分层结构
- 单一职责原则
- 易于理解和修改

### 2. 可扩展性提升
- 接口驱动设计
- 依赖注入
- 易于添加新功能

### 3. 可测试性提升
- 接口可 mock
- 依赖可注入
- 易于编写测试

### 4. 代码质量提升
- 统一的错误处理
- 统一的日志记录
- 统一的数据模型

---

## 🚀 下一步

1. **完成阶段 2**: 重构数据接入层
2. **编写适配器**: 为现有代码创建适配器
3. **编写测试**: 为新代码编写测试
4. **逐步迁移**: 将旧代码迁移到新架构

---

**文档版本**: v2.0.0  
**最后更新**: 2025-10-23  
**状态**: 🚧 进行中

