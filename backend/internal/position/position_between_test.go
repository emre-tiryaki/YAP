package position

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestBetween(t *testing.T) {
	tests := []struct {
		name   string
		lo, hi string
	}{
		{"both open", "", ""},
		{"before first", "", "a0"},
		{"after last", "a0", ""},
		{"adjacent digits", "a0", "a1"},
		{"shared prefix", "a0V", "a0W"},
		{"prefix of hi", "a", "a1"},
		{"lo is prefix of hi", "a0", "a0V"},
		{"hi starts with zero", "", "0V"},
		{"wide gap", "B", "y"},
		{"last digit", "az", ""},
		{"before carry key", "az", "az1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Between(tt.lo, tt.hi)
			if err != nil {
				t.Fatalf("Between(%q, %q) error = %v", tt.lo, tt.hi, err)
			}
			if got <= tt.lo || (tt.hi != "" && got >= tt.hi) {
				t.Errorf("Between(%q, %q) = %q, not strictly between", tt.lo, tt.hi, got)
			}
			if got[len(got)-1] == digits[0] {
				t.Errorf("Between(%q, %q) = %q ends in the zero digit", tt.lo, tt.hi, got)
			}
		})
	}
}

func TestBetweenRejectsInvalidBounds(t *testing.T) {
	tests := []struct {
		name   string
		lo, hi string
	}{
		{"equal", "a0", "a0"},
		{"reversed", "a1", "a0"},
		{"zero padded", "a", "a0"},
		{"unknown character", "a-", "a."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := Between(tt.lo, tt.hi); !errors.Is(err, ErrNoRoom) {
				t.Errorf("Between(%q, %q) = %q, %v, want ErrNoRoom", tt.lo, tt.hi, got, err)
			}
		})
	}
}

// Repeated random inserts mixed with After must keep a strictly ordered list.
func TestBetweenRandomInsertsStayOrdered(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	keys := []string{After("")}
	for i := 0; i < 2000; i++ {
		var key string
		var err error
		switch idx := rng.IntN(len(keys) + 2); {
		case idx == len(keys)+1:
			key = After(keys[len(keys)-1])
		case idx == 0:
			key, err = Between("", keys[0])
		case idx == len(keys):
			key, err = Between(keys[len(keys)-1], "")
		default:
			key, err = Between(keys[idx-1], keys[idx])
		}
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		keys = append(keys, key)
		slices.Sort(keys)
		if j := slices.Index(keys, key); j > 0 && keys[j-1] == key || j < len(keys)-1 && keys[j+1] == key {
			t.Fatalf("step %d: duplicate key %q", i, key)
		}
	}
}
