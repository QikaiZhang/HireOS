"""Phase 1 端到端验证：start → 多轮 chat → report 闭环。

运行方式（在 Agent/ 目录下）：
    uv run python scripts/verify_phase1_loop.py

无 LLM_API_KEY 时走 mock 模式，验证的是链路与状态流转；
配置 Key 后同一脚本验证真实 LLM 推理。
"""

import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))  # 保证可从项目根导入

from fastapi.testclient import TestClient

from config.setting import DB_PATH

# 每次验证都从全新持久化状态开始，保证脚本可重复运行
if os.path.exists(DB_PATH):
    os.remove(DB_PATH)

from api.routes import app

client = TestClient(app)
ROOM_ID = "verify-phase1-loop"
MAX_ROUNDS = 30  # 防御性上限，防止意外死循环

passed = []


def check(name: str, condition: bool, detail: str = ""):
    status = "PASS" if condition else "FAIL"
    passed.append((name, condition))
    print(f"  [{status}] {name}" + (f" — {detail}" if detail else ""))


# 1. 异常路径
resp = client.post("/interview/start", json={"room_id": "conflict-check"})
resp = client.post("/interview/start", json={"room_id": "conflict-check"})
print("\n== 异常路径 ==")
check("重复 room_id 创建返回 409", resp.status_code == 409, f"got {resp.status_code}")
resp = client.post("/interview/chat", json={"room_id": "ghost-room", "message": "hi"})
check("不存在会话的 chat 返回 404", resp.status_code == 404, f"got {resp.status_code}")
resp = client.get("/report/ghost-room")
check("不存在报告返回 404", resp.status_code == 404, f"got {resp.status_code}")

# 2. 正常闭环
print("\n== 正常闭环 ==")
resp = client.post(
    "/interview/start",
    json={"room_id": ROOM_ID, "jd": "Go 后端开发工程师", "resume": "3 年 Go 后端经验，做过分布式存储"},
)
check("start 返回 200", resp.status_code == 200, f"got {resp.status_code}")
body = resp.json()
check("start 返回第一题", bool(body.get("question")), f"question: {body.get('question', '')[:30]}")

rounds = 0
final_report = None
while rounds < MAX_ROUNDS:
    rounds += 1
    resp = client.post("/interview/chat", json={"room_id": ROOM_ID, "message": f"这是第 {rounds} 轮回答"})
    check(f"chat 第 {rounds} 轮返回 200", resp.status_code == 200, f"got {resp.status_code}")
    body = resp.json()
    if body.get("finished"):
        final_report = body.get("report")
        print(f"  [INFO] 面试在第 {rounds} 轮回答后结束")
        break
    check(f"chat 第 {rounds} 轮返回下一题", bool(body.get("question")), f"question: {body.get('question', '')[:30]}")

check("面试正常结束", final_report is not None)
check("对话轮数在防御上限内", rounds < MAX_ROUNDS, f"rounds={rounds}")

if final_report:
    print(f"\n== 报告内容 ==\n{final_report}")
    resp = client.get(f"/report/{ROOM_ID}")
    check("GET /report 返回 200", resp.status_code == 200, f"got {resp.status_code}")
    check("落库报告与结束时报表一致", resp.json().get("report") == final_report)

resp = client.post("/interview/chat", json={"room_id": ROOM_ID, "message": "还在吗"})
check("结束后再 chat 返回 409", resp.status_code == 409, f"got {resp.status_code}")

# 3. 汇总
failed = [name for name, ok in passed if not ok]
print("\n" + "=" * 50)
total = len(passed)
print(f"验证结果: {total - len(failed)}/{total} 通过")
if failed:
    print("失败项: " + "; ".join(failed))
    raise SystemExit(1)
print("Phase 1 闭环验证全部通过 ✅")
