from pydantic import BaseModel,Field
from typing import List,Dict


class InterviewState(BaseModel):
    #全局通用字段
    room_id = ""
    jd="" #岗位描述ß
    resume=""#这几个参数是啥意思

    global_phase :str = Field(default = "pre_build" ,description ="pre_build | online_interact")

    #前置离线阶段的数据
    base_question_pool :List[str]=Field(default_factory=list,description="预生成基础面试题库")
    pool_str:int =Field(default=0,description="题库游标，顺序取题")

    #在线实时面试会话数据
    chat_history:List[Dict[str,str]] = Field(default_factory=list)
    current_sub_state:List[Dict[str,str]] =Field(default_factory=list)
    asked_base_questions :List[str] =Field(default_factory=list) #已经问过的基础题流水账本，有点冗余
    deep_dive_questions :List[str] =Field(default_factory=list) #动态深挖追问集合，面试过程中实时生成
    candidate_latest_answer:str ="" #这个说的是候选人最后的回答吗

    # ReAct判断输出印记（路由依据
    react_action: str =Field(default="",description="loop_ask / next_stage /finish") #这里是不同阶段的结束不一样吗有的阶段是 finish
    need_deep :bool =Field(default =False)

    #输出内容
    current_output_question:str=""
    evaluation_memo :List[str] =Field(default_factory=list)
    final_report:Dict=Field(default_factory=Dict)