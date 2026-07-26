# Python Agent 核心设计

> Agent 是 HireOS 的大脑。它不是一个 `while True: input(); call_llm()` 的聊天机器人，而是一个**有状态的面试状态机**。

---

## 目录结构

```
agent-service/
├── app/
│   ├── __init__.py
│   └── config.py           # 配置管理
│
├── graph/
│   ├── __init__.py
│   ├── interview_graph.py  # LangGraph 状态图
│   └── state.py            # 状态定义
│
├── nodes/
│   ├── __init__.py
│   ├── interviewer.py      # 提问节点
│   ├── evaluator.py        # 回答评估节点
│   └── summarizer.py       # 报告生成节点
│
├── memory/
│   ├── __init__.py
│   ├── short_term.py       # 短期记忆（对话上下文）
│   └── long_term.py        # 长期记忆（候选人画像）
│
├── prompt/
│   ├── __init__.py
│   ├── interviewer.py      # 面试官 Prompt
│   ├── evaluator.py        # 评估 Prompt
│   └── summarizer.py       # 总结 Prompt
│
├── api/
│   ├── __init__.py
│   ├── routes.py           # FastAPI 路由
│   └── schemas.py          # Pydantic 模型
│
└── main.py                 # 入口
```

---

## 状态定义

```python
from typing import TypedDict, List, Dict, Any


class InterviewState(TypedDict):
    """面试核心状态 — 所有节点共享和更新这个状态"""

    # 面试阶段
    stage: str  # INIT | INTRO | TECH | FOLLOW_UP | SUMMARY | END

    # 对话历史
    messages: List[Dict[str, str]]  # [{"role": "interviewer", "content": "..."}, ...]

    # 输入上下文
    resume: Dict[str, Any]  # 解析后的简历
    jd: Dict[str, Any]      # 职位描述

    # 面试进度
    asked_questions: List[str]      # 已问问题列表
    current_question: str           # 当前问题
    current_topic: str              # 当前考察方向

    # 评估数据
    score: Dict[str, Any]   # 各维度评分
    notes: List[str]        # 面试官笔记
```

---

## 面试状态机

### 图结构

```text
                ┌─────────┐
                │   INIT  │  加载简历和 JD
                └────┬────┘
                     │
                     ▼
                ┌─────────┐
                │  INTRO  │  开场暖场问题
                └────┬────┘
                     │
                     ▼
          ┌──────────────────┐
          │      TECH        │  技术考察
          │  (核心循环节点)   │◄─────────────┐
          └────────┬─────────┘               │
                   │                          │
          ┌────────┴──────────┐              │
          ▼                   ▼              │
   ┌────────────┐      ┌────────────┐       │
   │  ANALYZE   │      │ FOLLOW_UP  │       │
   │  评估回答  │──────►│  决定追问  │───┐   │
   └────────────┘      └────────────┘   │   │
                                        │   │
                          ┌─────────────┘   │
                          │ 需要追问        │
                          └─────────────────┘
                          │ 不需要追问
                          ▼
                   ┌────────────┐
                   │  SUMMARY   │  生成面试报告
                   └────────────┘
```

### 状态转换逻辑

```python
# interview_graph.py

from langgraph.graph import StateGraph, END
from nodes.interviewer import init_node, question_node, follow_up_node
from nodes.evaluator import analyze_node
from nodes.summarizer import summary_node


def build_interview_graph() -> StateGraph:
    graph = StateGraph(InterviewState)

    # 添加节点
    graph.add_node("init", init_node)
    graph.add_node("question", question_node)
    graph.add_node("analyze", analyze_node)
    graph.add_node("follow_up", follow_up_node)
    graph.add_node("summary", summary_node)

    # 设置入口
    graph.set_entry_point("init")

    # 定义边
    graph.add_edge("init", "question")
    graph.add_edge("question", "analyze")
    graph.add_conditional_edges(
        "analyze",
        route_after_analyze,   # 路由函数
        {
            "follow_up": "follow_up",
            "summary": "summary",
        }
    )
    graph.add_edge("follow_up", "question")  # 追问后回到问题
    graph.add_edge("summary", END)

    return graph.compile()


def route_after_analyze(state: InterviewState) -> str:
    """决定下一步：继续追问还是生成报告"""
    # 条件 1：已问问题数达到上限
    if len(state["asked_questions"]) >= state.get("max_questions", 10):
        return "summary"

    # 条件 2：评估分数已足够
    # 条件 3：关键领域已覆盖
    return "follow_up"
```

