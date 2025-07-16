package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
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
	EventType     string                 `json:"event_type"`
	Timestamp     string                 `json:"timestamp"`
	WebhookConfig WebhookConfigInfo      `json:"webhook_config"`
	Trigger       WebhookTriggerInfo     `json:"trigger"`
	Message       WebhookMessageInfo     `json:"message"`
	Metadata      WebhookMetadata        `json:"metadata"`
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
	IsGroup          bool `json:"is_group"`
	GroupName        string `json:"group_name"`
	ParticipantCount int  `json:"participant_count"`
}

// WebhookLog represents a webhook delivery log entry
type WebhookLog struct {
	ID              int       `json:"id"`
	WebhookConfigID int       `json:"webhook_config_id"`
	MessageID       string    `json:"message_id"`
	ChatJID         string    `json:"chat_jid"`
	TriggerType     string    `json:"trigger_type"`
	TriggerValue    string    `json:"trigger_value"`
	Payload         string    `json:"payload"`
	ResponseStatus  int       `json:"response_status"`
	ResponseBody    string    `json:"response_body"`
	AttemptCount    int       `json:"attempt_count"`
	DeliveredAt     *time.Time `json:"delivered_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// Database handler for storing message history
type MessageStore struct {
	db *sql.DB
}

// Initialize message store
func NewMessageStore() (*MessageStore, error) {
	// Create directory for database if it doesn't exist
	if err := os.MkdirAll("store", 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	// Open SQLite database for messages
	db, err := sql.Open("sqlite3", "file:store/messages.db?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open message database: %v", err)
	}

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			jid TEXT PRIMARY KEY,
			name TEXT,
			last_message_time TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT,
			chat_jid TEXT,
			sender TEXT,
			content TEXT,
			timestamp TIMESTAMP,
			is_from_me BOOLEAN,
			media_type TEXT,
			filename TEXT,
			url TEXT,
			media_key BLOB,
			file_sha256 BLOB,
			file_enc_sha256 BLOB,
			file_length INTEGER,
			PRIMARY KEY (id, chat_jid),
			FOREIGN KEY (chat_jid) REFERENCES chats(jid)
		);

		CREATE TABLE IF NOT EXISTS webhook_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			webhook_url TEXT NOT NULL,
			secret_token TEXT,
			enabled BOOLEAN DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS webhook_triggers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			webhook_config_id INTEGER REFERENCES webhook_configs(id),
			trigger_type TEXT NOT NULL,
			trigger_value TEXT,
			match_type TEXT DEFAULT 'exact',
			enabled BOOLEAN DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS webhook_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			webhook_config_id INTEGER REFERENCES webhook_configs(id),
			message_id TEXT,
			chat_jid TEXT,
			trigger_type TEXT,
			trigger_value TEXT,
			payload TEXT,
			response_status INTEGER,
			response_body TEXT,
			attempt_count INTEGER DEFAULT 1,
			delivered_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &MessageStore{db: db}, nil
}

// Close the database connection
func (store *MessageStore) Close() error {
	return store.db.Close()
}

// Store a chat in the database
func (store *MessageStore) StoreChat(jid, name string, lastMessageTime time.Time) error {
	_, err := store.db.Exec(
		"INSERT OR REPLACE INTO chats (jid, name, last_message_time) VALUES (?, ?, ?)",
		jid, name, lastMessageTime,
	)
	return err
}

// Store a message in the database
func (store *MessageStore) StoreMessage(id, chatJID, sender, content string, timestamp time.Time, isFromMe bool,
	mediaType, filename, url string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error {
	// Only store if there's actual content or media
	if content == "" && mediaType == "" {
		return nil
	}

	_, err := store.db.Exec(
		`INSERT OR REPLACE INTO messages 
		(id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, url, media_key, file_sha256, file_enc_sha256, file_length) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, chatJID, sender, content, timestamp, isFromMe, mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength,
	)
	return err
}

// Get messages from a chat
func (store *MessageStore) GetMessages(chatJID string, limit int) ([]Message, error) {
	rows, err := store.db.Query(
		"SELECT sender, content, timestamp, is_from_me, media_type, filename FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC LIMIT ?",
		chatJID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var timestamp time.Time
		err := rows.Scan(&msg.Sender, &msg.Content, &timestamp, &msg.IsFromMe, &msg.MediaType, &msg.Filename)
		if err != nil {
			return nil, err
		}
		msg.Time = timestamp
		messages = append(messages, msg)
	}

	return messages, nil
}

