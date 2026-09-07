# Stage 进度总览 — HireOS

> 更新：2026-09-08 ｜ 本文档回答三个问题：现在到哪了、每个 stage 的完成标准是什么、接下来按什么顺序走。
> 状态口径：✅ verified（有测试/脚本证据）｜ 🟨 部分完成 ｜ ⬜ 未开始 ｜ 💤 可选（不做不影响产品闭环）

---

## 0. 一个前提认知：RTMP 已降级为可选项

原规划中 RTMP 是 Month 2 主题。重新评估后确认：**AI 语音面试产品闭环不需要 RTMP**。

| RTMP 能提供的 | 我们是否需要 |
|--------------|-------------|
| 高并发音频承载 | ❌ 网关（WebSocket + gRPC 中继）已提供 |
| LLM 阻塞隔离 | ❌ 水位丢帧策略已提供 |
| 推流生态兼容（OBS 等） | 🟡 面试场景用浏览器麦克风，无此需求 |
| 录像回放 | 💤 录制走"网关流复制 → FLV → MinIO"即可，不依赖 RTMP 摄入 |
| 面试叙事的工程深度 | 🟡 已由水位丢帧、守恒验证、媒体面拆分等覆盖 |

**架构上零欠债**：网关只搬字节、codec 是字符串标签、gRPC 契约不感知传输协议。未来若需要，只动网关入口层；不需要，不做任何事。

---

## 1. 全景阶段图

```text
Stage 1  Agent 大脑（文字闭环）        ✅ verified
Stage 2  媒体网关（音频实时链路）      🟨 M1 done，M2/M3 未做
Stage 3  Go 业务层（编排 + 存储）      ⬜
Stage 4  语音产品闭环（ASR/TTS 接入）  ⬜   ← Month 1 MVP 的核心缺口在这里
Stage 5  产品壳（前端 + 部署）         ⬜
Stage 6  工程化加固（压测/混沌/监控）  ⬜
Stage 7  RTMP/WebRTC/录制             💤 可选
```

## 2. 各 Stage 状态与完成标准

### Stage 1 ｜ Agent 大脑 ✅ verified（2026-09-07）

- LangGraph 状态机：prebuild → fetch → [中断] → listen → react_judge → summary
- 三个 HTTP 接口（start / chat / report）、短期滑动窗口 + SQLite 持久化
- LLM 全链路可插拔（无 Key mock 兜底显式化）
- **完成标准已达成**：29 项端到端断言全过 + 真实 uvicorn 冒烟；证据见 [PHASE1_REPORT](../agent/PHASE1_REPORT.md)

### Stage 2 ｜ 媒体网关 🟨 约 40%

| 子项 | 状态 |
|------|------|
| M1 骨架：proto 契约、WS 服务、relay 水位丢帧、gRPC 中继、指标、回声链路 | ✅ verified（[M1_REPORT](../gateway/M1_REPORT.md)） |
| M2 加固：Redis 会话路由、seq 断线续传、优雅 drain、JWT | ⬜ |
| M3 验收：2k 并发压测门、soak、混沌演练、告警规则 | ⬜ |
| **完成标准**：`wssmoke` 单节点 2k 路并发全绿；杀网关实例会话恢复 < 2s | 未达成 |

### Stage 3 ｜ Go 业务层 ⬜ 未开始（原 Week 2）

- backend 服务：面试编排（调 ASR→Agent→TTS）、Redis 会话态、MySQL 记录
- 消费网关 `SessionStream`（proto 契约已定，见 [API 与 proto](../gateway/GATEWAY_SPEC.md)）
- **完成标准**：创建面试 → 编排一轮"音频进 → 文字答 → AI 问 → TTS 出"的完整链路（可先用回声代替 TTS）

### Stage 4 ｜ 语音产品闭环 ⬜ 未开始（原 Week 3 的产品目标）

- ASR 接入（流式）、TTS 接入、VAD 句尾判断
- **完成标准**：对着麦克风说一段话 → ASR 出文字 → Agent 追问 → TTS 语音播出，端到端 < 5s
- 依赖：Stage 2（网关链路）+ Stage 3（编排）

### Stage 5 ｜ 产品壳 ⬜ 未开始（原 Week 4）

- 前端三页（创建/面试/报告）、MySQL 迁移、docker-compose 一键起
- **完成标准**：浏览器全流程走完"创建面试 → 语音面试 → 看报告"，一条命令拉起全栈

### Stage 6 ｜ 工程化加固 ⬜ 未开始（原 Month 3）

- 压测验收门（Stage 2 M3 的一部分）、监控告警落地、限流、日志聚合
- **完成标准**：GATEWAY_SPEC 验收表全绿并留档

### Stage 7 ｜ 可选演进 💤

- RTMP 摄入、WebRTC 客户端体验层、录像（流复制 → MinIO）、Opus 编码、多轮面试编排、题库管理
- 进入条件：按产品需要触发，无时间表

## 3. 整体进度估算

```text
Month 1 MVP 目标（可演示的语音 AI 面试）
├── Agent 层          ██████████ 100%  ✅
├── 媒体网关          ████       40%   链路通，加固/验收未做
├── Go 业务层         ░          0%
├── 语音产品闭环      ░          0%   ← MVP 的关键缺口
└── 产品壳            ░          0%
合计：约 40%——"大脑和血管"就绪，"嘴和耳朵"（ASR/TTS）与"身体"（编排/前端）未接
```

**对 MVP 的最短路径**：Stage 3 → Stage 4 → Stage 5（网关 M2/M3 的加固可以与 Stage 3 并行，不阻塞语音闭环；用回声 worker 即可先联调编排层）。

## 4. 面试口径提醒

- 可以说：Agent 闭环 verified、网关回声链路 verified、丢帧策略有守恒验证
- 不能说：语音面试产品已完成（ASR/TTS 未接入）、压测指标（未做）
- 现在的项目一句话：**"大脑（Agent）和血管（媒体网关）已验证跑通，正在接嘴和耳朵（ASR/TTS）"**
