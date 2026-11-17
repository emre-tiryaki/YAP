package model

import (
	"time"

	"github.com/google/uuid"
)

func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// NewIDAt returns a UUIDv7 whose time part is t instead of now, so rows
// created for past events still sort by when they happened.
func NewIDAt(t time.Time) string {
	u := uuid.Must(uuid.NewV7())
	ms := max(t.UnixMilli(), 0)
	for i := 5; i >= 0; i-- {
		u[i] = byte(ms)
		ms >>= 8
	}
	return u.String()
}
