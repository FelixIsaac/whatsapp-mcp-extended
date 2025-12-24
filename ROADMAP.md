# WhatsApp MCP Extended - Feature Roadmap

## Project Context

- **Base**: `whatsapp-mcp-extended` (Dockerized, with webhooks)
- **Core library**: whatsmeow (Go) - WhatsApp Web multi-device API
- **Status**: Docker support ✅, Webhooks ✅, Contact management ✅

## Current State

### Implemented (16+ MCP tools)
- `search_contacts`, `list_messages`, `list_chats`
- `get_chat`, `get_direct_chat_by_contact`, `get_contact_chats`
- `get_last_interaction`, `get_message_context`
- `send_message`, `send_file`, `send_audio_message`, `download_media`
- Contact nicknames: `set_contact_nickname`, `get_contact_nickname`, `remove_contact_nickname`, `list_contact_nicknames`
- Webhook system (via REST API)

### Architecture
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

---

## Tool Comparison: whatsapp vs whatsapp-extended

| Tool | `whatsapp` | `whatsapp-extended` |
|------|:----------:|:-------------------:|
| **Messages** | | |
| `search_contacts` | ✅ | ✅ |
| `list_messages` | ✅ | ✅ |
| `list_chats` | ✅ | ✅ |
| `get_chat` | ✅ | ✅ |
| `get_direct_chat_by_contact` | ✅ | ✅ |
| `get_contact_chats` | ✅ | ✅ |
| `get_last_interaction` | ✅ | ✅ |
| `get_message_context` | ✅ | ✅ |
| **Sending** | | |
| `send_message` | ✅ | ✅ |
| `send_file` | ✅ | ✅ |
| `send_audio_message` | ✅ | ✅ |
| `download_media` | ✅ | ✅ |
| **Contacts** | | |
| `get_contact_details` | ❌ | ✅ |
| `list_all_contacts` | ❌ | ✅ |
| **Nicknames** | | |
| `set_nickname` | ❌ | ✅ |
| `get_nickname` | ❌ | ✅ |
| `remove_nickname` | ❌ | ✅ |
| `list_nicknames` | ❌ | ✅ |
| **Infrastructure** | | |
| Docker support | ❌ | ✅ |
| Webhook system | ❌ | ✅ |
| Gradio UI | ❌ | ✅ |
| **Phase 1 Features** | | |
| `send_reaction` | ❌ | ✅ |
| `edit_message` | ❌ | ✅ |
| `delete_message` | ❌ | ✅ |
| `get_group_info` | ❌ | ✅ |
| `mark_read` | ❌ | ✅ |
| **Phase 2 Features** | | |
| `create_group` | ❌ | ✅ |
| `add_group_members` | ❌ | ✅ |
| `remove_group_members` | ❌ | ✅ |
| `promote_to_admin` | ❌ | ✅ |
| `demote_admin` | ❌ | ✅ |
| `leave_group` | ❌ | ✅ |
| `update_group` | ❌ | ✅ |
| **Phase 3 Features** | | |
| `create_poll` | ❌ | ✅ |
| **Phase 4 Features** | | |
| `request_history` | ❌ | ✅ |

**Total: 12 tools (whatsapp) → 32 tools (extended)**

---

## Phase 1: Quick Wins (Easy)

### 1.1 Reactions
- **Send**: `client.BuildReaction(chat, sender, msgID, emoji)`
- **Store**: Add `events.Reaction` handler, store in DB
- **MCP tool**: `send_reaction(message_id, chat_jid, emoji)`

### 1.2 Edit Message
- **Method**: `client.BuildEdit(chat, msgID, newContent)`
- **MCP tool**: `edit_message(message_id, chat_jid, new_content)`

### 1.3 Delete/Revoke Message
- **Method**: `client.BuildRevoke(chat, sender, msgID)`
- **MCP tool**: `delete_message(message_id, chat_jid)`

### 1.4 Get Group Info
- **Method**: `client.GetGroupInfo(jid)`
- **MCP tool**: `get_group_info(group_jid)` → returns name, description, participants, admins

### 1.5 Mark as Read
- **Method**: `client.MarkRead([]types.MessageID, timestamp, chat, sender)`
- **MCP tool**: `mark_read(message_ids, chat_jid)`

---

## Phase 2: Group Management (Medium)

### 2.1 Create Group
- **Method**: `client.CreateGroup(ctx, ReqCreateGroup{Name, Participants})`
- **MCP tool**: `create_group(name, participant_jids[])`

### 2.2 Add Group Members
- **Method**: `client.UpdateGroupParticipants(jid, []JID, ParticipantChangeAdd)`
- **MCP tool**: `add_group_members(group_jid, member_jids[])`

### 2.3 Remove Group Members
- **Method**: `client.UpdateGroupParticipants(jid, []JID, ParticipantChangeRemove)`
- **MCP tool**: `remove_group_members(group_jid, member_jids[])`

### 2.4 Promote/Demote Admin
- **Methods**: `ParticipantChangePromote`, `ParticipantChangeDemote`
- **MCP tools**: `promote_to_admin(group_jid, member_jid)`, `demote_admin(...)`

### 2.5 Leave Group
- **Method**: `client.LeaveGroup(jid)`
- **MCP tool**: `leave_group(group_jid)`

### 2.6 Update Group Settings
- **Methods**: `SetGroupName()`, `SetGroupPhoto()`, `SetGroupTopic()`
- **MCP tools**: `update_group_name(...)`, `update_group_photo(...)`

---

## Phase 3: Polls & Rich Messages (Medium)

### 3.1 Create Poll
- **Method**: `client.BuildPollCreation(name, options[], selectableCount)`
- **MCP tool**: `create_poll(chat_jid, question, options[], multi_select)`

