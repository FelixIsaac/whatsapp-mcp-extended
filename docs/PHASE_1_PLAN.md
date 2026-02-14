# Phase 1: Enhanced Metadata & Database Optimization

**Goal:** Add comprehensive metadata fields to Messages, Chats, and Contacts while optimizing database performance with indexes and denormalization.

**Philosophy:** Provide complete context upfront (raw data + metrics) to reduce LLM token waste and multiple queries.

---

## Overview

### What We're Adding

**Messages:** 12 new fields
**Chats:** 13 new fields
**Contacts:** 11 new fields

**Total:** 36 new data points + 6 database indexes + 3 new tables

### Why This Matters

- **Before:** LLM asks for messages → gets basic data → asks "who is this sender?" → asks "what's the reaction?" → 3+ queries
- **After:** LLM gets messages with sender_contact_info, reaction_summary, quoted context, media details → 1 query, complete context

---

## Database Changes (Required First)

### New Indexes (Critical)

```sql
-- messages table (biggest impact on query performance)
CREATE INDEX IF NOT EXISTS idx_messages_chat_timestamp
  ON messages(chat_jid, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_messages_sender
  ON messages(sender);

CREATE INDEX IF NOT EXISTS idx_messages_media_type
  ON messages(chat_jid, media_type);

-- chats table
CREATE INDEX IF NOT EXISTS idx_chats_last_message_time
  ON chats(last_message_time DESC);

-- whatsmeow_contacts (in whatsapp.db)
CREATE INDEX IF NOT EXISTS idx_whatsmeow_their_jid
  ON whatsmeow_contacts(their_jid);
```

**Why:** All current queries do full table scans. These indexes reduce 1000-message scans to 10-message lookups.

### New Columns (messages table)

```sql
-- Message threading & editing
ALTER TABLE messages ADD COLUMN quoted_message_id TEXT;
ALTER TABLE messages ADD COLUMN quoted_sender_name TEXT;
ALTER TABLE messages ADD COLUMN reply_to_id TEXT;
ALTER TABLE messages ADD COLUMN edit_count INTEGER DEFAULT 0;
ALTER TABLE messages ADD COLUMN is_edited BOOLEAN DEFAULT 0;
ALTER TABLE messages ADD COLUMN last_edited_at TIMESTAMP;

-- Message state & forward
ALTER TABLE messages ADD COLUMN is_forwarded BOOLEAN DEFAULT 0;
ALTER TABLE messages ADD COLUMN forwarded_from TEXT;
ALTER TABLE messages ADD COLUMN is_system_message BOOLEAN DEFAULT 0;
ALTER TABLE messages ADD COLUMN system_message_type TEXT;
```

### New Columns (chats table)

```sql
-- Denormalized counts (updated via triggers)
ALTER TABLE chats ADD COLUMN total_message_count INTEGER DEFAULT 0;
ALTER TABLE chats ADD COLUMN message_count_today INTEGER DEFAULT 0;
ALTER TABLE chats ADD COLUMN message_count_7days INTEGER DEFAULT 0;
ALTER TABLE chats ADD COLUMN last_activity TIMESTAMP;

-- Group metadata
ALTER TABLE chats ADD COLUMN is_group BOOLEAN DEFAULT 0;
ALTER TABLE chats ADD COLUMN admin_count INTEGER DEFAULT 0;
```

### New Tables

#### message_edits (Audit trail)
```sql
CREATE TABLE IF NOT EXISTS message_edits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL,
    old_content TEXT NOT NULL,
    new_content TEXT NOT NULL,
    edited_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id, chat_jid) REFERENCES messages(id, chat_jid),
    FOREIGN KEY (chat_jid) REFERENCES chats(jid)
);
CREATE INDEX idx_message_edits_msg ON message_edits(message_id, chat_jid);
```

