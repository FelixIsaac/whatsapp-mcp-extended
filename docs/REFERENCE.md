# Technology references and security notes

Moved from CLAUDE.md. Read before touching whatsmeow calls, message or JID handling, MCP tool output, API auth, webhooks, media paths, rate limits, CORS or containers.

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


### Chat names
- `GetChatName` keeps the first real name it stores for a chat and does not re-read it on later messages.
- A group rename arrives as `events.GroupInfo`; `main.go` passes the new name to `MessageStore.RenameChat`.
