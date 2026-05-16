"""Regression tests for whatsapp.py list_chats."""

import os
import sqlite3
import tempfile
from datetime import datetime, timedelta

from whatsapp import get_chat, list_chats


def _build_messages_db() -> str:
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
        db_path = f.name

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    cursor.execute("""
        CREATE TABLE chats (
            jid TEXT PRIMARY KEY,
            name TEXT,
            last_message_time TEXT
        )
    """)
    cursor.execute("""
        CREATE TABLE messages (
            id TEXT PRIMARY KEY,
            chat_jid TEXT,
            sender TEXT,
            content TEXT,
            timestamp TEXT,
            is_from_me INTEGER
        )
    """)

    t0 = datetime.now()
    t1 = (t0 + timedelta(seconds=1)).isoformat()
    t0s = t0.isoformat()

    cursor.execute(
        "INSERT INTO chats (jid, name, last_message_time) VALUES (?, ?, ?)",
        ("111@s.whatsapp.net", "Alice", t1),
    )
    cursor.execute(
        "INSERT INTO messages (id, chat_jid, sender, content, timestamp, is_from_me) VALUES (?, ?, ?, ?, ?, ?)",
        ("m1", "111@s.whatsapp.net", "111@s.whatsapp.net", "old", t0s, 0),
    )
    cursor.execute(
        "INSERT INTO messages (id, chat_jid, sender, content, timestamp, is_from_me) VALUES (?, ?, ?, ?, ?, ?)",
        ("m2", "111@s.whatsapp.net", "111@s.whatsapp.net", "new", t1, 0),
    )

    conn.commit()
    conn.close()
    return db_path


def test_list_chats_include_last_message_false_no_sql_error(monkeypatch):
    db_path = _build_messages_db()
    try:
        monkeypatch.setenv("MESSAGES_DB_PATH", db_path)
        monkeypatch.setattr("whatsapp.MESSAGES_DB_PATH", db_path)

        chats = list_chats(limit=10, include_last_message=False)

        assert len(chats) == 1
        assert chats[0]["jid"] == "111@s.whatsapp.net"
        assert chats[0]["last_message"] is None
        assert chats[0]["last_message_id"] is None
        assert chats[0]["last_sender"] is None
        assert chats[0]["last_is_from_me"] is None
    finally:
        os.unlink(db_path)


def test_list_chats_include_last_message_true_returns_latest(monkeypatch):
    db_path = _build_messages_db()
    try:
        monkeypatch.setenv("MESSAGES_DB_PATH", db_path)
        monkeypatch.setattr("whatsapp.MESSAGES_DB_PATH", db_path)

        chats = list_chats(limit=10, include_last_message=True)

        assert len(chats) == 1
        assert chats[0]["jid"] == "111@s.whatsapp.net"
        assert chats[0]["last_message"] == "new"
        assert chats[0]["last_message_id"] == "m2"
        assert chats[0]["last_sender"] == "111@s.whatsapp.net"
        assert chats[0]["last_is_from_me"] == 0
    finally:
        os.unlink(db_path)


def test_get_chat_include_last_message_false_no_sql_error(monkeypatch):
    db_path = _build_messages_db()
    try:
        monkeypatch.setenv("MESSAGES_DB_PATH", db_path)
        monkeypatch.setattr("whatsapp.MESSAGES_DB_PATH", db_path)

        chat = get_chat("111@s.whatsapp.net", include_last_message=False)

        assert chat is not None
        assert chat["jid"] == "111@s.whatsapp.net"
        assert chat["last_message"] is None
        assert chat["last_sender"] is None
        assert chat["last_is_from_me"] is None
    finally:
        os.unlink(db_path)
