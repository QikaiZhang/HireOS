# 设计决策记录（ADR）— Phase 1 + 媒体网关 M1

> 记录 HireOS 第一个阶段的关键技术决策：背景 → 选择 → 被否掉的方案 → 代价。
> 状态标注：✅ 已实现并验证 ｜ 🚧 已设计未实现 ｜ 面试追问见 [interview-package](../interview/interview-package.md)

---

## AD-01 ｜ Agent 用状态机而不是 while 循环

- **背景**：AI 面试不是单轮问答，是"提问→听答→评估→追问/推进→出报告"的多轮流程，且追问路径依赖运行时评估结果。
- **选择**：LangGraph StateGraph，节点只返回增量更新，路由边表达流程分支。✅
- **否掉的方案**：`while True: input(); call_llm()` 聊天循环——无法表达"追问几次后推进"、"回答充分就跳下一领域"这类控制流；状态散落在闭包里，无法持久化恢复。
- **代价**：学习成本 + pydantic state 每个字段必须显式注解（LangGraph 按 channel 建状态，无注解的字段不生效——我们踩过这个坑：`room_id = ""` 这种类属性写法不会注册为状态字段）。
- **收益证据**：`Agent/graph/builder.py` 图结构即流程文档；`interrupt_before=["listen"]` 一行代码获得 human-in-the-loop。

## AD-02 ｜ 人机交互用图中断，不用业务层轮询

- **背景**：面试是"AI 提问 → 等候选人 → 再推进"的交互，Agent 进程不能自己跑完。
- **选择**：`interrupt_before=["listen"]` + MemorySaver checkpointer，`/interview/chat` 用 `update_state` 注入回答后 `invoke(None)` 恢复。✅
- **否掉的方案**：API 层自管 dict 存会话、每轮手动调对应节点——等于在业务层重写一遍图的调度，状态机形同虚设。
- **代价**：MemorySaver 是进程内的，Agent 重启丢进行中会话（已记录 P2，Week 2 换 Redis checkpointer）。
- **边界**：中断点是唯一的人机边界，图内其余流转全自动，这让"面试推进逻辑"可以整体单测。

## AD-03 ｜ LLM 无 Key 时 mock 兜底，兜底必须显式可见

- **背景**：本地开发/CI 无 LLM_API_KEY；但闭环验证不能依赖 Key。
- **选择**：`llm/client.py` 统一走 OpenAI 兼容协议；无 Key 或调用失败返回 None，节点降级到确定性 mock，**每次兜底打 `[LLM MOCK]` 日志**。✅
- **否掉的方案**：强依赖真实 Key（本地不可验证）；或静默 mock（会把 mock 输出误当真实推理，是最危险的失败模式）。
- **代价**：两套行为并存，报告 JSON schema 在 mock 与 LLM 输出间不一致（已记录 P2，Week 4 前固化契约）。
- **收益**：Phase 1 全部 29 项验证在无 Key 环境跑通；配 Key 后同一脚本即验证真实推理。

## AD-04 ｜ Go 层拆分：媒体网关独立成服务（媒体面/控制面分离）

- **背景**：语音链路上行是持续字节流（每路 32KB/s PCM），Python Agent 进程里再叠 LLM 阻塞调用无法承载大规模并发；且 Month 2 要把 WS 换成 RTMP。
- **选择**：独立 `media-gateway/`（媒体面：连接承载、帧中继、背压丢帧）+ `backend/`（控制面：ASR/Agent/TTS 编排）。✅ M1
- **否掉的方案**：单一 Go 服务——部署简单，但 RTMP 替换会动业务层，且媒体与业务的故障域、扩容粒度耦合。
- **代价**：多一个服务的部署与运维成本；多一跳 gRPC（<1ms，loopback 可忽略）。
- **收益证据**：Month 2 换 RTMP 时 Agent 与 backend 契约零改动（架构文档 V1→V2 既定路线）；媒体面可独立水平扩容。

## AD-05 ｜ 网关只搬字节：不解码、不转码、不懂面试语义、不落盘

