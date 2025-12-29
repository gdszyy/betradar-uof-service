# Railway 环境变量配置指南

## 🔵 Integration Environment（集成测试环境）

如果您的Access Token来自 https://integration.portal.betradar.com/

### 必需的环境变量

```
BETRADAR_ACCESS_TOKEN=your_integration_token
BETRADAR_MESSAGING_HOST=mq.betradar.com:5671
BETRADAR_API_BASE_URL=https://api.betradar.com/v1
ROUTING_KEYS=#
DATABASE_URL=${{Postgres.DATABASE_URL}}
```

### 在Railway中配置步骤

1. 进入您的Go服务
2. 点击 **"Variables"** 标签
3. 添加以下变量：

| 变量名 | 值 |
|--------|-----|
| `BETRADAR_ACCESS_TOKEN` | 您的integration token |
| `BETRADAR_MESSAGING_HOST` | `mq.betradar.com:5671` |
| `BETRADAR_API_BASE_URL` | `https://api.betradar.com/v1` |
| `ROUTING_KEYS` | `#` |

4. 添加数据库引用：
   - 点击 **"+ New Variable"**
   - 选择 **"Add Reference"**
   - 选择PostgreSQL数据库
   - 选择 `DATABASE_URL`

---

## 🟡 Staging Environment（如果使用）

如果您使用的是staging环境：

```
BETRADAR_ACCESS_TOKEN=your_staging_token
BETRADAR_MESSAGING_HOST=stgmq.betradar.com:5671
BETRADAR_API_BASE_URL=https://stgapi.betradar.com/v1
ROUTING_KEYS=#
DATABASE_URL=${{Postgres.DATABASE_URL}}
```

---

## 🟢 Production Environment（生产环境）

生产环境使用相同的服务器地址：

```
BETRADAR_ACCESS_TOKEN=your_production_token
BETRADAR_MESSAGING_HOST=mq.betradar.com:5671
BETRADAR_API_BASE_URL=https://api.betradar.com/v1
ROUTING_KEYS=#
DATABASE_URL=${{Postgres.DATABASE_URL}}
```

**注意：** Integration和Production使用相同的服务器地址，但token不同！

---

## ⚠️ 重要说明

1. **Integration环境特点**
   - 24/5运行（周一到周五）
   - 周末会有计划性断开
   - 用于开发和测试
   - 免费使用

2. **Token区分**
   - Integration token只能访问Integration环境
   - Production token只能访问Production环境
   - 不能混用！

3. **验证配置**
   - 确保token来自正确的portal
   - Integration: https://integration.portal.betradar.com/
   - Production: https://portal.betradar.com/

---

## 🔧 修改现有配置

如果您已经部署但配置错误：

1. 进入Railway项目
2. 点击Go服务
3. 进入 **"Variables"** 标签
4. 修改以下变量：
   - `BETRADAR_MESSAGING_HOST` → `mq.betradar.com:5671`
   - `BETRADAR_API_BASE_URL` → `https://api.betradar.com/v1`
5. 保存后会自动重新部署

---

## ✅ 验证配置正确

部署后查看日志，应该看到：

```
✓ Bookmaker ID: 45426
✓ Virtual Host: /unifiedfeed/45426
✓ Connecting to AMQP (vhost: /unifiedfeed/45426)...
✓ Connected to AMQP server
✓ Queue declared: amq.gen-xxxxx
✓ Started consuming messages
```

如果看到 `403 no access to this vhost`，说明：
- Token环境不匹配
- 或服务器地址配置错误

