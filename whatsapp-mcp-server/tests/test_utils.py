"""Tests for lib/utils.py utility functions."""

import logging
import os
from unittest.mock import patch

from lib.utils import setup_logging, get_sender_name


class TestSetupLogging:
    """Tests for setup_logging function."""

    def test_setup_logging_default(self):
        """Test default logging setup (INFO level)."""
        logger = setup_logging(debug=False)

        assert logger.name == "whatsapp-mcp"
        assert logger.level <= logging.INFO

    def test_setup_logging_debug(self):
        """Test debug logging setup."""
        logger = setup_logging(debug=True)

        assert logger.level <= logging.DEBUG


class TestGetSenderName:
    """Tests for get_sender_name function."""

    @patch("lib.utils.get_contact_by_jid")
    def test_get_sender_name_with_contact(self, mock_get_contact):
        """Test getting sender name when contact exists."""
        from lib.models import Contact

        mock_get_contact.return_value = Contact(
            jid="123456789@s.whatsapp.net",
            phone_number="123456789",
            name="John Doe",
            push_name="Johnny",
        )

        result = get_sender_name("123456789@s.whatsapp.net")

        assert result == "John Doe"

    @patch("lib.utils.get_contact_by_jid")
    def test_get_sender_name_no_contact(self, mock_get_contact):
        """Test getting sender name when contact doesn't exist."""
        mock_get_contact.return_value = None

        result = get_sender_name("123456789@s.whatsapp.net")

        assert result == "123456789"

    def test_get_sender_name_non_individual(self):
        """Test getting sender name for non-individual JID."""
        result = get_sender_name("123456789@g.us")

        assert result == "123456789"
