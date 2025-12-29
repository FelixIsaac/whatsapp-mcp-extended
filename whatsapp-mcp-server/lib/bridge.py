"""Bridge API client for WhatsApp Go bridge."""
from pathlib import Path
try:
    from dotenv import load_dotenv
    possible_paths = [Path(__file__).parent.parent.parent / ".env", Path(__file__).parent.parent.parent.parent / ".env", Path.cwd() / ".env"]
    for env_path in possible_paths:
        if env_path.exists():
            load_dotenv(env_path)
            break
except ImportError:
    pass

import os
from typing import Any

import requests

from .utils import WHATSAPP_API_BASE_URL, logger


class BridgeError(Exception):
    """Exception for bridge API errors."""
    pass


def _get_headers() -> dict[str, str]:
    """Get request headers including API key if configured."""
    headers = {"Content-Type": "application/json"}
    api_key = os.getenv("API_KEY")
    logger.info(f"[BRIDGE-HEADERS] API_KEY loaded: {bool(api_key)}")
    if api_key:
        headers["X-API-Key"] = api_key
    return headers


def send_message(recipient: str, message: str) -> dict[str, Any]:
    """Send a text message via the bridge API.

    Args:
        recipient: WhatsApp JID or phone number.
        message: Message text to send.

    Returns:
        Response with success, message_id, timestamp.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/send",
            json={"recipient": recipient, "message": message},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in send_message: %s", e)
        raise BridgeError(f"Failed to send message: {e}") from e


def send_file(recipient: str, media_path: str) -> dict[str, Any]:
    """Send a file/media via the bridge API.

    Args:
        recipient: WhatsApp JID or phone number.
        media_path: Path to media file.

    Returns:
        Response with success, message_id, timestamp.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/send",
            json={"recipient": recipient, "message": "", "media_path": media_path},
            headers=_get_headers(),
            timeout=60
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in send_file: %s", e)
        raise BridgeError(f"Failed to send file: {e}") from e


def send_reaction(chat_jid: str, message_id: str, emoji: str) -> dict[str, Any]:
    """Send a reaction to a message.

    Args:
        chat_jid: Chat JID containing the message.
        message_id: ID of message to react to.
        emoji: Emoji to react with (empty to remove).

    Returns:
        Response with success status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/reaction",
            json={"chat_jid": chat_jid, "message_id": message_id, "emoji": emoji},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in send_reaction: %s", e)
        raise BridgeError(f"Failed to send reaction: {e}") from e


def edit_message(chat_jid: str, message_id: str, new_content: str) -> dict[str, Any]:
    """Edit a previously sent message.

    Args:
        chat_jid: Chat JID containing the message.
        message_id: ID of message to edit.
        new_content: New message content.

    Returns:
        Response with success status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/edit",
            json={"chat_jid": chat_jid, "message_id": message_id, "new_content": new_content},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in edit_message: %s", e)
        raise BridgeError(f"Failed to edit message: {e}") from e


def delete_message(chat_jid: str, message_id: str, sender_jid: str | None = None) -> dict[str, Any]:
    """Delete/revoke a message.

    Args:
        chat_jid: Chat JID containing the message.
        message_id: ID of message to delete.
        sender_jid: Sender JID (for admin revoking others' messages).

    Returns:
        Response with success status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        payload: dict[str, Any] = {"chat_jid": chat_jid, "message_id": message_id}
        if sender_jid:
            payload["sender_jid"] = sender_jid

        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/delete",
            json=payload,
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in delete_message: %s", e)
        raise BridgeError(f"Failed to delete message: {e}") from e


def mark_read(chat_jid: str, message_ids: list[str], sender_jid: str | None = None) -> dict[str, Any]:
    """Mark messages as read.

    Args:
        chat_jid: Chat JID containing the messages.
        message_ids: List of message IDs to mark as read.
        sender_jid: Sender JID (required for group chats).

    Returns:
        Response with success status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        payload: dict[str, Any] = {"chat_jid": chat_jid, "message_ids": message_ids}
        if sender_jid:
            payload["sender_jid"] = sender_jid

        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/read",
            json=payload,
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in mark_read: %s", e)
        raise BridgeError(f"Failed to mark as read: {e}") from e


