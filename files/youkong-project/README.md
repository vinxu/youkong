# 有空 - YouKong

> 一眼看到谁可能有空

## 项目简介

「有空」是一个帮助用户找到可能有空的朋友的社交应用。

**核心体验**：打开 APP → 看到朋友按"有空概率"排序的列表 → 点击进入聊天

```
┌─────────────────────────────────────────┐
│  🟢 李四                          92%  │
│     刷了40分钟手机，在家                  │
├─────────────────────────────────────────┤
│  🟢 王五                          78%  │
│     周五晚上，历史上他通常有空            │
├─────────────────────────────────────────┤
│  🟡 张三                          45%  │
│     数据不足                            │
└─────────────────────────────────────────┘
```

## 技术架构

**每人一个 AI Agent**：
- 每个用户有一个专属 Agent
- Agent 收集主人的屏幕使用数据、位置数据
- Agent 学习主人的行为规律
- 当用户打开 APP，自己的 Agent 向所有朋友的 Agent 请求数据
- 用 LLM 综合分析，计算每个朋友的有空概率

## 项目结构

```
youkong-project/
├── CLAUDE.md                        # AI 代码助手指南
├── docs/
│   ├── youkong-product-doc.md       # 产品需求文档
│   ├── youkong-agent-architecture.md # Agent 架构详细设计
│   └── youkong-dev-spec.md          # 开发技术规格
├── mobile/
│   ├── android/                     # Android 客户端
│   └── ios/                         # iOS 客户端
├── backend/
│   ├── api/                         # API 服务
│   ├── agent-hub/                   # Agent 通信中心
│   └── llm-service/                 # LLM 分析服务
└── shared/
    └── types/                       # 共享类型定义
```

## 核心权限

| 权限 | 用途 |
|------|------|
| 屏幕使用时间 | 判断用户是否在用手机、用什么类型APP |
| 地理位置 | 判断用户在哪里（家/公司/外面） |
| 通讯录 | 找到用户的朋友 |

## 开发文档

1. **快速开始**：阅读 `CLAUDE.md`
2. **产品需求**：阅读 `docs/youkong-product-doc.md`
3. **架构设计**：阅读 `docs/youkong-agent-architecture.md`
4. **技术规格**：阅读 `docs/youkong-dev-spec.md`（最重要）

## 使用 AI 辅助开发

本项目配置了 `CLAUDE.md` 文件，可以配合 Claude Code 或其他 AI 代码助手使用：

```bash
# 使用 Claude Code
claude code

# AI 会自动读取 CLAUDE.md 了解项目背景
```

## 技术栈

- **移动端**: React Native / Flutter
- **后端**: Node.js / Go
- **数据库**: PostgreSQL + Redis
- **LLM**: Claude API

## License

MIT
