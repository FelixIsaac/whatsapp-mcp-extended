"""WhatsApp MCP Server - stdio transport for Claude Code CLI"""
from typing import Any

from mcp.server.fastmcp import FastMCP

from whatsapp import delete_message as whatsapp_delete_message
from whatsapp import download_media as whatsapp_download_media
from whatsapp import edit_message as whatsapp_edit_message
from whatsapp import get_chat as whatsapp_get_chat
from whatsapp import get_contact_by_jid as whatsapp_get_contact_by_jid
from whatsapp import get_contact_by_phone as whatsapp_get_contact_by_phone
from whatsapp import get_contact_chats as whatsapp_get_contact_chats
from whatsapp import get_contact_nickname as whatsapp_get_contact_nickname
from whatsapp import get_direct_chat_by_contact as whatsapp_get_direct_chat_by_contact
from whatsapp import get_group_info as whatsapp_get_group_info
from whatsapp import get_last_interaction as whatsapp_get_last_interaction
from whatsapp import get_message_context as whatsapp_get_message_context
from whatsapp import list_all_contacts as whatsapp_list_all_contacts
from whatsapp import list_chats as whatsapp_list_chats
from whatsapp import list_contact_nicknames as whatsapp_list_contact_nicknames
from whatsapp import list_messages as whatsapp_list_messages
from whatsapp import mark_messages_read as whatsapp_mark_messages_read
from whatsapp import remove_contact_nickname as whatsapp_remove_contact_nickname
from whatsapp import search_contacts as whatsapp_search_contacts
from whatsapp import send_audio_message as whatsapp_audio_voice_message
from whatsapp import send_file as whatsapp_send_file
from whatsapp import send_message as whatsapp_send_message
from whatsapp import send_reaction as whatsapp_send_reaction  # Phase 1 features
from whatsapp import set_contact_nickname as whatsapp_set_contact_nickname

# Phase 2: Group Management
from whatsapp import add_group_members as whatsapp_add_group_members
from whatsapp import create_group as whatsapp_create_group
from whatsapp import demote_admin as whatsapp_demote_admin
from whatsapp import leave_group as whatsapp_leave_group
from whatsapp import promote_to_admin as whatsapp_promote_to_admin
from whatsapp import remove_group_members as whatsapp_remove_group_members
from whatsapp import update_group as whatsapp_update_group

# Initialize FastMCP server
mcp = FastMCP("whatsapp-extended")

@mcp.tool()
def search_contacts(query: str) -> list[dict[str, Any]]:
    """Search WhatsApp contacts by name or phone number.

    Args:
        query: Search term to match against contact names or phone numbers

    Returns:
        List of contact dicts with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname
    """
    return whatsapp_search_contacts(query)

