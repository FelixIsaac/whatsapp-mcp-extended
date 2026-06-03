"""Pytest fixtures for WhatsApp MCP Server tests."""

import os

# Must be set before any test module imports lib.utils — importing lib.utils raises
# FileNotFoundError when messages.db doesn't exist (by design, to prevent silent empty
# results in production). Tests use temp DBs and don't need the real store.
os.environ.setdefault("WA_SKIP_DB_CHECK", "1")

import sqlite3
import tempfile
from datetime import datetime

import pytest


@pytest.fixture
def temp_messages_db():
    """Create a temporary messages database with test data."""
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
        db_path = f.name

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    # Create tables
    cursor.execute("""
        CREATE TABLE chats (
            jid TEXT PRIMARY KEY,
            name TEXT
        )
    """)

    cursor.execute("""
        CREATE TABLE messages (
            id TEXT PRIMARY KEY,
            chat_jid TEXT,
            sender TEXT,
            content TEXT,
            timestamp TEXT,
            is_from_me INTEGER,
            media_type TEXT,
            filename TEXT,
            file_length INTEGER,
            sender_name TEXT,
            quoted_message_id TEXT,
            quoted_sender_name TEXT,
            reply_to_message_id TEXT,
            edit_count INTEGER DEFAULT 0,
            is_edited INTEGER DEFAULT 0,
            is_forwarded INTEGER DEFAULT 0,
            forwarded_from TEXT,
            is_system_message INTEGER DEFAULT 0,
            system_message_type TEXT,
            FOREIGN KEY (chat_jid) REFERENCES chats(jid)
        )
    """)

    cursor.execute("""
        CREATE TABLE contact_nicknames (
            jid TEXT PRIMARY KEY,
            nickname TEXT,
            created_at TEXT,
            updated_at TEXT
        )
    """)

    # Insert test data
    cursor.execute("INSERT INTO chats (jid, name) VALUES (?, ?)", ("123456789@s.whatsapp.net", "Test User"))
    cursor.execute("INSERT INTO chats (jid, name) VALUES (?, ?)", ("987654321@g.us", "Test Group"))

    # Insert test messages
    now = datetime.now().isoformat()
    cursor.execute(
        """INSERT INTO messages (id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, file_length, sender_name)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
        (
            "msg1",
            "123456789@s.whatsapp.net",
            "123456789@s.whatsapp.net",
            "Hello world",
            now,
            0,
            None,
            None,
            None,
            "Test User",
        ),
    )
    cursor.execute(
        """INSERT INTO messages (id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, file_length, sender_name)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
        ("msg2", "123456789@s.whatsapp.net", "me", "Hi there", now, 1, None, None, None, "Me"),
    )

    conn.commit()
    conn.close()

    yield db_path

    # Cleanup
    os.unlink(db_path)


@pytest.fixture
def temp_whatsapp_db():
    """Create a temporary WhatsApp database with test contacts."""
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
        db_path = f.name

    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    cursor.execute("""
        CREATE TABLE whatsmeow_contacts (
            their_jid TEXT PRIMARY KEY,
            first_name TEXT,
            full_name TEXT,
            push_name TEXT,
            business_name TEXT
        )
    """)

    cursor.execute(
        """INSERT INTO whatsmeow_contacts (their_jid, first_name, full_name, push_name, business_name)
           VALUES (?, ?, ?, ?, ?)""",
        ("123456789@s.whatsapp.net", "John", "John Doe", "Johnny", None),
    )

    conn.commit()
    conn.close()

    yield db_path

    os.unlink(db_path)


@pytest.fixture
def mock_bridge_url(monkeypatch):
    """Mock the bridge URL to prevent real API calls."""
    monkeypatch.setenv("BRIDGE_HOST", "localhost:9999")
