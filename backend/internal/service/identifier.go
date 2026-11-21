package service

import (
	"fmt"
	"regexp"
	"strings"
)

// identifierPattern is a project identifier: a letter, then 1 to 4 letters or
// digits, all uppercase (ABC, WEB2).
var identifierPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,4}$`)

// keyPattern matches a task key typed into the search, e.g. "abc-12".
var keyPattern = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]{1,4})-(\d+)$`)

const fallbackIdentifier = "PRJ"

// suggestIdentifier derives an identifier from a project name: its first three
// ASCII letters, uppercased (PRJ when it has fewer than two). When that is in
// taken, a number is appended (ABC2, ABC3, ...). It matches the backfill of
// migration 011.
func suggestIdentifier(name string, taken map[string]bool) string {
	var letters []byte
	for i := 0; i < len(name) && len(letters) < 3; i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c >= 'A' && c <= 'Z' {
			letters = append(letters, c)
		}
	}
	base := string(letters)
	if len(base) < 2 {
		base = fallbackIdentifier
	}
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		suffix := fmt.Sprint(n)
		candidate := base[:min(len(base), 5-len(suffix))] + suffix
		if !taken[candidate] {
			return candidate
		}
	}
}

// normalizeIdentifier trims and uppercases a user-supplied identifier and
// checks it.
func normalizeIdentifier(raw string) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(raw))
	if !identifierPattern.MatchString(id) {
		return "", fmt.Errorf("%w: identifier must be 2 to 5 letters or digits and start with a letter", ErrInvalidInput)
	}
	return id, nil
}
