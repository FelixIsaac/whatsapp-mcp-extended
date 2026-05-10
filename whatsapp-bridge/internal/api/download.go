package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/hkdf"
)

// maxMediaBytes caps the encrypted payload accepted from the WhatsApp CDN to
// bound memory use per request. WhatsApp's largest documented media (documents)
// is 64 MiB; 100 MiB leaves headroom for the trailing MAC and future limits.
const maxMediaBytes = 100 << 20

// hkdfInfo maps WhatsApp media types to the HKDF info string used for key derivation.
var hkdfInfo = map[string][]byte{
	"image":    []byte("WhatsApp Image Keys"),
	"video":    []byte("WhatsApp Video Keys"),
	"audio":    []byte("WhatsApp Audio Keys"),
	"document": []byte("WhatsApp Document Keys"),
}

var mediaExt = map[string]string{
	"image":    ".jpg",
	"video":    ".mp4",
	"audio":    ".ogg",
	"document": ".bin",
}

type downloadRequest struct {
	MessageID string `json:"message_id"`
	ChatJID   string `json:"chat_jid"`
}

// handleDownload handles POST /api/download — fetches and decrypts WhatsApp media
// using the URL/MediaKey/MediaType stored in the bridge's SQLite for the given message.
//
// Request body:
//   - message_id: WhatsApp message ID (required)
//   - chat_jid:   chat JID containing the message (required)
//
// Response: { success: bool, path: string, size: int, message?: string }
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	if req.MessageID == "" || req.ChatJID == "" {
		SendJSONError(w, "message_id and chat_jid are required", http.StatusBadRequest)
		return
	}

	var (
		mediaType, url, filename string
		mediaKey                 []byte
	)
	row := s.messageStore.GetDB().QueryRow(
		`SELECT media_type, url, media_key, COALESCE(filename, '')
		 FROM messages WHERE id = ? AND chat_jid = ?`,
		req.MessageID, req.ChatJID,
	)
	if err := row.Scan(&mediaType, &url, &mediaKey, &filename); err != nil {
		SendJSONError(w, "message not found: "+err.Error(), http.StatusNotFound)
		return
	}
	info, ok := hkdfInfo[mediaType]
	if !ok || url == "" || len(mediaKey) == 0 {
		SendJSONError(w, "no downloadable media for this message", http.StatusBadRequest)
		return
	}

	// Fetch encrypted media from WhatsApp CDN, bounded by request context and total timeout.
	httpClient := &http.Client{Timeout: 60 * time.Second}
	greq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		SendJSONError(w, "bad CDN URL: "+err.Error(), http.StatusBadGateway)
		return
	}
	greq.Header.Set("User-Agent", "WhatsApp/2.24.0")
	resp, err := httpClient.Do(greq)
	if err != nil {
		SendJSONError(w, "fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		SendJSONError(w, fmt.Sprintf("CDN returned HTTP %d", resp.StatusCode), http.StatusBadGateway)
		return
	}
	enc, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaBytes+1))
	if err != nil {
		SendJSONError(w, "read failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	if int64(len(enc)) > maxMediaBytes {
		SendJSONError(w, "media exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}
	if len(enc) < 26 {
		SendJSONError(w, "ciphertext too short", http.StatusBadGateway)
		return
	}

	// Derive keys: HKDF-SHA256(mediaKey, salt=zero32, info, 112 bytes).
	expanded := make([]byte, 112)
	kdf := hkdf.New(sha256.New, mediaKey, make([]byte, 32), info)
	if _, err := io.ReadFull(kdf, expanded); err != nil {
		SendJSONError(w, "hkdf failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	iv, cipherKey, macKey := expanded[:16], expanded[16:48], expanded[48:80]

	body, mac := enc[:len(enc)-10], enc[len(enc)-10:]
	hm := hmac.New(sha256.New, macKey)
	hm.Write(iv)
	hm.Write(body)
	expectedMAC := hm.Sum(nil)[:10]
	if !hmac.Equal(mac, expectedMAC) {
		SendJSONError(w, "MAC verification failed", http.StatusBadRequest)
		return
	}

	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		SendJSONError(w, "cipher init: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if len(body) == 0 || len(body)%aes.BlockSize != 0 {
		SendJSONError(w, "ciphertext not block-aligned", http.StatusBadRequest)
		return
	}
	plain := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, body)
	// Strict PKCS#7 unpad: validate every padding byte equals the pad length.
	pad := int(plain[len(plain)-1])
	if pad <= 0 || pad > aes.BlockSize || pad > len(plain) {
		SendJSONError(w, "bad PKCS7 padding", http.StatusInternalServerError)
		return
	}
	for i := len(plain) - pad; i < len(plain); i++ {
		if int(plain[i]) != pad {
			SendJSONError(w, "bad PKCS7 padding", http.StatusInternalServerError)
			return
		}
	}
	plain = plain[:len(plain)-pad]

	// Persist under store/media/<chat_jid>/<message_id><ext> (relative to bridge cwd).
	storeDir := filepath.Join("store", "media", sanitizePath(req.ChatJID))
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		SendJSONError(w, "mkdir failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	ext := mediaExt[mediaType]
	if mediaType == "document" && filename != "" {
		if e := filepath.Ext(filename); e != "" {
			ext = e
		}
	}
	outPath := filepath.Join(storeDir, sanitizePath(req.MessageID)+ext)
	// Atomic write: write to a sibling tmp file, then rename into place so partial
	// writes never become visible and concurrent requests for the same message
	// can't tear each other's output.
	tmpPath := outPath + ".tmp"
	if err := os.WriteFile(tmpPath, plain, 0o644); err != nil {
		SendJSONError(w, "write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		SendJSONError(w, "rename failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"path":    outPath,
		"size":    len(plain),
	})
}

// sanitizePath replaces every byte outside a strict allowlist with '_', so untrusted
// IDs can't escape the store dir, hide files via leading dots, or smuggle control bytes.
// WhatsApp message IDs and JIDs are constrained to a subset of these characters.
func sanitizePath(s string) string {
	if s == "" {
		return "_"
	}
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == '_', c == '-', c == '@':
			b[i] = c
		default:
			b[i] = '_'
		}
	}
	if b[0] == '.' || b[0] == '-' {
		b[0] = '_'
	}
	return string(b)
}
