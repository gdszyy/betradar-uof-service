# Railway 快速开始指南

5分钟快速部署Betradar UOF服务到Railway。

## 📋 准备清单

- [ ] Railway账号（https://railway.app/）
- [ ] GitHub账号（推荐）
- [ ] Betradar Access Token

## 🚀 快速部署（5步）

### 1️⃣ 上传代码到GitHub

```bash
# 解压项目
tar -xzf betradar-uof-go-service.tar.gz
cd uof-go-service

# 初始化Git仓库
git init
git add .
git commit -m "Initial commit"

# 推送到GitHub
git remote add origin https://github.com/your-username/uof-service.git
git push -u origin main
```

### 2️⃣ 在Railway创建项目

1. 访问 https://railway.app/
2. 点击 **"New Project"**
3. 选择 **"Deploy from GitHub repo"**
4. 选择刚才创建的仓库

### 3️⃣ 添加PostgreSQL

1. 点击 **"+ New"**
2. 选择 **"Database"** → **"Add PostgreSQL"**
3. 等待数据库启动（约10秒）

### 4️⃣ 配置环境变量

在Go服务中，点击 **"Variables"** 标签，添加：

```
BETRADAR_ACCESS_TOKEN=your_token_here
BETRADAR_MESSAGING_HOST=stgmq.betradar.com:5671
BETRADAR_API_BASE_URL=https://stgapi.betradar.com/v1
ROUTING_KEYS=#
```

然后添加数据库引用：
- 点击 **"+ New Variable"** → **"Add Reference"**
- 选择PostgreSQL → DATABASE_URL

### 5️⃣ 生成域名并访问

1. 在Go服务的 **"Settings"** 中
2. 找到 **"Networking"** → 点击 **"Generate Domain"**
3. 复制生成的域名（例如：`xxx.up.railway.app`）
4. 在浏览器中访问该域名

**完成！** 🎉

## ✅ 验证部署

### 检查日志

在 **"Logs"** 标签中，应该看到：

```
Starting Betradar UOF Service...
Database connected and migrated
AMQP consumer started
Bookmaker ID: xxxxx
Connected to AMQP server
Web server started on port 8080
```

### 测试API

```bash
curl https://your-domain.up.railway.app/api/health
```

应该返回：
```json
{"status":"ok","time":1234567890}
```

### 访问Web界面

打开浏览器访问：
```
https://your-domain.up.railway.app/
```

点击"连接"按钮，开始接收实时消息！

## 🔧 常用命令

### 查看实时日志
```bash
# 安装CLI
npm install -g @railway/cli

# 登录
railway login

# 链接项目
railway link

# 查看日志
railway logs
```

### 连接数据库
```bash
# 获取数据库URL
railway variables

# 连接
psql $DATABASE_URL
```

### 重新部署
```bash
# 推送新代码
git add .
git commit -m "Update"
git push

# Railway自动重新部署
```

## 📊 监控面板功能

访问 `https://your-domain.up.railway.app/` 后：

1. **查看统计** - 总消息数、赔率变化、投注停止等
2. **实时日志** - 查看接收到的所有消息
3. **跟踪赛事** - 查看正在跟踪的比赛
4. **订阅过滤** - 只接收特定赛事或消息类型

## 🎯 下一步

### 自定义Routing Keys

只接收特定类型的消息，减少存储和流量：

```
# 只接收足球实时赔率
ROUTING_KEYS=*.*.live.odds_change.1.#

# 接收多种消息
ROUTING_KEYS=*.*.live.odds_change.#,*.*.live.bet_stop.#,-.-.-.alive.#
```

### 集成到您的应用

```javascript
// 前端代码
<script src="https://your-domain.up.railway.app/uof-client.js"></script>
<script>
  const client = new UOFClient({
    wsUrl: 'wss://your-domain.up.railway.app/ws',
    apiUrl: 'https://your-domain.up.railway.app/api'
  });
  
  client.connect();
  
  client.on('odds_change', (msg) => {
    console.log('Odds changed:', msg.event_id);
    console.log('XML:', msg.xml);
  });
</script>
```

### 数据库查询

```sql
-- 查看最新消息
SELECT * FROM uof_messages ORDER BY received_at DESC LIMIT 10;

-- 查看特定赛事
SELECT * FROM uof_messages WHERE event_id = 'sr:match:12345';

-- 统计消息类型
SELECT message_type, COUNT(*) FROM uof_messages GROUP BY message_type;

-- 查看跟踪的赛事
SELECT * FROM tracked_events ORDER BY last_message_at DESC;
```

## ❓ 遇到问题？

### 部署失败
- 检查 "Deployments" 标签的构建日志
- 确保所有环境变量都已设置

### 无法接收消息
- 检查 BETRADAR_ACCESS_TOKEN 是否正确
- 查看 "Logs" 标签的错误信息
- 确认AMQP连接成功

### 数据库连接失败
- 确保已添加 DATABASE_URL 引用
- 检查PostgreSQL服务是否运行（状态为Active）

### WebSocket连接失败
- 确保使用 wss:// 而不是 ws://
- 检查浏览器控制台的错误信息
- 确认域名已生成

## 💰 费用

Railway免费额度：
- **$5/月** 免费额度
- 足够运行小型项目

预计成本：
- 轻度使用：$0-5/月（免费额度内）
- 中度使用：$10-15/月
- 重度使用：$20-30/月

## 📚 更多资源

- **完整文档**: 查看 `README.md`
- **详细部署**: 查看 `RAILWAY-DEPLOYMENT.md`
- **Railway文档**: https://docs.railway.app
- **Betradar文档**: https://docs.betradar.com

---

**开始使用吧！** 🚀

