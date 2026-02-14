# MCP Server Metadata Philosophy

## Core Principle

> **Provide raw facts and verifiable metrics. Let the consuming LLM infer, interpret, and decide.**

The WhatsApp MCP server prioritizes **complete information** with **minimal interpretation**, allowing AI models to make context-aware decisions without repeated queries.

---

## Three Design Rules

### 1. Include Raw Data, Not Derived Signals

**What we provide:**
- Raw message content (with emojis as typed)
- Parsed mentions as structured data
- Precise MIME types and file metadata
- Exact timestamps and creation dates
- Participant names and contact lists

**What we don't:**
- ~~`has_emoji`~~ (include actual emojis in content)
- ~~`is_question`~~ (LLM reads the "?")
- ~~`is_command_like`~~ (LLM infers from "do X", "check Y")
- ~~`sentiment_indicator`~~ (LLM reads tone)
- ~~`all_caps_portion`~~ (LLM sees the text)

**Why:** The consuming LLM is exceptionally good at inference. Server should be a data source, not an interpreter. Pre-computing signals wastes the LLM's token budget on labels it could derive itself.

---

### 2. Include Countable Metrics, Not Subjective Categories

**What we provide:**
- `message_count_last_7_days: 47`
- `typical_response_time_seconds: 300`
- `avg_message_length: 85`
- `participant_count: 5`
- `disappearing_ttl: 86400` (seconds)

**What we don't:**
- ~~`communication_frequency: "frequent"`~~ (ambiguous threshold)
- ~~`emoji_usage_frequency: "always"`~~ (subjective)
- ~~`group_personality: "casual"`~~ (opinion-based)

**Why:** Numbers are verifiable and explainable. Categories are arbitrary. An LLM receiving `message_count_last_7_days: 47` can reason: "That's ~6.7 msgs/day = moderately active." But telling it `"frequent"` requires it to guess your threshold.

---

### 3. Exclude Null/Empty Fields to Reduce Token Waste

**What we send:**
```json
{
  "jid": "123@s.whatsapp.net",
  "name": "Alice",
  "timezone": "UTC-5"
}
```

**What we don't send:**
```json
{
  "jid": "123@s.whatsapp.net",
  "name": "Alice",
  "timezone": "UTC-5",
  "organization": null,
  "email": null,
  "status_message": null
}
```

**Why:**
- Missing field = "we don't have it" (not null)
- LLM doesn't assume missing fields have values
- Smaller JSON payloads = faster responses
- Saves tokens for fields that actually have data

---

## When We *Do* Pre-Compute

We include computed metadata only when:

1. **It's cheaper for the server than the client**
   - Example: `unread_count` (one DB query vs LLM reading every message)

