package tools

import (
	"errors"
	"strings"
)

// ParseByteSize разбирает строки вида "10MB", "4KB", "1GB".
func ParseByteSize(s string) (int, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0, errors.New("empty size string")
	}

	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, errors.New("size must start with a digit")
	}

	n := 0
	for _, ch := range s[:i] {
		n = n*10 + int(ch-'0')
	}

	suffix := strings.ToUpper(s[i:])
	switch suffix {
	case "GB":
		return n << 30, nil
	case "MB":
		return n << 20, nil
	case "KB":
		return n << 10, nil
	case "B", "":
		return n, nil
	default:
		return 0, errors.New("unknown suffix: " + suffix)
	}
}
