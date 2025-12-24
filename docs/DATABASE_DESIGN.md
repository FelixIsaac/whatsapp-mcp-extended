# WhatsApp MCP Extended - Database Design Document

## Overview

This document details the complete database architecture, current schema, data flows, and future enhancement opportunities for the WhatsApp MCP Extended project.

---

## Database Architecture

The project uses **two separate SQLite databases**:

```
store/
├── messages.db      # Custom application data (messages, webhooks, nicknames)
└── whatsapp.db      # whatsmeow library data (session, contacts, encryption keys)
```

### Why Two Databases?

| Database | Owner | Purpose |
|----------|-------|---------|
| `messages.db` | This project | Store message history, webhooks, custom nicknames |
| `whatsapp.db` | whatsmeow library | WhatsApp session, E2E encryption keys, contact sync |

The separation exists because `whatsapp.db` is managed by the whatsmeow library and should not be modified directly.

---

## Current Schema

### Database 1: `messages.db`

#### Table: `chats`
Stores chat metadata for both individual and group conversations.

```sql
CREATE TABLE IF NOT EXISTS chats (
    jid TEXT PRIMARY KEY,           -- WhatsApp JID (e.g., 1234567890@s.whatsapp.net)
    name TEXT,                       -- Display name (contact name or group name)
    last_message_time TIMESTAMP      -- Timestamp of most recent message
);
```

**Notes:**
- JID format for individuals: `{phone}@s.whatsapp.net`
- JID format for groups: `{id}@g.us`
- Name is denormalized (copied at message receipt time, not dynamically looked up)

#### Table: `messages`
Stores all message content and metadata.

```sql
CREATE TABLE IF NOT EXISTS messages (
    id TEXT,                         -- WhatsApp message ID (hex string)
    chat_jid TEXT,                   -- Foreign key to chats.jid
    sender TEXT,                     -- Sender JID (who sent the message)
    content TEXT,                    -- Message text content
    timestamp TIMESTAMP,             -- When message was sent/received
    is_from_me BOOLEAN,              -- True if sent by account owner
    media_type TEXT,                 -- Type: image, video, audio, document, or NULL
    filename TEXT,                   -- Original filename for media
    url TEXT,                        -- WhatsApp CDN URL for media download
    media_key BLOB,                  -- Encryption key for media decryption
    file_sha256 BLOB,                -- SHA256 hash of decrypted media
    file_enc_sha256 BLOB,            -- SHA256 hash of encrypted media
    file_length INTEGER,             -- File size in bytes
    PRIMARY KEY (id, chat_jid),
    FOREIGN KEY (chat_jid) REFERENCES chats(jid)
);
```

**Notes:**
- Composite primary key allows same message ID across different chats
- Media files are NOT stored locally, only references (URL + decryption key)
- No indexes beyond primary key (performance issue for large datasets)

#### Table: `contact_nicknames`
User-defined custom nicknames that override WhatsApp contact names.

```sql
CREATE TABLE IF NOT EXISTS contact_nicknames (
    jid TEXT PRIMARY KEY,            -- Contact JID
    nickname TEXT NOT NULL,          -- Custom display name
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Notes:**
- Nickname takes highest priority in name resolution
- Separate from WhatsApp's native contact names

#### Table: `webhook_configs`
Webhook endpoint configurations.

```sql
CREATE TABLE IF NOT EXISTS webhook_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,              -- Descriptive name for the webhook
    webhook_url TEXT NOT NULL,       -- HTTP(S) URL to POST to
    secret_token TEXT,               -- HMAC-SHA256 signing secret
    enabled BOOLEAN DEFAULT 1,       -- Active/inactive toggle
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Table: `webhook_triggers`
Conditions that determine when webhooks fire.

