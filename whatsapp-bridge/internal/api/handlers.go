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

// handleWebhooks handles webhook CRUD operations
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

// Phase 2: Group Management Handlers

// handleCreateGroup handles creating a new group
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

// handleAddGroupMembers handles adding members to a group
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

// handleRemoveGroupMembers handles removing members from a group
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

// handlePromoteAdmin handles promoting a member to admin
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

// handleDemoteAdmin handles demoting an admin to regular member
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

// handleLeaveGroup handles leaving a group
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

// handleUpdateGroup handles updating group name/topic
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

// handleCreatePoll handles creating a new poll
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

// handleRequestHistory handles on-demand history requests
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

// handleSetPresence handles setting own presence (available/unavailable)
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

// handleSubscribePresence handles subscribing to a contact's presence
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

// handleGetProfilePicture handles getting a profile picture URL
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
			"success": true,
			"jid":     jid,
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

// handleGetBlocklist handles getting the list of blocked users
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

// handleUpdateBlocklist handles blocking/unblocking a user
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

// handleFollowNewsletter handles following a newsletter/channel
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

// handleUnfollowNewsletter handles unfollowing a newsletter/channel
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

// handleCreateNewsletter handles creating a new newsletter/channel
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