- **背景**：网关最容易膨胀成"什么都管"的泥球；同时音视频数据有隐私边界。
- **选择**：网关只看 `room_id / seq / 字节`，codec 只是字符串标签透传。✅
- **收益**：① 热路径零解析（转发 p99 预算 <5ms 的前提）；② ASR 换厂商、加 Opus、加视频都不动网关核心；③ 媒体数据内存即焚，隐私边界清晰；④ 录制/转码作为挂点后置。
- **代价**：v1 客户端必须发网关可直转的格式（PCM），放弃了 MediaRecorder/Opus 的省带宽（10k 路 PCM ≈ 2.6Gbps 是上限约束，已在容量表标注，Opus 是 Month 2 评估项）。

## AD-06 ｜ 实时音频"宁丢不等"：水位丢最旧帧

- **背景**：LLM 卡 2~3 秒时，若音频帧排队等待，延迟越积越多（head-of-line blocking），ASR 拿到的是过时上下文。
- **选择**：每会话双端有界缓冲（50 帧 ≈ 2s），80% 水位触发丢最旧；读泵永不阻塞；下行写超时断开；丢帧计数进指标。✅
- **否掉的方案**：无界队列（OOM 风险）＋ TCP 反压（采集端漂移，丢帧从策略变失控）。
- **代价**：消费慢时音频内容有缺口（对语音识别可接受：ASR 需要的是新鲜上下文而非完整录音）。
- **验证证据**：`TestRelayDropConservationUnderPressure`——500 帧裸灌，45 送达 + 455 丢弃 = 500，守恒、有界、不崩；`TestGRPCRelayPacedThroughput`——25fps 真实节奏 100 帧，零丢帧。

## AD-07 ｜ 传输协议：先 WebSocket，RTMP 是替换项而非起点

- **背景**：RTMP 协议解析是工程深度，但不影响"AI 面试产品是否成立"。
- **选择**：M1 用 WS 二进制帧（客户端发裸 PCM，seq/timestamp 由网关分配）；`codec` 字段预留，Month 2 RTMP 只动网关入口层。✅
- **否掉的方案**：WebRTC 直传服务端——SFU/TURN/SDP 协商的复杂度先于产品验证出现；WebRTC 留作客户端低延迟体验层的后续选项。
- **代价**：WS 上行带宽高于 Opus（见 AD-05 容量表）。

## AD-08 ｜ 下游契约：gRPC 双向流，worker 是 client

- **背景**：网关 ↔ worker 两个方向都是持续消息流（上行音频 / 下行 TTS+字幕）。
- **选择**：每房间一条 bidi stream，worker 拨号进网关（server），流生命周期 = 会话生命周期。✅
- **否掉的方案**：HTTP 轮询（延迟+无法表达"进行中"）；网关反连 worker（需要服务发现，worker 数量 M > 网关 N，反连方向错）。
- **实施修正**：SPEC 初稿把 AudioChunk 放在 worker→网关方向，与真实方向矛盾；实施时按方向重命名为 `WorkerFrame`/`GatewayFrame` 并在 proto 注释写明方向约定（见 M1 报告偏差表）。教训：**流式契约评审先画箭头方向图**。

## AD-09 ｜ 扩展模型：网关无状态 + Redis 路由，LB 不要求粘性

- **背景**：高可用要求任意网关实例可被杀掉、由其他实例接管。
- **选择**：🚧 网关仅持有活跃连接与在途缓冲；Redis 注册 `room_id → {gateway_node, worker_id, ttl}`；重连按 seq 续传（M1 进程内 registry 已留接口，Redis 是 M2 第一项）。
- **否掉的方案**：LB 会话粘性——无法处理实例崩溃后的接管，只能处理正常重路由。
- **代价**：引入 Redis 强依赖；降级策略明确为"存量会话继续，新会话 fail-fast"，禁止静默绕过注册表。

## AD-10 ｜ 持久化分层：SQLite 先行，MySQL 后置

- **背景**：Phase 1 需要证明"报告落库可查"的持久化边界，但 MySQL 部署在 Week 4。
- **选择**：stdlib SQLite，接口按 `(room_id, ...)` 粒度设计，Week 4 迁 MySQL 只换实现。✅
- **收益**：零部署成本验证了 `GET /report` 的重启可查语义（报告表与 session 表分离）。
