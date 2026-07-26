# HireOS

> AI 语音面试 Agent 平台 — 让候选人打开网页，与 AI 面试官进行一场真实的语音对话，最后生成面试报告。

HireOS 不是一个聊天机器人。它是一个**有状态的面试 Agent 系统**：输入 → 状态 → 推理 → 输出 → 持久化。

---

## 架构概览

```
浏览器（麦克风 + 摄像头）
       │
       ▼
  Go Backend（业务控制层）
       │
  gRPC Streaming
       │
       ▼
  Python Agent Service（LangGraph 状态机）
       │
       ▼
     LLM + TTS
       │
       ▼
  浏览器播放
```

**存储**：MySQL 保存面试记录 · Redis 保存 Session 状态

---

## 技术栈

| 层 | 技术 | 说明 |
|---|---|---|
| 前端 | HTML/JS | 面试页、报告页，MediaRecorder 采集语音 |
| 业务层 | Go | HTTP API、Session 管理、gRPC Client、WebSocket |
| Agent 层 | Python + LangGraph | 面试状态机、流程控制、报告生成 |
| 通信 | gRPC Streaming | Go ↔ Python 双向流 |
| LLM | OpenAI 兼容 API | 问题生成、回答分析、追问决策 |
| 语音 | ASR / TTS | 语音转文字 / 文字转语音 |
| 缓存 | Redis | Session 状态、对话上下文 |
| 数据库 | MySQL | 面试记录、候选人信息、报告 |
| 部署 | Docker Compose | 一键启动全栈 |

---

## 项目结构

```
hireos
├── agent-service/     # Python Agent（LangGraph 状态机）
│   ├── app/
│   ├── graph/         # 面试图定义
│   ├── nodes/         # 面试节点（提问、评估、总结）
│   ├── memory/        # 短期 + 长期记忆
│   ├── prompt/        # Prompt 模板
│   └── main.py        # FastAPI 入口
│
├── backend/           # Go 业务层
│   ├── api/           # HTTP API
│   ├── service/       # 业务逻辑
│   ├── grpc/          # gRPC Client
│   ├── session/       # Session 管理
│   └── repository/    # 数据访问
│
├── frontend/          # 前端页面
│
├── proto/             # Protobuf 定义
│
├── docs/              # 文档
│   ├── ARCHITECTURE.md
│   ├── ROADMAP.md
│   └── agent/
│
└── docker-compose.yml
```

---

## 核心设计思想

### 垂直切片优先

先切一条完整链路跑通，每层可以简单，但必须通：

```
用户输入 → 系统处理 → 业务逻辑 → 用户输出
```

### Agent 优先，流媒体后置

第一个月交付可用的语音 AI 面试。WebSocket 传音频就够了。RTMP 网关是第二个月的增强项——Agent 完全不用改。

### 变化隔离

```
V1:  Browser → WebSocket → Go → gRPC → Agent
V2:  Browser → RTMP → Go → gRPC → Agent     ← Agent 不变
```

好的架构让替换底层组件不影响业务逻辑。

---

## 快速开始

```bash
# 启动全部服务
docker-compose up -d

# 打开浏览器
open http://localhost:3000
```

---

## 文档索引

- [架构设计](docs/ARCHITECTURE.md) — 详细架构与设计决策
- [路线图](docs/ROADMAP.md) — 分阶段开发计划
- [Agent 设计](docs/agent/AGENT_DESIGN.md) — Python Agent 核心设计