```sql
CREATE TABLE IF NOT EXISTS webhook_triggers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_config_id INTEGER,       -- Foreign key to webhook_configs.id
    trigger_type TEXT NOT NULL,      -- Type: all, chat_jid, sender, keyword, media_type
    trigger_value TEXT,              -- Value to match against
    match_type TEXT DEFAULT 'exact', -- Matching: exact, contains, regex
    enabled BOOLEAN DEFAULT 1,       -- Active/inactive toggle
    FOREIGN KEY (webhook_config_id) REFERENCES webhook_configs(id)
);
```

**Trigger Types:**
| Type | Description | Example Value |
|------|-------------|---------------|
| `all` | Match every message | (none) |
| `chat_jid` | Match specific chat | `1234567890@s.whatsapp.net` |
| `sender` | Match specific sender | `1234567890@s.whatsapp.net` |
| `keyword` | Match message content | `urgent` |
| `media_type` | Match media type | `image` |

#### Table: `webhook_logs`
Audit trail of webhook delivery attempts.

```sql
CREATE TABLE IF NOT EXISTS webhook_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_config_id INTEGER,       -- Which webhook was triggered
    message_id TEXT,                 -- Associated message ID
    chat_jid TEXT,                   -- Which chat triggered it
    trigger_type TEXT,               -- Which trigger type matched
    trigger_value TEXT,              -- What value matched
    payload TEXT,                    -- Full JSON payload sent
    response_status INTEGER,         -- HTTP status code (200, 404, 500, etc.)
    response_body TEXT,              -- Server response (truncated to 1KB)
    attempt_count INTEGER DEFAULT 1, -- Number of delivery attempts
    delivered_at TIMESTAMP,          -- When successfully delivered (NULL if failed)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (webhook_config_id) REFERENCES webhook_configs(id)
);
```

---

### Database 2: `whatsapp.db` (whatsmeow managed)

These tables are created and managed by the whatsmeow library. **Do not modify directly.**

#### Key Tables (Read-Only Access)

```sql
-- Device/session information
whatsmeow_device
    - jid                    -- Account JID
    - registration_id        -- Device registration
    - noise_key             -- Encryption key
    - identity_key          -- Identity key pair
    - signed_pre_key        -- Signed pre-key
    - signed_pre_key_id
    - signed_pre_key_sig
    - adv_key               -- Advanced key
    - adv_details
    - adv_account_sig
    - adv_device_sig
    - platform              -- Device platform
    - business_name
    - push_name

-- Contact information synced from WhatsApp
whatsmeow_contacts
    - our_jid               -- Our account JID
    - their_jid             -- Contact's JID
    - first_name            -- First name
    - full_name             -- Full name
    - push_name             -- WhatsApp display name
    - business_name         -- Business account name

-- Chat settings
whatsmeow_chat_settings
    - our_jid
    - chat_jid
    - muted_until
    - pinned
    - archived

-- Encryption keys (E2E)
whatsmeow_identity_keys
whatsmeow_pre_keys
whatsmeow_sender_keys
whatsmeow_sessions
```

---

## Data Flow: WhatsApp → Database

```
┌─────────────────┐
│    WhatsApp     │
│    Servers      │
└────────┬────────┘
         │ WebSocket (encrypted)
         ▼
┌─────────────────┐
│    whatsmeow    │ ─────► whatsapp.db (session, contacts, keys)
│    library      │
└────────┬────────┘
         │ Go events
         ▼
┌─────────────────┐
│   handlers.go   │
│  (event router) │
└────────┬────────┘
         │
         ├──► events.Message ──────────────────┐
         │                                      │
         ├──► events.HistorySync ──────────────┤
         │                                      ▼
         │                              ┌──────────────┐
         │                              │ messages.db  │
         │                              │  - chats     │
         │                              │  - messages  │
         │                              └──────────────┘
         │
         ├──► events.Receipt ──────────► ❌ NOT CAPTURED
         ├──► events.Presence ─────────► ❌ NOT CAPTURED
         ├──► events.ChatPresence ─────► ❌ NOT CAPTURED
         └──► ReactionMessage ─────────► ❌ NOT CAPTURED
```

