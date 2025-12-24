"""WhatsApp MCP Server - stdio transport for Claude Code CLI"""
import os
from typing import List, Dict, Any, Optional
from mcp.server.fastmcp import FastMCP
from whatsapp import (
    search_contacts as whatsapp_search_contacts,
    list_messages as whatsapp_list_messages,
    list_chats as whatsapp_list_chats,
    get_chat as whatsapp_get_chat,
    get_direct_chat_by_contact as whatsapp_get_direct_chat_by_contact,
    get_contact_chats as whatsapp_get_contact_chats,
    get_last_interaction as whatsapp_get_last_interaction,
    get_message_context as whatsapp_get_message_context,
    send_message as whatsapp_send_message,
    send_file as whatsapp_send_file,
    send_audio_message as whatsapp_audio_voice_message,
    download_media as whatsapp_download_media,
    get_contact_by_jid as whatsapp_get_contact_by_jid,
    get_contact_by_phone as whatsapp_get_contact_by_phone,
    list_all_contacts as whatsapp_list_all_contacts,
    format_contact_info as whatsapp_format_contact_info,
    set_contact_nickname as whatsapp_set_contact_nickname,
    get_contact_nickname as whatsapp_get_contact_nickname,
    remove_contact_nickname as whatsapp_remove_contact_nickname,
    list_contact_nicknames as whatsapp_list_contact_nicknames,
    # Phase 1 features
    send_reaction as whatsapp_send_reaction,
    edit_message as whatsapp_edit_message,
    delete_message as whatsapp_delete_message,
    get_group_info as whatsapp_get_group_info,
    mark_messages_read as whatsapp_mark_messages_read
)

# Initialize FastMCP server
mcp = FastMCP("whatsapp-extended")

@mcp.tool()
def search_contacts(query: str) -> str:
    """Search WhatsApp contacts by name or phone number.

    Args:
        query: Search term to match against contact names or phone numbers
    """
    contacts = whatsapp_search_contacts(query)
    if not contacts:
        return "No contacts found matching your query."

    result = f"Found {len(contacts)} contact(s):\n\n"
    for contact in contacts:
        result += whatsapp_format_contact_info(contact) + "\n"
    return result

@mcp.tool()
def list_messages(
    after: Optional[str] = None,
    before: Optional[str] = None,
    chat_jid: Optional[str] = None,
    query: Optional[str] = None,
    limit: int = 20,
    page: int = 0
) -> List[Dict[str, Any]]:
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
    query: Optional[str] = None,
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active"
) -> List[Dict[str, Any]]:
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
def get_chat(chat_jid: str, include_last_message: bool = True) -> Dict[str, Any]:
    """Get WhatsApp chat metadata by JID.

    Args:
        chat_jid: The JID of the chat to retrieve
        include_last_message: Whether to include the last message (default True)
    """
    chat = whatsapp_get_chat(chat_jid, include_last_message)
    return chat

@mcp.tool()
def get_direct_chat_by_contact(sender_phone_number: str) -> Dict[str, Any]:
    """Get WhatsApp chat metadata by sender phone number.

    Args:
        sender_phone_number: The phone number to search for
    """
    chat = whatsapp_get_direct_chat_by_contact(sender_phone_number)
    return chat

@mcp.tool()
def get_contact_chats(jid: str, limit: int = 20, page: int = 0) -> List[Dict[str, Any]]:
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
) -> Dict[str, Any]:
    """Get context around a specific WhatsApp message.

    Args:
        message_id: The ID of the message to get context for
        before: Number of messages to include before the target message (default 5)
        after: Number of messages to include after the target message (default 5)
    """
    context = whatsapp_get_message_context(message_id, before, after)
    return context

@mcp.tool()
def send_message(recipient: str, message: str) -> Dict[str, Any]:
    """Send a WhatsApp message to a person or group.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        message: The message text to send

    Returns:
        A dictionary containing success status and a status message
    """
    if not recipient:
        return {"success": False, "message": "Recipient must be provided"}

    success, status_message = whatsapp_send_message(recipient, message)
    return {"success": success, "message": status_message}

@mcp.tool()
def send_file(recipient: str, media_path: str) -> Dict[str, Any]:
    """Send a file such as a picture, raw audio, video or document via WhatsApp.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        media_path: The absolute path to the media file to send (image, video, document)

    Returns:
        A dictionary containing success status and a status message
    """
    success, status_message = whatsapp_send_file(recipient, media_path)
    return {"success": success, "message": status_message}

@mcp.tool()
def send_audio_message(recipient: str, media_path: str) -> Dict[str, Any]:
    """Send any audio file as a WhatsApp voice message.

    Args:
        recipient: The recipient - either a phone number with country code but no + or other symbols,
                 or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
        media_path: The absolute path to the audio file to send (will be converted to Opus .ogg if needed)

    Returns:
        A dictionary containing success status and a status message
    """
    success, status_message = whatsapp_audio_voice_message(recipient, media_path)
    return {"success": success, "message": status_message}

