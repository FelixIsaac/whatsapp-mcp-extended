package types

import (
	"time"
)

// Message represents a chat message for our client
type Message struct {
	Time      time.Time
	Sender    string
	Content   string
	IsFromMe  bool
	MediaType string
	Filename  string
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
