"""Utility functions and logging setup for WhatsApp MCP server."""

import logging
import os
from pathlib import Path

# Load .env file from project root - check multiple locations
try:
    from dotenv import load_dotenv

    # Try multiple locations for .env
    possible_paths = [
        Path(__file__).parent.parent.parent / ".env",  # /app/.env (Docker)
        Path(__file__).parent.parent.parent.parent / ".env",  # /whatsapp-mcp-extended/.env (local)
        Path.cwd() / ".env",  # Current working directory
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
    logging.basicConfig(level=level, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
    return logging.getLogger("whatsapp-mcp")


# Initialize logger based on DEBUG env var
logger = setup_logging(os.getenv("DEBUG", "false").lower() == "true")


# Database paths — resolution order:
#   1. WA_STORE_PATH env var (explicit override)
#   2. /app/store  (Docker bind mount)
#   3. <project_root>/whatsapp-bridge/store  (local dev, bridge-relative)
#   4. <project_root>/store  (legacy / symlink)
_wa_store_env = os.getenv("WA_STORE_PATH")
if _wa_store_env:
    _store_path = _wa_store_env
elif os.path.exists("/app/store"):
    _store_path = "/app/store"
else:
    _project_root = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
    _local_dev_path = os.path.join(_project_root, "whatsapp-bridge", "store")
    _legacy_path = os.path.join(_project_root, "store")
    _store_path = _local_dev_path if os.path.exists(_local_dev_path) else _legacy_path

MESSAGES_DB_PATH = os.path.join(_store_path, "messages.db")
WHATSAPP_DB_PATH = os.path.join(_store_path, "whatsapp.db")

# Fail loudly on import if the bridge DB is missing — sqlite3.connect() silently
# creates an empty file at a wrong path, which causes every query to return empty
# results with no error. Set WA_SKIP_DB_CHECK=1 to bypass (tests, lint).
if not os.getenv("WA_SKIP_DB_CHECK") and not os.path.exists(MESSAGES_DB_PATH):
    raise FileNotFoundError(
        f"WhatsApp MCP: messages.db not found at {MESSAGES_DB_PATH}\n"
        f"  Bridge store resolved to: {_store_path}\n"
        f"  Override with: WA_STORE_PATH=/path/to/store\n"
        f"  Bridge not running or store not mounted?"
    )


# Bridge API configuration
_bridge_host = os.getenv("BRIDGE_HOST", "localhost:8080")
if ":" not in _bridge_host:
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
