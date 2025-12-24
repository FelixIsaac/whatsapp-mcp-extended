"""Tests for lib/bridge.py API client."""

from unittest.mock import MagicMock, patch

import pytest

from lib.bridge import (
    BridgeError,
    delete_message,
    edit_message,
    get_group_info,
    send_message,
    send_reaction,
)


class TestSendMessage:
    """Tests for send_message function."""

    @patch("lib.bridge.requests.post")
    def test_send_message_success(self, mock_post):
        """Test successful message sending."""
        mock_response = MagicMock()
        mock_response.json.return_value = {
            "success": True,
            "message_id": "msg123",
            "timestamp": 1234567890,
        }
        mock_response.raise_for_status = MagicMock()
        mock_post.return_value = mock_response

        result = send_message("123456789@s.whatsapp.net", "Hello")

        assert result["success"] is True
        assert result["message_id"] == "msg123"
        mock_post.assert_called_once()

    @patch("lib.bridge.requests.post")
    def test_send_message_failure(self, mock_post):
        """Test message sending failure raises BridgeError."""
        import requests

        mock_post.side_effect = requests.RequestException("Connection refused")

        with pytest.raises(BridgeError) as exc_info:
            send_message("123456789@s.whatsapp.net", "Hello")

        assert "Failed to send message" in str(exc_info.value)


class TestSendReaction:
    """Tests for send_reaction function."""

    @patch("lib.bridge.requests.post")
    def test_send_reaction_success(self, mock_post):
        """Test successful reaction sending."""
        mock_response = MagicMock()
        mock_response.json.return_value = {"success": True}
        mock_response.raise_for_status = MagicMock()
        mock_post.return_value = mock_response

        result = send_reaction("123@s.whatsapp.net", "msg123", "👍")

        assert result["success"] is True

    @patch("lib.bridge.requests.post")
    def test_remove_reaction(self, mock_post):
        """Test removing reaction with empty emoji."""
        mock_response = MagicMock()
        mock_response.json.return_value = {"success": True}
        mock_response.raise_for_status = MagicMock()
        mock_post.return_value = mock_response

        result = send_reaction("123@s.whatsapp.net", "msg123", "")

        assert result["success"] is True


class TestEditMessage:
    """Tests for edit_message function."""

    @patch("lib.bridge.requests.post")
    def test_edit_message_success(self, mock_post):
        """Test successful message editing."""
        mock_response = MagicMock()
        mock_response.json.return_value = {"success": True}
        mock_response.raise_for_status = MagicMock()
        mock_post.return_value = mock_response

        result = edit_message("123@s.whatsapp.net", "msg123", "New content")

        assert result["success"] is True


class TestDeleteMessage:
    """Tests for delete_message function."""

    @patch("lib.bridge.requests.post")
    def test_delete_message_success(self, mock_post):
        """Test successful message deletion."""
        mock_response = MagicMock()
        mock_response.json.return_value = {"success": True}
        mock_response.raise_for_status = MagicMock()
        mock_post.return_value = mock_response

        result = delete_message("123@s.whatsapp.net", "msg123")

        assert result["success"] is True

    @patch("lib.bridge.requests.post")
    def test_delete_message_with_sender(self, mock_post):
        """Test message deletion with sender JID for groups."""
        mock_response = MagicMock()
        mock_response.json.return_value = {"success": True}
        mock_response.raise_for_status = MagicMock()
        mock_post.return_value = mock_response

        result = delete_message("123@g.us", "msg123", sender_jid="456@s.whatsapp.net")

        assert result["success"] is True
        call_args = mock_post.call_args
        assert "sender_jid" in call_args.kwargs["json"]


class TestGetGroupInfo:
    """Tests for get_group_info function."""

    @patch("lib.bridge.requests.get")
    def test_get_group_info_success(self, mock_get):
        """Test successful group info retrieval."""
        mock_response = MagicMock()
        mock_response.json.return_value = {
            "success": True,
            "name": "Test Group",
            "participant_count": 5,
        }
        mock_response.raise_for_status = MagicMock()
        mock_get.return_value = mock_response

        result = get_group_info("123456789@g.us")

        assert result["success"] is True
        assert result["name"] == "Test Group"
