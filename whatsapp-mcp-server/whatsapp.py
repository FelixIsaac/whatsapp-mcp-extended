import json
import os
import os.path
import sqlite3
from dataclasses import dataclass
from datetime import datetime
from typing import Any

import requests

import audio

# Database paths - check for Docker (/app) or local development
if os.path.exists('/app/store'):
    # Docker environment
    MESSAGES_DB_PATH = '/app/store/messages.db'
    WHATSAPP_DB_PATH = '/app/store/whatsapp.db'
else:
    # Local development - use relative path from project root
    _store_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'store')
    MESSAGES_DB_PATH = os.path.join(_store_path, 'messages.db')
    WHATSAPP_DB_PATH = os.path.join(_store_path, 'whatsapp.db')


# Use environment variable for bridge host, default to localhost:8080 for development
# BRIDGE_HOST can be "hostname" (uses :8080) or "hostname:port" (uses specified port)
_bridge_host = os.getenv('BRIDGE_HOST', 'localhost:8080')
if ':' not in _bridge_host:
    _bridge_host = f"{_bridge_host}:8080"
BRIDGE_HOST = _bridge_host
WHATSAPP_API_BASE_URL = f"http://{BRIDGE_HOST}/api"

@dataclass
class Message:
    timestamp: datetime
    sender: str
    content: str
    is_from_me: bool
    chat_jid: str
    id: str
    chat_name: str | None = None
    media_type: str | None = None
    filename: str | None = None
    file_length: int | None = None

    def to_dict(self) -> dict[str, Any]:
        """Convert Message to dictionary for structured output."""
        result = {
            "id": self.id,
            "chat_jid": self.chat_jid,
            "chat_name": self.chat_name,
            "sender": self.sender,
            "content": self.content,
            "timestamp": self.timestamp.isoformat() if self.timestamp else None,
            "is_from_me": self.is_from_me,
            "media_type": self.media_type,
        }
        # Include media metadata if present
        if self.media_type:
            result["filename"] = self.filename
            result["file_length"] = self.file_length
        return result

@dataclass
class Chat:
    jid: str
    name: str | None
    last_message_time: datetime | None
    last_message: str | None = None
    last_message_id: str | None = None
    last_sender: str | None = None
    last_is_from_me: bool | None = None

    @property
    def is_group(self) -> bool:
        """Determine if chat is a group based on JID pattern."""
        return self.jid.endswith("@g.us")

    def to_dict(self) -> dict[str, Any]:
        """Convert Chat to dictionary for structured output."""
        return {
            "jid": self.jid,
            "name": self.name,
            "is_group": self.is_group,
            "last_message_time": self.last_message_time.isoformat() if self.last_message_time else None,
            "last_message": self.last_message,
            "last_message_id": self.last_message_id,
            "last_sender": self.last_sender,
            "last_is_from_me": self.last_is_from_me,
        }

@dataclass
class Contact:
    phone_number: str
    name: str | None
    jid: str
    first_name: str | None = None
    full_name: str | None = None
    push_name: str | None = None
    business_name: str | None = None
    nickname: str | None = None  # User-defined nickname

    def to_dict(self) -> dict[str, Any]:
        """Convert Contact to dictionary for structured output."""
        return {
            "jid": self.jid,
            "phone_number": self.phone_number,
            "name": self.name,
            "first_name": self.first_name,
            "full_name": self.full_name,
            "push_name": self.push_name,
            "business_name": self.business_name,
            "nickname": self.nickname,
        }

@dataclass
class MessageContext:
    message: Message
    before: list[Message]
    after: list[Message]

def get_sender_name(sender_jid: str) -> str:
    """Get the best available name for a sender using both contact and chat data."""
    try:
        # First check for custom nickname
        nickname = get_contact_nickname(sender_jid)
        if nickname:
            return nickname
        
        # Try to get rich contact information from WhatsApp store
        whatsapp_conn = sqlite3.connect(WHATSAPP_DB_PATH)
        whatsapp_cursor = whatsapp_conn.cursor()
        
        # Look for contact in WhatsApp contacts
        whatsapp_cursor.execute("""
            SELECT first_name, full_name, push_name, business_name
            FROM whatsmeow_contacts
            WHERE their_jid = ?
            LIMIT 1
        """, (sender_jid,))
        
        contact_result = whatsapp_cursor.fetchone()
        whatsapp_conn.close()
        
        if contact_result:
            first_name, full_name, push_name, business_name = contact_result
            # Return the best available name
            return full_name or push_name or first_name or business_name or sender_jid
        
        # Fall back to chat database
        messages_conn = sqlite3.connect(MESSAGES_DB_PATH)
        messages_cursor = messages_conn.cursor()
        
        # First try matching by exact JID
        messages_cursor.execute("""
            SELECT name
            FROM chats
            WHERE jid = ?
            LIMIT 1
        """, (sender_jid,))
        
        result = messages_cursor.fetchone()
        
        # If no result, try looking for the number within JIDs
        if not result:
            # Extract the phone number part if it's a JID
            if '@' in sender_jid:
                phone_part = sender_jid.split('@')[0]
            else:
                phone_part = sender_jid
                
            messages_cursor.execute("""
                SELECT name
                FROM chats
                WHERE jid LIKE ?
                LIMIT 1
            """, (f"%{phone_part}%",))
            
            result = messages_cursor.fetchone()
        
        if result and result[0]:
            return result[0]
        else:
            return sender_jid
        
    except sqlite3.Error as e:
        print(f"Database error while getting sender name: {e}")
        return sender_jid
    finally:
        if 'messages_conn' in locals():
            messages_conn.close()

