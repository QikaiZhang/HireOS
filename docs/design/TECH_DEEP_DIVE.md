# 技术深挖 — 实现细节与排障复盘

> 面向想理解"具体怎么做的、怎么排查问题的"的读者。所有代码路径可点击追溯。
> 配套：[设计决策](DESIGN_DECISIONS.md) ｜ [面试宣讲包](../interview/interview-package.md)

---

## 1. 水位丢帧的实现：非阻塞 + 三段式兜底

核心代码在 `media-gateway/internal/relay/relay.go` 的 `Session.Offer`：

```go
// 超水位：丢最旧一帧，给新帧腾位置
if len(s.inbound) >= s.watermark {        // 水位 = capacity × 80%
    s.dropOldestInbound(DropWatermark)
}
select {
case s.inbound <- f:                       // 常规路径：非阻塞入队
default:
    s.dropOldestInbound(DropWatermark)     // 竞态兜底：再丢一帧最旧
    select {
    case s.inbound <- f:
    default:
        s.droppedIn.Add(1)                 // 仍失败：丢新帧，计数
    }
}
```

三个设计点：

1. **为什么"丢最旧"而不是"丢最新"**：实时语音的价值随时间衰减，ASR 需要的是最新上下文。丢最新会导致"断在最近说的话"，丢最旧则保住候选人刚说的内容。
2. **为什么先检查水位再 select-default**：纯 `select/default` 只在缓冲全满时触发，此时缓冲里全是最旧的帧；水位检查让丢弃提前发生在 80% 处，保留 20% 吸收突发。这也是为什么实测稳态缓冲长度 = 水位（8=10×80%）而不是容量。
3. **为什么必须有计数**：丢帧是策略，但必须是**可观测的策略**。`gateway_frames_dropped_total{reason}` 让"策略性丢帧"和"系统性丢帧"在监控上可区分（见第 4 节复盘）。

**守恒验证**（`TestRelayDropConservationUnderPressure`）：灌入总数 = worker 实收 + 关闭后缓冲存量 + 丢弃计数。这条断言抓住了实现中最隐蔽的一类 bug：丢帧计数与真实丢弃不一致（意味着有人偷偷丢了帧没记账）。

## 2. 会话关闭协议：排干语义

`Consume`/`PopOutbound` 的语义：**会话关闭后不是立刻返回 false，而是先排干缓冲存量，再返回 false**。

```go
case <-s.done:                              // 会话已关闭
    select {
    case f := <-s.inbound: return f, true   // 先把存量送完
    default: return AudioFrame{}, false     // 排干才结束
    }
```

为什么：Close 的触发方（浏览器断开）和消费方（worker 泵）是异步的，缓冲里可能还有已合法接收的音频——直接丢弃会切掉说话的尾巴。代价是使用方必须理解协议：**排干循环前必须先 Close，否则消费方永远等不到 false**——我们自己的单测就死锁过一次（601s 超时），修复方式写进了 godoc。

## 3. LangGraph 人机循环：中断点的选择

`Agent/graph/builder.py`：

```python
compiled_graph = graph.compile(
    checkpointer=checkpointer or MemorySaver(),
    interrupt_before=["listen"],
)
```

图结构 `prebuild → fetch → [中断] → listen → react_judge → (追问|下一题|报告)`：

- **中断点选在 listen 之前**而不是 fetch 之后：listen 承担"记录回答进 chat_history"的职责，注入回答后从 listen 恢复，职责链自然衔接。
- **API 侧恢复**（`api/routes.py`）：`update_state(config, {"candidate_latest_answer": answer})` + `invoke(None, config)`——None 输入表示"从断点继续"而非新输入。
- **路由三分支**（`route_after_react`）：`loop_ask → listen`（追问直接复用中断等待）、`next_stage → fetch`、`finish → summary`。这里修掉过原骨架的一个 bug：旧实现深挖问题会被 `fetch_base_question` 用下一道基础题覆盖——追问的生成和提问的生成是两个动作，不能共用一个节点出口。

**踩坑记录**：重写 builder 时漏了 `listen → react_judge` 静态边，图在 listen 后直接终止，端到端验证第 2 轮就 409。LangGraph 里节点没有出边 = 静默终结，不会报错——**条件边改写时先画全节点出边清单再动手**。

## 4. 排障复盘：一次"伪性能瓶颈"的定位过程

**现象**：真实服务冒烟，200 帧灌入只回 144 帧，且下行速率掉到约 2 帧/秒，客户端 30 秒超时。

