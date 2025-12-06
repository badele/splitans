package types

import (
	"fmt"
	"strings"
)

/////////////////////////////////////////////////////////////////////////////
// COLOR
/////////////////////////////////////////////////////////////////////////////

type ColorType int

const (
	ColorDefault  ColorType = iota
	ColorStandard           // 0-15 (codes 30-37, 90-97, etc.)
	ColorIndexed            // 0-255 (ESC[38;5;n)
	ColorRGB                // RGB (ESC[38;2;r;g;b)
)

type ColorValue struct {
	Type    ColorType
	R, G, B uint8
	Index   uint8
}

func (c ColorValue) IsDefault() bool {
	return c.Type == ColorDefault
}

func (c ColorValue) String() string {
	switch c.Type {
	case ColorDefault:
		return "default"
	case ColorStandard:
		return fmt.Sprintf("std:%d", c.Index)
	case ColorIndexed:
		return fmt.Sprintf("idx:%d", c.Index)
	case ColorRGB:
		return fmt.Sprintf("rgb(%d,%d,%d)", c.R, c.G, c.B)
	}
	return "unknown"
}

// VGA Palette with exact VGA hardware color values
var VGAPalette = [16][3]uint8{
	{0x00, 0x00, 0x00}, // 0: Black
	{0xAA, 0x00, 0x00}, // 1: Red
	{0x00, 0xAA, 0x00}, // 2: Green
	{0xAA, 0x55, 0x00}, // 3: Yellow/Brown
	{0x00, 0x00, 0xAA}, // 4: Blue
	{0xAA, 0x00, 0xAA}, // 5: Magenta
	{0x00, 0xAA, 0xAA}, // 6: Cyan
	{0xAA, 0xAA, 0xAA}, // 7: White/Light Gray
	{0x55, 0x55, 0x55}, // 8: Bright Black (Dark Gray)
	{0xFF, 0x55, 0x55}, // 9: Bright Red
	{0x55, 0xFF, 0x55}, // 10: Bright Green
	{0xFF, 0xFF, 0x55}, // 11: Bright Yellow
	{0x55, 0x55, 0xFF}, // 12: Bright Blue
	{0xFF, 0x55, 0xFF}, // 13: Bright Magenta
	{0x55, 0xFF, 0xFF}, // 14: Bright Cyan
	{0xFF, 0xFF, 0xFF}, // 15: Bright White
}

/////////////////////////////////////////////////////////////////////////////
// SGR (Select Graphic Rendition)
/////////////////////////////////////////////////////////////////////////////

type SGR struct {
	FgColor       ColorValue
	BgColor       ColorValue
	Bold          bool
	Dim           bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Hidden        bool
	Strikethrough bool
}

func NewSGR() *SGR {
	return &SGR{
		FgColor: ColorValue{Type: ColorDefault},
		BgColor: ColorValue{Type: ColorDefault},
	}
}

func (s *SGR) Reset() {
	s.FgColor = ColorValue{Type: ColorDefault}
	s.BgColor = ColorValue{Type: ColorDefault}
	s.Bold = false
	s.Dim = false
	s.Italic = false
	s.Underline = false
	s.Blink = false
	s.Reverse = false
	s.Hidden = false
	s.Strikethrough = false
}

func (s *SGR) ApplyParams(params []int) {
	for i := 0; i < len(params); i++ {
		code := params[i]

		switch code {
		case 0:
			s.Reset()

		case 1:
			s.Bold = true
		case 21, 22:
			s.Bold = false

		case 2:
			s.Dim = true
		case 3:
			s.Italic = true
		case 4:
			s.Underline = true
		case 5:
			s.Blink = true
		case 7:
			s.Reverse = true
		case 8:
			s.Hidden = true
		case 9:
			s.Strikethrough = true

		case 30, 31, 32, 33, 34, 35, 36, 37:
			s.FgColor = ColorValue{Type: ColorStandard, Index: uint8(code - 30)}

		case 38: // Foreground extended
			i += s.applyExtendedColor(&s.FgColor, params, i+1)

		case 39:
			s.FgColor = ColorValue{Type: ColorDefault}

		case 40, 41, 42, 43, 44, 45, 46, 47:
			s.BgColor = ColorValue{Type: ColorStandard, Index: uint8(code - 40)}

		case 48: // Background extended
			i += s.applyExtendedColor(&s.BgColor, params, i+1)

		case 49:
			s.BgColor = ColorValue{Type: ColorDefault}

		case 90, 91, 92, 93, 94, 95, 96, 97:
			s.FgColor = ColorValue{Type: ColorStandard, Index: uint8(code - 90 + 8)}

		case 100, 101, 102, 103, 104, 105, 106, 107:
			s.BgColor = ColorValue{Type: ColorStandard, Index: uint8(code - 100 + 8)}
		}
	}
}

func (s *SGR) applyExtendedColor(color *ColorValue, params []int, start int) int {
	if start >= len(params) {
		return 0
	}

	colorType := params[start]

	switch colorType {
	case 5: // Indexed color (256 colors)
		// ESC[38;5;n
		if start+1 < len(params) {
			*color = ColorValue{
				Type:  ColorIndexed,
				Index: uint8(params[start+1]),
			}
			return 2
		}

	case 2: // RGB color
		// ESC[38;2;r;g;b
		if start+3 < len(params) {
			*color = ColorValue{
				Type: ColorRGB,
				R:    uint8(params[start+1]),
				G:    uint8(params[start+2]),
				B:    uint8(params[start+3]),
			}
			return 4
		}
	}

	return 1
}

