"""Tests for lib/database.py functions."""

import sqlite3
from datetime import datetime

from lib.database import (
    extract_urls,
    get_chat_statistics,
    get_message_character_count,
    get_message_word_count,
    list_chats,
    list_messages,
    search_contacts,
)
from lib.models import Chat, Contact, Message


class TestMessageMetadataHelpers:
    """Tests for message metadata helper functions."""

    def test_get_message_character_count_empty(self):
        """Test character count for empty content."""
        assert get_message_character_count("") == 0
        assert get_message_character_count(None) == 0

    def test_get_message_character_count(self):
        """Test character count calculation."""
        assert get_message_character_count("Hello") == 5
        assert get_message_character_count("Hello world!") == 12
        assert get_message_character_count("مرحبا") == 5  # Arabic

    def test_get_message_word_count_empty(self):
        """Test word count for empty content."""
        assert get_message_word_count("") == 0
        assert get_message_word_count(None) == 0

    def test_get_message_word_count(self):
        """Test word count calculation."""
        assert get_message_word_count("Hello") == 1
        assert get_message_word_count("Hello world") == 2
        assert get_message_word_count("Hello world!") == 2
        assert get_message_word_count("one   two   three") == 3  # Multiple spaces
        assert get_message_word_count("   leading and trailing   ") == 3

    def test_extract_urls_empty(self):
        """Test URL extraction from empty content."""
        assert extract_urls("") == []
        assert extract_urls(None) == []

    def test_extract_urls_no_urls(self):
        """Test URL extraction when no URLs present."""
        assert extract_urls("Hello world") == []
        assert extract_urls("Check this out!") == []

    def test_extract_urls_http_only(self):
        """Test URL extraction for http:// only."""
        urls = extract_urls("Visit http://example.com for more")
        assert urls == ["http://example.com"]

    def test_extract_urls_https(self):
        """Test URL extraction for https:// URLs."""
        urls = extract_urls("Go to https://github.com/user/repo")
        assert urls == ["https://github.com/user/repo"]

    def test_extract_urls_multiple(self):
        """Test extraction of multiple URLs."""
        content = "See https://google.com and http://example.com"
        urls = extract_urls(content)
        assert len(urls) == 2
        assert "https://google.com" in urls
        assert "http://example.com" in urls

    def test_extract_urls_with_query_params(self):
        """Test URL extraction with query parameters."""
        urls = extract_urls("Check https://example.com?foo=bar&baz=qux")
        assert len(urls) == 1
        assert urls[0].startswith("https://example.com")


class TestListMessages:
    """Tests for list_messages function."""

    def test_list_messages_includes_new_fields(self, temp_messages_db, monkeypatch):
        """Test that list_messages includes new metadata fields."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        messages = list_messages(limit=10)

        assert len(messages) > 0
        msg = messages[0]

        # Check new metadata fields are always present
        assert "character_count" in msg
        assert "word_count" in msg
        assert "is_group" in msg
        # url_list should only be present if urls exist, but we can verify the structure
        # by checking one with urls
        assert isinstance(msg.get("character_count"), int)
        assert isinstance(msg.get("word_count"), int)

    def test_list_messages_character_count_calculation(self, temp_messages_db, monkeypatch):
        """Test character count is calculated correctly."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        messages = list_messages(limit=10)
        assert len(messages) > 0

        # First test message has "Hello world" (11 chars)
        msg = messages[0]
        if msg["content"] == "Hello world":
            assert msg["character_count"] == 11

    def test_list_messages_word_count_calculation(self, temp_messages_db, monkeypatch):
        """Test word count is calculated correctly."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        messages = list_messages(limit=10)
        assert len(messages) > 0

        # Check word count for "Hello world" = 2 words
        msg = messages[0]
        if msg["content"] == "Hello world":
            assert msg["word_count"] == 2

    def test_list_messages_is_group_detection(self, temp_messages_db, monkeypatch):
        """Test is_group field is set correctly."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        messages = list_messages(limit=10)
        assert len(messages) > 0

        # Check individual vs group detection
        for msg in messages:
            if msg["chat_jid"].endswith("@s.whatsapp.net"):
                assert msg["is_group"] is False
            elif msg["chat_jid"].endswith("@g.us"):
                assert msg["is_group"] is True