---

## What's Captured vs. Not Captured

### Currently Captured ✅

| Data | Source | Storage |
|------|--------|---------|
| Text messages | `events.Message` | `messages.content` |
| Media metadata | `events.Message` | `messages.media_type`, `url`, `media_key` |
| Sender info | `events.Message` | `messages.sender` |
| Timestamps | `events.Message` | `messages.timestamp` |
| Chat names | `GetGroupInfo()` / contacts | `chats.name` |
| Contact names | `whatsmeow_contacts` | Read from whatsapp.db |
| Custom nicknames | User input | `contact_nicknames` |
| Webhook configs | User input | `webhook_configs`, `webhook_triggers` |
| Webhook logs | Delivery attempts | `webhook_logs` |

### NOT Captured ❌

| Data | whatsmeow Event | Why Not Captured |
|------|-----------------|------------------|
| Reactions (emoji) | `events.Message.ReactionMessage` | Handler ignores reaction messages |
| Read receipts | `events.Receipt` | No handler registered |
| Message edits | `events.Message` (edited flag) | Overwrites content, no history |
| Message deletions | `events.Message` (revoke) | No handler, DB unchanged |
| Typing indicators | `events.ChatPresence` | No handler registered |
| Online/offline | `events.Presence` | No handler registered |
| Profile pictures | On-demand API | Not persisted |
| Group member changes | `events.GroupInfo` | Not persisted |
| Last seen | On-demand API | Not persisted |
| Status/Stories | `events.Message` (status broadcast) | Filtered out |

---

## Future Enhancement: Additional Tables

### 1. Message Reactions

```sql
CREATE TABLE IF NOT EXISTS message_reactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,        -- Message that was reacted to
    chat_jid TEXT NOT NULL,          -- Chat containing the message
    reactor_jid TEXT NOT NULL,       -- Who sent the reaction
    emoji TEXT NOT NULL,             -- Reaction emoji (e.g., "👍", "❤️")
    timestamp TIMESTAMP,             -- When reaction was sent
    is_from_me BOOLEAN,              -- Did we send this reaction?
    UNIQUE(message_id, chat_jid, reactor_jid),
    FOREIGN KEY (message_id, chat_jid) REFERENCES messages(id, chat_jid)
);

CREATE INDEX idx_reactions_message ON message_reactions(message_id, chat_jid);
```

**Use Case for Claude Code:**
- "What messages got the most reactions?"
- "Did they react to my last message?"
- "Show me messages with ❤️ reactions"

### 2. Read Receipts

```sql
CREATE TABLE IF NOT EXISTS read_receipts (
    message_id TEXT NOT NULL,        -- Message that was read
    chat_jid TEXT NOT NULL,          -- Chat containing the message
    reader_jid TEXT NOT NULL,        -- Who read it
    receipt_type TEXT,               -- 'delivered', 'read', 'played' (for audio)
    timestamp TIMESTAMP,             -- When receipt was received
    PRIMARY KEY (message_id, chat_jid, reader_jid, receipt_type),
    FOREIGN KEY (message_id, chat_jid) REFERENCES messages(id, chat_jid)
);

CREATE INDEX idx_receipts_message ON read_receipts(message_id, chat_jid);
```

**Use Case for Claude Code:**
- "Did they read my message?"
- "Which messages are still unread?"
- "When did they see my message?"

### 3. Message Edit History

```sql
CREATE TABLE IF NOT EXISTS message_edits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,        -- Original message ID
    chat_jid TEXT NOT NULL,          -- Chat containing the message
    version INTEGER NOT NULL,        -- Edit version (1, 2, 3...)
    content TEXT,                    -- Content at this version
    edited_at TIMESTAMP,             -- When this edit was made
    FOREIGN KEY (message_id, chat_jid) REFERENCES messages(id, chat_jid)
);

CREATE INDEX idx_edits_message ON message_edits(message_id, chat_jid);
```