2. **It's unavailable to the LLM**
   - Example: `response_time_seconds` (LLM doesn't have message history for comparison)
   - Example: `last_seen_timestamp` (requires WhatsApp presence data)

3. **It's a pure fact, not interpretation**
   - Example: `created_date`, `participant_count`, `media_mime_type`
   - Example: `has_reactions` (boolean fact, not a category)

---

## Response Structure by Entity

### Messages
```json
{
  "id": "3EB028A580CF7CC9AAF3A2",
  "chat_jid": "120363123456789012@g.us",
  "chat_name": "Project Team",
  "sender": "1234567890@s.whatsapp.net",
  "sender_name": "Alice",
  "sender_contact_info": {
    "jid": "1234567890@s.whatsapp.net",
    "name": "Alice",
    "nickname": "Alice Dev",
    "is_business": false
  },
  "content": "Check out the new feature 🎉",
  "mentions": ["1111111111@s.whatsapp.net"],
  "timestamp": "2026-02-14T03:30:00Z",
  "timezone": "UTC-5",
  "is_group": true,
  "is_read": true,
  "is_edited": false,
  "reply_to_message_id": "3EB027A580CF7CC9AAF3A1",
  "has_reactions": true,
  "media_type": "image",
  "media_caption": "New UI mockup",
  "media_mime_type": "image/jpeg",
  "media_file_path": "/app/media/image123.jpg",
  "media_file_size": 245000,
  "response_time_seconds": 120,
  "is_first_message_today": false
}
```

### Chats
```json
{
  "jid": "120363123456789012@g.us",
  "name": "Project Team",
  "is_group": true,
  "participant_count": 5,
  "participant_names": ["Alice", "Bob", "Charlie"],
  "description": "Q1 2026 roadmap planning",
  "created_date": "2025-11-15",
  "created_by": "Alice",
  "is_archived": false,
  "is_pinned": true,
  "is_muted": false,
  "unread_count": 2,
  "is_disappearing_messages": true,
  "disappearing_ttl": 86400,
  "has_media": true,
  "last_message_content": "Meeting tomorrow at 9am",
  "last_message_timestamp": "2026-02-14T03:30:00Z",
  "last_sender_name": "Bob",
  "last_is_from_me": false,
  "timezone": "UTC-5"
}
```

### Contacts
```json
{
  "jid": "1234567890@s.whatsapp.net",
  "phone_number": "+1234567890",
  "name": "Alice",
  "first_name": "Alice",
  "full_name": "Alice Johnson",
  "nickname": "Alice Dev",
  "verified_name": "Alice Inc.",
  "timezone": "UTC-5",
  "organization": "Acme Corp",
  "status_message": "Currently in Singapore",
  "is_business": true,
  "is_blocked": false,
  "last_seen_timestamp": "2026-02-14T02:15:00Z",
  "has_active_chat": true,
  "recent_chat_jid": "1234567890@s.whatsapp.net",
  "avg_message_length": 95,
  "typical_response_time_seconds": 180,
  "message_count_last_7_days": 34
}
```

---

## How This Reduces Token Waste

### Traditional Approach ❌
```
LLM: "Who is sender 1234567890?"
Server: {id, jid}
LLM: "Need more info, can you get their name?"
Server: {id, jid, name}
LLM: "Is this person in contacts?"
Server: {id, jid, name, contact_info}
```
**Result:** 3 queries, 3 round-trips, wasted tokens

### Our Approach ✅
```
LLM: "Get recent messages"
Server: [{id, jid, sender_name, sender_contact_info, content, ...}]
LLM: "I have everything I need - Alice sent this to a group of 5, no need to ask more"
```
**Result:** 1 query, complete context, efficient

---

## What LLM Provides

The consuming LLM handles:
- **Reasoning**: "Why did they respond late? Last_seen_timestamp shows they were offline"
- **Interpretation**: "This message uses casual tone (emojis, short)"
- **Context**: "Same sender responded in 2min previously, now 30min - maybe urgent?"
- **Inference**: "3 people didn't respond = they may be offline"
- **Decisions**: "Should I ask for more context or work with what I have?"

---

## Documentation for API Users

When building on this API:

1. **Don't assume fields exist** - Check for presence, don't check for null
2. **Use raw numbers for logic** - If `message_count_last_7_days: 2`, you define what "quiet" means
3. **Combine signals** - `response_time_seconds` + `last_seen_timestamp` + `timezone` → understand availability
4. **Leverage contact info** - `sender_contact_info` included to avoid extra lookups
5. **Thread context is in content** - `mentions` and `reply_to_message_id` let you reconstruct threads

---

## Future: Computed Metadata vs Raw Data

We intentionally *don't* include:
- `inferred_topic` (requires NLP model)
- `conversation_sentiment` (requires ML)
- `likely_response_urgency` (too subjective)

**If needed in future:**
- Compute these **client-side** on the consuming LLM
- Or document the computation so LLMs can reason about it
- Never use a second LLM to interpret data for another LLM

---

## References

- **Philosophy inspired by:** [REST API best practices](https://restfulapi.net/), cursor-based pagination patterns, [JSON:API specification](https://jsonapi.org/)
- **Token optimization:** OpenAI token counting, context window management
- **Database design:** Normalization principles (avoid redundant derivations)
