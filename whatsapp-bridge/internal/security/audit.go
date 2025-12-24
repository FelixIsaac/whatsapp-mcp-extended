package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// AuditLogger logs security-relevant events
type AuditLogger struct {
	logger *log.Logger
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	Timestamp string `json:"timestamp"`
	EventType string `json:"event_type"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Action    string `json:"action,omitempty"`
	Status    string `json:"status"` // success, failure, blocked
	Details   string `json:"details,omitempty"`
}

var defaultAuditLogger *AuditLogger

func init() {
	defaultAuditLogger = NewAuditLogger()
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logger: log.New(os.Stdout, "[AUDIT] ", log.LstdFlags),
	}
}

// Log logs an audit event
func (a *AuditLogger) Log(event AuditEvent) {
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(event)
	if err != nil {
		a.logger.Printf("ERROR marshaling audit event: %v", err)
		return
	}
	a.logger.Println(string(data))
}

// LogAuthFailure logs an authentication failure
func LogAuthFailure(ip, userAgent, details string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "auth_failure",
		IP:        ip,
		UserAgent: userAgent,
		Status:    "failure",
		Details:   details,
	})
}

// LogAuthSuccess logs successful authentication
func LogAuthSuccess(ip, resource string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "auth_success",
		IP:        ip,
		Resource:  resource,
		Status:    "success",
	})
}

// LogRateLimitExceeded logs rate limit violations
func LogRateLimitExceeded(ip string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "rate_limit_exceeded",
		IP:        ip,
		Status:    "blocked",
	})
}

// LogWebhookCreated logs webhook creation
func LogWebhookCreated(ip string, webhookID int, webhookURL string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "webhook_created",
		IP:        ip,
		Resource:  webhookURL,
		Action:    "create",
		Status:    "success",
		Details:   fmt.Sprintf("id=%d", webhookID),
	})
}

// LogWebhookDeleted logs webhook deletion
func LogWebhookDeleted(ip string, webhookID int) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "webhook_deleted",
		IP:        ip,
		Action:    "delete",
		Status:    "success",
		Details:   fmt.Sprintf("id=%d", webhookID),
	})
}

// LogSSRFBlocked logs blocked SSRF attempts
func LogSSRFBlocked(ip, targetURL string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "ssrf_blocked",
		IP:        ip,
		Resource:  targetURL,
		Status:    "blocked",
		Details:   "Private IP or localhost target",
	})
}

// LogPathTraversalBlocked logs blocked path traversal attempts
func LogPathTraversalBlocked(ip, path string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "path_traversal_blocked",
		IP:        ip,
		Resource:  path,
		Status:    "blocked",
		Details:   "Path outside allowed directory",
	})
}

// LogMessageSent logs outgoing messages
func LogMessageSent(recipient, messageType string) {
	defaultAuditLogger.Log(AuditEvent{
		EventType: "message_sent",
		Resource:  recipient,
		Action:    messageType,
		Status:    "success",
	})
}
