"""离线前置任务：解析简历 + JD，批量生成基础面试题库。"""

from config.setting import MAX_BASE_QUESTIONS
from llm.client import ask_json

GENERATE_POOL_SYSTEM_PROMPT = """你是一个资深技术面试官。根据候选人的简历和岗位描述，生成基础面试题库。
只输出 JSON，格式：{"questions": ["问题1", "问题2", ...]}，不要输出其他内容。"""

# 无 LLM Key / 调用失败时的兜底题库，保证闭环可跑通
MOCK_QUESTION_POOL = [
    "请做一下自我介绍",
    "讲一下 GO 语言的协程 GMP 模型",
    "你项目中的分布式云盘上传是怎么设计的",
    "说说 TCP 滑动窗口与拥塞控制原理",
    "你遇到过的线上内存泄漏问题如何排查？",
]


def prebuild_generate_question_pool(state) -> dict:
    print("【执行节点】生成基础面试题库")
    pool = _generate_pool_by_llm(state.jd, state.resume)
    if pool is None:
        print("[LLM MOCK] 使用内置 mock 题库")
        pool = MOCK_QUESTION_POOL[:MAX_BASE_QUESTIONS]

    return {
        "base_question_pool": pool,
        "pool_cursor": 0,
        "global_phase": "online_interact",  # 状态推进：预生成 → 在线交互
    }


def _generate_pool_by_llm(jd: str, resume: str) -> list[str] | None:
    result = ask_json(
        GENERATE_POOL_SYSTEM_PROMPT,
        f"## 岗位描述\n{jd or '（未提供）'}\n\n## 候选人简历\n{resume or '（未提供）'}\n\n请生成 {MAX_BASE_QUESTIONS} 道题。",
    )
    if not result:
        return None
    questions = result.get("questions")
    if not isinstance(questions, list) or not questions:
        return None
    return [str(q) for q in questions][:MAX_BASE_QUESTIONS]
