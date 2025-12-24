"""Data models for WhatsApp MCP server."""

from dataclasses import dataclass
from datetime import datetime
from typing import Any


@dataclass
class Message:
    """Represents a WhatsApp message."""

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
    """Represents a WhatsApp chat/conversation."""

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
    """Represents a WhatsApp contact."""

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
    """Represents a message with surrounding context."""

    message: Message
    before: list[Message]
    after: list[Message]

    def to_dict(self) -> dict[str, Any]:
        """Convert MessageContext to dictionary."""
        return {
            "message": self.message.to_dict(),
            "before": [m.to_dict() for m in self.before],
            "after": [m.to_dict() for m in self.after],
        }