#### message_reactions (Real reactions)
```sql
CREATE TABLE IF NOT EXISTS message_reactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL,
    sender TEXT NOT NULL,
    sender_name TEXT,
    emoji TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id, chat_jid) REFERENCES messages(id, chat_jid),
    FOREIGN KEY (chat_jid) REFERENCES chats(jid),
    UNIQUE(message_id, chat_jid, sender, emoji)
);
CREATE INDEX idx_reactions_msg ON message_reactions(message_id, chat_jid);
CREATE INDEX idx_reactions_sender ON message_reactions(chat_jid, sender);
```

#### group_members (Group participant tracking)
```sql
CREATE TABLE IF NOT EXISTS group_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_jid TEXT NOT NULL,
    member_jid TEXT NOT NULL,
    member_name TEXT,
    is_admin BOOLEAN DEFAULT 0,
    is_owner BOOLEAN DEFAULT 0,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_jid) REFERENCES chats(jid),
    UNIQUE(group_jid, member_jid)
);
CREATE INDEX idx_group_members_group ON group_members(group_jid);
```

---

## API Response Changes

### Messages Response

**New fields added:**

```typescript
{
  // Existing
  id, chat_jid, chat_name, sender, sender_name, timestamp, timezone,
  is_group, is_read, is_from_me, content,
  media_type, media_caption, media_mime_type, media_file_path, media_file_size,

  // NEW: Sender context
  sender_contact_info: {
    jid, name, nickname, is_business, timezone
  },

  // NEW: Message content analysis
  character_count,  // LENGTH(content)
  word_count,       // (LENGTH - LENGTH(REPLACE)) / word_length
  url_list,         // array of extracted URLs

  // NEW: Threading
  mentions,                   // array of @JID mentions
  reply_to_message_id,        // if this is a reply
  quoted_message_id,          // if quoting
  quoted_sender_name,         // who said the quote
  quoted_text_preview,        // first 50 chars of quoted message

  // NEW: Message state
  is_edited,
  edit_count,
  is_forwarded,
  forwarded_from,
  is_system_message,
  system_message_type,        // "member_joined", "group_name_changed", etc

  // NEW: Engagement
  has_reactions,
  reaction_summary: {         // {emoji: count}
    "👍": 2,
    "❤️": 1
  },

  // NEW: Timing
  response_time_seconds,      // vs previous message from same sender
  is_first_message_today,
  message_position_in_thread  // 1st, 2nd, 5th in burst
}
```

**Cost:** Mostly SELECT-only (cheap). quoted_message_id requires 1 JOIN (indexed).

### Chats Response

**New fields added:**

```typescript
{
  // Existing
  jid, name, is_group, created_date, created_by, description,
  is_archived, is_pinned, is_muted, unread_count,
  last_message_content, last_message_timestamp,

  // NEW: Message activity
  total_message_count,        // denormalized
  message_count_today,        // denormalized, updated daily
  message_count_last_7_days,  // denormalized, updated daily
  message_velocity_last_7_days: {
    avg_messages_per_day,     // simple division
    total_messages,
    days_active
  },

  // NEW: Members
  participant_count,
  participant_names,          // array of names
  participant_list: [         // detailed list
    {
      name, jid, is_admin, message_count
    }
  ],
  most_active_member_name,
  most_active_member_message_count,
  admin_list: [               // from group_members table
    { name, jid, is_owner }
  ],

  // NEW: Chat state
  chat_type,                  // "dm", "group", "broadcast_list"
  silent_duration_seconds,    // time since last message
  is_recently_active,         // boolean: has_message_in_last_24h

  // NEW: Media presence
  media_count_by_type: {
    "image": 12,
    "video": 3,
    "document": 5,
    "audio": 1
  },
  recent_media: [             // last 3 media items
    {
      mime_type, file_path, size_bytes, timestamp, sender_name
    }
  ],
  has_media,                  // boolean

  // NEW: Disappearing messages
  is_disappearing_messages,
  disappearing_ttl,

  // NEW: Sender context (avoid extra lookup)
  last_sender_contact_info: {
    jid, name, nickname, is_business
  },

  // NEW: Time awareness
  timezone,                   // group timezone if known
  last_message_time_ago,      // "2 hours ago" style (optional)
}
```