@mcp.tool()
def list_messages(
    after: str | None = None,
    before: str | None = None,
    chat_jid: str | None = None,
    query: str | None = None,
    limit: int = 20,
    page: int = 0
) -> list[dict[str, Any]]:
    """Get WhatsApp messages matching specified criteria.

    Args:
        after: Optional ISO-8601 formatted string to only return messages after this date
        before: Optional ISO-8601 formatted string to only return messages before this date
        chat_jid: Optional chat JID to filter messages by chat
        query: Optional search term to filter messages by content
        limit: Maximum number of messages to return (default 20)
        page: Page number for pagination (default 0)
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
        context_after=1
    )
    return messages

@mcp.tool()
def list_chats(
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active"
) -> list[dict[str, Any]]:
    """Get WhatsApp chats matching specified criteria.

    Args:
        query: Optional search term to filter chats by name or JID
        limit: Maximum number of chats to return (default 20)
        page: Page number for pagination (default 0)
        include_last_message: Whether to include the last message in each chat (default True)
        sort_by: Field to sort results by, either "last_active" or "name" (default "last_active")
    """
    chats = whatsapp_list_chats(
        query=query,
        limit=limit,
        page=page,
        include_last_message=include_last_message,
        sort_by=sort_by
    )
    return chats

@mcp.tool()
def get_chat(chat_jid: str, include_last_message: bool = True) -> dict[str, Any]:
    """Get WhatsApp chat metadata by JID.

    Args:
        chat_jid: The JID of the chat to retrieve
        include_last_message: Whether to include the last message (default True)
    """
    chat = whatsapp_get_chat(chat_jid, include_last_message)
    return chat

@mcp.tool()
def get_direct_chat_by_contact(sender_phone_number: str) -> dict[str, Any]:
    """Get WhatsApp chat metadata by sender phone number.

    Args:
        sender_phone_number: The phone number to search for
    """
    chat = whatsapp_get_direct_chat_by_contact(sender_phone_number)
    return chat

@mcp.tool()
def get_contact_chats(jid: str, limit: int = 20, page: int = 0) -> list[dict[str, Any]]:
    """Get all WhatsApp chats involving the contact.

    Args:
        jid: The contact's JID to search for
        limit: Maximum number of chats to return (default 20)
        page: Page number for pagination (default 0)
    """
    chats = whatsapp_get_contact_chats(jid, limit, page)
    return chats

@mcp.tool()
def get_last_interaction(jid: str) -> str:
    """Get most recent WhatsApp message involving the contact.

    Args:
        jid: The JID of the contact to search for
    """
    message = whatsapp_get_last_interaction(jid)
    return message

@mcp.tool()
def get_message_context(
    message_id: str,
    before: int = 5,
    after: int = 5
) -> dict[str, Any]:
    """Get context around a specific WhatsApp message.

    Args:
        message_id: The ID of the message to get context for
        before: Number of messages to include before the target message (default 5)
        after: Number of messages to include after the target message (default 5)
    """
    context = whatsapp_get_message_context(message_id, before, after)
    return context

@mcp.tool()
def send_message(recipient: str, message: str) -> dict[str, Any]:
    """Send a WhatsApp message to a person or group. For group chats use the JID.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        message: The message text to send

    Returns:
        A dictionary containing success status and a status message
    """
    return whatsapp_send_message(recipient, message)

@mcp.tool()
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

@mcp.tool()
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

@mcp.tool()
def download_media(message_id: str, chat_jid: str) -> dict[str, Any]:
    """Download media from a WhatsApp message and get the local file path.

    Args:
        message_id: The ID of the message containing the media
        chat_jid: The JID of the chat containing the message

    Returns:
        A dictionary containing success status, a status message, and the file path if successful
    """
    file_path = whatsapp_download_media(message_id, chat_jid)

    if file_path:
        return {"success": True, "message": "Media downloaded successfully", "file_path": file_path}
    else:
        return {"success": False, "message": "Failed to download media"}

@mcp.tool()
def get_contact_details(identifier: str) -> dict[str, Any] | None:
    """Get detailed information about a WhatsApp contact.

    Args:
        identifier: Either a JID or phone number of the contact

    Returns:
        Contact dict with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname
        or None if not found
    """
    contact = whatsapp_get_contact_by_jid(identifier)
    if not contact:
        contact = whatsapp_get_contact_by_phone(identifier)
    return contact

@mcp.tool()
def list_all_contacts(limit: int = 100) -> list[dict[str, Any]]:
    """List all WhatsApp contacts with their information.

    Args:
        limit: Maximum number of contacts to return (default 100)

    Returns:
        List of contact dicts with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname
    """
    return whatsapp_list_all_contacts(limit)

@mcp.tool()
def set_nickname(jid: str, nickname: str) -> dict[str, Any]:
    """Set a custom nickname for a WhatsApp contact.

    Args:
        jid: The JID of the contact
        nickname: The custom nickname to set

    Returns:
        A dictionary containing success, jid, nickname, and updated_at
    """
    return whatsapp_set_contact_nickname(jid, nickname)

@mcp.tool()
def get_nickname(jid: str) -> str:
    """Get the custom nickname for a WhatsApp contact.

    Args:
        jid: The JID of the contact
    """
    nickname = whatsapp_get_contact_nickname(jid)
    if nickname:
        return f"Nickname for {jid}: {nickname}"
    return f"No nickname set for {jid}"

@mcp.tool()
def remove_nickname(jid: str) -> dict[str, Any]:
    """Remove the custom nickname for a WhatsApp contact.

    Args:
        jid: The JID of the contact

    Returns:
        A dictionary containing success and jid
    """
    return whatsapp_remove_contact_nickname(jid)

@mcp.tool()
def list_nicknames() -> list[dict[str, Any]]:
    """List all custom contact nicknames.

    Returns:
        List of dicts with jid, nickname, created_at, updated_at
    """
    return whatsapp_list_contact_nicknames()


# Phase 1 Features: Reactions, Edit, Delete, Group Info, Mark Read

@mcp.tool()
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


@mcp.tool()
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


@mcp.tool()
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


@mcp.tool()
def get_group_info(group_jid: str) -> dict[str, Any]:
    """Get information about a WhatsApp group.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")

    Returns:
        A dictionary containing success, group_jid, name, topic, participant_count, participants
    """
    return whatsapp_get_group_info(group_jid)


@mcp.tool()
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

@mcp.tool()
def create_group(name: str, participants: list[str]) -> dict[str, Any]:
    """Create a new WhatsApp group.

    Args:
        name: The name for the new group
        participants: List of participant JIDs to add (e.g., ["123456789@s.whatsapp.net"])

    Returns:
        A dictionary containing success, group_jid, name, and participants
    """
    return whatsapp_create_group(name, participants)


@mcp.tool()
def add_group_members(group_jid: str, participants: list[str]) -> dict[str, Any]:
    """Add members to a WhatsApp group.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")
        participants: List of participant JIDs to add

    Returns:
        A dictionary containing success, group_jid, added count
    """
    return whatsapp_add_group_members(group_jid, participants)


@mcp.tool()
def remove_group_members(group_jid: str, participants: list[str]) -> dict[str, Any]:
    """Remove members from a WhatsApp group.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")
        participants: List of participant JIDs to remove

    Returns:
        A dictionary containing success, group_jid, removed count
    """
    return whatsapp_remove_group_members(group_jid, participants)


@mcp.tool()
def promote_to_admin(group_jid: str, participant: str) -> dict[str, Any]:
    """Promote a group member to admin.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")
        participant: The JID of the participant to promote

    Returns:
        A dictionary containing success, group_jid, participant
    """
    return whatsapp_promote_to_admin(group_jid, participant)


@mcp.tool()
def demote_admin(group_jid: str, participant: str) -> dict[str, Any]:
    """Demote a group admin to regular member.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")
        participant: The JID of the admin to demote

    Returns:
        A dictionary containing success, group_jid, participant
    """
    return whatsapp_demote_admin(group_jid, participant)


@mcp.tool()
def leave_group(group_jid: str) -> dict[str, Any]:
    """Leave a WhatsApp group.

    Args:
        group_jid: The JID of the group to leave (e.g., "123456789@g.us")

    Returns:
        A dictionary containing success, group_jid
    """
    return whatsapp_leave_group(group_jid)


@mcp.tool()
def update_group(group_jid: str, name: str | None = None, topic: str | None = None) -> dict[str, Any]:
    """Update group name and/or topic (description).

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")
        name: New group name (optional)
        topic: New group topic/description (optional)

    Returns:
        A dictionary containing success, group_jid, updated fields
    """
    return whatsapp_update_group(group_jid, name, topic)


if __name__ == "__main__":
    # Run with stdio transport for Claude Code CLI
    mcp.run(transport='stdio')
