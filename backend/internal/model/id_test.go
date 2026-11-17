package model

import (
	"testing"
	"time"

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
func TestNewIDAtCarriesTheTime(t *testing.T) {
	at := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	id := NewIDAt(at)
	u, err := uuid.Parse(id)
	if err != nil || u.Version() != 7 {
		t.Fatalf("NewIDAt() = %q (err %v), want a UUIDv7", id, err)
	}
	sec, nsec := u.Time().UnixTime()
	if got := time.Unix(sec, nsec).UTC(); !got.Equal(at) {
		t.Errorf("time = %v, want %v", got, at)
	}
	if NewIDAt(at.Add(-time.Hour)) >= id || id >= NewID() {
		t.Error("ids do not sort by their time")
	}
}