**Cost:** Denormalized counts are O(1). GROUP BY queries cached/updated async.

### Contacts Response

**New fields added:**

```typescript
{
  // Existing
  jid, phone_number, name, first_name, full_name, push_name,
  business_name, nickname, verified_name,

  // NEW: Relationship
  relationship_type,          // "friend", "family", "work", "acquaintance", "unknown"
  is_favorite,
  contact_created_date,

  // NEW: Shared context
  shared_group_list: [        // groups we both in
    { jid, name, is_admin }
  ],

  // NEW: Activity metrics
  total_message_count,
  message_count_today,
  message_count_last_7_days,
  message_count_last_30_days,
  activity_trend: {           // comparing 7d vs 30d
    messages_last_7_days,
    messages_last_30_days,
    trend: "increasing" | "decreasing" | "stable"
  },

  // NEW: Response patterns
  typical_response_time_seconds,  // average delay
  typical_reply_rate,             // percentage
  is_responsive,                  // > 50% reply rate
  days_since_last_message,

  // NEW: Availability
  last_seen_timestamp,
  timezone,

  // NEW: Activity summary
  latest_message_preview,     // first line of recent msg
  latest_message_timestamp,
  has_active_chat,            // do we have recent chat
  recent_chat_jid,            // which chat to use

  // NEW: Organization
  organization,
  status_message
}
```

**Cost:** Mostly denormalized/cached except activity_trend (computed daily, cached).

---

## Testing Strategy

### Unit Tests

#### Database Tests
```go
// whatsapp-bridge/internal/database/store_test.go
func TestMessageCharacterCount(t *testing.T) {
  // Store message, retrieve with character_count
  // Verify LENGTH(content) calculation
}

func TestMessageWordCount(t *testing.T) {
  // Test word count regex on various messages
  // "hello world" = 2, "hello  world" = 2 (spaces), "hello" = 1
}

func TestQuotedMessageRetrieval(t *testing.T) {
  // Store msg A, store msg B with quoted_message_id = A's ID
  // Verify JOIN returns quoted content
}

func TestDenormalizedCounts(t *testing.T) {
  // Insert 5 messages to chat X
  // Verify total_message_count = 5
  // Test trigger maintains count on new message
}
```

#### Python Tests
```python
# whatsapp-mcp-server/tests/test_database.py
def test_list_messages_includes_sender_contact_info():
    """Verify sender_contact_info is populated"""
    messages = list_messages(chat_jid="...")
    assert messages[0]["sender_contact_info"]["name"] == "Alice"

def test_list_chats_includes_media_counts():
    """Verify media_count_by_type is accurate"""
    chats = list_chats()
    assert chats[0]["media_count_by_type"]["image"] == 12

def test_list_chats_includes_most_active_member():
    """Verify most_active_member calculation"""
    chats = list_chats()
    assert chats[0]["most_active_member_name"] is not None

def test_search_contacts_no_n_plus_one():
    """Verify nicknames loaded in single query"""
    contacts = search_contacts("alice")
    # Should be 1 DB query, not N+1
```

### Integration Tests

```bash
# End-to-end flow test
# 1. Send message with reaction
# 2. Query via API
# 3. Verify reaction_summary present
# 4. Verify sender_contact_info filled
# 5. Verify response_time_seconds calculated
```

### Performance Tests

```sql
-- Before/after query performance
-- Measure: list_messages (top 20 messages)
-- Expected: < 50ms with indexes, compare to baseline
EXPLAIN QUERY PLAN
SELECT messages.* FROM messages
WHERE chat_jid = '120363...'
ORDER BY timestamp DESC
LIMIT 20;
```

### Data Validation Tests

