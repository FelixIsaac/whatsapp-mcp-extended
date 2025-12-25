package types

import (
	"time"
)

// Message represents a chat message for our client
type Message struct {
	Time       time.Time
	Sender     string
	SenderName string
	Content    string
	IsFromMe   bool
	MediaType  string
	Filename   string
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
