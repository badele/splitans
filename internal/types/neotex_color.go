package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRGBHex parses a 6-character hex string (RRGGBB) and returns R, G, B values.
func ParseRGBHex(hexStr string) (r, g, b uint8, err error) {
	if len(hexStr) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid RGB hex string length: %d", len(hexStr))
	}

	var rgb uint64
	rgb, err = strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return 0, 0, 0, err
	}

	r = uint8((rgb >> 16) & 0xFF)
	g = uint8((rgb >> 8) & 0xFF)
	b = uint8(rgb & 0xFF)

	return r, g, b, nil
}

// ParseNeotexColor parses a neotex color code for a known target.
// The code may include an optional F/B prefix (e.g. Fk, B00FF00).
func ParseNeotexColor(code string, isForeground bool) (ColorValue, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return ColorValue{}, fmt.Errorf("empty neotex color code")
	}

	if len(code) >= 2 && (code[0] == 'F' || code[0] == 'B') {
		prefixIsForeground := code[0] == 'F'
		if prefixIsForeground != isForeground {
			target := "foreground"
			if !isForeground {
				target = "background"
			}
			return ColorValue{}, fmt.Errorf("unexpected %c prefix for %s color", code[0], target)
		}
		code = code[1:]
	} else if strings.HasPrefix(code, "F") || strings.HasPrefix(code, "B") {
		return ColorValue{}, fmt.Errorf("incomplete neotex color code: %q", code)
	}

	return parseNeotexColorValue(code, isForeground)
}

// ParseNeotexColorCode parses a neotex color code with an explicit F/B prefix.
// Returns ok=false when the code does not look like a color code.
func ParseNeotexColorCode(code string) (isForeground bool, color ColorValue, ok bool, err error) {
	code = strings.TrimSpace(code)
	if len(code) < 2 {
		return false, ColorValue{}, false, nil
	}
	if code[0] != 'F' && code[0] != 'B' {
		return false, ColorValue{}, false, nil
	}

	isForeground = code[0] == 'F'
	color, err = parseNeotexColorValue(code[1:], isForeground)
	return isForeground, color, true, err
}

func parseNeotexColorValue(code string, isForeground bool) (ColorValue, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return ColorValue{}, fmt.Errorf("empty neotex color code")
	}

	if strings.EqualFold(code, "D") {
		if isForeground {
			return ColorValue{Type: ColorStandard, Index: 7}, nil
		}
		return ColorValue{Type: ColorStandard, Index: 0}, nil
	}

	if len(code) == 1 {
		if idx, ok := neotexStandardColorIndex(code[0]); ok {
			return ColorValue{Type: ColorStandard, Index: idx}, nil
		}
	}

	if len(code) == 6 {
		if r, g, b, err := ParseRGBHex(code); err == nil {
			return ColorValue{Type: ColorRGB, R: r, G: g, B: b}, nil
		}
	}

	if idx, err := strconv.Atoi(code); err == nil {
		if idx < 0 || idx > 255 {
			return ColorValue{}, fmt.Errorf("indexed color out of range: %d", idx)
		}
		return ColorValue{Type: ColorIndexed, Index: uint8(idx)}, nil
	}

	return ColorValue{}, fmt.Errorf("invalid neotex color code: %q", code)
}

func neotexStandardColorIndex(code byte) (uint8, bool) {
	bright := false
	if code >= 'A' && code <= 'Z' {
		bright = true
		code = code - 'A' + 'a'
	} else if code < 'a' || code > 'z' {
		return 0, false
	}

	var index uint8
	switch code {
	case 'k':
		index = 0
	case 'r':
		index = 1
	case 'g':
		index = 2
	case 'y':
		index = 3
	case 'b':
		index = 4
	case 'm':
		index = 5
	case 'c':
		index = 6
	case 'w':
		index = 7
	default:
		return 0, false
	}

	if bright {
		index += 8
	}

	return index, true
}
