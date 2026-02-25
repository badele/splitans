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
	"sort"
	"strconv"
	"strings"

	"github.com/badele/splitans/internal/importer/ansi"
	"github.com/badele/splitans/internal/types"
)

type Tokenizer struct {
	textLines []string         // Lignes de texte (sans \n)
	seqLines  []string         // Lignes de séquences (sans \n)
	Tokens    []types.Token    `json:"tokens"`
	Stats     types.TokenStats `json:"stats"`
	pos       int
}

// parseHoverColorValue parse code (without HF/HB prefix) into ColorValue.
func parseHoverColorValue(code string) *types.ColorValue {
	if len(code) == 0 {
		return nil
	}
	// RGB hex 6 chars
	if len(code) == 6 {
		if r, g, b, err := parseRGBHex(code); err == nil {
			return &types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}
		}
	}
	// Indexed
	if idx, err := strconv.Atoi(code); err == nil && idx >= 0 && idx <= 255 {
		return &types.ColorValue{Type: types.ColorIndexed, Index: uint8(idx)}
	}
	// Standard single letter
	if len(code) == 1 {
		switch code {
		case "k":
			return &types.ColorValue{Type: types.ColorStandard, Index: 0}
		case "r":
			return &types.ColorValue{Type: types.ColorStandard, Index: 1}
		case "g":
			return &types.ColorValue{Type: types.ColorStandard, Index: 2}
		case "y":
			return &types.ColorValue{Type: types.ColorStandard, Index: 3}
		case "b":
			return &types.ColorValue{Type: types.ColorStandard, Index: 4}
		case "m":
			return &types.ColorValue{Type: types.ColorStandard, Index: 5}
		case "c":
			return &types.ColorValue{Type: types.ColorStandard, Index: 6}
		case "w":
			return &types.ColorValue{Type: types.ColorStandard, Index: 7}
		case "K":
			return &types.ColorValue{Type: types.ColorStandard, Index: 8}
		case "R":
			return &types.ColorValue{Type: types.ColorStandard, Index: 9}
		case "G":
			return &types.ColorValue{Type: types.ColorStandard, Index: 10}
		case "Y":
			return &types.ColorValue{Type: types.ColorStandard, Index: 11}
		case "B":
			return &types.ColorValue{Type: types.ColorStandard, Index: 12}
		case "M":
			return &types.ColorValue{Type: types.ColorStandard, Index: 13}
		case "C":
			return &types.ColorValue{Type: types.ColorStandard, Index: 14}
		case "W":
			return &types.ColorValue{Type: types.ColorStandard, Index: 15}
		}
	}
	return nil
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

	// Hover FG (standard)
	"HFk": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 0} },
	"HFr": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 1} },
	"HFg": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 2} },
	"HFy": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 3} },
	"HFb": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 4} },
	"HFm": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 5} },
	"HFc": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 6} },
	"HFw": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 7} },
	"HFK": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 8} },
	"HFR": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 9} },
	"HFG": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 10} },
	"HFY": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 11} },
	"HFB": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 12} },
	"HFM": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 13} },
	"HFC": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 14} },
	"HFW": func(s *types.SGR) { s.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: 15} },

	// Hover BG (standard)
	"HBk": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 0} },
	"HBr": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 1} },
	"HBg": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 2} },
	"HBy": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 3} },
	"HBb": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 4} },
	"HBm": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 5} },
	"HBc": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 6} },
	"HBw": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 7} },
	"HBK": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 8} },
	"HBR": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 9} },
	"HBG": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 10} },
	"HBY": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 11} },
	"HBB": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 12} },
	"HBM": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 13} },
	"HBC": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 14} },
	"HBW": func(s *types.SGR) { s.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: 15} },
}

// ApplyNeotexHyperlinkCode parses a neotex hyperlink code.
// Returns the hyperlink (nil for OFF) and true if the code was a hyperlink code.
// Format: "HL:<url>" for ON, "Hl" for OFF.
func ApplyNeotexHyperlinkCode(code string) (*types.Hyperlink, bool) {
	// Check for hyperlink ON: HL:<url>
	if strings.HasPrefix(code, "HL:<") && strings.HasSuffix(code, ">") {
		url := code[4 : len(code)-1] // Extract URL between < and >
		return types.NewHyperlink(url), true
	}
	// Check for hyperlink OFF: Hl
	if code == "Hl" {
		return nil, true
	}
	return nil, false
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
	if len(code) == 7 && (code[0] == 'F' || code[0] == 'B' || code[:2] == "HF" || code[:2] == "HB") {
		if r, g, b, err := parseRGBHex(code[1:]); err == nil {
			color := types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}
			switch {
			case code[0] == 'F':
				sgr.FgColor = color
			case code[0] == 'B':
				sgr.BgColor = color
			case code[:2] == "HF":
				sgr.LinkFgColor = color
			case code[:2] == "HB":
				sgr.LinkBgColor = color
			}
			return
		}
	}

	// Gérer Indexed: Fxxx ou Bxxx ou HFxxx/HBxxx (2-4 chars)
	if len(code) >= 2 && len(code) <= 4 && (code[0] == 'F' || code[0] == 'B' || code[:2] == "HF" || code[:2] == "HB") {
		if index, err := strconv.Atoi(code[1:]); err == nil && index >= 0 && index <= 255 {
			color := types.ColorValue{Type: types.ColorIndexed, Index: uint8(index)}
			switch {
			case code[0] == 'F':
				sgr.FgColor = color
			case code[0] == 'B':
				sgr.BgColor = color
			case code[:2] == "HF":
				sgr.LinkFgColor = color
			case code[:2] == "HB":
				sgr.LinkBgColor = color
			}
		}
	}
}