def format_message(message: Message, show_chat_info: bool = True) -> None:
    """Print a single message with consistent formatting."""
    output = ""
    
    if show_chat_info and message.chat_name:
        output += f"[{message.timestamp:%Y-%m-%d %H:%M:%S}] Chat: {message.chat_name} "
    else:
        output += f"[{message.timestamp:%Y-%m-%d %H:%M:%S}] "
        
    content_prefix = ""
    if hasattr(message, 'media_type') and message.media_type:
        content_prefix = f"[{message.media_type} - Message ID: {message.id} - Chat JID: {message.chat_jid}] "
    
    try:
        sender_name = get_sender_name(message.sender) if not message.is_from_me else "Me"
        output += f"From: {sender_name}: {content_prefix}{message.content}\n"
    except Exception as e:
        print(f"Error formatting message: {e}")
    return output

def format_messages_list(messages: list[Message], show_chat_info: bool = True) -> None:
    output = ""
    if not messages:
        output += "No messages to display."
        return output
    
    for message in messages:
        output += format_message(message, show_chat_info)
    return output

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
    """Get messages matching the specified criteria with optional context.

    Returns a list of message dictionaries with structured data.
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        # Build base query - include filename and file_length for media metadata
        query_parts = ["SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id, messages.media_type, messages.filename, messages.file_length FROM messages"]
        query_parts.append("JOIN chats ON messages.chat_jid = chats.jid")
        where_clauses = []
        params = []

        # Add filters
        if after:
            try:
                after_dt = datetime.fromisoformat(after)
            except ValueError:
                raise ValueError(f"Invalid date format for 'after': {after}. Please use ISO-8601 format.")
            where_clauses.append("messages.timestamp > ?")
            params.append(after_dt)

        if before:
            try:
                before_dt = datetime.fromisoformat(before)
            except ValueError:
                raise ValueError(f"Invalid date format for 'before': {before}. Please use ISO-8601 format.")
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

        # Add pagination
        offset = page * limit
        query_parts.append("ORDER BY messages.timestamp DESC")
        query_parts.append("LIMIT ? OFFSET ?")
        params.extend([limit, offset])

        cursor.execute(" ".join(query_parts), tuple(params))
        messages = cursor.fetchall()

        result = []
        for msg in messages:
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
                file_length=msg[9]
            )
            result.append(message)

        if include_context and result:
            # Add context for each message
            messages_with_context = []
            seen_ids = set()
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

        # Return messages as list of dicts
        return [msg.to_dict() for msg in result]

    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return []
    finally:
        if 'conn' in locals():
            conn.close()


def get_message_context(
    message_id: str,
    before: int = 5,
    after: int = 5
) -> MessageContext:
    """Get context around a specific message."""
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        # Get the target message first
        cursor.execute("""
            SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id, messages.chat_jid, messages.media_type, messages.filename, messages.file_length
            FROM messages
            JOIN chats ON messages.chat_jid = chats.jid
            WHERE messages.id = ?
        """, (message_id,))
        msg_data = cursor.fetchone()

        if not msg_data:
            raise ValueError(f"Message with ID {message_id} not found")

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
            file_length=msg_data[10]
        )

        # Get messages before
        cursor.execute("""
            SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id, messages.media_type, messages.filename, messages.file_length
            FROM messages
            JOIN chats ON messages.chat_jid = chats.jid
            WHERE messages.chat_jid = ? AND messages.timestamp < ?
            ORDER BY messages.timestamp DESC
            LIMIT ?
        """, (msg_data[7], msg_data[0], before))

        before_messages = []
        for msg in cursor.fetchall():
            before_messages.append(Message(
                timestamp=datetime.fromisoformat(msg[0]),
                sender=msg[1],
                chat_name=msg[2],
                content=msg[3],
                is_from_me=msg[4],
                chat_jid=msg[5],
                id=msg[6],
                media_type=msg[7],
                filename=msg[8],
                file_length=msg[9]
            ))

        # Get messages after
        cursor.execute("""
            SELECT messages.timestamp, messages.sender, chats.name, messages.content, messages.is_from_me, chats.jid, messages.id, messages.media_type, messages.filename, messages.file_length
            FROM messages
            JOIN chats ON messages.chat_jid = chats.jid
            WHERE messages.chat_jid = ? AND messages.timestamp > ?
            ORDER BY messages.timestamp ASC
            LIMIT ?
        """, (msg_data[7], msg_data[0], after))

        after_messages = []
        for msg in cursor.fetchall():
            after_messages.append(Message(
                timestamp=datetime.fromisoformat(msg[0]),
                sender=msg[1],
                chat_name=msg[2],
                content=msg[3],
                is_from_me=msg[4],
                chat_jid=msg[5],
                id=msg[6],
                media_type=msg[7],
                filename=msg[8],
                file_length=msg[9]
            ))

        return MessageContext(
            message=target_message,
            before=before_messages,
            after=after_messages
        )
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        raise
    finally:
        if 'conn' in locals():
            conn.close()


def list_chats(
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active"
) -> list[Chat]:
    """Get chats matching the specified criteria."""
    print(f"Debug: Database path: {MESSAGES_DB_PATH}")
    print(f"Debug: Database exists: {os.path.exists(MESSAGES_DB_PATH)}")
    
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()
        
        # Debug: Check if tables exist
        cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
        tables = cursor.fetchall()
        print(f"Debug: Available tables: {tables}")
        
        # Debug: Check row counts
        try:
            cursor.execute("SELECT COUNT(*) FROM chats")
            chat_count = cursor.fetchone()[0]
            print(f"Debug: Total chats in database: {chat_count}")
        except Exception as e:
            print(f"Debug: Error counting chats: {e}")
        
        # Build base query
        query_parts = ["""
            SELECT
                chats.jid,
                chats.name,
                chats.last_message_time,
                messages.content as last_message,
                messages.id as last_message_id,
                messages.sender as last_sender,
                messages.is_from_me as last_is_from_me
            FROM chats
        """]
        
        if include_last_message:
            query_parts.append("""
                LEFT JOIN messages ON chats.jid = messages.chat_jid 
                AND chats.last_message_time = messages.timestamp
            """)
            
        where_clauses = []
        params = []
        
        if query:
            where_clauses.append("(LOWER(chats.name) LIKE LOWER(?) OR chats.jid LIKE ?)")
            params.extend([f"%{query}%", f"%{query}%"])
            
        if where_clauses:
            query_parts.append("WHERE " + " AND ".join(where_clauses))
            
        # Add sorting
        order_by = "chats.last_message_time DESC" if sort_by == "last_active" else "chats.name"
        query_parts.append(f"ORDER BY {order_by}")
        
        # Add pagination
        offset = (page ) * limit
        query_parts.append("LIMIT ? OFFSET ?")
        params.extend([limit, offset])
        
        cursor.execute(" ".join(query_parts), tuple(params))
        chats = cursor.fetchall()

        result = []
        for chat_data in chats:
            chat = Chat(
                jid=chat_data[0],
                name=chat_data[1],
                last_message_time=datetime.fromisoformat(chat_data[2]) if chat_data[2] else None,
                last_message=chat_data[3],
                last_message_id=chat_data[4],
                last_sender=chat_data[5],
                last_is_from_me=chat_data[6]
            )
            result.append(chat.to_dict())

        return result

    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return []
    finally:
        if 'conn' in locals():
            conn.close()


def search_contacts(query: str) -> list[dict[str, Any]]:
    """Search contacts by name or phone number using both WhatsApp contacts and chat data."""
    try:
        # Connect to both databases
        whatsapp_conn = sqlite3.connect(WHATSAPP_DB_PATH)
        whatsapp_cursor = whatsapp_conn.cursor()

        # Split query into characters to support partial matching
        search_pattern = '%' + query + '%'

        # Query WhatsApp contacts database for rich contact information
        whatsapp_cursor.execute("""
            SELECT DISTINCT
                their_jid,
                first_name,
                full_name,
                push_name,
                business_name
            FROM whatsmeow_contacts
            WHERE
                (LOWER(COALESCE(first_name, '')) LIKE LOWER(?) OR
                 LOWER(COALESCE(full_name, '')) LIKE LOWER(?) OR
                 LOWER(COALESCE(push_name, '')) LIKE LOWER(?) OR
                 LOWER(COALESCE(business_name, '')) LIKE LOWER(?) OR
                 LOWER(their_jid) LIKE LOWER(?))
                AND their_jid NOT LIKE '%@g.us'
            ORDER BY
                CASE
                    WHEN full_name IS NOT NULL AND full_name != '' THEN full_name
                    WHEN push_name IS NOT NULL AND push_name != '' THEN push_name
                    WHEN first_name IS NOT NULL AND first_name != '' THEN first_name
                    ELSE their_jid
                END
            LIMIT 50
        """, (search_pattern, search_pattern, search_pattern, search_pattern, search_pattern))
        whatsapp_contacts = whatsapp_cursor.fetchall()
        result = []
        # If whatsapp_contacts is not empty, use only those
        for contact_data in whatsapp_contacts:
            jid = contact_data[0]
            phone_number = jid.split('@')[0] if '@' in jid else jid
            # Determine best display name
            first_name = contact_data[1]
            full_name = contact_data[2]
            push_name = contact_data[3]
            business_name = contact_data[4]
            display_name = full_name or push_name or first_name or business_name or phone_number
            contact = Contact(
                phone_number=phone_number,
                name=display_name,
                jid=jid,
                first_name=first_name,
                full_name=full_name,
                push_name=push_name,
                business_name=business_name
            )
            result.append(contact.to_dict())
        return result
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return []
    finally:
        if 'whatsapp_conn' in locals():
            whatsapp_conn.close()


def get_contact_chats(jid: str, limit: int = 20, page: int = 0) -> list[dict[str, Any]]:
    """Get all chats involving the contact.

    Args:
        jid: The contact's JID to search for
        limit: Maximum number of chats to return (default 20)
        page: Page number for pagination (default 0)
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT DISTINCT
                c.jid,
                c.name,
                c.last_message_time,
                m.content as last_message,
                m.id as last_message_id,
                m.sender as last_sender,
                m.is_from_me as last_is_from_me
            FROM chats c
            JOIN messages m ON c.jid = m.chat_jid
            WHERE m.sender = ? OR c.jid = ?
            ORDER BY c.last_message_time DESC
            LIMIT ? OFFSET ?
        """, (jid, jid, limit, page * limit))

        chats = cursor.fetchall()

        result = []
        for chat_data in chats:
            chat = Chat(
                jid=chat_data[0],
                name=chat_data[1],
                last_message_time=datetime.fromisoformat(chat_data[2]) if chat_data[2] else None,
                last_message=chat_data[3],
                last_message_id=chat_data[4],
                last_sender=chat_data[5],
                last_is_from_me=chat_data[6]
            )
            result.append(chat.to_dict())

        return result
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return []
    finally:
        if 'conn' in locals():
            conn.close()


def get_last_interaction(jid: str) -> dict[str, Any] | None:
    """Get most recent message involving the contact."""
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT
                m.timestamp,
                m.sender,
                c.name,
                m.content,
                m.is_from_me,
                c.jid,
                m.id,
                m.media_type,
                m.filename,
                m.file_length
            FROM messages m
            JOIN chats c ON m.chat_jid = c.jid
            WHERE m.sender = ? OR c.jid = ?
            ORDER BY m.timestamp DESC
            LIMIT 1
        """, (jid, jid))

        msg_data = cursor.fetchone()

        if not msg_data:
            return None

        message = Message(
            timestamp=datetime.fromisoformat(msg_data[0]),
            sender=msg_data[1],
            chat_name=msg_data[2],
            content=msg_data[3],
            is_from_me=msg_data[4],
            chat_jid=msg_data[5],
            id=msg_data[6],
            media_type=msg_data[7],
            filename=msg_data[8],
            file_length=msg_data[9]
        )

        return message.to_dict()
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return None
    finally:
        if 'conn' in locals():
            conn.close()