**Use Case for Claude Code:**
- "What did the original message say?"
- "How many times was this message edited?"
- "Show me the edit history"

### 4. Presence/Online Status

```sql
CREATE TABLE IF NOT EXISTS presence_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    jid TEXT NOT NULL,               -- Contact JID
    status TEXT NOT NULL,            -- 'available', 'unavailable'
    last_seen TIMESTAMP,             -- Last seen timestamp (if available)
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_presence_jid ON presence_history(jid, recorded_at DESC);
```

**Use Case for Claude Code:**
- "When was [contact] last online?"
- "Is [contact] typically online at this time?"
- "Show me [contact]'s activity pattern"

### 5. Typing Indicators (Transient)

```sql
CREATE TABLE IF NOT EXISTS typing_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_jid TEXT NOT NULL,          -- Chat where typing occurred
    sender_jid TEXT NOT NULL,        -- Who is typing
    event_type TEXT NOT NULL,        -- 'typing', 'stopped', 'recording'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Auto-cleanup old entries (transient data)
-- Consider: DELETE FROM typing_events WHERE timestamp < datetime('now', '-1 hour')
```

**Use Case for Claude Code:**
- "Are they currently typing?" (real-time)
- Limited historical value - consider NOT storing

### 6. Group Membership History

```sql
CREATE TABLE IF NOT EXISTS group_members (
    group_jid TEXT NOT NULL,         -- Group JID
    member_jid TEXT NOT NULL,        -- Member JID
    is_admin BOOLEAN DEFAULT FALSE,
    is_super_admin BOOLEAN DEFAULT FALSE,
    joined_at TIMESTAMP,
    left_at TIMESTAMP,               -- NULL if still member
    PRIMARY KEY (group_jid, member_jid)
);

CREATE TABLE IF NOT EXISTS group_member_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_jid TEXT NOT NULL,
    member_jid TEXT NOT NULL,
    event_type TEXT NOT NULL,        -- 'join', 'leave', 'promote', 'demote'
    actor_jid TEXT,                  -- Who performed the action
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Use Case for Claude Code:**
- "Who is in this group?"
- "Who left the group recently?"
- "Who are the admins?"

---

## Value Analysis: What Should Claude Code Store?

### High Value (Recommend Storing)

| Data | Value for Claude | Effort | Priority |
|------|------------------|--------|----------|
| **Reactions** | Know sentiment, engagement | Low | High |
| **Read Receipts** | Know if message was seen | Low | High |
| **Edit History** | Context for conversations | Low | Medium |
| **Group Members** | Know who's in conversations | Medium | Medium |

### Medium Value (Consider Storing)

| Data | Value for Claude | Effort | Priority |
|------|------------------|--------|----------|
| **Presence History** | Activity patterns | Low | Low |
| **Profile Pictures** | Visual context (if multimodal) | Medium | Low |
| **Message Deletions** | Know what was removed | Low | Low |

### Low Value (Skip)

| Data | Why Skip |
|------|----------|
| **Typing Indicators** | Transient, no historical value |
| **Real-time Presence** | Changes constantly, stale quickly |
| **Status/Stories** | 24hr expiry, different use case |

---

## Benefits for Claude Code Tasks

### With Current Schema

Claude can:
- ✅ Search message history
- ✅ Find conversations with contacts
- ✅ List recent chats
- ✅ Send messages and media
- ✅ Manage custom nicknames
- ✅ Get group info (on-demand)

### With Enhanced Schema

Claude could additionally:
- ✅ Know if messages were read → Better follow-up timing
- ✅ See reactions → Understand sentiment without asking
- ✅ Track edit history → Full conversation context
- ✅ Know group membership → Better group interactions
- ✅ Activity patterns → Smart scheduling suggestions

### Example Enhanced Queries

```sql
-- "Did they see my message about the meeting?"
SELECT m.content, r.receipt_type, r.timestamp
FROM messages m
LEFT JOIN read_receipts r ON m.id = r.message_id AND m.chat_jid = r.chat_jid
WHERE m.chat_jid = ? AND m.is_from_me = 1
ORDER BY m.timestamp DESC LIMIT 1;

