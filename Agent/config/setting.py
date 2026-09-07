import os

from dotenv import load_dotenv

load_dotenv()

# 大模型配置（OpenAI 兼容协议，例如 DashScope qwen 系列）
LLM_API_KEY = os.getenv("LLM_API_KEY", "")
LLM_BASE_URL = os.getenv("LLM_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1")
LLM_MODEL_NAME = os.getenv("LLM_MODEL_NAME", "qwen-turbo")

# 向量库配置
QDRANT_URL = os.getenv("QDRANT_URL", ":memory:")

# 本地持久化（Week 1 用 SQLite 演示持久化边界，Week 4 迁移 MySQL）
DB_PATH = os.getenv("AGENT_DB_PATH", "agent_data.sqlite")

# 业务常量约束
MAX_CHAT_WINDOW = 10            # 聊天滑动窗口最大轮数
MAX_DEEP_DIVE_PER_QUESTION = 2  # 单道题目最多深挖2次，防止无限追问
MAX_BASE_QUESTIONS = 5          # 基础题库题量
STAGE_LIST = [
    "self_intro",
    "tech_question",
    "project_drill",
    "candidate_ask",
    "summary",
]