---

## 节点详解

### 1. INIT Node — 初始化

```text
职责：加载候选人简历和职位描述，生成面试大纲

输入：resume_id, jd_id
处理：解析简历 → 提取技能点 → 匹配 JD 要求 → 生成考察计划
输出：state.resume, state.jd, state.asked_questions = []
```

```python
async def init_node(state: InterviewState) -> InterviewState:
    """初始化面试：加载简历和 JD，生成考察计划"""
    resume = await load_resume(state["resume_id"])
    jd = await load_jd(state["jd_id"])

    # 提取候选人的关键技能
    skills = extract_skills(resume)
    # 从 JD 中识别重点考察方向
    focus_areas = match_jd_requirements(jd, skills)

    return {
        **state,
        "stage": "INTRO",
        "resume": resume,
        "jd": jd,
        "focus_areas": focus_areas,
        "asked_questions": [],
        "messages": [],
        "score": {},
    }
```

### 2. Question Node — 生成问题

```text
职责：根据当前上下文生成下一个面试问题

输入：state.resume, state.jd, state.focus_areas, state.asked_questions
处理：
  1. 选择当前考察方向
  2. 结合候选人背景定制问题
  3. 避免重复已问问题
输出：state.current_question, state.messages += [new_question]
```

```python
async def question_node(state: InterviewState) -> InterviewState:
    """生成面试问题"""
    stage = state["stage"]

    if stage == "INTRO":
        # 暖场：开放性问题
        prompt = build_intro_prompt(state["resume"])
    elif stage == "TECH":
        # 技术考察：结合简历和 JD
        prompt = build_tech_question_prompt(
            resume=state["resume"],
            jd=state["jd"],
            focus_areas=state["focus_areas"],
            asked_questions=state["asked_questions"],
            current_topic=state.get("current_topic"),
        )
    else:
        prompt = build_general_prompt(state)

    question = await llm.generate(prompt)

    return {
        **state,
        "current_question": question,
        "asked_questions": state["asked_questions"] + [question],
        "messages": state["messages"] + [{"role": "interviewer", "content": question}],
    }
```

### 3. Analyze Node — 分析回答

```text
职责：评估候选人的回答质量

输入：state.current_question, 用户回答
处理：
  1. 判断回答是否切题
  2. 评估技术深度
  3. 识别亮点和不足
  4. 更新评分
输出：state.notes, state.score（增量更新）
```

```python
async def analyze_node(state: InterviewState, user_answer: str) -> InterviewState:
    """分析候选人的回答"""
    prompt = build_analysis_prompt(
        question=state["current_question"],
        answer=user_answer,
        jd=state["jd"],
    )

    analysis = await llm.generate(prompt)
    # analysis: {
    #   "relevance": 0.8,       # 切题度
    #   "depth": 0.6,           # 深度
    #   "highlights": [...],    # 亮点
    #   "weaknesses": [...],    # 不足
    #   "follow_up_suggestions": [...]
    # }

    return {
        **state,
        "messages": state["messages"] + [{"role": "candidate", "content": user_answer}],
        "notes": state["notes"] + [analysis],
        "score": update_score(state["score"], analysis),
        "last_analysis": analysis,
    }
```

### 4. Follow Up Node — 追问决策

```text
职责：决定是否追问，以及追问方向

输入：state.last_analysis
处理：
  1. 回答深度不够 → 深挖追问
  2. 回答模糊 → 澄清追问
  3. 发现疑点 → 挑战追问
  4. 回答充分 → 进入下一领域
输出：state.current_topic, state.stage
```

```python
async def follow_up_node(state: InterviewState) -> InterviewState:
    """决定追问策略"""
    analysis = state["last_analysis"]

    prompt = build_follow_up_decision_prompt(
        analysis=analysis,
        remaining_topics=get_remaining_topics(state),
    )

    decision = await llm.generate(prompt)
    # decision: {
    #   "should_follow_up": true/false,
    #   "direction": "deepen" | "clarify" | "challenge" | "next_topic",
    #   "target_topic": "...",
    # }

    if decision["should_follow_up"]:
        return {
            **state,
            "stage": "FOLLOW_UP",
            "current_topic": decision["target_topic"],
        }
    else:
        return {
            **state,
            "stage": "TECH",
            "current_topic": decision["next_topic"],
        }
```

### 5. Summary Node — 生成报告