@mcp.tool()
def download_media(message_id: str, chat_jid: str) -> Dict[str, Any]:
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
def get_contact_details(identifier: str) -> str:
    """Get detailed information about a WhatsApp contact.

    Args:
        identifier: Either a JID or phone number of the contact
    """
    contact = whatsapp_get_contact_by_jid(identifier)
    if not contact:
        contact = whatsapp_get_contact_by_phone(identifier)

    if contact:
        return whatsapp_format_contact_info(contact)
    return f"No contact found for: {identifier}"

@mcp.tool()
def list_all_contacts(limit: int = 100) -> str:
    """List all WhatsApp contacts with their information.

    Args:
        limit: Maximum number of contacts to return (default 100)
    """
    contacts = whatsapp_list_all_contacts(limit)
    if not contacts:
        return "No contacts found."

    result = f"Found {len(contacts)} contact(s):\n\n"
    for contact in contacts:
        result += whatsapp_format_contact_info(contact) + "\n"
    return result

@mcp.tool()
def set_nickname(jid: str, nickname: str) -> Dict[str, Any]:
    """Set a custom nickname for a WhatsApp contact.

    Args:
        jid: The JID of the contact
        nickname: The custom nickname to set
    """
    success, message = whatsapp_set_contact_nickname(jid, nickname)
    return {"success": success, "message": message}

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
def remove_nickname(jid: str) -> Dict[str, Any]:
    """Remove the custom nickname for a WhatsApp contact.

    Args:
        jid: The JID of the contact
    """
    success, message = whatsapp_remove_contact_nickname(jid)
    return {"success": success, "message": message}

@mcp.tool()
def list_nicknames() -> str:
    """List all custom contact nicknames."""
    nicknames = whatsapp_list_contact_nicknames()
    if not nicknames:
        return "No custom nicknames set."

    result = f"Found {len(nicknames)} nickname(s):\n\n"
    for jid, nickname in nicknames:
        result += f"  {nickname} → {jid}\n"
    return result


# Phase 1 Features: Reactions, Edit, Delete, Group Info, Mark Read

@mcp.tool()
def send_reaction(chat_jid: str, message_id: str, emoji: str) -> Dict[str, Any]:
    """Send an emoji reaction to a WhatsApp message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to react to
        emoji: The emoji to react with (empty string to remove reaction)

    Returns:
        A dictionary containing success status and a status message
    """
    success, message = whatsapp_send_reaction(chat_jid, message_id, emoji)
    return {"success": success, "message": message}


@mcp.tool()
def edit_message(chat_jid: str, message_id: str, new_content: str) -> Dict[str, Any]:
    """Edit a previously sent WhatsApp message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to edit
        new_content: The new message content

    Returns:
        A dictionary containing success status and a status message
    """
    success, message = whatsapp_edit_message(chat_jid, message_id, new_content)
    return {"success": success, "message": message}


@mcp.tool()
def delete_message(chat_jid: str, message_id: str, sender_jid: Optional[str] = None) -> Dict[str, Any]:
    """Delete/revoke a WhatsApp message.

    Args:
        chat_jid: The JID of the chat containing the message
        message_id: The ID of the message to delete
        sender_jid: Optional sender JID for admin revoking others' messages in groups

    Returns:
        A dictionary containing success status and a status message
    """
    success, message = whatsapp_delete_message(chat_jid, message_id, sender_jid)
    return {"success": success, "message": message}


@mcp.tool()
def get_group_info(group_jid: str) -> Dict[str, Any]:
    """Get information about a WhatsApp group.

    Args:
        group_jid: The JID of the group (e.g., "123456789@g.us")

    Returns:
        A dictionary containing group info (name, topic, participants, etc.)
    """
    info = whatsapp_get_group_info(group_jid)
    if info:
        return {"success": True, "data": info}
    return {"success": False, "message": "Failed to get group info"}


@mcp.tool()
def mark_read(chat_jid: str, message_ids: List[str], sender_jid: Optional[str] = None) -> Dict[str, Any]:
    """Mark WhatsApp messages as read (sends blue ticks).

    Args:
        chat_jid: The JID of the chat containing the messages
        message_ids: List of message IDs to mark as read
        sender_jid: Optional sender JID (required for group chats)

    Returns:
        A dictionary containing success status and a status message
    """
    success, message = whatsapp_mark_messages_read(chat_jid, message_ids, sender_jid)
    return {"success": success, "message": message}


if __name__ == "__main__":
    # Run with stdio transport for Claude Code CLI
    mcp.run(transport='stdio')
