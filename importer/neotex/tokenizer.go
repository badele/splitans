package neotex

// Format neotex
// <POSITION>:<STYLE1>, <STYLE2>, ...;
// <POSITION>:<STYLE1>, <STYLE2>, ...;
//
// Colors:
//   Foreground colors = F<color>
//   Background colors = B<color> (NOTE: no bright variants for background colors)
//   <color> lowercase = normal colors / uppercase = bright colors
//   k/K = Black, r/R = Red, g/G = Green, y/Y = Yellow
//   b/B = Blue, m/M = Magenta, c/C = Cyan, w/W = White
//   FD = Foreground Default, BD = Background Default
//
// RGB Colors:
//   FRRGGBB = Foreground RGB (e.g., FFF0080 for RGB(255, 0, 128))
//   BRRGGBB = Background RGB (e.g., B00FF00 for RGB(0, 255, 0))
//   RR, GG, BB are 2-digit hexadecimal values (00-FF)
//
// Indexed Colors (256 color palette):
//   Fxxx = Foreground indexed color (e.g., F123 for color index 123)
//   Bxxx = Background indexed color (e.g., B200 for color index 200)
//   xxx is a decimal number from 0 to 255
//
// Effects:
//   E<effect> uppercase = ON / lowercase = OFF
//   M/m = Dim, I/i = Italic, U/u = Underline
//   B/b = Blink, R/r = Reverse
//   Note: Bold is handled by color case (e.g., Fr=normal, FR=bright)
//
// Special:
//   R0 = Reset all styles
//
// Examples:
//   14:Fr, ED      -> Position 14: Foreground Red, Bold ON
//   16:Ed          -> Position 16: Bold OFF
//   20:FD, R0      -> Position 20: Foreground Default, Reset all
//   0:FFF0080      -> Position 0: Foreground RGB(255, 0, 128)
//   10:B00FF00     -> Position 10: Background RGB(0, 255, 0)
//   5:F123         -> Position 5: Foreground indexed color 123
//   15:B200, EU    -> Position 15: Background indexed color 200, Underline ON

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"splitans/importer/ansi"
	"splitans/types"
)

type Tokenizer struct {
	textLines []string         // Lignes de texte (sans \n)
	seqLines  []string         // Lignes de séquences (sans \n)
	Tokens    []types.Token    `json:"tokens"`
	Stats     types.TokenStats `json:"stats"`
}

// NeotexSGRModifier est une fonction qui modifie un SGR
type NeotexSGRModifier func(*types.SGR)

// neotexToSGRModifier mappe les codes neotex vers des modificateurs SGR
var neotexToSGRModifier = map[string]NeotexSGRModifier{
	// Reset
	"R0": func(s *types.SGR) { s.Reset() },

	// Foreground colors (lowercase = normal 0-7, uppercase = bright 8-15)
	"Fk": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 0} },
	"Fr": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 1} },
	"Fg": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 2} },
	"Fy": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 3} },
	"Fb": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 4} },
	"Fm": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 5} },
	"Fc": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 6} },
	"Fw": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 7} },
	"FK": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 8} },
	"FR": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 9} },
	"FG": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 10} },
	"FY": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 11} },
	"FB": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 12} },
	"FM": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 13} },
	"FC": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 14} },
	"FW": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 15} },
	"FD": func(s *types.SGR) { s.FgColor = types.ColorValue{Type: types.ColorStandard, Index: 7} },

	// Background colors
	"Bk": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 0} },
	"Br": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 1} },
	"Bg": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 2} },
	"By": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 3} },
	"Bb": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 4} },
	"Bm": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 5} },
	"Bc": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 6} },
	"Bw": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 7} },
	"BK": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 8} },
	"BR": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 9} },
	"BG": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 10} },
	"BY": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 11} },
	"BB": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 12} },
	"BM": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 13} },
	"BC": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 14} },
	"BW": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 15} },
	"BD": func(s *types.SGR) { s.BgColor = types.ColorValue{Type: types.ColorStandard, Index: 0} },

	// Effects (uppercase = ON, lowercase = OFF)
	"EM": func(s *types.SGR) { s.Dim = true },
	"Em": func(s *types.SGR) { s.Dim = false },
	"EI": func(s *types.SGR) { s.Italic = true },
	"Ei": func(s *types.SGR) { s.Italic = false },
	"EU": func(s *types.SGR) { s.Underline = true },
	"Eu": func(s *types.SGR) { s.Underline = false },
	"EB": func(s *types.SGR) { s.Blink = true },
	"Eb": func(s *types.SGR) { s.Blink = false },
	"ER": func(s *types.SGR) { s.Reverse = true },
	"Er": func(s *types.SGR) { s.Reverse = false },
}

