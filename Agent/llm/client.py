"""LLM 调用封装。

统一走 OpenAI 兼容协议。无 API Key 或调用失败时返回 None，
由调用方决定 mock 兜底，保证本地无 Key 也能跑通完整闭环。
"""

import json

from openai import OpenAI

from config.setting import LLM_API_KEY, LLM_BASE_URL, LLM_MODEL_NAME

_client = None


def _get_client() -> OpenAI:
    global _client
    if _client is None:
        _client = OpenAI(api_key=LLM_API_KEY, base_url=LLM_BASE_URL)
    return _client


def llm_enabled() -> bool:
    return bool(LLM_API_KEY)


def ask_json(system_prompt: str, user_prompt: str) -> dict | None:
    """请求 LLM 返回 JSON 对象，解析失败返回 None。

    system_prompt 里必须约束模型只输出 JSON。
    """
    if not llm_enabled():
        print("[LLM MOCK] 未配置 LLM_API_KEY，本次调用将走 mock 兜底")
        return None

    try:
        resp = _get_client().chat.completions.create(
            model=LLM_MODEL_NAME,
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            response_format={"type": "json_object"},
            temperature=0.3,
        )
        content = resp.choices[0].message.content
        return json.loads(content)
    except Exception as exc:
        print(f"[LLM ERROR] 模型 {LLM_MODEL_NAME} 调用失败，将走 mock 兜底: {exc}")
        return None
