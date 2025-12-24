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

	// Phase 1 features: Reactions, Edit, Delete, Group Info, Mark Read
	http.HandleFunc("/api/reaction", CorsMiddleware(s.handleReaction))
	http.HandleFunc("/api/edit", CorsMiddleware(s.handleEditMessage))
	http.HandleFunc("/api/delete", CorsMiddleware(s.handleDeleteMessage))
	http.HandleFunc("/api/group/", CorsMiddleware(s.handleGetGroupInfo))
	http.HandleFunc("/api/read", CorsMiddleware(s.handleMarkRead))

	// Phase 2: Group Management
	http.HandleFunc("/api/group/create", CorsMiddleware(s.handleCreateGroup))
	http.HandleFunc("/api/group/add-members", CorsMiddleware(s.handleAddGroupMembers))
	http.HandleFunc("/api/group/remove-members", CorsMiddleware(s.handleRemoveGroupMembers))
	http.HandleFunc("/api/group/promote", CorsMiddleware(s.handlePromoteAdmin))
	http.HandleFunc("/api/group/demote", CorsMiddleware(s.handleDemoteAdmin))
	http.HandleFunc("/api/group/leave", CorsMiddleware(s.handleLeaveGroup))
	http.HandleFunc("/api/group/update", CorsMiddleware(s.handleUpdateGroup))

	// Phase 3: Polls
	http.HandleFunc("/api/poll/create", CorsMiddleware(s.handleCreatePoll))

	// Phase 4: History Sync
	http.HandleFunc("/api/history/request", CorsMiddleware(s.handleRequestHistory))

	// Phase 5: Advanced Features
	http.HandleFunc("/api/presence/set", CorsMiddleware(s.handleSetPresence))
	http.HandleFunc("/api/presence/subscribe", CorsMiddleware(s.handleSubscribePresence))
	http.HandleFunc("/api/profile-picture", CorsMiddleware(s.handleGetProfilePicture))
	http.HandleFunc("/api/blocklist", CorsMiddleware(s.handleGetBlocklist))
	http.HandleFunc("/api/blocklist/update", CorsMiddleware(s.handleUpdateBlocklist))
	http.HandleFunc("/api/newsletter/follow", CorsMiddleware(s.handleFollowNewsletter))
	http.HandleFunc("/api/newsletter/unfollow", CorsMiddleware(s.handleUnfollowNewsletter))
	http.HandleFunc("/api/newsletter/create", CorsMiddleware(s.handleCreateNewsletter))
}
