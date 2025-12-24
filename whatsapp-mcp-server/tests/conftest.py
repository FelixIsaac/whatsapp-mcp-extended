"""Pytest fixtures for WhatsApp MCP Server tests."""

import os
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
    cursor.execute(
        "INSERT INTO chats (jid, name) VALUES (?, ?)",
        ("123456789@s.whatsapp.net", "Test User")
    )
    cursor.execute(
        "INSERT INTO chats (jid, name) VALUES (?, ?)",
        ("987654321@g.us", "Test Group")
    )

    # Insert test messages
    now = datetime.now().isoformat()
    cursor.execute(
        """INSERT INTO messages (id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, file_length)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)""",
        ("msg1", "123456789@s.whatsapp.net", "123456789@s.whatsapp.net", "Hello world", now, 0, None, None, None)
    )
    cursor.execute(
        """INSERT INTO messages (id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, file_length)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)""",
        ("msg2", "123456789@s.whatsapp.net", "me", "Hi there", now, 1, None, None, None)
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
        ("123456789@s.whatsapp.net", "John", "John Doe", "Johnny", None)
    )

    conn.commit()
    conn.close()

    yield db_path

    os.unlink(db_path)


@pytest.fixture
def mock_bridge_url(monkeypatch):
    """Mock the bridge URL to prevent real API calls."""
    monkeypatch.setenv("BRIDGE_HOST", "localhost:9999")
