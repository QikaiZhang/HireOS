# 架构设计

## 系统全景

```
┌────────────────────────────────────────────────────────────┐
│                        浏览器                                │
│   MediaRecorder (音频采集)  │  Audio API (TTS 播放)          │
└──────────────┬─────────────────────────────────────────────┘
               │  WebSocket (音频流)
               ▼
┌────────────────────────────────────────────────────────────┐
│                     Go Backend                              │
│                                                             │
│  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐    │
│  │ HTTP    │  │ Session  │  │ gRPC    │  │ WebSocket│    │
│  │ API     │  │ Manager  │  │ Client  │  │ Server   │    │
│  └────┬────┘  └────┬─────┘  └────┬────┘  └──────────┘    │
│       │            │             │                          │
│       │       ┌────┴─────┐       │                          │
│       │       │  Redis   │       │                          │
│       │       └──────────┘       │                          │
└───────┼──────────────────────────┼──────────────────────────┘
        │                          │
        │                     gRPC Streaming
        │                          │
        ▼                          ▼
┌────────────────────────────────────────────────────────────┐
│                  Python Agent Service                       │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              LangGraph State Machine                   │  │
│  │                                                        │  │
│  │   INIT → INTRO → TECH → FOLLOW_UP → SUMMARY           │  │
│  │     │        │       │         │           │           │  │
│  │     ▼        ▼       ▼         ▼           ▼           │  │
│  │  Load     Generate Analyze  Decide     Generate        │  │
│  │  Resume   Question Answer   Follow-up  Report          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐                 │
│  │ Memory  │  │ Prompts  │  │   LLM    │                 │
│  └─────────┘  └──────────┘  └──────────┘                 │
└────────────────────────────────────────────────────────────┘
```

---

## 分层职责

### 前端层

**职责**：用户交互，音频采集与播放

- 面试创建页面
- 面试进行页（摄像头 + 聊天界面）
- 面试报告页
- 使用 `MediaRecorder API` 采集麦克风音频
- 通过 WebSocket 发送音频数据
- 接收 TTS 音频并播放

**不含**：业务逻辑、AI 推理

### Go Backend（业务控制层）

> 2026-09 更新：Go 层按媒体面/控制面拆分为两个服务——**media-gateway**（媒体接入与中继）与
> **backend**（业务编排）。拆分决策与详细设计见 [媒体网关 SPEC](gateway/GATEWAY_SPEC.md)。

**职责**：业务编排，不包含 AI 推理

```
backend/
├── api/          # HTTP API（创建面试、查询报告）
├── service/      # 业务逻辑（面试流程编排）
├── grpc/         # gRPC Streaming Client（连接 Agent）
├── session/      # Redis Session 管理
└── repository/   # MySQL 数据访问
```

### 媒体网关层（media-gateway，M1 已落地）

**职责**：音视频流接入与中继，**不解码、不转码、不懂面试语义、不落盘**

```
media-gateway/
├── proto/          # media.proto（source of truth，protoc 生成）
├── internal/
│   ├── relay/      # 会话中继核心：有界缓冲、水位丢帧、seq（纯逻辑）
│   ├── server/     # 浏览器 WS 接入：读写泵、心跳、限额
│   ├── rpc/        # MediaRelay gRPC：上行音频泵 + 下行 TTS 路由
│   ├── registry/   # 房间 → 会话路由（M1 进程内，M2 Redis）
│   └── metrics/    # Prometheus 指标
└── cmd/            # gateway 入口 / echoworker 回声 / wssmoke 验证
```

**Go（网关）不负责**：ASR/TTS、面试状态机、业务语义、持久化
**Go（网关）负责**：大规模长连接承载（数千路/节点）、实时帧中继（水位丢帧、读泵永不阻塞）、传输协议隔离（WS → RTMP 只动本层）

核心结构：

```go
type InterviewSession struct {
    ID            string
    CandidateID   string
    Stage         string   // INIT | INTRO | TECH | FOLLOW_UP | SUMMARY
    AgentSession  string   // 对应 Agent 侧的会话 ID
}
```

