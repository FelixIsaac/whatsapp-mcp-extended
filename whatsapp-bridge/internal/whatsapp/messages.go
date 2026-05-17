package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"whatsapp-bridge/internal/antiban"
	"whatsapp-bridge/internal/database"
	bridgeTypes "whatsapp-bridge/internal/types"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// mentionPattern matches @phone_number patterns in message text (7-15 digits)
var mentionPattern = regexp.MustCompile(`@(\d{7,15})`)

func mediaTypeAndMimeType(mediaPath string) (whatsmeow.MediaType, string) {
	fileExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(mediaPath)), ".")

	switch fileExt {
	case "jpg", "jpeg":
		return whatsmeow.MediaImage, "image/jpeg"
	case "png":
		return whatsmeow.MediaImage, "image/png"
	case "gif":
		return whatsmeow.MediaImage, "image/gif"
	case "webp":
		return whatsmeow.MediaImage, "image/webp"
	case "ogg":
		return whatsmeow.MediaAudio, "audio/ogg; codecs=opus"
	case "mp3":
		return whatsmeow.MediaAudio, "audio/mpeg"
	case "m4a", "aac":
		return whatsmeow.MediaAudio, "audio/mp4"
	case "mp4":
		return whatsmeow.MediaVideo, "video/mp4"
	case "avi":
		return whatsmeow.MediaVideo, "video/avi"
	case "mov":
		return whatsmeow.MediaVideo, "video/quicktime"
	case "epub":
		return whatsmeow.MediaDocument, "application/epub+zip"
	case "pdf":
		return whatsmeow.MediaDocument, "application/pdf"
	case "cbz":
		return whatsmeow.MediaDocument, "application/x-cbz"
	case "doc":
		return whatsmeow.MediaDocument, "application/msword"
	case "docx":
		return whatsmeow.MediaDocument, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xls":
		return whatsmeow.MediaDocument, "application/vnd.ms-excel"
	case "xlsx":
		return whatsmeow.MediaDocument, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "ppt":
		return whatsmeow.MediaDocument, "application/vnd.ms-powerpoint"
	case "pptx":
		return whatsmeow.MediaDocument, "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case "txt":
		return whatsmeow.MediaDocument, "text/plain"
	case "csv":
		return whatsmeow.MediaDocument, "text/csv"
	case "zip":
		return whatsmeow.MediaDocument, "application/zip"
	default:
		return whatsmeow.MediaDocument, "application/octet-stream"
	}
}

func documentFileName(mediaPath string) string {
	return filepath.Base(strings.ReplaceAll(mediaPath, "\\", "/"))
}

// extractMentionsFromText detects @number mentions in text and returns WhatsApp JIDs
func extractMentionsFromText(text string) []string {
	matches := mentionPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var jids []string
	for _, m := range matches {
		jid := m[1] + "@s.whatsapp.net"
		if !seen[jid] {
			seen[jid] = true
			jids = append(jids, jid)
		}
	}
	return jids
}

// allowedMediaDirs contains directories allowed for media access
var allowedMediaDirs = []string{
	"/app/media",
	"/app/store",
	"/app/whatsapp-bridge/store",
	"/tmp",
}

// validateMediaPath checks if the path is within allowed directories
func validateMediaPath(mediaPath string) error {
	if mediaPath == "" {
		return nil
	}

	// Clean and get absolute path
	cleanPath := filepath.Clean(mediaPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid media path: %v", err)
	}

	// Check for path traversal attempts
	if strings.Contains(mediaPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Allow if DISABLE_PATH_CHECK is set (for development)
	if os.Getenv("DISABLE_PATH_CHECK") == "true" {
		return nil
	}

	// Check if path is within allowed directories
	for _, allowedDir := range allowedMediaDirs {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			return nil
		}
	}

	return fmt.Errorf("media path outside allowed directories")
}

