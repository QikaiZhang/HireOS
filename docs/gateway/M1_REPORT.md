# M1 交付报告 — 媒体网关骨架与回声链路

> 日期：2026-09-08 ｜ 对应 [GATEWAY_SPEC](GATEWAY_SPEC.md) 交付计划 M1 ｜ 前置知识：[MEDIA_BASICS.md](MEDIA_BASICS.md)

## 结论

M1 完成：`media-gateway/` Go 服务可用，链路 **浏览器 WS → 网关 → gRPC 双向流 → worker** 已打通，
真实服务冒烟 200 帧（25fps 真实语音节奏）逐帧一致、零丢帧；全部测试 `-race` 干净。

```text
✅ M1 骨架：proto 生成、WS 服务、relay 中继、回声 worker、指标
⬜ M2 加固：seq 续传、Redis 注册、优雅 drain、JWT
```

## 交付清单

| 组件 | 文件 | 说明 |
|------|------|------|
| 契约 | `media-gateway/proto/media.proto` | source of truth；`make generate` 产出 stub（`gen/mediav1/`），禁止手改 |
| 中继核心 | `internal/relay/relay.go` | 传输无关纯逻辑：双端有界缓冲、80% 水位丢最旧、seq 分配、会话关闭排干语义 |
| WS 服务 | `internal/server/ws.go` | coder/websocket；读写双泵、首帧/空闲超时、帧上限、连接数硬顶、心跳 |
| gRPC 服务 | `internal/rpc/relay_server.go` | MediaRelay.SessionStream：上行泵（含转发延迟埋点）+ 下行路由；worker 先于浏览器接入的会话时序处理 |
| 注册表 | `internal/registry/` | 接口 + 进程内实现；M2 换/加 Redis 实现，接口不变 |
| 指标 | `internal/metrics/` | SPEC 定义的 9 项 Prometheus 指标；`/metrics` 暴露 |
| 配置 | `internal/config/` | 全部限额带默认值，flag 可覆盖 |
| 入口 | `cmd/gateway/main.go` | HTTP(:8080: /ws /healthz /readyz /metrics /echo) + gRPC(:9090)，信号优雅退出 |
| 回声 worker | `cmd/echoworker/main.go` | M1 验证用 mock：上行音频原样回放；带拨号重试 |
| 冒烟工具 | `cmd/wssmoke/main.go` | WS 压测/验证客户端，默认 25fps 节奏，逐帧校验 |
| 回声页面 | `cmd/gateway/echo.html` | AudioWorklet 采集 40ms PCM → WS → 回声播放，人工听感验证用 |

关键设计在代码中的位置：水位丢帧 = `relay.Session.Offer/PushOutbound`；"实时流宁丢不等"的守恒验证 = `TestRelayDropConservationUnderPressure`；worker 先接入的时序处理 = `ws.serveConn` 的 `ClaimBrowser`。

## 验证记录

**单测（relay 包，-race）**：7 项全过 —— 水位丢最旧、低于水位零丢帧、关闭排干、Close 唤醒阻塞消费、下行同策略、并发灌入守恒、浏览器认领唯一性。

**集成测试（进程内 httptest + bufconn）**：3 项全过 —— 全链路回声 50 帧逐帧一致、重复房间拒绝、无 start 控制帧报错。

**中继层吞吐（真实 TCP gRPC）**：2 项全过 —

- 节奏灌帧：100 帧 @25fps = **恰好 4.0s，零丢帧**（`TestGRPCRelayPacedThroughput`）
- 压力守恒：500 帧裸灌，45 送达 + 455 丢弃 = 500，缓冲有界、不阻塞（`TestRelayDropConservationUnderPressure`）

**真实服务冒烟**：gateway(:8080/:9090) + echoworker + wssmoke，200 帧 @25fps（8.2s）：
`frames_in=200 frames_out=200 dropped=0`，逐帧内容一致，worker/网关关闭日志正常。

