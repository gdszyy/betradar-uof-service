# The Sports 认证方式修正说明

**版本**: v1.0.7  
**更新日期**: 2025-10-23  
**重要性**: 🔴 高 - 影响所有 The Sports API 调用

---

## 🔍 问题发现

根据 The Sports 官方文档,我们之前使用的认证方式是**错误的**。

### ❌ 之前的错误实现

#### HTTP API 认证
```go
// 错误: 使用 Bearer Token
req.Header.Set("Authorization", "Bearer "+c.apiToken)
```

#### 环境变量
```bash
THESPORTS_API_TOKEN=xxx      # ❌ 不需要
THESPORTS_USERNAME=xxx       # ✅ 需要
THESPORTS_SECRET=xxx         # ✅ 需要
```

---

## ✅ 正确的实现

### HTTP API 认证

根据官方文档:
> Each interface needs to pass the user name and key to verify the interface permissions

**正确方式**: 使用 URL 参数传递认证信息

```go
// 正确: 使用 URL 参数
params.Set("user", c.username)
params.Set("secret", c.secret)
```

**实际请求**:
```http
GET https://api.thesports.com/v1/football/match/list?user=your_username&secret=your_secret&date=2025-10-23
```

### MQTT 认证

MQTT 认证方式**保持不变**(之前就是正确的):

```go
opts.SetUsername(c.username)
opts.SetPassword(c.password)
```

---

## 📋 环境变量变更

### 之前 (错误)
```bash
THESPORTS_API_TOKEN=your_api_token          # ❌ 删除
THESPORTS_USERNAME=your_mqtt_username       # 只用于 MQTT
THESPORTS_SECRET=your_mqtt_secret           # 只用于 MQTT
```

### 现在 (正确)
```bash
THESPORTS_USERNAME=your_username            # ✅ 用于 HTTP API 和 MQTT
THESPORTS_SECRET=your_secret                # ✅ 用于 HTTP API 和 MQTT
```

---

## 🔧 代码变更

### 1. thesports/client.go

#### 修改前
```go
type Client struct {
    baseURL    string
    apiToken   string
    httpClient *http.Client
}

func NewClient(apiToken string) *Client {
    return NewClientWithConfig(Config{
        BaseURL:  DefaultBaseURL,
        APIToken: apiToken,
        Timeout:  DefaultTimeout,
    })
}

// 请求时使用 Bearer Token
req.Header.Set("Authorization", "Bearer "+c.apiToken)
```

#### 修改后
```go
type Client struct {
    baseURL    string
    username   string
    secret     string
    httpClient *http.Client
}

func NewClient(username, secret string) *Client {
    return NewClientWithConfig(Config{
        BaseURL:  DefaultBaseURL,
        Username: username,
        Secret:   secret,
        Timeout:  DefaultTimeout,
    })
}

// 请求时使用 URL 参数
params.Set("user", c.username)
params.Set("secret", c.secret)
```

### 2. config/config.go

#### 修改前
```go
type Config struct {
    // ...
    TheSportsAPIToken  string // The Sports API Token
    TheSportsUsername  string // The Sports MQTT Username
    TheSportsSecret    string // The Sports MQTT Secret
}

// 加载配置
TheSportsAPIToken: getEnv("THESPORTS_API_TOKEN", ""),
TheSportsUsername: getEnv("THESPORTS_USERNAME", ""),
TheSportsSecret:   getEnv("THESPORTS_SECRET", ""),
```

#### 修改后
```go
type Config struct {
    // ...
    TheSportsUsername  string // The Sports Username (for both HTTP API and MQTT)
    TheSportsSecret    string // The Sports Secret (for both HTTP API and MQTT)
}

// 加载配置
TheSportsUsername: getEnv("THESPORTS_USERNAME", ""),
TheSportsSecret:   getEnv("THESPORTS_SECRET", ""),
```

### 3. services/thesports_client.go

#### 修改前
```go
c.restClient = thesports.NewClient(c.config.TheSportsAPIToken)
```

#### 修改后
```go
c.restClient = thesports.NewClient(c.config.TheSportsUsername, c.config.TheSportsSecret)
```

### 4. .env.example

#### 修改前
```bash
# The Sports API 配置
THESPORTS_API_TOKEN=your_thesports_api_token        # The Sports API Token
THESPORTS_USERNAME=your_thesports_mqtt_username     # The Sports MQTT Username
THESPORTS_SECRET=your_thesports_mqtt_secret         # The Sports MQTT Secret
```

#### 修改后
```bash
# The Sports API 配置
THESPORTS_USERNAME=your_thesports_username          # The Sports Username (for both HTTP API and MQTT)
THESPORTS_SECRET=your_thesports_secret              # The Sports Secret (for both HTTP API and MQTT)
```

---

## 🚀 升级指南

