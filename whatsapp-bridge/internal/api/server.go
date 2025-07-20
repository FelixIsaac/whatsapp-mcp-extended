package api

import (
	"fmt"
	"net/http"

	"whatsapp-bridge/internal/database"
	"whatsapp-bridge/internal/webhook"
	"whatsapp-bridge/internal/whatsapp"
)

// Server represents the HTTP API server
type Server struct {
	client         *whatsapp.Client
	messageStore   *database.MessageStore
	webhookManager *webhook.Manager
	port           int
}

// NewServer creates a new API server instance
func NewServer(client *whatsapp.Client, messageStore *database.MessageStore, webhookManager *webhook.Manager, port int) *Server {
	return &Server{
		client:         client,
		messageStore:   messageStore,
		webhookManager: webhookManager,
		port:           port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() {
	// Register handlers
	s.registerHandlers()

	// Start the server
	serverAddr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Starting REST API server on %s...\n", serverAddr)

	// Run server in a goroutine so it doesn't block
	go func() {
		if err := http.ListenAndServe(serverAddr, nil); err != nil {
			fmt.Printf("REST API server error: %v\n", err)
		}
	}()
}

// registerHandlers registers all HTTP handlers
func (s *Server) registerHandlers() {
	// Message sending endpoint
	http.HandleFunc("/api/send", CorsMiddleware(s.handleSendMessage))

	// Webhook management endpoints
	http.HandleFunc("/api/webhooks", CorsMiddleware(s.handleWebhooks))
	http.HandleFunc("/api/webhooks/", CorsMiddleware(s.handleWebhookByID))
	http.HandleFunc("/api/webhook-logs", CorsMiddleware(s.handleWebhookLogs))
}
