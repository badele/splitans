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
		// FgColor: ColorValue{Type: ColorDefault},
		// BgColor: ColorValue{Type: ColorDefault},
		FgColor: ColorValue{Type: ColorStandard, Index: 7}, // Default to light gray for better visibility
		BgColor: ColorValue{Type: ColorStandard, Index: 0}, // Default to black
	}
}

func (s *SGR) Reset() {
	// s.FgColor = ColorValue{Type: ColorDefault}
	// s.BgColor = ColorValue{Type: ColorDefault}
	s.FgColor = ColorValue{Type: ColorStandard, Index: 7} // Default to light gray
	s.BgColor = ColorValue{Type: ColorStandard, Index: 0} // Default to black
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

func (s *SGR) ToANSI(useVGAColors bool, legacyMode bool) string {
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
					// Couleur normale (0-7)
					// if legacyMode {
					// 	codes = append(codes, "0") // Reset pour éviter bold précédent
					// }
					codes = append(codes, fmt.Sprintf("%d", 30+s.FgColor.Index))
				} else {
					// Couleur bright (8-15)
					if legacyMode {
						// Mode legacy : 91 → 0;31;1 (reset + color + bold)
						// codes = append(codes, "0")
						codes = append(codes, fmt.Sprintf("%d", 30+(s.FgColor.Index-8)))
						codes = append(codes, "1")
					} else {
						// Mode moderne : utilise 90-97
						codes = append(codes, fmt.Sprintf("%d", 82+s.FgColor.Index))
					}
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
					// Couleur background normale (0-7)
					// if legacyMode {
					// 	codes = append(codes, "0") // Reset pour éviter bold précédent
					// }
					codes = append(codes, fmt.Sprintf("%d", 40+s.BgColor.Index))
				} else {
					// Couleur background bright (8-15)
					if legacyMode {
						// Mode legacy : 101 → 0;41;1 (reset + color + bold)
						// codes = append(codes, "0")
						codes = append(codes, fmt.Sprintf("%d", 40+(s.BgColor.Index-8)))
						codes = append(codes, "1")
					} else {
						// Mode moderne : utilise 100-107
						codes = append(codes, fmt.Sprintf("%d", 92+s.BgColor.Index))
					}
				}
			}
		case ColorIndexed:
			codes = append(codes, fmt.Sprintf("48;5;%d", s.BgColor.Index))
		case ColorRGB:
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", s.BgColor.R, s.BgColor.G, s.BgColor.B))
		}
	}

	// Attributes
	// Bold - ne pas ajouter si déjà ajouté par legacyMode pour bright colors
	alreadyBold := legacyMode && ((!s.FgColor.IsDefault() && s.FgColor.Type == ColorStandard && s.FgColor.Index >= 8) ||
		(!s.BgColor.IsDefault() && s.BgColor.Type == ColorStandard && s.BgColor.Index >= 8))

	if s.Bold && !alreadyBold {
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

	parts = append(parts, fmt.Sprintf("default:%t", s.FgColor.IsDefault()))
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
// DIFFERENTIAL SGR ENCODING
/////////////////////////////////////////////////////////////////////////////

// countActiveAttributes returns the number of non-default attributes
func (s *SGR) countActiveAttributes() int {
	count := 0
	if s.Bold {
		count++
	}
	if s.Dim {
		count++
	}
	if s.Italic {
		count++
	}
	if s.Underline {
		count++
	}
	if s.Blink {
		count++
	}
	if s.Reverse {
		count++
	}
	if s.Hidden {
		count++
	}
	if s.Strikethrough {
		count++
	}
	if !s.FgColor.IsDefault() {
		count++
	}
	if !s.BgColor.IsDefault() {
		count++
	}
	return count
}

// hasAttributeTurnedOff checks if any attribute was turned OFF from previous to current
// In legacy mode, bright colors (8-15) use bold implicitly, so going from bright to normal
// is like turning off bold and requires a reset.
func (s *SGR) hasAttributeTurnedOff(previous *SGR) bool {
	if previous.Bold && !s.Bold {
		return true
	}
	if previous.Dim && !s.Dim {
		return true
	}
	if previous.Italic && !s.Italic {
		return true
	}
	if previous.Underline && !s.Underline {
		return true
	}
	if previous.Blink && !s.Blink {
		return true
	}
	if previous.Reverse && !s.Reverse {
		return true
	}
	if previous.Hidden && !s.Hidden {
		return true
	}
	if previous.Strikethrough && !s.Strikethrough {
		return true
	}
	// FG color changed to default
	if !previous.FgColor.IsDefault() && s.FgColor.IsDefault() {
		return true
	}
	// BG color changed to default
	if !previous.BgColor.IsDefault() && s.BgColor.IsDefault() {
		return true
	}
	// FG bright color (8-15) changed to normal color (0-7)
	// In legacy mode, bright colors use bold implicitly, so this is like turning off bold
	if previous.FgColor.Type == ColorStandard && s.FgColor.Type == ColorStandard {
		if previous.FgColor.Index >= 8 && s.FgColor.Index < 8 {
			return true
		}
	}
	// Also handle: previous was bright, current is different type (indexed, RGB, default)
	if previous.FgColor.Type == ColorStandard && previous.FgColor.Index >= 8 {
		if s.FgColor.Type != ColorStandard || s.FgColor.Index < 8 {
			return true
		}
	}
	// BG bright color (8-15) changed to normal color (0-7)
	if previous.BgColor.Type == ColorStandard && s.BgColor.Type == ColorStandard {
		if previous.BgColor.Index >= 8 && s.BgColor.Index < 8 {
			return true
		}
	}
	if previous.BgColor.Type == ColorStandard && previous.BgColor.Index >= 8 {
		if s.BgColor.Type != ColorStandard || s.BgColor.Index < 8 {
			return true
		}
	}
	return false
}

// fgColorCodesLegacy returns SGR codes for foreground color in legacy mode
// In legacy mode, bright colors (8-15) use bold (1) + base color (30-37)
func (s *SGR) fgColorCodesLegacy(legacyMode bool) []int {
	if s.FgColor.IsDefault() {
		return []int{39}
	}

	switch s.FgColor.Type {
	case ColorStandard:
		if s.FgColor.Index < 8 {
			return []int{30 + int(s.FgColor.Index)}
		}
		// Bright colors (8-15)
		if legacyMode {
			// Legacy mode: use bold + base color (e.g., bright red = 1;31)
			return []int{1, 30 + int(s.FgColor.Index) - 8}
		}
		return []int{90 + int(s.FgColor.Index) - 8}
	case ColorIndexed:
		return []int{38, 5, int(s.FgColor.Index)}
	case ColorRGB:
		return []int{38, 2, int(s.FgColor.R), int(s.FgColor.G), int(s.FgColor.B)}
	}
	return nil
}

// bgColorCodesLegacy returns SGR codes for background color in legacy mode
// In legacy mode, bright colors (8-15) use bold (1) + base color (40-47)
func (s *SGR) bgColorCodesLegacy(legacyMode bool) []int {
	if s.BgColor.IsDefault() {
		return []int{49}
	}

	switch s.BgColor.Type {
	case ColorStandard:
		if s.BgColor.Index < 8 {
			return []int{40 + int(s.BgColor.Index)}
		}
		// Bright background colors (8-15)
		if legacyMode {
			// Legacy mode: use bold + base color (e.g., bright red bg = 1;41)
			return []int{1, 40 + int(s.BgColor.Index) - 8}
		}
		return []int{100 + int(s.BgColor.Index) - 8}
	case ColorIndexed:
		return []int{48, 5, int(s.BgColor.Index)}
	case ColorRGB:
		return []int{48, 2, int(s.BgColor.R), int(s.BgColor.G), int(s.BgColor.B)}
	}
	return nil
}

// fgColorCodes returns SGR codes for foreground color (modern mode)
func (s *SGR) fgColorCodes() []int {
	return s.fgColorCodesLegacy(false)
}

// bgColorCodes returns SGR codes for background color (modern mode)
func (s *SGR) bgColorCodes() []int {
	return s.bgColorCodesLegacy(false)
}

// toFullCodesLegacy returns all active attribute codes (without reset prefix)
// In legacy mode, bright colors use bold + base color
func (s *SGR) toFullCodesLegacy(legacyMode bool) []int {
	var codes []int

	// In legacy mode, don't add bold separately if we have bright FG color
	// because it will be added by fgColorCodesLegacy
	addBold := s.Bold
	if legacyMode && !s.FgColor.IsDefault() && s.FgColor.Type == ColorStandard && s.FgColor.Index >= 8 {
		addBold = false // Bold will be added by fgColorCodesLegacy
	}

	if addBold {
		codes = append(codes, 1)
	}
	if s.Dim {
		codes = append(codes, 2)
	}
	if s.Italic {
		codes = append(codes, 3)
	}
	if s.Underline {
		codes = append(codes, 4)
	}
	if s.Blink {
		codes = append(codes, 5)
	}
	if s.Reverse {
		codes = append(codes, 7)
	}
	if s.Hidden {
		codes = append(codes, 8)
	}
	if s.Strikethrough {
		codes = append(codes, 9)
	}

	if !s.FgColor.IsDefault() {
		codes = append(codes, s.fgColorCodesLegacy(legacyMode)...)
	}
	if !s.BgColor.IsDefault() {
		codes = append(codes, s.bgColorCodesLegacy(legacyMode)...)
	}

	return codes
}

// toFullCodes returns all active attribute codes (modern mode)
func (s *SGR) toFullCodes() []int {
	return s.toFullCodesLegacy(false)
}

// Diff returns the minimal set of SGR codes to transition from previous to current state.
// If previous is nil, returns full state codes.
// If legacyMode is true, uses [0m + full state when any attribute needs to be turned OFF.
// If legacyMode is false, uses individual OFF codes (22, 23, 24, etc.).
func (s *SGR) Diff(previous *SGR, legacyMode bool) []int {
	// Handle nil previous - return full state
	if previous == nil {
		return s.toFullCodesLegacy(legacyMode)
	}

	// If equal, no changes needed
	if s.Equals(previous) {
		return nil
	}

	// Check if current is default state (full reset)
	if s.Equals(NewSGR()) {
		return []int{0}
	}

	// In legacy mode, if any attribute needs to be turned OFF, use reset + full state
	if legacyMode && s.hasAttributeTurnedOff(previous) {
		codes := []int{0}
		codes = append(codes, s.toFullCodesLegacy(legacyMode)...)
		return codes
	}

	// Calculate differential codes
	var codes []int

	// Boolean attributes with their ON/OFF codes
	// In legacy mode, don't add bold separately if we have bright FG color
	addBold := s.Bold != previous.Bold
	if legacyMode && s.FgColor != previous.FgColor && !s.FgColor.IsDefault() && s.FgColor.Type == ColorStandard && s.FgColor.Index >= 8 {
		addBold = false // Bold will be added by fgColorCodesLegacy
	}

	if addBold {
		if s.Bold {
			codes = append(codes, 1)
		} else {
			codes = append(codes, 22) // Bold off
		}
	}

	if s.Dim != previous.Dim {
		if s.Dim {
			codes = append(codes, 2)
		} else {
			codes = append(codes, 22) // Dim off (same as bold off)
		}
	}

	if s.Italic != previous.Italic {
		if s.Italic {
			codes = append(codes, 3)
		} else {
			codes = append(codes, 23)
		}
	}

	if s.Underline != previous.Underline {
		if s.Underline {
			codes = append(codes, 4)
		} else {
			codes = append(codes, 24)
		}
	}

	if s.Blink != previous.Blink {
		if s.Blink {
			codes = append(codes, 5)
		} else {
			codes = append(codes, 25)
		}
	}

	if s.Reverse != previous.Reverse {
		if s.Reverse {
			codes = append(codes, 7)
		} else {
			codes = append(codes, 27)
		}
	}

	if s.Hidden != previous.Hidden {
		if s.Hidden {
			codes = append(codes, 8)
		} else {
			codes = append(codes, 28)
		}
	}

	if s.Strikethrough != previous.Strikethrough {
		if s.Strikethrough {
			codes = append(codes, 9)
		} else {
			codes = append(codes, 29)
		}
	}

	// Foreground color
	if s.FgColor != previous.FgColor {
		codes = append(codes, s.fgColorCodesLegacy(legacyMode)...)
	}

	// Background color
	if s.BgColor != previous.BgColor {
		codes = append(codes, s.bgColorCodesLegacy(legacyMode)...)
	}

	return codes
}

// DiffToANSI generates the minimal ANSI escape sequence to transition from previous to current state.
// If legacyMode is true, uses [0m + full state when attributes need to be turned OFF (ANSI 1990 compatible).
// If legacyMode is false, uses individual OFF codes (modern terminals).
func (s *SGR) DiffToANSI(previous *SGR, useVGAColors bool, legacyMode bool) string {
	codes := s.Diff(previous, legacyMode)

	if len(codes) == 0 {
		return "" // No change needed
	}

	// Special handling for VGA colors mode
	if useVGAColors {
		return s.diffToVGAColors(previous, legacyMode)
	}

	// Build the escape sequence
	var parts []string
	for _, code := range codes {
		parts = append(parts, fmt.Sprintf("%d", code))
	}

	return fmt.Sprintf("\x1b[%sm", strings.Join(parts, ";"))
}

// diffToVGAColors handles differential encoding with VGA RGB color conversion
func (s *SGR) diffToVGAColors(previous *SGR, legacyMode bool) string {
	// For VGA mode, we need to convert standard colors to RGB
	// This is a simplified version - a full implementation would need more care

	// If any attribute turned off in legacy mode, reset + full state
	if legacyMode && previous != nil && s.hasAttributeTurnedOff(previous) {
		var codes []string
		codes = append(codes, "0")

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

		// FG color with VGA palette
		if !s.FgColor.IsDefault() && s.FgColor.Type == ColorStandard {
			colorIndex := s.FgColor.Index
			if s.Bold && colorIndex < 8 {
				colorIndex += 8
			}
			rgb := VGAPalette[colorIndex]
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]))
		} else if !s.FgColor.IsDefault() {
			for _, c := range s.fgColorCodes() {
				codes = append(codes, fmt.Sprintf("%d", c))
			}
		}

		// BG color with VGA palette
		if !s.BgColor.IsDefault() && s.BgColor.Type == ColorStandard {
			colorIndex := s.BgColor.Index
			rgb := VGAPalette[colorIndex]
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]))
		} else if !s.BgColor.IsDefault() {
			for _, c := range s.bgColorCodes() {
				codes = append(codes, fmt.Sprintf("%d", c))
			}
		}

		if len(codes) == 0 {
			return ""
		}
		return fmt.Sprintf("\x1b[%sm", strings.Join(codes, ";"))
	}

	// Build differential codes
	var codes []string

	// Attributes
	if previous == nil || s.Bold != previous.Bold {
		if s.Bold {
			codes = append(codes, "1")
		} else if !legacyMode {
			codes = append(codes, "22")
		}
	}
	if previous == nil || s.Dim != previous.Dim {
		if s.Dim {
			codes = append(codes, "2")
		} else if !legacyMode {
			codes = append(codes, "22")
		}
	}
	if previous == nil || s.Italic != previous.Italic {
		if s.Italic {
			codes = append(codes, "3")
		} else if !legacyMode {
			codes = append(codes, "23")
		}
	}
	if previous == nil || s.Underline != previous.Underline {
		if s.Underline {
			codes = append(codes, "4")
		} else if !legacyMode {
			codes = append(codes, "24")
		}
	}
	if previous == nil || s.Blink != previous.Blink {
		if s.Blink {
			codes = append(codes, "5")
		} else if !legacyMode {
			codes = append(codes, "25")
		}
	}
	if previous == nil || s.Reverse != previous.Reverse {
		if s.Reverse {
			codes = append(codes, "7")
		} else if !legacyMode {
			codes = append(codes, "27")
		}
	}
	if previous == nil || s.Hidden != previous.Hidden {
		if s.Hidden {
			codes = append(codes, "8")
		} else if !legacyMode {
			codes = append(codes, "28")
		}
	}
	if previous == nil || s.Strikethrough != previous.Strikethrough {
		if s.Strikethrough {
			codes = append(codes, "9")
		} else if !legacyMode {
			codes = append(codes, "29")
		}
	}

	// FG color - also recalculate when Bold changes for standard colors (VGA: bold affects brightness)
	fgChanged := previous == nil || s.FgColor != previous.FgColor
	boldChangedWithStdColor := previous != nil && s.Bold != previous.Bold &&
		!s.FgColor.IsDefault() && s.FgColor.Type == ColorStandard && s.FgColor.Index < 8

	if fgChanged || boldChangedWithStdColor {
		if !s.FgColor.IsDefault() && s.FgColor.Type == ColorStandard {
			colorIndex := s.FgColor.Index
			if s.Bold && colorIndex < 8 {
				colorIndex += 8
			}
			rgb := VGAPalette[colorIndex]
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]))
		} else if s.FgColor.IsDefault() {
			codes = append(codes, "39")
		} else {
			for _, c := range s.fgColorCodes() {
				codes = append(codes, fmt.Sprintf("%d", c))
			}
		}
	}
	// BG color
	if previous == nil || s.BgColor != previous.BgColor {
		if !s.BgColor.IsDefault() && s.BgColor.Type == ColorStandard {
			colorIndex := s.BgColor.Index
			rgb := VGAPalette[colorIndex]
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", rgb[0], rgb[1], rgb[2]))
		} else if s.BgColor.IsDefault() {
			codes = append(codes, "49")
		} else {
			for _, c := range s.bgColorCodes() {
				codes = append(codes, fmt.Sprintf("%d", c))
			}
		}
	}

	if len(codes) == 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%sm", strings.Join(codes, ";"))
}

/////////////////////////////////////////////////////////////////////////////
// LINE WITH SEQUENCES
/////////////////////////////////////////////////////////////////////////////

// SGRSequence represents a SGR style at a specific position in a line
type SGRSequence struct {
	Position int  // Position of the character in the line (0-indexed)
	SGR      *SGR // The SGR sequence to apply from this position
}

// LineWithSequences contains a line of text and all SGR changes within that line
type LineWithSequences struct {
	Text      string
	Sequences []SGRSequence
}
