package types

import (
	"time"
)

// ContactInfo represents condensed contact information embedded in messages/chats
type ContactInfo struct {
	JID       string `json:"jid,omitempty"`
	Name      string `json:"name,omitempty"`
	PhoneNum  string `json:"phone_number,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
}

// MessageVelocity represents message frequency metrics
type MessageVelocity struct {
	MessagesPerDay float64 `json:"messages_per_day,omitempty"`
	TrendDirection string  `json:"trend_direction,omitempty"` // "increasing", "stable", "decreasing"
}

// ActivityTrend represents activity pattern information
type ActivityTrend struct {
	Direction      string  `json:"direction,omitempty"` // "increasing", "stable", "decreasing"
	AveragePerDay  float64 `json:"average_per_day,omitempty"`
	PeakHourRange  string  `json:"peak_hour_range,omitempty"` // "HH:MM-HH:MM" or empty
	MostActiveDay  string  `json:"most_active_day,omitempty"` // day of week or empty
}

// Message represents a chat message for our client
type Message struct {
	ID                     string       `json:"id,omitempty"`
	Time                   time.Time    `json:"timestamp,omitempty"`
	Sender                 string       `json:"sender,omitempty"`
	SenderName             string       `json:"sender_name,omitempty"`
	SenderContactInfo      *ContactInfo `json:"sender_contact_info,omitempty"`
	Content                string       `json:"content,omitempty"`
	IsFromMe               bool         `json:"is_from_me,omitempty"`
	MediaType              string       `json:"media_type,omitempty"`
	Filename               string       `json:"filename,omitempty"`
	CharacterCount         int          `json:"character_count,omitempty"`
	WordCount              int          `json:"word_count,omitempty"`
	URLList                []string     `json:"url_list,omitempty"`
	Mentions               []string     `json:"mentions,omitempty"`
	ReactionSummary        map[string]int `json:"reaction_summary,omitempty"`
	QuotedMessageID        string       `json:"quoted_message_id,omitempty"`
	QuotedSenderName       string       `json:"quoted_sender_name,omitempty"`
	ReplyToMessageID       string       `json:"reply_to_message_id,omitempty"`
	EditCount              int          `json:"edit_count,omitempty"`
	IsEdited               bool         `json:"is_edited,omitempty"`
	IsForwarded            bool         `json:"is_forwarded,omitempty"`
	ForwardedFrom          string       `json:"forwarded_from,omitempty"`
	IsSystemMessage        bool         `json:"is_system_message,omitempty"`
	SystemMessageType      string       `json:"system_message_type,omitempty"`
	ResponseTimeSeconds    float64      `json:"response_time_seconds,omitempty"`
	IsFirstMessageToday    bool         `json:"is_first_message_today,omitempty"`
	HasReactions           bool         `json:"has_reactions,omitempty"`
	MessagePositionInThread int         `json:"message_position_in_thread,omitempty"`
	IsGroup                bool         `json:"is_group,omitempty"`
	IsRead                 bool         `json:"is_read,omitempty"`
}

// Chat represents a WhatsApp chat/conversation
type Chat struct {
	JID                      string                 `json:"jid,omitempty"`
	Name                     string                 `json:"name,omitempty"`
	IsGroup                  bool                   `json:"is_group,omitempty"`
	LastMessageTime          time.Time              `json:"last_message_time,omitempty"`
	LastMessage              string                 `json:"last_message,omitempty"`
	LastMessageID            string                 `json:"last_message_id,omitempty"`
	LastSender               string                 `json:"last_sender,omitempty"`
	LastSenderName           string                 `json:"last_sender_name,omitempty"`
	LastSenderContactInfo    *ContactInfo           `json:"last_sender_contact_info,omitempty"`
	LastIsFromMe             bool                   `json:"last_is_from_me,omitempty"`
	TotalMessageCount        int                    `json:"total_message_count,omitempty"`
	MessageCountToday        int                    `json:"message_count_today,omitempty"`
	MessageCountLast7Days    int                    `json:"message_count_last_7_days,omitempty"`
	MessageVelocityLast7Days *MessageVelocity       `json:"message_velocity_last_7_days,omitempty"`
	ParticipantCount         int                    `json:"participant_count,omitempty"`
	ParticipantNames         []string               `json:"participant_names,omitempty"`
	ParticipantList          []*ContactInfo         `json:"participant_list,omitempty"`
	MostActiveMemberName     string                 `json:"most_active_member_name,omitempty"`
	MostActiveMemberMsgCount int                    `json:"most_active_member_message_count,omitempty"`
	AdminList                []string               `json:"admin_list,omitempty"`
	ChatType                 string                 `json:"chat_type,omitempty"` // "individual", "group", "broadcast"
	SilentDurationSeconds    int                    `json:"silent_duration_seconds,omitempty"`
	IsRecentlyActive         bool                   `json:"is_recently_active,omitempty"`
	MediaCountByType         map[string]int         `json:"media_count_by_type,omitempty"`
	RecentMedia              []string               `json:"recent_media,omitempty"`
	HasMedia                 bool                   `json:"has_media,omitempty"`
	IsDisappearingMessages   bool                   `json:"is_disappearing_messages,omitempty"`
	DisappearingTTL          string                 `json:"disappearing_ttl,omitempty"`
	Timezone                 string                 `json:"timezone,omitempty"`
	LastMessageTimeAgo       string                 `json:"last_message_time_ago,omitempty"`
}

// Contact represents a WhatsApp contact
type Contact struct {
	JID                      string        `json:"jid,omitempty"`
	PhoneNumber              string        `json:"phone_number,omitempty"`
	Name                     string        `json:"name,omitempty"`
	FirstName                string        `json:"first_name,omitempty"`
	FullName                 string        `json:"full_name,omitempty"`
	PushName                 string        `json:"push_name,omitempty"`
	BusinessName             string        `json:"business_name,omitempty"`
	Nickname                 string        `json:"nickname,omitempty"`
	RelationshipType         string        `json:"relationship_type,omitempty"` // "friend", "colleague", "family", "other"
	IsFavorite               bool          `json:"is_favorite,omitempty"`
	ContactCreatedDate       time.Time     `json:"contact_created_date,omitempty"`
	SharedGroupList          []*ContactInfo `json:"shared_group_list,omitempty"`
	SharedGroupCount         int           `json:"shared_group_count,omitempty"`
	TotalMessageCount        int           `json:"total_message_count,omitempty"`
	MessageCountToday        int           `json:"message_count_today,omitempty"`
	MessageCountLast7Days    int           `json:"message_count_last_7_days,omitempty"`
	MessageCountLast30Days   int           `json:"message_count_last_30_days,omitempty"`
	ActivityTrend            *ActivityTrend `json:"activity_trend,omitempty"`
	TypicalResponseTimeSeconds float64     `json:"typical_response_time_seconds,omitempty"`
	TypicalReplyRate         float64       `json:"typical_reply_rate,omitempty"`
	IsResponsive             bool          `json:"is_responsive,omitempty"`
	DaysSinceLastMessage     int           `json:"days_since_last_message,omitempty"`
	LastSeenTimestamp        time.Time     `json:"last_seen_timestamp,omitempty"`
	Timezone                 string        `json:"timezone,omitempty"`
	Organization             string        `json:"organization,omitempty"`
	StatusMessage            string        `json:"status_message,omitempty"`
	LatestMessagePreview     string        `json:"latest_message_preview,omitempty"`
	LatestMessageTimestamp   time.Time     `json:"latest_message_timestamp,omitempty"`
	HasActiveChat            bool          `json:"has_active_chat,omitempty"`
	RecentChatJID            string        `json:"recent_chat_jid,omitempty"`
}

// WebhookConfig represents a webhook configuration
type WebhookConfig struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	WebhookURL  string           `json:"webhook_url"`
	SecretToken string           `json:"secret_token"`
	Enabled     bool             `json:"enabled"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Triggers    []WebhookTrigger `json:"triggers"`
}

