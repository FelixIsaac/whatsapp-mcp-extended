"""Utility functions and logging setup for WhatsApp MCP server."""

import logging
import os
from pathlib import Path

# Load .env file from project root - check multiple locations
try:
    from dotenv import load_dotenv
    
    # Try multiple locations for .env
    possible_paths = [
        Path(__file__).parent.parent.parent / '.env',  # /app/.env (Docker)
        Path(__file__).parent.parent.parent.parent / '.env',  # /whatsapp-mcp-extended/.env (local)
        Path.cwd() / '.env',  # Current working directory
    ]
    
    for env_path in possible_paths:
        if env_path.exists():
            load_dotenv(env_path)
            break
except ImportError:
    pass  # python-dotenv not available, continue without it


# Set up logging
def setup_logging(debug: bool = False) -> logging.Logger:
    """Configure and return the application logger.

    Args:
        debug: Enable DEBUG level logging if True.

    Returns:
        Configured logger instance.
    """
    level = logging.DEBUG if debug else logging.INFO
    logging.basicConfig(
        level=level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    return logging.getLogger("whatsapp-mcp")


# Initialize logger based on DEBUG env var
logger = setup_logging(os.getenv("DEBUG", "false").lower() == "true")


# Database paths - check for Docker (/app) or local development
if os.path.exists('/app/store'):
    # Docker environment
    MESSAGES_DB_PATH = '/app/store/messages.db'
    WHATSAPP_DB_PATH = '/app/store/whatsapp.db'
else:
    # Local development - use relative path from project root
    _store_path = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), 'store')
    MESSAGES_DB_PATH = os.path.join(_store_path, 'messages.db')
    WHATSAPP_DB_PATH = os.path.join(_store_path, 'whatsapp.db')


# Bridge API configuration
_bridge_host = os.getenv('BRIDGE_HOST', 'localhost:8080')
if ':' not in _bridge_host:
    _bridge_host = f"{_bridge_host}:8080"
BRIDGE_HOST = _bridge_host
WHATSAPP_API_BASE_URL = f"http://{BRIDGE_HOST}/api"


# Bridge API configuration
_bridge_host = os.getenv('BRIDGE_HOST', 'localhost:8080')
if ':' not in _bridge_host:
    _bridge_host = f"{_bridge_host}:8080"
BRIDGE_HOST = _bridge_host
WHATSAPP_API_BASE_URL = f"http://{BRIDGE_HOST}/api"

def get_sender_name(sender_jid: str) -> str:
    """Get display name for a sender JID.

    Args:
        sender_jid: WhatsApp JID of the sender.

    Returns:
        Display name or phone number if name not found.
    """
    from .database import get_contact_by_jid

    if sender_jid.endswith("@s.whatsapp.net"):
        contact = get_contact_by_jid(sender_jid)
        if contact:
            return contact.name or contact.push_name or contact.phone_number
    return sender_jid.split("@")[0]
