"""Database operations for WhatsApp MCP server."""

import re
import sqlite3
from datetime import datetime, timedelta
from typing import Any

from .models import Chat, Contact, Message, MessageContext
from .utils import MESSAGES_DB_PATH, WHATSAPP_DB_PATH, get_sender_name, logger


class DatabaseError(Exception):
    """Custom exception for database operations."""
    pass


def get_message_character_count(content: str) -> int:
    """Get character count in message content.

    Args:
        content: Message content string.

    Returns:
        Number of characters.
    """
    return len(content) if content else 0


def get_message_word_count(content: str) -> int:
    """Get word count in message content.

    Args:
        content: Message content string.

    Returns:
        Number of words.
    """
    if not content:
        return 0
    # Split on whitespace, filter empty strings
    words = [w for w in content.split() if w]
    return len(words)


def extract_urls(content: str) -> list[str]:
    """Extract URLs from message content.

    Args:
        content: Message content string.

    Returns:
        List of URLs found.
    """
    if not content:
        return []
    # Simple URL pattern matching
    url_pattern = r'https?://[^\s]+'
    return re.findall(url_pattern, content)


def get_chat_statistics(chat_jid: str, conn: sqlite3.Connection) -> tuple[int, int, int]:
    """Get message statistics for a chat.

    Args:
        chat_jid: Chat JID.
        conn: Database connection.

    Returns:
        Tuple of (total_messages, messages_today, messages_last_7_days).
    """
    cursor = conn.cursor()
    now = datetime.now()
    today_start = now.replace(hour=0, minute=0, second=0, microsecond=0)
    seven_days_ago = now - timedelta(days=7)

    # Total messages
    cursor.execute("SELECT COUNT(*) FROM messages WHERE chat_jid = ?", (chat_jid,))
    total = cursor.fetchone()[0] or 0

    # Messages today
    cursor.execute(
        "SELECT COUNT(*) FROM messages WHERE chat_jid = ? AND timestamp >= ?",
        (chat_jid, today_start.isoformat())
    )
    today = cursor.fetchone()[0] or 0

    # Messages last 7 days
    cursor.execute(
        "SELECT COUNT(*) FROM messages WHERE chat_jid = ? AND timestamp >= ?",
        (chat_jid, seven_days_ago.isoformat())
    )
    week = cursor.fetchone()[0] or 0

    return total, today, week