// WebhookConfigResponse is the API response format with masked secret
type WebhookConfigResponse struct {
	ID         int              `json:"id"`
	Name       string           `json:"name"`
	WebhookURL string           `json:"webhook_url"`
	HasSecret  bool             `json:"has_secret"`
	SecretHint string           `json:"secret_hint,omitempty"`
	Enabled    bool             `json:"enabled"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Triggers   []WebhookTrigger `json:"triggers"`
}

// MaskSecret returns a masked version of a secret token
func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

// ToResponse converts WebhookConfig to WebhookConfigResponse (masks secret)
func (c *WebhookConfig) ToResponse() WebhookConfigResponse {
	return WebhookConfigResponse{
		ID:         c.ID,
		Name:       c.Name,
		WebhookURL: c.WebhookURL,
		HasSecret:  c.SecretToken != "",
		SecretHint: MaskSecret(c.SecretToken),
		Enabled:    c.Enabled,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		Triggers:   c.Triggers,
	}
}

// WebhookTrigger represents a trigger condition for webhooks
type WebhookTrigger struct {
	ID              int    `json:"id"`
	WebhookConfigID int    `json:"webhook_config_id"`
	TriggerType     string `json:"trigger_type"` // chat_jid, sender, keyword, media_type, all
	TriggerValue    string `json:"trigger_value"`
	MatchType       string `json:"match_type"` // exact, contains, regex
	Enabled         bool   `json:"enabled"`
}

// WebhookPayload represents the standardized payload structure for webhook notifications
type WebhookPayload struct {
	EventType     string             `json:"event_type"`
	Timestamp     string             `json:"timestamp"`
	WebhookConfig WebhookConfigInfo  `json:"webhook_config"`
	Trigger       WebhookTriggerInfo `json:"trigger"`
	Message       WebhookMessageInfo `json:"message"`
	Metadata      WebhookMetadata    `json:"metadata"`
}

type WebhookConfigInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WebhookTriggerInfo struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	MatchType string `json:"match_type"`
}

type WebhookMessageInfo struct {
	ID               string `json:"id"`
	ChatJID          string `json:"chat_jid"`
	ChatName         string `json:"chat_name"`
	Sender           string `json:"sender"`
	SenderName       string `json:"sender_name"`
	Content          string `json:"content"`
	Timestamp        string `json:"timestamp"`
	PushName         string `json:"push_name,omitempty"`
	IsFromMe         bool   `json:"is_from_me"`
	MediaType        string `json:"media_type"`
	Filename         string `json:"filename"`
	MediaDownloadURL string `json:"media_download_url"`
}

type WebhookMetadata struct {
	GroupInfo        *GroupInfo `json:"group_info,omitempty"`
	DeliveryAttempt  int        `json:"delivery_attempt"`
	ProcessingTimeMs int64      `json:"processing_time_ms"`
}

type GroupInfo struct {
	IsGroup          bool   `json:"is_group"`
	GroupName        string `json:"group_name"`
	ParticipantCount int    `json:"participant_count"`
}

// WebhookLog represents a webhook delivery log entry
type WebhookLog struct {
	ID              int        `json:"id"`
	WebhookConfigID int        `json:"webhook_config_id"`
	MessageID       string     `json:"message_id"`
	ChatJID         string     `json:"chat_jid"`
	TriggerType     string     `json:"trigger_type"`
	TriggerValue    string     `json:"trigger_value"`
	Payload         string     `json:"payload"`
	ResponseStatus  int        `json:"response_status"`
	ResponseBody    string     `json:"response_body"`
	AttemptCount    int        `json:"attempt_count"`
	DeliveredAt     *time.Time `json:"delivered_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SendMessageRequest represents the request body for the send message API
type SendMessageRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	MediaPath string `json:"media_path,omitempty"`
}

