package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"whatsapp-bridge/internal/database"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// isPhoneNumber matches strings that look like raw phone numbers (no real name).
var isPhoneNumber = regexp.MustCompile(`^\+?\d{5,15}$`).MatchString

// GetChatName determines the appropriate name for a chat based on JID and other info
func (c *Client) GetChatName(messageStore *database.MessageStore, jid types.JID, chatJID string, conversation interface{}, sender string) string {
	// First, check if chat already exists in database with a non-phone-number name
	var existingName string
	err := messageStore.GetDB().QueryRow("SELECT name FROM chats WHERE jid = ?", chatJID).Scan(&existingName)
	if err == nil && existingName != "" && !isPhoneNumber(existingName) {
		// Chat exists with a real name, use that
		c.logger.Infof("Using existing chat name for %s: %s", chatJID, existingName)
		return existingName
	}

	// Need to determine chat name
	var name string

	if jid.Server == "g.us" {
		// This is a group chat
		c.logger.Infof("Getting name for group: %s", chatJID)

		// Use conversation data if provided (from history sync)
		if conversation != nil {
			// Extract name from conversation if available
			// This uses type assertions to handle different possible types
			var displayName, convName *string
			// Try to extract the fields we care about regardless of the exact type
			v := reflect.ValueOf(conversation)
			if v.Kind() == reflect.Ptr && !v.IsNil() {
				v = v.Elem()

				// Try to find DisplayName field
				if displayNameField := v.FieldByName("DisplayName"); displayNameField.IsValid() && displayNameField.Kind() == reflect.Ptr && !displayNameField.IsNil() {
					dn := displayNameField.Elem().String()
					displayName = &dn
				}

				// Try to find Name field
				if nameField := v.FieldByName("Name"); nameField.IsValid() && nameField.Kind() == reflect.Ptr && !nameField.IsNil() {
					n := nameField.Elem().String()
					convName = &n
				}
			}

			// Use the name we found
			if displayName != nil && *displayName != "" {
				name = *displayName
			} else if convName != nil && *convName != "" {
				name = *convName
			}
		}

		// If we didn't get a name, try group info
		if name == "" {
			groupInfo, err := c.Client.GetGroupInfo(context.Background(), jid)
			if err == nil && groupInfo.Name != "" {
				name = groupInfo.Name
			} else {
				// Fallback name for groups
				name = fmt.Sprintf("Group %s", jid.User)
			}
		}

		c.logger.Infof("Using group name: %s", name)
	} else {
		// This is an individual contact
		c.logger.Infof("Getting name for contact: %s", chatJID)

		// Use priority fallback: FullName > PushName > FirstName > BusinessName
		contact, err := c.Store.Contacts.GetContact(context.Background(), jid)
		if err == nil && contact.Found {
			if contact.FullName != "" {
				name = contact.FullName
			} else if contact.PushName != "" {
				name = contact.PushName
			} else if contact.FirstName != "" {
				name = contact.FirstName
			} else if contact.BusinessName != "" {
				name = contact.BusinessName
			}
		}
		if name == "" {
			if sender != "" {
				// Fallback to sender
				name = sender
			} else {
				// Last fallback to JID
				name = jid.User
			}
		}

		c.logger.Infof("Using contact name: %s", name)
	}

	return name
}

// HandleMessage processes regular incoming messages with media support and webhook processing
func (c *Client) HandleMessage(messageStore *database.MessageStore, webhookManager interface{}, msg *events.Message) {
	// Save message to database
	chatJID := msg.Info.Chat.String()
	sender := msg.Info.Sender.User

	// Get appropriate chat name (pass nil for conversation since we don't have one for regular messages)
	name := c.GetChatName(messageStore, msg.Info.Chat, chatJID, nil, sender)

	// Update chat in database with the message timestamp (keeps last message time updated)
	err := messageStore.StoreChat(chatJID, name, msg.Info.Timestamp)
	if err != nil {
		c.logger.Warnf("Failed to store chat: %v", err)
	}

	// Extract text content
	content := ExtractTextContent(msg.Message)

	// Extract media info
	mediaType, filename, url, directPath, mediaKey, fileSHA256, fileEncSHA256, fileLength := ExtractMediaInfo(msg.Message)

	// Skip if there's no content and no media
	if content == "" && mediaType == "" {
		return
	}

	// Get sender name (PushName from WhatsApp)
	senderName := msg.Info.PushName
	if senderName == "" {
		senderName = sender // fallback to JID
	}

	// Store message in database
	err = messageStore.StoreMessage(
		msg.Info.ID,
		chatJID,
		sender,
		senderName,
		content,
		msg.Info.Timestamp,
		msg.Info.IsFromMe,
		mediaType,
		filename,
		url,
		directPath,
		mediaKey,
		fileSHA256,
		fileEncSHA256,
		fileLength,
	)

	if err != nil {
		c.logger.Warnf("Failed to store message: %v", err)
	}

	// Auto-download media while CDN URL is still fresh
	if mediaType != "" && (url != "" || directPath != "") {
		go c.autoDownloadMedia(msg.Info.ID, chatJID, mediaType, filename, url, directPath, mediaKey, fileSHA256, fileEncSHA256, fileLength)
	}

	// Process webhooks if manager is available
	if webhookManager != nil {
		// Cast to webhook manager and process message
		if wm, ok := webhookManager.(interface {
			ProcessMessage(client interface{}, msg *events.Message, chatName string)
		}); ok {
			wm.ProcessMessage(c, msg, name)
		}
	}
}

