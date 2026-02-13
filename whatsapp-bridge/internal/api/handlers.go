package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"whatsapp-bridge/internal/types"
)

// handleSendMessage handles POST /api/send for sending WhatsApp messages.
//
// Request body:
//   - recipient: WhatsApp JID (required, e.g., "1234567890@s.whatsapp.net")
//   - message: Text content (required if media_path not provided)
//   - media_path: Path to media file (optional, for images/videos/documents)
//
// Response:
//   - success: boolean
//   - message_id: string (WhatsApp message ID on success)
//   - timestamp: int64 (Unix timestamp)
//   - recipient: string (echo of recipient JID)
//   - error: string (on failure)
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var req types.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Recipient == "" {
		SendJSONError(w, "Recipient is required", http.StatusBadRequest)
		return
	}

	if req.Message == "" && req.MediaPath == "" {
		SendJSONError(w, "Message or media path is required", http.StatusBadRequest)
		return
	}

	// Send the message
	result := s.client.SendMessage(s.messageStore, req.Recipient, req.Message, req.MediaPath)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Set appropriate status code
	if !result.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}

	// Send response with message_id, timestamp, recipient
	json.NewEncoder(w).Encode(types.SendMessageResponse{
		Success:   result.Success,
		Message:   result.Error,
		MessageID: result.MessageID,
		Timestamp: result.Timestamp,
		Recipient: req.Recipient,
	})
}

