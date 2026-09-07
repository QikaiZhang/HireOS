# 路线图

## 总览

| 阶段 | 主题 | 核心交付 |
|------|------|----------|
| Month 1 | MVP — Python Agent 为核心 | 可语音面试的完整产品 |
| Month 2 | 流媒体增强（💤 可选） | RTMP 网关替换 WebSocket / 录制 / 加固 |
| Month 3 | 工程化 | 稳定性、可扩展性、面试准备 |

---

## Month 1：HireOS MVP

> **目标**：证明 AI 面试这个产品成立。月底能演示：打开网页 → AI 问问题 → 我回答 → AI 追问 → 生成面试报告。

### Week 1：Python Agent 核心 ✅（2026-09-07 完成，见 [Phase 1 报告](agent/PHASE1_REPORT.md)）

**专注**：先把 Agent 大脑做出来，不要急着接语音。

| 天数 | 任务 | 产出 |
|------|------|------|
| Day 1 | LangGraph 基础骨架 | `InterviewState` 定义、图结构 |
| Day 2-3 | 面试状态机 | INIT → INTRO → TECH → FOLLOW_UP → SUMMARY |
| Day 4-5 | 加入 Memory | 短期消息记忆 + 长期知识存储 |
| Day 6-7 | Agent API 化 | FastAPI: `POST /interview/start`, `POST /interview/chat`, `GET /report/{id}` |

**结束时**：Python Agent 可独立运行，通过 HTTP 进行文字面试。

详见 [Agent 设计文档](agent/AGENT_DESIGN.md)

### Week 2：Go Backend 接入

**目标**：Go 做真正的业务控制层。

| 模块 | 说明 |
|------|------|
| Interview Session | Session 模型与管理 |
| gRPC Client | 连接 Agent，双向流通信 |
| Redis 状态 | 缓存 Session 状态、对话上下文 |
| HTTP API | 前端接口 |

**Go 学习**（每天 30-60 分钟）：gRPC、protobuf、context、goroutine

### Week 3：接入语音链路（设计见 [媒体网关 SPEC](gateway/GATEWAY_SPEC.md)）

> 进度：**网关 M1 已完成（2026-09-08）** — WS 接入、gRPC 中继、水位丢帧、回声链路验证通过，
> 见 [M1 报告](gateway/M1_REPORT.md)。剩余：Redis 注册/续传/drain（M2）、压测验收（M3）、真实 ASR/TTS 接入。

**目标**：真人说话 → 文字 → AI 回答。

| 组件 | 技术 |
|------|------|
| 浏览器采集 | MediaRecorder API |
| 传输 | WebSocket |
| 语音识别 | ASR 服务 |
| 后端处理 | Go 接收音频 → 调 ASR → 文字 → gRPC → Agent |

**Go 学习**：WebSocket Server，理解 TCP → Connection → Frame → Message

### Week 4：完善产品闭环

**交付内容**：

1. **TTS 集成**：AI 文字回答 → TTS → 音频 → 浏览器播放
2. **MySQL 存储**：面试记录、对话历史、评分报告
3. **简单前端**：首页（创建面试）、面试页（摄像头+聊天）、报告页（评分展示）
4. **Docker Compose**：一键启动全栈

**Go 学习**：流媒体基础概念（H264、AAC、RTMP 结构），先看，不写。

### Month 1 结束时仓库状态

```
hireos
├── backend/            ✅
├── agent-service/      ✅
├── frontend/           ✅
├── proto/              ✅
├── docs/               ✅
└── docker-compose.yml  ✅
```

---

## Month 2：流媒体增强

> **2026-09 更新：RTMP 已降级为可选项（💤）。** 评估结论：AI 语音面试产品闭环不需要 RTMP——高并发承载由网关提供、LLM 阻塞隔离由水位丢帧提供；录制走网关流复制即可。架构上变化隔离已就位（网关只搬字节、codec 为字符串标签），未来需要时只动网关入口层。详见 [Stage 总览](design/STAGES.md)。

> **目标**：WebSocket 音频传输升级为 RTMP 网关，Agent 完全不用改。（可选演进）

### Week 5：RTMP 基础

WebSocket Audio → 替换成 → RTMP Audio

```
gateway/
├── tcp.go
├── handshake.go
├── chunk.go
└── stream.go
```

### Week 6：媒体处理

- AAC 音频解析
- H264 视频解析
- FLV 封装
- MinIO 存储录像

### Week 7：工程优化

- 断流恢复
- 超时与限流
- 日志与监控
- 错误处理

### Week 8：面试准备

- 绘制架构图
- 整理技术难点
- 准备设计决策问答
- 为什么不用 WebRTC 上传？为什么 gRPC Streaming？如何水平扩展？

---

## Month 3+（展望）

- [ ] 面试题库管理
- [ ] 多轮面试编排（HR 面 → 技术面 → 主管面）
- [ ] 候选人 Dashboard
- [ ] 面试回放与分析
- [ ] 情感分析（语音语调）
- [ ] 多语言支持
- [ ] 水平扩展与负载均衡

---

## 设计原则

1. **垂直切片优先**：每层简单但完整跑通，不一上来就做深度优化
2. **变化隔离**：替换传输层不影响 Agent 层
3. **Agent 优先，流媒体后置**：先证明产品成立，再给心脏换肺
4. **每天碰 Go**：Python Agent 开发期间，每天抽 30-60 分钟学习 Go 相关技术