// SendMessage sends a WhatsApp message with optional media and mentions
func (c *Client) SendMessage(messageStore *database.MessageStore, recipient string, message string, mediaPath string, mentionedJIDs ...[]string) bridgeTypes.SendResult {
	if !c.IsConnected() {
		return bridgeTypes.SendResult{Success: false, Error: "Not connected to WhatsApp"}
	}

	// Create JID for recipient
	var recipientJID types.JID
	var err error

	// Check if recipient is a JID
	isJID := strings.Contains(recipient, "@")

	if isJID {
		// Parse the JID string
		recipientJID, err = types.ParseJID(recipient)
		if err != nil {
			return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("Error parsing JID: %v", err)}
		}
	} else {
		// Create JID from phone number
		recipientJID = types.JID{
			User:   recipient,
			Server: "s.whatsapp.net", // For personal chats
		}
	}

	msg := &waE2E.Message{}

	// Check if we have media to send
	if mediaPath != "" {
		// Validate media path (prevent path traversal)
		if err := validateMediaPath(mediaPath); err != nil {
			return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("Invalid media path: %v", err)}
		}

		// Read media file
		mediaData, err := os.ReadFile(mediaPath)
		if err != nil {
			return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("Error reading media file: %v", err)}
		}

		mediaType, mimeType := mediaTypeAndMimeType(mediaPath)

		// Upload media to WhatsApp servers
		resp, err := c.Upload(context.Background(), mediaData, mediaType)
		if err != nil {
			return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("Error uploading media: %v", err)}
		}

		// Create the appropriate message type based on media type
		switch mediaType {
		case whatsmeow.MediaImage:
			msg.ImageMessage = &waE2E.ImageMessage{
				Caption:       proto.String(message),
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case whatsmeow.MediaAudio:
			// Handle ogg audio files
			var seconds uint32 = 30 // Default fallback
			var waveform []byte = nil

			// Try to analyze the ogg file
			if strings.Contains(mimeType, "ogg") {
				analyzedSeconds, analyzedWaveform, err := AnalyzeOggOpus(mediaData)
				if err == nil {
					seconds = analyzedSeconds
					waveform = analyzedWaveform
				} else {
					return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("Failed to analyze Ogg Opus file: %v", err)}
				}
			}

			msg.AudioMessage = &waE2E.AudioMessage{
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
				Seconds:       proto.Uint32(seconds),
				PTT:           proto.Bool(true),
				Waveform:      waveform,
			}
		case whatsmeow.MediaVideo:
			msg.VideoMessage = &waE2E.VideoMessage{
				Caption:       proto.String(message),
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case whatsmeow.MediaDocument:
			docName := documentFileName(mediaPath)
			msg.DocumentMessage = &waE2E.DocumentMessage{
				Title:         proto.String(docName),
				FileName:      proto.String(docName),
				Caption:       proto.String(message),
				Mimetype:      proto.String(mimeType),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		}
	} else {
		// Collect mentions: explicit + auto-detected from @number patterns
		var mentions []string
		if len(mentionedJIDs) > 0 {
			mentions = append(mentions, mentionedJIDs[0]...)
		}
		mentions = append(mentions, extractMentionsFromText(message)...)

		if len(mentions) > 0 {
			msg.ExtendedTextMessage = &waE2E.ExtendedTextMessage{
				Text: proto.String(message),
				ContextInfo: &waE2E.ContextInfo{
					MentionedJID: mentions,
				},
			}
		} else {
			msg.Conversation = proto.String(message)
		}
	}

	// Antiban: simulate typing and enforce rate limit / warm-up before sending
	if c.antiban.Enabled() {
		_ = c.SendTypingIndicator(recipientJID.String(), "typing")
	}
	if err := c.antiban.BeforeSend(context.Background(), antiban.Text, len(message)); err != nil {
		_ = c.SendTypingIndicator(recipientJID.String(), "paused")
		return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("antiban blocked send: %v", err)}
	}

	// Send message
	sendResp, err := c.Client.SendMessage(context.Background(), recipientJID, msg)
	if err != nil {
		_ = c.SendTypingIndicator(recipientJID.String(), "paused")
		c.antiban.AfterSendFailed(antiban.Text, err)
		return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("Error sending message: %v", err)}
	}
	_ = c.SendTypingIndicator(recipientJID.String(), "paused")
	c.antiban.AfterSend(antiban.Text)

	content := ExtractTextContent(msg)
	mediaType, filename, url, directPath, mediaKey, fileSHA256, fileEncSHA256, fileLength := ExtractMediaInfo(msg)
	sender := ""
	if c.Store != nil && c.Store.ID != nil {
		sender = c.Store.ID.ToNonAD().String()
	}

	_ = messageStore.StoreMessage(
		sendResp.ID, // Use the ID from SendResponse
		recipientJID.String(),
		sender,
		sender,
		content,
		sendResp.Timestamp, // Use the Timestamp from SendResponse
		true,               // IsFromMe is true since we are sending this message
		mediaType,
		filename,
		url,
		directPath,
		mediaKey,
		fileSHA256,
		fileEncSHA256,
		fileLength,
	)

	return bridgeTypes.SendResult{
		Success:   true,
		MessageID: string(sendResp.ID),
		Timestamp: sendResp.Timestamp,
	}
}

