"""面试状态图构建。

交互模型：interrupt_before=["listen"]，图每次执行到 listen 前暂停，
等待 /interview/chat 注入候选人回答后恢复 —— 即「提问 → 等回答」的人机循环。
"""

from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import StateGraph, END

from nodes.online_nodes import (
    fetch_base_question_node,
    listen_answer_node,
    react_judge_node,
    route_after_fetch,
    route_after_react,
    summary_node,
)
from nodes.prebuild_nodes import prebuild_generate_question_pool
from state.interview_state import InterviewState


def build_graph(checkpointer=None):
    graph = StateGraph(InterviewState)

    # 1. 节点与函数绑定
    graph.add_node("prebuild", prebuild_generate_question_pool)
    graph.add_node("fetch_base_question", fetch_base_question_node)
    graph.add_node("listen", listen_answer_node)
    graph.add_node("react_judge", react_judge_node)
    graph.add_node("summary", summary_node)

    # 2. 基础流转边
    graph.set_entry_point("prebuild")
    graph.add_edge("prebuild", "fetch_base_question")
    graph.add_edge("listen", "react_judge")

    # 3. 核心条件路由
    graph.add_conditional_edges(
        source="fetch_base_question",
        path=route_after_fetch,
        path_map={"listen": "listen", "summary": "summary"},
    )
    graph.add_conditional_edges(
        source="react_judge",
        path=route_after_react,
        path_map={
            "listen": "listen",              # 深挖追问，再次中断等回答
            "fetch_base_question": "fetch_base_question",  # 推进下一题
            "summary": "summary",
        },
    )
    graph.add_edge("summary", END)

    # 4. 编译：listen 前中断，等待候选人回答（human-in-the-loop）
    compiled_graph = graph.compile(
        checkpointer=checkpointer or MemorySaver(),
        interrupt_before=["listen"],
    )
    return compiled_graph
