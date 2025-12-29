# 部署状态和说明

## 当前状态

### ✅ 代码已完成

所有Replay API功能的代码已经完成并推送到GitHub:
- ✅ 4个Replay API端点
- ✅ ReplayClient集成
- ✅ 完整的文档
- ✅ 测试脚本

**最新Commit**: `6e6d3cf` - "Fix: Add Username and Password fields to Config struct"

### ⏳ Railway部署状态

**问题**: Railway似乎没有自动部署最新的代码

**证据**:
```bash
# Replay端点返回404(应该返回503或XML)
curl https://betradar-uof-service-copy-production.up.railway.app/api/replay/status
# 返回: 404 page not found
```

---

## 手动触发Railway部署

### 方法1: 通过Railway Dashboard(推荐)

1. 打开 https://railway.app/
2. 选择您的项目
3. 点击您的服务
4. 点击 **Deployments** 标签
5. 点击右上角的 **Deploy** 按钮
6. 选择 **Deploy Latest Commit**

### 方法2: 通过Git推送

```bash
# 创建一个空commit强制触发部署
cd /home/ubuntu/uof-go-service
git commit --allow-empty -m "Trigger Railway deployment"
git push origin main
```

### 方法3: 通过Railway CLI

```bash
# 安装Railway CLI
npm install -g @railway/cli

# 登录
railway login

# 链接项目
railway link

# 触发部署
railway up
```

---

## 验证部署成功

### 1. 检查Replay端点是否可用

```bash
curl https://betradar-uof-service-copy-production.up.railway.app/api/replay/status
```

**预期结果**:
- ✅ **503** + "Replay client not configured" (如果没设置环境变量)
- ✅ **XML响应** (如果已设置环境变量)
- ❌ **404** (说明还是旧代码)

### 2. 检查编译日志

在Railway Dashboard → Deployments → Build Logs中查看:

**应该看到**:
```
Building with Dockerfile...
Successfully built...
```

**不应该看到**:
```
compilation error
undefined: Username
undefined: Password
```

### 3. 测试API

```bash
# 使用测试脚本
cd /home/ubuntu/uof-go-service/tools
./test_replay_api.sh https://betradar-uof-service-copy-production.up.railway.app
```

---

## 环境变量配置

### 必需的环境变量

在Railway Dashboard → Variables中设置:

```
UOF_USERNAME=your_betradar_username
UOF_PASSWORD=your_betradar_password
```

### 可选的环境变量

```
# 如果需要测试Replay(连接到Replay服务器)
AMQP_HOST=global.replaymq.betradar.com

# 标记为Replay模式
REPLAY_MODE=true
```

---

## 部署后测试

### 快速测试

```bash
# 1. 检查端点可用性
curl https://betradar-uof-service-copy-production.up.railway.app/api/replay/status

# 2. 启动重放测试
curl -X POST https://betradar-uof-service-copy-production.up.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 50,
    "duration": 45
  }'

# 3. 查看统计
curl https://betradar-uof-service-copy-production.up.railway.app/api/stats
```

### 完整测试

```bash
cd /home/ubuntu/uof-go-service/tools
./test_replay_api.sh https://betradar-uof-service-copy-production.up.railway.app
```

---

## 故障排查

### 问题1: 404 Not Found

**原因**: 旧代码还在运行

**解决**:
1. 检查Railway Deployments,确认最新commit已部署
2. 手动触发部署(参考上面的方法)
3. 检查构建日志是否有错误

### 问题2: 503 Service Unavailable

**原因**: 环境变量未设置

**解决**:
1. 在Railway中设置 `UOF_USERNAME` 和 `UOF_PASSWORD`
2. 重新部署服务

### 问题3: 编译错误

**原因**: 代码有问题

**解决**:
1. 本地测试编译:
   ```bash
   cd /home/ubuntu/uof-go-service
   go build -o /tmp/test main.go
   ```
2. 如果本地编译成功,检查Railway的Go版本是否匹配

### 问题4: 部署很慢

**原因**: Railway可能在排队或构建缓存问题

**解决**:
1. 等待5-10分钟
2. 检查Railway状态页面
3. 尝试清除构建缓存(在Railway设置中)

---

## 本地测试

如果Railway部署有问题,可以先在本地测试:

```bash
# 1. 设置环境变量
export UOF_USERNAME="your_username"
export UOF_PASSWORD="your_password"
export DATABASE_URL="your_database_url"
export PORT=8080

# 2. 编译并运行
cd /home/ubuntu/uof-go-service
go build -o uof-service main.go
./uof-service

# 3. 在另一个终端测试
curl -X POST http://localhost:8080/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":50,"duration":45}'
```

---

## 代码验证

### 确认代码已推送

```bash
cd /home/ubuntu/uof-go-service
git log --oneline -5
```

**应该看到**:
```
6e6d3cf Fix: Add Username and Password fields to Config struct
1546839 Docs: Add Replay API quickstart guide and test script
f63c773 Feature: Add Replay API endpoints
...
```

### 确认代码可编译

```bash
cd /home/ubuntu/uof-go-service
go build -o /tmp/test main.go && echo "✅ Compilation successful"
```

---

## 下一步

1. **手动触发Railway部署** (通过Dashboard或空commit)
2. **等待3-5分钟** 让部署完成
3. **验证端点** 使用上面的测试命令
4. **设置环境变量** (如果还没设置)
5. **运行完整测试** 使用test_replay_api.sh

---

## 联系信息

如果部署持续失败:
1. 检查Railway的构建日志
2. 查看Runtime日志是否有错误
3. 确认Dockerfile配置正确
4. 检查go.mod和go.sum是否有问题

---

## 总结

### ✅ 已完成
- 代码开发和测试
- 本地编译成功
- 推送到GitHub
- 完整文档

### ⏳ 待完成
- Railway自动部署(或手动触发)
- 设置环境变量
- 运行测试验证

### 🎯 目标
- Replay API端点可用
- 可以通过HTTP触发重放测试
- 验证数据管道功能

---

**当前时间**: 2025-10-21 02:04 UTC

**最新Commit**: 6e6d3cf

**Railway服务**: betradar-uof-service-copy-production.up.railway.app

**GitHub仓库**: https://github.com/gdszyy/betradar-uof-service