### 3.2 Vote on Poll
- **Method**: `client.BuildPollVote(ctx, pollInfo, selectedOptions[])`
- **MCP tool**: `vote_poll(poll_message_id, chat_jid, selected_options[])`

### 3.3 Store Rich Message Types
Extend `extractTextContent()` and DB schema for:
- Reactions (separate table or column)
- Polls (poll_id, question, options JSON)
- Poll votes
- Locations (lat, lng, name)
- Contacts (vCard)
- Stickers

---

## Phase 4: History Sync Enhancement (Medium-Hard)

### 4.1 Configurable Full Sync
Configure `DeviceProps.HistorySyncConfig` before device creation:

```go
store.DeviceProps.HistorySyncConfig = &waProto.DeviceProps_HistorySyncConfig{
    FullSyncDaysLimit:   proto.Uint32(365),   // 1 year
    FullSyncSizeMbLimit: proto.Uint32(5000),  // 5GB
    StorageQuotaMb:      proto.Uint32(5000),
}
```

**Limitations**:
- Only applies at initial device link
- Must re-link device to change
- Phone must have messages to sync

### 4.2 On-Demand History Request
- **Method**: `client.BuildHistorySyncRequest(lastKnownMsg, count)`
- **MCP tool**: `request_history(chat_jid, count)` - fetch older messages per-chat

---

## Phase 5: Advanced Features (Hard)

### 5.1 Presence/Online Status
- `client.SubscribePresence(jid)`
- `client.SendPresence(types.PresenceAvailable)`

### 5.2 Profile Management
- `client.GetProfilePictureInfo(jid, params)`
- `client.SetGroupPhoto(jid, avatar)`

### 5.3 Block/Unblock
- `client.GetBlocklist()`
- `client.UpdateBlocklist(changes)`

### 5.4 Newsletters (if needed)
- `client.CreateNewsletter(params)`
- `client.FollowNewsletter(jid)`

---

## Implementation Checklist

### Go Bridge Changes (`whatsapp-bridge/`)
- [ ] Add new HTTP endpoints for each feature
- [ ] Add event handlers for reactions, polls, etc.
- [ ] Extend SQLite schema for new message types
- [ ] Add history sync config option

### Python MCP Server Changes (`whatsapp-mcp-server/`)
- [ ] Add new MCP tool definitions
- [ ] Add corresponding functions in `whatsapp.py`
- [ ] Update type hints and docstrings

### Documentation
- [ ] Update README with new features
- [ ] Add configuration docs for history sync
- [ ] Document breaking changes (re-link for full sync)

---

## Database Schema Additions

```sql
-- Reactions table
CREATE TABLE IF NOT EXISTS reactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL,
    sender TEXT NOT NULL,
    emoji TEXT NOT NULL,
    timestamp TIMESTAMP,
    FOREIGN KEY (chat_jid) REFERENCES chats(jid)
);

-- Polls table
CREATE TABLE IF NOT EXISTS polls (
    id TEXT PRIMARY KEY,
    chat_jid TEXT NOT NULL,
    creator TEXT NOT NULL,
    question TEXT NOT NULL,
    options TEXT NOT NULL,  -- JSON array
    selectable_count INTEGER DEFAULT 1,
    created_at TIMESTAMP,
    FOREIGN KEY (chat_jid) REFERENCES chats(jid)
);

-- Poll votes table
CREATE TABLE IF NOT EXISTS poll_votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    poll_id TEXT NOT NULL,
    voter TEXT NOT NULL,
    selected_options TEXT NOT NULL,  -- JSON array
    timestamp TIMESTAMP,
    FOREIGN KEY (poll_id) REFERENCES polls(id)
);

-- Extend messages table
ALTER TABLE messages ADD COLUMN message_type TEXT DEFAULT 'text';
ALTER TABLE messages ADD COLUMN location_lat REAL;
ALTER TABLE messages ADD COLUMN location_lng REAL;
ALTER TABLE messages ADD COLUMN quoted_message_id TEXT;
```

---

## Completed Features ✅

- [x] Docker support (docker-compose)
- [x] Webhook system with triggers
- [x] Contact nickname management
- [x] Webhook management UI
- [x] **Phase 1 Quick Wins** (5 new MCP tools):
  - [x] `send_reaction` - React to messages with emoji
  - [x] `edit_message` - Edit previously sent messages
  - [x] `delete_message` - Delete/revoke messages
  - [x] `get_group_info` - Get group name, description, participants
  - [x] `mark_read` - Mark messages as read (blue ticks)
- [x] **Phase 2 Group Management** (7 new MCP tools):
  - [x] `create_group` - Create new WhatsApp group
  - [x] `add_group_members` - Add members to a group
  - [x] `remove_group_members` - Remove members from a group
  - [x] `promote_to_admin` - Promote member to admin
  - [x] `demote_admin` - Demote admin to regular member
  - [x] `leave_group` - Leave a group
  - [x] `update_group` - Update group name/topic
- [x] **Phase 3 Polls** (1 new MCP tool):
  - [x] `create_poll` - Create and send polls with single/multi-select options
  - [ ] `vote_poll` - Deferred (requires poll message storage)
- [x] **Phase 4 History Sync** (1 new MCP tool + config):
  - [x] `request_history` - Request older messages for a chat (on-demand sync)
  - [x] Configurable history sync via env vars (HISTORY_SYNC_DAYS_LIMIT, HISTORY_SYNC_SIZE_MB, STORAGE_QUOTA_MB)
  - [x] Research documented in `docs/HISTORY_SYNC_RESEARCH.md`

---

## Unresolved Questions

1. Reactions: separate table or JSON in messages?
2. Priority order for Phase 1 features?
3. Config file format for history sync days?
4. Backwards compatibility with existing DBs?
