package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"whatsapp-bridge/internal/api"
	"whatsapp-bridge/internal/config"
	"whatsapp-bridge/internal/database"
	"whatsapp-bridge/internal/webhook"
	"whatsapp-bridge/internal/whatsapp"
)

func main() {
	// Set up logger
	logger := waLog.Stdout("Client", "INFO", true)
	logger.Infof("Starting WhatsApp client...")

	// Security: Require API_KEY in production
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		if os.Getenv("DISABLE_AUTH_CHECK") != "true" {
			logger.Errorf("SECURITY: API_KEY environment variable is required")
			logger.Errorf("Set API_KEY or DISABLE_AUTH_CHECK=true for development")
			os.Exit(1)
		}
		logger.Warnf("WARNING: Running without API authentication (DISABLE_AUTH_CHECK=true)")
	} else {
		logger.Infof("API authentication enabled")
	}

	// Load configuration
	cfg := config.NewConfig()

	// Initialize database
	messageStore, err := database.NewMessageStore()
	if err != nil {
		logger.Errorf("Failed to initialize message store: %v", err)
		os.Exit(1)
	}
	defer messageStore.Close()

	// Create WhatsApp client with config (Phase 4: HistorySyncConfig)
	client, err := whatsapp.NewClientWithConfig(logger, cfg)
	if err != nil {
		logger.Errorf("Failed to create WhatsApp client: %v", err)
		os.Exit(1)
	}

	// Initialize webhook manager
	webhookManager := webhook.NewManager(messageStore, logger)
	err = webhookManager.LoadWebhookConfigs()
	if err != nil {
		logger.Errorf("Failed to load webhook configs: %v", err)
		os.Exit(1)
	}

	// Track active calls for duration calculation and missed/answered status
	type activeCall struct {
		ChatJID   string
		Sender    string
		Name      string
		Timestamp time.Time
	}
	var (
		activeCalls   = make(map[string]*activeCall) // CallID -> activeCall
		activeCallsMu sync.Mutex
	)

	// Cleanup stale calls that never received a terminate/reject event (e.g. network drop)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			activeCallsMu.Lock()
			for id, call := range activeCalls {
				if time.Since(call.Timestamp) > 5*time.Minute {
					delete(activeCalls, id)
					logger.Debugf("[CALL] Cleaned up stale call %s", id)
				}
			}
			activeCallsMu.Unlock()
		}
	}()

	// resolveCallJID resolves a LID JID to a regular phone number JID for call events
	resolveCallJID := func(jid types.JID) types.JID {
		if jid.Server == types.HiddenUserServer {
			resolved, err := client.Store.GetAltJID(context.Background(), jid)
			if err == nil && !resolved.IsEmpty() {
				logger.Infof("[CALL] Resolved LID %s → %s", jid, resolved)
				return resolved
			}
			logger.Warnf("[CALL] Could not resolve LID %s: %v", jid, err)
		}
		return jid
	}

	// Setup event handling for messages and history sync
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Process regular messages with webhook support
			client.HandleMessage(messageStore, webhookManager, v)

		case *events.HistorySync:
			// Process history sync events with detailed logging
			logger.Infof("[SYNC] Starting HistorySync (Type: %v, Conversations: %d)", v.Data.SyncType, len(v.Data.Conversations))
			client.HandleHistorySync(messageStore, v)
			logger.Infof("[SYNC] ✓ Completed (Type: %v, %d conversations)", v.Data.SyncType, len(v.Data.Conversations))

		case *events.Connected:
			client.MarkConnected()
			// Send presence to keep session active and receive real-time messages
			if err := client.SetPresence("available"); err != nil {
				logger.Warnf("Failed to set presence: %v", err)
			} else {
				logger.Infof("✓ Presence set to available")
			}
			logger.Infof("✓ Connected to WhatsApp")

		case *events.LoggedOut:
			logger.Warnf("✗ Device logged out - please scan QR code to log in again")

		case *events.PairSuccess:
			logger.Infof("✓ Phone pairing successful!")
			client.HandlePairingSuccess()

		case *events.PairError:
			logger.Errorf("✗ Phone pairing failed: %v", v.Error)
			client.HandlePairingError(v.Error)

		case *events.KeepAliveTimeout:
			logger.Warnf("⚠ KeepAlive timeout (errors: %d)", v.ErrorCount)
			if v.ErrorCount >= 3 {
				logger.Errorf("KeepAlive: %d consecutive failures, forcing disconnect+reconnect", v.ErrorCount)
				client.Disconnect()
				go func() {
					time.Sleep(2 * time.Second)
					if err := client.Client.Connect(); err != nil {
						logger.Errorf("Reconnect after KeepAlive failure: %v", err)
					}
				}()
			}

		case *events.StreamError:
			logger.Errorf("✗ Stream error: %v", v.Code)

		case *events.Disconnected:
			client.MarkDisconnected()
			logger.Warnf("⚠ Disconnected from WhatsApp - attempting reconnect")

		case *events.CallOffer:
			// Incoming call — resolve LID to regular JID, store as message
			resolvedJID := resolveCallJID(v.From)
			callFrom := resolvedJID.User
			chatJID := resolvedJID.String()
			isFromMe := client.Store.ID != nil && resolvedJID.User == client.Store.ID.User
			logger.Infof("[CALL] CallOffer from %s (CallID: %s, isFromMe: %v)", callFrom, v.CallID, isFromMe)
			name := client.GetChatName(messageStore, resolvedJID, chatJID, nil, callFrom)
			content := fmt.Sprintf("📞 Incoming call from %s", name)

			// Track call for duration/status updates
			activeCallsMu.Lock()
			activeCalls[v.CallID] = &activeCall{ChatJID: chatJID, Sender: callFrom, Name: name, Timestamp: v.Timestamp}
			activeCallsMu.Unlock()

			if err := messageStore.StoreChat(chatJID, name, v.Timestamp); err != nil {
				logger.Warnf("Failed to store chat for call: %v", err)
			}
			if err := messageStore.StoreMessage(
				"call-"+v.CallID, chatJID, callFrom, name,
				content, v.Timestamp, isFromMe,
				"call", "", "", nil, nil, nil, 0,
			); err != nil {
				logger.Warnf("Failed to store call message: %v", err)
			}

		case *events.CallTerminate:
			resolvedJID := resolveCallJID(v.From)
			logger.Infof("[CALL] CallTerminate from %s (CallID: %s, Reason: %s)", resolvedJID.User, v.CallID, v.Reason)

			// Update stored call message with duration and status
			activeCallsMu.Lock()
			call, exists := activeCalls[v.CallID]
			if exists {
				delete(activeCalls, v.CallID)
			}
			activeCallsMu.Unlock()

			if exists {
				duration := v.Timestamp.Sub(call.Timestamp)
				var content string
				switch v.Reason {
				case "timeout", "busy":
					content = fmt.Sprintf("📞 Missed call from %s", call.Name)
				default:
					content = fmt.Sprintf("📞 Call with %s (%s)", call.Name, formatDuration(duration))
				}
				// Update the stored message with final status
				if err := messageStore.StoreMessage(
					"call-"+v.CallID, call.ChatJID, call.Sender, call.Name,
					content, call.Timestamp, false,
					"call", "", "", nil, nil, nil, 0,
				); err != nil {
					logger.Warnf("Failed to update call message: %v", err)
				}
			}

		case *events.CallReject:
			resolvedJID := resolveCallJID(v.From)
			logger.Infof("[CALL] CallReject from %s (CallID: %s)", resolvedJID.User, v.CallID)

			activeCallsMu.Lock()
			call, exists := activeCalls[v.CallID]
			if exists {
				delete(activeCalls, v.CallID)
			}
			activeCallsMu.Unlock()

			if exists {
				content := fmt.Sprintf("📞 Missed call from %s", call.Name)
				if err := messageStore.StoreMessage(
					"call-"+v.CallID, call.ChatJID, call.Sender, call.Name,
					content, call.Timestamp, false,
					"call", "", "", nil, nil, nil, 0,
				); err != nil {
					logger.Warnf("Failed to update rejected call message: %v", err)
				}
			}

		case *events.CallAccept:
			logger.Infof("[CALL] CallAccept from %s (CallID: %s)", v.From.User, v.CallID)

		case *events.CallOfferNotice:
			// Group call notice — has Media field ("audio" or "video")
			resolvedJID := resolveCallJID(v.From)
			callFrom := resolvedJID.User
			isFromMe := client.Store.ID != nil && resolvedJID.User == client.Store.ID.User

			// Use GroupJID as chat for group calls, otherwise use caller's JID
			var chatJID string
			var chatResolvedJID types.JID
			if !v.GroupJID.IsEmpty() {
				chatJID = v.GroupJID.String()
				chatResolvedJID = v.GroupJID
			} else {
				chatJID = resolvedJID.String()
				chatResolvedJID = resolvedJID
			}

			logger.Infof("[CALL] CallOfferNotice from %s (CallID: %s, Media: %s, Group: %s)", callFrom, v.CallID, v.Media, chatJID)
			name := client.GetChatName(messageStore, chatResolvedJID, chatJID, nil, callFrom)
			content := fmt.Sprintf("📞 Incoming %s call from %s", v.Media, name)

			activeCallsMu.Lock()
			activeCalls[v.CallID] = &activeCall{ChatJID: chatJID, Sender: callFrom, Name: name, Timestamp: v.Timestamp}
			activeCallsMu.Unlock()

			if err := messageStore.StoreChat(chatJID, name, v.Timestamp); err != nil {
				logger.Warnf("Failed to store chat for group call: %v", err)
			}
			if err := messageStore.StoreMessage(
				"call-"+v.CallID, chatJID, callFrom, name,
				content, v.Timestamp, isFromMe,
				"call", "", "", nil, nil, nil, 0,
			); err != nil {
				logger.Warnf("Failed to store group call message: %v", err)
			}
		}
	})

	// Connection watchdog: exit process if disconnected >3 min (forces container restart)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, _, discAt, _ := client.ConnectionState()
			if !discAt.IsZero() && time.Since(discAt) > 3*time.Minute {
				logger.Errorf("WATCHDOG: disconnected for %v, exiting to force container restart", time.Since(discAt).Round(time.Second))
				os.Exit(1)
			}
		}
	}()

	// Periodic presence ping every 3 min to keep WhatsApp session active
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if client.IsConnected() {
				if err := client.SetPresence("available"); err != nil {
					logger.Debugf("Presence ping failed: %v", err)
				} else {
					logger.Debugf("Presence ping sent")
				}
			}
		}
	}()

	// Start REST API server with webhook support (BEFORE connecting to avoid blocking)
	server := api.NewServer(client, messageStore, webhookManager, cfg.APIPort)
	server.Start()
	fmt.Println("✓ REST API server started on port " + fmt.Sprintf("%d", cfg.APIPort))

	// Connect to WhatsApp in background (non-blocking so server can start)
	go func() {
		if err := client.Connect(); err != nil {
			logger.Errorf("Failed to connect to WhatsApp: %v", err)
		} else {
			fmt.Println("\n✓ Connected to WhatsApp!")
		}
	}()

	// Create a channel to keep the main goroutine alive
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("REST server is running. Press Ctrl+C to disconnect and exit.")
	fmt.Println("=" + fmt.Sprintf("%150s", ""))
	fmt.Println("Monitor sync progress:")
	fmt.Println("  curl -H 'X-API-Key: " + apiKey + "' http://localhost:" + fmt.Sprintf("%d", cfg.APIPort) + "/api/sync-status")
	fmt.Println("=" + fmt.Sprintf("%150s", ""))

	// Periodically log sync stats
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			logger.Debugf("[STATS] Connected: %v, JID: %v", client.IsConnected(), client.Store.ID)
		}
	}()

	// Wait for termination signal
	<-exitChan

	fmt.Println("Disconnecting...")
	// Disconnect client
	client.Disconnect()
}

// formatDuration formats a duration as "M:SS" or "H:MM:SS"
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