func NewNeotexTokenizer(data []byte, width int) (parsedWidth int, tokenizer *Tokenizer, err error) {
	parsedWidth, textLines, seqLines, err := SplitNeotexFormat(width, data)
	if err != nil {
		return parsedWidth, nil, err
	}

	return parsedWidth, &Tokenizer{
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
	}, nil
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

// SplitNeotexFormat sépare les données neotex en texte et séquences.
// Format: "texte (80 car) | séquence"
// Retourne des tableaux de lignes pour éviter les \n embeddés.
// Renvoie une erreur si une ligne ne respecte pas la largeur ou le séparateur.
func SplitNeotexFormat(width int, data []byte) (parsedWidth int, textLines []string, seqLines []string, err error) {
	separator := " | "

	lines := strings.Split(string(data), "\n")
	if width <= 0 {
		width = 80
	}
	if len(lines) > 0 {
		if twIndex := strings.Index(lines[0], "!TW"); twIndex >= 0 {
			rest := lines[0][twIndex+3:]
			if slashIndex := strings.Index(rest, "/"); slashIndex >= 0 {
				value := rest[slashIndex+1:]
				digitEnd := 0
				for digitEnd < len(value) && value[digitEnd] >= '0' && value[digitEnd] <= '9' {
					digitEnd++
				}
				if digitEnd > 0 {
					if v, err := strconv.Atoi(value[:digitEnd]); err == nil {
						width = v
					}
				}
			}
		}
	}

	parsedWidth = width

	for n, line := range lines {
		if line == "" && n == len(lines)-1 {
			continue
		}
		// Convert to runes to handle UTF-8 properly
		runes := []rune(line)
		sepRunes := []rune(separator)

		if len(runes) < width+len(sepRunes) {
			return parsedWidth, nil, nil, fmt.Errorf("invalid neotex line %d: expected at least %d characters for width %d and separator", n+1, width+len(sepRunes), width)
		}

		// Extract the separator at position width
		actualSep := string(runes[width : width+len(sepRunes)])

		if actualSep != separator {
			return parsedWidth, nil, nil, fmt.Errorf("invalid neotex line %d: expected separator %q at column %d", n+1, separator, width+1)
		}

		// Extract text and sequence using rune positions
		text := string(runes[:width])
		seq := string(runes[width+len(sepRunes):])
		textLines = append(textLines, text)
		seqLines = append(seqLines, seq)
	}

	return parsedWidth, textLines, seqLines, nil
}

func (t *Tokenizer) Tokenize() []types.Token {
	// Extract metadata including SAUCE from sequence lines
	meta := ExtractMetadata(t.seqLines)

	// Convert neotex format to ANSI format (for base tokens)
	ansiData, err := ConvertNeotexToANSI(t.textLines, t.seqLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing neotex: %v\n", err)
		os.Exit(1)
	}

	// Use the existing ANSI tokenizer
	ansiTokenizer := ansi.NewANSITokenizer(ansiData)
	ansiTokens := ansiTokenizer.Tokenize()
	t.Stats = ansiTokenizer.GetStats()

	// Start with ANSI tokens
	t.Tokens = append([]types.Token{}, ansiTokens...)

	// Append hover tokens parsed directly from neotex sequences
	t.appendHoverTokens()

	// If SAUCE metadata was found, add a TokenSauce token
	if meta.Sauce != nil {
		sauceToken := types.Token{
			Type:          types.TokenSauce,
			Pos:           len(t.Tokens),
			Sauce:         meta.Sauce,
			Signification: fmt.Sprintf("SAUCE: %s by %s (%dx%d)", meta.Sauce.Title, meta.Sauce.Author, meta.Sauce.TInfo1, meta.Sauce.TInfo2),
		}
		t.Tokens = append(t.Tokens, sauceToken)
	}

	// Recompute stats including hover and sauce tokens
	t.calculateStats()

	return t.Tokens
}

// GetStats returns tokenization statistics
func (t *Tokenizer) GetStats() types.TokenStats {
	return t.Stats
}

// calculateStats recomputes token statistics for the current token list.
func (t *Tokenizer) calculateStats() {
	// reset
	t.Stats.TotalTokens = len(t.Tokens)
	t.Stats.TokensByType = make(map[types.TokenType]int)
	t.Stats.SGRCodes = make(map[string]int)
	t.Stats.CSISequences = make(map[string]int)
	t.Stats.C0Codes = make(map[byte]int)
	t.Stats.C1Codes = make(map[string]int)
	t.Stats.TotalTextLength = 0

	for _, token := range t.Tokens {
		t.Stats.TokensByType[token.Type]++

		switch token.Type {
		case types.TokenText:
			t.Stats.TotalTextLength += len(token.Value)
		case types.TokenSGR:
			for _, param := range token.Parameters {
				t.Stats.SGRCodes[param]++
			}
		case types.TokenCSI:
			if token.CSINotation != "" {
				t.Stats.CSISequences[token.CSINotation]++
			}
		case types.TokenC0:
			t.Stats.C0Codes[token.C0Code]++
		case types.TokenC1:
			t.Stats.C1Codes[token.C1Code]++
		}
	}
}

// appendHoverTokens parses HF/HB codes from seqLines and appends TokenHoverFg/TokenHoverBg tokens.
// Positions are derived from line offsets so that ordering matches the original text stream.
func (t *Tokenizer) appendHoverTokens() {
	hoverTokens := make([]types.Token, 0)
	offset := 0

	for lineIdx, line := range t.seqLines {
		if line == "" {
			offset += len([]rune(t.textLines[lineIdx]))
			continue
		}
		entries := splitNeotexEntries(line)
		for _, entry := range entries {
			entry = strings.TrimSpace(entry)
			if entry == "" || strings.HasPrefix(entry, "!") {
				continue
			}
			parts := strings.SplitN(entry, ":", 2)
			if len(parts) != 2 {
				continue
			}
			stylesStr := strings.TrimSpace(parts[1])
			styleList := strings.Split(stylesStr, ",")
			for _, style := range styleList {
				style = strings.TrimSpace(style)
				if style == "" {
					continue
				}
				if strings.HasPrefix(style, "HF") {
					if color := parseHoverColorValue(style[2:]); color != nil {
						switch color.Type {
						case types.ColorStandard:
							hoverTokens = append(hoverTokens, types.Token{Type: types.TokenHoverFg, Parameters: []string{"std", fmt.Sprintf("%d", color.Index)}, Pos: offset, Raw: style})
						case types.ColorIndexed:
							hoverTokens = append(hoverTokens, types.Token{Type: types.TokenHoverFg, Parameters: []string{"idx", fmt.Sprintf("%d", color.Index)}, Pos: offset, Raw: style})
						case types.ColorRGB:
							hoverTokens = append(hoverTokens, types.Token{Type: types.TokenHoverFg, Parameters: []string{"rgb", fmt.Sprintf("%d", color.R), fmt.Sprintf("%d", color.G), fmt.Sprintf("%d", color.B)}, Pos: offset, Raw: style})
						}
					}
					continue
				}
				if strings.HasPrefix(style, "HB") {
					if color := parseHoverColorValue(style[2:]); color != nil {
						switch color.Type {
						case types.ColorStandard:
							hoverTokens = append(hoverTokens, types.Token{Type: types.TokenHoverBg, Parameters: []string{"std", fmt.Sprintf("%d", color.Index)}, Pos: offset, Raw: style})
						case types.ColorIndexed:
							hoverTokens = append(hoverTokens, types.Token{Type: types.TokenHoverBg, Parameters: []string{"idx", fmt.Sprintf("%d", color.Index)}, Pos: offset, Raw: style})
						case types.ColorRGB:
							hoverTokens = append(hoverTokens, types.Token{Type: types.TokenHoverBg, Parameters: []string{"rgb", fmt.Sprintf("%d", color.R), fmt.Sprintf("%d", color.G), fmt.Sprintf("%d", color.B)}, Pos: offset, Raw: style})
						}
					}
					continue
				}
			}
		}
		offset += len([]rune(t.textLines[lineIdx]))
	}

	if len(hoverTokens) == 0 {
		return
	}

	// Fusionner et trier par position pour conserver l'ordre du flux
	t.Tokens = append(t.Tokens, hoverTokens...)
	sort.SliceStable(t.Tokens, func(i, j int) bool {
		return t.Tokens[i].Pos < t.Tokens[j].Pos
	})
}
