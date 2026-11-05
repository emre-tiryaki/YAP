package position

import "testing"

func TestAfterChainIsStrictlyIncreasing(t *testing.T) {
	prev := ""
	for i := 0; i < 500; i++ {
		next := After(prev)
		if next <= prev {
			t.Fatalf("step %d: After(%q) = %q does not sort after prev", i, prev, next)
		}
		prev = next
	}
}

func TestAfterUnknownLastCharacter(t *testing.T) {
	tests := []struct {
		name string
		prev string
		want string
	}{
		{"punctuation", "a-", "a-1"},
		{"space", "a ", "a 1"},
		{"carry chain", "azz", "azz1"},
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