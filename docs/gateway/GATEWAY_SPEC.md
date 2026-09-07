# SPEC: Go Media Gateway — 实时音频网关（视频预留）

> 状态：**设计已确认，可实施**（2026-09-07）｜ D1 服务拆分经用户确认：项目目标为企业级高可用，接受架构复杂度，拒绝 toy。
> 前置：[Phase 1 报告](../agent/PHASE1_REPORT.md)（文字闭环已通）｜ 架构依据：[ARCHITECTURE.md](../ARCHITECTURE.md)

## Background

Phase 1 已打通文字面试闭环（HTTP → LangGraph → LLM → SQLite）。下一步进入语音链路：

```
浏览器（麦克风） → 【媒体网关】 → ASR → Agent → TTS → 【媒体网关】 → 浏览器播放
```

本 SPEC 定义其中的**媒体网关层**——即架构文档中"独立媒体接入网关"的落地：

- **本期只做音频**；帧模型预留 codec/媒体类型字段，Month 2 接视频（H264）与 RTMP 时网关契约不变
- 按**企业级标准**设计：明确的性能验收门（SLO）、背压/丢帧策略、高可用部署、安全、可观测性、混沌验证，全部可压测与演练

## 性能收益评估（决策前提，已确认）

### 网关不改善什么

单轮对话 E2E 延迟（约 2~4.5s）由 LLM 主导（1~3s），ASR/TTS 各占数百毫秒，网关转发开销 < 1ms（< 0.1%）。网关的定位不是"让面试变快"。

### 网关真正带来什么

| 收益 | 机制 | 量化依据 |
|------|------|----------|
| 连接承载剥离 | 每路 PCM 16kHz ≈ 32KB/s 持续流；1k 路 = 32MB/s。Python 进程内叠 LLM 阻塞调用无法承载；Go epoll + 每连接 goroutine 可达数千路/节点 | 10k 路连接内存预算 < 1GB（≤64KB/连接） |
| 实时性隔离 | LLM 阻塞 3s 时音频帧不堆积：有界缓冲 + 主动丢帧，媒体面与推理面解耦 | 消除 head-of-line blocking |
| 传输可替换 | Month 2 WS → RTMP，网关内部替换，gRPC 下游契约与 Agent 零改动 | 架构文档 V1→V2 演进既定路线 |
| 录制挂点 | 流复制给存储（FLV→MinIO），AI 消费与录制互不影响 | Month 2 Week 6 计划 |

**验收导向：并发密度、转发延迟与丢帧率，而非单轮延迟。**

## Goal

1. 浏览器 ↔ 网关：WebSocket（wss）二进制音频帧上行、TTS 音频下行
2. 网关 ↔ 业务层：房间级双向 gRPC 流，网关只做字节中继，**不解码、不转码、不懂面试语义、不落盘**
3. 背压与丢帧：每会话有界缓冲，慢消费按策略丢帧并计数上报，读泵永不阻塞
4. 高可用：网关多实例无状态化、零停机发布、断线重连续传（seq）、优雅退出 drain
5. 可观测：Prometheus 指标 + OpenTelemetry 追踪 + 结构化日志（room_id 贯穿）+ 告警规则
6. 安全：TLS、JWT 鉴权、连接/帧级资源保护
7. 通过性能验收门 + 混沌演练（见 Verification）

## Non-Goals

- 视频采集/转发（字段预留，逻辑不实现）
- RTMP 摄入（Month 2，网关内部替换项）
- ASR/TTS 的实现与调用（backend 调用外部服务；网关不碰）
- 面试状态机/业务编排（Agent 与 backend 职责）
- 录制存储（Month 2）
- 媒体数据持久化（隐私边界：网关内存即焚）

## Existing Implementation

- Agent 侧：三个 HTTP 接口已可用（[API.md](../agent/API.md)），会话态在 MemorySaver（进程内）
- Go 侧：**尚无任何代码**（media-gateway/、backend/ 均未创建）
- 相关约定：架构文档规定 Go 不调 LLM、不做状态机；网关面向的下游消费者是 backend 的会话工作线程

## Proposed Change

### 架构与数据路径

```text
┌─────────┐  wss 二进制帧  ┌────────────────────┐  房间级双向 gRPC 流     ┌──────────────────┐
│ 浏览器   │ ────────────► │  media-gateway ×N   │ ◄──────────────────►  │ backend           │
│AudioWorklet│ ◄────────── │  (无状态, LB 之后)   │  AudioChunk 上行       │ session worker ×M │
└─────────┘  TTS 音频下行  └────────────────────┘  ServerEvent 下行       │ ASR→Agent→TTS     │
                              │       ▲                                 └──────────────────┘
                              ▼       │ 注册/路由/续传
                         ┌────────────┐
                         │    Redis   │  room_id → {gateway_node, worker_id, ttl}
                         └────────────┘
```

