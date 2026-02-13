package webhook

import (
	"net"
	"os"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Private IPs (should return true)
		{"loopback", "127.0.0.1", true},
		{"loopback full", "127.255.255.255", true},
		{"private 10.x", "10.0.0.1", true},
		{"private 10.x max", "10.255.255.255", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 172.31.x", "172.31.255.255", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"private 192.168.x max", "192.168.255.255", true},
		{"link-local", "169.254.1.1", true},
		{"current network", "0.0.0.0", true},
		{"multicast", "224.0.0.1", true},
		{"reserved", "240.0.0.1", true},
		{"ipv6 loopback", "::1", true},

		// Public IPs (should return false)
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 1.1.1.1", "1.1.1.1", false},
		{"public cloudflare", "104.16.132.229", false},
		{"not private 172.32.x", "172.32.0.1", false},
		{"not private 192.169.x", "192.169.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}

			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantErr     bool
		errContains string
	}{
		// Invalid URLs
		{"invalid url", "not-a-url", true, "failed to resolve"},
		{"empty url", "", true, "failed to resolve"},

		// Blocked metadata endpoints
		{"google metadata", "http://metadata.google.internal/computeMetadata/v1/", true, "blocked"},
		{"aws metadata", "http://169.254.169.254/latest/meta-data/", true, "blocked"},
		{"azure metadata", "http://metadata.azure.com/", true, "blocked"},

		// Private IPs
		{"loopback", "http://127.0.0.1:8080/webhook", true, "private"},
		{"private 10.x", "http://10.0.0.1/webhook", true, "private"},
		{"private 192.168.x", "http://192.168.1.1/webhook", true, "private"},
		{"private 172.16.x", "http://172.16.0.1/webhook", true, "private"},
		{"localhost", "http://localhost/webhook", true, "private"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateWebhookURL(%s) = nil, want error containing %q", tt.url, tt.errContains)
					return
				}
				if tt.errContains != "" && !containsIgnoreCase(err.Error(), tt.errContains) {
					t.Errorf("ValidateWebhookURL(%s) error = %v, want error containing %q", tt.url, err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateWebhookURL(%s) = %v, want nil", tt.url, err)
				}
			}
		})
	}
}

func TestValidateWebhookURL_DisableCheck(t *testing.T) {
	// Save and restore env var
	original := os.Getenv("DISABLE_SSRF_CHECK")
	defer os.Setenv("DISABLE_SSRF_CHECK", original)

	// Enable SSRF bypass
	os.Setenv("DISABLE_SSRF_CHECK", "true")

	// Should now allow private IPs
	err := ValidateWebhookURL("http://127.0.0.1/webhook")
	if err != nil {
		t.Errorf("With DISABLE_SSRF_CHECK=true, ValidateWebhookURL should allow private IPs, got: %v", err)
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && len(substr) > 0 &&
			(s[0] == substr[0] || s[0]+32 == substr[0] || s[0]-32 == substr[0]) &&
			containsIgnoreCase(s[1:], substr[1:]) ||
		len(s) > 0 && containsIgnoreCase(s[1:], substr))
}
