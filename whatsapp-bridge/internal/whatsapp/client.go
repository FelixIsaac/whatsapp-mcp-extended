// Package whatsapp provides the WhatsApp client wrapper and message handling.
// It uses the whatsmeow library for WhatsApp Web multi-device API access.
package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
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

// Client wraps the whatsmeow client with additional functionality
// for message handling, media operations, and group management.
type Client struct {
	*whatsmeow.Client
	logger waLog.Logger

	// Pairing state
	pairingMutex      sync.Mutex
	pairingInProgress bool
	pairingCode       string
	pairingExpiry     time.Time
	pairingComplete   bool
	pairingError      error
}

// NewClient creates a new WhatsApp client with default configuration.
// Uses environment defaults for history sync settings.
func NewClient(logger waLog.Logger) (*Client, error) {
	return NewClientWithConfig(logger, config.NewConfig())
}

// NewClientWithConfig creates a new WhatsApp client with custom configuration.
// Configures history sync limits and creates/opens the session database.
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

// Connect establishes connection to WhatsApp servers.
// For new devices, displays QR code for phone pairing.
// For existing sessions, reconnects using stored credentials.
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

// SetPresence sets the client's online status.
// Valid values: "available" (online) or "unavailable" (offline).
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

// SubscribeToPresence subscribes to presence updates for a contact.
// After subscribing, presence events will be received via event handlers.
func (c *Client) SubscribeToPresence(jidStr string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	return c.Client.SubscribePresence(context.Background(), jid)
}

// GetProfilePicture retrieves the profile picture URL for a user or group.
// Set preview=true for thumbnail, false for full resolution image.
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

// GetBlockedUsers returns the list of currently blocked users.
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

// UpdateBlockedUser blocks or unblocks a user.
// Action must be "block" or "unblock".
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

// FollowNewsletterChannel subscribes to a WhatsApp newsletter/channel.
func (c *Client) FollowNewsletterChannel(jidStr string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	return c.FollowNewsletter(context.Background(), jid)
}

// UnfollowNewsletterChannel unsubscribes from a WhatsApp newsletter/channel.
func (c *Client) UnfollowNewsletterChannel(jidStr string) error {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %v", err)
	}
	return c.UnfollowNewsletter(context.Background(), jid)
}

// CreateNewsletterChannel creates a new WhatsApp newsletter/channel.
// Returns the created newsletter's JID, name, and description.
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

// Phase 6: Chat Features

// SendTypingIndicator sends a typing/recording indicator to a chat.
// State can be "typing" (composing), "paused" (stopped typing), or "recording" (voice message).
func (c *Client) SendTypingIndicator(chatJID string, state string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	var chatState types.ChatPresence
	var media types.ChatPresenceMedia

	switch state {
	case "typing":
		chatState = types.ChatPresenceComposing
		media = types.ChatPresenceMediaText
	case "paused":
		chatState = types.ChatPresencePaused
		media = ""
	case "recording":
		chatState = types.ChatPresenceComposing
		media = types.ChatPresenceMediaAudio
	default:
		return fmt.Errorf("invalid state: %s (must be 'typing', 'paused', or 'recording')", state)
	}

	return c.SendChatPresence(context.Background(), jid, chatState, media)
}

// SetAboutText updates the user's profile "About" status text.
// This is the text shown in the profile, not ephemeral status broadcasts.
func (c *Client) SetAboutText(text string) error {
	return c.SetStatusMessage(context.Background(), text)
}

// SetDisappearingTimer sets the disappearing messages timer for a chat.
// Valid durations: "off", "24h", "7d", "90d".
// In groups, only admins can change this setting.
func (c *Client) SetDisappearingTimer(chatJID string, duration string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	var timer time.Duration
	switch duration {
	case "off", "0":
		timer = 0
	case "24h":
		timer = 24 * time.Hour
	case "7d":
		timer = 7 * 24 * time.Hour
	case "90d":
		timer = 90 * 24 * time.Hour
	default:
		return fmt.Errorf("invalid duration: %s (must be 'off', '24h', '7d', or '90d')", duration)
	}

	return c.Client.SetDisappearingTimer(context.Background(), jid, timer, time.Now())
}

// GetPrivacySettings fetches the current privacy settings for the user.
// Returns a map of privacy setting categories and their values.
// Valid values: "all", "contacts", "contact_blacklist", "none", "known", "match_last_seen".
func (c *Client) GetPrivacySettings() (map[string]string, error) {
	settings, err := c.Client.TryFetchPrivacySettings(context.Background(), false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch privacy settings: %v", err)
	}

	if settings == nil {
		return map[string]string{}, nil
	}

	return map[string]string{
		"group_add":     string(settings.GroupAdd),
		"last_seen":     string(settings.LastSeen),
		"status":        string(settings.Status),
		"profile":       string(settings.Profile),
		"read_receipts": string(settings.ReadReceipts),
		"call_add":      string(settings.CallAdd),
		"online":        string(settings.Online),
	}, nil
}

