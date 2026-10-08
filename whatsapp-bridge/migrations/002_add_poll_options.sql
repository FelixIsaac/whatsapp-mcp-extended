-- Migration 002: Poll vote decryption support
-- Target: store/messages.db
-- Date: 2026-09-23
-- Backward compatible: YES (only adds a new table)
--
-- WhatsApp poll votes only ever transmit SHA-256 hashes of the selected option text, never the
-- text itself. Decrypting a vote (see internal/whatsapp/handlers.go, resolvePollVoteContent) is
-- only half the job — the original poll's option names must be known locally to turn a hash back
-- into a readable answer. This table is where CreatePoll and HandleMessage persist that mapping.

CREATE TABLE IF NOT EXISTS poll_options (
    message_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL,
    option_index INTEGER NOT NULL,
    option_name TEXT NOT NULL,
    PRIMARY KEY (message_id, chat_jid, option_index)
);