// Get all chats
func (store *MessageStore) GetChats() (map[string]time.Time, error) {
	rows, err := store.db.Query("SELECT jid, last_message_time FROM chats ORDER BY last_message_time DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := make(map[string]time.Time)
	for rows.Next() {
		var jid string
		var lastMessageTime time.Time
		err := rows.Scan(&jid, &lastMessageTime)
		if err != nil {
			return nil, err
		}
		chats[jid] = lastMessageTime
	}

	return chats, nil
}

// Webhook management methods

// StoreWebhookConfig stores a webhook configuration in the database
func (store *MessageStore) StoreWebhookConfig(config *WebhookConfig) error {
	result, err := store.db.Exec(
		`INSERT INTO webhook_configs (name, webhook_url, secret_token, enabled) 
		 VALUES (?, ?, ?, ?)`,
		config.Name, config.WebhookURL, config.SecretToken, config.Enabled,
	)
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	config.ID = int(id)
	
	// Store triggers
	for i := range config.Triggers {
		config.Triggers[i].WebhookConfigID = config.ID
		err = store.StoreWebhookTrigger(&config.Triggers[i])
		if err != nil {
			return err
		}
	}
	
	return nil
}

// GetWebhookConfig retrieves a webhook configuration by ID
func (store *MessageStore) GetWebhookConfig(id int) (*WebhookConfig, error) {
	config := &WebhookConfig{}
	err := store.db.QueryRow(
		`SELECT id, name, webhook_url, secret_token, enabled, created_at, updated_at 
		 FROM webhook_configs WHERE id = ?`, id,
	).Scan(&config.ID, &config.Name, &config.WebhookURL, &config.SecretToken, 
		&config.Enabled, &config.CreatedAt, &config.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	// Load triggers
	config.Triggers, err = store.GetWebhookTriggers(id)
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// GetAllWebhookConfigs retrieves all webhook configurations
func (store *MessageStore) GetAllWebhookConfigs() ([]*WebhookConfig, error) {
	rows, err := store.db.Query(
		`SELECT id, name, webhook_url, secret_token, enabled, created_at, updated_at 
		 FROM webhook_configs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var configs []*WebhookConfig
	for rows.Next() {
		config := &WebhookConfig{}
		err := rows.Scan(&config.ID, &config.Name, &config.WebhookURL, &config.SecretToken,
			&config.Enabled, &config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, err
		}
		
		// Load triggers for each config
		config.Triggers, err = store.GetWebhookTriggers(config.ID)
		if err != nil {
			return nil, err
		}
		
		configs = append(configs, config)
	}
	
	return configs, nil
}

// UpdateWebhookConfig updates a webhook configuration
func (store *MessageStore) UpdateWebhookConfig(config *WebhookConfig) error {
	_, err := store.db.Exec(
		`UPDATE webhook_configs SET name = ?, webhook_url = ?, secret_token = ?, 
		 enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		config.Name, config.WebhookURL, config.SecretToken, config.Enabled, config.ID,
	)
	return err
}

// DeleteWebhookConfig deletes a webhook configuration and its triggers
func (store *MessageStore) DeleteWebhookConfig(id int) error {
	// First check if the webhook exists
	var count int
	err := store.db.QueryRow("SELECT COUNT(*) FROM webhook_configs WHERE id = ?", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("webhook with ID %d not found", id)
	}
	
	// Delete triggers first (foreign key constraint)
	_, err = store.db.Exec("DELETE FROM webhook_triggers WHERE webhook_config_id = ?", id)
	if err != nil {
		return err
	}
	
	// Delete config
	_, err = store.db.Exec("DELETE FROM webhook_configs WHERE id = ?", id)
	return err
}

// StoreWebhookTrigger stores a webhook trigger
func (store *MessageStore) StoreWebhookTrigger(trigger *WebhookTrigger) error {
	result, err := store.db.Exec(
		`INSERT INTO webhook_triggers (webhook_config_id, trigger_type, trigger_value, match_type, enabled) 
		 VALUES (?, ?, ?, ?, ?)`,
		trigger.WebhookConfigID, trigger.TriggerType, trigger.TriggerValue, trigger.MatchType, trigger.Enabled,
	)
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	trigger.ID = int(id)
	
	return nil
}

// GetWebhookTriggers retrieves all triggers for a webhook config
func (store *MessageStore) GetWebhookTriggers(webhookConfigID int) ([]WebhookTrigger, error) {
	rows, err := store.db.Query(
		`SELECT id, webhook_config_id, trigger_type, trigger_value, match_type, enabled 
		 FROM webhook_triggers WHERE webhook_config_id = ?`, webhookConfigID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var triggers []WebhookTrigger
	for rows.Next() {
		trigger := WebhookTrigger{}
		err := rows.Scan(&trigger.ID, &trigger.WebhookConfigID, &trigger.TriggerType,
			&trigger.TriggerValue, &trigger.MatchType, &trigger.Enabled)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, trigger)
	}
	
	return triggers, nil
}

// DeleteWebhookTrigger deletes a webhook trigger
func (store *MessageStore) DeleteWebhookTrigger(id int) error {
	_, err := store.db.Exec("DELETE FROM webhook_triggers WHERE id = ?", id)
	return err
}

// StoreWebhookLog stores a webhook delivery log
func (store *MessageStore) StoreWebhookLog(log *WebhookLog) error {
	_, err := store.db.Exec(
		`INSERT INTO webhook_logs (webhook_config_id, message_id, chat_jid, trigger_type, trigger_value, 
		 payload, response_status, response_body, attempt_count, delivered_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.WebhookConfigID, log.MessageID, log.ChatJID, log.TriggerType, log.TriggerValue,
		log.Payload, log.ResponseStatus, log.ResponseBody, log.AttemptCount, log.DeliveredAt,
	)
	return err
}

// GetWebhookLogs retrieves webhook logs with optional filtering
func (store *MessageStore) GetWebhookLogs(webhookConfigID int, limit int) ([]*WebhookLog, error) {
	query := `SELECT id, webhook_config_id, message_id, chat_jid, trigger_type, trigger_value, 
		 payload, response_status, response_body, attempt_count, delivered_at, created_at 
		 FROM webhook_logs`
	
	var args []interface{}
	if webhookConfigID > 0 {
		query += " WHERE webhook_config_id = ?"
		args = append(args, webhookConfigID)
	}
	
	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	
	rows, err := store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []*WebhookLog
	for rows.Next() {
		log := &WebhookLog{}
		err := rows.Scan(&log.ID, &log.WebhookConfigID, &log.MessageID, &log.ChatJID,
			&log.TriggerType, &log.TriggerValue, &log.Payload, &log.ResponseStatus,
			&log.ResponseBody, &log.AttemptCount, &log.DeliveredAt, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	
	return logs, nil
}

// WebhookManager handles webhook processing and delivery
type WebhookManager struct {
	messageStore *MessageStore
	logger       waLog.Logger
	httpClient   *http.Client
	configs      []*WebhookConfig
	mutex        sync.RWMutex
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(messageStore *MessageStore, logger waLog.Logger) *WebhookManager {
	return &WebhookManager{
		messageStore: messageStore,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		configs: make([]*WebhookConfig, 0),
	}
}

// LoadWebhookConfigs loads webhook configurations from database
func (wm *WebhookManager) LoadWebhookConfigs() error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	
	configs, err := wm.messageStore.GetAllWebhookConfigs()
	if err != nil {
		return fmt.Errorf("failed to load webhook configs: %v", err)
	}
	
	wm.configs = configs
	wm.logger.Infof("Loaded %d webhook configurations", len(configs))
	return nil
}

// GetWebhookConfigs returns a copy of current webhook configurations
func (wm *WebhookManager) GetWebhookConfigs() []*WebhookConfig {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	
	// Return a copy to avoid race conditions
	configs := make([]*WebhookConfig, len(wm.configs))
	copy(configs, wm.configs)
	return configs
}

// MatchesTriggers checks if a message matches any webhook triggers
func (wm *WebhookManager) MatchesTriggers(msg *events.Message, chatName string) []*WebhookConfig {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	
	var matchedConfigs []*WebhookConfig
	
	// Extract message content
	content := extractTextContent(msg.Message)
	mediaType, _, _, _, _, _, _ := extractMediaInfo(msg.Message)
	
	for _, config := range wm.configs {
		if !config.Enabled {
			continue
		}
		
		matched := false
		for _, trigger := range config.Triggers {
			if !trigger.Enabled {
				continue
			}
			
			if wm.matchesTrigger(trigger, msg, content, mediaType, chatName) {
				matched = true
				break
			}
		}
		
		if matched {
			matchedConfigs = append(matchedConfigs, config)
		}
	}
	
	return matchedConfigs
}

// matchesTrigger checks if a single trigger matches the message
func (wm *WebhookManager) matchesTrigger(trigger WebhookTrigger, msg *events.Message, content, mediaType, chatName string) bool {
	switch trigger.TriggerType {
	case "all":
		return true
		
	case "chat_jid":
		return wm.matchesString(msg.Info.Chat.String(), trigger.TriggerValue, trigger.MatchType)
		
	case "sender":
		senderJID := msg.Info.Sender.String()
		senderUser := msg.Info.Sender.User
		return wm.matchesString(senderJID, trigger.TriggerValue, trigger.MatchType) ||
			   wm.matchesString(senderUser, trigger.TriggerValue, trigger.MatchType)
		
	case "keyword":
		return wm.matchesString(content, trigger.TriggerValue, trigger.MatchType)
		
	case "media_type":
		return wm.matchesString(mediaType, trigger.TriggerValue, trigger.MatchType)
		
	default:
		wm.logger.Warnf("Unknown trigger type: %s", trigger.TriggerType)
		return false
	}
}

// matchesString performs string matching based on match type
func (wm *WebhookManager) matchesString(text, pattern, matchType string) bool {
	switch matchType {
	case "exact":
		return text == pattern
		
	case "contains":
		return strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
		
	case "regex":
		matched, err := regexp.MatchString(pattern, text)
		if err != nil {
			wm.logger.Warnf("Invalid regex pattern '%s': %v", pattern, err)
			return false
		}
		return matched
		
	default:
		wm.logger.Warnf("Unknown match type: %s", matchType)
		return false
	}
}

// ProcessMessage processes a message and sends webhooks if triggers match
func (wm *WebhookManager) ProcessMessage(client *whatsmeow.Client, msg *events.Message, chatName string) {
	startTime := time.Now()
	
	// Find matching webhook configurations
	matchedConfigs := wm.MatchesTriggers(msg, chatName)
	if len(matchedConfigs) == 0 {
		return
	}
	
	wm.logger.Infof("Found %d matching webhook configs for message %s", len(matchedConfigs), msg.Info.ID)
	
	// Extract message content and media info
	content := extractTextContent(msg.Message)
	mediaType, filename, _, _, _, _, _ := extractMediaInfo(msg.Message)
	
	// Determine sender name
	senderName := msg.Info.Sender.User
	if contact, err := client.Store.Contacts.GetContact(context.Background(), msg.Info.Sender); err == nil && contact.FullName != "" {
		senderName = contact.FullName
	}
	
	// Build base payload
	basePayload := WebhookPayload{
		EventType: "message_received",
		Timestamp: msg.Info.Timestamp.Format(time.RFC3339),
		Message: WebhookMessageInfo{
			ID:        msg.Info.ID,
			ChatJID:   msg.Info.Chat.String(),
			ChatName:  chatName,
			Sender:    msg.Info.Sender.User,
			SenderName: senderName,
			Content:   content,
			Timestamp: msg.Info.Timestamp.Format(time.RFC3339),
			IsFromMe:  msg.Info.IsFromMe,
			MediaType: mediaType,
			Filename:  filename,
		},
		Metadata: WebhookMetadata{
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		},
	}
	
	// Add media download URL if it's a media message
	if mediaType != "" {
		basePayload.Message.MediaDownloadURL = fmt.Sprintf("http://localhost:8080/api/download")
	}
	
	// Add group info if it's a group chat
	if msg.Info.Chat.Server == "g.us" {
		basePayload.Metadata.GroupInfo = &GroupInfo{
			IsGroup:   true,
			GroupName: chatName,
			// ParticipantCount would require additional API call
		}
	}
	
	// Send webhooks for each matched configuration
	for _, config := range matchedConfigs {
		// Find the specific trigger that matched
		var matchedTrigger *WebhookTrigger
		content := extractTextContent(msg.Message)
		mediaType, _, _, _, _, _, _ := extractMediaInfo(msg.Message)
		
		for _, trigger := range config.Triggers {
			if trigger.Enabled && wm.matchesTrigger(trigger, msg, content, mediaType, chatName) {
				matchedTrigger = &trigger
				break
			}
		}
		
		if matchedTrigger == nil {
			continue
		}
		
		// Customize payload for this webhook
		payload := basePayload
		payload.WebhookConfig = WebhookConfigInfo{
			ID:   config.ID,
			Name: config.Name,
		}
		payload.Trigger = WebhookTriggerInfo{
			Type:      matchedTrigger.TriggerType,
			Value:     matchedTrigger.TriggerValue,
			MatchType: matchedTrigger.MatchType,
		}
		payload.Metadata.DeliveryAttempt = 1
		
		// Send webhook asynchronously
		go wm.deliverWebhook(config, &payload, msg.Info.ID, msg.Info.Chat.String(), matchedTrigger)
	}
}

// deliverWebhook delivers a webhook with retry logic
func (wm *WebhookManager) deliverWebhook(config *WebhookConfig, payload *WebhookPayload, messageID, chatJID string, trigger *WebhookTrigger) {
	maxRetries := 5
	backoffIntervals := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		wm.logger.Errorf("Failed to marshal webhook payload: %v", err)
		return
	}
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		payload.Metadata.DeliveryAttempt = attempt
		
		// Update payload with current attempt
		payloadBytes, _ = json.Marshal(payload)
		
		success, statusCode, responseBody := wm.sendHTTPRequest(config, payloadBytes)
		
		// Log the delivery attempt
		log := &WebhookLog{
			WebhookConfigID: config.ID,
			MessageID:       messageID,
			ChatJID:         chatJID,
			TriggerType:     trigger.TriggerType,
			TriggerValue:    trigger.TriggerValue,
			Payload:         string(payloadBytes),
			ResponseStatus:  statusCode,
			ResponseBody:    responseBody,
			AttemptCount:    attempt,
		}
		
		if success {
			now := time.Now()
			log.DeliveredAt = &now
			wm.logger.Infof("Webhook delivered successfully to %s (attempt %d)", config.WebhookURL, attempt)
		} else {
			wm.logger.Warnf("Webhook delivery failed to %s (attempt %d): status %d", config.WebhookURL, attempt, statusCode)
		}
		
		// Store log
		if err := wm.messageStore.StoreWebhookLog(log); err != nil {
			wm.logger.Errorf("Failed to store webhook log: %v", err)
		}
		
		if success {
			return // Success, no need to retry
		}
		
		// Wait before retry (except for last attempt)
		if attempt < maxRetries {
			time.Sleep(backoffIntervals[attempt-1])
		}
	}
	
	wm.logger.Errorf("Webhook delivery failed permanently to %s after %d attempts", config.WebhookURL, maxRetries)
}

// sendHTTPRequest sends the actual HTTP request
func (wm *WebhookManager) sendHTTPRequest(config *WebhookConfig, payload []byte) (success bool, statusCode int, responseBody string) {
	req, err := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		wm.logger.Errorf("Failed to create HTTP request: %v", err)
		return false, 0, err.Error()
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WhatsApp-Bridge-Webhook/1.0")
	
	// Add HMAC signature if secret token is provided
	if config.SecretToken != "" {
		signature := wm.generateHMACSignature(payload, config.SecretToken)
		req.Header.Set("X-Webhook-Signature", signature)
	}
	
	// Send request
	resp, err := wm.httpClient.Do(req)
	if err != nil {
		wm.logger.Errorf("HTTP request failed: %v", err)
		return false, 0, err.Error()
	}
	defer resp.Body.Close()
	
	// Read response body
	responseBytes := make([]byte, 1024) // Limit response size
	n, _ := resp.Body.Read(responseBytes)
	responseBody = string(responseBytes[:n])
	
	// Consider 2xx status codes as successful
	success = resp.StatusCode >= 200 && resp.StatusCode < 300
	
	return success, resp.StatusCode, responseBody
}

// generateHMACSignature generates HMAC-SHA256 signature for webhook authentication
func (wm *WebhookManager) generateHMACSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	signature := hex.EncodeToString(h.Sum(nil))
	return "sha256=" + signature
}

// ValidateWebhookConfig validates a webhook configuration
func (wm *WebhookManager) ValidateWebhookConfig(config *WebhookConfig) error {
	if config.Name == "" {
		return fmt.Errorf("webhook name is required")
	}
	
	if len(config.Name) > 255 {
		return fmt.Errorf("webhook name must be less than 256 characters")
	}
	
	if config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is required")
	}
	
	if len(config.WebhookURL) > 2048 {
		return fmt.Errorf("webhook URL must be less than 2048 characters")
	}
	
	if !strings.HasPrefix(config.WebhookURL, "http://") && !strings.HasPrefix(config.WebhookURL, "https://") {
		return fmt.Errorf("webhook URL must start with http:// or https://")
	}
	
	// Validate triggers
	for _, trigger := range config.Triggers {
		if trigger.TriggerType == "" {
			return fmt.Errorf("trigger type is required")
		}
		
		validTypes := []string{"all", "chat_jid", "sender", "keyword", "media_type"}
		valid := false
		for _, validType := range validTypes {
			if trigger.TriggerType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid trigger type: %s", trigger.TriggerType)
		}
		
		validMatchTypes := []string{"exact", "contains", "regex"}
		valid = false
		for _, validType := range validMatchTypes {
			if trigger.MatchType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid match type: %s", trigger.MatchType)
		}
		
		// Test regex patterns
		if trigger.MatchType == "regex" && trigger.TriggerValue != "" {
			_, err := regexp.Compile(trigger.TriggerValue)
			if err != nil {
				return fmt.Errorf("invalid regex pattern '%s': %v", trigger.TriggerValue, err)
			}
		}
	}
	
	return nil
}

// TestWebhook sends a test webhook to verify connectivity
func (wm *WebhookManager) TestWebhook(config *WebhookConfig) error {
	testPayload := WebhookPayload{
		EventType: "test",
		Timestamp: time.Now().Format(time.RFC3339),
		WebhookConfig: WebhookConfigInfo{
			ID:   config.ID,
			Name: config.Name,
		},
		Message: WebhookMessageInfo{
			ID:        "test-message-id",
			ChatJID:   "test@s.whatsapp.net",
			ChatName:  "Test Chat",
			Sender:    "test",
			SenderName: "Test User",
			Content:   "This is a test message",
			Timestamp: time.Now().Format(time.RFC3339),
			IsFromMe:  false,
		},
		Metadata: WebhookMetadata{
			DeliveryAttempt:  1,
			ProcessingTimeMs: 0,
		},
	}
	
	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal test payload: %v", err)
	}
	
	success, statusCode, responseBody := wm.sendHTTPRequest(config, payloadBytes)
	if !success {
		return fmt.Errorf("test webhook failed: status %d, response: %s", statusCode, responseBody)
	}
	
	return nil
}

// Extract text content from a message
func extractTextContent(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}

	// Try to get text content
	if text := msg.GetConversation(); text != "" {
		return text
	} else if extendedText := msg.GetExtendedTextMessage(); extendedText != nil {
		return extendedText.GetText()
	}

	// For now, we're ignoring non-text messages
	return ""
}

// API Response helper functions
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func sendJSONSuccess(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
	}
	if data != nil {
		response["data"] = data
	}
	if message != "" {
		response["message"] = message
	}
	json.NewEncoder(w).Encode(response)
}

// SendMessageResponse represents the response for the send message API
type SendMessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SendMessageRequest represents the request body for the send message API
type SendMessageRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	MediaPath string `json:"media_path,omitempty"`
}

// Function to send a WhatsApp message
func sendWhatsAppMessage(client *whatsmeow.Client,
						 messageStore *MessageStore,
						 recipient string, message string, mediaPath string) (bool, string) {
	if !client.IsConnected() {
		return false, "Not connected to WhatsApp"
	}

	// Create JID for recipient
	var recipientJID types.JID
	var err error

	// Check if recipient is a JID
	isJID := strings.Contains(recipient, "@")

	if isJID {
		// Parse the JID string
		recipientJID, err = types.ParseJID(recipient)
		if err != nil {
			return false, fmt.Sprintf("Error parsing JID: %v", err)
		}
	} else {
		// Create JID from phone number
		recipientJID = types.JID{
			User:   recipient,
			Server: "s.whatsapp.net", // For personal chats
		}
	}

	msg := &waProto.Message{}

	// Check if we have media to send
	if mediaPath != "" {
		// Read media file
		mediaData, err := os.ReadFile(mediaPath)
		if err != nil {
			return false, fmt.Sprintf("Error reading media file: %v", err)
		}

		// Determine media type and mime type based on file extension
		fileExt := strings.ToLower(mediaPath[strings.LastIndex(mediaPath, ".")+1:])
		var mediaType whatsmeow.MediaType
		var mimeType string

		// Handle different media types
		switch fileExt {
		// Image types
		case "jpg", "jpeg":
			mediaType = whatsmeow.MediaImage
			mimeType = "image/jpeg"
		case "png":
			mediaType = whatsmeow.MediaImage
			mimeType = "image/png"
		case "gif":
			mediaType = whatsmeow.MediaImage
			mimeType = "image/gif"
		case "webp":
			mediaType = whatsmeow.MediaImage
			mimeType = "image/webp"

		// Audio types
		case "ogg":
			mediaType = whatsmeow.MediaAudio
			mimeType = "audio/ogg; codecs=opus"

		// Video types
		case "mp4":
			mediaType = whatsmeow.MediaVideo
			mimeType = "video/mp4"
		case "avi":
			mediaType = whatsmeow.MediaVideo
			mimeType = "video/avi"
		case "mov":
			mediaType = whatsmeow.MediaVideo
			mimeType = "video/quicktime"

		// Document types (for any other file type)
		default:
			mediaType = whatsmeow.MediaDocument
			mimeType = "application/octet-stream"
		}

		// Upload media to WhatsApp servers
		resp, err := client.Upload(context.Background(), mediaData, mediaType)
		if err != nil {
			return false, fmt.Sprintf("Error uploading media: %v", err)
		}

		fmt.Println("Media uploaded", resp)

		// Create the appropriate message type based on media type
		switch mediaType {
		case whatsmeow.MediaImage:
			msg.ImageMessage = &waProto.ImageMessage{
				Caption:       proto.String(message),
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case whatsmeow.MediaAudio:
			// Handle ogg audio files
			var seconds uint32 = 30 // Default fallback
			var waveform []byte = nil

			// Try to analyze the ogg file
			if strings.Contains(mimeType, "ogg") {
				analyzedSeconds, analyzedWaveform, err := analyzeOggOpus(mediaData)
				if err == nil {
					seconds = analyzedSeconds
					waveform = analyzedWaveform
				} else {
					return false, fmt.Sprintf("Failed to analyze Ogg Opus file: %v", err)
				}
			} else {
				fmt.Printf("Not an Ogg Opus file: %s\n", mimeType)
			}

			msg.AudioMessage = &waProto.AudioMessage{
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
				Seconds:       proto.Uint32(seconds),
				PTT:           proto.Bool(true),
				Waveform:      waveform,
			}
		case whatsmeow.MediaVideo:
			msg.VideoMessage = &waProto.VideoMessage{
				Caption:       proto.String(message),
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case whatsmeow.MediaDocument:
			msg.DocumentMessage = &waProto.DocumentMessage{
				Title:         proto.String(mediaPath[strings.LastIndex(mediaPath, "/")+1:]),
				Caption:       proto.String(message),
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		}
	} else {
		msg.Conversation = proto.String(message)
	}

	// Send message
	sendResp, err := client.SendMessage(context.Background(), recipientJID, msg)
	if err != nil {
		return false, fmt.Sprintf("Error sending message: %v", err)
	}

	err = messageStore.StoreMessage(
		sendResp.ID, // Use the ID from SendResponse
		recipientJID.String(),
		client.Store.ID.User, // Use the client's user ID as sender
		msg.GetConversation(), // Use the conversation text
		sendResp.Timestamp, // Use the Timestamp from SendResponse
		true, // IsFromMe is true since we are sending this message
		"",
		"",
		"",
		nil, // Replace "" with nil for []byte arguments
		nil, // Replace "" with nil for []byte arguments
		nil, // Replace "" with nil for []byte arguments
		0,
	)

	return true, fmt.Sprintf("Message sent to %s", recipient)
}

// Extract media info from a message
func extractMediaInfo(msg *waProto.Message) (mediaType string, filename string, url string, mediaKey []byte, fileSHA256 []byte, fileEncSHA256 []byte, fileLength uint64) {
	if msg == nil {
		return "", "", "", nil, nil, nil, 0
	}

	// Check for image message
	if img := msg.GetImageMessage(); img != nil {
		return "image", "image_" + time.Now().Format("20060102_150405") + ".jpg",
			img.GetURL(), img.GetMediaKey(), img.GetFileSHA256(), img.GetFileEncSHA256(), img.GetFileLength()
	}

	// Check for video message
	if vid := msg.GetVideoMessage(); vid != nil {
		return "video", "video_" + time.Now().Format("20060102_150405") + ".mp4",
			vid.GetURL(), vid.GetMediaKey(), vid.GetFileSHA256(), vid.GetFileEncSHA256(), vid.GetFileLength()
	}

	// Check for audio message
	if aud := msg.GetAudioMessage(); aud != nil {
		return "audio", "audio_" + time.Now().Format("20060102_150405") + ".ogg",
			aud.GetURL(), aud.GetMediaKey(), aud.GetFileSHA256(), aud.GetFileEncSHA256(), aud.GetFileLength()
	}

	// Check for document message
	if doc := msg.GetDocumentMessage(); doc != nil {
		filename := doc.GetFileName()
		if filename == "" {
			filename = "document_" + time.Now().Format("20060102_150405")
		}
		return "document", filename,
			doc.GetURL(), doc.GetMediaKey(), doc.GetFileSHA256(), doc.GetFileEncSHA256(), doc.GetFileLength()
	}

	return "", "", "", nil, nil, nil, 0
}

// Handle regular incoming messages with media support and webhook processing
func handleMessage(client *whatsmeow.Client, messageStore *MessageStore, webhookManager *WebhookManager, msg *events.Message, logger waLog.Logger) {
	// Save message to database
	chatJID := msg.Info.Chat.String()
	sender := msg.Info.Sender.User

	// Get appropriate chat name (pass nil for conversation since we don't have one for regular messages)
	name := GetChatName(client, messageStore, msg.Info.Chat, chatJID, nil, sender, logger)

	// Update chat in database with the message timestamp (keeps last message time updated)
	err := messageStore.StoreChat(chatJID, name, msg.Info.Timestamp)
	if err != nil {
		logger.Warnf("Failed to store chat: %v", err)
	}

	// Extract text content
	content := extractTextContent(msg.Message)

	// Extract media info
	mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength := extractMediaInfo(msg.Message)

	// Skip if there's no content and no media
	if content == "" && mediaType == "" {
		return
	}

	// Store message in database
	err = messageStore.StoreMessage(
		msg.Info.ID,
		chatJID,
		sender,
		content,
		msg.Info.Timestamp,
		msg.Info.IsFromMe,
		mediaType,
		filename,
		url,
		mediaKey,
		fileSHA256,
		fileEncSHA256,
		fileLength,
	)

	if err != nil {
		logger.Warnf("Failed to store message: %v", err)
	} else {
		// Log message reception
		timestamp := msg.Info.Timestamp.Format("2006-01-02 15:04:05")
		direction := "←"
		if msg.Info.IsFromMe {
			direction = "→"
		}

		// Log based on message type
		if mediaType != "" {
			fmt.Printf("[%s] %s %s: [%s: %s] %s\n", timestamp, direction, sender, mediaType, filename, content)
		} else if content != "" {
			fmt.Printf("[%s] %s %s: %s\n", timestamp, direction, sender, content)
		}
	}

	// Process webhooks if manager is available
	if webhookManager != nil {
		webhookManager.ProcessMessage(client, msg, name)
	}
}

// GetChatName determines the appropriate name for a chat based on JID and other info
func GetChatName(client *whatsmeow.Client, messageStore *MessageStore, jid types.JID, chatJID string, conversation interface{}, sender string, logger waLog.Logger) string {
	// First, check if chat already exists in database with a name
	var existingName string
	err := messageStore.db.QueryRow("SELECT name FROM chats WHERE jid = ?", chatJID).Scan(&existingName)
	if err == nil && existingName != "" {
		// Chat exists with a name, use that
		logger.Infof("Using existing chat name for %s: %s", chatJID, existingName)
		return existingName
	}

	// Need to determine chat name
	var name string

	if jid.Server == "g.us" {
		// This is a group chat
		logger.Infof("Getting name for group: %s", chatJID)

		// Use conversation data if provided (from history sync)
		if conversation != nil {
			// Extract name from conversation if available
			// This uses type assertions to handle different possible types
			var displayName, convName *string
			// Try to extract the fields we care about regardless of the exact type
			v := reflect.ValueOf(conversation)
			if v.Kind() == reflect.Ptr && !v.IsNil() {
				v = v.Elem()

				// Try to find DisplayName field
				if displayNameField := v.FieldByName("DisplayName"); displayNameField.IsValid() && displayNameField.Kind() == reflect.Ptr && !displayNameField.IsNil() {
					dn := displayNameField.Elem().String()
					displayName = &dn
				}

				// Try to find Name field
				if nameField := v.FieldByName("Name"); nameField.IsValid() && nameField.Kind() == reflect.Ptr && !nameField.IsNil() {
					n := nameField.Elem().String()
					convName = &n
				}
			}

			// Use the name we found
			if displayName != nil && *displayName != "" {
				name = *displayName
			} else if convName != nil && *convName != "" {
				name = *convName
			}
		}

		// If we didn't get a name, try group info
		if name == "" {
			groupInfo, err := client.GetGroupInfo(jid)
			if err == nil && groupInfo.Name != "" {
				name = groupInfo.Name
			} else {
				// Fallback name for groups
				name = fmt.Sprintf("Group %s", jid.User)
			}
		}

		logger.Infof("Using group name: %s", name)
	} else {
		// This is an individual contact
		logger.Infof("Getting name for contact: %s", chatJID)

		// Just use contact info (full name)
		contact, err := client.Store.Contacts.GetContact(context.Background(), jid)
		if err == nil && contact.FullName != "" {
			name = contact.FullName
		} else if sender != "" {
			// Fallback to sender
			name = sender
		} else {
			// Last fallback to JID
			name = jid.User
		}

		logger.Infof("Using contact name: %s", name)
	}

	return name
}

// Start a REST API server to expose the WhatsApp client functionality
func startRESTServer(client *whatsmeow.Client, messageStore *MessageStore, webhookManager *WebhookManager, port int) {
	// Handler for sending messages
	http.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests
		if r.Method != http.MethodPost {
			sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the request body
		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONError(w, "Invalid request format", http.StatusBadRequest)
			return
		}		// Validate request
		if req.Recipient == "" {
			sendJSONError(w, "Recipient is required", http.StatusBadRequest)
			return
		}
		
		if req.Message == "" && req.MediaPath == "" {
			sendJSONError(w, "Message or media path is required", http.StatusBadRequest)
			return
		}

		fmt.Println("Received request to send message", req.Message, req.MediaPath)

		// Send the message
		success, message := sendWhatsAppMessage(client, messageStore, req.Recipient, req.Message, req.MediaPath)
		fmt.Println("Message sent", success, message)
		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Set appropriate status code
		if !success {
			w.WriteHeader(http.StatusInternalServerError)
		}

		// Send response
		json.NewEncoder(w).Encode(SendMessageResponse{
			Success: success,
			Message: message,
		})
	})

	// Webhook management endpoints
	http.HandleFunc("/api/webhooks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		switch r.Method {
		case http.MethodGet:
			// List all webhook configurations
			configs := webhookManager.GetWebhookConfigs()
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    configs,
			})
			
		case http.MethodPost:
			// Create new webhook configuration
			var config WebhookConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				sendJSONError(w, "Invalid request format", http.StatusBadRequest)
				return
			}
			
			// Validate configuration
			if err := webhookManager.ValidateWebhookConfig(&config); err != nil {
				sendJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}
			
			// Store configuration
			if err := messageStore.StoreWebhookConfig(&config); err != nil {
				sendJSONError(w, fmt.Sprintf("Failed to store webhook config: %v", err), http.StatusInternalServerError)
				return
			}
			
			// Reload configurations
			webhookManager.LoadWebhookConfigs()
			
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    config,
			})
			
		default:
			sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Individual webhook management endpoints
	http.HandleFunc("/api/webhooks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Parse webhook ID from URL path
		pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/webhooks/"), "/")
		if len(pathParts) == 0 || pathParts[0] == "" {
			sendJSONError(w, "Webhook ID is required", http.StatusBadRequest)
			return
		}
		
		webhookIDStr := pathParts[0]
		webhookID := 0
		if _, err := fmt.Sscanf(webhookIDStr, "%d", &webhookID); err != nil {
			sendJSONError(w, "Invalid webhook ID", http.StatusBadRequest)
			return
		}
		
		// Handle different sub-paths
		switch {
		case len(pathParts) == 1: // /api/webhooks/{id}
			switch r.Method {
			case http.MethodGet:
				// Get specific webhook configuration
				config, err := messageStore.GetWebhookConfig(webhookID)
				if err != nil {
					sendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
					return
				}
				
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"data":    config,
				})
				
			case http.MethodPut:
				// Update webhook configuration
				var config WebhookConfig
				if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
					sendJSONError(w, "Invalid request format", http.StatusBadRequest)
					return
				}
				
				config.ID = webhookID // Ensure ID matches URL
				
				// Validate configuration
				if err := webhookManager.ValidateWebhookConfig(&config); err != nil {
					sendJSONError(w, err.Error(), http.StatusBadRequest)
					return
				}
				
				// Update configuration
				if err := messageStore.UpdateWebhookConfig(&config); err != nil {
					sendJSONError(w, fmt.Sprintf("Failed to update webhook config: %v", err), http.StatusInternalServerError)
					return
				}
				
				// Reload configurations
				webhookManager.LoadWebhookConfigs()
				
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"data":    config,
				})
				
			case http.MethodDelete:
				// Delete webhook configuration
				if err := messageStore.DeleteWebhookConfig(webhookID); err != nil {
					sendJSONError(w, fmt.Sprintf("Failed to delete webhook config: %v", err), http.StatusInternalServerError)
					return
				}
				
				// Reload configurations
				webhookManager.LoadWebhookConfigs()
				
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"message": "Webhook deleted successfully",
				})
				
			default:
				sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			
		case len(pathParts) == 2 && pathParts[1] == "test": // /api/webhooks/{id}/test
			if r.Method != http.MethodPost {
				sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}				// Get webhook configuration
				config, err := messageStore.GetWebhookConfig(webhookID)
				if err != nil {
					sendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
					return
				}
			
			// Test webhook
			if err := webhookManager.TestWebhook(config); err != nil {
				sendJSONError(w, fmt.Sprintf("Webhook test failed: %v", err), http.StatusInternalServerError)
				return
			}
			
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Webhook test successful",
			})
			
		case len(pathParts) == 2 && pathParts[1] == "logs": // /api/webhooks/{id}/logs
			if r.Method != http.MethodGet {
				sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			
			// Get webhook logs
			logs, err := messageStore.GetWebhookLogs(webhookID, 100) // Limit to 100 recent logs
			if err != nil {
				sendJSONError(w, fmt.Sprintf("Failed to get webhook logs: %v", err), http.StatusInternalServerError)
				return
			}
			
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    logs,
			})
			
		case len(pathParts) == 2 && pathParts[1] == "enable": // /api/webhooks/{id}/enable
			if r.Method != http.MethodPost {
				sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			
			// Parse request body to get enabled status
			var req struct {
				Enabled bool `json:"enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				sendJSONError(w, "Invalid request format", http.StatusBadRequest)
				return
			}
			
			// Get current config
			config, err := messageStore.GetWebhookConfig(webhookID)
			if err != nil {
				sendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
				return
			}
			
			// Update enabled status
			config.Enabled = req.Enabled
			if err := messageStore.UpdateWebhookConfig(config); err != nil {
				sendJSONError(w, fmt.Sprintf("Failed to update webhook config: %v", err), http.StatusInternalServerError)
				return
			}
			
			// Reload configurations
			webhookManager.LoadWebhookConfigs()
			
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Webhook %s successfully", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]),
				"data":    config,
			})
			
		default:
			sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Webhook logs endpoint (all logs)
	http.HandleFunc("/api/webhook-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		
		// Get all webhook logs
		logs, err := messageStore.GetWebhookLogs(0, 100) // Get last 100 logs for all webhooks
		if err != nil {
			sendJSONError(w, fmt.Sprintf("Failed to get webhook logs: %v", err), http.StatusInternalServerError)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    logs,
		})
	})

	// Start the server
	serverAddr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting REST API server on %s...\n", serverAddr)

	// Run server in a goroutine so it doesn't block
	go func() {
		if err := http.ListenAndServe(serverAddr, nil); err != nil {
			fmt.Printf("REST API server error: %v\n", err)
		}
	}()
}

