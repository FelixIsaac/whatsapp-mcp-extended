package database

import (
	"database/sql"
	"regexp"
	"strings"
	"time"

	"whatsapp-bridge/internal/types"
)

// StoreChat stores a chat in the database
func (store *MessageStore) StoreChat(jid, name string, lastMessageTime time.Time) error {
	_, err := store.db.Exec(
		"INSERT OR REPLACE INTO chats (jid, name, last_message_time) VALUES (?, ?, ?)",
		jid, name, lastMessageTime,
	)
	return err
}

// StoreMessage stores a message in the database
func (store *MessageStore) StoreMessage(id, chatJID, sender, senderName, content string, timestamp time.Time, isFromMe bool,
	mediaType, filename, url string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error {
	// Only store if there's actual content or media
	if content == "" && mediaType == "" {
		return nil
	}

	// Use sender JID as fallback if senderName is empty
	if senderName == "" {
		senderName = sender
	}

	_, err := store.db.Exec(
		`INSERT OR REPLACE INTO messages
		(id, chat_jid, sender, sender_name, content, timestamp, is_from_me, media_type, filename, url, media_key, file_sha256, file_enc_sha256, file_length)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, chatJID, sender, senderName, content, timestamp, isFromMe, mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength,
	)
	return err
}

// GetMessages gets messages from a chat
func (store *MessageStore) GetMessages(chatJID string, limit int) ([]types.Message, error) {
	rows, err := store.db.Query(
		"SELECT sender, sender_name, content, timestamp, is_from_me, media_type, filename FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC LIMIT ?",
		chatJID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var msg types.Message
		var timestamp time.Time
		var senderName sql.NullString
		err := rows.Scan(&msg.Sender, &senderName, &msg.Content, &timestamp, &msg.IsFromMe, &msg.MediaType, &msg.Filename)
		if err != nil {
			return nil, err
		}
		msg.Time = timestamp
		if senderName.Valid {
			msg.SenderName = senderName.String
		} else {
			msg.SenderName = msg.Sender // fallback to JID
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessageCount returns total message count.
func (store *MessageStore) GetMessageCount() (int, error) {
	var count int
	err := store.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

// GetChatCount returns total chat count.
func (store *MessageStore) GetChatCount() (int, error) {
	var count int
	err := store.db.QueryRow("SELECT COUNT(*) FROM chats").Scan(&count)
	return count, err
}

// GetChats gets all chats
func (store *MessageStore) GetChats() (map[string]time.Time, error) {
	rows, err := store.db.Query("SELECT jid, last_message_time FROM chats ORDER BY last_message_time DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := make(map[string]time.Time)
	for rows.Next() {
		var jid string
		var lastMessageTime time.Time
		err := rows.Scan(&jid, &lastMessageTime)
		if err != nil {
			return nil, err
		}
		chats[jid] = lastMessageTime
	}

	return chats, nil
}

// GetCharacterCount returns character count of message content.
// Counts all characters including spaces and punctuation.
func GetCharacterCount(content string) int {
	return len([]rune(content))
}

// GetWordCount returns approximate word count using whitespace splitting.
// Words are sequences of characters separated by whitespace.
func GetWordCount(content string) int {
	if content == "" {
		return 0
	}
	words := strings.Fields(content)
	return len(words)
}

// ExtractURLs extracts all URLs from content using regex.
// Matches http://, https://, and common domain patterns.
func ExtractURLs(content string) []string {
	if content == "" {
		return []string{}
	}

	// Pattern matches http(s)://, ftp://, and www. URLs
	urlPattern := regexp.MustCompile(`(?:https?|ftp)://[^\s]+|www\.[^\s]+`)
	matches := urlPattern.FindAllString(content, -1)

	if matches == nil {
		return []string{}
	}
	return matches
}

// ExtractMentions extracts @mentions from content.
// Looks for @JID patterns or common @username patterns.
func ExtractMentions(content string) []string {
	if content == "" {
		return []string{}
	}

	// Pattern matches @-prefixed text (JIDs or usernames)
	mentionPattern := regexp.MustCompile(`@[a-zA-Z0-9._-]+(?:@[a-z.]+)?`)
	matches := mentionPattern.FindAllString(content, -1)

	if matches == nil {
		return []string{}
	}
	return matches
}

// GetMessageCountForPeriod returns message count for chat in a time period.
// Period is specified in days (e.g., 1 for today, 7 for last 7 days).
func (store *MessageStore) GetMessageCountForPeriod(chatJID string, days int) (int, error) {
	query := `
	SELECT COUNT(*) FROM messages
	WHERE chat_jid = ? AND datetime(timestamp) >= datetime('now', '-' || ? || ' days')
	`
	var count int
	err := store.db.QueryRow(query, chatJID, days).Scan(&count)
	return count, err
}

// GetContactInfo retrieves contact information by JID from whatsmeow_contacts and contact_nicknames.
func (store *MessageStore) GetContactInfo(jid string) (*types.ContactInfo, error) {
	var nickname sql.NullString

	// Query contact_nicknames for custom nickname
	nicknameQuery := `SELECT nickname FROM contact_nicknames WHERE jid = ? LIMIT 1`
	_ = store.db.QueryRow(nicknameQuery, jid).Scan(&nickname)

	// Contact info will be populated with just JID if not found in contacts DB
	contact := &types.ContactInfo{
		JID: jid,
	}

	// Try to extract phone number from JID if it's a regular contact
	if !strings.HasSuffix(jid, "@g.us") {
		// Extract phone number from JID format: {phone}@s.whatsapp.net
		parts := strings.Split(jid, "@")
		if len(parts) > 0 {
			contact.PhoneNum = parts[0]
		}
	}

	// Use nickname if found
	if nickname.Valid {
		contact.Nickname = nickname.String
		contact.Name = nickname.String
	}

	return contact, nil
}

// GetMessageReactionSummary returns a map of emoji -> count for message reactions.
func (store *MessageStore) GetMessageReactionSummary(chatJID, msgID string) (map[string]int, error) {
	// This is a placeholder - reactions storage depends on whatsmeow implementation
	// For now, return empty map as reactions aren't stored in Phase 1
	return make(map[string]int), nil
}

// GetGroupParticipants retrieves all participants in a group with their details.
func (store *MessageStore) GetGroupParticipants(groupJID string) ([]*types.ContactInfo, error) {
	// This requires group_members table which will be added in Phase 2
	// For now, return empty slice
	return []*types.ContactInfo{}, nil
}

// GetGroupAdmins retrieves list of admin JIDs for a group.
func (store *MessageStore) GetGroupAdmins(groupJID string) ([]string, error) {
	// This requires group_members table which will be added in Phase 2
	// For now, return empty slice
	return []string{}, nil
}

// GetMostActiveGroupMember returns the name and message count of the most active group member.
func (store *MessageStore) GetMostActiveGroupMember(groupJID string) (string, int, error) {
	query := `
	SELECT sender_name, COUNT(*) as count
	FROM messages
	WHERE chat_jid = ? AND sender_name != ''
	GROUP BY sender
	ORDER BY count DESC
	LIMIT 1
	`
	var name string
	var count int
	err := store.db.QueryRow(query, groupJID).Scan(&name, &count)
	if err == sql.ErrNoRows {
		return "", 0, nil
	}
	return name, count, err
}

// GetMediaCountByType returns count of media messages by type (image, video, audio, document).
func (store *MessageStore) GetMediaCountByType(chatJID string) (map[string]int, error) {
	query := `
	SELECT media_type, COUNT(*) as count
	FROM messages
	WHERE chat_jid = ? AND media_type != '' AND media_type IS NOT NULL
	GROUP BY media_type
	`
	rows, err := store.db.Query(query, chatJID)
	if err != nil {
		return make(map[string]int), err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var mediaType string
		var count int
		if err := rows.Scan(&mediaType, &count); err != nil {
			return make(map[string]int), err
		}
		result[mediaType] = count
	}
	return result, rows.Err()
}

// GetRecentMedia returns filenames of recent media messages in a chat.
func (store *MessageStore) GetRecentMedia(chatJID string, limit int) ([]string, error) {
	query := `
	SELECT filename
	FROM messages
	WHERE chat_jid = ? AND media_type != '' AND media_type IS NOT NULL AND filename != '' AND filename IS NOT NULL
	ORDER BY timestamp DESC
	LIMIT ?
	`
	rows, err := store.db.Query(query, chatJID, limit)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return []string{}, err
		}
		result = append(result, filename)
	}
	return result, rows.Err()
}

// GetPreviousMessageTime returns the timestamp of the message sent immediately before the given time in same chat.
func (store *MessageStore) GetPreviousMessageTime(chatJID string, currentTime time.Time) (time.Time, error) {
	query := `
	SELECT timestamp
	FROM messages
	WHERE chat_jid = ? AND timestamp < ?
	ORDER BY timestamp DESC
	LIMIT 1
	`
	var prevTime time.Time
	err := store.db.QueryRow(query, chatJID, currentTime).Scan(&prevTime)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return prevTime, err
}
