# SPEC: Phase 1 — Python Agent 文字面试闭环（Week 1 收口）

> 状态：已实施 | 日期：2026-09-07 | 工作流：NORMAL

## Background

Week 1 目标是「Agent 大脑 + HTTP API 化」：`打开会话 → AI 提问 → 候选人回答 → AI 追问/推进 → 生成报告 → 报告可查`。

当前代码只完成了 LangGraph 骨架（4 个 mock 节点），存在以下缺口：

| 缺口 | 位置 | 问题 |
|------|------|------|
| 配置文件损坏 | `config/setting.py` | 内容被转义符破坏（`LLM\_API\_KEY`），import 即语法错误 |
| State 字段未注册 | `state/interview_state.py` | `room_id`/`jd`/`resume` 等无类型注解，只是类属性不是 pydantic 字段，LangGraph 不会为其建 channel；另有字段拼写不一致（`current_output_quesiton`） |
| 节点全是 mock | `nodes/*.py` | 无任何 LLM 调用；`prebuild` 节点未接入图 |
| 图流转 bug | `graph/builder.py` | 深挖问题（deep question）生成后会被 `fetch_base_question` 覆盖丢弃；`loop_ask` 路由错误 |
| Memory 缺失 | — | 无短期滑动窗口、无长期持久化 |
| API 缺失 | `main.py` | 仍是 Hello World，无 FastAPI 三个接口 |
| 依赖为空 | `pyproject.toml` | `dependencies = []` |

## Goal

打通 **输入 → 状态 → 推理 → 输出 → 持久化** 的文字面试闭环，可通过 HTTP 独立运行：

1. `POST /interview/start` — 创建面试，Agent 生成题库并返回第一题
2. `POST /interview/chat` — 提交回答，Agent 评估后追问或推进，返回下一题
3. `GET /report/{room_id}` — 面试结束后可查询评分报告

## Non-Goals

- 语音链路（ASR/TTS/WebSocket）— Week 3
- Go Backend / gRPC — Week 2
- MySQL / Redis — Week 2/4（当前用进程内 checkpointer + SQLite 演示持久化边界）
- 向量库（QDRANT 配置保留但不接逻辑）
- Prompt 精细化调优

## Proposed Change

### 图结构（修正后）

```text
prebuild（LLM 生成基础题库，失败降级 mock 题库）
   ↓
fetch_base_question（按游标取题，取完 → summary）
   ⇢ [interrupt]  ← API 返回 current_output_question 给候选人
   ↓
listen（记录回答到 chat_history）
   ↓
react_judge（LLM 决策：深挖追问 / 下一题 / 结束；深挖时直接生成追问）
   ↓
   ├─ loop_ask   → listen（再次中断等待回答）   ← 修复点：追问不再被覆盖
   ├─ next_stage → fetch_base_question
   └─ finish     → summary（LLM 生成报告，失败降级 mock 报告）
```

`interrupt_before=["listen"]` 是人机交互边界：每次图执行到 listen 前暂停，等 `/interview/chat` 注入候选人回答后恢复。

### 关键设计决策

| 决策 | 选择 | 理由 | 备选 | 影响 |
|------|------|------|------|------|
| 状态 Schema | pydantic BaseModel，全字段显式注解 | LangGraph 需要为每个字段建立 channel；沿用现有风格 | TypedDict | 无 |
| LLM 客户端 | OpenAI 兼容 SDK + **无 Key 时 mock 兜底** | 本地无 LLM_API_KEY 也能完整验证闭环；上线只改 env | 强依赖 Key | 兜底路径必须显式打日志，防止误判为真实推理 |
| 人机交互 | `interrupt_before=["listen"]` + MemorySaver checkpointer | LangGraph 原生 human-in-the-loop，session 状态由 checkpointer 管理 | 自维护 dict | 进程重启丢状态 → Week 2 换 Redis/Postgres checkpointer |
| 长期持久化 | SQLite（stdlib）存 session 与 report | 演示持久化边界，零部署成本 | MySQL | Week 4 换 MySQL |
| 追问上限 | `MAX_DEEP_DIVE_PER_QUESTION=2` | 防止无限追问（复用已有常量） | — | 无 |

## Change Scope

### Modify

- `Agent/config/setting.py`（修复 + 新增 DB_PATH）
- `Agent/state/interview_state.py`（修字段）
- `Agent/graph/builder.py`（prebuild 接入 + 中断 + 路由修复）
- `Agent/nodes/prebuild_nodes.py`、`Agent/nodes/online_nodes.py`（LLM + 兜底）
- `Agent/main.py`（uvicorn 入口）
- `Agent/pyproject.toml`（依赖）

### Add

- `Agent/llm/client.py`（LLM JSON 调用封装）
- `Agent/memory/short_term.py`（滑动窗口）
- `Agent/memory/long_term.py`（SQLite 持久化）
- `Agent/api/routes.py`、`Agent/api/schemas.py`（FastAPI）
- `Agent/scripts/verify_phase1_loop.py`（端到端验证脚本）
- `Agent/.env.example`

### Do Not Modify

- `docs/` 已有设计文档（AGENT_DESIGN.md 保留为目标态参考）
- `Agent/tools/vector_store.py`（空文件，后续阶段使用）

## Minimal Verifiable Slice

```text
start(jd+resume) → LLM 题库 → 第 1 题
chat(回答) → 评估 → 深挖/下一题 → … → 题库耗尽 → LLM 报告
GET /report/{room_id} → 与生成报告一致
```

验证方式：`uv run python scripts/verify_phase1_loop.py`（TestClient 走完整 HTTP 流程，mock LLM 模式，断言每一轮返回与最终报告落库）。

## Risks And Compatibility

| 级别 | 风险 | 处置 |
|------|------|------|
| P1 | 无 LLM Key 时误把 mock 当真实推理输出 | 兜底时显式打 `[LLM MOCK]` 日志；API 响应不含任何"已评估"误导语义 |
| P2 | MemorySaver 进程重启丢会话 | 记录到 PHASE1_REPORT 遗留项，Week 2 迁移 |
| P2 | 同步 OpenAI 调用阻塞事件循环 | 路由用 `def`（FastAPI 自动跑线程池），MVP 可接受 |
| P3 | `asked_base_questions` 与 `chat_history` 信息有冗余 | 保留，供 react_judge 低成本读取（原注释已标注"有点冗余"） |

## Acceptance Criteria

- [ ] 三个 HTTP 接口可用，完整跑通 start → N 轮 chat → report
- [ ] 有 LLM Key 时走真实 LLM，无 Key 时 mock 兜底闭环依然完整
- [ ] session 与 report 落 SQLite，`GET /report` 可查
- [ ] 验证脚本全部断言通过，输出留档至 PHASE1_REPORT.md
