"""Resolve LID/phone JID equivalents stored in the whatsmeow LID map.

WhatsApp's LID migration splits one-to-one conversations across two
``chat_jid`` values in ``messages.db`` (``user@lid`` and
``phone@s.whatsapp.net``). The ``whatsmeow_lid_map`` table in
``whatsapp.db`` maps ``lid`` <-> ``pn`` so the read paths can query
every known equivalent instead of dropping messages stored under the
other format.

Read paths such as ``list_messages``, ``get_contact_chats``,
``get_last_interaction`` and ``get_direct_chat_by_contact`` should use
:func:`chat_jid_where_clause` to build their ``WHERE`` filters so both
JID formats are matched.
"""

from __future__ import annotations

import os
import sqlite3

LID_SERVER = "lid"
PN_SERVER = "s.whatsapp.net"
_USER_JID_SERVERS = {LID_SERVER, PN_SERVER}


def _whatsapp_db_path() -> str:
    """Return the path of ``whatsapp.db`` (holds ``whatsmeow_lid_map``)."""
    explicit = os.getenv("WHATSAPP_DB_PATH")
    if explicit:
        return explicit
    messages_db = os.getenv("MESSAGES_DB_PATH")
    if messages_db:
        directory = os.path.dirname(messages_db)
        return os.path.join(directory, "whatsapp.db") if directory else "whatsapp.db"
    return "whatsapp.db"


def _normalize_lid(value: str) -> str:
    """Return ``value`` as a full LID JID (``user@lid``)."""
    return value if "@" in value else f"{value}@{LID_SERVER}"


def _normalize_pn(value: str) -> str:
    """Return ``value`` as a full phone JID (``user@s.whatsapp.net``)."""
    return value if "@" in value else f"{value}@{PN_SERVER}"


def resolve_chat_jid_variants(chat_jid: str) -> list[str]:
    """Return every known equivalent JID for a user chat JID.

    The list always starts with the original ``chat_jid`` so callers can
    use it in a ``WHERE chat_jid IN (...)`` clause. When no equivalent is
    known (non-user JID, missing map, unreadable DB) the list contains only
    the original JID, so callers degrade to the old single-JID behaviour.
    """
    variants: list[str] = [chat_jid]
    if not chat_jid or "@" not in chat_jid:
        return variants

    user, server = chat_jid.rpartition("@")
    if server not in _USER_JID_SERVERS:
        return variants

    db_path = _whatsapp_db_path()
    if not os.path.exists(db_path):
        return variants

    conn = sqlite3.connect(db_path)
    try:
        # The map stores full JIDs, but some deployments keep bare user
        # parts in the ``pn`` column, so match both the full and bare forms.
        cursor = conn.execute(
            "SELECT lid, pn FROM whatsmeow_lid_map WHERE lid = ? OR pn = ? OR lid = ? OR pn = ?",
            (chat_jid, chat_jid, user, user),
        )
        for lid, pn in cursor.fetchall():
            if lid:
                variants.append(_normalize_lid(lid))
            if pn:
                variants.append(_normalize_pn(pn))
    except sqlite3.Error:
        # Missing table / unreadable map: keep the original JID only.
        pass
    finally:
        conn.close()

    return list(dict.fromkeys(variants))


def chat_jid_where_clause(chat_jid: str) -> tuple[str, tuple[str, ...]]:
    """Build a LID-aware ``WHERE`` clause for a single chat JID.

    Returns ``(clause, params)`` where ``clause`` is ``chat_jid IN (?, ...)``
    with one placeholder per resolved variant, e.g.::

        clause, params = chat_jid_where_clause(chat_jid)
        rows = conn.execute(
            f"SELECT * FROM messages WHERE {clause} ORDER BY timestamp DESC LIMIT ?",
            (*params, limit),
        ).fetchall()
    """
    variants = resolve_chat_jid_variants(chat_jid)
    placeholders = ", ".join("?" for _ in variants)
    return f"chat_jid IN ({placeholders})", tuple(variants)
