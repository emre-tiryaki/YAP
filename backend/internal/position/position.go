package position

import (
	"errors"
	"strings"
)

const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var ErrNoRoom = errors.New("position: no key between bounds")

func After(prev string) string {
	if prev == "" {
		return "a0"
	}
	last := prev[len(prev)-1]
	i := strings.IndexByte(digits, last)
	if i < 0 || i == len(digits)-1 {
		return prev + "1"
	}
	return prev[:len(prev)-1] + string(digits[i+1])
}

// Between returns a key that sorts strictly between lo and hi in byte order.
// An empty lo means "before everything", an empty hi means "after everything".
// Keys are treated as base-62 fractions; the result never ends in the zero
// digit, so a later key can always be placed between it and its neighbours.
func Between(lo, hi string) (string, error) {
	if hi != "" && lo >= hi {
		return "", ErrNoRoom
	}
	return midpoint(lo, hi, hi == "")
}

func midpoint(lo, hi string, open bool) (string, error) {
	if !open {
		n := 0
		for n < len(hi) && digitAt(lo, n) == hi[n] {
			n++
		}
		if n == len(hi) {
			// lo equals hi padded with zeros: no key fits in between.
			return "", ErrNoRoom
		}
		if n > 0 {
			rest, err := midpoint(tail(lo, n), hi[n:], false)
			if err != nil {
				return "", err
			}
			return hi[:n] + rest, nil
		}
	}

	a := 0
	if lo != "" {
		a = strings.IndexByte(digits, lo[0])
	}
	b := len(digits)
	if !open {
		b = strings.IndexByte(digits, hi[0])
	}
	if a < 0 || b < 0 {
		return "", ErrNoRoom
	}
	if b-a > 1 {
		return string(digits[(a+b+1)/2]), nil
	}
	rest, err := midpoint(tail(lo, 1), "", true)
	if err != nil {
		return "", err
	}
	return string(digits[a]) + rest, nil
}

func digitAt(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return digits[0]
}

func tail(s string, n int) string {
	if n >= len(s) {
		return ""
	}
	return s[n:]
}