func main() {
	// Set up logger
	logger := waLog.Stdout("Client", "INFO", true)
	logger.Infof("Starting WhatsApp client...")

	// Create database connection for storing session data
	dbLog := waLog.Stdout("Database", "INFO", true)

	// Create directory for database if it doesn't exist
	if err := os.MkdirAll("store", 0755); err != nil {
		logger.Errorf("Failed to create store directory: %v", err)
		return
	}

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:store/whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		logger.Errorf("Failed to connect to database: %v", err)
		return
	}

	// Get device store - This contains session information
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			// No device exists, create one
			deviceStore = container.NewDevice()
			logger.Infof("Created new device")
		} else {
			logger.Errorf("Failed to get device: %v", err)
			return
		}
	}

	// Create client instance
	client := whatsmeow.NewClient(deviceStore, logger)
	if client == nil {
		logger.Errorf("Failed to create WhatsApp client")
		return
	}

	// Initialize message store
	messageStore, err := NewMessageStore()
	if err != nil {
		logger.Errorf("Failed to initialize message store: %v", err)
		return
	}
	defer messageStore.Close()

	// Initialize webhook manager
	webhookManager := NewWebhookManager(messageStore, logger)
	err = webhookManager.LoadWebhookConfigs()
	if err != nil {
		logger.Errorf("Failed to load webhook configs: %v", err)
		return
	}

	// Setup event handling for messages and history sync
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Process regular messages with webhook support
			handleMessage(client, messageStore, webhookManager, v, logger)

		case *events.HistorySync:
			// Process history sync events
			handleHistorySync(client, messageStore, v, logger)

		case *events.Connected:
			logger.Infof("Connected to WhatsApp")

		case *events.LoggedOut:
			logger.Warnf("Device logged out, please scan QR code to log in again")
		}
	})

	// Create channel to track connection success
	connected := make(chan bool, 1)

	// Connect to WhatsApp
	if client.Store.ID == nil {
		// No ID stored, this is a new client, need to pair with phone
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			logger.Errorf("Failed to connect: %v", err)
			return
		}

		// Print QR code for pairing with phone
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan this QR code with your WhatsApp app:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else if evt.Event == "success" {
				connected <- true
				break
			}
		}

		// Wait for connection
		select {
		case <-connected:
			fmt.Println("\nSuccessfully connected and authenticated!")
		case <-time.After(3 * time.Minute):
			logger.Errorf("Timeout waiting for QR code scan")
			return
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			logger.Errorf("Failed to connect: %v", err)
			return
		}
		connected <- true
	}

	// Wait a moment for connection to stabilize
	time.Sleep(2 * time.Second)

	if !client.IsConnected() {
		logger.Errorf("Failed to establish stable connection")
		return
	}

	fmt.Println("\n✓ Connected to WhatsApp! Type 'help' for commands.")

	// Start REST API server with webhook support
	startRESTServer(client, messageStore, webhookManager, 8080)

	// Create a channel to keep the main goroutine alive
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("REST server is running. Press Ctrl+C to disconnect and exit.")

	// Wait for termination signal
	<-exitChan

	fmt.Println("Disconnecting...")
	// Disconnect client
	client.Disconnect()
}

