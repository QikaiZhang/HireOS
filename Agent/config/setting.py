import os
from dotenv import load_dotenv


load_dotenv()

# 大模型配置
LLM\_API\_KEY = os.getenv("LLM\_API\_KEY", "")
LLM\_BASE\_URL = os.getenv("LLM\_BASE\_URL", "")
LLM\_MODEL\_NAME = os.getenv("LLM\_MODEL\_NAME", "qwen-turbo")

# 向量库配置
QDRANT\_URL = os.getenv("QDRANT\_URL", ":memory:")

# 业务常量约束
MAX\_CHAT\_WINDOW = 10          # 聊天滑动窗口最大轮数
MAX\_DEEP\_DIVE\_PER\_QUESTION = 2 # 单道题目最多深挖2次，防止无限追问
STAGE\_LIST = \[
    "self\_intro",
    "tech\_question",
    "project\_drill",
    "candidate\_ask",
    "summary"
\]