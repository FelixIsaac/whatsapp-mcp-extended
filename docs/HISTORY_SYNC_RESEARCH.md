# WhatsApp History Sync Research

> Research conducted: 2025-12-24

## Summary

**Goal**: Get 600+ days of WhatsApp message history via whatsmeow
**Reality**: Cannot reliably get 600+ days - limited by WhatsApp's architecture

---

## How WhatsApp History Sync Works

### Architecture
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Your Phone    │────►│  WhatsApp       │────►│  Linked Device  │
│  (Source of     │     │  Servers        │     │  (whatsmeow)    │
│   Truth)        │     │  (Relay Only)   │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

- **WhatsApp servers do NOT store messages** (E2E encrypted)
- **Your phone is the source of truth** for message history
- Linked devices receive history **from your phone**, not servers
- Initial sync happens at device link time

### Sync Types

| Type | When | Amount |
|------|------|--------|
| Initial Full Sync | Device link | ~90 days to 1 year |
| On-Demand Sync | Per-chat request | 50 messages at a time |
| Real-time Sync | Ongoing | All new messages |

---

## HistorySyncConfig Protobuf

From [wa-proto](https://github.com/wppconnect-team/wa-proto/blob/main/WAProto.proto):

```protobuf
message HistorySyncConfig {
    uint32 fullSyncDaysLimit = 1;           // Max days for full sync
    uint32 fullSyncSizeMbLimit = 2;         // Max size in MB
    uint32 storageQuotaMb = 3;              // Storage allocation
    bool inlineInitialPayloadInE2EeMsg = 4;
    uint32 recentSyncDaysLimit = 5;         // Recent sync limit
    bool supportCallLogHistory = 6;
    bool supportBotUserAgentChatHistory = 7;
    bool supportCagReactionsAndPolls = 8;
    bool supportBizHostedMsg = 9;
    bool supportRecentSyncChunkMessageCountTuning = 10;
    bool supportHostedGroupMsg = 11;
    bool supportFbidBotChatHistory = 12;
    bool supportAddOnHistorySyncMigration = 13;
    bool supportMessageAssociation = 14;
    bool supportGroupHistory = 15;
    bool onDemandReady = 16;
    bool supportGuestChat = 17;
}
```

### Current whatsmeow Defaults

From [clientpayload.go](https://github.com/tulir/whatsmeow/blob/main/store/clientpayload.go):

```go
HistorySyncConfig: &waCompanionReg.DeviceProps_HistorySyncConfig{
    StorageQuotaMb:                 proto.Uint32(10240),  // 10GB
    InlineInitialPayloadInE2EeMsg: proto.Bool(true),
    SupportCallLogHistory:          proto.Bool(false),
    SupportBotUserAgentChatHistory: proto.Bool(true),
    SupportCagReactionsAndPolls:    proto.Bool(true),
    SupportGroupHistory:            proto.Bool(false),
}
```

**Note**: `fullSyncDaysLimit` is NOT explicitly set by default.

---

## Known Limitations

### Hard Limits

| Limitation | Description |
|------------|-------------|
| Phone dependency | Phone must have messages; if deleted, they're gone |
| One-time initial sync | Config only applies at device link time |
| Re-link required | Must unlink and re-link to change sync config |
| WhatsApp control | WhatsApp can change server-side limits anytime |

### Observed Behavior

| Scenario | Typical Result |
|----------|----------------|
| Fresh device link | 90 days - 1 year of history |
| Known bug | Some devices stuck at 90 days |
| On-demand request | Works but tedious for many chats |
| Phone offline 14+ days | Linked device disconnected |

---

## On-Demand History Request

whatsmeow supports requesting older messages per-chat:

```go
// Build request for 50 messages older than lastKnownMsg
msg := client.BuildHistorySyncRequest(lastKnownMessageInfo, 50)

// Send with Peer flag
resp, err := client.SendMessage(ctx, chat, msg, whatsmeow.SendRequestExtra{Peer: true})

// Response comes as events.HistorySync with type ON_DEMAND
```

### Limitations
- Requires `lastKnownMessageInfo` (need existing message reference)
- Phone must be online and have the messages
- Recommended: 50 messages per request
- Response is asynchronous via event handler

---

## Recommendations

### For Maximum History at Link Time

```go
// Set BEFORE creating device/linking
store.DeviceProps.HistorySyncConfig = &waCompanionReg.DeviceProps_HistorySyncConfig{
    FullSyncDaysLimit:   proto.Uint32(730),   // 2 years
    FullSyncSizeMbLimit: proto.Uint32(5000),  // 5GB
    StorageQuotaMb:      proto.Uint32(10240), // 10GB
    // ... other fields
}
```

**Caveat**: Phone ultimately decides what to send.

### For Long-Term History

1. **Link device early** - Start capturing messages ASAP
2. **Store locally** - Current MCP server stores all received messages
3. **Keep phone connected** - Phone must stay online
4. **Use on-demand sparingly** - For specific older chats when needed

---

## Implementation Plan

### Phase 4A: Configurable HistorySyncConfig
- Add environment variables for sync config
- Set config before device creation
- Document that re-link is required

### Phase 4B: On-Demand History Request
- Add MCP tool: `request_history(chat_jid, count)`
- Store oldest known message per chat
- Handle async response via event handler

---

## Sources

- [WhatsApp Help: Message History on Linked Devices](https://faq.whatsapp.com/653480766448040)
- [whatsmeow GitHub](https://github.com/tulir/whatsmeow)
- [whatsmeow clientpayload.go](https://github.com/tulir/whatsmeow/blob/main/store/clientpayload.go)
- [wa-proto WAProto.proto](https://github.com/wppconnect-team/wa-proto/blob/main/WAProto.proto)
- [whatsmeow Go Package Docs](https://pkg.go.dev/go.mau.fi/whatsmeow)
- [whatsmeow Issue #654](https://github.com/tulir/whatsmeow/issues/654)
