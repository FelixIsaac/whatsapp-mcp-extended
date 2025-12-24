package whatsapp

import (
	"os"
	"strings"
	"testing"
)

func TestValidateMediaPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		// Valid paths
		{"empty path", "", false, ""},
		{"allowed /app/media", "/app/media/file.jpg", false, ""},
		{"allowed /app/store", "/app/store/data.db", false, ""},
		{"allowed /tmp", "/tmp/upload.png", false, ""},

		// Path traversal attempts (should fail)
		{"path traversal ../", "/app/media/../etc/passwd", true, "traversal"},
		{"path traversal multiple", "/app/media/../../etc/passwd", true, "traversal"},
		{"path traversal in middle", "/app/../../../etc/passwd", true, "traversal"},

		// Outside allowed directories (should fail when DISABLE_PATH_CHECK is not set)
		{"outside allowed /etc", "/etc/passwd", true, "outside allowed"},
		{"outside allowed /home", "/home/user/file.txt", true, "outside allowed"},
		{"outside allowed /var", "/var/log/syslog", true, "outside allowed"},
	}

	// Ensure DISABLE_PATH_CHECK is not set for these tests
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
			} else {
				if err != nil {
					t.Errorf("validateMediaPath(%s) = %v, want nil", tt.path, err)
				}
			}
		})
	}
}

func TestValidateMediaPath_DisableCheck(t *testing.T) {
	// Save and restore env var
	original := os.Getenv("DISABLE_PATH_CHECK")
	defer os.Setenv("DISABLE_PATH_CHECK", original)

	// Enable path check bypass
	os.Setenv("DISABLE_PATH_CHECK", "true")

	// Should now allow paths outside allowed directories
	// Note: Path traversal attempts still blocked
	err := validateMediaPath("/home/user/file.txt")
	if err != nil {
		t.Errorf("With DISABLE_PATH_CHECK=true, validateMediaPath should allow external paths, got: %v", err)
	}
}

func TestValidateMediaPath_TraversalAlwaysBlocked(t *testing.T) {
	// Save and restore env var
	original := os.Getenv("DISABLE_PATH_CHECK")
	defer os.Setenv("DISABLE_PATH_CHECK", original)

	// Even with DISABLE_PATH_CHECK=true, path traversal should be blocked
	os.Setenv("DISABLE_PATH_CHECK", "true")

	err := validateMediaPath("/app/media/../../../etc/passwd")
	if err == nil {
		t.Error("Path traversal should be blocked even with DISABLE_PATH_CHECK=true")
	}
}
