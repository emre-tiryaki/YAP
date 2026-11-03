package position

import "testing"

func TestAfter(t *testing.T) {
	tests := []struct {
		name string
		prev string
		want string
	}{
		{"empty", "", "a0"},
		{"digit", "a0", "a1"},
		{"digit to upper", "a9", "aA"},
		{"upper to lower", "aZ", "aa"},
		{"last character", "az", "az1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := After(tt.prev)
			if got != tt.want {
				t.Errorf("After(%q) = %q, want %q", tt.prev, got, tt.want)
			}
			if got <= tt.prev {
				t.Errorf("After(%q) = %q does not sort after prev", tt.prev, got)
			}
		})
	}
}