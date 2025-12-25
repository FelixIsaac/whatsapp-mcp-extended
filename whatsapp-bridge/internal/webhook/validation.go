package webhook

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"whatsapp-bridge/internal/types"
)

// privateIPBlocks contains CIDR ranges for private/reserved IPs
var privateIPBlocks []*net.IPNet

func init() {
	// Initialize private IP ranges
	cidrs := []string{
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link-local
		"0.0.0.0/8",      // Current network
		"224.0.0.0/4",    // Multicast
		"240.0.0.0/4",    // Reserved
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
	}

	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

// isPrivateIP checks if an IP is in a private/reserved range
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateWebhookURL checks if the webhook URL is safe (no SSRF)
func ValidateWebhookURL(webhookURL string) error {
	// Skip SSRF check if explicitly disabled (for testing)
	if os.Getenv("DISABLE_SSRF_CHECK") == "true" {
		return nil
	}

	u, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %v", err)
	}

	hostname := u.Hostname()

	// Block common metadata endpoints
	blockedHosts := []string{
		"metadata.google.internal",
		"169.254.169.254",
		"metadata.azure.com",
	}
	for _, blocked := range blockedHosts {
		if strings.EqualFold(hostname, blocked) {
			return fmt.Errorf("webhook URL hostname is blocked: %s", hostname)
		}
	}

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve webhook URL hostname: %v", err)
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL resolves to private/reserved IP: %s -> %s", hostname, ip.String())
		}
	}

	return nil
}

// ValidateWebhookConfig validates a webhook configuration
func (wm *Manager) ValidateWebhookConfig(config *types.WebhookConfig) error {
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

	// SSRF prevention: validate webhook URL doesn't resolve to private IP
	if err := ValidateWebhookURL(config.WebhookURL); err != nil {
		return err
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
func (wm *Manager) TestWebhook(config *types.WebhookConfig) error {
	testPayload := types.WebhookPayload{
		EventType: "test",
		Timestamp: time.Now().Format(time.RFC3339),
		WebhookConfig: types.WebhookConfigInfo{
			ID:   config.ID,
			Name: config.Name,
		},
		Message: types.WebhookMessageInfo{
			ID:         "test-message-id",
			ChatJID:    "test@s.whatsapp.net",
			ChatName:   "Test Chat",
			Sender:     "test",
			SenderName: "Test User",
			Content:    "This is a test message",
			Timestamp:  time.Now().Format(time.RFC3339),
			IsFromMe:   false,
		},
		Metadata: types.WebhookMetadata{
			DeliveryAttempt:  1,
			ProcessingTimeMs: 0,
		},
	}

	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal test payload: %v", err)
	}

	success, statusCode, responseBody := wm.delivery.sendHTTPRequest(config, payloadBytes)
	if !success {
		return fmt.Errorf("test webhook failed: status %d, response: %s", statusCode, responseBody)
	}

	return nil
}
