"""短期记忆：滑动窗口对话上下文。"""

from config.setting import MAX_CHAT_WINDOW


def get_context(chat_history: list[dict], max_rounds: int = MAX_CHAT_WINDOW) -> list[dict]:
    """返回最近 N 轮对话（提问 + 回答 = 2 条/轮），供 Prompt 组装，防止 token 无限膨胀。"""
    return chat_history[-max_rounds * 2:]
