package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
			logger.Warnf("⚠ KeepAlive timeout - connection may be unstable (sync may fail)")

		case *events.StreamError:
			logger.Errorf("✗ Stream error: %v", v.Code)

		case *events.Disconnected:
			logger.Warnf("⚠ Disconnected from WhatsApp - attempting reconnect")
		}
	})

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
