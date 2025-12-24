package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"whatsapp-bridge/internal/config"
	localTypes "whatsapp-bridge/internal/types"
)

// Client wraps the WhatsApp client with additional functionality
type Client struct {
	*whatsmeow.Client
	logger waLog.Logger
}

// NewClient creates a new WhatsApp client instance
func NewClient(logger waLog.Logger) (*Client, error) {
	return NewClientWithConfig(logger, config.NewConfig())
}

// NewClientWithConfig creates a new WhatsApp client with custom configuration
func NewClientWithConfig(logger waLog.Logger, cfg *config.Config) (*Client, error) {
	// Create database connection for storing session data
	dbLog := waLog.Stdout("Database", "INFO", true)

	// Create directory for database if it doesn't exist
	if err := os.MkdirAll("store", 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	// Configure HistorySyncConfig BEFORE creating device (Phase 4)
	// This affects how much message history is synced on first link
	store.DeviceProps.HistorySyncConfig = &waProto.DeviceProps_HistorySyncConfig{
		FullSyncDaysLimit:              proto.Uint32(cfg.HistorySyncDaysLimit),
		FullSyncSizeMbLimit:            proto.Uint32(cfg.HistorySyncSizeMB),
		StorageQuotaMb:                 proto.Uint32(cfg.StorageQuotaMB),
		InlineInitialPayloadInE2EeMsg:  proto.Bool(true),
		SupportCallLogHistory:          proto.Bool(false),
		SupportBotUserAgentChatHistory: proto.Bool(true),
		SupportCagReactionsAndPolls:    proto.Bool(true),
		SupportGroupHistory:            proto.Bool(true), // Enable group history
	}

	logger.Infof("HistorySyncConfig: days=%d, size=%dMB, quota=%dMB",
		cfg.HistorySyncDaysLimit, cfg.HistorySyncSizeMB, cfg.StorageQuotaMB)

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:store/whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Get device store - This contains session information
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			// No device exists, create one
			deviceStore = container.NewDevice()
			logger.Infof("Created new device with HistorySyncConfig")
		} else {
			return nil, fmt.Errorf("failed to get device: %v", err)
		}
	}

	// Create client instance
	client := whatsmeow.NewClient(deviceStore, logger)
	if client == nil {
		return nil, fmt.Errorf("failed to create WhatsApp client")
	}

	return &Client{
		Client: client,
		logger: logger,
	}, nil
}

// Connect establishes connection to WhatsApp with QR code handling if needed
func (c *Client) Connect() error {
	// Create channel to track connection success
	connected := make(chan bool, 1)

	if c.Store.ID == nil {
		// No ID stored, this is a new client, need to pair with phone
		qrChan, _ := c.GetQRChannel(context.Background())
		err := c.Client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		// Print QR code for pairing with phone
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan this QR code with your WhatsApp app:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else if evt.Event == "success" {
				connected <- true
				break
			}
		}

		// Wait for connection
		select {
		case <-connected:
			fmt.Println("\nSuccessfully connected and authenticated!")
		case <-time.After(3 * time.Minute):
			return fmt.Errorf("timeout waiting for QR code scan")
		}
	} else {
		// Already logged in, just connect
		err := c.Client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}
		connected <- true
	}

	// Wait a moment for connection to stabilize
	time.Sleep(2 * time.Second)

	if !c.IsConnected() {
		return fmt.Errorf("failed to establish stable connection")
	}

	c.logger.Infof("✓ Connected to WhatsApp!")
	return nil
}

// Phase 5: Advanced Features

// SetPresence sets the client's own presence (available/unavailable)
func (c *Client) SetPresence(presence string) error {
	var p types.Presence
	switch presence {
	case "available":
		p = types.PresenceAvailable
	case "unavailable":
		p = types.PresenceUnavailable
	default:
		return fmt.Errorf("invalid presence: %s (must be 'available' or 'unavailable')", presence)
	}
	return c.SendPresence(context.Background(), p)
}

// SubscribePresence subscribes to presence updates for a specific user
func (c *Client) SubscribeToPresence(jidStr string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	return c.Client.SubscribePresence(context.Background(), jid)
}

// GetProfilePicture gets the profile picture URL for a user or group
func (c *Client) GetProfilePicture(jidStr string, preview bool) (*localTypes.ProfilePictureInfo, error) {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JID: %v", err)
	}

	params := &whatsmeow.GetProfilePictureParams{
		Preview: preview,
	}

	info, err := c.GetProfilePictureInfo(context.Background(), jid, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile picture: %v", err)
	}

	if info == nil {
		return nil, nil // No profile picture
	}

	return &localTypes.ProfilePictureInfo{
		URL:        info.URL,
		ID:         info.ID,
		Type:       info.Type,
		DirectPath: info.DirectPath,
	}, nil
}

// GetBlockedUsers returns the list of blocked users
func (c *Client) GetBlockedUsers() ([]localTypes.BlockedUser, error) {
	blocklist, err := c.GetBlocklist(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get blocklist: %v", err)
	}

	users := make([]localTypes.BlockedUser, len(blocklist.JIDs))
	for i, jid := range blocklist.JIDs {
		users[i] = localTypes.BlockedUser{JID: jid.String()}
	}
	return users, nil
}

// UpdateBlockedUser blocks or unblocks a user
func (c *Client) UpdateBlockedUser(jidStr string, action string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}

	var blockAction events.BlocklistChangeAction
	switch action {
	case "block":
		blockAction = events.BlocklistChangeActionBlock
	case "unblock":
		blockAction = events.BlocklistChangeActionUnblock
	default:
		return fmt.Errorf("invalid action: %s (must be 'block' or 'unblock')", action)
	}

	_, err = c.UpdateBlocklist(context.Background(), jid, blockAction)
	if err != nil {
		return fmt.Errorf("failed to update blocklist: %v", err)
	}
	return nil
}

// FollowNewsletterChannel follows (joins) a newsletter/channel
func (c *Client) FollowNewsletterChannel(jidStr string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	return c.FollowNewsletter(context.Background(), jid)
}

// UnfollowNewsletterChannel unfollows a newsletter/channel
func (c *Client) UnfollowNewsletterChannel(jidStr string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	return c.UnfollowNewsletter(context.Background(), jid)
}

// CreateNewsletterChannel creates a new newsletter/channel
func (c *Client) CreateNewsletterChannel(name, description string) (*localTypes.NewsletterInfo, error) {
	params := whatsmeow.CreateNewsletterParams{
		Name:        name,
		Description: description,
	}

	meta, err := c.CreateNewsletter(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create newsletter: %v", err)
	}

	return &localTypes.NewsletterInfo{
		JID:         meta.ID.String(),
		Name:        meta.ThreadMeta.Name.Text,
		Description: meta.ThreadMeta.Description.Text,
	}, nil
}
