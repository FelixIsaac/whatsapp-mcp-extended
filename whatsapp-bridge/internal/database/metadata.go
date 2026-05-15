package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GetChatMetadata retrieves extended metadata for a specific chat.
// Returns message counts, media counts, and other statistics.
func (store *MessageStore) GetChatMetadata(chatJID string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	// Total message count
	var totalCount int
	err := store.db.QueryRow(
		"SELECT COUNT(*) FROM messages WHERE chat_jid = ?",
		chatJID,
	).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total message count: %w", err)
	}
	metadata["total_message_count"] = totalCount

	// Message count today
	todayCount := 0
	err = store.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE chat_jid = ? AND date(timestamp) = date('now')
	`, chatJID).Scan(&todayCount)
	if err != nil {
		todayCount = 0
	}
	metadata["message_count_today"] = todayCount

	// Message count last 7 days
	last7Count := 0
	err = store.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE chat_jid = ? AND datetime(timestamp) >= datetime('now', '-7 days')
	`, chatJID).Scan(&last7Count)
	if err != nil {
		last7Count = 0
	}
	metadata["message_count_last_7_days"] = last7Count

	// Media count by type
	var mediaType sql.NullString
	var count int
	rows, err := store.db.Query(`
		SELECT media_type, COUNT(*) as count FROM messages
		WHERE chat_jid = ? AND media_type != ''
		GROUP BY media_type
	`, chatJID)
	if err == nil {
		defer rows.Close()
		mediaCountByType := make(map[string]int)
		for rows.Next() {
			if err := rows.Scan(&mediaType, &count); err == nil && mediaType.Valid {
				mediaCountByType[mediaType.String] = count
			}
		}
		if len(mediaCountByType) > 0 {
			metadata["media_count_by_type"] = mediaCountByType
		}
	}

	// Last message metadata
	var lastSender sql.NullString
	var lastTimestamp sql.NullTime
	var lastContent sql.NullString
	err = store.db.QueryRow(`
		SELECT sender, timestamp, content FROM messages
		WHERE chat_jid = ?
		ORDER BY timestamp DESC LIMIT 1
	`, chatJID).Scan(&lastSender, &lastTimestamp, &lastContent)
	if err == nil {
		if lastTimestamp.Valid {
			metadata["last_message_timestamp"] = lastTimestamp.Time
			metadata["last_message_time_ago"] = formatTimeAgo(lastTimestamp.Time)
		}
		if lastContent.Valid {
			metadata["last_message_preview"] = truncateString(lastContent.String, 100)
		}
	}

	return metadata, nil
}

// GetContactMetadata retrieves extended metadata for a specific contact.
// Returns message counts, activity trends, and communication patterns.
func (store *MessageStore) GetContactMetadata(senderJID string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	// Total message count from this contact
	var totalCount int
	err := store.db.QueryRow(
		"SELECT COUNT(*) FROM messages WHERE sender = ?",
		senderJID,
	).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}
	metadata["total_message_count"] = totalCount

	// Message count today
	todayCount := 0
	err = store.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE sender = ? AND date(timestamp) = date('now')
	`, senderJID).Scan(&todayCount)
	if err != nil {
		todayCount = 0
	}
	metadata["message_count_today"] = todayCount

	// Message count last 7 days
	last7Count := 0
	err = store.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE sender = ? AND datetime(timestamp) >= datetime('now', '-7 days')
	`, senderJID).Scan(&last7Count)
	if err != nil {
		last7Count = 0
	}
	metadata["message_count_last_7_days"] = last7Count

	// Message count last 30 days
	last30Count := 0
	err = store.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE sender = ? AND datetime(timestamp) >= datetime('now', '-30 days')
	`, senderJID).Scan(&last30Count)
	if err != nil {
		last30Count = 0
	}
	metadata["message_count_last_30_days"] = last30Count

	// Days since last message
	var lastTimestamp sql.NullTime
	err = store.db.QueryRow(`
		SELECT MAX(timestamp) FROM messages WHERE sender = ?
	`, senderJID).Scan(&lastTimestamp)
	if err == nil && lastTimestamp.Valid {
		daysSince := int(time.Since(lastTimestamp.Time).Hours() / 24)
		metadata["days_since_last_message"] = daysSince
		metadata["last_message_timestamp"] = lastTimestamp.Time
	}

	// Latest message preview
	var lastContent sql.NullString
	err = store.db.QueryRow(`
		SELECT content FROM messages
		WHERE sender = ?
		ORDER BY timestamp DESC LIMIT 1
	`, senderJID).Scan(&lastContent)
	if err == nil && lastContent.Valid {
		metadata["latest_message_preview"] = truncateString(lastContent.String, 100)
	}

	return metadata, nil
}

// Helper function to format time difference as human-readable string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d minute(s) ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hour(s) ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d day(s) ago", days)
	default:
		weeks := int(duration.Hours() / (24 * 7))
		return fmt.Sprintf("%d week(s) ago", weeks)
	}
}

// Helper function to truncate string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