```text
职责：汇总面试全过程，生成评估报告

输入：state.messages, state.score, state.notes, state.resume, state.jd
处理：
  1. 总结每个考察维度的表现
  2. 给出综合评分
  3. 列出优点和待提升方向
  4. 给出录用建议
输出：最终报告 + state.stage = "END"
```

```python
async def summary_node(state: InterviewState) -> InterviewState:
    """生成面试报告"""
    prompt = build_summary_prompt(
        messages=state["messages"],
        score=state["score"],
        notes=state["notes"],
        resume=state["resume"],
        jd=state["jd"],
    )

    report = await llm.generate(prompt)
    # report: {
    #   "overall_score": 85,
    #   "dimensions": {
    #     "technical": {"score": 80, "comment": "..."},
    #     "communication": {"score": 90, "comment": "..."},
    #     "problem_solving": {"score": 85, "comment": "..."},
    #   },
    #   "strengths": [...],
    #   "weaknesses": [...],
    #   "recommendation": "hire" | "reject" | "next_round",
    #   "summary": "...",
    # }

    return {
        **state,
        "stage": "END",
        "report": report,
    }
```

---

## 记忆系统

### 短期记忆（对话上下文）

在 `state.messages` 中维护完整对话历史。每次 LLM 调用时携带最近 N 轮对话。

```python
# memory/short_term.py

class ShortTermMemory:
    """滑动窗口对话记忆"""

    def __init__(self, window_size: int = 10):
        self.window_size = window_size
        self.messages: List[Dict] = []

    def add(self, role: str, content: str):
        self.messages.append({"role": role, "content": content})

    def get_context(self) -> List[Dict]:
        """返回最近 N 条消息"""
        return self.messages[-self.window_size * 2:]  # 提问+回答 = 2条/轮
```

### 长期记忆（候选人画像）

随着面试进行，动态更新对候选人的认知。

```json
{
  "candidate": "zhang",
  "skills": {
    "confirmed": ["golang", "redis"],
    "exploring": ["distributed systems", "k8s"]
  },
  "weaknesses": ["分布式系统设计经验不足"],
  "interview_progress": {
    "topics_covered": ["编程语言", "缓存设计"],
    "topics_remaining": ["系统设计", "项目经验"]
  }
}
```

```python
# memory/long_term.py

class LongTermMemory:
    """候选人画像 — 随着面试逐步构建"""

    def __init__(self, storage: "StorageBackend"):
        self.storage = storage  # SQLite / Redis / MySQL

    async def update_skill(self, candidate_id: str, skill: str,
                           confidence: float):
        ...

    async def get_profile(self, candidate_id: str) -> dict:
        ...
```

---

## API 设计

```python
# api/routes.py

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()


class StartInterviewRequest(BaseModel):
    resume_id: str
    jd_id: str
    candidate_name: str


class ChatRequest(BaseModel):
    session_id: str
    message: str  # 候选人的回答（文字）


class ReportResponse(BaseModel):
    session_id: str
    overall_score: float
    dimensions: dict
    strengths: list
    weaknesses: list
    recommendation: str
    summary: str


@app.post("/interview/start")
async def start_interview(req: StartInterviewRequest):
    """创建新面试会话，Agent 返回第一个问题"""
    ...


@app.post("/interview/chat")
async def chat(req: ChatRequest):
    """发送候选人回答，Agent 返回下一个问题或结束"""
    ...


@app.get("/report/{session_id}")
async def get_report(session_id: str):
    """查询面试报告"""
    ...
```

---

## Prompt 设计原则

1. **角色明确**：每个 Prompt 有清晰的 Persona（面试官、评估者、总结者）
2. **结构化输出**：使用 JSON 格式约束 LLM 输出，便于程序处理
3. **上下文精简**：只传必要信息，避免 token 浪费
4. **模板分离**：Prompt 模板统一在 `prompt/` 目录管理，方便调优

```python
# prompt/interviewer.py

TECH_QUESTION_PROMPT = """你是一个资深技术面试官。

## 候选人背景
{resume_summary}

## 职位要求
{jd_summary}

## 当前考察方向
{current_topic}

## 已问问题
{asked_questions}

## 要求
1. 生成一个开放性的技术问题，考察候选人在 {current_topic} 上的深度
2. 问题应该结合候选人的实际项目经验
3. 不要问"是什么"类问题，要问"为什么"、"怎么做"、"如果...会怎样"
4. 问题长度不超过 100 字

请返回 JSON：
{{"question": "...", "expected_depth": "shallow|medium|deep", "follow_up_hints": [...]}}
"""
```