// SendMessageResponse represents the response for the send message API
type SendMessageResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message,omitempty"`
	MessageID string    `json:"message_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Recipient string    `json:"recipient,omitempty"`
}

// SendResult contains the result of sending a message (internal use)
type SendResult struct {
	Success   bool
	Error     string
	MessageID string
	Timestamp time.Time
}

// ReactionRequest represents the request body for sending reactions
type ReactionRequest struct {
	ChatJID   string `json:"chat_jid"`
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"` // empty string to remove reaction
}

// EditMessageRequest represents the request body for editing messages
type EditMessageRequest struct {
	ChatJID    string `json:"chat_jid"`
	MessageID  string `json:"message_id"`
	NewContent string `json:"new_content"`
}

// DeleteMessageRequest represents the request body for deleting/revoking messages
type DeleteMessageRequest struct {
	ChatJID   string `json:"chat_jid"`
	MessageID string `json:"message_id"`
	SenderJID string `json:"sender_jid,omitempty"` // for admin revoking others' msgs
}

// MarkReadRequest represents the request body for marking messages as read
type MarkReadRequest struct {
	ChatJID    string   `json:"chat_jid"`
	MessageIDs []string `json:"message_ids"`
	SenderJID  string   `json:"sender_jid,omitempty"` // required for group chats
}