def list_messages(
    after: str | None = None,
    before: str | None = None,
    sender_phone_number: str | None = None,
    chat_jid: str | None = None,
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_context: bool = False,
    context_before: int = 1,
    context_after: int = 1
) -> list[dict[str, Any]]:
    """Get messages matching the specified criteria.

    Args:
        after: ISO-8601 datetime to filter messages after.
        before: ISO-8601 datetime to filter messages before.
        sender_phone_number: Filter by sender phone number.
        chat_jid: Filter by chat JID.
        query: Text search query.
        limit: Maximum messages to return.
        page: Page number for pagination.
        include_context: Include surrounding messages.
        context_before: Messages to include before each match.
        context_after: Messages to include after each match.

    Returns:
        List of message dictionaries.

    Raises:
        DatabaseError: If database query fails.
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        query_parts = [
            "SELECT messages.timestamp, messages.sender, chats.name, "
            "messages.content, messages.is_from_me, chats.jid, messages.id, "
            "messages.media_type, messages.filename, messages.file_length, "
            "messages.sender_name FROM messages"
        ]
        query_parts.append("JOIN chats ON messages.chat_jid = chats.jid")
        where_clauses = []
        params: list[Any] = []

        if after:
            try:
                after_dt = datetime.fromisoformat(after)
            except ValueError:
                raise ValueError(f"Invalid date format for 'after': {after}")
            where_clauses.append("messages.timestamp > ?")
            params.append(after_dt)

        if before:
            try:
                before_dt = datetime.fromisoformat(before)
            except ValueError:
                raise ValueError(f"Invalid date format for 'before': {before}")
            where_clauses.append("messages.timestamp < ?")
            params.append(before_dt)

        if sender_phone_number:
            where_clauses.append("messages.sender = ?")
            params.append(sender_phone_number)

        if chat_jid:
            where_clauses.append("messages.chat_jid = ?")
            params.append(chat_jid)

        if query:
            where_clauses.append("LOWER(messages.content) LIKE LOWER(?)")
            params.append(f"%{query}%")

        if where_clauses:
            query_parts.append("WHERE " + " AND ".join(where_clauses))

        offset = page * limit
        query_parts.append("ORDER BY messages.timestamp DESC")
        query_parts.append("LIMIT ? OFFSET ?")
        params.extend([limit, offset])

        cursor.execute(" ".join(query_parts), tuple(params))
        messages = cursor.fetchall()

        result = []
        for msg in messages:
            # Use stored sender_name if available, otherwise fallback to lookup
            sender_name = msg[10] if msg[10] else get_sender_name(msg[1])

            # Compute metadata fields
            char_count = get_message_character_count(msg[3])
            word_count = get_message_word_count(msg[3])
            urls = extract_urls(msg[3])
            is_group = msg[5].endswith("@g.us")

            message = Message(
                timestamp=datetime.fromisoformat(msg[0]),
                sender=msg[1],
                chat_name=msg[2],
                content=msg[3],
                is_from_me=msg[4],
                chat_jid=msg[5],
                id=msg[6],
                media_type=msg[7],
                filename=msg[8],
                file_length=msg[9],
                sender_name=sender_name,
                character_count=char_count,
                word_count=word_count,
                url_list=urls,
                is_group=is_group
            )
            result.append(message)

        if include_context and result:
            messages_with_context = []
            seen_ids: set[str] = set()
            for msg in result:
                context = get_message_context(msg.id, context_before, context_after)
                for ctx_msg in context.before:
                    if ctx_msg.id not in seen_ids:
                        messages_with_context.append(ctx_msg.to_dict())
                        seen_ids.add(ctx_msg.id)
                if context.message.id not in seen_ids:
                    messages_with_context.append(context.message.to_dict())
                    seen_ids.add(context.message.id)
                for ctx_msg in context.after:
                    if ctx_msg.id not in seen_ids:
                        messages_with_context.append(ctx_msg.to_dict())
                        seen_ids.add(ctx_msg.id)
            return messages_with_context

        return [msg.to_dict() for msg in result]

    except sqlite3.Error as e:
        logger.error("Database error in list_messages: %s", e)
        raise DatabaseError(f"Failed to list messages: {e}") from e
    finally:
        if 'conn' in locals():
            conn.close()


def get_message_context(message_id: str, before: int = 5, after: int = 5) -> MessageContext:
    """Get context around a specific message.

    Args:
        message_id: ID of the target message.
        before: Number of messages before.
        after: Number of messages after.

    Returns:
        MessageContext with target message and surrounding messages.

    Raises:
        DatabaseError: If database query fails.
        ValueError: If message not found.
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT messages.timestamp, messages.sender, chats.name, messages.content,
                   messages.is_from_me, chats.jid, messages.id, messages.chat_jid,
                   messages.media_type, messages.filename, messages.file_length,
                   messages.sender_name
            FROM messages
            JOIN chats ON messages.chat_jid = chats.jid
            WHERE messages.id = ?
        """, (message_id,))
        msg_data = cursor.fetchone()

        if not msg_data:
            raise ValueError(f"Message with ID {message_id} not found")

        # Use stored sender_name if available, otherwise fallback to lookup
        sender_name = msg_data[11] if msg_data[11] else get_sender_name(msg_data[1])

        # Compute metadata fields
        char_count = get_message_character_count(msg_data[3])
        word_count = get_message_word_count(msg_data[3])
        urls = extract_urls(msg_data[3])
        is_group = msg_data[5].endswith("@g.us")

        target_message = Message(
            timestamp=datetime.fromisoformat(msg_data[0]),
            sender=msg_data[1],
            chat_name=msg_data[2],
            content=msg_data[3],
            is_from_me=msg_data[4],
            chat_jid=msg_data[5],
            id=msg_data[6],
            media_type=msg_data[8],
            filename=msg_data[9],
            file_length=msg_data[10],
            sender_name=sender_name,
            character_count=char_count,
            word_count=word_count,
            url_list=urls,
            is_group=is_group
        )

        # Get messages before
        cursor.execute("""
            SELECT messages.timestamp, messages.sender, chats.name, messages.content,
                   messages.is_from_me, chats.jid, messages.id, messages.media_type,
                   messages.filename, messages.file_length, messages.sender_name
            FROM messages
            JOIN chats ON messages.chat_jid = chats.jid
            WHERE messages.chat_jid = ? AND messages.timestamp < ?
            ORDER BY messages.timestamp DESC
            LIMIT ?
        """, (msg_data[7], msg_data[0], before))

        before_messages = []
        for msg in cursor.fetchall():
            sender_name = msg[10] if msg[10] else get_sender_name(msg[1])
            char_count = get_message_character_count(msg[3])
            word_count = get_message_word_count(msg[3])
            urls = extract_urls(msg[3])
            is_group = msg[5].endswith("@g.us")

            before_messages.append(Message(
                timestamp=datetime.fromisoformat(msg[0]),
                sender=msg[1], chat_name=msg[2], content=msg[3],
                is_from_me=msg[4], chat_jid=msg[5], id=msg[6],
                media_type=msg[7], filename=msg[8], file_length=msg[9],
                sender_name=sender_name,
                character_count=char_count,
                word_count=word_count,
                url_list=urls,
                is_group=is_group
            ))

        # Get messages after
        cursor.execute("""
            SELECT messages.timestamp, messages.sender, chats.name, messages.content,
                   messages.is_from_me, chats.jid, messages.id, messages.media_type,
                   messages.filename, messages.file_length, messages.sender_name
            FROM messages
            JOIN chats ON messages.chat_jid = chats.jid
            WHERE messages.chat_jid = ? AND messages.timestamp > ?
            ORDER BY messages.timestamp ASC
            LIMIT ?
        """, (msg_data[7], msg_data[0], after))

        after_messages = []
        for msg in cursor.fetchall():
            sender_name = msg[10] if msg[10] else get_sender_name(msg[1])
            char_count = get_message_character_count(msg[3])
            word_count = get_message_word_count(msg[3])
            urls = extract_urls(msg[3])
            is_group = msg[5].endswith("@g.us")

            after_messages.append(Message(
                timestamp=datetime.fromisoformat(msg[0]),
                sender=msg[1], chat_name=msg[2], content=msg[3],
                is_from_me=msg[4], chat_jid=msg[5], id=msg[6],
                media_type=msg[7], filename=msg[8], file_length=msg[9],
                sender_name=sender_name,
                character_count=char_count,
                word_count=word_count,
                url_list=urls,
                is_group=is_group
            ))

        return MessageContext(
            message=target_message,
            before=list(reversed(before_messages)),
            after=after_messages
        )

    except sqlite3.Error as e:
        logger.error("Database error in get_message_context: %s", e)
        raise DatabaseError(f"Failed to get message context: {e}") from e
    finally:
        if 'conn' in locals():
            conn.close()


def list_chats(
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active"
) -> list[dict[str, Any]]:
    """Get chats matching specified criteria.

    Args:
        query: Search term for chat name/JID.
        limit: Maximum chats to return.
        page: Page number for pagination.
        include_last_message: Include last message details.
        sort_by: Sort field ("last_active" or "name").

    Returns:
        List of chat dictionaries.
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        if include_last_message:
            query_sql = """
                SELECT c.jid, c.name,
                       m.timestamp, m.content, m.id, m.sender, m.is_from_me, m.sender_name
                FROM chats c
                LEFT JOIN (
                    SELECT chat_jid, timestamp, content, id, sender, is_from_me, sender_name,
                           ROW_NUMBER() OVER (PARTITION BY chat_jid ORDER BY timestamp DESC) as rn
                    FROM messages
                ) m ON c.jid = m.chat_jid AND m.rn = 1
            """
        else:
            query_sql = "SELECT jid, name, NULL, NULL, NULL, NULL, NULL, NULL FROM chats"

        where_clauses = []
        params: list[Any] = []

        if query:
            where_clauses.append("(LOWER(c.name) LIKE LOWER(?) OR LOWER(c.jid) LIKE LOWER(?))")
            params.extend([f"%{query}%", f"%{query}%"])

        if where_clauses:
            query_sql += " WHERE " + " AND ".join(where_clauses)

        if sort_by == "name":
            query_sql += " ORDER BY c.name ASC"
        else:
            query_sql += " ORDER BY m.timestamp DESC NULLS LAST"

        offset = page * limit
        query_sql += " LIMIT ? OFFSET ?"
        params.extend([limit, offset])

        cursor.execute(query_sql, tuple(params))
        chats = cursor.fetchall()

        result = []
        for chat in chats:
            # Use stored sender_name if available, otherwise fallback to lookup
            last_sender_name = None
            if chat[5]:  # if there's a last_sender
                last_sender_name = chat[7] if chat[7] else get_sender_name(chat[5])

            # Get chat statistics
            total_msgs, msgs_today, msgs_week = get_chat_statistics(chat[0], conn)

            # Determine chat type
            chat_type = "group" if chat[0].endswith("@g.us") else "individual"

            chat_obj = Chat(
                jid=chat[0],
                name=chat[1],
                last_message_time=datetime.fromisoformat(chat[2]) if chat[2] else None,
                last_message=chat[3],
                last_message_id=chat[4],
                last_sender=chat[5],
                last_sender_name=last_sender_name,
                last_is_from_me=bool(chat[6]) if chat[6] is not None else None,
                total_message_count=total_msgs,
                message_count_today=msgs_today,
                message_count_last_7_days=msgs_week,
                chat_type=chat_type
            )
            result.append(chat_obj.to_dict())

        return result

    except sqlite3.Error as e:
        logger.error("Database error in list_chats: %s", e)
        raise DatabaseError(f"Failed to list chats: {e}") from e
    finally:
        if 'conn' in locals():
            conn.close()


