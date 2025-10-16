#!/bin/bash

# Betradar UOF Service - GitHub发布脚本
# 使用方法: ./publish-to-github.sh <your-github-username>

set -e

# 检查参数
if [ -z "$1" ]; then
    echo "❌ 错误: 请提供GitHub用户名"
    echo "使用方法: ./publish-to-github.sh <your-github-username>"
    exit 1
fi

GITHUB_USERNAME=$1
REPO_NAME="betradar-uof-service"

echo "🚀 开始发布到GitHub..."
echo "📦 仓库: https://github.com/$GITHUB_USERNAME/$REPO_NAME"
echo ""

# 检查是否已安装git
if ! command -v git &> /dev/null; then
    echo "❌ 错误: 未安装git"
    echo "请先安装git: https://git-scm.com/downloads"
    exit 1
fi

# 检查是否已安装gh CLI
if ! command -v gh &> /dev/null; then
    echo "⚠️  警告: 未安装GitHub CLI (gh)"
    echo "将使用传统方式创建仓库"
    echo "建议安装gh CLI: https://cli.github.com/"
    echo ""
    USE_GH=false
else
    USE_GH=true
fi

# 初始化Git仓库
echo "📝 初始化Git仓库..."
if [ -d .git ]; then
    echo "⚠️  Git仓库已存在，跳过初始化"
else
    git init
    echo "✅ Git仓库初始化完成"
fi

# 添加所有文件
echo "📁 添加文件到Git..."
git add .

# 检查是否有更改
if git diff --cached --quiet; then
    echo "⚠️  没有需要提交的更改"
else
    # 提交
    echo "💾 提交更改..."
    git commit -m "Initial commit: Betradar UOF Service

Features:
- AMQP consumer for Betradar UOF messages
- PostgreSQL database storage
- WebSocket real-time streaming
- REST API for querying messages
- Web monitoring dashboard
- Railway deployment ready
- Complete documentation"
    echo "✅ 提交完成"
fi

# 设置主分支名称
echo "🌿 设置主分支..."
git branch -M main
echo "✅ 主分支设置为 main"

# 创建GitHub仓库
if [ "$USE_GH" = true ]; then
    echo ""
    echo "🔐 使用GitHub CLI创建仓库..."
    echo "请确保已登录GitHub CLI (运行 'gh auth login')"
    echo ""
    
    # 检查是否已登录
    if ! gh auth status &> /dev/null; then
        echo "❌ 错误: 未登录GitHub CLI"
        echo "请运行: gh auth login"
        exit 1
    fi
    
    # 创建仓库
    echo "📦 创建GitHub仓库..."
    gh repo create $REPO_NAME \
        --public \
        --description "Betradar Unified Odds Feed (UOF) service with AMQP consumer, PostgreSQL storage, and WebSocket streaming. Ready for Railway deployment." \
        --source=. \
        --remote=origin \
        --push
    
    echo "✅ 仓库创建并推送完成"
else
    echo ""
    echo "📦 请手动创建GitHub仓库..."
    echo ""
    echo "1. 访问: https://github.com/new"
    echo "2. 仓库名称: $REPO_NAME"
    echo "3. 描述: Betradar UOF service with AMQP, PostgreSQL, and WebSocket"
    echo "4. 选择 Public"
    echo "5. 不要初始化README、.gitignore或LICENSE (我们已经有了)"
    echo "6. 点击 'Create repository'"
    echo ""
    read -p "创建完成后按Enter继续..."
    
    # 添加远程仓库
    echo "🔗 添加远程仓库..."
    if git remote | grep -q origin; then
        git remote set-url origin https://github.com/$GITHUB_USERNAME/$REPO_NAME.git
    else
        git remote add origin https://github.com/$GITHUB_USERNAME/$REPO_NAME.git
    fi
    echo "✅ 远程仓库已添加"
    
    # 推送到GitHub
    echo "⬆️  推送到GitHub..."
    git push -u origin main
    echo "✅ 推送完成"
fi

echo ""
echo "🎉 发布成功！"
echo ""
echo "📍 仓库地址:"
echo "   https://github.com/$GITHUB_USERNAME/$REPO_NAME"
echo ""
echo "🚀 下一步 - 部署到Railway:"
echo "   1. 访问 https://railway.app/"
echo "   2. 点击 'New Project'"
echo "   3. 选择 'Deploy from GitHub repo'"
echo "   4. 选择 '$REPO_NAME' 仓库"
echo "   5. 添加PostgreSQL数据库"
echo "   6. 配置环境变量 (参考 RAILWAY-QUICKSTART.md)"
echo ""
echo "📚 文档:"
echo "   - 快速开始: RAILWAY-QUICKSTART.md"
echo "   - 详细部署: RAILWAY-DEPLOYMENT.md"
echo "   - 完整文档: README.md"
echo ""
echo "✨ 祝使用愉快！"