// ApplyNeotexCode applique un code neotex à un SGR
// Gère les codes standards, RGB (FRRGGBB/BRRGGBB) et indexed (Fxxx/Bxxx)
func ApplyNeotexCode(code string, sgr *types.SGR) {
	// Vérifier d'abord la map des codes standards
	if modifier, ok := neotexToSGRModifier[code]; ok {
		modifier(sgr)
		return
	}

	// Gérer RGB: FRRGGBB ou BRRGGBB (7 chars)
	if len(code) == 7 && (code[0] == 'F' || code[0] == 'B') {
		if r, g, b, err := parseRGBHex(code[1:]); err == nil {
			color := types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}
			if code[0] == 'F' {
				sgr.FgColor = color
			} else {
				sgr.BgColor = color
			}
			return
		}
	}

	// Gérer Indexed: Fxxx ou Bxxx (2-4 chars)
	if len(code) >= 2 && len(code) <= 4 && (code[0] == 'F' || code[0] == 'B') {
		if index, err := strconv.Atoi(code[1:]); err == nil && index >= 0 && index <= 255 {
			color := types.ColorValue{Type: types.ColorIndexed, Index: uint8(index)}
			if code[0] == 'F' {
				sgr.FgColor = color
			} else {
				sgr.BgColor = color
			}
		}
	}
}

func NewNeotexTokenizer(data []byte, width int) *Tokenizer {
	textLines, seqLines := SplitNeotexFormat(width, data)

	return &Tokenizer{
		textLines: textLines,
		seqLines:  seqLines,
		Tokens:    make([]types.Token, 0),
		Stats: types.TokenStats{
			TokensByType: make(map[types.TokenType]int),
			SGRCodes:     make(map[string]int),
			CSISequences: make(map[string]int),
			C0Codes:      make(map[byte]int),
			C1Codes:      make(map[string]int),
		},
	}
}

// parseRGBHex parses a 6-character hex string (RRGGBB) and returns R, G, B values
func parseRGBHex(hexStr string) (r, g, b uint8, err error) {
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

// SplitNeotexFormat sépare les données neotex en texte et séquences
// Format: "texte (80 car) | séquence"
// Note: width est en nombre de runes (caractères), pas en bytes
// Retourne des tableaux de lignes pour éviter les \n embeddés
func SplitNeotexFormat(width int, data []byte) (textLines []string, seqLines []string) {
	separator := " | "

	lines := strings.Split(string(data), "\n")

	for n, line := range lines {
		// Convert to runes to handle UTF-8 properly
		runes := []rune(line)
		sepRunes := []rune(separator)

		if len(runes) < width+len(sepRunes) {
			break
		}

		// Extract the separator at position width
		actualSep := string(runes[width : width+len(sepRunes)])

		if actualSep != separator {
			fmt.Printf("Separator not found at position %d, found '%s' instead of '%s' at %d \n",
				width, actualSep, separator, n)
			os.Exit(1)
		}

		// Extract text and sequence using rune positions
		text := string(runes[:width])
		seq := string(runes[width+len(sepRunes):])
		textLines = append(textLines, text)
		seqLines = append(seqLines, seq)
	}

	return textLines, seqLines
}

func (t *Tokenizer) Tokenize() []types.Token {
	// Convert neotex format to ANSI format
	ansiData := ConvertNeotexToANSI(t.textLines, t.seqLines)

	// Use the existing ANSI tokenizer
	ansiTokenizer := ansi.NewANSITokenizer(ansiData)
	t.Tokens = ansiTokenizer.Tokenize()
	t.Stats = ansiTokenizer.GetStats()

	return t.Tokens
}

// GetStats returns tokenization statistics
func (t *Tokenizer) GetStats() types.TokenStats {
	return t.Stats
}
