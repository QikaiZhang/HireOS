"""Agent HTTP API。

三个接口构成 Week 1 闭环：
- POST /interview/start  创建面试 → 返回第一题
- POST /interview/chat   提交回答 → 返回追问/下一题，或面试结束返回报告
- GET  /report/{room_id} 查询已生成的报告

注意：路由用 def 而非 async def，让同步的图执行/LLM 调用跑在线程池里，
不阻塞事件循环（MVP 阶段的取舍，见 PHASE1_SPEC 风险表）。
"""

from fastapi import FastAPI, HTTPException

from api.schemas import ChatRequest, ReportResponse, StartInterviewRequest
from graph.builder import build_graph
from memory import long_term

app = FastAPI(title="HireOS Agent Service", version="0.1.0")
graph = build_graph()


def _config(room_id: str):
    return {"configurable": {"thread_id": room_id}}


@app.post("/interview/start")
def start_interview(req: StartInterviewRequest):
    if long_term.session_exists(req.room_id):
        raise HTTPException(status_code=409, detail=f"面试会话已存在: {req.room_id}")

    final_state = graph.invoke(
        {"room_id": req.room_id, "jd": req.jd, "resume": req.resume},
        _config(req.room_id),
    )
    long_term.save_session(req.room_id, req.jd, req.resume)

    return {
        "room_id": req.room_id,
        "global_phase": final_state["global_phase"],
        "question": final_state["current_output_question"],
        "finished": False,
    }


@app.post("/interview/chat")
def chat(req: ChatRequest):
    config = _config(req.room_id)
    snapshot = graph.get_state(config)
    if not snapshot.values:
        raise HTTPException(status_code=404, detail=f"面试会话不存在: {req.room_id}")
    if not snapshot.next:
        raise HTTPException(status_code=409, detail="面试已结束，无法继续对话")

    # 注入候选人回答，从中断点（listen 之前）恢复执行
    graph.update_state(config, {"candidate_latest_answer": req.message})
    final_state = graph.invoke(None, config)

    finished = final_state["global_phase"] == "end"
    if finished:
        long_term.update_session_status(req.room_id, "finished")
        long_term.save_report(req.room_id, final_state["final_report"])
        return {
            "room_id": req.room_id,
            "global_phase": final_state["global_phase"],
            "question": None,
            "finished": True,
            "report": final_state["final_report"],
        }

    return {
        "room_id": req.room_id,
        "global_phase": final_state["global_phase"],
        "question": final_state["current_output_question"],
        "finished": False,
    }


@app.get("/report/{room_id}", response_model=ReportResponse)
def get_report(room_id: str):
    report = long_term.load_report(room_id)
    if report is None:
        raise HTTPException(status_code=404, detail=f"报告不存在或面试未结束: {room_id}")
    return ReportResponse(room_id=room_id, report=report)
