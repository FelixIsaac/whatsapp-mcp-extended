"""Data models for WhatsApp MCP server."""

from dataclasses import dataclass, field
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
    sender_name: str | None = None
    media_type: str | None = None
    filename: str | None = None
    file_length: int | None = None
    # Phase 1 metadata fields
    sender_contact_info: dict[str, Any] | None = None
    character_count: int | None = None
    word_count: int | None = None
    url_list: list[str] = field(default_factory=list)
    mentions: list[str] = field(default_factory=list)
    reply_to_message_id: str | None = None
    quoted_message_id: str | None = None
    quoted_sender_name: str | None = None
    quoted_text_preview: str | None = None
    reaction_summary: dict[str, Any] = field(default_factory=dict)
    edit_count: int | None = None
    is_edited: bool = False
    is_forwarded: bool = False
    forwarded_from: str | None = None
    is_system_message: bool = False
    system_message_type: str | None = None
    response_time_seconds: int | None = None
    is_first_message_today: bool = False
    message_position_in_thread: int | None = None
    has_reactions: bool = False
    is_group: bool = False
    is_read: bool = False

    def to_dict(self) -> dict[str, Any]:
        """Convert Message to dictionary for structured output, omitting empty/null fields."""
        result = {
            "id": self.id,
            "chat_jid": self.chat_jid,
            "timestamp": self.timestamp.isoformat() if self.timestamp else None,
            "sender": self.sender,
            "content": self.content,
            "is_from_me": self.is_from_me,
            "is_group": self.is_group,
        }

        # Add optional fields only if they have values
        if self.chat_name:
            result["chat_name"] = self.chat_name
        if self.sender_name:
            result["sender_name"] = self.sender_name
        if self.media_type:
            result["media_type"] = self.media_type
            result["filename"] = self.filename
            result["file_length"] = self.file_length
        if self.sender_contact_info:
            result["sender_contact_info"] = self.sender_contact_info
        if self.character_count is not None:
            result["character_count"] = self.character_count
        if self.word_count is not None:
            result["word_count"] = self.word_count
        if self.url_list:
            result["url_list"] = self.url_list
        if self.mentions:
            result["mentions"] = self.mentions
        if self.reply_to_message_id:
            result["reply_to_message_id"] = self.reply_to_message_id
        if self.quoted_message_id:
            result["quoted_message_id"] = self.quoted_message_id
        if self.quoted_sender_name:
            result["quoted_sender_name"] = self.quoted_sender_name
        if self.quoted_text_preview:
            result["quoted_text_preview"] = self.quoted_text_preview
        if self.reaction_summary:
            result["reaction_summary"] = self.reaction_summary
        if self.edit_count is not None and self.edit_count > 0:
            result["edit_count"] = self.edit_count
        if self.is_edited:
            result["is_edited"] = self.is_edited
        if self.is_forwarded:
            result["is_forwarded"] = self.is_forwarded
        if self.forwarded_from:
            result["forwarded_from"] = self.forwarded_from
        if self.is_system_message:
            result["is_system_message"] = self.is_system_message
        if self.system_message_type:
            result["system_message_type"] = self.system_message_type
        if self.response_time_seconds is not None:
            result["response_time_seconds"] = self.response_time_seconds
        if self.is_first_message_today:
            result["is_first_message_today"] = self.is_first_message_today
        if self.message_position_in_thread is not None:
            result["message_position_in_thread"] = self.message_position_in_thread
        if self.has_reactions:
            result["has_reactions"] = self.has_reactions
        if self.is_read:
            result["is_read"] = self.is_read

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
    last_sender_name: str | None = None
    last_is_from_me: bool | None = None
    # Phase 1 metadata fields
    total_message_count: int | None = None
    message_count_today: int | None = None
    message_count_last_7_days: int | None = None
    message_velocity_last_7_days: dict[str, Any] = field(default_factory=dict)
    participant_count: int | None = None
    participant_names: list[str] = field(default_factory=list)
    participant_list: list[dict[str, Any]] = field(default_factory=list)
    most_active_member_name: str | None = None
    most_active_member_message_count: int | None = None
    admin_list: list[dict[str, Any]] = field(default_factory=list)
    chat_type: str | None = None
    silent_duration_seconds: int | None = None
    is_recently_active: bool = False
    media_count_by_type: dict[str, int] = field(default_factory=dict)
    recent_media: list[dict[str, Any]] = field(default_factory=list)
    has_media: bool = False
    is_disappearing_messages: bool = False
    disappearing_ttl: int | None = None
    last_sender_contact_info: dict[str, Any] | None = None
    timezone: str | None = None

    @property
    def is_group(self) -> bool:
        """Determine if chat is a group based on JID pattern."""
        return self.jid.endswith("@g.us")

    def to_dict(self) -> dict[str, Any]:
        """Convert Chat to dictionary for structured output, omitting empty/null fields."""
        result = {
            "jid": self.jid,
            "name": self.name,
            "is_group": self.is_group,
        }

        # Add timestamp fields
        if self.last_message_time:
            result["last_message_time"] = self.last_message_time.isoformat()
        if self.last_message:
            result["last_message"] = self.last_message
        if self.last_message_id:
            result["last_message_id"] = self.last_message_id
        if self.last_sender:
            result["last_sender"] = self.last_sender
        if self.last_sender_name:
            result["last_sender_name"] = self.last_sender_name
        if self.last_is_from_me is not None:
            result["last_is_from_me"] = self.last_is_from_me

        # Add Phase 1 metadata fields
        if self.total_message_count is not None:
            result["total_message_count"] = self.total_message_count
        if self.message_count_today is not None:
            result["message_count_today"] = self.message_count_today
        if self.message_count_last_7_days is not None:
            result["message_count_last_7_days"] = self.message_count_last_7_days
        if self.message_velocity_last_7_days:
            result["message_velocity_last_7_days"] = self.message_velocity_last_7_days
        if self.participant_count is not None:
            result["participant_count"] = self.participant_count
        if self.participant_names:
            result["participant_names"] = self.participant_names
        if self.participant_list:
            result["participant_list"] = self.participant_list
        if self.most_active_member_name:
            result["most_active_member_name"] = self.most_active_member_name
        if self.most_active_member_message_count is not None:
            result["most_active_member_message_count"] = self.most_active_member_message_count
        if self.admin_list:
            result["admin_list"] = self.admin_list
        if self.chat_type:
            result["chat_type"] = self.chat_type
        if self.silent_duration_seconds is not None:
            result["silent_duration_seconds"] = self.silent_duration_seconds
        if self.is_recently_active:
            result["is_recently_active"] = self.is_recently_active
        if self.media_count_by_type:
            result["media_count_by_type"] = self.media_count_by_type
        if self.recent_media:
            result["recent_media"] = self.recent_media
        if self.has_media:
            result["has_media"] = self.has_media
        if self.is_disappearing_messages:
            result["is_disappearing_messages"] = self.is_disappearing_messages
        if self.disappearing_ttl is not None:
            result["disappearing_ttl"] = self.disappearing_ttl
        if self.last_sender_contact_info:
            result["last_sender_contact_info"] = self.last_sender_contact_info
        if self.timezone:
            result["timezone"] = self.timezone

        return result


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
    # Phase 1 metadata fields
    relationship_type: str | None = None
    is_favorite: bool = False
    contact_created_date: datetime | None = None
    shared_group_list: list[str] = field(default_factory=list)
    shared_group_count: int | None = None
    total_message_count: int | None = None
    message_count_today: int | None = None
    message_count_last_7_days: int | None = None
    message_count_last_30_days: int | None = None
    activity_trend: dict[str, Any] = field(default_factory=dict)
    typical_response_time_seconds: int | None = None
    typical_reply_rate: float | None = None
    is_responsive: bool = False
    days_since_last_message: int | None = None
    last_seen_timestamp: datetime | None = None
    timezone: str | None = None
    organization: str | None = None
    status_message: str | None = None
    latest_message_preview: str | None = None
    latest_message_timestamp: datetime | None = None
    has_active_chat: bool = False
    recent_chat_jid: str | None = None

    def to_dict(self) -> dict[str, Any]:
        """Convert Contact to dictionary for structured output, omitting empty/null fields."""
        result = {
            "jid": self.jid,
            "phone_number": self.phone_number,
            "name": self.name,
        }

        # Add optional fields if present
        if self.first_name:
            result["first_name"] = self.first_name
        if self.full_name:
            result["full_name"] = self.full_name
        if self.push_name:
            result["push_name"] = self.push_name
        if self.business_name:
            result["business_name"] = self.business_name
        if self.nickname:
            result["nickname"] = self.nickname

        # Add Phase 1 metadata fields
        if self.relationship_type:
            result["relationship_type"] = self.relationship_type
        if self.is_favorite:
            result["is_favorite"] = self.is_favorite
        if self.contact_created_date:
            result["contact_created_date"] = self.contact_created_date.isoformat()
        if self.shared_group_list:
            result["shared_group_list"] = self.shared_group_list
        if self.shared_group_count is not None:
            result["shared_group_count"] = self.shared_group_count
        if self.total_message_count is not None:
            result["total_message_count"] = self.total_message_count
        if self.message_count_today is not None:
            result["message_count_today"] = self.message_count_today
        if self.message_count_last_7_days is not None:
            result["message_count_last_7_days"] = self.message_count_last_7_days
        if self.message_count_last_30_days is not None:
            result["message_count_last_30_days"] = self.message_count_last_30_days
        if self.activity_trend:
            result["activity_trend"] = self.activity_trend
        if self.typical_response_time_seconds is not None:
            result["typical_response_time_seconds"] = self.typical_response_time_seconds
        if self.typical_reply_rate is not None:
            result["typical_reply_rate"] = self.typical_reply_rate
        if self.is_responsive:
            result["is_responsive"] = self.is_responsive
        if self.days_since_last_message is not None:
            result["days_since_last_message"] = self.days_since_last_message
        if self.last_seen_timestamp:
            result["last_seen_timestamp"] = self.last_seen_timestamp.isoformat()
        if self.timezone:
            result["timezone"] = self.timezone
        if self.organization:
            result["organization"] = self.organization
        if self.status_message:
            result["status_message"] = self.status_message
        if self.latest_message_preview:
            result["latest_message_preview"] = self.latest_message_preview
        if self.latest_message_timestamp:
            result["latest_message_timestamp"] = self.latest_message_timestamp.isoformat()
        if self.has_active_chat:
            result["has_active_chat"] = self.has_active_chat
        if self.recent_chat_jid:
            result["recent_chat_jid"] = self.recent_chat_jid

        return result


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
