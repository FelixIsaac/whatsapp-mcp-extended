# WhatsApp MCP Extended

An extended Model Context Protocol (MCP) server for WhatsApp with a curated **17-tool core profile** and a backward-compatible legacy profile for advanced messaging, group management, webhooks, presence, and more.

> Built on [AdamRussak/whatsapp-mcp](https://github.com/AdamRussak/whatsapp-mcp) (webhooks, containers) which forked [lharries/whatsapp-mcp](https://github.com/lharries/whatsapp-mcp) (original). Extended with reactions, message editing, polls, group management, presence, newsletters, and more.

![WhatsApp MCP](./example-use.png)

## What's New (vs Original)

| Feature | Original | Extended |
|---------|----------|----------|
| MCP Tools | 12 | **17 core / 46 legacy** |
| Reactions | - | ✅ |
| Edit/Delete Messages | - | ✅ |
| Group Management | - | ✅ |
| Polls | - | ✅ |
| History Sync | - | ✅ |
| Presence/Online Status | - | ✅ |
| Newsletters | - | ✅ |
| Webhooks | - | ✅ |
| Custom Nicknames | - | ✅ |

## Architecture

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│   whatsapp-bridge   │     │   whatsapp-mcp      │     │    webhook-ui       │
│   (Go + whatsmeow)  │◄────│   (Python + MCP)    │     │   (HTML/JS SPA)     │
│   Port: 8080        │     │   Ports: 8081,8082  │     │   Port: 8089        │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
         │                           │
         ▼                           ▼
    ┌─────────────────────────────────────┐
    │           SQLite (store/)           │
    │  messages.db │ whatsapp.db          │
    └─────────────────────────────────────┘
```

## Quick Start

### Docker (Recommended)

```bash
git clone https://github.com/felixisaac/whatsapp-mcp-extended
cd whatsapp-mcp-extended

docker network create n8n_n8n_traefik_network
docker-compose up -d

# Scan QR code to authenticate
docker-compose logs -f whatsapp-bridge
```

### Claude Desktop / Cursor Integration

Add to your MCP config (`claude_desktop_config.json` or Cursor settings):

```json
{
  "mcpServers": {
    "whatsapp": {
      "command": "uv",
      "args": ["run", "--directory", "/path/to/whatsapp-mcp-extended/whatsapp-mcp-server", "python", "main.py"]
    }
  }
}
```

## MCP Tools

The default tool profile is `legacy` to avoid breaking existing users. It exposes the historical narrow tools plus newer merged tools.

For new installs, prefer the smaller core profile:

```json
{
  "mcpServers": {
    "whatsapp": {
      "command": "uv",
      "args": ["run", "--directory", "/path/to/whatsapp-mcp-extended/whatsapp-mcp-server", "python", "main.py"],
      "env": {
        "WHATSAPP_MCP_TOOL_PROFILE": "core"
      }
    }
  }
}
```

Profiles:

| Profile | Tools | Use |
|---------|-------|-----|
| `legacy` | 46 | Default. Keeps old tool names working. |
| `core` | 17 | Smaller agent-facing surface for normal chat/search/send/media use. |

Merged replacements:

| Prefer | Replaces |
|--------|----------|
| `get_contact_context` | `get_contact_details`, `get_direct_chat_by_contact`, `get_contact_chats`, `get_last_interaction` |
| `manage_nickname` | `set_nickname`, `get_nickname`, `remove_nickname`, `list_nicknames` |
| `manage_group` | `create_group`, `add_group_members`, `remove_group_members`, `promote_to_admin`, `demote_admin`, `leave_group`, `update_group` |
| `manage_blocklist` | `get_blocklist`, `block_user`, `unblock_user` |
| `manage_newsletter` | `follow_newsletter`, `unfollow_newsletter`, `create_newsletter` |

### Messaging
| Tool | Description |
|------|-------------|
| `send_message` | Send text message |
| `send_file` | Send image/video/document |
| `send_audio_message` | Send voice message |
| `download_media` | Download received media |
| `send_reaction` | React to message with emoji |
| `edit_message` | Edit sent message |
| `delete_message` | Delete/revoke message |
| `mark_read` | Mark messages as read (blue ticks) |

### Chats & Messages
| Tool | Description |
|------|-------------|
| `list_chats` | List all chats |
| `get_chat` | Get chat by JID |
| `list_messages` | Search messages with filters |
| `get_message_context` | Get messages around a specific message |
| `get_direct_chat_by_contact` | Find DM with contact |
| `get_contact_chats` | All chats involving contact |
| `get_last_interaction` | Most recent message with contact |
| `request_history` | Request older message history |

### Contacts
| Tool | Description |
|------|-------------|
| `search_contacts` | Search by name/phone |
| `list_all_contacts` | List all contacts |
| `get_contact_details` | Full contact info |
| `set_nickname` | Set custom nickname |
| `get_nickname` | Get custom nickname |
| `remove_nickname` | Remove nickname |
| `list_nicknames` | List all nicknames |

### Groups
| Tool | Description |
|------|-------------|
| `get_group_info` | Group metadata & participants |
| `create_group` | Create new group |
| `add_group_members` | Add members |
| `remove_group_members` | Remove members |
| `promote_to_admin` | Promote to admin |
| `demote_admin` | Demote admin |
| `leave_group` | Leave group |
| `update_group` | Update name/topic |
| `create_poll` | Create poll in chat |

### Presence & Profile
| Tool | Description |
|------|-------------|
| `set_presence` | Set online/offline status |
| `subscribe_presence` | Subscribe to contact's presence |
| `get_profile_picture` | Get profile picture URL |
| `get_blocklist` | List blocked users |
| `block_user` | Block user |
| `unblock_user` | Unblock user |

### Newsletters (Channels)
| Tool | Description |
|------|-------------|
| `follow_newsletter` | Follow channel |
| `unfollow_newsletter` | Unfollow channel |
| `create_newsletter` | Create new channel |

## API Design Philosophy

Response data prioritizes **complete context with minimal interpretation**. See [METADATA_PHILOSOPHY.md](./docs/METADATA_PHILOSOPHY.md) for:

- Why we include raw data instead of pre-computed signals
- How we reduce token waste for consuming LLMs
- Response structure examples (Messages, Chats, Contacts)
- Design rules: raw facts, countable metrics, exclude nulls

**TL;DR:** Get all contact info in one response instead of repeated queries. LLM infers tone, urgency, relationships from raw data + metrics.

## Webhook System

Real-time HTTP webhooks for incoming messages with:
- **Triggers**: all, chat_jid, sender, keyword, media_type
- **Matching**: exact, contains, regex
- **Security**: HMAC-SHA256 signatures
- **Retry**: Exponential backoff

Access webhook UI at `http://localhost:8089`

## Development

### Manual Setup

```bash
# Bridge (Go 1.25+)
cd whatsapp-bridge && go run main.go

# MCP Server (Python 3.11+)
cd whatsapp-mcp-server && uv sync && uv run python main.py

# Webhook UI
cd whatsapp-webhook-ui && python3 -m http.server 8089
```

### Pre-build Checks

```bash
cd whatsapp-mcp-server
uv run python check.py  # Catches errors before docker build
```

### Updating whatsmeow

When you see `Client outdated (405)` errors:

```bash
cd whatsapp-bridge
go get -u go.mau.fi/whatsmeow@latest
go mod tidy
docker-compose build whatsapp-bridge
docker-compose up -d whatsapp-bridge
```

## Ports

| Service | Port | Description |
|---------|------|-------------|
| Bridge API | 8080 (→8180) | REST API |
| MCP Server | 8081 | SSE transport |
| Gradio UI | 8082 | Web testing UI |
| Webhook UI | 8089 | Webhook management |

## Troubleshooting

### Messages Not Delivering

If API returns success but messages show single checkmark:

```bash
docker-compose restart whatsapp-bridge
docker-compose logs --tail=10 whatsapp-bridge
# Should see: "✓ Connected to WhatsApp!"
```

### QR Code Issues

```bash
docker-compose logs -f whatsapp-bridge
# Scan QR with WhatsApp mobile app
```

## Credits

**Fork chain:**
- [lharries/whatsapp-mcp](https://github.com/lharries/whatsapp-mcp) - Original MCP server (12 tools)
- [AdamRussak/whatsapp-mcp](https://github.com/AdamRussak/whatsapp-mcp) - Added webhooks, container split, webhook UI
- This repo - Added reactions, edit/delete, groups, polls, presence, newsletters (41 tools)

**Libraries:**
- [whatsmeow](https://github.com/tulir/whatsmeow) - Go WhatsApp Web API
- [FastMCP](https://github.com/jlowin/fastmcp) - Python MCP SDK

### Community Acknowledgements

Several forks independently solved real problems and their ideas have been incorporated into this repo. Credit where it's due:

| Contributor | What they figured out |
|---|---|
| [simonseifert](https://github.com/simonseifert/whatsapp-mcp-extended-pro) | First to track the `direct_path` DB column needed for CDN fallback during media download; whatsmeow-native `Download()` approach in `/api/download`; correct DB path resolution across Docker/local environments; inline `Image` content blocks in `download_media` |
| [laudite](https://github.com/laudite/whatsapp-mcp-extended) | Media captions were silently dropped for images/video/docs — fixed in `ExtractTextContent()`; quoted/reply context in webhook payloads; `@mention` auto-detection; `request_history` peer-message target bug (was sending to group JID instead of own device JID) |
| [kasperpeulen](https://github.com/kasperpeulen/whatsapp-mcp-extended) | Contact name resolution priority chain (`FullName > PushName > FirstName > Business`) and the phone-number-cache bug; full call event pipeline (offer/accept/terminate/reject with duration); LID → phone JID resolution via `GetAltJID()` |
| [Coriatel](https://github.com/Coriatel/whatsapp-mcp-extended) | First working `/api/download` implementation with manual HKDF/AES-CBC decryption |
| [jedijashwa](https://github.com/jedijashwa/whatsapp-mcp-extended) | Reactions silently failing fix (wrong sender JID lookup); extended MIME type support for audio/document types |
| [slarrain](https://github.com/slarrain/whatsapp-mcp-extended) | LID JID normalization — diagnosed the silent conversation-splitting bug where WhatsApp's new LID format caused messages to land in separate chat threads |

If you've forked this repo and built something useful, open a PR or issue — good ideas deserve to flow upstream.

## License

MIT License - see [LICENSE](LICENSE) file.
