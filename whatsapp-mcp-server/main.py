"""WhatsApp MCP Server - stdio transport for Claude Code CLI"""

from pathlib import Path
from typing import Any

from mcp.server.fastmcp import FastMCP
from mcp.server.fastmcp.utilities.types import Image
from mcp.types import ToolAnnotations

# Phase 2: Group Management
from whatsapp import add_group_members as whatsapp_add_group_members
from whatsapp import create_group as whatsapp_create_group
from whatsapp import create_newsletter as whatsapp_create_newsletter

# Phase 3: Polls
from whatsapp import create_poll as whatsapp_create_poll
from whatsapp import delete_message as whatsapp_delete_message
from whatsapp import demote_admin as whatsapp_demote_admin
from whatsapp import download_media as whatsapp_download_media
from whatsapp import edit_message as whatsapp_edit_message
from whatsapp import follow_newsletter as whatsapp_follow_newsletter
from whatsapp import get_blocklist as whatsapp_get_blocklist
from whatsapp import get_chat as whatsapp_get_chat
from whatsapp import get_contact_by_jid as whatsapp_get_contact_by_jid
from whatsapp import get_contact_by_phone as whatsapp_get_contact_by_phone
from whatsapp import get_contact_chats as whatsapp_get_contact_chats
from whatsapp import get_contact_nickname as whatsapp_get_contact_nickname
from whatsapp import get_direct_chat_by_contact as whatsapp_get_direct_chat_by_contact
from whatsapp import get_group_info as whatsapp_get_group_info
from whatsapp import get_last_interaction as whatsapp_get_last_interaction
from whatsapp import get_message_context as whatsapp_get_message_context
from whatsapp import get_profile_picture as whatsapp_get_profile_picture
from whatsapp import leave_group as whatsapp_leave_group
from whatsapp import list_all_contacts as whatsapp_list_all_contacts
from whatsapp import list_chats as whatsapp_list_chats
from whatsapp import list_contact_nicknames as whatsapp_list_contact_nicknames
from whatsapp import list_messages as whatsapp_list_messages
from whatsapp import mark_messages_read as whatsapp_mark_messages_read
from whatsapp import promote_to_admin as whatsapp_promote_to_admin
from whatsapp import remove_contact_nickname as whatsapp_remove_contact_nickname
from whatsapp import remove_group_members as whatsapp_remove_group_members

# Phase 4: History Sync
from whatsapp import request_chat_history as whatsapp_request_chat_history
from whatsapp import search_contacts as whatsapp_search_contacts
from whatsapp import send_audio_message as whatsapp_audio_voice_message
from whatsapp import send_file as whatsapp_send_file
from whatsapp import send_message as whatsapp_send_message
from whatsapp import send_reaction as whatsapp_send_reaction  # Phase 1 features
from whatsapp import set_contact_nickname as whatsapp_set_contact_nickname

# Phase 5: Advanced Features
from whatsapp import set_presence as whatsapp_set_presence
from whatsapp import subscribe_presence as whatsapp_subscribe_presence
from whatsapp import unfollow_newsletter as whatsapp_unfollow_newsletter
from whatsapp import update_blocklist as whatsapp_update_blocklist
from whatsapp import update_group as whatsapp_update_group

_INLINE_IMAGE_EXTS = {".jpg", ".jpeg", ".png", ".gif", ".webp"}

# Initialize FastMCP server
mcp = FastMCP("whatsapp-extended")

READ_ONLY = ToolAnnotations(readOnlyHint=True, destructiveHint=False, idempotentHint=True, openWorldHint=False)
WRITE = ToolAnnotations(readOnlyHint=False, destructiveHint=False, idempotentHint=False, openWorldHint=True)
DESTRUCTIVE = ToolAnnotations(readOnlyHint=False, destructiveHint=True, idempotentHint=False, openWorldHint=True)


def _noop_tool(*args: Any, **kwargs: Any) -> Any:
    if args and callable(args[0]) and len(args) == 1 and not kwargs:
        return args[0]

    def decorator(func: Any) -> Any:
        return func

    return decorator


def advanced_tool(*args: Any, **kwargs: Any) -> Any:
    """Register advanced/destructive MCP tools."""
    return mcp.tool(*args, **kwargs)


def _invalid_action(action: str, allowed: tuple[str, ...], replacement: str | None = None) -> dict[str, Any]:
    result: dict[str, Any] = {
        "success": False,
        "error": f"Invalid action '{action}'. Use one of: {', '.join(allowed)}",
        "allowed_actions": list(allowed),
    }
    if replacement:
        result["use_tool"] = replacement
    return result