// handleWebhooks handles GET/POST /api/webhooks for webhook management.
//
// GET: List all webhook configurations (secrets are masked)
// POST: Create a new webhook configuration
//
// POST Request body:
//   - name: Webhook name (required)
//   - webhook_url: HTTP(S) URL to POST to (required)
//   - secret_token: HMAC-SHA256 signing secret (optional)
//   - enabled: boolean (default true)
//   - triggers: array of trigger configurations
//
// Response: { success: bool, data: WebhookConfig[] | WebhookConfig }
func (s *Server) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List all webhook configurations (with masked secrets)
		configs := s.webhookManager.GetWebhookConfigs()
		responses := make([]types.WebhookConfigResponse, len(configs))
		for i := range configs {
			responses[i] = configs[i].ToResponse()
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    responses,
		})

	case http.MethodPost:
		// Create new webhook configuration
		var config types.WebhookConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			SendJSONError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if err := s.webhookManager.ValidateWebhookConfig(&config); err != nil {
			SendJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Store configuration
		if err := s.messageStore.StoreWebhookConfig(&config); err != nil {
			SendJSONError(w, fmt.Sprintf("Failed to store webhook config: %v", err), http.StatusInternalServerError)
			return
		}

		// Reload configurations
		s.webhookManager.LoadWebhookConfigs()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    config,
		})

	default:
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWebhookByID handles operations on individual webhooks.
//
// Routes:
//   - GET    /api/webhooks/{id}        - Get webhook config
//   - PUT    /api/webhooks/{id}        - Update webhook config
//   - DELETE /api/webhooks/{id}        - Delete webhook
//   - POST   /api/webhooks/{id}/test   - Test webhook delivery
//   - GET    /api/webhooks/{id}/logs   - Get delivery logs
//   - POST   /api/webhooks/{id}/enable - Enable/disable webhook
func (s *Server) handleWebhookByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse webhook ID from URL path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/webhooks/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		SendJSONError(w, "Webhook ID is required", http.StatusBadRequest)
		return
	}

	webhookIDStr := pathParts[0]
	webhookID := 0
	if _, err := fmt.Sscanf(webhookIDStr, "%d", &webhookID); err != nil {
		SendJSONError(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	// Handle different sub-paths
	switch {
	case len(pathParts) == 1: // /api/webhooks/{id}
		switch r.Method {
		case http.MethodGet:
			// Get specific webhook configuration (with masked secret)
			config, err := s.messageStore.GetWebhookConfig(webhookID)
			if err != nil {
				SendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    config.ToResponse(),
			})

		case http.MethodPut:
			// Update webhook configuration
			var config types.WebhookConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				SendJSONError(w, "Invalid request format", http.StatusBadRequest)
				return
			}

			config.ID = webhookID // Ensure ID matches URL

			// Validate configuration
			if err := s.webhookManager.ValidateWebhookConfig(&config); err != nil {
				SendJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update configuration
			if err := s.messageStore.UpdateWebhookConfig(&config); err != nil {
				SendJSONError(w, fmt.Sprintf("Failed to update webhook config: %v", err), http.StatusInternalServerError)
				return
			}

			// Reload configurations
			s.webhookManager.LoadWebhookConfigs()

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    config.ToResponse(),
			})

		case http.MethodDelete:
			// Delete webhook configuration
			if err := s.messageStore.DeleteWebhookConfig(webhookID); err != nil {
				SendJSONError(w, fmt.Sprintf("Failed to delete webhook config: %v", err), http.StatusInternalServerError)
				return
			}

			// Reload configurations
			s.webhookManager.LoadWebhookConfigs()

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Webhook deleted successfully",
			})

		default:
			SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	case len(pathParts) == 2 && pathParts[1] == "test": // /api/webhooks/{id}/test
		if r.Method != http.MethodPost {
			SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get webhook configuration
		config, err := s.messageStore.GetWebhookConfig(webhookID)
		if err != nil {
			SendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
			return
		}

		// Test webhook
		if err := s.webhookManager.TestWebhook(config); err != nil {
			SendJSONError(w, fmt.Sprintf("Webhook test failed: %v", err), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Webhook test successful",
		})

	case len(pathParts) == 2 && pathParts[1] == "logs": // /api/webhooks/{id}/logs
		if r.Method != http.MethodGet {
			SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get webhook logs
		logs, err := s.messageStore.GetWebhookLogs(webhookID, 100) // Limit to 100 recent logs
		if err != nil {
			SendJSONError(w, fmt.Sprintf("Failed to get webhook logs: %v", err), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    logs,
		})

	case len(pathParts) == 2 && pathParts[1] == "enable": // /api/webhooks/{id}/enable
		if r.Method != http.MethodPost {
			SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request body to get enabled status
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendJSONError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Get current config
		config, err := s.messageStore.GetWebhookConfig(webhookID)
		if err != nil {
			SendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
			return
		}

		// Update enabled status
		config.Enabled = req.Enabled
		if err := s.messageStore.UpdateWebhookConfig(config); err != nil {
			SendJSONError(w, fmt.Sprintf("Failed to update webhook config: %v", err), http.StatusInternalServerError)
			return
		}

		// Reload configurations
		s.webhookManager.LoadWebhookConfigs()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Webhook %s successfully", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]),
			"data":    config,
		})

	default:
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWebhookLogs handles GET /api/webhook-logs for all webhook delivery logs.
//
// Returns the last 100 webhook delivery attempts across all webhooks.
// For logs of a specific webhook, use GET /api/webhooks/{id}/logs instead.
func (s *Server) handleWebhookLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get all webhook logs
	logs, err := s.messageStore.GetWebhookLogs(0, 100) // Get last 100 logs for all webhooks
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to get webhook logs: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    logs,
	})
}

// handleReaction handles POST /api/reaction for sending emoji reactions.
//
// Request body:
//   - chat_jid: Chat containing the message (required)
//   - message_id: Target message ID (required)
//   - emoji: Reaction emoji (empty string to remove reaction)
//
// Response: { success: bool, message: string }
func (s *Server) handleReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.ReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" || req.MessageID == "" {
		SendJSONError(w, "chat_jid and message_id are required", http.StatusBadRequest)
		return
	}

	if err := s.client.SendReaction(req.ChatJID, req.MessageID, req.Emoji); err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to send reaction: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Reaction sent",
	})
}