```python
# Verify data consistency
def test_denormalized_counts_accuracy():
    """Compare denormalized count to actual COUNT"""
    actual = db.query("SELECT COUNT(*) FROM messages WHERE chat_jid=?")
    denorm = db.query("SELECT total_message_count FROM chats WHERE jid=?")
    assert actual == denorm
```

---

## Implementation Order

### Phase 1a: Database Foundation (Day 1)
1. ✅ Create migration files
2. ✅ Run index creation script (non-blocking)
3. ✅ Create new tables (message_edits, message_reactions, group_members)
4. ✅ Add new columns to messages & chats

**Verification:** `sqlite3 store/messages.db ".schema messages"` shows new columns

### Phase 1b: Query Optimization (Day 2)
1. Fix N+1 in `search_contacts()` - JOIN contact_nicknames
2. Test query performance with indexes
3. Optimize `list_messages` with index on (chat_jid, timestamp)

**Verification:** Queries run < 100ms on 10k+ messages

### Phase 1c: API Response Updates (Day 2-3)

**Go Bridge (`internal/api/handlers.go`):**
1. Update `/api/messages` to include new fields
2. Update `/api/chats` to include new fields
3. Add `/api/contacts` endpoint (if not exists)
4. Update types in `internal/types/`

**Python MCP (`whatsapp-mcp-server/lib/database.py`):**
1. Update response builders for all entities
2. Add new fields to dataclasses (Message, Chat, Contact)
3. Update docstrings with new field descriptions

**Verification:** `curl` requests return new fields

### Phase 1d: Testing (Day 3-4)
1. Write unit tests for calculations (character_count, word_count, etc)
2. Write integration tests (send message → verify in API response)
3. Run performance benchmarks before/after indexes
4. Test with Docker (full container stack)

**Verification:** All tests pass, performance improved

### Phase 1e: Documentation & Rollout (Day 4)
1. Update CLAUDE.md with migration instructions
2. Update README.md with new fields
3. Create MIGRATION.md for existing users
4. Tag release

---

## Migration Scripts

### For Existing Users (store/messages.db)

**File:** `whatsapp-bridge/migrations/001_add_metadata_fields.sql`

Includes:
- Index creation (safe, non-blocking)
- Column additions (safe, backward-compatible)
- New table creation
- Triggers for denormalized counts

### For Python MCP

**File:** `whatsapp-mcp-server/migrations/001_add_metadata_fields.sql`

Same as Go (shared SQLite databases)

### How to Run

```bash
# Go
sqlite3 store/messages.db < migrations/001_add_metadata_fields.sql

# Docker
docker exec whatsapp-bridge sqlite3 /app/whatsapp-bridge/store/messages.db < migrations/001_add_metadata_fields.sql

# Python (if direct DB access needed)
python -m sqlite3 store/messages.db < migrations/001_add_metadata_fields.sql
```

---

## Skipped for Phase 2

- `shared_group_count` - Async compute, cache 1hr
- Complex relationship graphs
- Full-text search indexing
- Message archive optimization

---

## Success Criteria

- ✅ All 36 new fields present in API responses
- ✅ No null fields returned (only populated fields)
- ✅ Query performance improved (< 100ms for list_messages on 10k+ msgs)
- ✅ N+1 pattern fixed in search_contacts
- ✅ Test coverage > 70% for new code
- ✅ Existing users can run migrations without data loss
- ✅ Docker build works cleanly

---

## Rollback Plan

If issues arise:

1. **Before deployment:** Backup `store/messages.db`
2. **If needed:** Restore backup
3. **New columns:** Can be dropped if unused
4. **New tables:** Can be dropped safely
5. **Indexes:** Can be dropped/recreated

```sql
-- Rollback indexes
DROP INDEX idx_messages_chat_timestamp;
DROP INDEX idx_messages_sender;

-- Rollback columns (requires table rebuild in SQLite)
-- SQLite doesn't support DROP COLUMN easily; use restore approach
```