@mcp.tool(annotations=READ_ONLY)
def search_contacts(query: str) -> list[dict[str, Any]]:
    """Search WhatsApp contacts by name or phone number.

    Args:
        query: Search term to match against contact names or phone numbers

    Returns:
        List of contact dicts with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname

    Hints:
        - Use returned jid with `list_messages` to filter messages by contact
        - Use `get_contact_details` for more detailed contact info
        - Use `get_last_interaction` to find most recent message with contact
    """
    return whatsapp_search_contacts(query)


@mcp.tool(annotations=READ_ONLY)
def list_messages(
    after: str | None = None,
    before: str | None = None,
    chat_jid: str | None = None,
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
) -> list[dict[str, Any]]:
    """Get WhatsApp messages matching specified criteria.

    Args:
        after: Optional ISO-8601 formatted string to only return messages after this date
        before: Optional ISO-8601 formatted string to only return messages before this date
        chat_jid: Optional chat JID to filter messages by chat
        query: Optional search term to filter messages by content
        limit: Maximum number of messages to return (default 20)
        page: Page number for pagination (default 0)

    Hints:
        - If expected messages are missing, use `request_history` to sync older messages from phone
        - Use `get_message_context` to get surrounding messages for a specific message_id
        - Use `search_contacts` first to find a contact's JID for filtering
    """
    messages = whatsapp_list_messages(
        after=after,
        before=before,
        sender_phone_number=None,
        chat_jid=chat_jid,
        query=query,
        limit=limit,
        page=page,
        include_context=False,
        context_before=1,
        context_after=1,
    )
    return messages


@mcp.tool(annotations=READ_ONLY)
def list_chats(
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active",
) -> list[dict[str, Any]]:
    """Get WhatsApp chats matching specified criteria.

    Args:
        query: Optional search term to filter chats by name or JID
        limit: Maximum number of chats to return (default 20)
        page: Page number for pagination (default 0)
        include_last_message: Whether to include the last message in each chat (default True)
        sort_by: Field to sort results by, either "last_active" or "name" (default "last_active")

    Hints:
        - Use returned jid with `list_messages` to get all messages in a specific chat
        - Use `get_chat` for detailed metadata of a specific chat by JID
        - For group chats (jid ends with @g.us), use `get_group_info` for participant list
    """
    chats = whatsapp_list_chats(
        query=query, limit=limit, page=page, include_last_message=include_last_message, sort_by=sort_by
    )
    return chats


@mcp.tool(annotations=READ_ONLY)
def get_chat(chat_jid: str, include_last_message: bool = True) -> dict[str, Any]:
    """Get WhatsApp chat metadata by JID.

    Args:
        chat_jid: The JID of the chat to retrieve
        include_last_message: Whether to include the last message (default True)
    """
    chat = whatsapp_get_chat(chat_jid, include_last_message)
    return chat


@mcp.tool(annotations=READ_ONLY)
def get_message_context(message_id: str, before: int = 5, after: int = 5) -> dict[str, Any]:
    """Get context around a specific WhatsApp message.

    Args:
        message_id: The ID of the message to get context for
        before: Number of messages to include before the target message (default 5)
        after: Number of messages to include after the target message (default 5)

    Hints:
        - Get message_id from `list_messages` results first
        - Use for understanding conversation flow around a specific message
        - Useful after finding a message via search to see full context
    """
    context = whatsapp_get_message_context(message_id, before, after)
    return context


@mcp.tool(annotations=WRITE)
def send_message(recipient: str, message: str, mentioned_jids: list[str] | None = None) -> dict[str, Any]:
    """Send a WhatsApp message to a person or group. For group chats use the JID.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        message: The message text to send
        mentioned_jids: Optional list of JIDs to mention in the message (e.g. ["123456789@s.whatsapp.net"])

    Returns:
        A dictionary containing success status and a status message
    """
    return whatsapp_send_message(recipient, message, mentioned_jids)


@mcp.tool(annotations=WRITE)
def send_file(recipient: str, media_path: str) -> dict[str, Any]:
    """Send a file such as a picture, raw audio, video or document via WhatsApp to the specified recipient. For group messages use the JID.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        media_path: The absolute path to the media file to send (image, video, document)

    Returns:
        A dictionary containing success status and a status message
    """
    return whatsapp_send_file(recipient, media_path)