- 上行：AudioWorklet 采集 40ms PCM 帧（1280B）→ wss binary → 网关校验 seq/timestamp → 会话有界缓冲 → gRPC 流 → worker → ASR
- 下行：worker 产出 TTS 音频块 → 同一 gRPC 流下发 → 网关按 room_id 路由到浏览器 WS → 播放
- 网关是**纯媒体中继**：只看 room_id / seq / 字节，不解析音频内容
- LB 不要求会话粘性：重连靠 seq 续传 + Redis 注册表重挂，任意网关实例可接管

### 职责边界（与架构文档对齐）

| 层 | 负责 | 不负责 |
|----|------|--------|
| media-gateway | 连接管理、帧中继、背压/丢帧、路由注册、指标 | ASR/TTS、业务语义、持久化、转码、落盘 |
| backend | 会话编排、调用 ASR/Agent/TTS、Redis 会话态 | 大规模长连接承载 |
| Agent | 面试状态机、LLM | 一切媒体 |

### 关键设计决策

| # | 决策 | 选择 | 理由 | 备选 | 状态 |
|---|------|------|------|------|------|
| D1 | 服务形态 | 独立 `media-gateway/`（媒体面）与 `backend/`（控制面）分离 | RTMP 替换只动网关；独立扩容/发版；媒体面故障不传染业务面 | 单一 Go 服务 | ✅ 用户确认 |
| D2 | 帧模型 | `AudioChunk{room_id, seq, timestamp_ms, codec, data, final}`，codec 字符串标签，网关不解码 | 视频预留：Month 2 新增 `VideoChunk` 走同一流 | 音频专用流 | ✅ |
| D3 | v1 音频编码 | PCM 16kHz/16bit/mono，AudioWorklet 40ms 帧 | 网关零解封装；PCM 直喂 ASR 最短路径 | MediaRecorder+opus（需容器解析，Month 2 评估） | ✅ |
| D4 | 连接模型 | 每连接 read/write 双 goroutine + 每会话有界 channel（50 帧 ≈ 2s） | Go 惯用；10k 连接 = 20k goroutine 可行 | 自研 epoll 事件循环 | ✅ |
| D5 | 丢帧策略 | 水位 > 80% 丢最旧帧（计数+指标）；下行写超时断开 | 实时音频宁丢不堆；僵死连接必须回收 | 阻塞反压（采集漂移） | ✅ |
| D6 | 会话路由 | Redis 注册 `room_id → {gateway_node, worker_id, ttl}`，重连重挂 | 断线重连/零停机迁移前提 | 纯 LB sticky | ✅ |
| D7 | gRPC 拓扑 | 每房间一条双向流（worker 为 client），与 WS 连接 1:1 绑定 | 房间级故障域隔离 | 连接池多路复用 | ✅ |

### gRPC 契约（`proto/media.proto` v1，source of truth）

```protobuf
service MediaRelay {
  // 房间级双向流：backend worker 为 client；网关把浏览器音频灌入、把下行事件推回
  rpc SessionStream(stream ClientFrame) returns (stream ServerEvent);
}

message ClientFrame {
  oneof payload {
    AudioChunk audio;      // v1
    StreamControl control; // start/resume/stop（含 last_seq，支持重连续传）
  }
}

message AudioChunk {
  string room_id      = 1;
  uint32 seq          = 2;   // 会话内单调递增
  int64  timestamp_ms = 3;   // 采集时间戳
  string codec        = 4;   // "pcm_s16le_16k_mono"（v1）；视频预留 "h264"
  bytes  data         = 5;
  bool   final        = 6;   // 句尾/会话尾标记
}

message StreamControl {
  string room_id    = 1;
  string event      = 2;   // start | resume | stop
  uint32 last_seq   = 3;   // resume 时客户端最后收到的 seq
  string token      = 4;   // JWT（网关校验签名与 room 归属）
}

message ServerEvent {
  oneof payload {
    AudioChunk tts_audio;  // TTS 音频下行
    AgentText  text;       // 字幕/转写文本（流式）
    FlowControl fc;        // 下行背压提示（预留）
    SessionEnded ended;    // 会话终止（reason: normal | drain | timeout | error）
  }
}

message AgentText     { string room_id = 1; string text = 2; string role = 3; }
message FlowControl   { string room_id = 1; uint32 buffer_ms = 2; }
message SessionEnded  { string room_id = 1; string reason = 2; }
```

### 仓库结构

