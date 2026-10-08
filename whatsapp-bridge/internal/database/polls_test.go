package database

import (
	"reflect"
	"testing"
)

func TestStoreAndGetPollOptions(t *testing.T) {
	tests := []struct {
		name      string
		messageID string
		chatJID   string
		options   []string
	}{
		{
			name:      "typical poll",
			messageID: "3EB0POLL1",
			chatJID:   "918668575753@s.whatsapp.net",
			options:   []string{"Expected — dev/test, no action needed", "Not expected — please escalate/fix", "Need to discuss first"},
		},
		{
			name:      "single option",
			messageID: "3EB0POLL2",
			chatJID:   "918668575753@s.whatsapp.net",
			options:   []string{"Yes"},
		},
		{
			name:      "empty options",
			messageID: "3EB0POLL3",
			chatJID:   "918668575753@s.whatsapp.net",
			options:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestMessageStore(t)

			if err := store.StorePollOptions(tt.messageID, tt.chatJID, tt.options); err != nil {
				t.Fatalf("StorePollOptions: %v", err)
			}

			got, err := store.GetPollOptions(tt.messageID, tt.chatJID)
			if err != nil {
				t.Fatalf("GetPollOptions: %v", err)
			}
			if len(got) != len(tt.options) {
				t.Fatalf("got %d options, want %d (%v vs %v)", len(got), len(tt.options), got, tt.options)
			}
			for i := range tt.options {
				if got[i] != tt.options[i] {
					t.Errorf("option[%d] = %q, want %q", i, got[i], tt.options[i])
				}
			}
		})
	}
}

