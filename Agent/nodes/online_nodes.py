

#在线所有 graph 的空节点

from state.interview_state import  InterviewState

#统一接受state字典，返回增量更新字典，无 LLM调用，只是打印日志
def fetch_base_question_node(state:InterviewState) ->dict:
    print("【执行节点】获取基础面试题")
    pool = state.base_question_pool
    ptr = state.pool_str

    if ptr >= len(pool):
        #结束逻辑
        return {"current_output_question":""}

    q = pool[ptr]
    return {
        "pool_str":ptr+1,
        "current_output_quesiton":q,
        "asked_base_question":state.asked_base_questions + [q] #拼接 list?
    }

# ReAct思考判断核心节点(空壳，后续接入 LLM做思考）

def react_judge_node(state:InterviewState) ->dict:
    print("【执行节点】ReAct思考判断下一步动作")
    # 模拟判断：暂时不需要深挖，继续下一题
    return {
        "react_action": "next_stage",
        "need_deep": False,
        "deep_question": ""
    }

def summary_node(state: InterviewState) ->dict:
    mock_report = {
        "是否进入下一轮":False,
        "total_score": 75,
        "advantage": "熟悉Go后端与网络底层",
        "shortcoming": "分布式事务理解不足"
    }
    return {"final_report": mock_report}


# 2. 接收ASR转写的候选人回答
def listen_answer_node(state: InterviewState) -> dict:
    print("【执行节点】接收候选人语音回答")
    # 模拟ASR返回文本
    mock_ans = "我过往项目使用Go开发分布式存储，理解TCP协议栈底层细节"
    new_history = state.chat_history + [
        {"role": "candidate", "content": mock_ans}
    ]
    return {
        "candidate_latest_answer": mock_ans,
        "chat_history": new_history
    }