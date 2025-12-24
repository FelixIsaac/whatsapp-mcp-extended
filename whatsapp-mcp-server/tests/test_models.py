"""Tests for lib/models.py data classes."""

from datetime import datetime

from lib.models import Chat, Contact, Message, MessageContext


class TestMessage:
    """Tests for Message dataclass."""

    def test_message_to_dict(self):
        """Test Message.to_dict() returns correct structure."""
        msg = Message(
            timestamp=datetime(2024, 1, 15, 10, 30, 0),
            sender="123456789@s.whatsapp.net",
            content="Hello world",
            is_from_me=False,
            chat_jid="123456789@s.whatsapp.net",
            id="msg123",
            chat_name="Test User",
        )

        result = msg.to_dict()

        assert result["id"] == "msg123"
        assert result["content"] == "Hello world"
        assert result["sender"] == "123456789@s.whatsapp.net"
        assert result["is_from_me"] is False
        assert result["timestamp"] == "2024-01-15T10:30:00"
        assert "media_type" in result

    def test_message_with_media(self):
        """Test Message.to_dict() includes media fields when present."""
        msg = Message(
            timestamp=datetime(2024, 1, 15, 10, 30, 0),
            sender="123@s.whatsapp.net",
            content="",
            is_from_me=True,
            chat_jid="123@s.whatsapp.net",
            id="msg456",
            media_type="image",
            filename="photo.jpg",
            file_length=1024,
        )

        result = msg.to_dict()

        assert result["media_type"] == "image"
        assert result["filename"] == "photo.jpg"
        assert result["file_length"] == 1024


class TestChat:
    """Tests for Chat dataclass."""

    def test_chat_to_dict(self):
        """Test Chat.to_dict() returns correct structure."""
        chat = Chat(
            jid="123456789@s.whatsapp.net",
            name="Test User",
            last_message_time=datetime(2024, 1, 15, 10, 30, 0),
            last_message="Hello",
        )

        result = chat.to_dict()

        assert result["jid"] == "123456789@s.whatsapp.net"
        assert result["name"] == "Test User"
        assert result["is_group"] is False
        assert result["last_message"] == "Hello"

    def test_chat_is_group(self):
        """Test Chat.is_group property."""
        individual = Chat(jid="123@s.whatsapp.net", name="User", last_message_time=None)
        group = Chat(jid="123@g.us", name="Group", last_message_time=None)

        assert individual.is_group is False
        assert group.is_group is True


class TestContact:
    """Tests for Contact dataclass."""

    def test_contact_to_dict(self):
        """Test Contact.to_dict() returns correct structure."""
        contact = Contact(
            phone_number="123456789",
            name="John Doe",
            jid="123456789@s.whatsapp.net",
            first_name="John",
            full_name="John Doe",
            push_name="Johnny",
            nickname="JD",
        )

        result = contact.to_dict()

        assert result["phone_number"] == "123456789"
        assert result["name"] == "John Doe"
        assert result["nickname"] == "JD"
        assert result["jid"] == "123456789@s.whatsapp.net"


class TestMessageContext:
    """Tests for MessageContext dataclass."""

    def test_message_context_to_dict(self):
        """Test MessageContext.to_dict() includes all messages."""
        target = Message(
            timestamp=datetime(2024, 1, 15, 10, 30, 0),
            sender="123@s.whatsapp.net",
            content="Target",
            is_from_me=False,
            chat_jid="123@s.whatsapp.net",
            id="msg2",
        )
        before = Message(
            timestamp=datetime(2024, 1, 15, 10, 29, 0),
            sender="123@s.whatsapp.net",
            content="Before",
            is_from_me=False,
            chat_jid="123@s.whatsapp.net",
            id="msg1",
        )
        after = Message(
            timestamp=datetime(2024, 1, 15, 10, 31, 0),
            sender="123@s.whatsapp.net",
            content="After",
            is_from_me=False,
            chat_jid="123@s.whatsapp.net",
            id="msg3",
        )

        ctx = MessageContext(message=target, before=[before], after=[after])
        result = ctx.to_dict()

        assert result["message"]["id"] == "msg2"
        assert len(result["before"]) == 1
        assert len(result["after"]) == 1
        assert result["before"][0]["id"] == "msg1"
        assert result["after"][0]["id"] == "msg3"