// HandleHistorySync processes history sync events
func (c *Client) HandleHistorySync(messageStore *database.MessageStore, historySync *events.HistorySync) {
	c.logger.Infof("Received history sync event with %d conversations", len(historySync.Data.Conversations))

	syncedCount := 0
	for _, conversation := range historySync.Data.Conversations {
		// Parse JID from the conversation
		if conversation.ID == nil {
			continue
		}

		chatJID := *conversation.ID

		// Try to parse the JID
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			c.logger.Warnf("Failed to parse JID %s: %v", chatJID, err)
			continue
		}

		// Get appropriate chat name by passing the history sync conversation directly
		name := c.GetChatName(messageStore, jid, chatJID, conversation, "")

		messages := conversation.Messages
		if len(messages) > 0 {
			timestamp, ok := latestHistoryMessageTime(messages)
			if !ok {
				continue
			}
			if err := messageStore.StoreChat(chatJID, name, timestamp); err != nil {
				c.logger.Warnf("Failed to store chat: %v", err)
			}

			// Store messages
			for _, msg := range messages {
				if msg == nil || msg.Message == nil {
					continue
				}

				// Extract text content
				var content string
				if msg.Message.Message != nil {
					content = ExtractTextContent(msg.Message.Message)
				}

				// Extract media info
				var mediaType, filename, url, directPath string
				var mediaKey, fileSHA256, fileEncSHA256 []byte
				var fileLength uint64

				if msg.Message.Message != nil {
					mediaType, filename, url, directPath, mediaKey, fileSHA256, fileEncSHA256, fileLength = ExtractMediaInfo(msg.Message.Message)
				}

				// Log the message content for debugging
				c.logger.Infof("Message content: %v, Media Type: %v", content, mediaType)

				// Skip messages with no content and no media
				if content == "" && mediaType == "" {
					continue
				}

				sender, isFromMe := c.historyMessageSender(jid, msg.Message.Key)

				// Store message
				msgID := ""
				if msg.Message.Key != nil && msg.Message.Key.ID != nil {
					msgID = *msg.Message.Key.ID
				}

				// Get message timestamp
				ts2 := msg.Message.GetMessageTimestamp()
				if ts2 == 0 {
					continue
				}
				timestamp := time.Unix(int64(ts2), 0)

				// For history sync, use sender as senderName fallback (PushName not directly available)
				senderName := sender

				err = messageStore.StoreMessage(
					msgID,
					chatJID,
					sender,
					senderName,
					content,
					timestamp,
					isFromMe,
					mediaType,
					filename,
					url,
					directPath,
					mediaKey,
					fileSHA256,
					fileEncSHA256,
					fileLength,
				)
				if err != nil {
					c.logger.Warnf("Failed to store history message: %v", err)
				} else {
					syncedCount++
					// Log successful message storage
					if mediaType != "" {
						c.logger.Infof("Stored message: [%s] %s -> %s: [%s: %s] %s",
							timestamp.Format("2006-01-02 15:04:05"), sender, chatJID, mediaType, filename, content)
					} else {
						c.logger.Infof("Stored message: [%s] %s -> %s: %s",
							timestamp.Format("2006-01-02 15:04:05"), sender, chatJID, content)
					}
				}
			}
		}
	}

	c.logger.Infof("History sync complete. Stored %d messages.", syncedCount)
}