-- "What messages got positive reactions?"
SELECT m.content, GROUP_CONCAT(mr.emoji) as reactions
FROM messages m
JOIN message_reactions mr ON m.id = mr.message_id AND m.chat_jid = mr.chat_jid
GROUP BY m.id, m.chat_jid
HAVING COUNT(*) > 0;

-- "Who's in this group and are they admins?"
SELECT member_jid, is_admin, joined_at
FROM group_members
WHERE group_jid = ? AND left_at IS NULL;
```

---

## Implementation Roadmap

### Phase 1: Quick Wins (Low Effort, High Value)

1. **Add message_reactions table** + handler for `ReactionMessage`
2. **Add read_receipts table** + handler for `events.Receipt`
3. **Add indexes** on frequently queried columns

### Phase 2: Medium Effort

4. **Add message_edits table** + track edit events
5. **Add group_members table** + sync on group events
6. **Add deletion tracking** (soft delete with `deleted_at` column)

### Phase 3: Infrastructure

7. **Data retention policies** (auto-cleanup old data)
8. **Multi-account support** (add `account_jid` to all tables)
9. **Database migrations** (versioned schema changes)

---

## Current Limitations

| Limitation | Impact | Solution |
|------------|--------|----------|
| Single account only | Can't manage multiple WhatsApp accounts | Add account_jid FK |
| No indexes | Slow queries on large datasets | Add indexes |
| No retention policy | Database grows forever | Add cleanup jobs |
| Chat name denormalized | Stale names if contact changes | Reference instead of copy |
| No foreign key enforcement | SQLite FK off by default | Enable with PRAGMA |
| Media not stored locally | Requires network to fetch | Optional local cache |

---

## Recommended Indexes

```sql
-- Speed up message queries
CREATE INDEX IF NOT EXISTS idx_messages_chat_time
ON messages(chat_jid, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_messages_sender
ON messages(sender);

CREATE INDEX IF NOT EXISTS idx_messages_content
ON messages(content) WHERE content IS NOT NULL;

-- Speed up webhook queries
CREATE INDEX IF NOT EXISTS idx_webhook_logs_config
ON webhook_logs(webhook_config_id, created_at DESC);

-- Speed up chat queries
CREATE INDEX IF NOT EXISTS idx_chats_time
ON chats(last_message_time DESC);
```

---

## Summary

| Aspect | Current State | Recommended |
|--------|---------------|-------------|
| Message storage | ✅ Complete | ✅ Keep |
| Reactions | ❌ Not stored | ✅ Add |
| Read receipts | ❌ Not stored | ✅ Add |
| Edit history | ❌ Not stored | ✅ Add |
| Presence | ❌ Not stored | ⚠️ Optional |
| Typing | ❌ Not stored | ❌ Skip |
| Group members | ❌ Not stored | ✅ Add |
| Multi-account | ❌ Not supported | ⚠️ Future |
| Indexes | ❌ Missing | ✅ Add |
| Retention | ❌ None | ✅ Add |

---

## Files Reference

| File | Purpose |
|------|---------|
| `whatsapp-bridge/internal/database/store.go` | Schema definitions, table creation |
| `whatsapp-bridge/internal/database/messages.go` | Message CRUD operations |
| `whatsapp-bridge/internal/database/webhooks.go` | Webhook CRUD operations |
| `whatsapp-bridge/internal/whatsapp/handlers.go` | Event handlers (where to add new captures) |
| `whatsapp-mcp-server/whatsapp.py` | Python database queries |
