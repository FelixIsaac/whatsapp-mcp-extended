package api

import (
	"testing"

	"whatsapp-bridge/internal/database"
)

// TestCharacterCount verifies character count calculation handles Unicode correctly.
func TestCharacterCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "empty string",
			content:  "",
			expected: 0,
		},
		{
			name:     "ascii text",
			content:  "Hello World",
			expected: 11,
		},
		{
			name:     "unicode emojis",
			content:  "Hello 👋 World",
			expected: 13,
		},
		{
			name:     "special characters",
			content:  "Test!@#$%^&*()",
			expected: 14,
		},
		{
			name:     "newlines and tabs",
			content:  "Line1\nLine2\tTabbed",
			expected: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := database.GetCharacterCount(tt.content)
			if result != tt.expected {
				t.Errorf("GetCharacterCount(%q) = %d, want %d", tt.content, result, tt.expected)
			}
		})
	}
}

// TestWordCount verifies word count calculation using whitespace splitting.
func TestWordCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "empty string",
			content:  "",
			expected: 0,
		},
		{
			name:     "single word",
			content:  "Hello",
			expected: 1,
		},
		{
			name:     "multiple words",
			content:  "Hello World from Go",
			expected: 4,
		},
		{
			name:     "extra whitespace",
			content:  "Hello  	  World",
			expected: 2,
		},
		{
			name:     "words with punctuation",
			content:  "Hello, world! How are you?",
			expected: 5,
		},
		{
			name:     "newlines and tabs",
			content:  "Hello\nWorld\t\tTest",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := database.GetWordCount(tt.content)
			if result != tt.expected {
				t.Errorf("GetWordCount(%q) = %d, want %d", tt.content, result, tt.expected)
			}
		})
	}
}

// TestExtractURLs verifies URL extraction from content.
func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty string",
			content:  "",
			expected: []string{},
		},
		{
			name:     "no urls",
			content:  "Just plain text",
			expected: []string{},
		},
		{
			name:     "single https url",
			content:  "Check this https://example.com",
			expected: []string{"https://example.com"},
		},
		{
			name:     "multiple urls",
			content:  "Visit http://test.com and https://another.com",
			expected: []string{"http://test.com", "https://another.com"},
		},
		{
			name:     "www urls",
			content:  "Go to www.google.com for search",
			expected: []string{"www.google.com"},
		},
		{
			name:     "url with path",
			content:  "Read https://example.com/path/to/page",
			expected: []string{"https://example.com/path/to/page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := database.ExtractURLs(tt.content)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractURLs(%q) returned %d URLs, want %d", tt.content, len(result), len(tt.expected))
				return
			}
			for i, url := range result {
				if url != tt.expected[i] {
					t.Errorf("ExtractURLs(%q)[%d] = %q, want %q", tt.content, i, url, tt.expected[i])
				}
			}
		})
	}
}

// TestExtractMentions verifies @mention extraction from content.
func TestExtractMentions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty string",
			content:  "",
			expected: []string{},
		},
		{
			name:     "no mentions",
			content:  "Just plain text",
			expected: []string{},
		},
		{
			name:     "single mention",
			content:  "Hey @john check this",
			expected: []string{"@john"},
		},
		{
			name:     "multiple mentions",
			content:  "@alice and @bob should see this @charlie",
			expected: []string{"@alice", "@bob", "@charlie"},
		},
		{
			name:     "jid mention",
			content:  "Ping @1234567890@s.whatsapp.net",
			expected: []string{"@1234567890@s.whatsapp.net"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := database.ExtractMentions(tt.content)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractMentions(%q) returned %d mentions, want %d", tt.content, len(result), len(tt.expected))
				return
			}
			for i, mention := range result {
				if mention != tt.expected[i] {
					t.Errorf("ExtractMentions(%q)[%d] = %q, want %q", tt.content, i, mention, tt.expected[i])
				}
			}
		})
	}
}

// TestMessageFormatting verifies that messages include new metadata fields.
// This is a placeholder test for integration with handler logic.
func TestMessageFormatting(t *testing.T) {
	t.Run("message with metadata", func(t *testing.T) {
		content := "Hello @john, check out https://example.com"

		charCount := database.GetCharacterCount(content)
		wordCount := database.GetWordCount(content)
		urls := database.ExtractURLs(content)
		mentions := database.ExtractMentions(content)

		if charCount != 43 {
			t.Errorf("expected 43 chars, got %d", charCount)
		}
		if wordCount != 6 {
			t.Errorf("expected 6 words, got %d", wordCount)
		}
		if len(urls) != 1 || urls[0] != "https://example.com" {
			t.Errorf("expected 1 URL, got %v", urls)
		}
		if len(mentions) != 1 || mentions[0] != "@john" {
			t.Errorf("expected 1 mention, got %v", mentions)
		}
	})
}
