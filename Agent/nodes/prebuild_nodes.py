
from state.interview_state import  InterviewState

def prebuild_generate_question_pool(state:InterviewState)->InterviewState:
        """
        离线前置任务：解析简历+JD，批量生成基础面试题库
        Day1 模拟固定题库，后续接入LLM
        """
        mock_pool=[
            "请做一下自我介绍",
            "讲一下 GO语言的协程 GMP 模型",
            "你项目中的分布式云盘上传是怎么设计的",
            "说说TCP滑动窗口与拥塞控制原理",
            "你遇到过的线上内存泄漏问题如何排查？"
        ]

        state.base_question_pool =mock_pool
        state.pool_str=0
        state.global_phase="online_interact" #算是一步状态推动
        return state