// handleEditMessage handles POST /api/edit for editing sent messages.
//
// Request body:
//   - chat_jid: Chat containing the message (required)
//   - message_id: Message to edit (required, must be your own message)
//   - new_content: Updated message text (required)
//
// Response: { success: bool, message: string }
func (s *Server) handleEditMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.EditMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" || req.MessageID == "" || req.NewContent == "" {
		SendJSONError(w, "chat_jid, message_id, and new_content are required", http.StatusBadRequest)
		return
	}

	if err := s.client.EditMessage(req.ChatJID, req.MessageID, req.NewContent); err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to edit message: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message edited",
	})
}

// handleDeleteMessage handles POST /api/delete for revoking messages.
//
// Request body:
//   - chat_jid: Chat containing the message (required)
//   - message_id: Message to delete (required)
//   - sender_jid: Original sender (required for group admin revoking others' messages)
//
// Response: { success: bool, message: string }
func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.DeleteMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" || req.MessageID == "" {
		SendJSONError(w, "chat_jid and message_id are required", http.StatusBadRequest)
		return
	}

	if err := s.client.DeleteMessage(req.ChatJID, req.MessageID, req.SenderJID); err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to delete message: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Message deleted",
	})
}

// handleGetGroupInfo handles GET /api/group/{jid} for group metadata.
//
// URL parameter:
//   - jid: Group JID (e.g., "123456789@g.us")
//
// Response includes: jid, name, topic, owner_jid, participant_count,
// participants (with is_admin, is_owner flags), created_at
func (s *Server) handleGetGroupInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse group JID from URL path: /api/group/{jid}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/group/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		SendJSONError(w, "Group JID is required", http.StatusBadRequest)
		return
	}

	groupJID := pathParts[0]

	groupInfo, err := s.client.GetGroupInfo(groupJID)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to get group info: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert participants to a more JSON-friendly format
	participants := make([]map[string]interface{}, len(groupInfo.Participants))
	for i, p := range groupInfo.Participants {
		participants[i] = map[string]interface{}{
			"jid":      p.JID.String(),
			"is_admin": p.IsAdmin,
			"is_owner": p.IsSuperAdmin,
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"jid":               groupInfo.JID.String(),
			"name":              groupInfo.Name,
			"topic":             groupInfo.Topic,
			"owner_jid":         groupInfo.OwnerJID.String(),
			"participant_count": len(groupInfo.Participants),
			"participants":      participants,
			"created_at":        groupInfo.GroupCreated,
		},
	})
}

// handleMarkRead handles POST /api/read for sending read receipts (blue ticks).
//
// Request body:
//   - chat_jid: Chat containing the messages (required)
//   - message_ids: Array of message IDs to mark read (required)
//   - sender_jid: Sender JID (required for group chats)
//
// Response: { success: bool, message: string }
func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" || len(req.MessageIDs) == 0 {
		SendJSONError(w, "chat_jid and message_ids are required", http.StatusBadRequest)
		return
	}

	if err := s.client.MarkMessagesRead(req.ChatJID, req.MessageIDs, req.SenderJID); err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to mark messages as read: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Messages marked as read",
	})
}

// Phase 2: Group Management Handlers

// handleCreateGroup handles POST /api/group/create for creating WhatsApp groups.
//
// Request body:
//   - name: Group name (required)
//   - participants: Array of JIDs to add (optional)
//
// Response: { success: bool, group_jid: string, name: string }
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		SendJSONError(w, "Group name is required", http.StatusBadRequest)
		return
	}

	groupInfo, err := s.client.CreateGroup(req.Name, req.Participants)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to create group: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"group_jid": groupInfo.JID.String(),
		"name":      groupInfo.Name,
	})
}

