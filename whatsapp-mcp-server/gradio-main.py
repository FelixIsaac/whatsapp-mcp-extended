import logging
import os

import gradio as gr
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

# Configure logging
logging.basicConfig(
    level=logging.DEBUG if os.environ.get('DEBUG') == 'true' else logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

# Initialize FastMCP server
mcp = FastMCP(
    "whatsapp",
    log_level="DEBUG" if os.environ.get('DEBUG') == 'true' else "INFO",
    
    
)

# Define MCP tools (these will be exposed through both MCP and Gradio)

@mcp.tool()
def search_contacts(query: str) -> str:
    """Search WhatsApp contacts by name or phone number.

    Parameters:
    - query: Search term to match against contact names or phone numbers

    Returns:
        JSON list of contact dicts with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname
    """
    return str(whatsapp_search_contacts(query))

@mcp.tool()
def list_messages(
    after: str = "",
    before: str = "",
    sender_phone_number: str = "",
    chat_jid: str = "",
    query: str = "",
    limit: int = 20,
    page: int = 0,
    include_context: bool = True,
    context_before: int = 1,
    context_after: int = 1
) -> str:
    """Get WhatsApp messages matching specified criteria with optional context.
    
    Parameters:
    - after: ISO-8601 formatted date string to only return messages after this date (optional, leave empty if not needed)
    - before: ISO-8601 formatted date string to only return messages before this date (optional, leave empty if not needed)
    - sender_phone_number: Phone number to filter messages by sender (optional, leave empty if not needed)
    - chat_jid: Chat JID to filter messages by chat (optional, leave empty if not needed)
    - query: Search term to filter messages by content (optional, leave empty if not needed)
    - limit: Maximum number of messages to return (default: 20)
    - page: Page number for pagination (default: 0)
    - include_context: Whether to include messages before and after matches (default: true)
    - context_before: Number of messages to include before each match (default: 1)
    - context_after: Number of messages to include after each match (default: 1)
    """
    # Convert empty strings to None for internal processing
    after_param = after if after else None
    before_param = before if before else None
    sender_param = sender_phone_number if sender_phone_number else None
    chat_param = chat_jid if chat_jid else None
    query_param = query if query else None
    
    messages = whatsapp_list_messages(
        after=after_param,
        before=before_param,
        sender_phone_number=sender_param,
        chat_jid=chat_param,
        query=query_param,
        limit=limit,
        page=page,
        include_context=include_context,
        context_before=context_before,
        context_after=context_after
    )
    return str(messages)

@mcp.tool()
def list_chats(
    query: str = "",
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active"
) -> str:
    """Get WhatsApp chats matching specified criteria.
    
    Parameters:
    - query: Search term to filter chats by name or JID (optional, leave empty if not needed)
    - limit: Maximum number of chats to return (default: 20)
    - page: Page number for pagination (default: 0)
    - include_last_message: Whether to include the last message in each chat (default: true)
    - sort_by: Field to sort results by, either "last_active" or "name" (default: "last_active")
    """
    # Convert empty string to None for internal processing
    query_param = query if query else None
    
    chats = whatsapp_list_chats(
        query=query_param,
        limit=limit,
        page=page,
        include_last_message=include_last_message,
        sort_by=sort_by
    )
    return str(chats)

@mcp.tool()
def get_chat(chat_jid: str, include_last_message: bool = True) -> str:
    """Get WhatsApp chat metadata by JID.
    
    Parameters:
    - chat_jid: The JID of the chat to retrieve
    - include_last_message: Whether to include the last message (default: true)
    """
    chat = whatsapp_get_chat(chat_jid, include_last_message)
    return str(chat)

@mcp.tool()
def get_direct_chat_by_contact(sender_phone_number: str) -> str:
    """Get WhatsApp chat metadata by sender phone number.
    
    Parameters:
    - sender_phone_number: The phone number to search for
    """
    chat = whatsapp_get_direct_chat_by_contact(sender_phone_number)
    return str(chat)

@mcp.tool()
def get_contact_chats(jid: str, limit: int = 20, page: int = 0) -> str:
    """Get all WhatsApp chats involving the contact.
    
    Parameters:
    - jid: The contact's JID to search for
    - limit: Maximum number of chats to return (default: 20)
    - page: Page number for pagination (default: 0)
    """
    chats = whatsapp_get_contact_chats(jid, limit, page)
    return str(chats)

@mcp.tool()
def get_last_interaction(jid: str) -> str:
    """Get most recent WhatsApp message involving the contact.
    
    Parameters:
    - jid: The JID of the contact to search for
    """
    message = whatsapp_get_last_interaction(jid)
    return message

@mcp.tool()
def get_message_context(
    message_id: str,
    before: int = 5,
    after: int = 5
) -> str:
    """Get context around a specific WhatsApp message.
    
    Parameters:
    - message_id: The ID of the message to get context for
    - before: Number of messages to include before the target message (default: 5)
    - after: Number of messages to include after the target message (default: 5)
    """
    context = whatsapp_get_message_context(message_id, before, after)
    return str(context)

@mcp.tool()
def send_message(recipient: str, message: str) -> str:
    """Send a WhatsApp message to a person or group. For group chats use the JID.

    Parameters:
    - recipient: The recipient - either a phone number with country code but no + or other symbols, or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
    - message: The message text to send
    """
    result = whatsapp_send_message(recipient, message)
    return str(result)

@mcp.tool()
def send_file(recipient: str, media_path: str) -> str:
    """Send a file such as a picture, raw audio, video or document via WhatsApp to the specified recipient. For group messages use the JID.

    Parameters:
    - recipient: The recipient - either a phone number with country code but no + or other symbols, or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
    - media_path: The absolute path to the media file to send (image, video, document)
    """
    result = whatsapp_send_file(recipient, media_path)
    return str(result)

@mcp.tool()
def send_audio_message(recipient: str, media_path: str) -> str:
    """Send any audio file as a WhatsApp audio message to the specified recipient. For group messages use the JID. If it errors due to ffmpeg not being installed, use send_file instead.

    Parameters:
    - recipient: The recipient - either a phone number with country code but no + or other symbols, or a JID (e.g., "123456789@s.whatsapp.net" or a group JID like "123456789@g.us")
    - media_path: The absolute path to the audio file to send (will be converted to Opus .ogg if it's not a .ogg file)
    """
    result = whatsapp_audio_voice_message(recipient, media_path)
    return str(result)

@mcp.tool()
def download_media(message_id: str, chat_jid: str) -> str:
    """Download media from a WhatsApp message and get the local file path.
    
    Parameters:
    - message_id: The ID of the message containing the media
    - chat_jid: The JID of the chat containing the message
    """
    file_path = whatsapp_download_media(message_id, chat_jid)
    
    if file_path:
        result = {
            "success": True,
            "message": "Media downloaded successfully",
            "file_path": file_path
        }
    else:
        result = {
            "success": False,
            "message": "Failed to download media"
        }
    return str(result)

@mcp.tool()
def get_contact_details(identifier: str) -> str:
    """Get detailed contact information.

    Parameters:
    - identifier: Either a JID or phone number of the contact

    Returns:
        Contact dict with jid, phone_number, name, first_name, full_name, push_name, business_name, nickname
    """
    contact = whatsapp_get_contact_by_jid(identifier)
    if not contact:
        contact = whatsapp_get_contact_by_phone(identifier)
    return str(contact)


@mcp.tool()
def list_all_contacts(limit: int = 100) -> str:
    """Get all contacts with their detailed information.

    Parameters:
    - limit: Maximum number of contacts to return

    Returns:
        JSON list of contact dicts
    """
    return str(whatsapp_list_all_contacts(limit))


@mcp.tool()
def set_contact_nickname(jid: str, nickname: str) -> str:
    """Set a custom nickname for a contact.

    Parameters:
    - jid: WhatsApp JID of the contact
    - nickname: Custom nickname to set for the contact
    """
    return str(whatsapp_set_contact_nickname(jid, nickname))


@mcp.tool()
def get_contact_nickname(jid: str) -> str:
    """Get a contact's custom nickname.
    
    Parameters:
    - jid: WhatsApp JID of the contact
    """
    nickname = whatsapp_get_contact_nickname(jid)
    result = {"jid": jid, "nickname": nickname}
    return str(result)


@mcp.tool()
def remove_contact_nickname(jid: str) -> str:
    """Remove a contact's custom nickname.

    Parameters:
    - jid: WhatsApp JID of the contact
    """
    return str(whatsapp_remove_contact_nickname(jid))


@mcp.tool()
def list_contact_nicknames() -> str:
    """List all custom contact nicknames with timestamps.

    Parameters:
    None required
    """
    return str(whatsapp_list_contact_nicknames())


# Phase 1 Features: Reactions, Edit, Delete, Group Info, Mark Read

@mcp.tool()
def send_reaction(chat_jid: str, message_id: str, emoji: str) -> str:
    """Send an emoji reaction to a WhatsApp message.

    Parameters:
    - chat_jid: The JID of the chat containing the message
    - message_id: The ID of the message to react to
    - emoji: The emoji to react with (empty string to remove reaction)
    """
    return str(whatsapp_send_reaction(chat_jid, message_id, emoji))


@mcp.tool()
def edit_message(chat_jid: str, message_id: str, new_content: str) -> str:
    """Edit a previously sent WhatsApp message.

    Parameters:
    - chat_jid: The JID of the chat containing the message
    - message_id: The ID of the message to edit
    - new_content: The new message content
    """
    return str(whatsapp_edit_message(chat_jid, message_id, new_content))


@mcp.tool()
def delete_message(chat_jid: str, message_id: str, sender_jid: str = "") -> str:
    """Delete/revoke a WhatsApp message.

    Parameters:
    - chat_jid: The JID of the chat containing the message
    - message_id: The ID of the message to delete
    - sender_jid: Optional sender JID for admin revoking others' messages in groups
    """
    sender = sender_jid if sender_jid else None
    return str(whatsapp_delete_message(chat_jid, message_id, sender))


@mcp.tool()
def get_group_info(group_jid: str) -> str:
    """Get information about a WhatsApp group.

    Parameters:
    - group_jid: The JID of the group (e.g., "123456789@g.us")
    """
    return str(whatsapp_get_group_info(group_jid))


@mcp.tool()
def mark_read(chat_jid: str, message_ids: str, sender_jid: str = "") -> str:
    """Mark WhatsApp messages as read (sends blue ticks).

    Parameters:
    - chat_jid: The JID of the chat containing the messages
    - message_ids: Comma-separated list of message IDs to mark as read
    - sender_jid: Optional sender JID (required for group chats)
    """
    ids = [mid.strip() for mid in message_ids.split(",") if mid.strip()]
    sender = sender_jid if sender_jid else None
    return str(whatsapp_mark_messages_read(chat_jid, ids, sender))


# Phase 2: Group Management

@mcp.tool()
def create_group(name: str, participants: str) -> str:
    """Create a new WhatsApp group.

    Parameters:
    - name: The name for the new group
    - participants: Comma-separated list of participant JIDs (e.g., "123@s.whatsapp.net,456@s.whatsapp.net")
    """
    participant_list = [p.strip() for p in participants.split(",") if p.strip()]
    return str(whatsapp_create_group(name, participant_list))


@mcp.tool()
def add_group_members(group_jid: str, participants: str) -> str:
    """Add members to a WhatsApp group.

    Parameters:
    - group_jid: The JID of the group (e.g., "123456789@g.us")
    - participants: Comma-separated list of participant JIDs to add
    """
    participant_list = [p.strip() for p in participants.split(",") if p.strip()]
    return str(whatsapp_add_group_members(group_jid, participant_list))


@mcp.tool()
def remove_group_members(group_jid: str, participants: str) -> str:
    """Remove members from a WhatsApp group.

    Parameters:
    - group_jid: The JID of the group (e.g., "123456789@g.us")
    - participants: Comma-separated list of participant JIDs to remove
    """
    participant_list = [p.strip() for p in participants.split(",") if p.strip()]
    return str(whatsapp_remove_group_members(group_jid, participant_list))


@mcp.tool()
def promote_to_admin(group_jid: str, participant: str) -> str:
    """Promote a group member to admin.

    Parameters:
    - group_jid: The JID of the group (e.g., "123456789@g.us")
    - participant: The JID of the participant to promote
    """
    return str(whatsapp_promote_to_admin(group_jid, participant))


@mcp.tool()
def demote_admin(group_jid: str, participant: str) -> str:
    """Demote a group admin to regular member.

    Parameters:
    - group_jid: The JID of the group (e.g., "123456789@g.us")
    - participant: The JID of the admin to demote
    """
    return str(whatsapp_demote_admin(group_jid, participant))


@mcp.tool()
def leave_group(group_jid: str) -> str:
    """Leave a WhatsApp group.

    Parameters:
    - group_jid: The JID of the group to leave (e.g., "123456789@g.us")
    """
    return str(whatsapp_leave_group(group_jid))


@mcp.tool()
def update_group(group_jid: str, name: str = "", topic: str = "") -> str:
    """Update group name and/or topic (description).

    Parameters:
    - group_jid: The JID of the group (e.g., "123456789@g.us")
    - name: New group name (optional, leave empty to not change)
    - topic: New group topic/description (optional, leave empty to not change)
    """
    name_param = name if name else None
    topic_param = topic if topic else None
    return str(whatsapp_update_group(group_jid, name_param, topic_param))


# Gradio UI functions (these wrap the MCP tools for use with the Gradio UI)

def gradio_search_contacts(query):
    contacts = search_contacts(query)
    if contacts:
        return gr.update(value=str(contacts), visible=True)
    else:
        return gr.update(value="No contacts found", visible=True)

def gradio_list_chats(query, limit, include_last_message, sort_by):
    chats = list_chats(
        query=query if query else None, 
        limit=int(limit), 
        page=0,
        include_last_message=include_last_message, 
        sort_by=sort_by
    )
    if chats:
        return gr.update(value=str(chats), visible=True)
    else:
        return gr.update(value="No chats found", visible=True)

def gradio_list_messages(chat_jid, query, limit):
    messages = list_messages(
        chat_jid=chat_jid if chat_jid else None,
        query=query if query else None,
        limit=int(limit),
        page=0
    )
    if messages:
        return gr.update(value=str(messages), visible=True)
    else:
        return gr.update(value="No messages found", visible=True)

def gradio_send_message(recipient, message):
    result = send_message(recipient, message)
    return f"Status: {result['success']}, Message: {result['message']}"

def gradio_send_file(recipient, file):
    result = send_file(recipient, file.name)
    return f"Status: {result['success']}, Message: {result['message']}"

def gradio_send_audio(recipient, file):
    result = send_audio_message(recipient, file.name)
    return f"Status: {result['success']}, Message: {result['message']}"

# Gradio wrapper functions for contact management

def gradio_get_contact_details(jid, phone_number):
    """Gradio wrapper for get_contact_details"""
    if not jid and not phone_number:
        return "Error: Either JID or phone number must be provided"
    
    result = get_contact_details(jid=jid if jid else None, phone_number=phone_number if phone_number else None)
    
    if "error" in result:
        return result["error"]
    else:
        return result["formatted_info"]

def gradio_list_all_contacts(limit):
    """Gradio wrapper for list_all_contacts"""
    contacts = list_all_contacts(limit=int(limit))
    
    if contacts:
        formatted_contacts = []
        for contact in contacts:
            name_to_display = contact.get('name', '') if contact.get('name', '') != '*' else contact.get('push_name', 'Unknown')
            formatted_contacts.append(
                f"📱 {name_to_display} ({contact.get('phone_number', 'N/A')})\n"
                f"   JID: {contact.get('jid', 'N/A')}\n"
                f"   Full Name: {contact.get('full_name') or 'N/A'}\n"
                f"   Push Name: {contact.get('push_name') or 'N/A'}\n"
                f"   Nickname: {contact.get('nickname') or 'N/A'}\n"
                f"   Business: {contact.get('business_name') or 'N/A'}\n"
            )
        return "\n".join(formatted_contacts)
    else:
        return "No contacts found"

def gradio_set_contact_nickname(jid, nickname):
    """Gradio wrapper for set_contact_nickname"""
    if not jid or not nickname:
        return "Error: Both JID and nickname must be provided"
    
    result = set_contact_nickname(jid, nickname)
    return f"Status: {result['success']}, Message: {result['message']}"

def gradio_get_contact_nickname(jid):
    """Gradio wrapper for get_contact_nickname"""
    if not jid:
        return "Error: JID must be provided"
    
    result = get_contact_nickname(jid)
    nickname = result.get('nickname')
    
    if nickname:
        return f"Nickname for {jid}: {nickname}"
    else:
        return f"No nickname set for {jid}"

def gradio_remove_contact_nickname(jid):
    """Gradio wrapper for remove_contact_nickname"""
    if not jid:
        return "Error: JID must be provided"
    
    result = remove_contact_nickname(jid)
    return f"Status: {result['success']}, Message: {result['message']}"

def gradio_list_contact_nicknames():
    """Gradio wrapper for list_contact_nicknames"""
    nicknames = list_contact_nicknames()
    
    if nicknames:
        formatted_nicknames = []
        for item in nicknames:
            formatted_nicknames.append(f"📝 {item['nickname']} -> {item['jid']}")
        return "\n".join(formatted_nicknames)
    else:
        return "No custom nicknames found"

# Create Gradio UI
def create_gradio_ui():
    with gr.Blocks(title="WhatsApp MCP Interface") as app:
        gr.Markdown("# WhatsApp MCP Interface")
        gr.Markdown("This interface allows you to interact with your WhatsApp account through the Model Context Protocol (MCP).")
        
        with gr.Tab("Search Contacts"):
            with gr.Row():
                search_query = gr.Textbox(label="Search Query", placeholder="Enter name or phone number")
                search_button = gr.Button("Search")
            
            search_results = gr.Textbox(label="Results", visible=False, lines=10)
            search_button.click(gradio_search_contacts, inputs=search_query, outputs=search_results)
        
        with gr.Tab("Contact Details"):
            gr.Markdown("### Get detailed contact information")
            with gr.Row():
                contact_jid = gr.Textbox(label="Contact JID (optional)", placeholder="e.g., 123456789@s.whatsapp.net")
                contact_phone = gr.Textbox(label="Phone Number (optional)", placeholder="e.g., 123456789")
            
            get_contact_button = gr.Button("Get Contact Details")
            contact_details_result = gr.Textbox(label="Contact Details", lines=10)
            
            get_contact_button.click(
                gradio_get_contact_details,
                inputs=[contact_jid, contact_phone],
                outputs=contact_details_result
            )
        
        with gr.Tab("All Contacts"):
            gr.Markdown("### List all contacts with detailed information")
            with gr.Row():
                contacts_limit = gr.Slider(label="Limit", minimum=10, maximum=500, value=100, step=10)
            
            list_contacts_button = gr.Button("List All Contacts")
            all_contacts_result = gr.Textbox(label="All Contacts", lines=15)
            
            list_contacts_button.click(
                gradio_list_all_contacts,
                inputs=[ contacts_limit],
                outputs=all_contacts_result
            )
        
        with gr.Tab("Contact Nicknames"):
            gr.Markdown("### Manage custom contact nicknames")
            
            with gr.Row():
                with gr.Column():
                    gr.Markdown("#### Set Nickname")
                    set_nickname_jid = gr.Textbox(label="Contact JID", placeholder="e.g., 123456789@s.whatsapp.net")
                    set_nickname_text = gr.Textbox(label="Nickname", placeholder="Enter custom nickname")
                    set_nickname_button = gr.Button("Set Nickname")
                    set_nickname_result = gr.Textbox(label="Result", lines=2)
                
                with gr.Column():
                    gr.Markdown("#### Get Nickname")
                    get_nickname_jid = gr.Textbox(label="Contact JID", placeholder="e.g., 123456789@s.whatsapp.net")
                    get_nickname_button = gr.Button("Get Nickname")
                    get_nickname_result = gr.Textbox(label="Result", lines=2)
            
            with gr.Row():
                with gr.Column():
                    gr.Markdown("#### Remove Nickname")
                    remove_nickname_jid = gr.Textbox(label="Contact JID", placeholder="e.g., 123456789@s.whatsapp.net")
                    remove_nickname_button = gr.Button("Remove Nickname")
                    remove_nickname_result = gr.Textbox(label="Result", lines=2)
                
                with gr.Column():
                    gr.Markdown("#### List All Nicknames")
                    list_nicknames_button = gr.Button("List All Nicknames")
                    list_nicknames_result = gr.Textbox(label="All Nicknames", lines=10)
            
            # Connect the nickname management buttons
            set_nickname_button.click(
                gradio_set_contact_nickname,
                inputs=[set_nickname_jid, set_nickname_text],
                outputs=set_nickname_result
            )
            
            get_nickname_button.click(
                gradio_get_contact_nickname,
                inputs=get_nickname_jid,
                outputs=get_nickname_result
            )
            
            remove_nickname_button.click(
                gradio_remove_contact_nickname,
                inputs=remove_nickname_jid,
                outputs=remove_nickname_result
            )
            
            list_nicknames_button.click(
                gradio_list_contact_nicknames,
                outputs=list_nicknames_result
            )
        
        with gr.Tab("List Chats"):
            with gr.Row():
                chat_query = gr.Textbox(label="Search Query (optional)", placeholder="Enter chat name")
                chat_limit = gr.Slider(label="Limit", minimum=1, maximum=50, value=20, step=1)
                chat_include_last = gr.Checkbox(label="Include Last Message", value=True)
                chat_sort = gr.Dropdown(label="Sort By", choices=["last_active", "name"], value="last_active")
                
            chat_search_button = gr.Button("List Chats")
            chat_results = gr.Textbox(label="Results", visible=False, lines=10)
            
            chat_search_button.click(
                gradio_list_chats, 
                inputs=[chat_query, chat_limit, chat_include_last, chat_sort], 
                outputs=chat_results
            )
        
        with gr.Tab("List Messages"):
            with gr.Row():
                msg_chat_jid = gr.Textbox(label="Chat JID (optional)", placeholder="Enter chat JID")
                msg_query = gr.Textbox(label="Search Query (optional)", placeholder="Enter message content to search")
                msg_limit = gr.Slider(label="Limit", minimum=1, maximum=50, value=20, step=1)
                
            msg_search_button = gr.Button("List Messages")
            msg_results = gr.Textbox(label="Results", visible=False, lines=10)
            
            msg_search_button.click(
                gradio_list_messages, 
                inputs=[msg_chat_jid, msg_query, msg_limit], 
                outputs=msg_results
            )
        
        with gr.Tab("Send Message"):
            with gr.Row():
                send_recipient = gr.Textbox(label="Recipient", placeholder="Phone number or JID")
                send_message_text = gr.Textbox(label="Message", placeholder="Type your message here", lines=3)
                
            send_button = gr.Button("Send Message")
            send_result = gr.Textbox(label="Result", lines=2)
            
            send_button.click(
                gradio_send_message, 
                inputs=[send_recipient, send_message_text], 
                outputs=send_result
            )
        
        with gr.Tab("Send Media"):
            with gr.Row():
                media_recipient = gr.Textbox(label="Recipient", placeholder="Phone number or JID")
                media_file = gr.File(label="Select Media File")
                
            send_file_button = gr.Button("Send File")
            send_audio_button = gr.Button("Send as Audio Message")
            media_result = gr.Textbox(label="Result", lines=2)
            
            send_file_button.click(
                gradio_send_file, 
                inputs=[media_recipient, media_file], 
                outputs=media_result
            )
            
            send_audio_button.click(
                gradio_send_audio, 
                inputs=[media_recipient, media_file], 
                outputs=media_result
            )
        
        with gr.Tab("get_last_interaction"):
            with gr.Row():
                interaction_chat_jid = gr.Textbox(label="Chat JID", placeholder="Enter chat JID")

            interaction_search_button = gr.Button("Get Last Interaction")
            interaction_results = gr.Textbox(label="Results", visible=False, lines=5)

            interaction_search_button.click(
                get_last_interaction,
                inputs=[interaction_chat_jid],
                outputs=interaction_results
            )
    return app

# Main function
if __name__ == "__main__":
    # Get configuration from environment variables or use defaults
    host = os.environ.get('HOST', '0.0.0.0')
    port = int(os.environ.get('PORT', '8081'))  # Use a different port to avoid conflicts with the Inspector
    gradio_port = int(os.environ.get('GRADIO_PORT', '8082'))
    # Check if Gradio should be enabled (default: True for backward compatibility)
    enable_gradio = os.environ.get('GRADIO', 'true').lower() in ('true', '1', 'yes', 'on')
    
    if enable_gradio:
        # Start MCP server in a separate thread
        import threading
        def start_mcp_server():
            logging.info(f"Starting WhatsApp MCP server with SSE transport on {host}:{port}")
            try:
                # Initialize and run the server with SSE transport
                mcp.run(
                    transport='sse'
                )
            except Exception as e:
                logging.error(f"Error starting MCP server: {e}")
                import traceback
                traceback.print_exc()
        
        # Start MCP server in a thread
        mcp_thread = threading.Thread(target=start_mcp_server)
        mcp_thread.daemon = True
        mcp_thread.start()
        
        # Start Gradio UI
        logging.info(f"Starting Gradio UI on port {gradio_port}")
        app = create_gradio_ui()
        app.launch(server_name=host, server_port=gradio_port, share=False, mcp_server=True)
    else:
        # Run MCP server only (no Gradio UI)
        logging.info(f"Starting WhatsApp MCP server (API only) with SSE transport on {host}:{port}")
        logging.info("Gradio UI disabled via GRADIO environment variable")
        try:
            mcp.settings.host = host
            mcp.settings.port = port
            # Initialize and run the server with SSE transport
            mcp.run(
                transport='sse'
            )
        except Exception as e:
            logging.error(f"Error starting MCP server: {e}")
            import traceback
            traceback.print_exc()
