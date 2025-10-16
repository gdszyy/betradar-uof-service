# Railway 详细部署指南

本指南将详细说明如何通过Railway Dashboard部署Betradar UOF Go服务。

## 前置准备

1. **Railway账号**
   - 访问 https://railway.app/
   - 使用GitHub账号登录（推荐）或邮箱注册
   - 新用户有$5免费额度/月

2. **GitHub仓库**（推荐）
   - 将项目代码推送到GitHub
   - 或者准备好项目文件夹用于直接上传

3. **Betradar访问令牌**
   - 准备好您的`BETRADAR_ACCESS_TOKEN`

## 部署步骤

### 第一步：创建新项目

1. 登录Railway后，点击 **"New Project"**

2. 选择部署方式：
   - **推荐方式A：从GitHub部署**
     - 点击 "Deploy from GitHub repo"
     - 授权Railway访问您的GitHub
     - 选择包含项目代码的仓库
   
   - **方式B：空项目（手动上传）**
     - 点击 "Empty Project"
     - 稍后通过Railway CLI上传代码

3. 项目创建成功后，进入项目面板

### 第二步：添加PostgreSQL数据库

1. 在项目面板中，点击 **"+ New"** 按钮

2. 选择 **"Database"** → **"Add PostgreSQL"**

3. Railway会自动创建PostgreSQL数据库实例
   - 数据库会自动启动
   - `DATABASE_URL` 环境变量会自动生成
   - 可以在数据库服务的 "Variables" 标签查看连接信息

4. 记下数据库信息（可选，Railway会自动配置）：
   ```
   PGHOST=xxxxx.railway.app
   PGPORT=5432
   PGUSER=postgres
   PGPASSWORD=xxxxx
   PGDATABASE=railway
   DATABASE_URL=postgresql://postgres:xxxxx@xxxxx.railway.app:5432/railway
   ```

### 第三步：配置Go服务

#### 3.1 如果从GitHub部署

1. Railway会自动检测到Dockerfile并开始构建

2. 等待首次构建完成（可能失败，因为还没配置环境变量）

#### 3.2 如果是空项目

1. 点击项目面板中的 **"+ New"** → **"Empty Service"**

2. 给服务命名，例如 "uof-service"

3. 使用Railway CLI上传代码：
   ```bash
   # 安装Railway CLI
   npm install -g @railway/cli
   
   # 登录
   railway login
   
   # 链接到项目
   railway link
   
   # 上传代码
   railway up
   ```

### 第四步：配置环境变量

1. 点击Go服务（不是数据库）

2. 切换到 **"Variables"** 标签

3. 点击 **"+ New Variable"** 添加以下环境变量：

   **必填变量：**
   
   | 变量名 | 值 | 说明 |
   |--------|-----|------|
   | `BETRADAR_ACCESS_TOKEN` | `your_token_here` | 您的Betradar访问令牌 |
   | `BETRADAR_MESSAGING_HOST` | `stgmq.betradar.com:5671` | AMQP服务器地址 |
   | `BETRADAR_API_BASE_URL` | `https://stgapi.betradar.com/v1` | API服务器地址 |
   | `ROUTING_KEYS` | `#` | 路由键（#表示所有消息） |

   **可选变量：**
   
   | 变量名 | 值 | 说明 |
   |--------|-----|------|
   | `ENVIRONMENT` | `production` | 环境标识 |
   | `PORT` | `8080` | HTTP端口（Railway会自动设置） |

4. **重要：连接数据库**
   
   Railway会自动生成`DATABASE_URL`，但需要确保Go服务能访问：
   
   - 点击 **"+ New Variable"** → **"Add Reference"**
   - 选择PostgreSQL数据库
   - 选择 `DATABASE_URL` 变量
   - 这样Go服务就能访问数据库了

5. 点击 **"Save"** 或直接关闭（自动保存）

### 第五步：配置服务设置

1. 在Go服务页面，切换到 **"Settings"** 标签

2. **部署设置：**
   - **Root Directory**: 如果代码在子目录，设置路径（通常留空）
   - **Build Command**: 留空（使用Dockerfile）
   - **Start Command**: 留空（使用Dockerfile的CMD）

3. **域名设置：**
   - 在 "Settings" 中找到 **"Networking"** 部分
   - 点击 **"Generate Domain"** 生成公开访问域名
   - 例如：`uof-service-production.up.railway.app`
   - 记下这个域名，稍后访问Web界面使用

4. **健康检查（可选）：**
   - 在 "Settings" → "Health Check" 中
   - 设置 Health Check Path: `/api/health`
   - 这样Railway会定期检查服务是否正常

### 第六步：触发部署

1. 如果环境变量修改后没有自动重新部署：
   - 切换到 **"Deployments"** 标签
   - 点击最新部署右侧的 **"⋮"** 菜单
   - 选择 **"Redeploy"**

2. 或者，如果是GitHub部署：
   - 推送新的commit到GitHub
   - Railway会自动检测并重新部署

3. 观察部署日志：
   - 在 "Deployments" 标签中点击最新部署
   - 查看构建和运行日志
   - 等待状态变为 **"Active"**（绿色）

### 第七步：验证部署

1. **检查服务状态：**
   ```
   状态应该显示为 "Active" 并有绿色指示灯
   ```

2. **查看日志：**
   - 切换到 **"Logs"** 标签
   - 应该看到类似输出：
   ```
   Starting Betradar UOF Service...
   Database connected and migrated
   AMQP consumer started
   Bookmaker ID: xxxxx
   Connected to AMQP server
   Bound to routing key: #
   Started consuming messages
   Web server started on port 8080
   Service is running. Press Ctrl+C to stop.
   ```