// handleAddGroupMembers handles POST /api/group/add for adding group members.
//
// Request body:
//   - group_jid: Target group (required)
//   - participants: Array of JIDs to add (required)
//
// Response: { success: bool, participants: [{jid, error}] }
func (s *Server) handleAddGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.GroupParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.GroupJID == "" || len(req.Participants) == 0 {
		SendJSONError(w, "group_jid and participants are required", http.StatusBadRequest)
		return
	}

	results, err := s.client.AddGroupParticipants(req.GroupJID, req.Participants)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to add members: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert results to JSON-friendly format
	added := make([]map[string]interface{}, len(results))
	for i, p := range results {
		added[i] = map[string]interface{}{
			"jid":   p.JID.String(),
			"error": p.Error,
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"participants": added,
	})
}

// handleRemoveGroupMembers handles POST /api/group/remove for removing members.
//
// Request body:
//   - group_jid: Target group (required)
//   - participants: Array of JIDs to remove (required)
//
// Response: { success: bool, participants: [{jid, error}] }
func (s *Server) handleRemoveGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.GroupParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.GroupJID == "" || len(req.Participants) == 0 {
		SendJSONError(w, "group_jid and participants are required", http.StatusBadRequest)
		return
	}

	results, err := s.client.RemoveGroupParticipants(req.GroupJID, req.Participants)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to remove members: %v", err), http.StatusInternalServerError)
		return
	}

	removed := make([]map[string]interface{}, len(results))
	for i, p := range results {
		removed[i] = map[string]interface{}{
			"jid":   p.JID.String(),
			"error": p.Error,
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"participants": removed,
	})
}

// handlePromoteAdmin handles POST /api/group/promote for promoting to admin.
//
// Request body:
//   - group_jid: Target group (required)
//   - participant: JID to promote (required)
//
// Response: { success: bool, group_jid, participant, action: "promoted" }
func (s *Server) handlePromoteAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.GroupAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.GroupJID == "" || req.Participant == "" {
		SendJSONError(w, "group_jid and participant are required", http.StatusBadRequest)
		return
	}

	_, err := s.client.PromoteGroupParticipant(req.GroupJID, req.Participant)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to promote admin: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"group_jid":   req.GroupJID,
		"participant": req.Participant,
		"action":      "promoted",
	})
}

// handleDemoteAdmin handles POST /api/group/demote for demoting admins.
//
// Request body:
//   - group_jid: Target group (required)
//   - participant: Admin JID to demote (required)
//
// Response: { success: bool, group_jid, participant, action: "demoted" }
func (s *Server) handleDemoteAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.GroupAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.GroupJID == "" || req.Participant == "" {
		SendJSONError(w, "group_jid and participant are required", http.StatusBadRequest)
		return
	}

	_, err := s.client.DemoteGroupParticipant(req.GroupJID, req.Participant)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to demote admin: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"group_jid":   req.GroupJID,
		"participant": req.Participant,
		"action":      "demoted",
	})
}

// handleLeaveGroup handles POST /api/group/leave for leaving a group.
//
// Request body:
//   - group_jid: Group to leave (required)
//
// Response: { success: bool, group_jid, action: "left" }
func (s *Server) handleLeaveGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.LeaveGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.GroupJID == "" {
		SendJSONError(w, "group_jid is required", http.StatusBadRequest)
		return
	}

	err := s.client.LeaveGroup(req.GroupJID)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to leave group: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"group_jid": req.GroupJID,
		"action":    "left",
	})
}

// handleUpdateGroup handles POST /api/group/update for modifying group settings.
//
// Request body:
//   - group_jid: Target group (required)
//   - name: New group name (optional)
//   - topic: New group description (optional)
//
// At least one of name or topic must be provided.
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.GroupJID == "" {
		SendJSONError(w, "group_jid is required", http.StatusBadRequest)
		return
	}

	if req.Name == "" && req.Topic == "" {
		SendJSONError(w, "name or topic is required", http.StatusBadRequest)
		return
	}

	var errors []string

	if req.Name != "" {
		if err := s.client.SetGroupName(req.GroupJID, req.Name); err != nil {
			errors = append(errors, fmt.Sprintf("name: %v", err))
		}
	}

	if req.Topic != "" {
		if err := s.client.SetGroupTopic(req.GroupJID, req.Topic); err != nil {
			errors = append(errors, fmt.Sprintf("topic: %v", err))
		}
	}

	if len(errors) > 0 {
		SendJSONError(w, fmt.Sprintf("Partial failure: %v", errors), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"group_jid": req.GroupJID,
	})
}