// SendReaction sends an emoji reaction to a message.
// It looks up the original message sender from the store so that
// BuildReaction constructs the correct MessageKey (FromMe flag).
func (c *Client) SendReaction(messageStore *database.MessageStore, chatJID, messageID, emoji string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	msgID := types.MessageID(messageID)

	// Look up the actual sender of the message being reacted to.
	// BuildReaction needs the original message sender to set FromMe correctly.
	senderJID := c.Store.ID.ToNonAD() // default: assume own message
	if messageStore != nil {
		senderStr, isFromMe, lookupErr := messageStore.GetMessageSender(messageID, chatJID)
		if lookupErr == nil && !isFromMe {
			if parsed, parseErr := types.ParseJID(senderStr); parseErr == nil {
				senderJID = parsed.ToNonAD()
			}
		}
	}

	msg := c.Client.BuildReaction(chat, senderJID, msgID, emoji)

	if err := c.antiban.BeforeSend(context.Background(), antiban.Reaction, 0); err != nil {
		return fmt.Errorf("antiban blocked reaction: %v", err)
	}
	_, err = c.Client.SendMessage(context.Background(), chat, msg)
	if err != nil {
		c.antiban.AfterSendFailed(antiban.Reaction, err)
		return fmt.Errorf("failed to send reaction: %v", err)
	}
	c.antiban.AfterSend(antiban.Reaction)

	return nil
}

// EditMessage edits a previously sent message
func (c *Client) EditMessage(chatJID, messageID, newContent string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	msgID := types.MessageID(messageID)

	newMsg := &waE2E.Message{
		Conversation: proto.String(newContent),
	}
	msg := c.Client.BuildEdit(chat, msgID, newMsg)

	if c.antiban.Enabled() {
		_ = c.SendTypingIndicator(chatJID, "typing")
	}
	if err := c.antiban.BeforeSend(context.Background(), antiban.Edit, len(newContent)); err != nil {
		_ = c.SendTypingIndicator(chatJID, "paused")
		return fmt.Errorf("antiban blocked edit: %v", err)
	}
	_, err = c.Client.SendMessage(context.Background(), chat, msg)
	_ = c.SendTypingIndicator(chatJID, "paused")
	if err != nil {
		c.antiban.AfterSendFailed(antiban.Edit, err)
		return fmt.Errorf("failed to edit message: %v", err)
	}
	c.antiban.AfterSend(antiban.Edit)

	return nil
}

// DeleteMessage revokes/deletes a message
func (c *Client) DeleteMessage(chatJID, messageID, senderJID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	msgID := types.MessageID(messageID)

	var sender types.JID
	if senderJID == "" {
		sender = c.Store.ID.ToNonAD() // own message
	} else {
		sender, err = types.ParseJID(senderJID)
		if err != nil {
			return fmt.Errorf("invalid sender JID: %v", err)
		}
	}

	msg := c.Client.BuildRevoke(chat, sender, msgID)

	if err := c.antiban.BeforeSend(context.Background(), antiban.Delete, 0); err != nil {
		return fmt.Errorf("antiban blocked delete: %v", err)
	}
	_, err = c.Client.SendMessage(context.Background(), chat, msg)
	if err != nil {
		c.antiban.AfterSendFailed(antiban.Delete, err)
		return fmt.Errorf("failed to delete message: %v", err)
	}
	c.antiban.AfterSend(antiban.Delete)

	return nil
}

