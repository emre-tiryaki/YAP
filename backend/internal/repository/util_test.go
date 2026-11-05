package repository

import "testing"

func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want any
	}{
		{"empty", "", nil},
		{"value", "text", "text"},
		{"whitespace is kept", " ", " "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nullIfEmpty(tt.in); got != tt.want {
				t.Errorf("nullIfEmpty(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}