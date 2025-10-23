# Release v1.0.5 - Documentation Fix: Producer Definitions

**发布日期**: 2025-10-23  
**类型**: 文档修正  
**标签**: v1.0.5

---

## 📝 Documentation Fix

### Changes

- **修正了 Producer 定义** 在 `SPORTRADAR_INTEGRATION_GUIDE.md` 中的错误
- 删除了错误的 `Ctrl (ID: 3)` producer 条目(之前被错误描述为"控制和管理")
- 明确了 **Ctrl** 是管理后台界面,而不是数据生产者
- UOF 的两个主要 producer 是:
  - **Pre-match Odds (ID: 3)**: 赛前赔率数据
  - **Live Odds (ID: 1)**: 比赛进行中的赔率数据

### Why This Matters

之前的文档错误地将 "Ctrl (ID: 3)" 列为一个独立的控制 producer,这可能会让开发者感到困惑。实际情况是:

- **Ctrl** 指的是 SportRadar 控制后台(用于管理预订和 token 的 Web 界面)
- **Producer ID 3** 是 Pre-match Odds producer,而不是 "Ctrl" producer

### Documentation

- 更新文件: `docs/SPORTRADAR_INTEGRATION_GUIDE.md`
- 文档版本: 1.0.5

### No Code Changes

此版本**仅包含文档修正**,服务代码无任何功能性变更。

---

## 如何在 GitHub 创建 Release

1. 访问仓库: https://github.com/extra-time-zone/betradar-uof-service
2. 点击 "Releases" 标签
3. 点击 "Draft a new release"
4. 选择标签: `v1.0.5`
5. 设置标题: `v1.0.5 - Documentation Fix: Producer Definitions`
6. 复制上述 release notes 到描述框
7. 点击 "Publish release"

---

## Git 信息

- **Commit**: 852c196
- **Tag**: v1.0.5
- **Branch**: main
- **Repository**: https://github.com/extra-time-zone/betradar-uof-service

---

## 完整变更日志

```
docs: fix Producer definition in SPORTRADAR_INTEGRATION_GUIDE

- Remove incorrect 'Ctrl (ID: 3)' producer definition
- Ctrl is actually the Pre-match Odds producer (ID: 3), not a separate control producer
- Update document version to 1.0.5
```

---

**注意**: 如果您有 GitHub Personal Access Token,可以使用以下命令自动创建 release:

```bash
export GH_TOKEN=your_github_token
gh release create v1.0.5 \
  --title "v1.0.5 - Documentation Fix: Producer Definitions" \
  --notes-file RELEASE_NOTES_v1.0.5.md
```