// Phase 2: Group Management

// CreateGroupRequest represents the request body for creating a group
type CreateGroupRequest struct {
	Name         string   `json:"name"`
	Participants []string `json:"participants"` // JIDs of participants to add
}

// GroupParticipantsRequest represents the request body for adding/removing group members
type GroupParticipantsRequest struct {
	GroupJID     string   `json:"group_jid"`
	Participants []string `json:"participants"` // JIDs of participants
}

// GroupAdminRequest represents the request body for promoting/demoting admins
type GroupAdminRequest struct {
	GroupJID    string `json:"group_jid"`
	Participant string `json:"participant"` // JID of participant
}

// LeaveGroupRequest represents the request body for leaving a group
type LeaveGroupRequest struct {
	GroupJID string `json:"group_jid"`
}

// UpdateGroupRequest represents the request body for updating group settings
type UpdateGroupRequest struct {
	GroupJID string `json:"group_jid"`
	Name     string `json:"name,omitempty"`
	Topic    string `json:"topic,omitempty"`
}

// Phase 3: Polls

// CreatePollRequest represents the request body for creating a poll
type CreatePollRequest struct {
	ChatJID     string   `json:"chat_jid"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	MultiSelect bool     `json:"multi_select"`
}

// Phase 4: History Sync

// RequestHistoryRequest represents the request body for on-demand history request
type RequestHistoryRequest struct {
	ChatJID            string `json:"chat_jid"`
	OldestMsgID        string `json:"oldest_msg_id"`
	OldestMsgFromMe    bool   `json:"oldest_msg_from_me"`
	OldestMsgTimestamp int64  `json:"oldest_msg_timestamp"` // Unix milliseconds
	Count              int    `json:"count"`                // Max 50
}

// Phase 5: Advanced Features

// SetPresenceRequest represents request to set own presence
type SetPresenceRequest struct {
	Presence string `json:"presence"` // "available" or "unavailable"
}

// SubscribePresenceRequest represents request to subscribe to a contact's presence
type SubscribePresenceRequest struct {
	JID string `json:"jid"`
}

// PresenceInfo represents presence information for a contact
type PresenceInfo struct {
	JID          string `json:"jid"`
	Available    bool   `json:"available"`
	LastSeen     string `json:"last_seen,omitempty"` // ISO-8601 or empty
	LastSeenUnix int64  `json:"last_seen_unix,omitempty"`
}

// GetProfilePictureRequest represents request to get profile picture
type GetProfilePictureRequest struct {
	JID     string `json:"jid"`
	Preview bool   `json:"preview,omitempty"`
}

// ProfilePictureInfo represents profile picture information
type ProfilePictureInfo struct {
	URL        string `json:"url"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	DirectPath string `json:"direct_path,omitempty"`
}

// BlocklistRequest represents request to block/unblock a user
type BlocklistRequest struct {
	JID    string `json:"jid"`
	Action string `json:"action"` // "block" or "unblock"
}

