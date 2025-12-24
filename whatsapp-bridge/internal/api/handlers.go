package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"whatsapp-bridge/internal/types"
)

// handleSendMessage handles the message sending API endpoint
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

	fmt.Println("Received request to send message", req.Message, req.MediaPath)

	// Send the message
	result := s.client.SendMessage(s.messageStore, req.Recipient, req.Message, req.MediaPath)
	fmt.Println("Message sent", result.Success, result.Error)

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

// handleWebhooks handles webhook CRUD operations
func (s *Server) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List all webhook configurations
		configs := s.webhookManager.GetWebhookConfigs()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    configs,
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

// handleWebhookByID handles individual webhook operations
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
			// Get specific webhook configuration
			config, err := s.messageStore.GetWebhookConfig(webhookID)
			if err != nil {
				SendJSONError(w, fmt.Sprintf("Webhook not found: %v", err), http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    config,
			})

		case http.MethodPut:
			// Update webhook configuration
			var config types.WebhookConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				SendJSONError(w, "Invalid request format", http.StatusBadRequest)
				return
			}

			config.ID = webhookID // Ensure ID matches URL

			fmt.Printf("Updating webhook %d with %d triggers\n", webhookID, len(config.Triggers))
			for i, trigger := range config.Triggers {
				fmt.Printf("  Trigger %d: type=%s, value=%s, match=%s, enabled=%t\n",
					i, trigger.TriggerType, trigger.TriggerValue, trigger.MatchType, trigger.Enabled)
			}

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

			fmt.Printf("Successfully updated webhook %d\n", webhookID)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    config,
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

// handleWebhookLogs handles webhook logs endpoint
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

// handleReaction handles emoji reactions to messages
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

// handleEditMessage handles editing previously sent messages
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

// handleDeleteMessage handles deleting/revoking messages
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

// handleGetGroupInfo handles getting group information
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

// handleMarkRead handles marking messages as read
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
