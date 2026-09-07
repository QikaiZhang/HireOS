from pydantic import BaseModel, Field


class StartInterviewRequest(BaseModel):
    room_id: str = Field(..., description="面试房间 ID，同时作为 LangGraph thread_id")
    jd: str = Field(default="", description="岗位描述")
    resume: str = Field(default="", description="候选人简历（纯文本）")


class ChatRequest(BaseModel):
    room_id: str = Field(..., description="面试房间 ID")
    message: str = Field(..., description="候选人的回答（文字）")


class ReportResponse(BaseModel):
    room_id: str
    report: dict = Field(..., description="面试评估报告")
