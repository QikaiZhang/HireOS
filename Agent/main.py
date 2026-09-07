import uvicorn

if __name__ == "__main__":
    # 需在 Agent/ 目录下运行：uv run python main.py
    uvicorn.run("api.routes:app", host="0.0.0.0", port=8000)
