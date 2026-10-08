package whatsapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"whatsapp-bridge/internal/database"

	"go.mau.fi/whatsmeow/types/events"
)

// resolvePollVoteContent decrypts an incoming poll-vote event and renders it as readable
// message content, e.g. "[Poll Vote: Expected — dev/test, no action needed]".
//
// WhatsApp's poll protocol never transmits option text for a vote, only SHA-256 hashes of the
// selected option(s) (see whatsmeow's DecryptPollVote docs) — so this has two jobs, not one:
// decrypt the vote via the poll's message secret (which whatsmeow's SDK already persisted
// automatically when the poll was sent, nothing extra needed there), then match each selected
// hash against the original poll's option list, which this bridge must have stored itself (see
// HandleMessage's PollCreationMessage branch and CreatePoll). If either step can't complete —
// the secret isn't in the local store, or the original poll's options were never captured — this
// returns a descriptive placeholder instead of silently dropping the vote, which is what
// happened before this existed: no content, no media, message discarded with no trace anywhere.
func (c *Client) resolvePollVoteContent(messageStore *database.MessageStore, msg *events.Message) string {
	ctx := context.Background()

	vote, err := c.DecryptPollVote(ctx, msg)
	if err != nil {
		c.logger.Warnf("Failed to decrypt poll vote %s: %v", msg.Info.ID, err)
		return "[Poll Vote: could not decrypt]"
	}

	pollMsgID := msg.Message.GetPollUpdateMessage().GetPollCreationMessageKey().GetID()
	chatJID := msg.Info.Chat.String()
	options, err := messageStore.GetPollOptions(pollMsgID, chatJID)
	if err != nil {
		c.logger.Warnf("Failed to look up options for poll %s: %v", pollMsgID, err)
		return "[Poll Vote: decrypted, but couldn't look up the original poll's options]"
	}
	if len(options) == 0 {
		return "[Poll Vote: decrypted, but this bridge never saw the original poll's options]"
	}

	selected := matchSelectedOptions(options, vote.GetSelectedOptions())

	// Record this as the voter's CURRENT answer, not just a line in the message history: a
	// history row alone means "what does this poll currently show" requires finding the newest
	// row per voter yourself. This upsert keeps exactly one row per voter that's always the
	// latest, ordered by WhatsApp's own SenderTimestampMS so a late-arriving stale vote can never
	// clobber a newer one. Runs for retractions too (selected == nil) — "voted, now selected
	// nothing" is a real state, not the same as "never voted."
	voteTimestampMS := msg.Message.GetPollUpdateMessage().GetSenderTimestampMS()
	if err := messageStore.UpsertPollCurrentVote(pollMsgID, chatJID, msg.Info.Sender.String(), selected, voteTimestampMS); err != nil {
		c.logger.Warnf("Failed to record current vote for poll %s: %v", pollMsgID, err)
	}

	if len(selected) == 0 {
		// A valid, empty selection is how WhatsApp represents retracting a vote.
		return "[Poll Vote: retracted]"
	}
	return fmt.Sprintf("[Poll Vote: %s]", strings.Join(selected, ", "))
}

// matchSelectedOptions maps a poll vote's selected-option hashes back to their plaintext option
// names, by hashing each known option the same way WhatsApp does (raw SHA-256, no salt — see
// whatsmeow's HashPollOptions) and comparing. Preserves the original poll's option order rather
// than the order votes arrived in, and silently ignores any hash that doesn't match a known
// option (defensive: a stale/mismatched local option list should degrade, not error out).
func matchSelectedOptions(options []string, selectedHashes [][]byte) []string {
	var matched []string
	for _, opt := range options {
		hash := sha256.Sum256([]byte(opt))
		for _, selected := range selectedHashes {
			if bytes.Equal(hash[:], selected) {
				matched = append(matched, opt)
				break
			}
		}
	}
	return matched
}