// Phase 3: Polls

// handleCreatePoll handles POST /api/poll for creating WhatsApp polls.
//
// Request body:
//   - chat_jid: Target chat (required)
//   - question: Poll question (required)
//   - options: Array of 2-12 answer options (required)
//   - multi_select: Allow multiple selections (default false)
//
// Response: { success, message_id, timestamp, chat_jid, question, options }
func (s *Server) handleCreatePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.CreatePollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" || req.Question == "" || len(req.Options) < 2 {
		SendJSONError(w, "chat_jid, question, and at least 2 options are required", http.StatusBadRequest)
		return
	}

	if len(req.Options) > 12 {
		SendJSONError(w, "Maximum 12 options allowed", http.StatusBadRequest)
		return
	}

	result, err := s.client.CreatePoll(req.ChatJID, req.Question, req.Options, req.MultiSelect)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to create poll: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    result.Success,
		"message_id": result.MessageID,
		"timestamp":  result.Timestamp,
		"chat_jid":   req.ChatJID,
		"question":   req.Question,
		"options":    req.Options,
	})
}

// Phase 4: History Sync

// handleRequestHistory handles POST /api/history for requesting older messages.
//
// Request body:
//   - chat_jid: Target chat (required)
//   - oldest_msg_id: ID of oldest known message (required)
//   - oldest_msg_timestamp: Unix timestamp in ms of oldest message (required)
//   - oldest_msg_from_me: Whether oldest message was sent by you (default false)
//   - count: Number of messages to request (max 50, default 50)
//
// Note: Messages arrive asynchronously via HistorySync events.
func (s *Server) handleRequestHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.RequestHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" || req.OldestMsgID == "" || req.OldestMsgTimestamp == 0 {
		SendJSONError(w, "chat_jid, oldest_msg_id, and oldest_msg_timestamp are required", http.StatusBadRequest)
		return
	}

	if req.Count <= 0 || req.Count > 50 {
		req.Count = 50
	}

	err := s.client.RequestChatHistory(req.ChatJID, req.OldestMsgID, req.OldestMsgFromMe, req.OldestMsgTimestamp, req.Count)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to request history: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "History request sent. Messages will arrive via HistorySync event.",
		"chat_jid": req.ChatJID,
		"count":    req.Count,
	})
}

// Phase 5: Advanced Features

// handleSetPresence handles POST /api/presence for setting online status.
//
// Request body:
//   - presence: "available" or "unavailable" (required)
//
// Response: { success: bool, presence: string }
func (s *Server) handleSetPresence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.SetPresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Presence == "" {
		SendJSONError(w, "presence is required ('available' or 'unavailable')", http.StatusBadRequest)
		return
	}

	err := s.client.SetPresence(req.Presence)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to set presence: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"presence": req.Presence,
	})
}

// handleSubscribePresence handles POST /api/presence/subscribe for presence updates.
//
// Request body:
//   - jid: Contact JID to subscribe to (required)
//
// After subscribing, presence events will arrive via event handlers.
func (s *Server) handleSubscribePresence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.SubscribePresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.JID == "" {
		SendJSONError(w, "jid is required", http.StatusBadRequest)
		return
	}

	err := s.client.SubscribeToPresence(req.JID)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to subscribe to presence: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"jid":     req.JID,
		"message": "Subscribed to presence updates. Use event handler to receive updates.",
	})
}