// Handle history sync events
func handleHistorySync(client *whatsmeow.Client, messageStore *MessageStore, historySync *events.HistorySync, logger waLog.Logger) {
	fmt.Printf("Received history sync event with %d conversations\n", len(historySync.Data.Conversations))

	syncedCount := 0
	for _, conversation := range historySync.Data.Conversations {
		// Parse JID from the conversation
		if conversation.ID == nil {
			continue
		}

		chatJID := *conversation.ID

		// Try to parse the JID
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			logger.Warnf("Failed to parse JID %s: %v", chatJID, err)
			continue
		}

		// Get appropriate chat name by passing the history sync conversation directly
		name := GetChatName(client, messageStore, jid, chatJID, conversation, "", logger)

		// Process messages
		messages := conversation.Messages
		if len(messages) > 0 {
			// Update chat with latest message timestamp
			latestMsg := messages[0]
			if latestMsg == nil || latestMsg.Message == nil {
				continue
			}

			// Get timestamp from message info
			timestamp := time.Time{}
			if ts := latestMsg.Message.GetMessageTimestamp(); ts != 0 {
				timestamp = time.Unix(int64(ts), 0)
			} else {
				continue
			}

			messageStore.StoreChat(chatJID, name, timestamp)

			// Store messages
			for _, msg := range messages {
				if msg == nil || msg.Message == nil {
					continue
				}

				// Extract text content
				var content string
				if msg.Message.Message != nil {
					if conv := msg.Message.Message.GetConversation(); conv != "" {
						content = conv
					} else if ext := msg.Message.Message.GetExtendedTextMessage(); ext != nil {
						content = ext.GetText()
					}
				}

				// Extract media info
				var mediaType, filename, url string
				var mediaKey, fileSHA256, fileEncSHA256 []byte
				var fileLength uint64

				if msg.Message.Message != nil {
					mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength = extractMediaInfo(msg.Message.Message)
				}

				// Log the message content for debugging
				logger.Infof("Message content: %v, Media Type: %v", content, mediaType)

				// Skip messages with no content and no media
				if content == "" && mediaType == "" {
					continue
				}

				// Determine sender
				var sender string
				isFromMe := false
				if msg.Message.Key != nil {
					if msg.Message.Key.FromMe != nil {
						isFromMe = *msg.Message.Key.FromMe
					}
					if !isFromMe && msg.Message.Key.Participant != nil && *msg.Message.Key.Participant != "" {
						sender = *msg.Message.Key.Participant
					} else if isFromMe {
						sender = client.Store.ID.User
					} else {
						sender = jid.User
					}
				} else {
					sender = jid.User
				}

				// Store message
				msgID := ""
				if msg.Message.Key != nil && msg.Message.Key.ID != nil {
					msgID = *msg.Message.Key.ID
				}

				// Get message timestamp
				timestamp := time.Time{}
				if ts := msg.Message.GetMessageTimestamp(); ts != 0 {
					timestamp = time.Unix(int64(ts), 0)
				} else {
					continue
				}

				err = messageStore.StoreMessage(
					msgID,
					chatJID,
					sender,
					content,
					timestamp,
					isFromMe,
					mediaType,
					filename,
					url,
					mediaKey,
					fileSHA256,
					fileEncSHA256,
					fileLength,
				)
				if err != nil {
					logger.Warnf("Failed to store history message: %v", err)
				} else {
					syncedCount++
					// Log successful message storage
					if mediaType != "" {
						logger.Infof("Stored message: [%s] %s -> %s: [%s: %s] %s",
							timestamp.Format("2006-01-02 15:04:05"), sender, chatJID, mediaType, filename, content)
					} else {
						logger.Infof("Stored message: [%s] %s -> %s: %s",
							timestamp.Format("2006-01-02 15:04:05"), sender, chatJID, content)
					}
				}
			}
		}
	}

	fmt.Printf("History sync complete. Stored %d messages.\n", syncedCount)
}

