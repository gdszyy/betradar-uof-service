# GitHub 发布指南

本指南说明如何将项目发布到GitHub。

## 方法1: 使用自动化脚本（推荐）

### 前置要求

1. **安装Git**
   ```bash
   # macOS
   brew install git
   
   # Ubuntu/Debian
   sudo apt install git
   
   # Windows
   # 下载安装: https://git-scm.com/downloads
   ```

2. **安装GitHub CLI（推荐）**
   ```bash
   # macOS
   brew install gh
   
   # Ubuntu/Debian
   sudo apt install gh
   
   # Windows
   # 下载安装: https://cli.github.com/
   ```

3. **登录GitHub CLI**
   ```bash
   gh auth login
   ```
   按提示选择：
   - GitHub.com
   - HTTPS
   - Login with a web browser

### 执行发布

```bash
# 进入项目目录
cd uof-go-service

# 运行发布脚本
./publish-to-github.sh your-github-username

# 例如:
./publish-to-github.sh john-doe
```

脚本会自动：
- ✅ 初始化Git仓库
- ✅ 添加所有文件
- ✅ 创建初始提交
- ✅ 在GitHub创建仓库
- ✅ 推送代码到GitHub

完成后，访问：
```
https://github.com/your-username/betradar-uof-service
```

## 方法2: 手动发布

### 步骤1: 初始化Git仓库

```bash
cd uof-go-service
git init
git add .
git commit -m "Initial commit"
git branch -M main
```

### 步骤2: 在GitHub创建仓库

1. 访问 https://github.com/new

2. 填写信息：
   - **Repository name**: `betradar-uof-service`
   - **Description**: `Betradar UOF service with AMQP, PostgreSQL, and WebSocket`
   - **Visibility**: Public
   - ❌ 不要勾选 "Add a README file"
   - ❌ 不要勾选 ".gitignore"
   - ❌ 不要勾选 "Choose a license"

3. 点击 **"Create repository"**

### 步骤3: 推送代码

```bash
# 添加远程仓库
git remote add origin https://github.com/your-username/betradar-uof-service.git

# 推送代码
git push -u origin main
```

### 步骤4: 验证

访问您的仓库：
```
https://github.com/your-username/betradar-uof-service
```

应该看到所有文件已上传。

## 方法3: 使用GitHub Desktop

### 步骤1: 安装GitHub Desktop

下载并安装: https://desktop.github.com/

### 步骤2: 登录GitHub账号

打开GitHub Desktop，点击 "Sign in to GitHub.com"

### 步骤3: 添加本地仓库

1. File → Add Local Repository
2. 选择 `uof-go-service` 文件夹
3. 如果提示 "not a git repository"，点击 "create a repository"

### 步骤4: 创建初始提交

1. 在左侧看到所有文件
2. 在底部输入提交信息: "Initial commit"
3. 点击 "Commit to main"

### 步骤5: 发布到GitHub

1. 点击顶部的 "Publish repository"
2. 填写信息：
   - Name: `betradar-uof-service`
   - Description: `Betradar UOF service`
   - ✅ Keep this code private (取消勾选，使其公开)
3. 点击 "Publish Repository"

完成！

## 配置仓库

### 添加仓库描述

1. 访问仓库页面
2. 点击右上角的 ⚙️ Settings
3. 在 "About" 部分添加：
   - **Description**: `Betradar Unified Odds Feed service with AMQP consumer, PostgreSQL storage, WebSocket streaming, and Railway deployment support`
   - **Website**: 您的部署域名（可选）
   - **Topics**: `betradar`, `uof`, `odds`, `betting`, `golang`, `websocket`, `postgresql`, `railway`

### 添加README徽章

编辑 `README.md`，在顶部添加：

```markdown
# Betradar UOF Go Service

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/your-template-id)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21-blue.svg)](https://golang.org/)
```

### 设置GitHub Pages（可选）

如果想托管文档：

1. Settings → Pages
2. Source: Deploy from a branch
3. Branch: main
4. Folder: /docs
5. Save

