package service

import "time"

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// parseStamp reads a timestamp produced by nowUTC.
func parseStamp(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