// GetGroupInfo retrieves group metadata
func (c *Client) GetGroupInfo(groupJID string) (*types.GroupInfo, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to WhatsApp")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %v", err)
	}

	return c.Client.GetGroupInfo(context.Background(), jid)
}

// MarkMessagesRead marks messages as read
func (c *Client) MarkMessagesRead(chatJID string, messageIDs []string, senderJID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	ids := make([]types.MessageID, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = types.MessageID(id)
	}

	var sender types.JID
	if senderJID != "" {
		sender, err = types.ParseJID(senderJID)
		if err != nil {
			return fmt.Errorf("invalid sender JID: %v", err)
		}
	}

	return c.Client.MarkRead(context.Background(), ids, time.Now(), chat, sender)
}

// Phase 2: Group Management

// CreateGroup creates a new WhatsApp group
func (c *Client) CreateGroup(name string, participants []string) (*types.GroupInfo, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to WhatsApp")
	}

	// Parse participant JIDs
	participantJIDs := make([]types.JID, len(participants))
	for i, p := range participants {
		jid, err := types.ParseJID(p)
		if err != nil {
			return nil, fmt.Errorf("invalid participant JID %s: %v", p, err)
		}
		participantJIDs[i] = jid
	}

	req := whatsmeow.ReqCreateGroup{
		Name:         name,
		Participants: participantJIDs,
	}

	return c.Client.CreateGroup(context.Background(), req)
}

// AddGroupParticipants adds members to a group
func (c *Client) AddGroupParticipants(groupJID string, participants []string) ([]types.GroupParticipant, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %v", err)
	}

	participantJIDs := make([]types.JID, len(participants))
	for i, p := range participants {
		jid, err := types.ParseJID(p)
		if err != nil {
			return nil, fmt.Errorf("invalid participant JID %s: %v", p, err)
		}
		participantJIDs[i] = jid
	}

	return c.Client.UpdateGroupParticipants(context.Background(), group, participantJIDs, whatsmeow.ParticipantChangeAdd)
}

// RemoveGroupParticipants removes members from a group
func (c *Client) RemoveGroupParticipants(groupJID string, participants []string) ([]types.GroupParticipant, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %v", err)
	}

	participantJIDs := make([]types.JID, len(participants))
	for i, p := range participants {
		jid, err := types.ParseJID(p)
		if err != nil {
			return nil, fmt.Errorf("invalid participant JID %s: %v", p, err)
		}
		participantJIDs[i] = jid
	}

	return c.Client.UpdateGroupParticipants(context.Background(), group, participantJIDs, whatsmeow.ParticipantChangeRemove)
}

// PromoteGroupParticipant promotes a participant to admin
func (c *Client) PromoteGroupParticipant(groupJID string, participant string) ([]types.GroupParticipant, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %v", err)
	}

	jid, err := types.ParseJID(participant)
	if err != nil {
		return nil, fmt.Errorf("invalid participant JID: %v", err)
	}

	return c.Client.UpdateGroupParticipants(context.Background(), group, []types.JID{jid}, whatsmeow.ParticipantChangePromote)
}

// DemoteGroupParticipant demotes an admin to regular participant
func (c *Client) DemoteGroupParticipant(groupJID string, participant string) ([]types.GroupParticipant, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %v", err)
	}

	jid, err := types.ParseJID(participant)
	if err != nil {
		return nil, fmt.Errorf("invalid participant JID: %v", err)
	}

	return c.Client.UpdateGroupParticipants(context.Background(), group, []types.JID{jid}, whatsmeow.ParticipantChangeDemote)
}

// LeaveGroup leaves a WhatsApp group
func (c *Client) LeaveGroup(groupJID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %v", err)
	}

	return c.Client.LeaveGroup(context.Background(), group)
}

// SetGroupName updates the group name
func (c *Client) SetGroupName(groupJID string, name string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %v", err)
	}

	return c.Client.SetGroupName(context.Background(), group, name)
}