## 连接到Railway

发布到GitHub后，在Railway部署：

### 快速部署

1. 访问 https://railway.app/
2. 点击 "New Project"
3. 选择 "Deploy from GitHub repo"
4. 选择 `betradar-uof-service` 仓库
5. Railway会自动检测Dockerfile并开始构建

### 配置环境变量

在Railway项目中添加：
```
BETRADAR_ACCESS_TOKEN=your_token
BETRADAR_MESSAGING_HOST=stgmq.betradar.com:5671
BETRADAR_API_BASE_URL=https://stgapi.betradar.com/v1
ROUTING_KEYS=#
```

详细步骤参考 `RAILWAY-QUICKSTART.md`

## 更新代码

### 推送更新

```bash
# 修改代码后
git add .
git commit -m "Update: description of changes"
git push
```

Railway会自动检测推送并重新部署。

### 创建发布版本

```bash
# 创建标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

在GitHub上：
1. 进入仓库
2. 点击 "Releases"
3. 点击 "Create a new release"
4. 选择标签 v1.0.0
5. 填写发布说明
6. 点击 "Publish release"

## 协作开发

### 克隆仓库

其他开发者可以克隆仓库：

```bash
git clone https://github.com/your-username/betradar-uof-service.git
cd betradar-uof-service
```

### 创建分支

```bash
# 创建功能分支
git checkout -b feature/new-feature

# 修改代码
# ...

# 提交更改
git add .
git commit -m "Add new feature"

# 推送分支
git push origin feature/new-feature
```

### 创建Pull Request

1. 访问GitHub仓库
2. 点击 "Pull requests"
3. 点击 "New pull request"
4. 选择分支
5. 填写PR描述
6. 点击 "Create pull request"

## 常见问题

### 问题1: 推送被拒绝

```bash
# 错误: Updates were rejected because the remote contains work
```

解决：
```bash
git pull origin main --rebase
git push origin main
```

### 问题2: 认证失败

从2021年8月起，GitHub不再支持密码认证。

解决方案：

**方案A: 使用Personal Access Token**

1. GitHub → Settings → Developer settings → Personal access tokens
2. Generate new token (classic)
3. 勾选 `repo` 权限
4. 生成并复制token
5. 推送时使用token作为密码

**方案B: 使用SSH**

```bash
# 生成SSH密钥
ssh-keygen -t ed25519 -C "your_email@example.com"

# 添加到ssh-agent
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519

# 复制公钥
cat ~/.ssh/id_ed25519.pub

# 添加到GitHub: Settings → SSH and GPG keys → New SSH key
```

修改远程URL：
```bash
git remote set-url origin git@github.com:your-username/betradar-uof-service.git
```

### 问题3: 文件太大

GitHub限制单个文件最大100MB。

如果有大文件：
```bash
# 使用Git LFS
git lfs install
git lfs track "*.bin"
git add .gitattributes
git commit -m "Add Git LFS"
```

### 问题4: 忘记添加.gitignore

如果已经提交了不该提交的文件：

```bash
# 从Git中移除但保留本地文件
git rm --cached .env
git rm --cached -r vendor/

# 提交更改
git commit -m "Remove ignored files"
git push
```

## 最佳实践

1. **经常提交** - 小而频繁的提交比大的提交好
2. **写好提交信息** - 清晰描述更改内容
3. **使用分支** - 为新功能创建分支
4. **代码审查** - 使用Pull Request进行代码审查
5. **保护主分支** - 在Settings中设置分支保护规则
6. **使用标签** - 为重要版本打标签
7. **写好文档** - 保持README和文档更新

## 下一步

✅ 代码已发布到GitHub
✅ 准备部署到Railway

查看：
- **Railway快速部署**: `RAILWAY-QUICKSTART.md`
- **详细部署指南**: `RAILWAY-DEPLOYMENT.md`
- **完整文档**: `README.md`

---

**祝发布顺利！** 🚀