func (s *SGR) ToANSI(useVGAColors bool) string {
	var codes []string

	// Foreground color
	if !s.FgColor.IsDefault() {
		switch s.FgColor.Type {
		case ColorStandard:
			if useVGAColors {
				// Use exact VGA RGB values
				// In VGA terminals, bold + color 0-7 = bright color 8-15
				colorIndex := s.FgColor.Index
				if s.Bold && colorIndex < 8 {
					colorIndex += 8
				}
				rgb := VGAPalette[colorIndex]
				codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]))
			} else {
				// Use standard ANSI codes
				if s.FgColor.Index < 8 {
					codes = append(codes, fmt.Sprintf("%d", 30+s.FgColor.Index))
				} else {
					codes = append(codes, fmt.Sprintf("%d", 82+s.FgColor.Index))
				}
			}
		case ColorIndexed:
			codes = append(codes, fmt.Sprintf("38;5;%d", s.FgColor.Index))
		case ColorRGB:
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", s.FgColor.R, s.FgColor.G, s.FgColor.B))
		}
	}

	// Background color
	if !s.BgColor.IsDefault() {
		switch s.BgColor.Type {
		case ColorStandard:
			if useVGAColors {
				// Use exact VGA RGB values
				// In VGA terminals, bold + color 0-7 = bright color 8-15
				colorIndex := s.BgColor.Index
				if s.Bold && colorIndex < 8 {
					colorIndex += 8
				}
				rgb := VGAPalette[colorIndex]
				codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]))
			} else {
				// Use standard ANSI codes
				if s.BgColor.Index < 8 {
					codes = append(codes, fmt.Sprintf("%d", 40+s.BgColor.Index))
				} else {
					codes = append(codes, fmt.Sprintf("%d", 92+s.BgColor.Index))
				}
			}
		case ColorIndexed:
			codes = append(codes, fmt.Sprintf("48;5;%d", s.BgColor.Index))
		case ColorRGB:
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", s.BgColor.R, s.BgColor.G, s.BgColor.B))
		}
	}

	// Attributes
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Dim {
		codes = append(codes, "2")
	}
	if s.Italic {
		codes = append(codes, "3")
	}
	if s.Underline {
		codes = append(codes, "4")
	}
	if s.Blink {
		codes = append(codes, "5")
	}
	if s.Reverse {
		codes = append(codes, "7")
	}
	if s.Hidden {
		codes = append(codes, "8")
	}
	if s.Strikethrough {
		codes = append(codes, "9")
	}

	if len(codes) == 0 {
		return "\x1b[0m"
	}

	return fmt.Sprintf("\x1b[%sm", strings.Join(codes, ";"))
}

func (s *SGR) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("fg:%s", s.FgColor.String()))
	parts = append(parts, fmt.Sprintf("bg:%s", s.BgColor.String()))

	parts = append(parts, fmt.Sprintf("bold:%t", s.Bold))
	parts = append(parts, fmt.Sprintf("dim:%t", s.Dim))
	parts = append(parts, fmt.Sprintf("italic:%t", s.Italic))
	parts = append(parts, fmt.Sprintf("underline:%t", s.Underline))
	parts = append(parts, fmt.Sprintf("blink:%t", s.Blink))
	parts = append(parts, fmt.Sprintf("reverse:%t", s.Reverse))
	parts = append(parts, fmt.Sprintf("hidden:%t", s.Hidden))
	parts = append(parts, fmt.Sprintf("strikethrough:%t", s.Strikethrough))

	return strings.Join(parts, ", ")
}

func (s *SGR) Equals(other *SGR) bool {
	if s == nil || other == nil {
		return s == other
	}
	return s.FgColor == other.FgColor &&
		s.BgColor == other.BgColor &&
		s.Bold == other.Bold &&
		s.Dim == other.Dim &&
		s.Italic == other.Italic &&
		s.Underline == other.Underline &&
		s.Blink == other.Blink &&
		s.Reverse == other.Reverse &&
		s.Hidden == other.Hidden &&
		s.Strikethrough == other.Strikethrough
}

func (s *SGR) Copy() *SGR {
	return &SGR{
		FgColor:       s.FgColor,
		BgColor:       s.BgColor,
		Bold:          s.Bold,
		Dim:           s.Dim,
		Italic:        s.Italic,
		Underline:     s.Underline,
		Blink:         s.Blink,
		Reverse:       s.Reverse,
		Hidden:        s.Hidden,
		Strikethrough: s.Strikethrough,
	}
}

/////////////////////////////////////////////////////////////////////////////
// LINE WITH SEQUENCES
/////////////////////////////////////////////////////////////////////////////

// SGRChange represents a change in SGR style at a specific position in a line
type SGRChange struct {
	Position int  // Position of the character in the line (0-indexed)
	SGR      *SGR // The SGR style to apply from this position
}

// LineWithSequences contains a line of text and all SGR changes within that line
type LineWithSequences struct {
	Text      string
	Sequences []SGRChange
}