func latestHistoryMessageTime(messages []*waHistorySync.HistorySyncMsg) (time.Time, bool) {
	var latest time.Time
	for _, msg := range messages {
		if msg == nil || msg.Message == nil {
			continue
		}
		ts := msg.Message.GetMessageTimestamp()
		if ts == 0 {
			continue
		}
		timestamp := time.Unix(int64(ts), 0)
		if latest.IsZero() || timestamp.After(latest) {
			latest = timestamp
		}
	}
	return latest, !latest.IsZero()
}

func (c *Client) historyMessageSender(chat types.JID, key *waCommon.MessageKey) (string, bool) {
	isFromMe := false
	if key != nil && key.FromMe != nil {
		isFromMe = *key.FromMe
	}
	if !isFromMe && key != nil && key.Participant != nil && *key.Participant != "" {
		return *key.Participant, false
	}
	if isFromMe && c.Store != nil && c.Store.ID != nil {
		return c.Store.ID.ToNonAD().String(), true
	}
	return chat.ToNonAD().String(), isFromMe
}

// autoDownloadMedia downloads and saves media to disk immediately after receiving a message,
// before the WhatsApp CDN URL expires. Runs as a goroutine; all errors are logged and ignored.
func (c *Client) autoDownloadMedia(msgID, chatJID, mediaType, filename, rawURL, directPath string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) {
	// Sanitize JID for filesystem (replace colons and other special chars)
	sanitizedJID := regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(chatJID, "_")
	outDir := filepath.Join("store", sanitizedJID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		c.logger.Warnf("auto-download: mkdir %s: %v", outDir, err)
		return
	}

	if filename == "" {
		filename = msgID + ".bin"
	}
	// Sanitize filename
	safeFilename := regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(filename, "_")
	outPath := filepath.Join(outDir, safeFilename)

	// Skip if already downloaded
	if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
		c.logger.Infof("auto-download: %s already exists, skipping", outPath)
		return
	}

	// Resolve directPath from URL if not provided
	if directPath == "" {
		directPath = autoDownloadExtractDirectPath(rawURL)
	}

	// Classify media type for whatsmeow
	var wmMediaType whatsmeow.MediaType
	switch strings.ToLower(mediaType) {
	case "image":
		wmMediaType = whatsmeow.MediaImage
	case "video":
		wmMediaType = whatsmeow.MediaVideo
	case "audio":
		wmMediaType = whatsmeow.MediaAudio
	case "document":
		wmMediaType = whatsmeow.MediaDocument
	default:
		wmMediaType = whatsmeow.MediaImage
	}

	dl := &autoDownloadableMedia{
		url:           rawURL,
		directPath:    directPath,
		mediaKey:      mediaKey,
		fileLength:    fileLength,
		fileSHA256:    fileSHA256,
		fileEncSHA256: fileEncSHA256,
		mediaType:     wmMediaType,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	data, err := c.Client.Download(ctx, dl)
	if err != nil {
		c.logger.Warnf("auto-download: %s/%s: %v", chatJID, msgID, err)
		return
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		c.logger.Warnf("auto-download: write %s: %v", outPath, err)
		return
	}

	c.logger.Infof("auto-download: saved %s (%d bytes)", outPath, len(data))
}

// autoDownloadableMedia implements whatsmeow.DownloadableMessage for auto-download.
type autoDownloadableMedia struct {
	url           string
	directPath    string
	mediaKey      []byte
	fileLength    uint64
	fileSHA256    []byte
	fileEncSHA256 []byte
	mediaType     whatsmeow.MediaType
}

func (d *autoDownloadableMedia) GetDirectPath() string             { return d.directPath }
func (d *autoDownloadableMedia) GetMediaKey() []byte               { return d.mediaKey }
func (d *autoDownloadableMedia) GetFileLength() uint64             { return d.fileLength }
func (d *autoDownloadableMedia) GetFileSHA256() []byte             { return d.fileSHA256 }
func (d *autoDownloadableMedia) GetFileEncSHA256() []byte          { return d.fileEncSHA256 }
func (d *autoDownloadableMedia) GetMediaType() whatsmeow.MediaType { return d.mediaType }
func (d *autoDownloadableMedia) GetUrl() string                    { return d.url }

// autoDownloadExtractDirectPath strips the scheme+host from a WhatsApp CDN URL to get the path.
func autoDownloadExtractDirectPath(rawURL string) string {
	if idx := strings.Index(rawURL, ".net/"); idx >= 0 {
		return rawURL[idx+4:]
	}
	if i := strings.Index(rawURL, "://"); i >= 0 {
		rest := rawURL[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			return rest[j:]
		}
	}
	return rawURL
}
