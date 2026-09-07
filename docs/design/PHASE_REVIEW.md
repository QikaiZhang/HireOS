# 阶段复盘 — Phase 1（Agent 闭环）+ 网关 M1

> 日期：2026-09-08 ｜ 覆盖：第一个阶段的两条主线 ｜ 明细报告：[PHASE1_REPORT](../agent/PHASE1_REPORT.md) ｜ [M1_REPORT](../gateway/M1_REPORT.md)

## 这一阶段做了什么

### 主线一：文字面试闭环（Phase 1，对标 Week 1）

从"LangGraph 骨架 + 全 mock"补齐到可独立运行的 Agent 服务：

```text
创建面试 → LLM 生成题库 → 逐题提问 → [图中断，等候选人]
       → 记录回答 → ReAct 评估（深挖/下一题/结束）→ 报告 → SQLite 落库可查
```

- 三个 HTTP 接口：`/interview/start`、`/interview/chat`、`/report/{room_id}`
- 短期记忆（滑动窗口）+ 长期记忆（SQLite 双表）
- LLM 全链路可插拔：无 Key 时确定性 mock 兜底（显式日志），闭环验证不依赖 Key
- 验证：29 项端到端断言全过（含 404/409 异常路径）+ 真实 uvicorn 冒烟

同时清掉了四个历史问题：`config/setting.py` 转义损坏（import 即崩）、图路由 bug（追问被下一题覆盖）、state 字段未注解（LangGraph 不建 channel）、`pyproject.toml` 构建后端无效。

### 主线二：媒体网关 M1（对标 Week 3 提前量）

```text
浏览器 AudioWorklet(40ms PCM) → WS → 网关 → gRPC 双向流 → worker（回声）→ 网关 → 浏览器播放
```

- proto 契约 source-of-truth + 生成流程；relay 纯逻辑核心（水位丢帧/seq/关闭排干）
- 生产要件：读写泵、心跳、帧上限、连接硬顶、9 项 Prometheus 指标、健康检查、信号优雅退出
- 验证：单测 7 项 + 集成 3 项（-race 干净）；25fps 节奏灌帧零丢帧；压力灌帧守恒（45+455=500）；真实服务 200 帧逐帧一致

### 文档体系（本阶段同步产出）

| 文档 | 定位 |
|------|------|
| [PHASE1_SPEC / REPORT](../agent/) | Agent 闭环的方案与交付证据 |
| [GATEWAY_SPEC](../gateway/GATEWAY_SPEC.md) | 网关完整设计：性能收益评估、容量模型、HA、安全、验收门 |
| [MEDIA_BASICS](../gateway/MEDIA_BASICS.md) | 音视频基础（采样/帧/丢帧策略/RTMP），零基础可读 |
| [DESIGN_DECISIONS](DESIGN_DECISIONS.md) | 10 条 ADR：每个决策的备选方案与代价 |
| [TECH_DEEP_DIVE](TECH_DEEP_DIVE.md) | 实现细节与排障复盘 |
| [interview-package](../interview/interview-package.md) | 宣讲话术、STAR-L 故事线、三层追问 |

## 设计理念（一句话版）

1. **先证明产品成立，再给心脏换肺**：文字闭环先通，语音后接，RTMP 后置——传输层替换不动 Agent。
2. **媒体面与推理面物理隔离**：LLM 卡 2 秒不能拖垮音频流；Go 管并发承载，Python 管 AI 推理，gRPC 解耦。
3. **实时流宁丢不等**：有界缓冲 + 水位丢最旧 + 丢帧可观测；不追求"不丢"，追求"丢得可控、可解释、可验证"。
4. **无 Key 可验证**：mock 兜底显式化，闭环验证不依赖外部服务，配 Key 即切真实推理。
5. **证据先于结论**：每个"完成"绑定测试/脚本/日志；planned 与 verified 严格区分。

## 量化状态（截至本复盘）

| 项 | 状态 |
|----|------|
| Agent 闭环验证 | 29/29 断言通过（verified） |
| 网关链路 | 回声端到端跑通，真实服务 200 帧零丢帧（verified） |
| 丢帧策略 | 节奏灌帧零丢帧；压力守恒 45+455=500（verified） |
| 修复历史缺陷 | 4 处 P0/P1（配置损坏、路由 bug、state 未注解、构建后端） |
| 未完成（planned） | Redis 路由、seq 续传、drain、JWT、2k 并发压测、真实 ASR/TTS 接入、RTMP |

## 下一阶段入口

1. **Week 2**：Go backend（业务编排）+ Redis（会话路由 + Agent checkpointer 统一）
2. **网关 M2**：seq 断线续传、优雅 drain、JWT、Origin 收敛
3. **网关 M3**：2k 并发压测门 + 混沌演练 + 真实 ASR/Agent/TTS 接入
4. **Week 4**：TTS 集成、MySQL、前端三页、docker-compose