@mcp.tool(annotations=WRITE)
def send_audio_message(recipient: str, media_path: str) -> dict[str, Any]:
    """Send any audio file as a WhatsApp audio message to the specified recipient. For group messages use the JID. If it errors due to ffmpeg not being installed, use send_file instead.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        media_path: The absolute path to the audio file to send (will be converted to Opus .ogg if it's not a .ogg file)

    Returns:
        A dictionary containing success status and a status message
    """
    return whatsapp_audio_voice_message(recipient, media_path)


@mcp.tool(annotations=READ_ONLY)
def download_media(message_id: str, chat_jid: str) -> Any:
    """Download media from a WhatsApp message and get the local file path.

    For image media (jpg/jpeg/png/gif/webp) the response also embeds the image
    inline as an MCP image content block, so MCP clients without filesystem
    access can still view and analyze it without a separate Read step.

    Args:
        message_id: The ID of the message containing the media
        chat_jid: The JID of the chat containing the message

    Returns:
        For images: a list with [inline image, status dict].
        For other media: a status dict with success, message, file_path.

    Hints:
        - Use `list_messages` first to find messages with media_type set (image, video, audio, document)
        - The message_id and chat_jid come from `list_messages` results
        - Check message has media_type before attempting download
    """
    file_path = whatsapp_download_media(message_id, chat_jid)

    if not file_path:
        return {"success": False, "message": "Failed to download media"}

    status = {
        "success": True,
        "message": "Media downloaded successfully",
        "file_path": file_path,
    }

    if Path(file_path).suffix.lower() in _INLINE_IMAGE_EXTS:
        return [Image(path=file_path), status]
    return status


@mcp.tool(annotations=READ_ONLY)
def list_all_contacts(limit: int = 100) -> list[dict[str, Any]]:
    """List all WhatsApp contacts with their information.

    Args:
        limit: Maximum number of contacts to return (default 100)

    Returns:
        List of contact dicts with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname
    """
    return whatsapp_list_all_contacts(limit)


@mcp.tool(
    annotations=READ_ONLY,
    description=(
        "Get contact details plus optional related chats and last interaction. "
        "Prefer this over get_contact_details, get_direct_chat_by_contact, get_contact_chats, and get_last_interaction."
    ),
)
def get_contact_context(
    identifier: str,
    include_chats: bool = False,
    include_last_interaction: bool = False,
    limit: int = 20,
    page: int = 0,
) -> dict[str, Any]:
    """Get contact details and optional chat/message context in one composable call."""
    contact = whatsapp_get_contact_by_jid(identifier)
    if not contact:
        contact = whatsapp_get_contact_by_phone(identifier)

    jid = contact.get("jid") if contact else identifier
    result: dict[str, Any] = {"contact": contact}

    if include_chats:
        result["chats"] = whatsapp_get_contact_chats(jid, limit, page)
        result["direct_chat"] = whatsapp_get_direct_chat_by_contact(jid.split("@")[0])
    if include_last_interaction:
        result["last_interaction"] = whatsapp_get_last_interaction(jid)

    return result


@mcp.tool(
    annotations=WRITE,
    description=(
        "Manage custom contact nicknames. "
        "Use action='set', 'get', 'remove', or 'list'. Prefer this over set_nickname/get_nickname/remove_nickname/list_nicknames."
    ),
)
def manage_nickname(action: str, jid: str | None = None, nickname: str | None = None) -> dict[str, Any]:
    """Manage custom contact nicknames with one action-based tool."""
    allowed = ("set", "get", "remove", "list")
    if action not in allowed:
        return _invalid_action(action, allowed, "manage_nickname")

    if action == "list":
        return {"success": True, "nicknames": whatsapp_list_contact_nicknames()}
    if not jid:
        return {"success": False, "error": "jid is required for set/get/remove", "use_tool": "manage_nickname"}
    if action == "set":
        if nickname is None:
            return {"success": False, "error": "nickname is required for action='set'", "use_tool": "manage_nickname"}
        return whatsapp_set_contact_nickname(jid, nickname)
    if action == "get":
        return {"success": True, "jid": jid, "nickname": whatsapp_get_contact_nickname(jid)}
    return whatsapp_remove_contact_nickname(jid)


# Phase 1 Features: Reactions, Edit, Delete, Group Info, Mark Read


@mcp.tool(annotations=WRITE)
def send_reaction(chat_jid: str, message_id: str, emoji: str) -> dict[str, Any]:
    """Send an emoji reaction to a WhatsApp message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to react to
        emoji: The emoji to react with (empty string to remove reaction)

    Returns:
        A dictionary containing success status, chat_jid, message_id, emoji, and action
    """
    return whatsapp_send_reaction(chat_jid, message_id, emoji)


