"""Tests for LID resolution on the read path (Issue #72).

WhatsApp's LID migration splits one-to-one conversations across two
``chat_jid`` values (``user@lid`` and ``phone@s.whatsapp.net``); the
read helpers must resolve every known equivalent via
``whatsmeow_lid_map`` so no messages are silently dropped.
"""

import sqlite3
from pathlib import Path

from lib.lid_resolution import chat_jid_where_clause, resolve_chat_jid_variants

LID = "abc123@lid"
PN_USER = "15551234567"
PHONE = f"{PN_USER}@s.whatsapp.net"


def _create_lid_map(db_path: Path, rows: list[tuple[str, str]]) -> None:
    conn = sqlite3.connect(db_path)
    try:
        conn.execute("CREATE TABLE whatsmeow_lid_map (lid TEXT PRIMARY KEY, pn TEXT)")
        conn.executemany("INSERT INTO whatsmeow_lid_map (lid, pn) VALUES (?, ?)", rows)
        conn.commit()
    finally:
        conn.close()


def test_phone_jid_resolves_to_lid(tmp_path, monkeypatch):
    db = tmp_path / "whatsapp.db"
    _create_lid_map(db, [(LID, PN_USER)])
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(db))

    assert resolve_chat_jid_variants(PHONE) == [PHONE, LID]


def test_lid_jid_resolves_to_phone(tmp_path, monkeypatch):
    db = tmp_path / "whatsapp.db"
    _create_lid_map(db, [(LID, PN_USER)])
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(db))

    variants = resolve_chat_jid_variants(LID)
    assert variants[0] == LID
    assert set(variants) == {LID, PHONE}


def test_map_storing_full_phone_jid(tmp_path, monkeypatch):
    db = tmp_path / "whatsapp.db"
    _create_lid_map(db, [(LID, PHONE)])
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(db))

    assert resolve_chat_jid_variants(PHONE) == [PHONE, LID]


def test_unknown_jid_unchanged(tmp_path, monkeypatch):
    db = tmp_path / "whatsapp.db"
    _create_lid_map(db, [(LID, PN_USER)])
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(db))

    assert resolve_chat_jid_variants("9999999999@s.whatsapp.net") == ["9999999999@s.whatsapp.net"]


def test_group_jid_unchanged(tmp_path, monkeypatch):
    db = tmp_path / "whatsapp.db"
    _create_lid_map(db, [(LID, PN_USER)])
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(db))

    assert resolve_chat_jid_variants("120363-abc@g.us") == ["120363-abc@g.us"]


def test_missing_map_degrades_to_single_jid(tmp_path, monkeypatch):
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(tmp_path / "does-not-exist.db"))

    assert resolve_chat_jid_variants(PHONE) == [PHONE]


def test_where_clause_lists_all_variants(tmp_path, monkeypatch):
    db = tmp_path / "whatsapp.db"
    _create_lid_map(db, [(LID, PN_USER)])
    monkeypatch.setenv("WHATSAPP_DB_PATH", str(db))

    clause, params = chat_jid_where_clause(PHONE)
    assert clause == "chat_jid IN (?, ?)"
    assert list(params) == [PHONE, LID]
