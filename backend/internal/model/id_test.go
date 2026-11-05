package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewID(t *testing.T) {
	id := NewID()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("NewID() = %q is not a valid UUID: %v", id, err)
	}
	if parsed.Version() != 7 {
		t.Errorf("version = %d, want 7", parsed.Version())
	}
	if len(id) != 36 {
		t.Errorf("len = %d, want 36", len(id))
	}
}

func TestNewIDUniqueAndSortable(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	prev := ""
	for i := 0; i < n; i++ {
		id := NewID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = struct{}{}
		if id <= prev {
			t.Fatalf("id %q does not sort after %q", id, prev)
		}
		prev = id
	}
}