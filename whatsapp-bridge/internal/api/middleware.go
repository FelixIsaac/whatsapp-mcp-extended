package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"whatsapp-bridge/internal/security"
)

// Rate limiter state
var (
	rateLimitMu     sync.Mutex
	requestCounts   = make(map[string]int)
	requestWindows  = make(map[string]time.Time)
	rateLimit       = 100 // requests per window
	rateLimitWindow = time.Minute
)

// getAllowedOrigins returns the list of allowed CORS origins
func getAllowedOrigins() map[string]bool {
	origins := map[string]bool{
		"http://localhost:8089": true, // Webhook UI
		"http://localhost:8082": true, // Gradio UI
		"http://localhost:8090": true, // Pairing UI
	}

	// Allow additional origins from env var (comma-separated)
	if extra := os.Getenv("CORS_ORIGINS"); extra != "" {
		for _, origin := range strings.Split(extra, ",") {
			origins[strings.TrimSpace(origin)] = true
		}
	}

	return origins
}

// AuthMiddleware validates API key authentication using constant-time comparison
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedKey := os.Getenv("API_KEY")

		// Skip auth if no API_KEY is configured (dev mode)
		if expectedKey == "" {
			next(w, r)
			return
		}

		// Get client IP
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.Split(forwarded, ",")[0]
		}

		// Check X-API-Key header using constant-time comparison to prevent timing attacks
		apiKey := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedKey)) != 1 {
			security.LogAuthFailure(ip, r.Header.Get("User-Agent"), "Invalid API key")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		security.LogAuthSuccess(ip, r.URL.Path)
		next(w, r)
	}
}

// RateLimitMiddleware limits requests per IP address
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.Split(forwarded, ",")[0]
		}

		rateLimitMu.Lock()
		now := time.Now()

		// Reset window if expired
		if window, exists := requestWindows[ip]; !exists || now.Sub(window) > rateLimitWindow {
			requestWindows[ip] = now
			requestCounts[ip] = 0
		}

		requestCounts[ip]++
		count := requestCounts[ip]
		rateLimitMu.Unlock()

		if count > rateLimit {
			security.LogRateLimitExceeded(ip)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// CorsMiddleware adds CORS headers with restricted origins
func CorsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	allowedOrigins := getAllowedOrigins()

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		// If origin not allowed, don't set Access-Control-Allow-Origin (browser blocks)

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// XSS protection (legacy but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy for API
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// Permissions policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next(w, r)
	}
}

// SecureMiddleware chains security headers, auth, rate limiting, and CORS middleware
func SecureMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return SecurityHeadersMiddleware(CorsMiddleware(RateLimitMiddleware(AuthMiddleware(next))))
}
