package position

import "strings"

const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func After(prev string) string {
	if prev == "" {
		return "a0"
	}
	last := prev[len(prev)-1]
	i := strings.IndexByte(digits, last)
	if i < 0 || i == len(digits)-1 {
		return  prev + "1"
	}
	return prev[:len(prev)-1] + string(digits[i+1])
}