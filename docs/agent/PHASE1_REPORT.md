# Phase 1 完成报告 — Python Agent 文字面试闭环

> 日期：2026-09-07 ｜ 对应 ROADMAP Week 1（Day 1-7）｜ SPEC 见 [PHASE1_SPEC.md](PHASE1_SPEC.md)

## 结论

**Week 1 目标达成**：Python Agent 可独立运行，通过 HTTP 完成
`创建面试 → AI 提问 → 候选人回答 → AI 评估追问/推进 → 生成报告 → 报告落库可查` 的完整闭环。

端到端验证 29/29 通过（无 LLM Key 的 mock 模式），真实 uvicorn 服务冒烟通过。

## 交付内容

| 模块 | 文件 | 说明 |
|------|------|------|
| 状态定义 | `Agent/state/interview_state.py` | 修复：无注解字段不会成为 LangGraph channel、字段拼写不一致；新增 `deep_dive_count` 支撑深挖上限 |
| LLM 客户端 | `Agent/llm/client.py` | OpenAI 兼容协议 JSON 调用；无 Key/失败返回 None，由节点 mock 兜底并打 `[LLM MOCK]` 日志 |
| 短期记忆 | `Agent/memory/short_term.py` | 滑动窗口（`MAX_CHAT_WINDOW=10` 轮），Prompt 组装时防止 token 膨胀 |
| 长期记忆 | `Agent/memory/long_term.py` | SQLite：`interview_session` + `interview_report` 两张表，服务重启后报告仍可查 |
| 面试图 | `Agent/graph/builder.py` | `prebuild → fetch → [interrupt] → listen → react_judge → (追问/下一题/报告)`，`interrupt_before=["listen"]` 实现 human-in-the-loop |
| 节点 | `Agent/nodes/*.py` | prebuild（LLM 生成题库）、fetch（游标取题）、listen（记录回答）、react_judge（LLM 决策：深挖/下一题/结束）、summary（LLM 报告） |
| API | `Agent/api/routes.py`、`schemas.py` | 三接口 + 404/409 异常路径 |
| 验证 | `Agent/scripts/verify_phase1_loop.py` | 29 项断言的端到端闭环验证脚本，可重复运行 |

## 本轮修复的历史问题（P0/P1）

1. **`config/setting.py` 语法损坏**（P0）：全文被反斜杠转义破坏（`LLM\_API\_KEY`），import 即崩溃 → 重写。
2. **图路由 bug**（P1）：原 `route_func` 中深挖（`loop_ask`）会被 `fetch_base_question` 用下一道基础题覆盖追问内容 → 现由 react_judge 直接生成追问并路由回 `listen`。
3. **实施中新发现并修复**：重写 builder 时遗漏 `listen → react_judge` 边，导致图在 listen 后提前终止（被验证脚本第 2 轮即 409 捕获）。
4. **`pyproject.toml` 构建后端无效**（P1）：`uv.build` 不存在导致 `uv add` 失败 → 改为 `tool.uv package=false` 应用模式。

## 验证记录

**Build / 依赖**

- 命令：`uv sync`
- 实际：Resolved 48 packages，安装成功

**端到端验证**

- 命令：`uv run python scripts/verify_phase1_loop.py`
- 实际：**29/29 通过**，关键断言：
  - start 返回第一题（mock 题库首题）
  - 10 轮 chat（5 基础题 × 每题 1 次深挖追问）全部返回有效追问
  - 第 10 轮后 `finished=true` 且返回报告
  - `GET /report/{room_id}` 与结束时报表逐字段一致
  - 异常路径：重复创建 409、不存在会话 404、不存在报告 404、结束后 chat 409

**真实服务冒烟**

- 命令：`uv run python main.py` + curl start/chat
- 实际：真实 uvicorn 服务返回正确 JSON，`[LLM MOCK]` 日志符合预期

**未验证**

- 真实 LLM 推理路径（本机无 `LLM_API_KEY`）：代码路径已就绪，配置 Key 后重跑同一验证脚本即可
- 多进程/并发会话、长对话 token 压力

## 遗留问题（按优先级）

| 级别 | 问题 | 计划 |
|------|------|------|
| P2 | MemorySaver 为进程内 checkpointer，Agent 重启丢进行中会话 | Week 2 接 Redis/Postgres checkpointer（与 Go Redis 状态打通） |
| P2 | 报告 JSON Schema 未固化（LLM 自由输出 vs mock 键不一致） | Week 4 接 MySQL 前固化为结构化契约 |
| P2 | react_judge 的 LLM 决策与追问生成耦合在一次调用 | Week 2+ 拆分为「评估 → 决策 → 生成」三步，提升可控性 |
| P3 | `asked_base_questions` 与 `chat_history` 信息冗余 | 保留原设计（低成本读取），题库结构化时一并处理 |
| P3 | 日志用 print | 引入结构化 logging 与 Week 7 工程优化一起做 |

## 与 Week 2 的衔接

- Go Backend 的第一个消费场景就是本 API（见 [API.md](API.md)），gRPC 化时保持 `start/chat/report` 三个语义不变
- Redis 状态设计可直接复用 `room_id → (global_phase, conversation 状态)` 的现有模型