// handleGetProfilePicture handles GET/POST /api/profile-picture for avatars.
//
// GET query params or POST body:
//   - jid: User or group JID (required)
//   - preview: Return thumbnail instead of full image (default false)
//
// Response: { success, jid, has_picture, url, id, type, direct_path }
func (s *Server) handleGetProfilePicture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var jid string
	var preview bool

	if r.Method == http.MethodGet {
		jid = r.URL.Query().Get("jid")
		preview = r.URL.Query().Get("preview") == "true"
	} else {
		var req types.GetProfilePictureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendJSONError(w, "Invalid request format", http.StatusBadRequest)
			return
		}
		jid = req.JID
		preview = req.Preview
	}

	if jid == "" {
		SendJSONError(w, "jid is required", http.StatusBadRequest)
		return
	}

	info, err := s.client.GetProfilePicture(jid, preview)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to get profile picture: %v", err), http.StatusInternalServerError)
		return
	}

	if info == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"jid":         jid,
			"has_picture": false,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"jid":         jid,
		"has_picture": true,
		"url":         info.URL,
		"id":          info.ID,
		"type":        info.Type,
		"direct_path": info.DirectPath,
	})
}

// handleGetBlocklist handles GET /api/blocklist for listing blocked users.
//
// Response: { success: bool, users: [{jid}], count: int }
func (s *Server) handleGetBlocklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	users, err := s.client.GetBlockedUsers()
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to get blocklist: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"users":   users,
		"count":   len(users),
	})
}

// handleUpdateBlocklist handles POST /api/blocklist for blocking/unblocking.
//
// Request body:
//   - jid: User JID to block/unblock (required)
//   - action: "block" or "unblock" (required)
//
// Response: { success: bool, jid, action }
func (s *Server) handleUpdateBlocklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.BlocklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.JID == "" || req.Action == "" {
		SendJSONError(w, "jid and action ('block' or 'unblock') are required", http.StatusBadRequest)
		return
	}

	err := s.client.UpdateBlockedUser(req.JID, req.Action)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to update blocklist: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"jid":     req.JID,
		"action":  req.Action,
	})
}

// handleFollowNewsletter handles POST /api/newsletter/follow for joining channels.
//
// Request body:
//   - jid: Newsletter/channel JID (required)
//
// Response: { success: bool, jid, message }
func (s *Server) handleFollowNewsletter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.NewsletterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.JID == "" {
		SendJSONError(w, "jid is required", http.StatusBadRequest)
		return
	}

	err := s.client.FollowNewsletterChannel(req.JID)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to follow newsletter: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"jid":     req.JID,
		"message": "Successfully followed newsletter",
	})
}

// handleUnfollowNewsletter handles POST /api/newsletter/unfollow for leaving channels.
//
// Request body:
//   - jid: Newsletter/channel JID (required)
//
// Response: { success: bool, jid, message }
func (s *Server) handleUnfollowNewsletter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.NewsletterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.JID == "" {
		SendJSONError(w, "jid is required", http.StatusBadRequest)
		return
	}

	err := s.client.UnfollowNewsletterChannel(req.JID)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to unfollow newsletter: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"jid":     req.JID,
		"message": "Successfully unfollowed newsletter",
	})
}

// handleCreateNewsletter handles POST /api/newsletter/create for new channels.
//
// Request body:
//   - name: Newsletter name (required)
//   - description: Newsletter description (optional)
//
// Response: { success: bool, jid, name, description }
func (s *Server) handleCreateNewsletter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.CreateNewsletterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		SendJSONError(w, "name is required", http.StatusBadRequest)
		return
	}

	info, err := s.client.CreateNewsletterChannel(req.Name, req.Description)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to create newsletter: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"jid":         info.JID,
		"name":        info.Name,
		"description": info.Description,
	})
}

// Phase 6: Chat Features

