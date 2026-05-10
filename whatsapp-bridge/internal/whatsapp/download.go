package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"

	"whatsapp-bridge/internal/database"
)

// ErrNoMedia is returned when the message exists but carries no media.
var ErrNoMedia = errors.New("message has no media")

// ErrMessageNotFound is returned when the message id+jid is not in the store.
var ErrMessageNotFound = errors.New("message not found")

// DownloadMessageMedia decrypts and saves media for a stored message.
// Returns the absolute path on disk and the media type ("image"/"video"/"audio"/"document").
func (c *Client) DownloadMessageMedia(ctx context.Context, store *database.MessageStore, mediaDir, messageID, chatJID string) (string, string, error) {
	mediaType, filename, url, directPath, mediaKey, fileSHA256, fileEncSHA256, fileLength, err := store.GetMessageMedia(messageID, chatJID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrMessageNotFound
		}
		return "", "", fmt.Errorf("lookup message: %w", err)
	}
	if mediaType == "" {
		return "", "", ErrNoMedia
	}

	urlCopy := url
	directPathCopy := directPath
	lengthCopy := fileLength
	var msg whatsmeow.DownloadableMessage
	switch mediaType {
	case "image":
		msg = &waE2E.ImageMessage{
			URL: &urlCopy, DirectPath: &directPathCopy, MediaKey: mediaKey,
			FileSHA256: fileSHA256, FileEncSHA256: fileEncSHA256, FileLength: &lengthCopy,
		}
	case "video":
		msg = &waE2E.VideoMessage{
			URL: &urlCopy, DirectPath: &directPathCopy, MediaKey: mediaKey,
			FileSHA256: fileSHA256, FileEncSHA256: fileEncSHA256, FileLength: &lengthCopy,
		}
	case "audio":
		msg = &waE2E.AudioMessage{
			URL: &urlCopy, DirectPath: &directPathCopy, MediaKey: mediaKey,
			FileSHA256: fileSHA256, FileEncSHA256: fileEncSHA256, FileLength: &lengthCopy,
		}
	case "document":
		msg = &waE2E.DocumentMessage{
			URL: &urlCopy, DirectPath: &directPathCopy, MediaKey: mediaKey,
			FileSHA256: fileSHA256, FileEncSHA256: fileEncSHA256, FileLength: &lengthCopy,
		}
	default:
		return "", "", fmt.Errorf("unsupported media type %q", mediaType)
	}

	data, err := c.Client.Download(ctx, msg)
	if err != nil {
		return "", "", fmt.Errorf("whatsmeow download: %w", err)
	}

	// Strip path separators defensively before joining.
	safeJID := strings.ReplaceAll(chatJID, "/", "_")
	safeName := strings.ReplaceAll(filename, "/", "_")
	if safeName == "" {
		safeName = messageID
	}
	dir := filepath.Join(mediaDir, safeJID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	out := filepath.Join(dir, safeName)
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", "", fmt.Errorf("write %s: %w", out, err)
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return out, mediaType, nil
	}
	return abs, mediaType, nil
}
