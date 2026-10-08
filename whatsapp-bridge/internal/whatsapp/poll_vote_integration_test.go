package whatsapp

import (
	"context"
	"testing"
	"time"

	"whatsapp-bridge/internal/database"

	"go.mau.fi/whatsmeow"
	waAdv "go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// TestHandleMessagePollVoteEndToEnd drives HandleMessage exactly as it runs in production — a
// poll creation event, then vote events, both decrypted for real through a real whatsmeow client
// and store — rather than unit-testing resolvePollVoteContent or the database layer in isolation.
//
// This is deliberate, not extra caution: the original bug (votes silently dropped) was invisible
// to unit tests of ExtractTextContent alone, because it only manifested in HandleMessage's
// empty-content-and-no-media check, which nothing isolated ever exercised. The same class of gap
// could hide here too — e.g. resolvePollVoteContent calling UpsertPollCurrentVote with the wrong
// JID or timestamp would pass every unit test of UpsertPollCurrentVote on its own. Only driving
// the real chain catches that.
func TestHandleMessagePollVoteEndToEnd(t *testing.T) {
	t.Chdir(t.TempDir())

	messageStore, err := database.NewMessageStore()
	if err != nil {
		t.Fatalf("NewMessageStore: %v", err)
	}

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+t.TempDir()+"/test.db?_foreign_keys=on", waLog.Noop)
	if err != nil {
		t.Fatalf("sqlstore.New: %v", err)
	}
	device := container.NewDevice()
	ownJID := types.JID{User: "111111", Server: types.DefaultUserServer}
	device.ID = &ownJID
	// NewDevice() leaves device.Account nil (only populated by a real pairing handshake, which
	// this test never does) — but PutDevice dereferences it unconditionally, so a bare NewDevice()
	// SIGSEGVs there. A zero-value identity is enough to satisfy that; found by adding a probe
	// test that logged every field's nil-ness after NewDevice() rather than guessing at the crash.
	// The schema enforces the real signature/key lengths (64-byte signatures, 32-byte keys) even
	// though this test never performs an actual pairing handshake — dummy-but-correctly-sized
	// values satisfy the CHECK constraints without needing to run real crypto here.
	device.Account = &waAdv.ADVSignedDeviceIdentity{
		Details:             []byte{},
		AccountSignature:    make([]byte, 64),
		AccountSignatureKey: make([]byte, 32),
		DeviceSignature:     make([]byte, 64),
	}
	if err := container.PutDevice(ctx, device); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	rawClient := whatsmeow.NewClient(device, waLog.Noop)
	c := &Client{Client: rawClient, logger: waLog.Noop}

	// A second, genuinely separate device for the voter. This isn't extra caution — a single
	// shared client can't correctly simulate two real people: BuildPollVote signs the vote using
	// the CALLING client's own device identity (cli.getOwnID()), so if the bridge's own client
	// were used to build "the voter's" vote too, the vote would be cryptographically signed as
	// the bridge itself, not the voter — decrypt would then compute a different key than encrypt
	// did and fail authentication, exactly as it did on the first attempt at this test (found
	// empirically, not anticipated up front).
	voterJID := types.JID{User: "333333", Server: types.DefaultUserServer}
	voterDevice := container.NewDevice()
	voterDevice.ID = &voterJID
	voterDevice.Account = &waAdv.ADVSignedDeviceIdentity{
		Details:             []byte{},
		AccountSignature:    make([]byte, 64),
		AccountSignatureKey: make([]byte, 32),
		DeviceSignature:     make([]byte, 64),
	}
	if err := container.PutDevice(ctx, voterDevice); err != nil {
		t.Fatalf("PutDevice (voter): %v", err)
	}
	voterClient := whatsmeow.NewClient(voterDevice, waLog.Noop)

	// In a 1:1 chat, the "chat" JID from either side IS the counterparty — there's no separate
	// chat identity the way a group has one. From the bridge's (ownJID's) own perspective, the
	// chat with the voter is simply the voter's JID.
	chat := voterJID
	const pollMsgID = "POLL1"

	// Real poll creation message, real random message secret — exactly what CreatePoll builds.
	pollCreation := rawClient.BuildPollCreation("Continue?", []string{"Yes", "No", "Maybe"}, 1)
	secret := pollCreation.GetMessageContextInfo().GetMessageSecret()

	// This device created the poll, so — same as CreatePoll's own comment explains — it never
	// sees its own poll come back as an incoming event, but whatsmeow's send path DOES store the
	// message secret automatically (verified by reading send.go directly, not assumed). Simulate
	// that effect here, on EACH device's own store, matching its own (chat, sender) perspective
	// of the same 1:1 conversation — the bridge's view is chat=voter; the voter's view is
	// chat=sender=ownJID, since for an incoming 1:1 message the chat IS the sender. Both stores
	// end up holding the same underlying secret bytes, exactly as two real separate devices would.
	if err := rawClient.Store.MsgSecrets.PutMessageSecret(ctx, chat, ownJID, pollMsgID, secret); err != nil {
		t.Fatalf("seed message secret (bridge's own store): %v", err)
	}
	if err := voterClient.Store.MsgSecrets.PutMessageSecret(ctx, ownJID, ownJID, pollMsgID, secret); err != nil {
		t.Fatalf("seed message secret (voter's own store): %v", err)
	}

	pollCreateEvt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: ownJID, IsFromMe: true},
			ID:            pollMsgID,
		},
		Message: pollCreation,
	}
	c.HandleMessage(messageStore, nil, pollCreateEvt)

	storedOptions, err := messageStore.GetPollOptions(pollMsgID, chat.String())
	if err != nil {
		t.Fatalf("GetPollOptions: %v", err)
	}
	if len(storedOptions) != 3 {
		t.Fatalf("expected 3 options stored via HandleMessage's real code path, got %v", storedOptions)
	}

	// This is the poll AS THE VOTER WOULD BUILD A VOTE AGAINST IT: from their side, chat=sender=
	// ownJID (the poll creator), and IsFromMe is false because they didn't create it. Confirmed
	// against whatsmeow's actual key-resolution logic (msgsecret.go, getOrigSenderFromKey) rather
	// than assumed — getting Chat/IsFromMe wrong here is exactly what silently breaks decryption
	// without any type error to catch it, since everything still compiles and runs, it just
	// resolves to the wrong original sender.
	pollInfo := &types.MessageInfo{
		MessageSource: types.MessageSource{Chat: ownJID, Sender: ownJID, IsFromMe: false},
		ID:            pollMsgID,
	}

	// A real, genuinely encrypted vote from the voter, decrypted through the real client.
	voteMsg, err := voterClient.BuildPollVote(ctx, pollInfo, []string{"No"})
	if err != nil {
		t.Fatalf("BuildPollVote: %v", err)
	}
	voteEvt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: voterJID, IsFromMe: false},
			ID:            "VOTE1",
		},
		Message: voteMsg,
	}
	c.HandleMessage(messageStore, nil, voteEvt)

	votes, err := messageStore.GetPollCurrentVotes(pollMsgID, chat.String())
	if err != nil {
		t.Fatalf("GetPollCurrentVotes: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected 1 voter recorded, got %d (%+v)", len(votes), votes)
	}
	if votes[0].VoterJID != voterJID.String() {
		t.Errorf("voter JID = %q, want %q", votes[0].VoterJID, voterJID.String())
	}
	if len(votes[0].SelectedOptions) != 1 || votes[0].SelectedOptions[0] != "No" {
		t.Errorf("selected options = %v, want [No]", votes[0].SelectedOptions)
	}

	// Guarantee strict timestamp progression before the second vote — BuildPollVote timestamps
	// itself off time.Now(), and two calls back-to-back could otherwise land in the same
	// millisecond, which would make the upsert's out-of-order guard (correctly) reject the
	// change and turn this into a flaky test rather than a real assertion.
	time.Sleep(2 * time.Millisecond)

	// Same voter changes their answer. Proves both "overwrite, don't append" AND that this is
	// actually wired into HandleMessage, not just correct in isolation.
	voteMsg2, err := voterClient.BuildPollVote(ctx, pollInfo, []string{"Yes"})
	if err != nil {
		t.Fatalf("BuildPollVote (2nd): %v", err)
	}
	voteEvt2 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: voterJID, IsFromMe: false},
			ID:            "VOTE2",
		},
		Message: voteMsg2,
	}
	c.HandleMessage(messageStore, nil, voteEvt2)

	votes, err = messageStore.GetPollCurrentVotes(pollMsgID, chat.String())
	if err != nil {
		t.Fatalf("GetPollCurrentVotes after change: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected still exactly 1 voter row after a vote change, got %d (%+v)", len(votes), votes)
	}
	if len(votes[0].SelectedOptions) != 1 || votes[0].SelectedOptions[0] != "Yes" {
		t.Errorf("after change, selected options = %v, want [Yes]", votes[0].SelectedOptions)
	}

	// Retraction: an empty selection is a real, distinct state, not row deletion.
	time.Sleep(2 * time.Millisecond)
	voteMsg3, err := voterClient.BuildPollVote(ctx, pollInfo, nil)
	if err != nil {
		t.Fatalf("BuildPollVote (retraction): %v", err)
	}
	voteEvt3 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: voterJID, IsFromMe: false},
			ID:            "VOTE3",
		},
		Message: voteMsg3,
	}
	c.HandleMessage(messageStore, nil, voteEvt3)

	votes, err = messageStore.GetPollCurrentVotes(pollMsgID, chat.String())
	if err != nil {
		t.Fatalf("GetPollCurrentVotes after retraction: %v", err)
	}
	if len(votes) != 1 {
		t.Fatalf("expected the voter's row to still exist after retraction (not deleted), got %d rows", len(votes))
	}
	if len(votes[0].SelectedOptions) != 0 {
		t.Errorf("after retraction, selected options = %v, want empty", votes[0].SelectedOptions)
	}
}