**浏览器回声**：需人工验证 —— `./bin/gateway` 启动后访问 `http://localhost:8080/echo`，点"开始回声"，
听到自己的声音经网关环路返回即为通过（自动化只能到字节层，听感验证留给部署者）。

## 实施中的关键发现（比结论更有价值）

1. **"吞吐瓶颈"复盘（虚惊一场）**：冒烟最初 200 帧裸灌只回 144 帧，疑似 gRPC 流控饥饿。
   隔离实验（纯 gRPC、无 WS）复现"恰好 40 帧停摆"后定位真相：**不是 bug，是水位丢帧策略在正确工作**——
   灌入速率（瞬时 500 帧）远超消费速率，超出水位（40 帧）的部分按设计被丢弃。
   教训：实时流的测试必须按真实节奏灌帧（25fps），"吞吐下降"要先区分"策略性丢帧"与"系统性阻塞"，
   二者的证据分别是 `frames_dropped_total` 有无对应增长、缓冲是否始终有界。
2. **生命周期时序 bug（已修）**：worker 先于浏览器接入是正常时序（gRPC 先建流），最初实现把它误判为
   "重复房间"而拒绝浏览器。修复：`Session.ClaimBrowser()` 原子认领，浏览器端唯一性由它保证，
   与"worker 先接入"解耦。
3. **测试自身死锁（已修）**：`Consume` 只在"会话关闭且缓冲排空"后返回 false，排干循环前必须先 Close——
   relay 的语义要求使用方正确理解关闭协议，这一点已写入 godoc。

## 与 SPEC 的偏差

| 偏差 | 说明 | 处置 |
|------|------|------|
| proto 方向修正 | SPEC 原稿把 `AudioChunk` 放在 `ClientFrame`（worker→网关），但上行音频实际由网关发往 worker，原稿方向矛盾 | 已修正为按方向命名：`WorkerFrame`（控制/TTS/文本）、`GatewayFrame`（上行音频/终止通知），proto 注释写明方向约定。**下游消费方（backend）以修正版为准** |
| protoc 替代 buf | 生成流程用 protoc + Makefile（`make generate`）等价实现 | M3 若需要 breaking check 再引入 buf |
| WS 上行帧无 seq/timestamp 头 | 客户端发纯字节，seq/timestamp 由网关分配（SPEC 未明确到这一层） | 已定义进 ws.go 协议注释；重连续传的 last_seq 语义在 M2 落地时复核 |
| 鉴权未启用 | `StreamControl.token` 字段已预留，M1 不校验（SPEC 把 JWT 放在 M2） | M2 落地时 WS 握手与控制帧双端校验 |

## 遗留问题

| 级别 | 问题 | 计划 |
|------|------|------|
| P1 | 断线重连无续传：浏览器断开即 `Unbind`，重连是新会话（SPEC 要求 seq 续传） | M2：Redis 注册 + last_seq 语义 |
| P2 | worker 断流后会话无 grace TTL，浏览器侧缓冲静默丢帧 | M2：`--session-grace` 定时器 |
| P2 | `readyz` 恒为 ready，未检查依赖 | M2 随 Redis 接入补齐 |
| P2 | `perf_probe` 压力守恒测试收尾依赖 ctx 超时，拖慢测试 60s | 已知问题，M2 重构会话关闭语义时一并处理 |
| P3 | `/echo` 测试页随二进制分发，生产环境应可关闭 | 加 flag 开关即可，暂缓 |

## 下一步（M2 入口）

1. Redis 注册表实现 `Registry` 接口（接口已稳定，实现即可插拔）
2. seq 续传：WS 握手带 `last_seq`，网关/worker 对齐
3. 优雅 drain：SIGTERM → 摘流 → 通知 worker/浏览器 → 超时强断
4. JWT 校验 + Origin 白名单收敛
5. 压测基线建立（wssmoke 并发版 → loadtest/）
