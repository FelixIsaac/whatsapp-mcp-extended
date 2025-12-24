# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WhatsApp MCP Server - a Model Context Protocol server enabling AI integration with personal WhatsApp accounts. Containerized microservices architecture with Go bridge, Python MCP server, and web UI.

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

**whatsapp-bridge/** (Go): WhatsApp API connection via whatsmeow, message handling, webhook delivery, REST API
**whatsapp-mcp-server/** (Python): MCP protocol implementation, message search, media handling, contact management
**whatsapp-webhook-ui/**: Web interface for webhook configuration

## Commands

### Docker (recommended)
```bash
docker network create n8n_n8n_traefik_network  # first time only
docker-compose up -d                           # start all services
docker-compose logs -f whatsapp-bridge         # watch for QR code
docker-compose build                           # rebuild all
docker-compose build whatsapp-bridge           # rebuild specific
```

**IMPORTANT: No hot-reload** - code is COPY'd into containers at build time. After any code changes:
```bash
docker-compose build <service>   # rebuild changed service
docker-compose up -d <service>   # restart with new image
```

### Development
```bash
# Bridge (Go 1.24+)
cd whatsapp-bridge && go run main.go
cd whatsapp-bridge && go test ./...

# MCP Server (Python 3.11+, requires uv)
cd whatsapp-mcp-server && uv sync
cd whatsapp-mcp-server && uv run python whatsapp.py

# Webhook UI
cd whatsapp-webhook-ui && python3 -m http.server 8089
```

### Pre-build Checks (run BEFORE docker-compose build)
```bash
cd whatsapp-mcp-server

# Install dev tools (first time only)
uv sync --all-extras

# Quick syntax check (~1s) - catches missing imports, syntax errors
uv run python check.py --quick

# Full check (~5s) - includes ruff linting + mypy type checking
uv run python check.py
```

This catches errors like missing imports before waiting 4-5 min for Docker build.

### Updating whatsmeow (when 405 errors appear)
```bash
cd whatsapp-bridge
go get -u go.mau.fi/whatsmeow@latest
go mod tidy
```

## Key Patterns

### Go Bridge Structure
- `internal/api/` - HTTP handlers, CORS middleware, JSON responses
- `internal/whatsapp/` - WhatsApp client wrapper, message/media handling
- `internal/webhook/` - Webhook manager, trigger matching, delivery with retries
- `internal/database/` - SQLite message store, webhook config persistence
- `internal/types/` - Shared type definitions (WebhookConfig, WebhookPayload, etc.)

### Python MCP Server
- `whatsapp.py` - Core library: dataclasses (Message, Chat, Contact), database queries, API calls to bridge
- `main.py` - MCP server with **stdio** transport (for Claude Code CLI)
- `gradio-main.py` - MCP server with **SSE** transport + Gradio UI (for Docker/network)
- Uses `BRIDGE_HOST` env var to connect to Go bridge (e.g., `localhost:8180` or `hostname:port`)

### Webhook System
Trigger types: `all`, `chat_jid`, `sender`, `keyword`, `media_type`
Match types: `exact`, `contains`, `regex`
Delivery: async with exponential backoff, HMAC-SHA256 signatures

### JID Formats
- Individual: `{phone}@s.whatsapp.net`
- Group: `{id}@g.us`

## Ports
- 8080: Bridge REST API (mapped to 8180 in docker-compose)
- 8081: MCP SSE server
- 8082: Gradio UI
- 8089: Webhook management UI

## Environment Variables
- `BRIDGE_HOST`: Go bridge hostname (default: localhost, set to container name in docker)
- `GRADIO`: Enable/disable Gradio UI (true/false)
- `DEBUG`: Enable debug logging

## Technology References

### whatsmeow (Go WhatsApp Library)
- **Repo:** https://github.com/tulir/whatsmeow
- **Docs:** https://pkg.go.dev/go.mau.fi/whatsmeow
- **Key Types:**
  - `SendResponse` - Returns ID, Timestamp from SendMessage
  - `events.Message` - Incoming message structure
  - `types.GroupInfo` - Group metadata (name, topic, participants)
  - `types.JID` - WhatsApp identifier format

### Message ID Format
- Format: Hex string (e.g., `3EB028A580CF7CC9AAF3A2`)
- Used for: edit, delete, react, mark_read operations
- Returned by: send_message, send_file, send_audio_message

### JID (Jabber ID) Format
- Individual: `{phone}@s.whatsapp.net` (e.g., `6593439326@s.whatsapp.net`)
- Group: `{id}@g.us` (e.g., `120363123456789012@g.us`)
- LID (linked device): `{id}@lid`
- Broadcast: `status@broadcast`

### MCP Tool Output Format
All tools return structured dictionaries:
- `send_message/send_file/send_audio`: `{success, message_id, timestamp, recipient, error?}`
- `list_messages`: `[{id, chat_jid, chat_name, sender, content, timestamp, is_from_me, media_type, filename?, file_length?}]`
- `list_chats`: `[{jid, name, is_group, last_message_time, last_message, last_sender, last_is_from_me}]`
- `search_contacts`: `[{jid, phone_number, name, first_name, full_name, push_name, business_name, nickname}]`