def get_group_info(group_jid: str) -> dict[str, Any]:
    """Get information about a group.

    Args:
        group_jid: Group JID.

    Returns:
        Group metadata including participants.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.get(
            f"{WHATSAPP_API_BASE_URL}/group/{group_jid}",
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in get_group_info: %s", e)
        raise BridgeError(f"Failed to get group info: {e}") from e


def create_group(name: str, participants: list[str]) -> dict[str, Any]:
    """Create a new WhatsApp group.

    Args:
        name: Group name.
        participants: List of participant JIDs.

    Returns:
        Response with group info.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/group/create",
            json={"name": name, "participants": participants},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in create_group: %s", e)
        raise BridgeError(f"Failed to create group: {e}") from e


def create_poll(
    chat_jid: str,
    question: str,
    options: list[str],
    multi_select: bool = False
) -> dict[str, Any]:
    """Create and send a poll.

    Args:
        chat_jid: Chat to send poll to.
        question: Poll question.
        options: List of poll options.
        multi_select: Allow multiple selections.

    Returns:
        Response with message_id.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/poll/create",
            json={
                "chat_jid": chat_jid,
                "question": question,
                "options": options,
                "multi_select": multi_select
            },
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in create_poll: %s", e)
        raise BridgeError(f"Failed to create poll: {e}") from e


def send_typing(chat_jid: str, state: str = "typing") -> dict[str, Any]:
    """Send typing indicator to a chat.

    Args:
        chat_jid: Target chat JID.
        state: "typing", "paused", or "recording" (default: "typing").

    Returns:
        Response with success status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/typing",
            json={"chat_jid": chat_jid, "state": state},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in send_typing: %s", e)
        raise BridgeError(f"Failed to send typing indicator: {e}") from e


def set_about_text(text: str) -> dict[str, Any]:
    """Set profile "About" status text.

    Args:
        text: The new about/status text for the profile.

    Returns:
        Response with success status and text.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/set-about",
            json={"text": text},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in set_about_text: %s", e)
        raise BridgeError(f"Failed to set about text: {e}") from e


def set_disappearing_timer(chat_jid: str, duration: str) -> dict[str, Any]:
    """Set disappearing messages timer for a chat.

    Args:
        chat_jid: Target chat JID.
        duration: Timer duration - "off", "24h", "7d", or "90d".

    Returns:
        Response with success status, chat_jid, and duration.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/disappearing",
            json={"chat_jid": chat_jid, "duration": duration},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in set_disappearing_timer: %s", e)
        raise BridgeError(f"Failed to set disappearing timer: {e}") from e


def get_privacy_settings() -> dict[str, Any]:
    """Fetch the user's privacy settings.

    Returns:
        Response with success status and privacy settings dict containing:
        - group_add: Who can add you to groups
        - last_seen: Who can see your last seen
        - status: Who can see your status
        - profile: Who can see your profile picture
        - read_receipts: Who can see read receipts
        - call_add: Who can call you
        - online: Who can see your online status

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.get(
            f"{WHATSAPP_API_BASE_URL}/privacy",
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in get_privacy_settings: %s", e)
        raise BridgeError(f"Failed to fetch privacy settings: {e}") from e


def pin_chat(chat_jid: str, pin: bool = True) -> dict[str, Any]:
    """Pin or unpin a chat.

    Args:
        chat_jid: Target chat JID.
        pin: True to pin, False to unpin (default: True).

    Returns:
        Response with success status, chat_jid, and pin status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/pin",
            json={"chat_jid": chat_jid, "pin": pin},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in pin_chat: %s", e)
        raise BridgeError(f"Failed to pin chat: {e}") from e


def mute_chat(chat_jid: str, mute: bool = True, duration: str = "forever") -> dict[str, Any]:
    """Mute or unmute a chat.

    Args:
        chat_jid: Target chat JID.
        mute: True to mute, False to unmute (default: True).
        duration: Mute duration - "forever", "15m", "1h", "8h", "1w" (default: "forever", ignored if mute=False).

    Returns:
        Response with success status, chat_jid, mute, and duration.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/mute",
            json={"chat_jid": chat_jid, "mute": mute, "duration": duration},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in mute_chat: %s", e)
        raise BridgeError(f"Failed to mute chat: {e}") from e


def archive_chat(chat_jid: str, archive: bool = True) -> dict[str, Any]:
    """Archive or unarchive a chat.

    Args:
        chat_jid: Target chat JID.
        archive: True to archive, False to unarchive (default: True).

    Returns:
        Response with success status, chat_jid, and archive status.

    Raises:
        BridgeError: If API call fails.
    """
    try:
        response = requests.post(
            f"{WHATSAPP_API_BASE_URL}/archive",
            json={"chat_jid": chat_jid, "archive": archive},
            headers=_get_headers(),
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error("Bridge API error in archive_chat: %s", e)
        raise BridgeError(f"Failed to archive chat: {e}") from e