### Railway 用户

#### 1. 删除旧的环境变量
```bash
railway variables delete THESPORTS_API_TOKEN
```

#### 2. 确认现有变量
```bash
railway variables | grep THESPORTS
```

应该只看到:
```
THESPORTS_USERNAME=xxx
THESPORTS_SECRET=xxx
```

#### 3. 重新部署
```bash
railway up
```

或者等待 GitHub 自动部署。

### 本地开发

#### 1. 更新 .env 文件
```bash
# 删除这一行
# THESPORTS_API_TOKEN=xxx

# 保留这两行
THESPORTS_USERNAME=your_username
THESPORTS_SECRET=your_secret
```

#### 2. 重启服务
```bash
go run main.go
```

---

## 📊 官方文档参考

### HTTP API 认证

官方文档说明:
```
http protocol
The domain name is: api.thesports.com
Support http and https
Each interface needs to pass the user name and key to verify the interface permissions, whitelist ip to get data

Global request parameters
parameter   description
user        Username, please contact our business staff
secret      User key, please contact our business staff
```

### MQTT 认证

官方文档说明:
```
websocket protocol
The domain name is: mq.thesports.com
Realized by mqtt's websocket protocol
The user name, key and whitelist IP must be correct to subscribe to data, otherwise authorization will fail

Sample code:
client.username_pw_set(username=username, password=password)
```

---

## ⚠️ 重要提示

### 1. IP 白名单

官方文档提到:
> whitelist ip to get data

**需要将服务器 IP 加入白名单**,否则即使认证正确也无法获取数据。

**Railway 用户**: 联系 The Sports 技术支持,提供 Railway 的出口 IP。

### 2. 凭证获取

需要联系 The Sports 商务人员获取:
- Username (用户名)
- Secret (密钥)

### 3. 测试连接

修改后测试 API 调用:
```bash
# 测试 HTTP API
curl "https://api.thesports.com/v1/football/match/list?user=YOUR_USERNAME&secret=YOUR_SECRET&date=2025-10-23"

# 应该返回比赛列表,而不是认证错误
```

---

## 🔍 故障排查

### 问题 1: 认证失败

**错误信息**:
```json
{
  "code": 401,
  "message": "Authentication failed"
}
```

**解决方案**:
1. 检查 `THESPORTS_USERNAME` 和 `THESPORTS_SECRET` 是否正确
2. 确认 IP 是否在白名单中
3. 联系 The Sports 技术支持

### 问题 2: 无法获取数据

**错误信息**:
```json
{
  "code": 403,
  "message": "Access denied"
}
```

**解决方案**:
1. 确认 IP 白名单配置
2. 检查账户权限
3. 联系 The Sports 技术支持

### 问题 3: MQTT 连接失败

**错误信息**:
```
Connection refused - bad username or password
```

**解决方案**:
1. 检查 `THESPORTS_USERNAME` 和 `THESPORTS_SECRET`
2. 确认 MQTT 服务器地址: `ssl://mq.thesports.com:443`
3. 检查 IP 白名单

---

## 📈 验证修改

### 1. 查看日志

```bash
railway logs | grep "\[TheSports\]"
```

**成功的日志**:
```
[TheSports] 🔌 Connecting to The Sports MQTT...
[TheSports] ✅ Successfully subscribed to football/live/#
[TheSports] ✅ Connected to The Sports MQTT successfully
```

### 2. 测试 REST API

```bash
curl https://your-app.railway.app/api/thesports/today
```

**成功的响应**:
```json
{
  "status": "success",
  "count": 10,
  "matches": [...]
}
```

### 3. 测试 MQTT 连接

```bash
curl https://your-app.railway.app/api/thesports/status
```

**成功的响应**:
```json
{
  "connected": true,
  "subscriptions": {
    "football": "subscribed",
    "basketball": "subscribed",
    "esports": "subscribed"
  }
}
```

---

## 📚 相关文档

- [The Sports 官方文档](官方文档链接)
- [MQTT 监控指南](THESPORTS_MQTT_MONITORING.md)
- [Release Notes v1.0.7](../RELEASE_NOTES_v1.0.7.md)

---

## 🎯 总结

### 关键变更
1. ✅ HTTP API 改用 URL 参数认证
2. ✅ 移除 `THESPORTS_API_TOKEN` 环境变量
3. ✅ 统一使用 `THESPORTS_USERNAME` 和 `THESPORTS_SECRET`
4. ✅ MQTT 认证保持不变

### 影响范围
- ✅ 所有 HTTP API 调用
- ❌ MQTT 连接不受影响
- ✅ 需要更新环境变量配置

### 兼容性
- ❌ 不向后兼容 (必须更新环境变量)
- ✅ 代码自动处理新的认证方式

---

**文档版本**: v1.0.7  
**最后更新**: 2025-10-23

