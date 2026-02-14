-- Phase 1 Database Migration: Enhanced Metadata & Indexes
-- Target: ../whatsapp-bridge/store/messages.db (shared SQLite database)
-- This is the same migration as whatsapp-bridge/migrations/001_add_metadata_fields.sql
-- It's mirrored here for Python layer reference

-- ============================================================================
-- PART 1: CRITICAL INDEXES (Performance optimization)
-- ============================================================================

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

ALTER TABLE messages ADD COLUMN quoted_message_id TEXT;
ALTER TABLE messages ADD COLUMN quoted_sender_name TEXT;
ALTER TABLE messages ADD COLUMN reply_to_id TEXT;
ALTER TABLE messages ADD COLUMN edit_count INTEGER DEFAULT 0;
ALTER TABLE messages ADD COLUMN is_edited BOOLEAN DEFAULT 0;
ALTER TABLE messages ADD COLUMN last_edited_at TIMESTAMP;
ALTER TABLE messages ADD COLUMN is_forwarded BOOLEAN DEFAULT 0;
ALTER TABLE messages ADD COLUMN forwarded_from TEXT;
ALTER TABLE messages ADD COLUMN is_system_message BOOLEAN DEFAULT 0;
ALTER TABLE messages ADD COLUMN system_message_type TEXT;

-- ============================================================================
-- PART 3: NEW COLUMNS FOR CHATS TABLE
-- ============================================================================

ALTER TABLE chats ADD COLUMN total_message_count INTEGER DEFAULT 0;
ALTER TABLE chats ADD COLUMN message_count_today INTEGER DEFAULT 0;
ALTER TABLE chats ADD COLUMN message_count_7days INTEGER DEFAULT 0;
ALTER TABLE chats ADD COLUMN last_activity TIMESTAMP;
ALTER TABLE chats ADD COLUMN is_group BOOLEAN DEFAULT 0;
ALTER TABLE chats ADD COLUMN admin_count INTEGER DEFAULT 0;

-- ============================================================================
-- PART 4: NEW TABLES
-- ============================================================================

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
-- PART 5: TRIGGERS
-- ============================================================================

CREATE TRIGGER IF NOT EXISTS trg_messages_insert_update_count
AFTER INSERT ON messages
FOR EACH ROW
BEGIN
    UPDATE chats
    SET total_message_count = total_message_count + 1,
        last_activity = NEW.timestamp
    WHERE jid = NEW.chat_jid;
END;

CREATE TRIGGER IF NOT EXISTS trg_messages_delete_update_count
AFTER DELETE ON messages
FOR EACH ROW
BEGIN
    UPDATE chats
    SET total_message_count = MAX(0, total_message_count - 1)
    WHERE jid = OLD.chat_jid;
END;

-- ============================================================================
-- PART 6: DATA CONSISTENCY
-- ============================================================================

UPDATE chats
SET total_message_count = (
    SELECT COUNT(*) FROM messages WHERE messages.chat_jid = chats.jid
)
WHERE total_message_count = 0;

UPDATE chats
SET is_group = (jid LIKE '%@g.us')
WHERE is_group = 0;
