
from langgraph.graph import StateGraph, END
from state.interview_state import InterviewState

from nodes.online_nodes import (
    fetch_base_question_node,
    listen_answer_node,
    react_judge_node,
    summary_node
)


def route_func(state:InterviewState) ->str:
    action = state.react_action
    if action == "loop_ask":
        return "fetch_base_question"
    elif action == "next_stage":
        return "fetch_base_question"
    elif action == "finish":
        return "summary"
    return "fetch_base_question"

def build_graph():
    #1.初始化所有节点
    graph = StateGraph(InterviewState) #这个东西是 langgraph提供的

    #2.初始化所有节点
    graph.add_node("fetch_base_question",fetch_base_question_node) #节点和函数绑定
    graph.add_node("listen",listen_answer_node)
    graph.add_node("react_judge",react_judge_node)
    graph.add_node("summary",summary_node)

    #3.基础流转边
    graph.add_edge("fetch_base_question","listen")
    graph.add_edge("listen","react_judge")

    #核心条件路由分支
    graph.add_conditional_edges(
        source= "react_judge",
        path = route_func,
        path_map={
        "fetch_base_question": "fetch_base_question",
        "summary": "summary"
    }
    )

    graph.add_edge("summary",END)

    graph.set_entry_point("fetch_base_question")

    compiled_graph = graph.compile()
    return compiled_graph