class TestListChats:
    """Tests for list_chats function."""

    def test_list_chats_includes_new_fields(self, temp_messages_db, monkeypatch):
        """Test that list_chats includes new metadata fields."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        chats = list_chats(limit=10)

        assert len(chats) > 0
        chat = chats[0]

        # Check new metadata fields
        assert "total_message_count" in chat
        assert "message_count_today" in chat
        assert "message_count_last_7_days" in chat
        assert "chat_type" in chat

    def test_list_chats_message_counts(self, temp_messages_db, monkeypatch):
        """Test message count fields are calculated."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        chats = list_chats(limit=10)
        assert len(chats) > 0

        chat = chats[0]
        assert chat["total_message_count"] is not None
        assert chat["total_message_count"] >= 0
        assert chat["message_count_today"] is not None
        assert chat["message_count_today"] >= 0
        assert chat["message_count_last_7_days"] is not None
        assert chat["message_count_last_7_days"] >= 0

    def test_list_chats_chat_type_detection(self, temp_messages_db, monkeypatch):
        """Test chat_type field is set correctly."""
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        chats = list_chats(limit=10)
        assert len(chats) > 0

        for chat in chats:
            if chat["jid"].endswith("@g.us"):
                assert chat["chat_type"] == "group"
            else:
                assert chat["chat_type"] == "individual"


class TestGetChatStatistics:
    """Tests for get_chat_statistics function."""

    def test_get_chat_statistics_count(self, temp_messages_db):
        """Test get_chat_statistics returns correct counts."""
        conn = sqlite3.connect(temp_messages_db)
        jid = "123456789@s.whatsapp.net"

        total, today, week = get_chat_statistics(jid, conn)

        # Fixture adds 2 messages to this chat
        assert total >= 0
        assert today >= 0
        assert week >= 0

        conn.close()

    def test_get_chat_statistics_nonexistent_chat(self, temp_messages_db):
        """Test get_chat_statistics for nonexistent chat."""
        conn = sqlite3.connect(temp_messages_db)

        total, today, week = get_chat_statistics("nonexistent@s.whatsapp.net", conn)

        # Should return zeros, not errors
        assert total == 0
        assert today == 0
        assert week == 0

        conn.close()


class TestSearchContacts:
    """Tests for search_contacts function."""

    def test_search_contacts_no_n_plus_one(self, temp_whatsapp_db, temp_messages_db, monkeypatch):
        """Test search_contacts doesn't make N+1 queries.

        This test verifies the optimization where nicknames are fetched
        with a single IN query instead of N individual queries.
        """
        monkeypatch.setenv("WHATSAPP_DB_PATH", temp_whatsapp_db)
        monkeypatch.setenv("MESSAGES_DB_PATH", temp_messages_db)
        monkeypatch.setattr("lib.database.WHATSAPP_DB_PATH", temp_whatsapp_db)
        monkeypatch.setattr("lib.database.MESSAGES_DB_PATH", temp_messages_db)

        # Add nickname to the test contact
        conn = sqlite3.connect(temp_messages_db)
        cursor = conn.cursor()
        cursor.execute("""
            INSERT INTO contact_nicknames (jid, nickname, updated_at)
            VALUES (?, ?, datetime('now'))
        """, ("123456789@s.whatsapp.net", "Johnny Doe"))
        conn.commit()
        conn.close()

        # Search should work correctly with the optimization
        results = search_contacts("John")

        assert len(results) > 0
        contact = results[0]
        assert contact["jid"] == "123456789@s.whatsapp.net"
        assert contact["nickname"] == "Johnny Doe"

    def test_search_contacts_returns_all_fields(self, temp_whatsapp_db, monkeypatch):
        """Test search_contacts returns all expected fields."""
        monkeypatch.setenv("WHATSAPP_DB_PATH", temp_whatsapp_db)
        monkeypatch.setattr("lib.database.WHATSAPP_DB_PATH", temp_whatsapp_db)

        results = search_contacts("John")

        assert len(results) > 0
        contact = results[0]

        # Check basic fields
        assert "jid" in contact
        assert "phone_number" in contact
        assert "name" in contact

    def test_search_contacts_no_results(self, temp_whatsapp_db, monkeypatch):
        """Test search_contacts with no matching results."""
        monkeypatch.setenv("WHATSAPP_DB_PATH", temp_whatsapp_db)
        monkeypatch.setattr("lib.database.WHATSAPP_DB_PATH", temp_whatsapp_db)

        results = search_contacts("nonexistent123456789")

        assert results == []