// PinChat pins a chat to the top of the chat list.
func (c *Client) PinChat(chatJID string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	patch := appstate.BuildPin(jid, true)
	return c.Client.SendAppState(context.Background(), patch)
}

// UnpinChat unpins a chat from the top of the chat list.
func (c *Client) UnpinChat(chatJID string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	patch := appstate.BuildPin(jid, false)
	return c.Client.SendAppState(context.Background(), patch)
}

// MuteChat mutes a chat for the specified duration.
// Duration examples: "forever" (0), "15m", "1h", "8h", "1w".
func (c *Client) MuteChat(chatJID string, duration string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	var muteDuration time.Duration
	switch duration {
	case "forever", "0":
		muteDuration = 0
	case "15m":
		muteDuration = 15 * time.Minute
	case "1h":
		muteDuration = 1 * time.Hour
	case "8h":
		muteDuration = 8 * time.Hour
	case "1w":
		muteDuration = 7 * 24 * time.Hour
	default:
		return fmt.Errorf("invalid duration: %s (must be 'forever', '15m', '1h', '8h', or '1w')", duration)
	}

	patch := appstate.BuildMute(jid, true, muteDuration)
	return c.Client.SendAppState(context.Background(), patch)
}

// UnmuteChat unmutes a chat.
func (c *Client) UnmuteChat(chatJID string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	patch := appstate.BuildMute(jid, false, 0)
	return c.Client.SendAppState(context.Background(), patch)
}

// ArchiveChat archives a chat.
func (c *Client) ArchiveChat(chatJID string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	patch := appstate.BuildArchive(jid, true, time.Time{}, nil)
	return c.Client.SendAppState(context.Background(), patch)
}

// UnarchiveChat unarchives a chat.
func (c *Client) UnarchiveChat(chatJID string) error {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	patch := appstate.BuildArchive(jid, false, time.Time{}, nil)
	return c.Client.SendAppState(context.Background(), patch)
}

// Phase 7: Phone Number Pairing

// PairWithPhone initiates phone number pairing and returns 8-digit code
func (c *Client) PairWithPhone(phoneNumber string) (string, error) {
	c.pairingMutex.Lock()
	defer c.pairingMutex.Unlock()

	if c.pairingInProgress {
		return "", fmt.Errorf("pairing already in progress")
	}

	if c.Store.ID != nil {
		return "", fmt.Errorf("device already linked")
	}

	// Reset state
	c.pairingInProgress = true
	c.pairingComplete = false
	c.pairingError = nil

	// Connect if not connected
	if !c.IsConnected() {
		if err := c.Client.Connect(); err != nil {
			c.pairingInProgress = false
			return "", fmt.Errorf("failed to connect: %v", err)
		}
	}

	// Request pairing code
	code, err := c.Client.PairPhone(context.Background(), phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		c.pairingInProgress = false
		return "", fmt.Errorf("failed to request pairing code: %v", err)
	}

	c.pairingCode = code
	c.pairingExpiry = time.Now().Add(160 * time.Second)

	c.logger.Infof("Pairing code generated: %s (expires in 160s)", code)
	return code, nil
}

// GetPairingStatus returns current pairing state
func (c *Client) GetPairingStatus() (inProgress bool, code string, expiresIn int, complete bool, err error) {
	c.pairingMutex.Lock()
	defer c.pairingMutex.Unlock()

	var expiresInSec int
	if c.pairingInProgress && !c.pairingExpiry.IsZero() {
		remaining := time.Until(c.pairingExpiry)
		if remaining > 0 {
			expiresInSec = int(remaining.Seconds())
		}
	}

	return c.pairingInProgress, c.pairingCode, expiresInSec, c.pairingComplete, c.pairingError
}

// HandlePairingSuccess called when pairing succeeds
func (c *Client) HandlePairingSuccess() {
	c.pairingMutex.Lock()
	defer c.pairingMutex.Unlock()

	c.pairingComplete = true
	c.pairingInProgress = false
	c.logger.Infof("Pairing successful!")
}

// HandlePairingError called when pairing fails
func (c *Client) HandlePairingError(err error) {
	c.pairingMutex.Lock()
	defer c.pairingMutex.Unlock()

	c.pairingError = err
	c.pairingInProgress = false
	c.logger.Errorf("Pairing failed: %v", err)
}
