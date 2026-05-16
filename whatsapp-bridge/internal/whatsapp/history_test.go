package whatsapp

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	waWeb "go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func TestLatestHistoryMessageTimeUsesMaxTimestamp(t *testing.T) {
	older := uint64(1_768_000_000)
	newer := older + 3600

	got, ok := latestHistoryMessageTime([]*waHistorySync.HistorySyncMsg{
		historyMsg("old", older, false),
		nil,
		historyMsg("new", newer, false),
	})
	if !ok {
		t.Fatal("latestHistoryMessageTime() ok = false, want true")
	}

	want := time.Unix(int64(newer), 0)
	if !got.Equal(want) {
		t.Fatalf("latestHistoryMessageTime() = %v, want %v", got, want)
	}
}

func TestHistoryMessageSenderUsesFullOwnJID(t *testing.T) {
	own := types.JID{User: "111", Server: "s.whatsapp.net"}
	client := &Client{
		Client: &whatsmeow.Client{Store: &store.Device{ID: &own}},
	}
	chat := types.JID{User: "222", Server: "s.whatsapp.net"}

	sender, fromMe := client.historyMessageSender(chat, &waCommon.MessageKey{FromMe: proto.Bool(true)})
	if !fromMe {
		t.Fatal("fromMe = false, want true")
	}
	if sender != "111@s.whatsapp.net" {
		t.Fatalf("sender = %q, want full own JID", sender)
	}
}

func TestHistoryMessageSenderPrefersParticipant(t *testing.T) {
	client := &Client{}
	chat := types.JID{User: "12345", Server: "g.us"}

	sender, fromMe := client.historyMessageSender(chat, &waCommon.MessageKey{
		FromMe:      proto.Bool(false),
		Participant: proto.String("333@s.whatsapp.net"),
	})
	if fromMe {
		t.Fatal("fromMe = true, want false")
	}
	if sender != "333@s.whatsapp.net" {
		t.Fatalf("sender = %q, want participant JID", sender)
	}
}

func historyMsg(id string, timestamp uint64, fromMe bool) *waHistorySync.HistorySyncMsg {
	return &waHistorySync.HistorySyncMsg{
		Message: &waWeb.WebMessageInfo{
			Key: &waCommon.MessageKey{
				ID:     proto.String(id),
				FromMe: proto.Bool(fromMe),
			},
			MessageTimestamp: proto.Uint64(timestamp),
			Message: &waE2E.Message{
				Conversation: proto.String(id),
			},
		},
	}
}