**Go 不负责**：
- 不调用 LLM
- 不做 ASR/TTS（调用外部服务）
- 不管理面试状态机（由 Agent 管理）

**Go 负责**：
- 用户请求路由
- Session 生命周期管理
- 连接 Agent 并转发数据
- 数据持久化

### Python Agent Service（AI 推理层）

**职责**：面试状态机、LLM 调用、报告生成

详见 [Agent 设计文档](agent/AGENT_DESIGN.md)

```
agent-service/
├── app/
├── graph/          # LangGraph 状态图定义
│   ├── interview_graph.py
│   └── state.py
├── nodes/          # 图节点
│   ├── interviewer.py   # 提问节点
│   ├── evaluator.py     # 回答评估节点
│   └── summarizer.py    # 报告生成节点
├── memory/         # 记忆管理
├── prompt/         # Prompt 模板
└── main.py         # FastAPI + gRPC Server
```

**Agent 不负责**：
- 音频处理
- 用户认证
- Session 管理

**Agent 负责**：
- 根据简历和 JD 生成问题
- 分析候选人回答质量
- 决定是否追问
- 生成面试报告和评分

---

## 通信协议

### Go ↔ Python: gRPC Streaming

```
Go (Client)                        Python (Server)
    │                                    │
    │ ── UserMessage(text) ────────────► │
    │                                    │  Agent 处理
    │ ◄── AgentResponse(text, stage) ── │
    │                                    │
    │ ── UserMessage(text) ────────────► │
    │ ◄── AgentResponse(text, stage) ── │
    │                                    │
    │ ◄── AgentResponse(report) ──────── │  (面试结束)
```

Go 通过双向流将用户回答传给 Agent，Agent 流式返回 AI 问题和最终报告。

### 浏览器 ↔ Go: WebSocket + HTTP

- **WebSocket**：音频数据流（实时）
- **HTTP**：创建面试、查询报告（非实时）

---

## 数据存储

### MySQL（持久化）

```sql
-- 面试记录
interview_session (
    id,
    candidate_id,
    start_time,
    end_time,
    score,
    report TEXT
)

-- 对话记录
interview_message (
    id,
    session_id,
    role,          -- interviewer | candidate
    content,
    timestamp
)
```

### Redis（运行时状态）

```
session:{id} → {
    "stage": "TECH",
    "conversation_id": "conv_xxx",
    "asked_questions": [...],
    "current_topic": "Redis"
}
```

---

## 设计决策

### 为什么 gRPC Streaming 而不是 HTTP 轮询？

Agent 的推理是流式的——LLM 生成 token 时就应该逐步返回，而不是等全部生成完。gRPC Streaming 天然支持双向流：

- Go 发送用户回答
- Agent 流式返回 AI 生成内容
- 面试结束时 Agent 推送报告

HTTP 轮询意味着延迟、不必要的连接开销，而且无法表达"面试进行中"这个语义。

### 为什么 Go 和 Python 分离？

- **Go** 擅长并发、网络 IO、低延迟——适合做网关和业务编排
- **Python** 有最丰富的 AI 生态——LangGraph、LLM SDK、Prompt 管理

两者通过 gRPC 解耦，各自做擅长的事。

### 为什么不用 WebRTC 直传服务端？

WebRTC 适合实时互动，但服务端媒体处理链路复杂（SFU、TURN、SDP 协商）。当前方案将 WebRTC 作为客户端低延迟体验层（后续可选），同时设计独立媒体接入网关负责协议解析、流复制和 AI 消费，使媒体处理和业务 Agent 解耦。

### 为什么第一个月不用 RTMP？

RTMP 协议解析是纯工程深度——不影响产品是否成立。先用 WebSocket 传音频，证明"AI 面试"这个产品成立，第二个月再替换传输层。Agent 完全不用改。

---

## 演进路线

```
V1.0 (Month 1): WebSocket Audio → Go → gRPC → Agent
                                   ↓ 升级传输层
V2.0 (Month 2): RTMP Audio → Go → gRPC → Agent  (Agent 不变)
```

详见 [路线图](ROADMAP.md)
