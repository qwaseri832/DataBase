package tools

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func ParseByteSize(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("size must start with a digit: %q", s)
	}

	n, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse size %q: %w", s, err)
	}

	var shift uint
	switch strings.ToUpper(strings.TrimSpace(s[i:])) {
	case "GB":
		shift = 30
	case "MB":
		shift = 20
	case "KB":
		shift = 10
	case "B", "":
		shift = 0
	default:
		return 0, fmt.Errorf("unknown size suffix: %q", s[i:])
	}

	if shift > 0 && n > math.MaxInt64>>shift {
		return 0, fmt.Errorf("size %q overflows int64", s)
	}
	n <<= shift

	if n > math.MaxInt {
		return 0, fmt.Errorf("size %q overflows int on this platform", s)
	}
	return int(n), nil
}