```
media-gateway/
├── cmd/gateway/main.go        # 入口：配置加载、信号处理、优雅退出
├── internal/config/           # 配置（env/flag），含全部限额默认值
├── internal/server/           # WS 服务：连接生命周期、心跳、鉴权、限额
├── internal/relay/            # 会话中继：缓冲、水位丢帧、seq 管理（纯逻辑，重点单测）
├── internal/rpc/              # MediaRelay gRPC server 侧实现
├── internal/registry/         # Redis 注册与路由
├── internal/metrics/          # Prometheus 指标
├── proto/media.proto          # source of truth（buf 生成 stub，禁止手改）
├── Makefile                   # generate / build / test / bench / loadtest
└── loadtest/                  # WS 压测端（模拟 N 会话 + 慢连接 + 断线注入）
```

## 容量与资源模型

单路资源（PCM 16kHz/16bit/mono，40ms 帧）：

- 带宽：25 帧/s × 1280B ≈ 32KB/s（0.26 Mbps）上行
- 内存：双 goroutine 栈 ~8KB + 会话缓冲 50×1280B ≈ 64KB + WS/gRPC 开销
- 消息率：25 msg/s/会话

| 并发路数 | 上行带宽 | 堆内存预算 | 备注 |
|---------|---------|-----------|------|
| 500 | 130 Mbps | ~32 MB | 单节点舒适区 |
| 2,000 | 520 Mbps | ~130 MB | **v1 性能验收门** |
| 5,000 | 1.28 Gbps | ~320 MB | 目标容量，需内核与 GC 调优 |
| 10,000 | 2.56 Gbps | ~640 MB | 需切 Opus（Month 2 评估，D3 预留） |

运行时要求（写入部署清单）：

- 内核：`nofile ≥ 100k`、`somaxconn`、`tcp_max_syn_backlog` 按目标容量调整
- Go：`GOMEMLIMIT` 显式设定（防 OOM 优先于 GC 频率）、`GOGC` 压测定参、`GOMAXPROCS` 对齐容器配额
- 缓冲复用：`sync.Pool` 管理 1280B 帧缓冲，压测验证 0 alloc 热路径

扩容触发条件（接 HPA 或人工）：节点活跃会话 > 预算 70%，或转发 p99 > 3ms 持续 5 分钟。

## 高可用与部署

- **无状态原则**：网关仅持有活跃连接与在途缓冲；会话注册与续传依据在 Redis；任何实例可被杀掉并由其他实例接管
- 部署拓扑：LB（TLS 终结）→ N × gateway；backend worker ×M；Redis 高可用（v1 单实例 + 备份演练，M3 起评估 Sentinel）
- **零停机发布流程**：SIGTERM → `/readyz` 返回未就绪（摘流）→ 停止接新连接 → 等待在途会话 drain（上限 `--drain-timeout`，默认 30s）→ 超时会话发送 `SessionEnded(reason=drain)` → 客户端按 Redis 注册表重连新节点、按 `last_seq` 续传
- 健康检查：`/healthz`（liveness：进程存活）、`/readyz`（readiness：Redis 可达 且 活跃连接 < 节点预算）
- 失败场景与恢复（runbook，M3 演练项）：

| 场景 | 表现 | 自动恢复 | 人工动作 |
|------|------|----------|----------|
| 网关实例崩溃 | 该实例会话断开 | 客户端重连 → 重挂注册表 → seq 续传 | 无 |
| Redis 不可用 | 新会话 fail-fast；存量会话继续（本地路由） | Redis 恢复后自动重注册 | 排查 Redis，禁止绕过注册表的静默降级 |
| worker 断流 | gRPC 流断开 | 网关保留会话 `--session-grace`（默认 10s）等重挂 | 排查 backend |
| LB 故障转移 | 连接全断 | 同网关崩溃路径 | 无 |
| 慢消费者 | 水位丢帧上升 | 丢帧策略自动；写超时断开 | 观察指标定位端侧 |

## 安全

- TLS：wss，证书在 LB 终结（v1）；直连模式网关自持证书
- 鉴权：WS 握手与 `StreamControl.token` 均携带 JWT；网关校验签名与 `room_id` 归属，防跨房间注入；v1 不做细粒度权限（Week 4 完善认证体系）
- 资源保护（防资源耗尽类攻击）：
  - 单帧上限 8KB、握手超时 5s、首帧超时 10s
  - 每连接消息率超 2× 正常值（>50 msg/s）告警并断开
  - 节点最大连接数硬顶（默认 8,000），超限拒绝并返回明确错误
  - Origin 校验（配置白名单）
- 隐私：网关不落盘、不缓存历史帧、日志不含媒体内容

## 可观测性

指标（Prometheus，命名前缀 `gateway_`）：

