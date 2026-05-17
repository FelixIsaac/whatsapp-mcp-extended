# WhatsApp MCP Extended

An extended Model Context Protocol (MCP) server for WhatsApp with a toolset-gated, agent-facing surface for messaging, search, media, group management, webhooks, presence, and more.

> Built on [AdamRussak/whatsapp-mcp](https://github.com/AdamRussak/whatsapp-mcp) (webhooks, containers) which forked [lharries/whatsapp-mcp](https://github.com/lharries/whatsapp-mcp) (original). Extended with reactions, message editing, polls, group management, presence, newsletters, and more.

![WhatsApp MCP](./example-use.png)

## What's New (vs Original)

| Feature | Original | Extended |
|---------|----------|----------|
| MCP Tools | 12 | **26 default / 15 lean** |
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
│   whatsapp-bridge   │     │   whatsapp-mcp      │     │   whatsapp-web-ui   │
│   (Go + whatsmeow)  │◄────│   (Python + MCP)    │     │   (HTML/JS SPA)     │
│   Port: 8080        │     │   Port: 8081        │     │   Port: 8090        │
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

Version `0.2.0` exposes the full curated MCP surface by default for compatibility. Users who want a leaner agent context can opt into smaller toolsets.

Default toolsets:

```bash
WHATSAPP_MCP_TOOLSETS=all
```

Lean recommended toolsets:

```bash
WHATSAPP_MCP_TOOLSETS=core,send,media
```

Toolsets:

| Toolset | Default | Tools |
|---------|---------|-------|
| `core` | Yes | Search/read tools, contact context, group info, profile picture |
| `send` | Yes | `send_message`, `send_reaction`, `create_poll` |
| `media` | Yes | `send_file`, `send_audio_message`, `download_media` |
| `history` | No | `request_history` |
| `contacts_write` | No | `manage_nickname` |
| `message_admin` | No | `edit_message`, `delete_message`, `mark_read` |
| `groups` | No | `manage_group` |
| `presence` | No | `set_presence`, `subscribe_presence` |
| `account_admin` | No | `get_blocklist`, `manage_blocklist` |
| `newsletter` | No | `manage_newsletter` |

You can also expose individual tools with `WHATSAPP_MCP_TOOLS=manage_group,delete_message`.

Breaking change in `0.2.0`: older narrow tools are no longer exposed to the agent. Use the merged replacements below.

Migration:

| Prefer | Replaces |
|--------|----------|
| `get_contact_context` | `get_contact_details`, `get_direct_chat_by_contact`, `get_contact_chats`, `get_last_interaction` |
| `manage_nickname` | `set_nickname`, `get_nickname`, `remove_nickname`, `list_nicknames` |
| `manage_group` | `create_group`, `add_group_members`, `remove_group_members`, `promote_to_admin`, `demote_admin`, `leave_group`, `update_group` |
| `get_blocklist`, `manage_blocklist` | `get_blocklist`, `block_user`, `unblock_user` |
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
| `request_history` | Request older message history |

### Contacts
| Tool | Description |
|------|-------------|
| `search_contacts` | Search by name/phone |
| `list_all_contacts` | List all contacts |
| `get_contact_context` | Full contact info, related chats, and last interaction |
| `manage_nickname` | Set/get/remove/list custom nicknames |

### Groups
| Tool | Description |
|------|-------------|
| `get_group_info` | Group metadata & participants |
| `manage_group` | Create/update/leave groups and manage members/admins |
| `create_poll` | Create poll in chat |

### Presence & Profile
| Tool | Description |
|------|-------------|
| `set_presence` | Set online/offline status |
| `subscribe_presence` | Subscribe to contact's presence |
| `get_profile_picture` | Get profile picture URL |
| `get_blocklist` | List blocked users |
| `manage_blocklist` | Block/unblock users |

### Newsletters (Channels)
| Tool | Description |
|------|-------------|
| `manage_newsletter` | Follow/unfollow/create channels |

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

Access the web UI at `http://localhost:8090`

## Development

### Manual Setup

```bash
# Bridge (Go 1.25+)
cd whatsapp-bridge && go run main.go

# MCP Server (Python 3.11+)
cd whatsapp-mcp-server && uv sync && uv run python main.py

# Web UI
cd whatsapp-web-ui && npm install && npm run dev
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
| MCP Server | 8081 | Streamable HTTP transport |
| Web UI | 8090 | Chat, contacts, and webhook management |

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
- This repo - Added reactions, edit/delete, groups, polls, presence, newsletters, and a curated MCP tool surface

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
