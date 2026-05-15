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

### Database Migrations (for schema/structure changes)

**IMPORTANT:** When changes affect database schema (new columns, indexes, tables), we provide migration scripts for existing users. Don't rebuild containers from scratch - run migrations on existing databases.

**For existing database (`store/messages.db`):**

```bash
# Find latest migration in whatsapp-bridge/migrations/
ls -la whatsapp-bridge/migrations/

# Run migration (safe - non-destructive, adds only)
sqlite3 whatsapp-bridge/store/messages.db < whatsapp-bridge/migrations/001_add_metadata_fields.sql

# Verify migration
sqlite3 whatsapp-bridge/store/messages.db "SELECT name FROM sqlite_master WHERE type='index' ORDER BY name;"
```

**For Docker deployment:**

```bash
# Enter container, run migration
docker exec whatsapp-bridge sqlite3 /app/whatsapp-bridge/store/messages.db < whatsapp-bridge/migrations/001_add_metadata_fields.sql

# Or: copy migration into container and run
docker cp whatsapp-bridge/migrations/001_add_metadata_fields.sql whatsapp-bridge:/tmp/
docker exec whatsapp-bridge sqlite3 /app/whatsapp-bridge/store/messages.db < /tmp/001_add_metadata_fields.sql
```

**Migration Safety:**
- ✅ All migrations are **append-only** (add columns, indexes, tables)
- ✅ **Backward compatible** (old code still works)
- ✅ **Idempotent** (use `IF NOT EXISTS`, safe to run multiple times)
- ✅ **Rollback** (backup `store/messages.db` before running)

**When creating new migrations:**
- Place in `whatsapp-bridge/migrations/`
- Use sequential naming: `001_feature.sql`, `002_feature.sql`
- Include comments explaining each change
- Ensure all statements use `IF NOT EXISTS` guards
- Don't use `DROP TABLE` or destructive commands
- Verify with existing schema before merging

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

## Security

### API Authentication
**Required in production.** Set `API_KEY` env var - bridge fails to start if unset.
- Dev mode: `DISABLE_AUTH_CHECK=true` to skip
- Generate: `openssl rand -hex 32`
- Uses constant-time comparison

### Webhook URLs
Private IPs blocked by default (10.x, 172.16.x, 192.168.x, 127.x, 169.254.x). Set `DISABLE_SSRF_CHECK=true` for testing.

### Media Paths
Must be within `/app/media`, `/app/store`, or `/tmp`. Set `DISABLE_PATH_CHECK=true` for development.

### Rate Limiting
100 requests/minute per IP on bridge API.

### CORS
Allowed origins: `localhost:8089`, `localhost:8082`. Configurable in `middleware.go`.

### Containers
Run as non-root `appuser` in production.

## Code Standards

### Go
- Use `logger` for logging, not `fmt.Println` (except QR/server status)
- All exported functions need godoc comments
- Table-driven tests preferred
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Critical startup errors: `os.Exit(1)` not `return`

### Python
- Use `logger` from `lib.utils`, not `print()`
- Type hints on all functions
- Docstrings on all public functions
- Raise exceptions, don't return empty on error
- Python modules live in `lib/`: models, database, bridge, utils

## Testing

### Running Tests
```bash
# Python
cd whatsapp-mcp-server && uv run pytest --cov=lib -v

# Go
cd whatsapp-bridge && go test -v -race ./...
```

### Coverage Target
Minimum 50% coverage. See `docs/TESTING_ROADMAP.md` for gaps and priorities.

### CI/CD
GitHub Actions run on push/PR to main:
- `.github/workflows/go-test.yml` - Go tests + build
- `.github/workflows/python-test.yml` - Python tests + type check
- `.github/workflows/lint.yml` - golangci-lint, ruff, hadolint
