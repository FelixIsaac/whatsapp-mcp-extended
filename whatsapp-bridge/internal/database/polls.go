package database

import "encoding/json"

// StorePollOptions persists the option list of a poll creation message, keyed by the poll
// message's own ID and chat JID. This is required to resolve incoming poll votes: WhatsApp
// only ever sends SHA-256 hashes of the selected option text, never the option text itself,
// so decrypting a vote is only half the job — the original option names have to be known
// locally to turn a hash back into something readable.
func (store *MessageStore) StorePollOptions(messageID, chatJID string, options []string) error {
	tx, err := store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM poll_options WHERE message_id = ? AND chat_jid = ?`, messageID, chatJID); err != nil {
		return err
	}
	for i, opt := range options {
		if _, err := tx.Exec(
			`INSERT INTO poll_options (message_id, chat_jid, option_index, option_name) VALUES (?, ?, ?, ?)`,
			messageID, chatJID, i, opt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetPollOptions returns the option names of a previously stored poll, in their original
// order. Returns an empty slice (not an error) if the poll wasn't created/seen by this bridge
// — e.g. a poll from before this feature existed, or a poll from a chat we never captured.
func (store *MessageStore) GetPollOptions(messageID, chatJID string) ([]string, error) {
	rows, err := store.db.Query(
		`SELECT option_name FROM poll_options WHERE message_id = ? AND chat_jid = ? ORDER BY option_index ASC`,
		messageID, chatJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []string
	for rows.Next() {
		var opt string
		if err := rows.Scan(&opt); err != nil {
			return nil, err
		}
		options = append(options, opt)
	}
	return options, rows.Err()
}

// UpsertPollCurrentVote records a voter's CURRENT selection on a poll, overwriting whatever was
// there before for that voter — not appending. A vote event only ever carries a voter's complete
// current selection (WhatsApp never sends incremental deltas), so overwrite is the correct
// semantics, and it's what makes "what does this poll currently show" a single direct read
// instead of "find the newest history row per voter yourself."
//
// selectedOptions may be empty, which is how a retracted vote is represented — that's a real,
// distinct state ("voted, currently selected nothing"), not the same as "never voted" (no row).
//
// voteTimestampMS is WhatsApp's own SenderTimestampMS for the vote event. The write only applies
// if it's newer than whatever timestamp is already stored for this voter — enforced by the SQL
// itself (the WHERE clause on the upsert), not by the caller reading-then-deciding-then-writing,
// which would be a race if two vote events for the same voter arrive close together.
func (store *MessageStore) UpsertPollCurrentVote(pollMessageID, chatJID, voterJID string, selectedOptions []string, voteTimestampMS int64) error {
	if selectedOptions == nil {
		selectedOptions = []string{}
	}
	encoded, err := json.Marshal(selectedOptions)
	if err != nil {
		return err
	}
	_, err = store.db.Exec(
		`INSERT INTO poll_current_votes (poll_message_id, chat_jid, voter_jid, selected_options, vote_timestamp_ms, updated_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (poll_message_id, chat_jid, voter_jid) DO UPDATE SET
		   selected_options = excluded.selected_options,
		   vote_timestamp_ms = excluded.vote_timestamp_ms,
		   updated_at = CURRENT_TIMESTAMP
		 WHERE excluded.vote_timestamp_ms > poll_current_votes.vote_timestamp_ms`,
		pollMessageID, chatJID, voterJID, string(encoded), voteTimestampMS,
	)
	return err
}

// PollVote is one voter's current, resolved selection on a poll.
type PollVote struct {
	VoterJID        string
	SelectedOptions []string
	VoteTimestampMS int64
}

// GetPollCurrentVotes returns every voter's current selection on a poll — the live tally, one
// entry per voter who has ever voted (including voters who later retracted, shown with an empty
// SelectedOptions). Returns an empty slice, not an error, if nobody has voted yet.
func (store *MessageStore) GetPollCurrentVotes(pollMessageID, chatJID string) ([]PollVote, error) {
	rows, err := store.db.Query(
		`SELECT voter_jid, selected_options, vote_timestamp_ms FROM poll_current_votes
		 WHERE poll_message_id = ? AND chat_jid = ? ORDER BY voter_jid ASC`,
		pollMessageID, chatJID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []PollVote
	for rows.Next() {
		var v PollVote
		var encoded string
		if err := rows.Scan(&v.VoterJID, &encoded, &v.VoteTimestampMS); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(encoded), &v.SelectedOptions); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}
