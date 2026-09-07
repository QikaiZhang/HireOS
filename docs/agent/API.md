# Agent HTTP API 契约（v0.1）

> Phase 1（文字面试）接口契约。Week 2 的 Go Backend 将作为本 API 的第一个消费者，
> 契约变更需同步更新本文档并通知下游。

Base URL: `http://localhost:8000`（uvicorn 默认）

通用约定：

- Content-Type: `application/json`
- `room_id` 是全局会话标识，同时作为 LangGraph 的 `thread_id`
- LLM 未配置 Key 时服务以 mock 模式运行（响应结构不变，见「mock 模式」）

---

## POST /interview/start

创建面试会话。Agent 生成题库并返回第一个问题。

**请求**

```json
{
  "room_id": "room-abc-123",
  "jd": "Go 后端开发工程师，熟悉分布式系统",
  "resume": "3 年 Go 后端经验，做过分布式存储"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| room_id | string | 是 | 面试房间 ID，全局唯一 |
| jd | string | 否 | 岗位描述 |
| resume | string | 否 | 候选人简历（纯文本） |

**响应 200**

```json
{
  "room_id": "room-abc-123",
  "global_phase": "online_interact",
  "question": "请做一下自我介绍",
  "finished": false
}
```

**错误**

| 状态码 | 场景 |
|--------|------|
| 409 | `room_id` 已存在（会话已创建） |

---

## POST /interview/chat

提交候选人回答。Agent 评估后返回追问/下一题；面试结束时返回报告。

**请求**

```json
{
  "room_id": "room-abc-123",
  "message": "我做过一个分布式云盘项目，上传采用分片…"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| room_id | string | 是 | 面试房间 ID |
| message | string | 是 | 候选人本轮回答 |

**响应 200（面试继续）**

```json
{
  "room_id": "room-abc-123",
  "global_phase": "online_interact",
  "question": "关于「请做一下自我介绍…」你刚才提到…，能结合具体场景再深入讲讲吗？",
  "finished": false
}
```

**响应 200（面试结束）**

```json
{
  "room_id": "room-abc-123",
  "global_phase": "end",
  "question": null,
  "finished": true,
  "report": {
    "total_score": 75,
    "recommendation": "是",
    "advantage": "…",
    "shortcoming": "…",
    "summary": "…"
  }
}
```

> `report` 的字段由 LLM 输出，mock 模式下的键为
> `{是否进入下一轮, total_score, advantage, shortcoming}`。Week 4 接 MySQL 前
> 需要把报告 Schema 固化为结构化契约（P2 遗留项，见 PHASE1_REPORT）。

**错误**

| 状态码 | 场景 |
|--------|------|
| 404 | `room_id` 不存在 |
| 409 | 面试已结束，无法继续对话 |

---

## GET /report/{room_id}

查询已生成的面试报告（SQLite 持久化，服务重启后仍可查）。

**响应 200**

```json
{
  "room_id": "room-abc-123",
  "report": { "total_score": 75, "...": "..." }
}
```

**错误**

| 状态码 | 场景 |
|--------|------|
| 404 | 报告不存在或面试未结束 |

---

## mock 模式

未配置 `LLM_API_KEY` 时：

- 题库使用内置 5 道 mock 题
- 每道基础题深挖一次后推进（确定性策略，便于端到端测试）
- 报告为固定 mock 报告
- 每次兜底都会输出 `[LLM MOCK]` 日志，**不要把 mock 输出当作真实推理结果**

配置方式：复制 `Agent/.env.example` 为 `Agent/.env` 并填写 `LLM_API_KEY`。

---

## 内部行为（供 Go Backend 联调参考）

- 会话状态由 LangGraph MemorySaver（进程内）管理，**Agent 进程重启后进行中的会话丢失**（P2 遗留项）
- 面试循环：`prebuild → fetch → [等回答] → listen → react_judge → (追问 | 下一题 | 报告)`
- 单道基础题最多深挖 `MAX_DEEP_DIVE_PER_QUESTION=2` 次
- 图结构见 `Agent/graph/builder.py`
