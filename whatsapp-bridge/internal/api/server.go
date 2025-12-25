package api

import (
	"fmt"
	"net/http"

	"whatsapp-bridge/internal/database"
	"whatsapp-bridge/internal/webhook"
	"whatsapp-bridge/internal/whatsapp"
)

// Server is the HTTP REST API server for the WhatsApp bridge.
// It exposes endpoints for sending messages, managing webhooks,
// group operations, and other WhatsApp features.
type Server struct {
	client         *whatsapp.Client
	messageStore   *database.MessageStore
	webhookManager *webhook.Manager
	port           int
}

// NewServer creates a new API server with the given dependencies.
//
// Parameters:
//   - client: WhatsApp client for sending messages and interacting with WhatsApp
//   - messageStore: Database for message history and webhook configurations
//   - webhookManager: Manager for webhook trigger matching and delivery
//   - port: TCP port to listen on (e.g., 8080)
func NewServer(client *whatsapp.Client, messageStore *database.MessageStore, webhookManager *webhook.Manager, port int) *Server {
	return &Server{
		client:         client,
		messageStore:   messageStore,
		webhookManager: webhookManager,
		port:           port,
	}
}

// Start launches the HTTP server in a background goroutine.
// The server listens on the configured port and serves the REST API.
// This method returns immediately; use a blocking mechanism in main().
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

// registerHandlers sets up all API routes with security middleware.
// All endpoints are protected by SecureMiddleware which enforces:
// API key authentication, rate limiting, CORS, and security headers.
func (s *Server) registerHandlers() {
	// Message sending endpoint
	http.HandleFunc("/api/send", SecureMiddleware(s.handleSendMessage))

	// Webhook management endpoints
	http.HandleFunc("/api/webhooks", SecureMiddleware(s.handleWebhooks))
	http.HandleFunc("/api/webhooks/", SecureMiddleware(s.handleWebhookByID))
	http.HandleFunc("/api/webhook-logs", SecureMiddleware(s.handleWebhookLogs))

	// Phase 1 features: Reactions, Edit, Delete, Group Info, Mark Read
	http.HandleFunc("/api/reaction", SecureMiddleware(s.handleReaction))
	http.HandleFunc("/api/edit", SecureMiddleware(s.handleEditMessage))
	http.HandleFunc("/api/delete", SecureMiddleware(s.handleDeleteMessage))
	http.HandleFunc("/api/group/", SecureMiddleware(s.handleGetGroupInfo))
	http.HandleFunc("/api/read", SecureMiddleware(s.handleMarkRead))

	// Phase 2: Group Management
	http.HandleFunc("/api/group/create", SecureMiddleware(s.handleCreateGroup))
	http.HandleFunc("/api/group/add-members", SecureMiddleware(s.handleAddGroupMembers))
	http.HandleFunc("/api/group/remove-members", SecureMiddleware(s.handleRemoveGroupMembers))
	http.HandleFunc("/api/group/promote", SecureMiddleware(s.handlePromoteAdmin))
	http.HandleFunc("/api/group/demote", SecureMiddleware(s.handleDemoteAdmin))
	http.HandleFunc("/api/group/leave", SecureMiddleware(s.handleLeaveGroup))
	http.HandleFunc("/api/group/update", SecureMiddleware(s.handleUpdateGroup))

	// Phase 3: Polls
	http.HandleFunc("/api/poll/create", SecureMiddleware(s.handleCreatePoll))

	// Phase 4: History Sync
	http.HandleFunc("/api/history/request", SecureMiddleware(s.handleRequestHistory))

	// Phase 5: Advanced Features
	http.HandleFunc("/api/presence/set", SecureMiddleware(s.handleSetPresence))
	http.HandleFunc("/api/presence/subscribe", SecureMiddleware(s.handleSubscribePresence))
	http.HandleFunc("/api/profile-picture", SecureMiddleware(s.handleGetProfilePicture))
	http.HandleFunc("/api/blocklist", SecureMiddleware(s.handleGetBlocklist))
	http.HandleFunc("/api/blocklist/update", SecureMiddleware(s.handleUpdateBlocklist))
	http.HandleFunc("/api/newsletter/follow", SecureMiddleware(s.handleFollowNewsletter))
	http.HandleFunc("/api/newsletter/unfollow", SecureMiddleware(s.handleUnfollowNewsletter))
	http.HandleFunc("/api/newsletter/create", SecureMiddleware(s.handleCreateNewsletter))

	// Phase 6: Chat Features
	http.HandleFunc("/api/typing", SecureMiddleware(s.handleSendTyping))
	http.HandleFunc("/api/set-about", SecureMiddleware(s.handleSetAbout))
	http.HandleFunc("/api/disappearing", SecureMiddleware(s.handleSetDisappearingTimer))
	http.HandleFunc("/api/privacy", SecureMiddleware(s.handleGetPrivacySettings))
	http.HandleFunc("/api/pin", SecureMiddleware(s.handlePinChat))
}
