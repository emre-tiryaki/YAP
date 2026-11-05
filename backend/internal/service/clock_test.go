package service

import (
	"strings"
	"testing"
	"time"
)

func TestNowUTC(t *testing.T) {
	before := time.Now().UTC().Truncate(time.Second)
	got := nowUTC()
	after := time.Now().UTC().Add(time.Second)

	if !strings.HasSuffix(got, "Z") {
		t.Errorf("nowUTC() = %q, want UTC suffix Z", got)
	}
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("nowUTC() = %q is not RFC 3339: %v", got, err)
	}
	if parsed.Before(before) || parsed.After(after) {
		t.Errorf("nowUTC() = %v, want between %v and %v", parsed, before, after)
	}
}