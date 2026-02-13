package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"whatsapp-bridge/internal/database"
	"whatsapp-bridge/internal/types"

	waLog "go.mau.fi/whatsmeow/util/log"
)

// DeliveryService handles webhook delivery with retry logic
type DeliveryService struct {
	messageStore *database.MessageStore
	logger       waLog.Logger
	httpClient   *http.Client
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(messageStore *database.MessageStore, logger waLog.Logger) *DeliveryService {
	return &DeliveryService{
		messageStore: messageStore,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DeliverWebhook delivers a webhook with retry logic
func (ds *DeliveryService) DeliverWebhook(config *types.WebhookConfig, payload *types.WebhookPayload, messageID, chatJID string, trigger *types.WebhookTrigger) {
	maxRetries := 5
	backoffIntervals := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}

	if _, err := json.Marshal(payload); err != nil {
		ds.logger.Errorf("Failed to marshal webhook payload: %v", err)
		return
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		payload.Metadata.DeliveryAttempt = attempt

		// Update payload with current attempt
		payloadBytes, _ := json.Marshal(payload)

		success, statusCode, responseBody := ds.sendHTTPRequest(config, payloadBytes)

		// Log the delivery attempt
		log := &types.WebhookLog{
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
			ds.logger.Infof("Webhook delivered successfully to %s (attempt %d)", config.WebhookURL, attempt)
		} else {
			ds.logger.Warnf("Webhook delivery failed to %s (attempt %d): status %d", config.WebhookURL, attempt, statusCode)
		}

		// Store log
		if err := ds.messageStore.StoreWebhookLog(log); err != nil {
			ds.logger.Errorf("Failed to store webhook log: %v", err)
		}

		if success {
			return // Success, no need to retry
		}

		// Wait before retry (except for last attempt)
		if attempt < maxRetries {
			time.Sleep(backoffIntervals[attempt-1])
		}
	}

	ds.logger.Errorf("Webhook delivery failed permanently to %s after %d attempts", config.WebhookURL, maxRetries)
}

// sendHTTPRequest sends the actual HTTP request
func (ds *DeliveryService) sendHTTPRequest(config *types.WebhookConfig, payload []byte) (success bool, statusCode int, responseBody string) {
	req, err := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		ds.logger.Errorf("Failed to create HTTP request: %v", err)
		return false, 0, err.Error()
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WhatsApp-Bridge-Webhook/1.0")

	// Add HMAC signature if secret token is provided
	if config.SecretToken != "" {
		signature := ds.generateHMACSignature(payload, config.SecretToken)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request
	resp, err := ds.httpClient.Do(req)
	if err != nil {
		ds.logger.Errorf("HTTP request failed: %v", err)
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
func (ds *DeliveryService) generateHMACSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	signature := hex.EncodeToString(h.Sum(nil))
	return "sha256=" + signature
}
