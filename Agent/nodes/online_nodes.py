"""在线交互节点：取题 → 听答 → ReAct 判断 → 总结。"""

from config.setting import MAX_DEEP_DIVE_PER_QUESTION
from llm.client import ask_json
from memory.short_term import get_context


def fetch_base_question_node(state) -> dict:
    """按游标从题库顺序取题；题库取空时返回空问题，由路由边转入 summary。"""
    print("【执行节点】获取基础面试题")
    pool = state.base_question_pool
    cursor = state.pool_cursor

    if cursor >= len(pool):
        return {"current_output_question": ""}

    question = pool[cursor]
    return {
        "pool_cursor": cursor + 1,
        "current_output_question": question,
        "asked_base_questions": state.asked_base_questions + [question],
        "deep_dive_count": 0,  # 新题重置深挖计数
    }


def route_after_fetch(state) -> str:
    return "summary" if not state.current_output_question else "listen"


def listen_answer_node(state) -> dict:
    """记录候选人最近一次回答（文字模式由 API 注入，语音模式后续由 ASR 注入）。"""
    print(f"【执行节点】接收候选人回答: {state.candidate_latest_answer[:30]}...")
    new_history = state.chat_history + [
        {"role": "interviewer", "content": state.current_output_question},
        {"role": "candidate", "content": state.candidate_latest_answer},
    ]
    return {"chat_history": new_history}


REACT_JUDGE_SYSTEM_PROMPT = """你是一个资深技术面试官的评估大脑。根据岗位要求、当前问题、候选人回答和历史对话，决定下一步动作：
- "loop_ask": 回答不够深入或存疑，需要深挖追问（deep_question 给出追问内容）
- "next_stage": 回答合格，进入下一道题
- "finish": 面试信息已足够（或题目已问完），生成报告

只输出 JSON：{"action": "loop_ask|next_stage|finish", "deep_question": "追问内容或空串", "memo": "一句话评估"}"""


def react_judge_node(state) -> dict:
    """ReAct 核心决策：评估回答并决定 深挖 / 下一题 / 结束。"""
    print("【执行节点】ReAct 思考判断下一步动作")
    question = state.current_output_question
    answer = state.candidate_latest_answer

    decision = _judge_by_llm(state, question, answer)
    if decision is None:
        print("[LLM MOCK] 使用 mock 决策：每道基础题深挖一次后推进")
        decision = _mock_decision(state, question, answer)

    action = decision["action"]
    memo = decision.get("memo", "")

    updates = {
        "evaluation_memo": state.evaluation_memo + [memo],
        "need_deep": action == "loop_ask",
    }

    if action == "loop_ask":
        deep_question = decision.get("deep_question", "")
        if not deep_question:
            # 决策要深挖但没给出追问内容，退化为推进下一题
            return {**updates, "react_action": "next_stage"}
        return {
            **updates,
            "react_action": "loop_ask",
            "current_output_question": deep_question,  # 追问直接作为下一轮输出，等待听答
            "deep_dive_questions": state.deep_dive_questions + [deep_question],
            "deep_dive_count": state.deep_dive_count + 1,
        }

    return {**updates, "react_action": action}


def route_after_react(state) -> str:
    if state.react_action == "loop_ask":
        return "listen"
    if state.react_action == "finish":
        return "summary"
    return "fetch_base_question"


def _judge_by_llm(state, question: str, answer: str) -> dict | None:
    context = "\n".join(f"{m['role']}: {m['content']}" for m in get_context(state.chat_history))
    result = ask_json(
        REACT_JUDGE_SYSTEM_PROMPT,
        f"""## 岗位描述
{state.jd or '（未提供）'}

## 历史对话（最近若干轮）
{context or '（无）'}

## 当前问题
{question}

## 候选人回答
{answer}

## 约束
- 当前题目已深挖 {state.deep_dive_count} 次，上限 {MAX_DEEP_DIVE_PER_QUESTION} 次，达到上限不允许再 loop_ask
- 已问过的基础题数：{len(state.asked_base_questions)}""",
    )
    if not result or result.get("action") not in ("loop_ask", "next_stage", "finish"):
        return None
    # 强制执行深挖上限
    if result["action"] == "loop_ask" and state.deep_dive_count >= MAX_DEEP_DIVE_PER_QUESTION:
        result["action"] = "next_stage"
    return result


def _mock_decision(state, question: str, answer: str) -> dict:
    memo = f"问题「{question[:20]}…」回答切题，深度一般"
    # mock 策略：每道基础题（deep_dive_count==0）深挖一次，之后推进
    if state.deep_dive_count == 0:
        return {
            "action": "loop_ask",
            "deep_question": f"关于「{question[:20]}…」你刚才提到「{answer[:20]}…」，能结合具体场景再深入讲讲吗？",
            "memo": memo,
        }
    return {"action": "next_stage", "deep_question": "", "memo": memo}


SUMMARY_SYSTEM_PROMPT = """你是一个面试评估总结官。根据完整对话历史和每轮评估要点，生成面试评估报告。
只输出 JSON：{"total_score": 0-100整数, "recommendation": "是/否", "advantage": "...", "shortcoming": "...", "summary": "两三句话总评"}"""


def summary_node(state) -> dict:
    print("【执行节点】生成面试报告")
    report = _summary_by_llm(state)
    if report is None:
        print("[LLM MOCK] 使用 mock 报告")
        report = {
            "是否进入下一轮": False,
            "total_score": 75,
            "advantage": "熟悉 Go 后端与网络底层",
            "shortcoming": "分布式事务理解不足",
        }
    return {
        "final_report": report,
        "global_phase": "end",
    }


def _summary_by_llm(state) -> dict | None:
    context = "\n".join(f"{m['role']}: {m['content']}" for m in get_context(state.chat_history))
    memos = "\n".join(f"- {m}" for m in state.evaluation_memo)
    result = ask_json(
        SUMMARY_SYSTEM_PROMPT,
        f"""## 岗位描述
{state.jd or '（未提供）'}

## 完整对话（滑动窗口）
{context}

## 每轮评估要点
{memos or '（无）'}""",
    )
    if not result or "total_score" not in result:
        return None
    return result