| 指标 | 类型 | 说明 |
|------|------|------|
| sessions_active | Gauge | 活跃会话数（容量水位核心指标） |
| frames_in_total / frames_out_total | Counter | 上下行帧数 |
| frames_dropped_total{reason} | Counter | 丢帧（watermark / write_timeout） |
| relay_latency_seconds | Histogram | 帧从 WS 入口到 gRPC 出口的转发延迟 |
| ws_write_timeouts_total | Counter | 下行写超时 |
| grpc_stream_errors_total{kind} | Counter | 流错误（broken / refused / auth） |
| reconnects_total | Counter | 客户端重连次数 |
| throttled_connections_total | Counter | 超限拒绝的连接 |

- 日志：slog JSON 结构化，字段 `room_id / seq / event / node`，不记录媒体内容
- 追踪：OpenTelemetry，一轮对话（上行帧 → worker 回执 → 下行音频）一个 span，采样 1%
- 告警规则（M3 定义到 alertmanager 规则文件）：
  - 丢帧率 > 1%（5min 窗口）→ P1
  - relay p99 > 20ms 持续 5min → P2
  - sessions_active 相对基线骤降 > 50% → P1
  - grpc_stream_errors 持续增长 → P2

## 测试与发布门禁

| 层级 | 内容 | 门禁 |
|------|------|------|
| 单测 | relay 水位/丢帧/seq（表驱动）、registry 路由、config 限额 | race 开启，覆盖率 ≥ 80%（relay 包） |
| 契约测试 | mock worker 对 media.proto 全消息类型收发 | buf breaking 检查通过 |
| 负载测试 | 2k 会话 + 1% 慢连接 + 断线重连注入 | 全部 SLO 达标（见 Verification） |
| Soak | 2k 会话 × 30min | 堆内存无持续增长（pprof diff 归零） |
| 混沌 | kill 网关实例 / kill Redis / worker 停流 / 网络延迟注入 | 恢复路径符合 runbook，无 OOM/死锁 |
| CI | golangci-lint、race、benchstat 基准回归 | 全绿才可合并 |

## Verification

**性能验收门（v1 上线前必须全绿，mock 回声 worker 压测，数据留档）**

| 指标 | 门限 | 工具 |
|------|------|------|
| 单节点并发会话 | ≥ 2,000 路（目标 5,000） | loadtest/ 压测端 |
| 帧转发延迟（WS→gRPC 出口） | p99 < 5ms | relay_latency 指标 |
| 无策略性丢帧率 | < 0.1%（水位丢帧单列） | 指标 |
| 内存 | ≤ 64KB/连接，soak 无增长 | runtime metrics + pprof heap |
| GC | STW p99 < 10ms | gctrace 采集 |
| 优雅重启 | 活跃会话恢复 < 2s，媒体空洞 ≤ 200ms | 滚动发布脚本 + 客户端 seq 校验 |
| 慢消费者注入 | 网关存活、丢帧计数上报、无 OOM | loadtest 慢连接模式 |
| 混沌 | 上表场景演练通过 | M3 演练记录 |

**功能验证**

- 回声链路：浏览器录音 → 回声播放，人耳无感知断裂
- 断线重连：kill WS 后 2s 内 resume，seq 连续无重复
- 异常路径：未注册 room 拒绝、重复 room 拒绝、非法 token 拒绝、心跳超时回收、超限连接拒绝

## 交付计划

| 里程碑 | 内容 | 时间 |
|--------|------|------|
| M1 骨架 | media.proto + buf 生成、WS 服务、relay 骨架、回声 worker 链路打通、基础指标 | Week 3 上半 |
| M2 加固 | 背压/水位丢帧、seq 续传、Redis 注册、优雅退出、鉴权透传、单测覆盖 | Week 3 下半 |
| M3 验收 | 压测达标留档、soak、混沌演练、告警规则、runbook；backend 接真实 ASR/Agent/TTS | Week 4 |
| M4 演进 | RTMP 摄入替换、流复制录制（FLV→MinIO）、Opus 编码评估 | Month 2 |

## Change Scope

### Add

- `media-gateway/`（上述全部）
- `proto/media.proto` + buf 生成流程
- `docs/gateway/GATEWAY_SPEC.md`（本文档）

### Modify

- `docs/ARCHITECTURE.md`：Go 层拆分为 media-gateway + backend 的图示与职责表（M1 实施时更新）
- `docs/ROADMAP.md`：Week 3 任务挂到本 SPEC

### Do Not Modify

- `Agent/`（Phase 1 闭环代码，本期零改动）
- `docs/agent/API.md`（文本接口契约不变）

## Acceptance Criteria

- [x] 决策门确认：D1（服务拆分）、D3（PCM v1）、D6（Redis 路由）
- [ ] 回声链路走通（浏览器 → 网关 → mock worker → 网关 → 浏览器）
- [ ] 性能验收门全绿并留档压测数据
- [ ] soak 与混沌演练通过，runbook 落档
- [ ] Agent 代码零改动；proto 契约评审通过
- [ ] 指标/日志/追踪/告警/优雅退出可用
