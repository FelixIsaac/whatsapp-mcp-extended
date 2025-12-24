"""WhatsApp MCP Server Library.

This module provides the core functionality for the WhatsApp MCP server,
organized into submodules:

- models: Data classes (Message, Chat, Contact, MessageContext)
- database: Database operations (list_messages, list_chats, search_contacts, etc.)
- bridge: Bridge API calls (send_message, send_reaction, edit_message, etc.)
- utils: Logging, configuration, helper functions
"""

# Models
from .models import Message, Chat, Contact, MessageContext

# Database operations
from .database import (
    DatabaseError,
    list_messages,
    get_message_context,
    list_chats,
    get_contact_by_jid,
    get_contact_nickname,
    set_contact_nickname,
    search_contacts,
)

# Bridge API
from .bridge import (
    BridgeError,
    send_message,
    send_file,
    send_reaction,
    edit_message,
    delete_message,
    mark_read,
    get_group_info,
    create_group,
    create_poll,
)

# Utilities
from .utils import (
    logger,
    setup_logging,
    MESSAGES_DB_PATH,
    WHATSAPP_DB_PATH,
    BRIDGE_HOST,
    WHATSAPP_API_BASE_URL,
    get_sender_name,
)

__all__ = [
    # Models
    "Message",
    "Chat",
    "Contact",
    "MessageContext",
    # Database
    "DatabaseError",
    "list_messages",
    "get_message_context",
    "list_chats",
    "get_contact_by_jid",
    "get_contact_nickname",
    "set_contact_nickname",
    "search_contacts",
    # Bridge
    "BridgeError",
    "send_message",
    "send_file",
    "send_reaction",
    "edit_message",
    "delete_message",
    "mark_read",
    "get_group_info",
    "create_group",
    "create_poll",
    # Utils
    "logger",
    "setup_logging",
    "MESSAGES_DB_PATH",
    "WHATSAPP_DB_PATH",
    "BRIDGE_HOST",
    "WHATSAPP_API_BASE_URL",
    "get_sender_name",
]
