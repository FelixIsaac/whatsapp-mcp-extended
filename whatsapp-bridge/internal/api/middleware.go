package api

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
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

		// Check X-API-Key header using constant-time comparison to prevent timing attacks
		apiKey := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedKey)) != 1 {
			log.Printf("SECURITY: Unauthorized request from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

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
			log.Printf("SECURITY: Rate limit exceeded for %s", ip)
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

// SecureMiddleware chains auth, rate limiting, and CORS middleware
func SecureMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return CorsMiddleware(RateLimitMiddleware(AuthMiddleware(next)))
}