// analyzeOggOpus tries to extract duration and generate a simple waveform from an Ogg Opus file
func analyzeOggOpus(data []byte) (duration uint32, waveform []byte, err error) {
	// Try to detect if this is a valid Ogg file by checking for the "OggS" signature
	// at the beginning of the file
	if len(data) < 4 || string(data[0:4]) != "OggS" {
		return 0, nil, fmt.Errorf("not a valid Ogg file (missing OggS signature)")
	}

	// Parse Ogg pages to find the last page with a valid granule position
	var lastGranule uint64
	var sampleRate uint32 = 48000 // Default Opus sample rate
	var preSkip uint16 = 0
	var foundOpusHead bool

	// Scan through the file looking for Ogg pages
	for i := 0; i < len(data); {
		// Check if we have enough data to read Ogg page header
		if i+27 >= len(data) {
			break
		}

		// Verify Ogg page signature
		if string(data[i:i+4]) != "OggS" {
			// Skip until next potential page
			i++
			continue
		}

		// Extract header fields
		granulePos := binary.LittleEndian.Uint64(data[i+6 : i+14])
		pageSeqNum := binary.LittleEndian.Uint32(data[i+18 : i+22])
		numSegments := int(data[i+26])

		// Extract segment table
		if i+27+numSegments >= len(data) {
			break
		}
		segmentTable := data[i+27 : i+27+numSegments]

		// Calculate page size
		pageSize := 27 + numSegments
		for _, segLen := range segmentTable {
			pageSize += int(segLen)
		}

		// Check if we're looking at an OpusHead packet (should be in first few pages)
		if !foundOpusHead && pageSeqNum <= 1 {
			// Look for "OpusHead" marker in this page
			pageData := data[i : i+pageSize]
			headPos := bytes.Index(pageData, []byte("OpusHead"))
			if headPos >= 0 && headPos+12 < len(pageData) {
				// Found OpusHead, extract sample rate and pre-skip
				// OpusHead format: Magic(8) + Version(1) + Channels(1) + PreSkip(2) + SampleRate(4) + ...
				headPos += 8 // Skip "OpusHead" marker
				// PreSkip is 2 bytes at offset 10
				if headPos+12 <= len(pageData) {
					preSkip = binary.LittleEndian.Uint16(pageData[headPos+10 : headPos+12])
					sampleRate = binary.LittleEndian.Uint32(pageData[headPos+12 : headPos+16])
					foundOpusHead = true
					fmt.Printf("Found OpusHead: sampleRate=%d, preSkip=%d\n", sampleRate, preSkip)
				}
			}
		}

		// Keep track of last valid granule position
		if granulePos != 0 {
			lastGranule = granulePos
		}

		// Move to next page
		i += pageSize
	}

	if !foundOpusHead {
		fmt.Println("Warning: OpusHead not found, using default values")
	}

	// Calculate duration based on granule position
	if lastGranule > 0 {
		// Formula for duration: (lastGranule - preSkip) / sampleRate
		durationSeconds := float64(lastGranule-uint64(preSkip)) / float64(sampleRate)
		duration = uint32(math.Ceil(durationSeconds))
		fmt.Printf("Calculated Opus duration from granule: %f seconds (lastGranule=%d)\n",
			durationSeconds, lastGranule)
	} else {
		// Fallback to rough estimation if granule position not found
		fmt.Println("Warning: No valid granule position found, using estimation")
		durationEstimate := float64(len(data)) / 2000.0 // Very rough approximation
		duration = uint32(durationEstimate)
	}

	// Make sure we have a reasonable duration (at least 1 second, at most 300 seconds)
	if duration < 1 {
		duration = 1
	} else if duration > 300 {
		duration = 300
	}

	// Generate waveform
	waveform = placeholderWaveform(duration)

	fmt.Printf("Ogg Opus analysis: size=%d bytes, calculated duration=%d sec, waveform=%d bytes\n",
		len(data), duration, len(waveform))

	return duration, waveform, nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// placeholderWaveform generates a synthetic waveform for WhatsApp voice messages
// that appears natural with some variability based on the duration
func placeholderWaveform(duration uint32) []byte {
	// WhatsApp expects a 64-byte waveform for voice messages
	const waveformLength = 64
	waveform := make([]byte, waveformLength)

	// Seed the random number generator for consistent results with the same duration
	rand.Seed(int64(duration))

	// Create a more natural looking waveform with some patterns and variability
	// rather than completely random values

	// Base amplitude and frequency - longer messages get faster frequency
	baseAmplitude := 35.0
	frequencyFactor := float64(min(int(duration), 120)) / 30.0

	for i := range waveform {
		// Position in the waveform (normalized 0-1)
		pos := float64(i) / float64(waveformLength)

		// Create a wave pattern with some randomness
		// Use multiple sine waves of different frequencies for more natural look
		val := baseAmplitude * math.Sin(pos*math.Pi*frequencyFactor*8)
		val += (baseAmplitude / 2) * math.Sin(pos*math.Pi*frequencyFactor*16)

		// Add some randomness to make it look more natural
		val += (rand.Float64() - 0.5) * 15

		// Add some fade-in and fade-out effects
		fadeInOut := math.Sin(pos * math.Pi)
		val = val * (0.7 + 0.3*fadeInOut)

		// Center around 50 (typical voice baseline)
		val = val + 50

		// Ensure values stay within WhatsApp's expected range (0-100)
		if val < 0 {
			val = 0
		} else if val > 100 {
			val = 100
		}

		waveform[i] = byte(val)
	}

	return waveform
}