package service

import (
	"strconv"
	"testing"
)

func TestSuggestIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		taken map[string]bool
		want  string
	}{
		{"first three letters", "Kitap Kulübü", nil, "KIT"},
		{"skips non letters", "  1-go lang", nil, "GOL"},
		{"two letters are enough", "go", nil, "GO"},
		{"one letter falls back", "x", nil, "PRJ"},
		{"no letters falls back", "12345", nil, "PRJ"},
		{"non ASCII letters are skipped", "Öğrenme", nil, "REN"},
		{"taken gets a number", "Kitchen", map[string]bool{"KIT": true}, "KIT2"},
		{"next free number", "Kitchen", map[string]bool{"KIT": true, "KIT2": true}, "KIT3"},
		{"two letters with a number", "go", map[string]bool{"GO": true}, "GO2"},
		{"stays within five characters", "Kitchen", takenUpTo("KIT", 99), "KI100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := suggestIdentifier(tt.input, tt.taken); got != tt.want {
				t.Errorf("suggestIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// takenUpTo marks base and base2 .. baseN as taken, the way suggestIdentifier
// shortens the base to keep a long number within five characters.
func takenUpTo(base string, n int) map[string]bool {
	taken := map[string]bool{base: true}
	for i := 2; i <= n; i++ {
		suffix := strconv.Itoa(i)
		taken[base[:min(len(base), 5-len(suffix))]+suffix] = true
	}
	return taken
}

func TestNormalizeIdentifier(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"abc", "ABC", false},
		{" web2 ", "WEB2", false},
		{"AB", "AB", false},
		{"A", "", true},
		{"ABCDEF", "", true},
		{"1AB", "", true},
		{"A-B", "", true},
		{"", "", true},
		{"ÇAY", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := normalizeIdentifier(tt.raw)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("normalizeIdentifier(%q) = %q, %v; want %q (error %v)", tt.raw, got, err, tt.want, tt.wantErr)
			}
		})
	}
}