3. **测试API：**
   - 访问健康检查端点：
   ```
   https://your-service.up.railway.app/api/health
   ```
   - 应该返回：
   ```json
   {
     "status": "ok",
     "time": 1234567890
   }
   ```

4. **访问Web界面：**
   - 在浏览器中打开：
   ```
   https://your-service.up.railway.app/
   ```
   - 应该看到监控面板
   - 点击"连接"按钮，开始接收实时消息

### 第八步：监控和维护

#### 查看实时日志

1. 在Railway Dashboard中，进入Go服务

2. 切换到 **"Logs"** 标签

3. 实时查看服务日志：
   - AMQP连接状态
   - 接收到的消息
   - 数据库操作
   - 错误信息

#### 查看资源使用

1. 切换到 **"Metrics"** 标签

2. 查看：
   - CPU使用率
   - 内存使用
   - 网络流量
   - 请求数量

#### 数据库管理

1. 点击PostgreSQL数据库服务

2. 切换到 **"Data"** 标签

3. 可以：
   - 查看表结构
   - 执行SQL查询
   - 导出数据

4. 或者使用外部工具连接：
   ```bash
   # 使用psql
   psql $DATABASE_URL
   
   # 查看消息数量
   SELECT COUNT(*) FROM uof_messages;
   
   # 查看跟踪的赛事
   SELECT * FROM tracked_events ORDER BY last_message_at DESC LIMIT 10;
   ```

## 常见问题排查

### 问题1：部署失败 - "Build failed"

**可能原因：**
- Dockerfile语法错误
- go.mod依赖问题

**解决方法：**
1. 查看构建日志找到具体错误
2. 确保go.mod文件正确
3. 本地测试Docker构建：
   ```bash
   docker build -t uof-test .
   ```

### 问题2：服务启动失败 - "Application failed to respond"

**可能原因：**
- 环境变量配置错误
- 数据库连接失败
- AMQP连接失败

**解决方法：**
1. 检查 "Logs" 标签的错误信息
2. 验证所有环境变量是否正确设置
3. 确认DATABASE_URL已正确引用
4. 测试BETRADAR_ACCESS_TOKEN是否有效

### 问题3：无法连接数据库

**解决方法：**
1. 确保PostgreSQL服务正在运行（状态为Active）
2. 在Go服务的Variables中添加DATABASE_URL引用：
   - New Variable → Add Reference → 选择PostgreSQL → DATABASE_URL
3. 重新部署服务

### 问题4：AMQP连接失败

**检查项：**
1. BETRADAR_ACCESS_TOKEN是否正确
2. BETRADAR_MESSAGING_HOST是否为 `stgmq.betradar.com:5671`
3. 查看日志中的具体错误信息
4. 确认Railway服务器能访问外部AMQP服务器

### 问题5：WebSocket连接失败

**可能原因：**
- 域名未生成
- CORS配置问题

**解决方法：**
1. 确保已生成公开域名
2. 检查浏览器控制台的错误信息
3. 确认使用正确的WebSocket URL（wss://而不是ws://）

## 成本估算

Railway免费额度：
- **$5/月** 免费额度
- 包含：500小时运行时间
- 超出后按使用量计费

典型使用成本（估算）：
- **Go服务**: ~$5-10/月（持续运行）
- **PostgreSQL**: ~$5/月（500MB存储）
- **总计**: ~$10-15/月

节省成本建议：
1. 使用Railway的免费额度
2. 限制数据库存储（定期清理旧数据）
3. 优化消息过滤（减少存储量）

## 自动化部署

### 设置GitHub自动部署

1. 在Railway项目中，Go服务会自动监听GitHub仓库

2. 每次推送到main分支，Railway会自动：
   - 拉取最新代码
   - 构建Docker镜像
   - 部署新版本
   - 零停机更新

3. 配置部署分支：
   - Settings → Source → Branch: `main`

### 设置部署通知

1. 在项目Settings中配置Webhook

2. 可以接收部署状态通知：
   - Slack
   - Discord
   - 自定义Webhook

## 生产环境优化

### 1. 配置自定义域名

1. 在 Settings → Networking 中
2. 点击 "Custom Domain"
3. 添加您的域名（例如：uof.yourdomain.com）
4. 配置DNS CNAME记录指向Railway提供的域名

### 2. 配置环境变量管理

使用Railway的环境变量功能：
- 开发环境和生产环境分离
- 敏感信息加密存储
- 版本控制

### 3. 数据库备份

1. 在PostgreSQL服务中
2. Settings → Backups
3. 配置自动备份策略

### 4. 监控告警

1. 使用Railway的Metrics功能
2. 配置资源使用告警
3. 集成第三方监控工具（如Sentry）

## 下一步

部署成功后，您可以：

1. **访问Web监控面板**
   ```
   https://your-service.up.railway.app/
   ```

2. **使用REST API**
   ```bash
   # 获取消息
   curl https://your-service.up.railway.app/api/messages
   
   # 获取统计
   curl https://your-service.up.railway.app/api/stats
   ```

3. **集成到您的应用**
   ```javascript
   const client = new UOFClient({
     wsUrl: 'wss://your-service.up.railway.app/ws',
     apiUrl: 'https://your-service.up.railway.app/api'
   });
   ```

4. **自定义开发**
   - 修改消息处理逻辑
   - 添加新的API端点
   - 自定义前端界面
   - 集成其他服务

## 技术支持

如遇问题：
1. 查看Railway文档：https://docs.railway.app
2. Railway Discord社区：https://discord.gg/railway
3. 检查项目日志和错误信息
4. 参考本项目的README.md

---

**祝部署顺利！🚀**