@advanced_tool(annotations=WRITE)
def edit_message(chat_jid: str, message_id: str, new_content: str) -> dict[str, Any]:
    """Edit a previously sent WhatsApp message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to edit
        new_content: The new message content

    Returns:
        A dictionary containing success status, chat_jid, message_id, and new_content
    """
    return whatsapp_edit_message(chat_jid, message_id, new_content)


@advanced_tool(annotations=DESTRUCTIVE)
def delete_message(chat_jid: str, message_id: str, sender_jid: str | None = None) -> dict[str, Any]:
    """Delete/revoke a WhatsApp message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to delete
        sender_jid: Optional sender JID for admin revoking others' messages in groups

    Returns:
        A dictionary containing success status, chat_jid, and message_id
    """
    return whatsapp_delete_message(chat_jid, message_id, sender_jid)


@mcp.tool(annotations=READ_ONLY)
def get_group_info(group_jid: str) -> dict[str, Any]:
    """Get information about a WhatsApp group.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")

    Returns:
        A dictionary containing success, group_jid, name, topic, participant_count, participants
    """
    return whatsapp_get_group_info(group_jid)


@advanced_tool(annotations=WRITE)
def mark_read(chat_jid: str, message_ids: list[str], sender_jid: str | None = None) -> dict[str, Any]:
    """Mark WhatsApp messages as read (sends blue ticks).

    Args:
        chat_jid: The JID of the chat containing the messages
        message_ids: List of message IDs to mark as read
        sender_jid: Optional sender JID (required for group chats)

    Returns:
        A dictionary containing success status, chat_jid, message_ids, and count
    """
    return whatsapp_mark_messages_read(chat_jid, message_ids, sender_jid)


# Phase 2: Group Management


@advanced_tool(
    annotations=DESTRUCTIVE,
    description=(
        "Manage WhatsApp groups with one composable tool. "
        "Use action='create', 'add_members', 'remove_members', 'promote_admin', 'demote_admin', 'leave', or 'update'. "
        "Prefer this over create_group/add_group_members/remove_group_members/promote_to_admin/demote_admin/leave_group/update_group."
    ),
)
def manage_group(
    action: str,
    group_jid: str | None = None,
    name: str | None = None,
    participants: list[str] | None = None,
    participant: str | None = None,
    topic: str | None = None,
) -> dict[str, Any]:
    """Create/update/administer groups through one action-based tool."""
    allowed = ("create", "add_members", "remove_members", "promote_admin", "demote_admin", "leave", "update")
    if action not in allowed:
        return _invalid_action(action, allowed, "manage_group")

    if action == "create":
        if not name or not participants:
            return {
                "success": False,
                "error": "name and participants are required for action='create'",
                "use_tool": "manage_group",
            }
        return whatsapp_create_group(name, participants)

    if not group_jid:
        return {"success": False, "error": "group_jid is required for this action", "use_tool": "manage_group"}
    if action == "add_members":
        if not participants:
            return {
                "success": False,
                "error": "participants is required for action='add_members'",
                "use_tool": "manage_group",
            }
        return whatsapp_add_group_members(group_jid, participants)
    if action == "remove_members":
        if not participants:
            return {
                "success": False,
                "error": "participants is required for action='remove_members'",
                "use_tool": "manage_group",
            }
        return whatsapp_remove_group_members(group_jid, participants)
    if action == "promote_admin":
        if not participant:
            return {
                "success": False,
                "error": "participant is required for action='promote_admin'",
                "use_tool": "manage_group",
            }
        return whatsapp_promote_to_admin(group_jid, participant)
    if action == "demote_admin":
        if not participant:
            return {
                "success": False,
                "error": "participant is required for action='demote_admin'",
                "use_tool": "manage_group",
            }
        return whatsapp_demote_admin(group_jid, participant)
    if action == "leave":
        return whatsapp_leave_group(group_jid)
    return whatsapp_update_group(group_jid, name, topic)


# Phase 3: Polls


@mcp.tool(annotations=WRITE)
def create_poll(chat_jid: str, question: str, options: list[str], multi_select: bool = False) -> dict[str, Any]:
    """Create and send a poll to a WhatsApp chat.

    Args:
        chat_jid: The JID of the chat to send the poll to
        question: The poll question
        options: List of poll options (2-12 options required)
        multi_select: If True, allows multiple selections; if False, single selection only (default: False)

    Returns:
        A dictionary containing success, message_id, timestamp, chat_jid, question, options
    """
    return whatsapp_create_poll(chat_jid, question, options, multi_select)