func TestStorePollOptionsOverwritesPreviousSet(t *testing.T) {
	store := newTestMessageStore(t)
	const msgID, chatJID = "3EB0POLL", "918668575753@s.whatsapp.net"

	if err := store.StorePollOptions(msgID, chatJID, []string{"A", "B", "C"}); err != nil {
		t.Fatalf("first StorePollOptions: %v", err)
	}
	if err := store.StorePollOptions(msgID, chatJID, []string{"X", "Y"}); err != nil {
		t.Fatalf("second StorePollOptions: %v", err)
	}

	got, err := store.GetPollOptions(msgID, chatJID)
	if err != nil {
		t.Fatalf("GetPollOptions: %v", err)
	}
	want := []string{"X", "Y"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("option[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGetPollOptionsUnknownPollReturnsEmptyNotError(t *testing.T) {
	store := newTestMessageStore(t)

	got, err := store.GetPollOptions("never-seen-this-poll", "918668575753@s.whatsapp.net")
	if err != nil {
		t.Fatalf("GetPollOptions on unknown poll should not error, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no options for an unknown poll, got %v", got)
	}
}

func TestPollOptionsAreScopedByChat(t *testing.T) {
	store := newTestMessageStore(t)
	const msgID = "3EB0POLL"

	if err := store.StorePollOptions(msgID, "chatA@s.whatsapp.net", []string{"A1", "A2"}); err != nil {
		t.Fatalf("store chatA: %v", err)
	}
	if err := store.StorePollOptions(msgID, "chatB@s.whatsapp.net", []string{"B1"}); err != nil {
		t.Fatalf("store chatB: %v", err)
	}

	gotA, err := store.GetPollOptions(msgID, "chatA@s.whatsapp.net")
	if err != nil {
		t.Fatalf("GetPollOptions chatA: %v", err)
	}
	if len(gotA) != 2 || gotA[0] != "A1" || gotA[1] != "A2" {
		t.Errorf("chatA options = %v, want [A1 A2]", gotA)
	}

	gotB, err := store.GetPollOptions(msgID, "chatB@s.whatsapp.net")
	if err != nil {
		t.Fatalf("GetPollOptions chatB: %v", err)
	}
	if len(gotB) != 1 || gotB[0] != "B1" {
		t.Errorf("chatB options = %v, want [B1]", gotB)
	}
}

func TestUpsertPollCurrentVoteCreatesAndOverwrites(t *testing.T) {
	store := newTestMessageStore(t)
	const pollID, chatJID, voter = "POLL1", "chat@s.whatsapp.net", "voter@s.whatsapp.net"

	if err := store.UpsertPollCurrentVote(pollID, chatJID, voter, []string{"A"}, 1000); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	votes, err := store.GetPollCurrentVotes(pollID, chatJID)
	if err != nil {
		t.Fatalf("GetPollCurrentVotes: %v", err)
	}
	if len(votes) != 1 || !reflect.DeepEqual(votes[0].SelectedOptions, []string{"A"}) {
		t.Fatalf("after first vote, got %+v, want one vote for [A]", votes)
	}

	// A newer vote from the same voter overwrites — doesn't add a second row.
	if err := store.UpsertPollCurrentVote(pollID, chatJID, voter, []string{"B"}, 2000); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	votes, err = store.GetPollCurrentVotes(pollID, chatJID)
	if err != nil {
		t.Fatalf("GetPollCurrentVotes after change: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected still exactly 1 row after a vote change, got %d", len(votes))
	}
	if !reflect.DeepEqual(votes[0].SelectedOptions, []string{"B"}) {
		t.Errorf("selected options = %v, want [B]", votes[0].SelectedOptions)
	}
}

func TestUpsertPollCurrentVoteRejectsOutOfOrderDelivery(t *testing.T) {
	store := newTestMessageStore(t)
	const pollID, chatJID, voter = "POLL1", "chat@s.whatsapp.net", "voter@s.whatsapp.net"

	if err := store.UpsertPollCurrentVote(pollID, chatJID, voter, []string{"Newer"}, 5000); err != nil {
		t.Fatalf("newer vote: %v", err)
	}
	// A vote with an OLDER timestamp arriving after (e.g. redelivered, or a network reorder)
	// must not overwrite the newer one — enforced atomically by the SQL itself, not by the
	// caller checking first, which would be a race under concurrent delivery.
	if err := store.UpsertPollCurrentVote(pollID, chatJID, voter, []string{"OlderShouldBeIgnored"}, 1000); err != nil {
		t.Fatalf("older vote (should be silently ignored, not error): %v", err)
	}

	votes, err := store.GetPollCurrentVotes(pollID, chatJID)
	if err != nil {
		t.Fatalf("GetPollCurrentVotes: %v", err)
	}
	if len(votes) != 1 || !reflect.DeepEqual(votes[0].SelectedOptions, []string{"Newer"}) {
		t.Fatalf("expected the newer vote to survive, got %+v", votes)
	}
	if votes[0].VoteTimestampMS != 5000 {
		t.Errorf("vote_timestamp_ms = %d, want 5000 (unchanged)", votes[0].VoteTimestampMS)
	}
}

func TestUpsertPollCurrentVoteRetractionIsExplicitNotDeletion(t *testing.T) {
	store := newTestMessageStore(t)
	const pollID, chatJID, voter = "POLL1", "chat@s.whatsapp.net", "voter@s.whatsapp.net"

	if err := store.UpsertPollCurrentVote(pollID, chatJID, voter, []string{"A"}, 1000); err != nil {
		t.Fatalf("initial vote: %v", err)
	}
	if err := store.UpsertPollCurrentVote(pollID, chatJID, voter, nil, 2000); err != nil {
		t.Fatalf("retraction: %v", err)
	}

	votes, err := store.GetPollCurrentVotes(pollID, chatJID)
	if err != nil {
		t.Fatalf("GetPollCurrentVotes: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected the voter's row to still exist after retraction (not deleted), got %d rows", len(votes))
	}
	if len(votes[0].SelectedOptions) != 0 {
		t.Errorf("selected options after retraction = %v, want empty", votes[0].SelectedOptions)
	}
}

func TestPollCurrentVotesIsolatedPerVoter(t *testing.T) {
	store := newTestMessageStore(t)
	const pollID, chatJID = "POLL1", "group@g.us"

	if err := store.UpsertPollCurrentVote(pollID, chatJID, "alice@s.whatsapp.net", []string{"A"}, 1000); err != nil {
		t.Fatalf("alice's vote: %v", err)
	}
	if err := store.UpsertPollCurrentVote(pollID, chatJID, "bob@s.whatsapp.net", []string{"B"}, 1000); err != nil {
		t.Fatalf("bob's vote: %v", err)
	}
	// Bob changes his mind; must not touch Alice's row.
	if err := store.UpsertPollCurrentVote(pollID, chatJID, "bob@s.whatsapp.net", []string{"C"}, 2000); err != nil {
		t.Fatalf("bob's second vote: %v", err)
	}

	votes, err := store.GetPollCurrentVotes(pollID, chatJID)
	if err != nil {
		t.Fatalf("GetPollCurrentVotes: %v", err)
	}
	if len(votes) != 2 {
		t.Fatalf("expected 2 independent voters, got %d (%+v)", len(votes), votes)
	}
	byVoter := map[string][]string{}
	for _, v := range votes {
		byVoter[v.VoterJID] = v.SelectedOptions
	}
	if !reflect.DeepEqual(byVoter["alice@s.whatsapp.net"], []string{"A"}) {
		t.Errorf("alice's vote = %v, want [A] (should be untouched by bob's changes)", byVoter["alice@s.whatsapp.net"])
	}
	if !reflect.DeepEqual(byVoter["bob@s.whatsapp.net"], []string{"C"}) {
		t.Errorf("bob's vote = %v, want [C]", byVoter["bob@s.whatsapp.net"])
	}
}

func TestGetPollCurrentVotesUnknownPollReturnsEmptyNotError(t *testing.T) {
	store := newTestMessageStore(t)

	votes, err := store.GetPollCurrentVotes("never-seen", "chat@s.whatsapp.net")
	if err != nil {
		t.Fatalf("should not error on an unknown poll, got: %v", err)
	}
	if len(votes) != 0 {
		t.Fatalf("expected no votes for an unknown poll, got %v", votes)
	}
}