// handleSendTyping handles POST /api/typing for sending typing indicators.
//
// Request body:
//   - chat_jid: Target chat (required)
//   - state: "typing", "paused", or "recording" (default: "typing")
//
// Response: { success: bool, chat_jid, state }
func (s *Server) handleSendTyping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.SendTypingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" {
		SendJSONError(w, "chat_jid is required", http.StatusBadRequest)
		return
	}

	// Default to "typing" if not specified
	if req.State == "" {
		req.State = "typing"
	}

	err := s.client.SendTypingIndicator(req.ChatJID, req.State)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to send typing indicator: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"chat_jid": req.ChatJID,
		"state":    req.State,
	})
}

// handleSetAbout handles POST /api/set-about for updating profile status text.
//
// Request body:
//   - text: The new "About" text for the profile (required)
//
// Response: { success: bool, text: string }
func (s *Server) handleSetAbout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.SetAboutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	err := s.client.SetAboutText(req.Text)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to set about text: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"text":    req.Text,
	})
}

// handleSetDisappearingTimer handles POST /api/disappearing for setting disappearing messages timer.
//
// Request body:
//   - chat_jid: Target chat JID (required)
//   - duration: "off", "24h", "7d", or "90d" (required)
//
// Response: { success: bool, chat_jid, duration }
func (s *Server) handleSetDisappearingTimer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.SetDisappearingTimerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" {
		SendJSONError(w, "chat_jid is required", http.StatusBadRequest)
		return
	}

	if req.Duration == "" {
		SendJSONError(w, "duration is required", http.StatusBadRequest)
		return
	}

	err := s.client.SetDisappearingTimer(req.ChatJID, req.Duration)
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to set disappearing timer: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"chat_jid": req.ChatJID,
		"duration": req.Duration,
	})
}

// handleGetPrivacySettings handles GET /api/privacy for fetching privacy settings.
//
// Response: { success: bool, settings: { group_add, last_seen, status, profile, read_receipts, call_add, online } }
func (s *Server) handleGetPrivacySettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	settings, err := s.client.GetPrivacySettings()
	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to fetch privacy settings: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"settings": settings,
	})
}

// handlePinChat handles POST /api/pin for pinning/unpinning chats.
//
// Request body:
//   - chat_jid: Target chat JID (required)
//   - pin: true to pin, false to unpin (required)
//
// Response: { success: bool, chat_jid, pin: bool }
func (s *Server) handlePinChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.PinChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" {
		SendJSONError(w, "chat_jid is required", http.StatusBadRequest)
		return
	}

	var err error
	if req.Pin {
		err = s.client.PinChat(req.ChatJID)
	} else {
		err = s.client.UnpinChat(req.ChatJID)
	}

	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to pin chat: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"chat_jid": req.ChatJID,
		"pin":      req.Pin,
	})
}

// handleMuteChat handles POST /api/mute for muting/unmuting chats.
//
// Request body:
//   - chat_jid: Target chat JID (required)
//   - mute: true to mute, false to unmute (required)
//   - duration: "forever", "15m", "1h", "8h", "1w" (required if mute=true)
//
// Response: { success: bool, chat_jid, mute: bool, duration?: string }
func (s *Server) handleMuteChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.MuteChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" {
		SendJSONError(w, "chat_jid is required", http.StatusBadRequest)
		return
	}

	var err error
	if req.Mute {
		if req.Duration == "" {
			SendJSONError(w, "duration is required when muting", http.StatusBadRequest)
			return
		}
		err = s.client.MuteChat(req.ChatJID, req.Duration)
	} else {
		err = s.client.UnmuteChat(req.ChatJID)
	}

	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to mute chat: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"chat_jid": req.ChatJID,
		"mute":     req.Mute,
	}
	if req.Mute {
		response["duration"] = req.Duration
	}

	json.NewEncoder(w).Encode(response)
}