// BlockedUser represents a blocked user
type BlockedUser struct {
	JID string `json:"jid"`
}

// BlocklistResponse represents the blocklist response
type BlocklistResponse struct {
	Success bool          `json:"success"`
	Users   []BlockedUser `json:"users"`
}

// NewsletterRequest represents request to follow/unfollow a newsletter
type NewsletterRequest struct {
	JID string `json:"jid"`
}

// CreateNewsletterRequest represents request to create a newsletter
type CreateNewsletterRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NewsletterInfo represents newsletter metadata
type NewsletterInfo struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Phase 6: Chat Features

// SendTypingRequest represents request to send typing indicator
type SendTypingRequest struct {
	ChatJID string `json:"chat_jid"`
	State   string `json:"state"` // "typing", "paused", or "recording"
}

// SetAboutRequest represents request to set profile about/status text
type SetAboutRequest struct {
	Text string `json:"text"`
}

// SetDisappearingTimerRequest represents request to set disappearing messages timer
type SetDisappearingTimerRequest struct {
	ChatJID  string `json:"chat_jid"`
	Duration string `json:"duration"` // "off", "24h", "7d", "90d"
}

// PinChatRequest represents request to pin or unpin a chat
type PinChatRequest struct {
	ChatJID string `json:"chat_jid"`
	Pin     bool   `json:"pin"` // true to pin, false to unpin
}

// MuteChatRequest represents request to mute or unmute a chat
type MuteChatRequest struct {
	ChatJID  string `json:"chat_jid"`
	Mute     bool   `json:"mute"`           // true to mute, false to unmute
	Duration string `json:"duration"`      // "forever", "15m", "1h", "8h", "1w" (ignored if mute=false)
}

// ArchiveChatRequest represents request to archive or unarchive a chat
type ArchiveChatRequest struct {
	ChatJID string `json:"chat_jid"`
	Archive bool   `json:"archive"` // true to archive, false to unarchive
}

// Phase 7: Phone Number Pairing

// PairPhoneRequest initiates phone number pairing
type PairPhoneRequest struct {
	PhoneNumber string `json:"phone_number"` // Format: country code + number, no leading zeros (e.g., "1234567890")
}

// PairPhoneResponse returns pairing code
type PairPhoneResponse struct {
	Success   bool   `json:"success"`
	Code      string `json:"code,omitempty"`       // 8-digit pairing code
	ExpiresIn int    `json:"expires_in,omitempty"` // Seconds until expiration (160)
	Error     string `json:"error,omitempty"`
}

// PairingStatusResponse returns current pairing state
type PairingStatusResponse struct {
	Success    bool   `json:"success"`
	InProgress bool   `json:"in_progress"`
	Code       string `json:"code,omitempty"`
	ExpiresIn  int    `json:"expires_in,omitempty"` // Remaining seconds
	Complete   bool   `json:"complete"`
	Error      string `json:"error,omitempty"`
}

// ConnectionStatusResponse returns WhatsApp connection state
type ConnectionStatusResponse struct {
	Success             bool   `json:"success"`
	Connected           bool   `json:"connected"`
	Linked              bool   `json:"linked"`                         // Device has valid session
	JID                 string `json:"jid,omitempty"`                  // WhatsApp ID if linked
	Uptime              string `json:"uptime,omitempty"`               // Process uptime
	LastConnected       string `json:"last_connected,omitempty"`       // ISO-8601 timestamp
	DisconnectedFor     string `json:"disconnected_for,omitempty"`     // Duration string
	AutoReconnectErrors int    `json:"auto_reconnect_errors,omitempty"`
}

// SyncStatusResponse returns current message sync state
type SyncStatusResponse struct {
	Success       bool   `json:"success"`
	Syncing       bool   `json:"syncing"`
	LastSync      string `json:"last_sync,omitempty"`
	SyncProgress  int    `json:"sync_progress"`        // 0-100 percent
	MessageCount  int    `json:"message_count"`
	ConversationCount int `json:"conversation_count"`
	Error         string `json:"error,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}
