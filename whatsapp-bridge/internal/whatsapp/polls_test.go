package whatsapp

import (
	"crypto/sha256"
	"reflect"
	"testing"
)

func hashOf(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func TestMatchSelectedOptions(t *testing.T) {
	tests := []struct {
		name           string
		options        []string
		selectedHashes [][]byte
		want           []string
	}{
		{
			name:           "single selection matches",
			options:        []string{"Expected", "Not expected", "Need to discuss"},
			selectedHashes: [][]byte{hashOf("Not expected")},
			want:           []string{"Not expected"},
		},
		{
			name:    "multi-select preserves poll's original option order, not vote order",
			options: []string{"A", "B", "C"},
			// Hashes given out of order (C then A) — output must still be A, C.
			selectedHashes: [][]byte{hashOf("C"), hashOf("A")},
			want:           []string{"A", "C"},
		},
		{
			name:           "no hashes means retracted vote",
			options:        []string{"Yes", "No"},
			selectedHashes: nil,
			want:           nil,
		},
		{
			name:           "unknown hash is ignored, not errored",
			options:        []string{"Yes", "No"},
			selectedHashes: [][]byte{hashOf("Maybe")},
			want:           nil,
		},
		{
			name:           "one known, one unknown hash — only the known one is returned",
			options:        []string{"Yes", "No"},
			selectedHashes: [][]byte{hashOf("Yes"), hashOf("Maybe")},
			want:           []string{"Yes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchSelectedOptions(tt.options, tt.selectedHashes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchSelectedOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
