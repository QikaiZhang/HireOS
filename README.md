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
├── Agent/             # Python Agent（LangGraph 状态机）✅ Phase 1
│   ├── graph/         # 面试图定义
│   ├── nodes/         # 面试节点（提问、评估、总结）
│   ├── memory/        # 短期 + 长期记忆
│   ├── api/           # FastAPI 接口
│   └── main.py        # 入口
│
├── media-gateway/     # Go 媒体网关（音视频流接入与中继）✅ M1
│   ├── proto/         # media.proto（source of truth，make generate）
│   ├── internal/      # relay 中继核心 / server WS / rpc / registry / metrics
│   └── cmd/           # gateway 入口 / echoworker 回声 / wssmoke 验证
│
├── backend/           # Go 业务层（Week 2-4）
├── frontend/          # 前端页面（Week 4）
├── docs/              # 文档
└── docker-compose.yml # Week 4
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

### Agent Service（Phase 1 已可用）

```bash
cd Agent
uv sync                          # 安装依赖
uv run python main.py            # 启动 Agent（默认 :8000）

# 配置真实 LLM（可选）：复制 .env.example 为 .env 并填写 LLM_API_KEY
# 未配置 Key 时以 mock 模式运行，闭环可完整跑通
```

```bash
# 创建面试 → 返回第一题
curl -X POST localhost:8000/interview/start \
  -H 'Content-Type: application/json' \
  -d '{"room_id":"room-1","jd":"Go 后端","resume":"3 年 Go 经验"}'

# 提交回答 → 返回追问或下一题；面试结束后返回报告
curl -X POST localhost:8000/interview/chat \
  -H 'Content-Type: application/json' \
  -d '{"room_id":"room-1","message":"我叫张三，做 Go 后端…"}'

# 查询报告
curl localhost:8000/report/room-1
```

端到端验证：`cd Agent && uv run python scripts/verify_phase1_loop.py`

### Media Gateway（M1 已可用）

```bash
cd media-gateway
make generate    # proto → Go stub（需 protoc、protoc-gen-go、protoc-gen-go-grpc）
make build
./bin/gateway &                     # 网关：HTTP/WS :8080，gRPC :9090
./bin/echoworker -room demo-1 &     # 回声 worker（M1 mock）
open http://localhost:8080/echo     # 浏览器回声测试：说话 → 听到网关环回
./bin/wssmoke -room demo-1          # 自动化冒烟：25fps 灌帧，逐帧校验
```

### 全栈（Month 1 后期）

```bash
docker-compose up -d
open http://localhost:3000
```

---

## 文档索引

- [架构设计](docs/ARCHITECTURE.md) — 详细架构与设计决策
- [路线图](docs/ROADMAP.md) — 分阶段开发计划
- [Agent 设计](docs/agent/AGENT_DESIGN.md) — Python Agent 核心设计
- [Agent API](docs/agent/API.md) — Agent HTTP 接口契约（Go Backend 对接依据）
- [Phase 1 SPEC](docs/agent/PHASE1_SPEC.md) — Phase 1 闭环改造方案
- [Phase 1 报告](docs/agent/PHASE1_REPORT.md) — Phase 1 交付、验证与遗留项
- [媒体网关 SPEC](docs/gateway/GATEWAY_SPEC.md) — Go 实时音频网关设计（含性能收益评估）
- [音视频流基础](docs/gateway/MEDIA_BASICS.md) — PCM/帧/丢帧策略/RTMP 入门讲解
- [网关 M1 报告](docs/gateway/M1_REPORT.md) — 网关骨架交付与验证记录
- [阶段进度总览](docs/design/STAGES.md) — 现在到哪了、各 Stage 完成标准、最短路径
- [设计决策记录](docs/design/DESIGN_DECISIONS.md) — 10 条 ADR：选择、备选与代价
- [技术深挖](docs/design/TECH_DEEP_DIVE.md) — 水位丢帧实现、排障复盘、踩坑记录
- [阶段复盘](docs/design/PHASE_REVIEW.md) — 本阶段做了什么、设计理念、量化状态
- [面试宣讲包](docs/interview/interview-package.md) — 宣讲话术、STAR-L、分层追问