**第一直觉（错误方向）**：gRPC 流控饥饿——特征太像了（约 40~100 帧后卡住 ≈ 2×64KB 窗口量级）。

**隔离实验**（`TestGRPCRelayPacedThroughput` 的前身）：绕过 WS 层，纯 gRPC + 真实 TCP 直灌 500 帧 → 复现"恰好 40 帧停摆"。调大 `InitialWindowSize` 到 1MB → **依旧 40 帧**。流控假设被证伪。

**重新读数**：40 = 缓冲容量 50 × 水位 80%。灌入是瞬时 500 帧，消费端（echo worker）来不及消化——**这不是停摆，是水位丢帧策略在正确工作**：500 帧 = 45 送达 + 455 策略丢弃，守恒成立。

**结论与教训**：

1. 实时系统的测试必须按真实节奏灌帧（25fps = 40ms 间隔）。"吞吐下降"有两种完全不同的成因：策略性丢帧（正常，看 `frames_dropped_total`）和系统性阻塞（异常，看缓冲是否有界）。区别它们靠的是守恒断言，不是吞吐曲线。
2. 隔离验证顺序：先拆层（WS / gRPC / relay），再调参数（窗口大小）。每次实验只证伪一个假设——"调大窗口没变化"直接排除了流控假设，避免继续在错误方向调参。

## 5. 生命周期时序 bug：worker 先于浏览器接入

**现象**：集成测试全链路回声中，浏览器 WS 连接被拒："房间已有活跃连接"。

**根因**：worker 是 gRPC client，会先拨号建流并在 registry 创建会话（get-or-create）；浏览器 WS 随后接入时，看到"已存在活跃会话"就当成重复接入拒绝了。但**worker 先接入恰恰是正常时序**。

**修复**（`relay.Session.ClaimBrowser`）：

```go
func (s *Session) ClaimBrowser() bool {
    return s.browserAttached.CompareAndSwap(false, true)
}
```

把"重复接入"的判定从"会话存在"改为"浏览器端已被认领"：会话存在但浏览器未认领 → 允许接入；已认领 → 拒绝。原子 CAS 保证并发连接下只有一个能认领成功。**通用教训**：共享对象的"占用"判定要用"端点认领"语义，而不是"对象存在"语义——存在 ≠ 冲突。

## 6. 延迟预算：为什么网关的验收指标不是延迟

一次"说话→听到 AI 回复"的分解（详见 [MEDIA_BASICS](../gateway/MEDIA_BASICS.md) 第 6 节）：

| 环节 | 延迟 |
|------|------|
| 采集攒帧（40ms） + 上行 | ~50ms |
| **网关转发** | **< 1ms** |
| ASR 句尾识别 | 200~800ms |
| LLM 生成追问 | 1~3s（主导） |
| TTS 首字节 | 200~500ms |

网关 <1ms 占比 <0.1%——所以它的验收指标是**并发密度（≥2k 路/节点）、转发 p99（<5ms）、策略丢帧率（<0.1%）**，而不是端到端延迟。这个认知决定了 SPEC 里所有 SLO 的形状。

## 7. 指标设计：防基数爆炸的 label 策略

`internal/metrics/metrics.go`：所有指标禁止用 `room_id` 做 label（基数随会话数无界增长会打爆 Prometheus）。会话维度的问题用日志（room_id 贯穿）+ 追踪（一轮对话一个 span）解决，指标只保留聚合视图。丢帧按 `reason` 分桶（watermark / session_ended / write_timeout），reason 是有限枚举——**label 的合法值集合必须是闭集**。

## 8. 项目侧 Agent 的工程细节速记

| 细节 | 位置 | 要点 |
|------|------|------|
| 深挖上限 | `nodes/online_nodes.py` | `deep_dive_count` 在 fetch 新题时重置，react_judge 强制执行 `MAX_DEEP_DIVE_PER_QUESTION=2`，LLM 决策也受硬上限约束（不信任模型输出） |
| LLM JSON 约束 | `llm/client.py` | `response_format=json_object` + 解析失败返回 None → 节点 mock 兜底，两层防御 |
| 会话状态恢复 | `api/routes.py` | `get_state().next` 为空 = 图已终结 → 409 拒绝继续对话；不存在 → 404 |
| 报告持久化时机 | `api/routes.py` chat | 图执行返回且 `global_phase == "end"` 时落库，`GET /report` 只读 DB——图状态与持久化职责分离 |
