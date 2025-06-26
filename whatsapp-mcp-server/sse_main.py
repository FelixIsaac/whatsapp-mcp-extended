from mcp.server.fastmcp import FastMCP
from mcp.server.sse import SseServerTransport
from starlette.applications import Starlette
from starlette.routing import Mount
import uvicorn
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
    download_media as whatsapp_download_media
)

# Initialize FastMCP server
mcp = FastMCP("whatsapp")

# All your @mcp.tool() decorated functions remain the same.
# (The code for all the tools from main.py goes here)

# --- Tool definitions from main.py ---
@mcp.tool()
def search_contacts(query: str):
    """Search WhatsApp contacts by name or phone number."""
    return whatsapp_search_contacts(query)

@mcp.tool()
def list_messages(
    after: str | None = None,
    before: str | None = None,
    sender_phone_number: str | None = None,
    chat_jid: str | None = None,
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_context: bool = True,
    context_before: int = 1,
    context_after: int = 1,
):
    """Get WhatsApp messages matching specified criteria with optional context."""
    return whatsapp_list_messages(
        after, before, sender_phone_number, chat_jid, query, limit, page, include_context, context_before, context_after
    )

@mcp.tool()
def list_chats(
    query: str | None = None,
    limit: int = 20,
    page: int = 0,
    include_last_message: bool = True,
    sort_by: str = "last_active",
):
    """Get WhatsApp chats matching specified criteria."""
    return whatsapp_list_chats(query, limit, page, include_last_message, sort_by)

@mcp.tool()
def get_chat(chat_jid: str, include_last_message: bool = True):
    """Get WhatsApp chat metadata by JID."""
    return whatsapp_get_chat(chat_jid, include_last_message)

@mcp.tool()
def get_direct_chat_by_contact(sender_phone_number: str):
    """Get WhatsApp chat metadata by sender phone number."""
    return whatsapp_get_direct_chat_by_contact(sender_phone_number)

@mcp.tool()
def get_contact_chats(jid: str, limit: int = 20, page: int = 0):
    """Get all WhatsApp chats involving the contact."""
    return whatsapp_get_contact_chats(jid, limit, page)

@mcp.tool()
def get_last_interaction(jid: str):
    """Get most recent WhatsApp message involving the contact."""
    return whatsapp_get_last_interaction(jid)

@mcp.tool()
def get_message_context(message_id: str, before: int = 5, after: int = 5):
    """Get context around a specific WhatsApp message."""
    return whatsapp_get_message_context(message_id, before, after)

@mcp.tool()
def send_message(recipient: str, message: str):
    """Send a WhatsApp message to a person or group."""
    return whatsapp_send_message(recipient, message)

@mcp.tool()
def send_file(recipient: str, media_path: str):
    """Send a file such as a picture, raw audio, video or document via WhatsApp."""
    return whatsapp_send_file(recipient, media_path)

@mcp.tool()
def send_audio_message(recipient: str, media_path: str):
    """Send any audio file as a WhatsApp audio message."""
    return whatsapp_audio_voice_message(recipient, media_path)

@mcp.tool()
def download_media(message_id: str, chat_jid: str):
    """Download media from a WhatsApp message and get the local file path."""
    return whatsapp_download_media(message_id, chat_jid)

# --- End of Tool definitions ---


def create_sse_app(mcp_server: FastMCP) -> Starlette:
    sse_transport = SseServerTransport("/sse")

    async def handle_sse(scope, receive, send):
        async with sse_transport.connect_sse(scope, receive, send) as (
            read_stream,
            write_stream,
        ):
            await mcp_server.run(
                read_stream,
                write_stream,
                mcp_server.create_initialization_options(),
            )

    app = Starlette(
        routes=[
            Mount("/sse", handle_sse),
        ]
    )
    return app

app = create_sse_app(mcp)

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8081)