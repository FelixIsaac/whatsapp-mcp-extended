-- Phase 1 Database Migration: Enhanced Metadata & Indexes
-- Target: store/messages.db
-- Date: 2026-02-14
-- Backward compatible: YES (only adds, doesn't remove)

-- ============================================================================
-- PART 1: CRITICAL INDEXES (Performance optimization)
-- ============================================================================
-- These should run first as they don't lock the table for long

CREATE INDEX IF NOT EXISTS idx_messages_chat_timestamp
  ON messages(chat_jid, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_messages_sender
  ON messages(sender);

CREATE INDEX IF NOT EXISTS idx_messages_media_type
  ON messages(chat_jid, media_type);

CREATE INDEX IF NOT EXISTS idx_chats_last_message_time
  ON chats(last_message_time DESC);

-- ============================================================================
-- PART 2: NEW COLUMNS FOR MESSAGES TABLE
-- ============================================================================
-- Message threading & editing fields

ALTER TABLE messages ADD COLUMN quoted_message_id TEXT;
-- ^-- FK to another message's ID if this message is quoting/replying

ALTER TABLE messages ADD COLUMN quoted_sender_name TEXT;
-- ^-- Name of the person who wrote the quoted message (for context)

ALTER TABLE messages ADD COLUMN reply_to_id TEXT;
-- ^-- Self-reference to another message if this is a reply (thread context)

ALTER TABLE messages ADD COLUMN edit_count INTEGER DEFAULT 0;
-- ^-- How many times this message was edited

ALTER TABLE messages ADD COLUMN is_edited BOOLEAN DEFAULT 0;
-- ^-- Flag: message has been edited

ALTER TABLE messages ADD COLUMN last_edited_at TIMESTAMP;
-- ^-- When was last edit

-- Message state & forward fields

ALTER TABLE messages ADD COLUMN is_forwarded BOOLEAN DEFAULT 0;
-- ^-- Flag: message is forwarded from another chat

ALTER TABLE messages ADD COLUMN forwarded_from TEXT;
-- ^-- JID of original sender if forwarded

ALTER TABLE messages ADD COLUMN is_system_message BOOLEAN DEFAULT 0;
-- ^-- Flag: system message (group changes, etc)

ALTER TABLE messages ADD COLUMN system_message_type TEXT;
-- ^-- Type of system message: "member_joined", "group_name_changed", "description_changed"

-- ============================================================================
-- PART 3: NEW COLUMNS FOR CHATS TABLE
-- ============================================================================
-- Denormalized counts (maintained via triggers for performance)

ALTER TABLE chats ADD COLUMN total_message_count INTEGER DEFAULT 0;
-- ^-- Total messages in this chat ever

ALTER TABLE chats ADD COLUMN message_count_today INTEGER DEFAULT 0;
-- ^-- Messages sent today (reset daily via scheduled job)

ALTER TABLE chats ADD COLUMN message_count_7days INTEGER DEFAULT 0;
-- ^-- Messages in last 7 days (updated daily)

ALTER TABLE chats ADD COLUMN last_activity TIMESTAMP;
-- ^-- Last time activity occurred (updated on each message)

-- Group metadata

ALTER TABLE chats ADD COLUMN is_group BOOLEAN DEFAULT 0;
-- ^-- Flag: is this a group chat (derived from JID suffix, but stored for perf)

ALTER TABLE chats ADD COLUMN admin_count INTEGER DEFAULT 0;
-- ^-- Number of admins in group (from group_members table)

-- ============================================================================
-- PART 4: NEW TABLES
-- ============================================================================

-- Message edits audit trail (keep history of what was edited)
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
CREATE INDEX IF NOT EXISTS idx_message_edits_msg ON message_edits(message_id, chat_jid);

-- Message reactions (who reacted with what emoji)
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
CREATE INDEX IF NOT EXISTS idx_reactions_msg ON message_reactions(message_id, chat_jid);
CREATE INDEX IF NOT EXISTS idx_reactions_sender ON message_reactions(chat_jid, sender);

-- Group members tracking (for admin_list, participant info)
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
CREATE INDEX IF NOT EXISTS idx_group_members_group ON group_members(group_jid);

-- ============================================================================
-- PART 5: TRIGGERS (Maintain denormalized counts)
-- ============================================================================

-- Trigger: Update chat's denormalized count on new message
CREATE TRIGGER IF NOT EXISTS trg_messages_insert_update_count
AFTER INSERT ON messages
FOR EACH ROW
BEGIN
    UPDATE chats
    SET total_message_count = total_message_count + 1,
        last_activity = NEW.timestamp
    WHERE jid = NEW.chat_jid;
END;

-- Trigger: Update chat's denormalized count on message delete
CREATE TRIGGER IF NOT EXISTS trg_messages_delete_update_count
AFTER DELETE ON messages
FOR EACH ROW
BEGIN
    UPDATE chats
    SET total_message_count = MAX(0, total_message_count - 1)
    WHERE jid = OLD.chat_jid;
END;

-- ============================================================================
-- PART 6: DATA CONSISTENCY (Initial population)
-- ============================================================================

-- Backfill total_message_count for existing chats
UPDATE chats
SET total_message_count = (
    SELECT COUNT(*) FROM messages WHERE messages.chat_jid = chats.jid
)
WHERE total_message_count = 0;

-- Backfill is_group flag (group JIDs end with @g.us)
UPDATE chats
SET is_group = (jid LIKE '%@g.us')
WHERE is_group = 0;

-- ============================================================================
-- Migration complete. Verify with:
-- SELECT * FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%';
-- PRAGMA table_info(messages);
-- PRAGMA table_info(chats);
-- ============================================================================