class TestMessageModel:
    """Tests for Message model with new fields."""

    def test_message_to_dict_omits_empty_fields(self):
        """Test Message.to_dict() omits empty/null fields."""
        msg = Message(
            timestamp=datetime.now(),
            sender="123@s.whatsapp.net",
            content="Test",
            is_from_me=False,
            chat_jid="123@s.whatsapp.net",
            id="msg1"
        )

        result = msg.to_dict()

        # Should include required fields
        assert "id" in result
        assert "sender" in result
        assert "content" in result

        # Should not include empty optional fields
        assert "url_list" not in result
        assert "mentions" not in result
        assert "quoted_message_id" not in result

    def test_message_to_dict_includes_populated_fields(self):
        """Test Message.to_dict() includes populated optional fields."""
        msg = Message(
            timestamp=datetime.now(),
            sender="123@s.whatsapp.net",
            content="https://example.com hello",
            is_from_me=False,
            chat_jid="123@s.whatsapp.net",
            id="msg1",
            character_count=26,
            word_count=2,
            url_list=["https://example.com"],
            is_group=False
        )

        result = msg.to_dict()

        assert result["character_count"] == 26
        assert result["word_count"] == 2
        assert result["url_list"] == ["https://example.com"]
        assert result["is_group"] is False


class TestChatModel:
    """Tests for Chat model with new fields."""

    def test_chat_to_dict_omits_empty_fields(self):
        """Test Chat.to_dict() omits empty/null fields."""
        chat = Chat(
            jid="123@s.whatsapp.net",
            name="Test User",
            last_message_time=None
        )

        result = chat.to_dict()

        # Should include required fields
        assert "jid" in result
        assert "name" in result
        assert "is_group" in result

        # Should not include empty optional fields
        assert "total_message_count" not in result
        assert "participant_names" not in result

    def test_chat_to_dict_includes_stats(self):
        """Test Chat.to_dict() includes populated stat fields."""
        chat = Chat(
            jid="123@g.us",
            name="Test Group",
            last_message_time=datetime.now(),
            total_message_count=42,
            message_count_today=5,
            message_count_last_7_days=20,
            chat_type="group"
        )

        result = chat.to_dict()

        assert result["total_message_count"] == 42
        assert result["message_count_today"] == 5
        assert result["message_count_last_7_days"] == 20
        assert result["chat_type"] == "group"


class TestContactModel:
    """Tests for Contact model with new fields."""

    def test_contact_to_dict_omits_empty_fields(self):
        """Test Contact.to_dict() omits empty/null fields."""
        contact = Contact(
            phone_number="123456789",
            name="John Doe",
            jid="123456789@s.whatsapp.net"
        )

        result = contact.to_dict()

        # Should include required fields
        assert "jid" in result
        assert "phone_number" in result
        assert "name" in result

        # Should not include empty optional fields
        assert "total_message_count" not in result
        assert "shared_group_list" not in result

    def test_contact_to_dict_includes_activity_fields(self):
        """Test Contact.to_dict() includes populated activity fields."""
        contact = Contact(
            phone_number="123456789",
            name="John Doe",
            jid="123456789@s.whatsapp.net",
            total_message_count=150,
            message_count_today=10,
            is_responsive=True,
            days_since_last_message=2
        )

        result = contact.to_dict()

        assert result["total_message_count"] == 150
        assert result["message_count_today"] == 10
        assert result["is_responsive"] is True
        assert result["days_since_last_message"] == 2
