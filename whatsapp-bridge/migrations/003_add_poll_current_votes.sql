-- Migration 003: Poll current-vote tracking
-- Target: store/messages.db
-- Date: 2026-09-23
-- Backward compatible: YES (only adds a new table)
--
-- poll_options (migration 002) resolves a vote's hashes back to option text. That alone only
-- gives you a history of vote EVENTS, one row per event -- if a voter changes their answer, or
-- retracts it, the old row is still there, so "what does this poll currently show" means finding
-- the newest row per voter yourself. This table holds exactly one row per (poll, chat, voter),
-- always overwritten to the latest vote, so "current results" is a direct, unambiguous read.
--
-- Retraction is stored explicitly as an empty selected_options array ('[]'), not row deletion --
-- "voted, currently selected nothing" and "never voted" must stay distinguishable.
--
-- vote_timestamp_ms guards against out-of-order delivery: a write only applies if its timestamp
-- is newer than what's already stored for that voter, enforced atomically by the upsert's WHERE
-- clause (see internal/database/polls.go, UpsertPollCurrentVote) -- not by the application
-- reading-then-writing, which would be a race under concurrent vote delivery.

CREATE TABLE IF NOT EXISTS poll_current_votes (
    poll_message_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL,
    voter_jid TEXT NOT NULL,
    selected_options TEXT NOT NULL,
    vote_timestamp_ms INTEGER NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (poll_message_id, chat_jid, voter_jid)
);
