"""长期记忆：SQLite 持久化面试会话与报告。

Week 1 演示持久化边界，Week 4 计划迁移 MySQL，接口签名保持 (room_id, ...) 粒度。
"""

import json
import sqlite3
from datetime import datetime

from config.setting import DB_PATH


def _connect() -> sqlite3.Connection:
    conn = sqlite3.connect(DB_PATH)
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS interview_session (
            room_id TEXT PRIMARY KEY,
            jd TEXT,
            resume TEXT,
            status TEXT,
            created_at TEXT,
            updated_at TEXT
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS interview_report (
            room_id TEXT PRIMARY KEY,
            report_json TEXT,
            created_at TEXT
        )
        """
    )
    return conn


def save_session(room_id: str, jd: str, resume: str, status: str = "ongoing"):
    now = datetime.now().isoformat()
    with _connect() as conn:
        conn.execute(
            "INSERT OR REPLACE INTO interview_session VALUES (?, ?, ?, ?, ?, ?)",
            (room_id, jd, resume, status, now, now),
        )


def update_session_status(room_id: str, status: str):
    with _connect() as conn:
        conn.execute(
            "UPDATE interview_session SET status = ?, updated_at = ? WHERE room_id = ?",
            (status, datetime.now().isoformat(), room_id),
        )


def session_exists(room_id: str) -> bool:
    with _connect() as conn:
        row = conn.execute(
            "SELECT 1 FROM interview_session WHERE room_id = ?", (room_id,)
        ).fetchone()
    return row is not None


def save_report(room_id: str, report: dict):
    with _connect() as conn:
        conn.execute(
            "INSERT OR REPLACE INTO interview_report VALUES (?, ?, ?)",
            (room_id, json.dumps(report, ensure_ascii=False), datetime.now().isoformat()),
        )


def load_report(room_id: str) -> dict | None:
    with _connect() as conn:
        row = conn.execute(
            "SELECT report_json FROM interview_report WHERE room_id = ?", (room_id,)
        ).fetchone()
    return json.loads(row[0]) if row else None