// handleArchiveChat handles POST /api/archive for archiving/unarchiving chats.
//
// Request body:
//   - chat_jid: Target chat JID (required)
//   - archive: true to archive, false to unarchive (required)
//
// Response: { success: bool, chat_jid, archive: bool }
func (s *Server) handleArchiveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req types.ArchiveChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.ChatJID == "" {
		SendJSONError(w, "chat_jid is required", http.StatusBadRequest)
		return
	}

	var err error
	if req.Archive {
		err = s.client.ArchiveChat(req.ChatJID)
	} else {
		err = s.client.UnarchiveChat(req.ChatJID)
	}

	if err != nil {
		SendJSONError(w, fmt.Sprintf("Failed to archive chat: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"chat_jid": req.ChatJID,
		"archive":  req.Archive,
	})
}

// handlePairPhone initiates phone number pairing
// POST /api/pair
// Request: { phone_number: string }
// Response: { success: bool, code: string, expires_in: int, error?: string }
func (s *Server) handlePairPhone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.PairPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PhoneNumber == "" {
		SendJSONError(w, "phone_number is required", http.StatusBadRequest)
		return
	}

	code, err := s.client.PairWithPhone(req.PhoneNumber)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(types.PairPhoneResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.PairPhoneResponse{
		Success:   true,
		Code:      code,
		ExpiresIn: 160,
	})
}

// handlePairingStatus returns current pairing state
// GET /api/pairing
// Response: { success: bool, in_progress: bool, code?: string, expires_in?: int, complete: bool, error?: string }
func (s *Server) handlePairingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	inProgress, code, expiresIn, complete, err := s.client.GetPairingStatus()

	resp := types.PairingStatusResponse{
		Success:    true,
		InProgress: inProgress,
		Code:       code,
		ExpiresIn:  expiresIn,
		Complete:   complete,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleHealth returns 200 if connected, 503 if not. No auth required.
// GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connected := s.client.IsConnected()
	startedAt, lastConn, discAt, reconnErrs := s.client.ConnectionState()

	resp := map[string]interface{}{
		"connected":      connected,
		"uptime":         time.Since(startedAt).Round(time.Second).String(),
		"reconnect_errs": reconnErrs,
	}
	if !lastConn.IsZero() {
		resp["last_connected"] = lastConn.Format(time.RFC3339)
	}
	if !discAt.IsZero() {
		resp["disconnected_for"] = time.Since(discAt).Round(time.Second).String()
	}

	w.Header().Set("Content-Type", "application/json")
	if !connected {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}

// handleConnectionStatus returns WhatsApp connection state
// GET /api/connection
func (s *Server) handleConnectionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connected := s.client.IsConnected()
	linked := s.client.Store.ID != nil
	startedAt, lastConn, discAt, reconnErrs := s.client.ConnectionState()

	resp := types.ConnectionStatusResponse{
		Success:             true,
		Connected:           connected,
		Linked:              linked,
		Uptime:              time.Since(startedAt).Round(time.Second).String(),
		AutoReconnectErrors: reconnErrs,
	}

	if linked {
		resp.JID = s.client.Store.ID.String()
	}
	if !lastConn.IsZero() {
		resp.LastConnected = lastConn.Format(time.RFC3339)
	}
	if !discAt.IsZero() {
		resp.DisconnectedFor = time.Since(discAt).Round(time.Second).String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleSyncStatus returns current sync state and recommendations
// GET /api/sync-status
func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get message count from database
	msgCount := 0 // TODO: implement in messageStore
	
	resp := types.SyncStatusResponse{
		Success:       true,
		Syncing:       false, // TODO: track from HistorySync events
		SyncProgress:  100,
		MessageCount:  msgCount,
		ConversationCount: 0, // TODO: implement in messageStore
	}

	// Provide sync troubleshooting recommendations
	resp.Recommendations = []string{
		"If sync is still syncing, wait 2-3 minutes for completion",
		"Ensure phone has stable internet throughout sync process",
		"Do not switch WiFi/data networks during sync",
		"If sync fails, clear WhatsApp cache and restart phone",
		"Check linked device status in WhatsApp Settings > Linked devices",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