def get_chat(chat_jid: str, include_last_message: bool = True) -> dict[str, Any] | None:
    """Get chat metadata by JID."""
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        query = """
            SELECT
                c.jid,
                c.name,
                c.last_message_time,
                m.content as last_message,
                m.sender as last_sender,
                m.is_from_me as last_is_from_me
            FROM chats c
        """

        if include_last_message:
            query += """
                LEFT JOIN messages m ON c.jid = m.chat_jid
                AND c.last_message_time = m.timestamp
            """

        query += " WHERE c.jid = ?"

        cursor.execute(query, (chat_jid,))
        chat_data = cursor.fetchone()

        if not chat_data:
            return None

        chat = Chat(
            jid=chat_data[0],
            name=chat_data[1],
            last_message_time=datetime.fromisoformat(chat_data[2]) if chat_data[2] else None,
            last_message=chat_data[3],
            last_sender=chat_data[4],
            last_is_from_me=chat_data[5]
        )
        return chat.to_dict()
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return None
    finally:
        if 'conn' in locals():
            conn.close()


def get_direct_chat_by_contact(sender_phone_number: str) -> dict[str, Any] | None:
    """Get chat metadata by sender phone number."""
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT
                c.jid,
                c.name,
                c.last_message_time,
                m.content as last_message,
                m.sender as last_sender,
                m.is_from_me as last_is_from_me
            FROM chats c
            LEFT JOIN messages m ON c.jid = m.chat_jid
                AND c.last_message_time = m.timestamp
            WHERE c.jid LIKE ? AND c.jid NOT LIKE '%@g.us'
            LIMIT 1
        """, (f"%{sender_phone_number}%",))

        chat_data = cursor.fetchone()

        if not chat_data:
            return None

        chat = Chat(
            jid=chat_data[0],
            name=chat_data[1],
            last_message_time=datetime.fromisoformat(chat_data[2]) if chat_data[2] else None,
            last_message=chat_data[3],
            last_sender=chat_data[4],
            last_is_from_me=chat_data[5]
        )
        return chat.to_dict()
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return None
    finally:
        if 'conn' in locals():
            conn.close()

def send_message(recipient: str, message: str) -> dict[str, Any]:
    """Send a WhatsApp message and return structured result with message_id."""
    try:
        # Validate input
        if not recipient:
            return {"success": False, "error": "Recipient must be provided"}

        url = f"{WHATSAPP_API_BASE_URL}/send"
        payload = {
            "recipient": recipient,
            "message": message,
        }

        response = requests.post(url, json=payload)

        # Check if the request was successful
        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "message_id": result.get("message_id"),
                "timestamp": result.get("timestamp"),
                "recipient": result.get("recipient"),
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "error": f"Unexpected error: {str(e)}"}

def send_file(recipient: str, media_path: str) -> dict[str, Any]:
    """Send a file via WhatsApp and return structured result with message_id."""
    try:
        # Validate input
        if not recipient:
            return {"success": False, "error": "Recipient must be provided"}

        if not media_path:
            return {"success": False, "error": "Media path must be provided"}

        if not os.path.isfile(media_path):
            return {"success": False, "error": f"Media file not found: {media_path}"}

        url = f"{WHATSAPP_API_BASE_URL}/send"
        payload = {
            "recipient": recipient,
            "media_path": media_path
        }

        response = requests.post(url, json=payload)

        # Check if the request was successful
        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "message_id": result.get("message_id"),
                "timestamp": result.get("timestamp"),
                "recipient": result.get("recipient"),
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "error": f"Unexpected error: {str(e)}"}

def send_audio_message(recipient: str, media_path: str) -> dict[str, Any]:
    """Send an audio message via WhatsApp and return structured result with message_id."""
    try:
        # Validate input
        if not recipient:
            return {"success": False, "error": "Recipient must be provided"}

        if not media_path:
            return {"success": False, "error": "Media path must be provided"}

        if not os.path.isfile(media_path):
            return {"success": False, "error": f"Media file not found: {media_path}"}

        if not media_path.endswith(".ogg"):
            try:
                media_path = audio.convert_to_opus_ogg_temp(media_path)
            except Exception as e:
                return {"success": False, "error": f"Error converting file to opus ogg. You likely need to install ffmpeg: {str(e)}"}

        url = f"{WHATSAPP_API_BASE_URL}/send"
        payload = {
            "recipient": recipient,
            "media_path": media_path
        }

        response = requests.post(url, json=payload)

        # Check if the request was successful
        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "message_id": result.get("message_id"),
                "timestamp": result.get("timestamp"),
                "recipient": result.get("recipient"),
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "error": f"Unexpected error: {str(e)}"}

def download_media(message_id: str, chat_jid: str) -> str | None:
    """Download media from a message and return the local file path.
    
    Args:
        message_id: The ID of the message containing the media
        chat_jid: The JID of the chat containing the message
    
    Returns:
        The local file path if download was successful, None otherwise
    """
    try:
        url = f"{WHATSAPP_API_BASE_URL}/download"
        payload = {
            "message_id": message_id,
            "chat_jid": chat_jid
        }
        
        response = requests.post(url, json=payload)
        
        if response.status_code == 200:
            result = response.json()
            if result.get("success", False):
                path = result.get("path")
                print(f"Media downloaded successfully: {path}")
                return path
            else:
                print(f"Download failed: {result.get('message', 'Unknown error')}")
                return None
        else:
            print(f"Error: HTTP {response.status_code} - {response.text}")
            return None
            
    except requests.RequestException as e:
        print(f"Request error: {str(e)}")
        return None
    except json.JSONDecodeError:
        print(f"Error parsing response: {response.text}")
        return None
    except Exception as e:
        print(f"Unexpected error: {str(e)}")
        return None

def get_contact_by_jid(jid: str) -> Contact | None:
    """Get detailed contact information by JID."""
    try:
        # First try WhatsApp contacts database
        whatsapp_conn = sqlite3.connect(WHATSAPP_DB_PATH)
        whatsapp_cursor = whatsapp_conn.cursor()
        
        whatsapp_cursor.execute("""
            SELECT their_jid, first_name, full_name, push_name, business_name
            FROM whatsmeow_contacts
            WHERE their_jid = ?
            LIMIT 1
        """, (jid,))
        
        contact_data = whatsapp_cursor.fetchone()
        whatsapp_conn.close()
        
        # Get custom nickname
        nickname = get_contact_nickname(jid)
        
        if contact_data:
            phone_number = contact_data[0].split('@')[0] if '@' in contact_data[0] else contact_data[0]
            display_name = nickname or contact_data[2] or contact_data[3] or contact_data[1] or contact_data[4] or phone_number
            
            return Contact(
                phone_number=phone_number,
                name=display_name,
                jid=contact_data[0],
                first_name=contact_data[1],
                full_name=contact_data[2],
                push_name=contact_data[3],
                business_name=contact_data[4],
                nickname=nickname
            )
        
        # Fall back to chats database
        messages_conn = sqlite3.connect(MESSAGES_DB_PATH)
        messages_cursor = messages_conn.cursor()
        
        messages_cursor.execute("""
            SELECT jid, name
            FROM chats
            WHERE jid = ? AND jid NOT LIKE '%@g.us'
            LIMIT 1
        """, (jid,))
        
        chat_data = messages_cursor.fetchone()
        messages_conn.close()
        
        if chat_data:
            phone_number = chat_data[0].split('@')[0] if '@' in chat_data[0] else chat_data[0]
            display_name = nickname or chat_data[1] or phone_number
            
            return Contact(
                phone_number=phone_number,
                name=display_name,
                jid=chat_data[0],
                nickname=nickname
            )
        
        return None
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return None


def get_contact_by_phone(phone_number: str) -> Contact | None:
    """Get contact information by phone number."""
    try:
        # Try different JID formats
        possible_jids = [
            f"{phone_number}@s.whatsapp.net",
            f"{phone_number}@c.us",
            phone_number
        ]
        
        for jid in possible_jids:
            contact = get_contact_by_jid(jid)
            if contact:
                return contact
        
        # Try partial matching in chats
        messages_conn = sqlite3.connect(MESSAGES_DB_PATH)
        messages_cursor = messages_conn.cursor()
        
        messages_cursor.execute("""
            SELECT jid, name
            FROM chats
            WHERE jid LIKE ? AND jid NOT LIKE '%@g.us'
            LIMIT 1
        """, (f"%{phone_number}%",))
        
        chat_data = messages_cursor.fetchone()
        messages_conn.close()
        
        if chat_data:
            actual_phone = chat_data[0].split('@')[0] if '@' in chat_data[0] else chat_data[0]
            return Contact(
                phone_number=actual_phone,
                name=chat_data[1] or actual_phone,
                jid=chat_data[0]
            )
        
        return None
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return None


def list_all_contacts(limit: int = 100) -> list[Contact]:
    """Get all contacts with their detailed information."""
    try:
        contacts = []
        
        # Get contacts from WhatsApp store
        whatsapp_conn = sqlite3.connect(WHATSAPP_DB_PATH)
        whatsapp_cursor = whatsapp_conn.cursor()
                
        whatsapp_cursor.execute("""
            SELECT their_jid, first_name, full_name, push_name, business_name
            FROM whatsmeow_contacts
            WHERE 1=1 AND their_jid NOT LIKE '%@g.us'
            ORDER BY 
                CASE 
                    WHEN full_name IS NOT NULL AND full_name != '' THEN full_name
                    WHEN push_name IS NOT NULL AND push_name != '' THEN push_name
                    WHEN first_name IS NOT NULL AND first_name != '' THEN first_name
                    ELSE their_jid
                END
            LIMIT ?
        """, (limit,))
        
        whatsapp_contacts = whatsapp_cursor.fetchall()
        whatsapp_conn.close()
        
        for contact_data in whatsapp_contacts:
            jid = contact_data[0]
            phone_number = jid.split('@')[0] if '@' in jid else jid
            
            first_name = contact_data[1]
            full_name = contact_data[2]
            push_name = contact_data[3]
            business_name = contact_data[4]
            
            display_name = full_name or push_name or first_name or business_name or phone_number
            
            contact = Contact(
                phone_number=phone_number,
                name=display_name,
                jid=jid,
                first_name=first_name,
                full_name=full_name,
                push_name=push_name,
                business_name=business_name
            )
            contacts.append(contact)
        
        return contacts
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return []


def format_contact_info(contact: Contact) -> str:
    """Format contact information for display."""
    output = f"📱 {contact.name} ({contact.phone_number})\n"
    output += f"   JID: {contact.jid}\n"
    
    if contact.full_name and contact.full_name != contact.name:
        output += f"   Full Name: {contact.full_name}\n"
    
    if contact.first_name and contact.first_name != contact.name:
        output += f"   First Name: {contact.first_name}\n"
    
    if contact.push_name and contact.push_name != contact.name:
        output += f"   Display Name: {contact.push_name}\n"
    
    if contact.business_name:
        output += f"   Business: {contact.business_name}\n"
    
    if contact.nickname:
        output += f"   Nickname: {contact.nickname}\n"
    
    return output

def set_contact_nickname(jid: str, nickname: str) -> dict[str, Any]:
    """Set a custom nickname for a contact.

    Returns:
        Structured dict with success, jid, nickname, updated_at
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        # Insert or update nickname
        cursor.execute("""
            INSERT OR REPLACE INTO contact_nicknames (jid, nickname, updated_at)
            VALUES (?, ?, CURRENT_TIMESTAMP)
        """, (jid, nickname))

        conn.commit()

        # Get the updated timestamp
        cursor.execute("SELECT updated_at FROM contact_nicknames WHERE jid = ?", (jid,))
        row = cursor.fetchone()
        updated_at = row[0] if row else None

        return {
            "success": True,
            "jid": jid,
            "nickname": nickname,
            "updated_at": updated_at
        }

    except sqlite3.Error as e:
        return {"success": False, "jid": jid, "error": str(e)}
    finally:
        if 'conn' in locals():
            conn.close()


