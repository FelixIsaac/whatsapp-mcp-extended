package whatsapp

import (
	"os"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow"
)

func TestValidateMediaPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{"empty path", "", false, ""},
		{"allowed /app/media", "/app/media/file.jpg", false, ""},
		{"allowed /app/store", "/app/store/data.db", false, ""},
		{"allowed /tmp", "/tmp/upload.png", false, ""},
		{"path traversal ../", "/app/media/../etc/passwd", true, "traversal"},
		{"path traversal multiple", "/app/media/../../etc/passwd", true, "traversal"},
		{"path traversal in middle", "/app/../../../etc/passwd", true, "traversal"},
		{"outside allowed /etc", "/etc/passwd", true, "outside allowed"},
		{"outside allowed /home", "/home/user/file.txt", true, "outside allowed"},
		{"outside allowed /var", "/var/log/syslog", true, "outside allowed"},
	}

	os.Unsetenv("DISABLE_PATH_CHECK")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMediaPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateMediaPath(%s) = nil, want error containing %q", tt.path, tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("validateMediaPath(%s) error = %v, want error containing %q", tt.path, err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("validateMediaPath(%s) = %v, want nil", tt.path, err)
			}
		})
	}
}

func TestValidateMediaPath_DisableCheck(t *testing.T) {
	original := os.Getenv("DISABLE_PATH_CHECK")
	defer os.Setenv("DISABLE_PATH_CHECK", original)

	os.Setenv("DISABLE_PATH_CHECK", "true")

	if err := validateMediaPath("/home/user/file.txt"); err != nil {
		t.Errorf("With DISABLE_PATH_CHECK=true, validateMediaPath should allow external paths, got: %v", err)
	}
}

func TestValidateMediaPath_TraversalAlwaysBlocked(t *testing.T) {
	original := os.Getenv("DISABLE_PATH_CHECK")
	defer os.Setenv("DISABLE_PATH_CHECK", original)

	os.Setenv("DISABLE_PATH_CHECK", "true")

	if err := validateMediaPath("/app/media/../../../etc/passwd"); err == nil {
		t.Error("Path traversal should be blocked even with DISABLE_PATH_CHECK=true")
	}
}

func TestMediaTypeAndMimeType(t *testing.T) {
	tests := []struct {
		name          string
		mediaPath     string
		wantMediaType whatsmeow.MediaType
		wantMimeType  string
	}{
		{
			name:          "pdf document",
			mediaPath:     "/tmp/report.pdf",
			wantMediaType: whatsmeow.MediaDocument,
			wantMimeType:  "application/pdf",
		},
		{
			name:          "docx document",
			mediaPath:     "/tmp/spec.DOCX",
			wantMediaType: whatsmeow.MediaDocument,
			wantMimeType:  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:          "xlsx document",
			mediaPath:     "/tmp/budget.xlsx",
			wantMediaType: whatsmeow.MediaDocument,
			wantMimeType:  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		},
		{
			name:          "cbz document",
			mediaPath:     "/tmp/comic.cbz",
			wantMediaType: whatsmeow.MediaDocument,
			wantMimeType:  "application/x-cbz",
		},
		{
			name:          "jpg image",
			mediaPath:     "/tmp/photo.jpg",
			wantMediaType: whatsmeow.MediaImage,
			wantMimeType:  "image/jpeg",
		},
		{
			name:          "unknown extension fallback",
			mediaPath:     "/tmp/archive.weird",
			wantMediaType: whatsmeow.MediaDocument,
			wantMimeType:  "application/octet-stream",
		},
		{
			name:          "no extension fallback",
			mediaPath:     "/tmp/readme",
			wantMediaType: whatsmeow.MediaDocument,
			wantMimeType:  "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMediaType, gotMimeType := mediaTypeAndMimeType(tt.mediaPath)
			if gotMediaType != tt.wantMediaType {
				t.Fatalf("mediaTypeAndMimeType() mediaType = %v, want %v", gotMediaType, tt.wantMediaType)
			}
			if gotMimeType != tt.wantMimeType {
				t.Fatalf("mediaTypeAndMimeType() mimeType = %q, want %q", gotMimeType, tt.wantMimeType)
			}
		})
	}
}

func TestDocumentFileName(t *testing.T) {
	tests := []struct {
		name      string
		mediaPath string
		want      string
	}{
		{
			name:      "unix path",
			mediaPath: "/tmp/Presidio-statement-May-2026.pdf",
			want:      "Presidio-statement-May-2026.pdf",
		},
		{
			name:      "windows path",
			mediaPath: `C:\tmp\Quarterly Report.xlsx`,
			want:      "Quarterly Report.xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := documentFileName(tt.mediaPath)
			if got != tt.want {
				t.Fatalf("documentFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}