def get_contact_by_jid(jid: str) -> Contact | None:
    """Get contact information by JID.

    Args:
        jid: WhatsApp JID.

    Returns:
        Contact object if found, None otherwise.
    """
    try:
        conn = sqlite3.connect(WHATSAPP_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT their_jid, first_name, full_name, push_name, business_name
            FROM whatsmeow_contacts
            WHERE their_jid = ?
            LIMIT 1
        """, (jid,))

        result = cursor.fetchone()
        if not result:
            return None

        phone = jid.split("@")[0] if "@" in jid else jid

        # Check for nickname
        nickname = get_contact_nickname(jid)

        return Contact(
            jid=result[0],
            phone_number=phone,
            name=result[2] or result[3] or result[1],
            first_name=result[1],
            full_name=result[2],
            push_name=result[3],
            business_name=result[4],
            nickname=nickname
        )

    except sqlite3.Error as e:
        logger.error("Database error in get_contact_by_jid: %s", e)
        return None
    finally:
        if 'conn' in locals():
            conn.close()


def get_contact_nickname(jid: str) -> str | None:
    """Get custom nickname for a contact.

    Args:
        jid: WhatsApp JID.

    Returns:
        Nickname if set, None otherwise.
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT nickname FROM contact_nicknames WHERE jid = ?
        """, (jid,))

        result = cursor.fetchone()
        return result[0] if result else None

    except sqlite3.Error:
        return None
    finally:
        if 'conn' in locals():
            conn.close()


def set_contact_nickname(jid: str, nickname: str) -> dict[str, Any]:
    """Set custom nickname for a contact.

    Args:
        jid: WhatsApp JID.
        nickname: Nickname to set.

    Returns:
        Result dictionary with success status.
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            INSERT OR REPLACE INTO contact_nicknames (jid, nickname, updated_at)
            VALUES (?, ?, datetime('now'))
        """, (jid, nickname))

        conn.commit()

        return {
            "success": True,
            "jid": jid,
            "nickname": nickname,
            "updated_at": datetime.now().isoformat()
        }

    except sqlite3.Error as e:
        logger.error("Database error in set_contact_nickname: %s", e)
        raise DatabaseError(f"Failed to set nickname: {e}") from e
    finally:
        if 'conn' in locals():
            conn.close()


def search_contacts(query: str) -> list[dict[str, Any]]:
    """Search contacts by name or phone number.

    Optimized to avoid N+1 queries by JOINing contact_nicknames table.

    Args:
        query: Search term.

    Returns:
        List of matching contact dictionaries.
    """
    try:
        whatsapp_conn = sqlite3.connect(WHATSAPP_DB_PATH)
        whatsapp_cursor = whatsapp_conn.cursor()

        # Query WhatsApp contacts
        whatsapp_cursor.execute("""
            SELECT their_jid, first_name, full_name, push_name, business_name
            FROM whatsmeow_contacts
            WHERE LOWER(first_name) LIKE LOWER(?)
               OR LOWER(full_name) LIKE LOWER(?)
               OR LOWER(push_name) LIKE LOWER(?)
               OR their_jid LIKE ?
            LIMIT 50
        """, (f"%{query}%", f"%{query}%", f"%{query}%", f"%{query}%"))

        contacts = whatsapp_cursor.fetchall()
        whatsapp_conn.close()

        # Now fetch nicknames with a single query (JOIN, not N+1)
        if not contacts:
            return []

        jids = [row[0] for row in contacts]

        # Use messages DB for nicknames (single query with IN clause)
        messages_conn = sqlite3.connect(MESSAGES_DB_PATH)
        messages_cursor = messages_conn.cursor()

        # Use parameter placeholders to avoid SQL injection
        placeholders = ",".join("?" * len(jids))
        messages_cursor.execute(f"""
            SELECT jid, nickname FROM contact_nicknames
            WHERE jid IN ({placeholders})
        """, jids)

        nickname_map = {row[0]: row[1] for row in messages_cursor.fetchall()}
        messages_conn.close()

        # Build results
        results = []
        for row in contacts:
            jid = row[0]
            phone = jid.split("@")[0] if "@" in jid else jid
            nickname = nickname_map.get(jid)

            contact = Contact(
                jid=jid,
                phone_number=phone,
                name=row[2] or row[3] or row[1],
                first_name=row[1],
                full_name=row[2],
                push_name=row[3],
                business_name=row[4],
                nickname=nickname
            )
            results.append(contact.to_dict())

        return results

    except sqlite3.Error as e:
        logger.error("Database error in search_contacts: %s", e)
        raise DatabaseError(f"Failed to search contacts: {e}") from e
    finally:
        if 'whatsapp_conn' in locals():
            whatsapp_conn.close()
        if 'messages_conn' in locals():
            messages_conn.close()