def get_contact_nickname(jid: str) -> str | None:
    """Get a contact's custom nickname."""
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()
        
        cursor.execute("""
            SELECT nickname
            FROM contact_nicknames
            WHERE jid = ?
        """, (jid,))
        
        result = cursor.fetchone()
        return result[0] if result else None
        
    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return None
    finally:
        if 'conn' in locals():
            conn.close()


def remove_contact_nickname(jid: str) -> dict[str, Any]:
    """Remove a contact's custom nickname.

    Returns:
        Structured dict with success, jid
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("DELETE FROM contact_nicknames WHERE jid = ?", (jid,))

        if cursor.rowcount > 0:
            conn.commit()
            return {"success": True, "jid": jid}
        else:
            return {"success": False, "jid": jid, "error": "No nickname found"}

    except sqlite3.Error as e:
        return {"success": False, "jid": jid, "error": str(e)}
    finally:
        if 'conn' in locals():
            conn.close()


def list_contact_nicknames() -> list[dict[str, Any]]:
    """List all custom contact nicknames with timestamps.

    Returns:
        List of dicts with jid, nickname, created_at, updated_at
    """
    try:
        conn = sqlite3.connect(MESSAGES_DB_PATH)
        cursor = conn.cursor()

        cursor.execute("""
            SELECT jid, nickname, created_at, updated_at
            FROM contact_nicknames
            ORDER BY nickname
        """)

        return [
            {
                "jid": row[0],
                "nickname": row[1],
                "created_at": row[2],
                "updated_at": row[3]
            }
            for row in cursor.fetchall()
        ]

    except sqlite3.Error as e:
        print(f"Database error: {e}")
        return []
    finally:
        if 'conn' in locals():
            conn.close()


# Phase 1 Features: Reactions, Edit, Delete, Group Info, Mark Read

def send_reaction(chat_jid: str, message_id: str, emoji: str) -> dict[str, Any]:
    """Send an emoji reaction to a message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to react to
        emoji: The emoji to react with (empty string to remove reaction)

    Returns:
        Structured dict with success, chat_jid, message_id, emoji, error
    """
    try:
        url = f"{WHATSAPP_API_BASE_URL}/reaction"
        payload = {
            "chat_jid": chat_jid,
            "message_id": message_id,
            "emoji": emoji
        }

        response = requests.post(url, json=payload)

        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "chat_jid": chat_jid,
                "message_id": message_id,
                "emoji": emoji,
                "action": "remove" if emoji == "" else "add",
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Unexpected error: {str(e)}"}


def edit_message(chat_jid: str, message_id: str, new_content: str) -> dict[str, Any]:
    """Edit a previously sent message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to edit
        new_content: The new message content

    Returns:
        Structured dict with success, chat_jid, message_id, new_content, error
    """
    try:
        url = f"{WHATSAPP_API_BASE_URL}/edit"
        payload = {
            "chat_jid": chat_jid,
            "message_id": message_id,
            "new_content": new_content
        }

        response = requests.post(url, json=payload)

        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "chat_jid": chat_jid,
                "message_id": message_id,
                "new_content": new_content,
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Unexpected error: {str(e)}"}


def delete_message(chat_jid: str, message_id: str, sender_jid: str | None = None) -> dict[str, Any]:
    """Delete/revoke a message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to delete
        sender_jid: Optional sender JID for admin revoking others' messages in groups

    Returns:
        Structured dict with success, chat_jid, message_id, error
    """
    try:
        url = f"{WHATSAPP_API_BASE_URL}/delete"
        payload = {
            "chat_jid": chat_jid,
            "message_id": message_id
        }
        if sender_jid:
            payload["sender_jid"] = sender_jid

        response = requests.post(url, json=payload)

        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "chat_jid": chat_jid,
                "message_id": message_id,
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "chat_jid": chat_jid, "message_id": message_id, "error": f"Unexpected error: {str(e)}"}


def get_group_info(group_jid: str) -> dict[str, Any]:
    """Get information about a WhatsApp group.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")

    Returns:
        Structured dict with success, group_jid, name, topic, participants, error
    """
    try:
        url = f"{WHATSAPP_API_BASE_URL}/group/{group_jid}"

        response = requests.get(url)

        if response.status_code == 200:
            result = response.json()
            if result.get("success", False):
                data = result.get("data", {})
                # Enrich participants with names from contacts
                participants = data.get("participants", [])
                enriched_participants = []
                for p in participants:
                    jid = p.get("jid", "")
                    phone = jid.split("@")[0] if "@" in jid else jid
                    # Try to get contact name
                    contact_name = None
                    try:
                        contacts = search_contacts(phone)
                        if contacts:
                            contact_name = contacts[0].get("name") or contacts[0].get("full_name") or contacts[0].get("push_name")
                    except Exception:
                        pass
                    enriched_participants.append({
                        "jid": jid,
                        "name": contact_name,
                        "is_admin": p.get("is_admin", False),
                        "is_super_admin": p.get("is_super_admin", False),
                    })
                return {
                    "success": True,
                    "group_jid": group_jid,
                    "name": data.get("name"),
                    "topic": data.get("topic"),
                    "created_at": data.get("created_at"),
                    "created_by": data.get("created_by"),
                    "participant_count": len(enriched_participants),
                    "participants": enriched_participants,
                }
            else:
                return {"success": False, "group_jid": group_jid, "error": result.get("message", "Unknown error")}
        else:
            return {"success": False, "group_jid": group_jid, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "group_jid": group_jid, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "group_jid": group_jid, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "group_jid": group_jid, "error": f"Unexpected error: {str(e)}"}


def mark_messages_read(chat_jid: str, message_ids: list[str], sender_jid: str | None = None) -> dict[str, Any]:
    """Mark messages as read.

    Args:
        chat_jid: The JID of the chat containing the messages
        message_ids: List of message IDs to mark as read
        sender_jid: Optional sender JID (required for group chats)

    Returns:
        Structured dict with success, chat_jid, message_ids, count, error
    """
    try:
        url = f"{WHATSAPP_API_BASE_URL}/read"
        payload = {
            "chat_jid": chat_jid,
            "message_ids": message_ids
        }
        if sender_jid:
            payload["sender_jid"] = sender_jid

        response = requests.post(url, json=payload)

        if response.status_code == 200:
            result = response.json()
            return {
                "success": result.get("success", False),
                "chat_jid": chat_jid,
                "message_ids": message_ids,
                "count": len(message_ids),
                "error": result.get("message") if not result.get("success") else None,
            }
        else:
            return {"success": False, "chat_jid": chat_jid, "message_ids": message_ids, "error": f"HTTP {response.status_code} - {response.text}"}

    except requests.RequestException as e:
        return {"success": False, "chat_jid": chat_jid, "message_ids": message_ids, "error": f"Request error: {str(e)}"}
    except json.JSONDecodeError:
        return {"success": False, "chat_jid": chat_jid, "message_ids": message_ids, "error": f"Error parsing response: {response.text}"}
    except Exception as e:
        return {"success": False, "chat_jid": chat_jid, "message_ids": message_ids, "error": f"Unexpected error: {str(e)}"}