# Phase 4: History Sync


@mcp.tool(annotations=WRITE)
def request_history(
    chat_jid: str, oldest_msg_id: str, oldest_msg_timestamp: int, oldest_msg_from_me: bool = False, count: int = 50
) -> dict[str, Any]:
    """Request older messages for a chat (on-demand history sync).

    This requests WhatsApp to sync older messages for a specific chat.
    The messages will appear in the database after the sync completes.
    Note: Only works if the phone has older messages available.

    Args:
        chat_jid: The JID of the chat to request history for
        oldest_msg_id: The ID of the oldest message currently in the chat
        oldest_msg_timestamp: Unix timestamp in milliseconds of the oldest message
        oldest_msg_from_me: Whether the oldest message was sent by you (default: False)
        count: Number of messages to request (max 50, default: 50)

    Returns:
        A dictionary containing success status and message

    Hints:
        - Use `list_messages` first to get the oldest_msg_id and oldest_msg_timestamp
        - After requesting, wait a few seconds then use `list_messages` to see synced messages
        - Use when `list_messages` returns fewer messages than expected
    """
    return whatsapp_request_chat_history(chat_jid, oldest_msg_id, oldest_msg_timestamp, oldest_msg_from_me, count)


# Phase 5: Advanced Features


@advanced_tool(annotations=WRITE)
def set_presence(presence: str) -> dict[str, Any]:
    """Set your own presence status (available/unavailable).

    Args:
        presence: Either "available" or "unavailable"

    Returns:
        A dictionary containing success status and presence
    """
    return whatsapp_set_presence(presence)


@advanced_tool(annotations=WRITE)
def subscribe_presence(jid: str) -> dict[str, Any]:
    """Subscribe to presence updates for a contact.

    After subscribing, you'll receive presence events for this contact.
    Note: Presence events arrive asynchronously via event handlers.

    Args:
        jid: The JID of the contact to subscribe to (e.g., "123456789@s.whatsapp.net")

    Returns:
        A dictionary containing success status
    """
    return whatsapp_subscribe_presence(jid)


@mcp.tool(annotations=READ_ONLY)
def get_profile_picture(jid: str, preview: bool = False) -> dict[str, Any]:
    """Get the profile picture URL for a user or group.

    Args:
        jid: The JID of the user or group
        preview: If True, get thumbnail instead of full resolution (default: False)

    Returns:
        A dictionary with url, id, type, direct_path (or has_picture=False if none)
    """
    return whatsapp_get_profile_picture(jid, preview)


@advanced_tool(
    annotations=DESTRUCTIVE,
    description=(
        "Manage blocked WhatsApp users. Use action='list', 'block', or 'unblock'. "
        "Prefer this over get_blocklist/block_user/unblock_user."
    ),
)
def manage_blocklist(action: str, jid: str | None = None) -> dict[str, Any]:
    """List, block, or unblock users through one action-based tool."""
    allowed = ("list", "block", "unblock")
    if action not in allowed:
        return _invalid_action(action, allowed, "manage_blocklist")
    if action == "list":
        return whatsapp_get_blocklist()
    if not jid:
        return {"success": False, "error": "jid is required for block/unblock", "use_tool": "manage_blocklist"}
    return whatsapp_update_blocklist(jid, action)


@advanced_tool(
    annotations=DESTRUCTIVE,
    description=(
        "Manage WhatsApp newsletters/channels. Use action='follow', 'unfollow', or 'create'. "
        "Prefer this over follow_newsletter/unfollow_newsletter/create_newsletter."
    ),
)
def manage_newsletter(
    action: str,
    jid: str | None = None,
    name: str | None = None,
    description: str = "",
) -> dict[str, Any]:
    """Follow, unfollow, or create newsletters through one action-based tool."""
    allowed = ("follow", "unfollow", "create")
    if action not in allowed:
        return _invalid_action(action, allowed, "manage_newsletter")
    if action == "create":
        if not name:
            return {"success": False, "error": "name is required for action='create'", "use_tool": "manage_newsletter"}
        return whatsapp_create_newsletter(name, description)
    if not jid:
        return {"success": False, "error": "jid is required for follow/unfollow", "use_tool": "manage_newsletter"}
    if action == "follow":
        return whatsapp_follow_newsletter(jid)
    return whatsapp_unfollow_newsletter(jid)


if __name__ == "__main__":
    # Run with stdio transport for Claude Code CLI
    mcp.run(transport="stdio")