// SetGroupTopic updates the group description/topic
func (c *Client) SetGroupTopic(groupJID string, topic string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	group, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid group JID: %v", err)
	}

	return c.Client.SetGroupTopic(context.Background(), group, "", "", topic)
}

// Phase 3: Polls

// CreatePoll creates and sends a poll to a chat
func (c *Client) CreatePoll(chatJID string, question string, options []string, multiSelect bool) (bridgeTypes.SendResult, error) {
	if !c.IsConnected() {
		return bridgeTypes.SendResult{Success: false, Error: "not connected to WhatsApp"}, fmt.Errorf("not connected to WhatsApp")
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("invalid chat JID: %v", err)}, err
	}

	// Determine selectable count based on multiSelect
	selectableCount := 1
	if multiSelect {
		selectableCount = len(options)
	}

	// Build poll creation message
	pollMsg := c.Client.BuildPollCreation(question, options, selectableCount)

	// Antiban: typing indicator + rate limit
	if c.antiban.Enabled() {
		_ = c.SendTypingIndicator(chatJID, "typing")
	}
	if err := c.antiban.BeforeSend(context.Background(), antiban.Poll, len(question)); err != nil {
		_ = c.SendTypingIndicator(chatJID, "paused")
		return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("antiban blocked poll: %v", err)}, err
	}

	// Send the poll
	resp, err := c.Client.SendMessage(context.Background(), chat, pollMsg)
	_ = c.SendTypingIndicator(chatJID, "paused")
	if err != nil {
		c.antiban.AfterSendFailed(antiban.Poll, err)
		return bridgeTypes.SendResult{Success: false, Error: fmt.Sprintf("failed to send poll: %v", err)}, err
	}
	c.antiban.AfterSend(antiban.Poll)

	return bridgeTypes.SendResult{
		Success:   true,
		MessageID: string(resp.ID),
		Timestamp: resp.Timestamp,
	}, nil
}

// Phase 4: On-Demand History Request

// RequestChatHistory requests older messages for a specific chat.
// The response will come asynchronously via the HistorySync event handler.
// This requires knowing the oldest message in the chat to request messages before it.
func (c *Client) RequestChatHistory(chatJID string, oldestMsgID string, oldestMsgFromMe bool, oldestMsgSender string, oldestMsgTimestamp int64, count int) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	chat, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID: %v", err)
	}

	ownJID := c.Store.ID.ToNonAD()

	// Create MessageInfo for the oldest known message
	msgInfo := &types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     chat,
			IsFromMe: oldestMsgFromMe,
		},
		ID:        types.MessageID(oldestMsgID),
		Timestamp: time.UnixMilli(oldestMsgTimestamp),
	}

	// Set sender correctly: fromMe uses own JID, incoming uses provided sender JID
	if oldestMsgFromMe {
		msgInfo.MessageSource.Sender = ownJID
	} else if oldestMsgSender != "" {
		senderJID, parseErr := types.ParseJID(oldestMsgSender)
		if parseErr == nil {
			msgInfo.MessageSource.Sender = senderJID
		} else {
			msgInfo.MessageSource.Sender = ownJID
		}
	} else {
		msgInfo.MessageSource.Sender = ownJID
	}

	// Build the history sync request
	// Recommended count is 50 messages at a time
	if count <= 0 || count > 50 {
		count = 50
	}

	msg := c.Client.BuildHistorySyncRequest(msgInfo, count)

	// Send as peer message to own device (NOT to the group/chat).
	// History sync is a device-to-device protocol request to our own phone.
	if err := c.antiban.BeforeSend(context.Background(), antiban.Peer, 0); err != nil {
		return fmt.Errorf("antiban blocked history request: %v", err)
	}
	_, err = c.Client.SendMessage(context.Background(), ownJID, msg, whatsmeow.SendRequestExtra{Peer: true})
	if err != nil {
		c.antiban.AfterSendFailed(antiban.Peer, err)
		return fmt.Errorf("failed to send history request: %v", err)
	}
	c.antiban.AfterSend(antiban.Peer)

	return nil
}
