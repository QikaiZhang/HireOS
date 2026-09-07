from typing import Dict, List

from pydantic import BaseModel, Field


class InterviewState(BaseModel):
    """面试核心状态 — 所有节点共享，节点只返回增量更新 dict"""

    # 全局通用字段
    room_id: str = ""
    jd: str = ""      # 岗位描述
    resume: str = ""  # 候选人简历（MVP 阶段为纯文本）

    global_phase: str = Field(default="pre_build", description="pre_build | online_interact | end")

    # 前置离线阶段的数据
    base_question_pool: List[str] = Field(default_factory=list, description="预生成基础面试题库")
    pool_cursor: int = Field(default=0, description="题库游标，顺序取题")

    # 在线实时面试会话数据
    chat_history: List[Dict[str, str]] = Field(default_factory=list)
    asked_base_questions: List[str] = Field(default_factory=list)  # 已问过的基础题流水账本
    deep_dive_questions: List[str] = Field(default_factory=list)   # 动态深挖追问集合，面试过程中实时生成
    deep_dive_count: int = Field(default=0, description="当前题目的深挖次数，配合 MAX_DEEP_DIVE_PER_QUESTION 防无限追问")
    candidate_latest_answer: str = ""  # 候选人最近一次回答（由 /interview/chat 注入）

    # ReAct 判断输出印记（路由依据）
    react_action: str = Field(default="", description="loop_ask / next_stage / finish")
    need_deep: bool = Field(default=False)

    # 输出内容
    current_output_question: str = ""
    evaluation_memo: List[str] = Field(default_factory=list)  # 每轮回答的评估要点，供 summary 汇总
    final_report: Dict = Field(default_factory=dict)
