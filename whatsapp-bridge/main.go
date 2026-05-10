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
	"whatsapp-bridge/internal/antiban"
	"whatsapp-bridge/internal/api"
	"whatsapp-bridge/internal/config"
	"whatsapp-bridge/internal/database"
	"whatsapp-bridge/internal/webhook"
	"whatsapp-bridge/internal/whatsapp"
)

// activeCall tracks an ongoing call for duration/status calculation.
type activeCall struct {
	ChatJID   string
	Sender    string
	Name      string
	Timestamp time.Time
	IsFromMe  bool
}

var (
	activeCalls   = make(map[string]*activeCall)
	activeCallsMu sync.Mutex
)

// formatDuration formats a duration as "M:SS".
func formatDuration(d time.Duration) string {
	secs := int(d.Seconds())
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

// resolveCallJID converts LID (hidden user) JIDs to regular phone JIDs.
func resolveCallJID(client *whatsapp.Client, logger waLog.Logger, jid types.JID) types.JID {
	if jid.Server == types.HiddenUserServer {
		resolved, err := client.Store.GetAltJID(context.Background(), jid)
		if err == nil && !resolved.IsEmpty() {
			logger.Debugf("[CALL] Resolved LID %s → %s", jid, resolved)
			return resolved
		}
		logger.Warnf("[CALL] Could not resolve LID %s: %v", jid, err)
	}
	return jid
}

// isCallFromMe checks if a call originated from this device.
func isCallFromMe(client *whatsapp.Client, from types.JID, resolvedFrom types.JID, callCreator types.JID) bool {
	if client.Store.ID == nil {
		return false
	}
	ownUser := client.Store.ID.ToNonAD().User
	if from.User == ownUser || resolvedFrom.User == ownUser {
		return true
	}
	if !callCreator.IsEmpty() && callCreator.User == ownUser {
		return true
	}
	return false
}

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
			_, _, discAt, _ := client.ConnectionState()
			client.MarkConnected()
			client.Antiban().RecordEvent(antiban.EventConnected)
			// Send presence to keep session active and receive real-time messages
			if err := client.SetPresence("available"); err != nil {
				logger.Warnf("Failed to set presence: %v", err)
			} else {
				logger.Infof("✓ Presence set to available")
			}
			logger.Infof("✓ Connected to WhatsApp")

			// If we were disconnected for >30s, attempt best-effort history backfill
			// for recently active chats to recover any messages missed during the gap.
			if !discAt.IsZero() {
				gap := time.Since(discAt)
				if gap > 30*time.Second {
					logger.Warnf("[RECONNECT] Gap detected: offline for %v — attempting history backfill", gap.Round(time.Second))
					go func() {
						time.Sleep(5 * time.Second) // let WA session stabilise first
						chats, err := messageStore.GetChats()
						if err != nil {
							logger.Warnf("[RECONNECT] Failed to get chats for backfill: %v", err)
							return
						}
						cutoff := time.Now().Add(-24 * time.Hour)
						requested := 0
						for chatJID, lastMsgTime := range chats {
							if lastMsgTime.Before(cutoff) {
								continue // skip inactive chats
							}
							msgs, err := messageStore.GetMessages(chatJID, 1)
							if err != nil || len(msgs) == 0 {
								continue
							}
							newest := msgs[0]
							err = client.RequestChatHistory(chatJID, newest.ID, newest.IsFromMe, newest.Sender, newest.Time.UnixMilli(), 50)
							if err != nil {
								logger.Warnf("[RECONNECT] History request failed for %s: %v", chatJID, err)
							} else {
								logger.Infof("[RECONNECT] History requested for %s (last active: %v)", chatJID, lastMsgTime.Format("15:04:05"))
								requested++
							}
							time.Sleep(500 * time.Millisecond) // avoid rate limiting
						}
						logger.Infof("[RECONNECT] Backfill requested for %d active chats", requested)
					}()
				}
			}

		case *events.LoggedOut:
			client.Antiban().RecordEvent(antiban.EventLoggedOut)
			logger.Warnf("✗ Device logged out - please scan QR code to log in again")

		case *events.PairSuccess:
			logger.Infof("✓ Phone pairing successful!")
			client.HandlePairingSuccess()

		case *events.PairError:
			logger.Errorf("✗ Phone pairing failed: %v", v.Error)
			client.HandlePairingError(v.Error)

		case *events.KeepAliveTimeout:
			client.Antiban().RecordEvent(antiban.EventKeepAliveTimeout)
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
			client.Antiban().RecordEvent(antiban.EventStreamError)
			logger.Errorf("✗ Stream error: %v", v.Code)

		case *events.Disconnected:
			client.MarkDisconnected()
			client.Antiban().RecordEvent(antiban.EventDisconnected)
			logger.Warnf("⚠ Disconnected from WhatsApp - attempting reconnect")

		case *events.CallOffer:
			resolvedJID := resolveCallJID(client, logger, v.From)
			callFrom := resolvedJID.User
			fromMe := isCallFromMe(client, v.From, resolvedJID, v.CallCreator)
			var chatJID string
			var chatResolvedJID types.JID
			if !v.GroupJID.IsEmpty() {
				chatJID = v.GroupJID.String()
				chatResolvedJID = v.GroupJID
			} else {
				chatJID = resolvedJID.String()
				chatResolvedJID = resolvedJID
			}
			logger.Infof("[CALL] CallOffer from %s (CallID: %s, isFromMe: %v)", callFrom, v.CallID, fromMe)
			name := client.GetChatName(messageStore, chatResolvedJID, chatJID, nil, callFrom)
			var content string
			if fromMe {
				content = fmt.Sprintf("📞 Outgoing call to %s", name)
			} else {
				content = fmt.Sprintf("📞 Incoming call from %s", name)
			}
			activeCallsMu.Lock()
			activeCalls[v.CallID] = &activeCall{ChatJID: chatJID, Sender: callFrom, Name: name, Timestamp: v.Timestamp, IsFromMe: fromMe}
			activeCallsMu.Unlock()
			if err := messageStore.StoreChat(chatJID, name, v.Timestamp); err != nil {
				logger.Warnf("Failed to store chat for call: %v", err)
			}
			if err := messageStore.StoreMessage("call-"+v.CallID, chatJID, callFrom, name, content, v.Timestamp, fromMe, "call", "", "", "", nil, nil, nil, 0); err != nil {
				logger.Warnf("Failed to store call message: %v", err)
			}

		case *events.CallOfferNotice:
			resolvedJID := resolveCallJID(client, logger, v.From)
			callFrom := resolvedJID.User
			fromMe := isCallFromMe(client, v.From, resolvedJID, v.CallCreator)
			var chatJID string
			var chatResolvedJID types.JID
			if !v.GroupJID.IsEmpty() {
				chatJID = v.GroupJID.String()
				chatResolvedJID = v.GroupJID
			} else {
				chatJID = resolvedJID.String()
				chatResolvedJID = resolvedJID
			}
			logger.Infof("[CALL] CallOfferNotice from %s (CallID: %s, Media: %s)", callFrom, v.CallID, v.Media)
			name := client.GetChatName(messageStore, chatResolvedJID, chatJID, nil, callFrom)
			var content string
			if fromMe {
				content = fmt.Sprintf("📞 Outgoing %s call to %s", v.Media, name)
			} else {
				content = fmt.Sprintf("📞 Incoming %s call from %s", v.Media, name)
			}
			activeCallsMu.Lock()
			activeCalls[v.CallID] = &activeCall{ChatJID: chatJID, Sender: callFrom, Name: name, Timestamp: v.Timestamp, IsFromMe: fromMe}
			activeCallsMu.Unlock()
			if err := messageStore.StoreChat(chatJID, name, v.Timestamp); err != nil {
				logger.Warnf("Failed to store chat for group call: %v", err)
			}
			if err := messageStore.StoreMessage("call-"+v.CallID, chatJID, callFrom, name, content, v.Timestamp, fromMe, "call", "", "", "", nil, nil, nil, 0); err != nil {
				logger.Warnf("Failed to store group call message: %v", err)
			}

		case *events.CallAccept:
			resolvedJID := resolveCallJID(client, logger, v.From)
			logger.Infof("[CALL] CallAccept from %s (CallID: %s)", resolvedJID.User, v.CallID)
			activeCallsMu.Lock()
			if call, exists := activeCalls[v.CallID]; exists {
				call.Timestamp = v.Timestamp
			}
			activeCallsMu.Unlock()

		case *events.CallTerminate:
			resolvedJID := resolveCallJID(client, logger, v.From)
			logger.Infof("[CALL] CallTerminate from %s (CallID: %s, Reason: %s)", resolvedJID.User, v.CallID, v.Reason)
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
				if err := messageStore.StoreMessage("call-"+v.CallID, call.ChatJID, call.Sender, call.Name, content, call.Timestamp, call.IsFromMe, "call", "", "", "", nil, nil, nil, 0); err != nil {
					logger.Warnf("Failed to update call message: %v", err)
				}
			}

		case *events.CallReject:
			resolvedJID := resolveCallJID(client, logger, v.From)
			logger.Infof("[CALL] CallReject from %s (CallID: %s)", resolvedJID.User, v.CallID)
			activeCallsMu.Lock()
			call, exists := activeCalls[v.CallID]
			if exists {
				delete(activeCalls, v.CallID)
			}
			activeCallsMu.Unlock()
			if exists {
				content := fmt.Sprintf("📞 Missed call from %s", call.Name)
				if err := messageStore.StoreMessage("call-"+v.CallID, call.ChatJID, call.Sender, call.Name, content, call.Timestamp, call.IsFromMe, "call", "", "", "", nil, nil, nil, 0); err != nil {
					logger.Warnf("Failed to update rejected call message: %v", err)
				}
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

	// Stale call cleanup: remove calls older than 5 minutes without terminate event
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-5 * time.Minute)
			activeCallsMu.Lock()
			for id, call := range activeCalls {
				if call.Timestamp.Before(cutoff) {
					delete(activeCalls, id)
					logger.Debugf("[CALL] Cleaned up stale call %s", id)
				}
			}
			activeCallsMu.Unlock()
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
	// Flush antiban state before disconnect
	if err := client.Antiban().Close(); err != nil {
		logger.Warnf("Antiban close error: %v", err)
	}
	// Disconnect client
	client.Disconnect()
}
