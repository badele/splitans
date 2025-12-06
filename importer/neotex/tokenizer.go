package neotex

// Format neotex
// <POSITION>:<STYLE1>, <STYLE2>, ...;
// <POSITION>:<STYLE1>, <STYLE2>, ...;
//
// Colors:
//   Foreground colors = F<color>
//   Background colors = B<color>
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

	"splitans/types"
)

type Tokenizer struct {
	inputText     []byte
	inputSequence []byte
	Tokens        []types.Token    `json:"tokens"`
	Stats         types.TokenStats `json:"stats"`
}

// Mapping neotex codes to SGR parameters
var neotexToSGR = map[string]string{
	// Foreground colors (lowercase = normal, uppercase = bright)
	"Fk": "30", "FK": "90", // Black
	"Fr": "31", "FR": "91", // Red
	"Fg": "32", "FG": "92", // Green
	"Fy": "33", "FY": "93", // Yellow
	"Fb": "34", "FB": "94", // Blue
	"Fm": "35", "FM": "95", // Magenta
	"Fc": "36", "FC": "96", // Cyan
	"Fw": "37", "FW": "97", // White
	"FD": "39", // Foreground Default

	// Background colors (lowercase = normal, uppercase = bright)
	"Bk": "40", "BK": "100", // Black
	"Br": "41", "BR": "101", // Red
	"Bg": "42", "BG": "102", // Green
	"By": "43", "BY": "103", // Yellow
	"Bb": "44", "BB": "104", // Blue
	"Bm": "45", "BM": "105", // Magenta
	"Bc": "46", "BC": "106", // Cyan
	"Bw": "47", "BW": "107", // White
	"BD": "49", // Background Default

	// Effects (uppercase = ON, lowercase = OFF)
	// Note: Bold is handled by color case (Fr=normal, FR=bright)
	"EM": "2", "Em": "22", // Dim
	"EI": "3", "Ei": "23", // Italic
	"EU": "4", "Eu": "24", // Underline
	"EB": "5", "Eb": "25", // Blink
	"ER": "7", "Er": "27", // Reverse

	// Special
	"R0": "0", // Reset
}

func NewNeotexTokenizer(inputText []byte, inputSequence []byte) *Tokenizer {
	return &Tokenizer{
		inputText:     inputText,
		inputSequence: inputSequence,
		Tokens:        make([]types.Token, 0),
		Stats: types.TokenStats{
			TokensByType: make(map[types.TokenType]int),
			SGRCodes:     make(map[string]int),
			CSISequences: make(map[string]int),
			C0Codes:      make(map[byte]int),
			C1Codes:      make(map[string]int),
		},
	}
}

func NewNeopackTokenizer(data []byte) *Tokenizer {
	inputText, inputSeq := SplitNeopackFormat(80, data)

	return &Tokenizer{
		inputText:     inputText,
		inputSequence: inputSeq,
		Tokens:        make([]types.Token, 0),
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

// SplitNeopackFormat sépare les données neopack en texte et séquences
// Format: "texte (80 car) | séquence"
// Note: width est en nombre de runes (caractères), pas en bytes
func SplitNeopackFormat(width int, data []byte) (inputText []byte, inputSeq []byte) {
	separator := " | "

	lines := strings.Split(string(data), "\n")

	var textLines []string
	var seqLines []string

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

	inputText = []byte(strings.Join(textLines, "\n"))
	inputSeq = []byte(strings.Join(seqLines, "\n"))

	return inputText, inputSeq
}

func (t *Tokenizer) Tokenize() []types.Token {
	// 1. Parse sequences and get style change positions
	styleChanges := t.parseSequenceToMap()

	// 2. Parse text with splits at style change positions
	t.parseTextWithSplits(styleChanges)

	// 3. Sort all tokens by position to interleave text and SGR
	sort.Slice(t.Tokens, func(i, j int) bool {
		return t.Tokens[i].Pos < t.Tokens[j].Pos
	})

	t.calculateStats()
	t.Stats.ParsedPercent = 100

	return t.Tokens
}

// parseTextWithSplits parses text and splits it at style change positions
func (t *Tokenizer) parseTextWithSplits(styleChanges map[int][]types.Token) {
	// Get all style change positions and sort them
	changePositions := make([]int, 0, len(styleChanges))
	for pos := range styleChanges {
		changePositions = append(changePositions, pos)
	}
	sort.Ints(changePositions)

	// Add SGR tokens to the token list
	for _, sgrTokens := range styleChanges {
		t.Tokens = append(t.Tokens, sgrTokens...)
	}

	// Parse text with splits at style change positions
	pos := 0
	changeIdx := 0

	for pos < len(t.inputText) {
		c := t.inputText[pos]

		// Handle C0 control codes
		if c < 0x20 {
			switch c {
			case 0x0A: // LF
				t.Tokens = append(t.Tokens, types.Token{
					Type:   types.TokenC0,
					Pos:    pos,
					Raw:    string(c),
					C0Code: c,
				})
			case 0x0D: // CR
				t.Tokens = append(t.Tokens, types.Token{
					Type:   types.TokenC0,
					Pos:    pos,
					Raw:    string(c),
					C0Code: c,
				})
			case 0x09: // TAB
				t.Tokens = append(t.Tokens, types.Token{
					Type:   types.TokenC0,
					Pos:    pos,
					Raw:    string(c),
					C0Code: c,
				})
			}
			pos++
			continue
		}

		// Find the next style change position
		nextChange := len(t.inputText)
		if changeIdx < len(changePositions) {
			for changeIdx < len(changePositions) && changePositions[changeIdx] <= pos {
				changeIdx++
			}
			if changeIdx < len(changePositions) {
				nextChange = changePositions[changeIdx]
			}
		}

		// Regular character - find continuous text until next style change
		start := pos
		for pos < len(t.inputText) && pos < nextChange && t.inputText[pos] >= 0x20 && t.inputText[pos] != 0x1B {
			pos++
		}

		if pos > start {
			text := string(t.inputText[start:pos])
			t.Tokens = append(t.Tokens, types.Token{
				Type:  types.TokenText,
				Pos:   start,
				Raw:   text,
				Value: text,
			})
		}
	}
}

// parseSequenceToMap parses sequences and returns a map of position -> SGR tokens
func (t *Tokenizer) parseSequenceToMap() map[int][]types.Token {
	styleChanges := make(map[int][]types.Token)

	if len(t.inputSequence) == 0 {
		return styleChanges
	}

	sequenceStr := string(t.inputSequence)

	// Split by semicolons to get position entries
	// Format: "14:FGMN, BD1; 16:BD0; 62:BD1"
	entries := strings.Split(sequenceStr, ";")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Split position from styles
		// Format: "14:FGMN, BD1"
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}

		position, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}

		// Parse styles separated by commas
		// Format: "FGMN, BD1"
		stylesStr := strings.TrimSpace(parts[1])
		styles := strings.Split(stylesStr, ",")

		// Convert neotex codes to SGR parameters
		params := make([]string, 0)
		for _, style := range styles {
			style = strings.TrimSpace(style)

			// Check for RGB format: FRRGGBB or BRRGGBB (7 chars: F/B + 6 hex)
			if len(style) == 7 && (style[0] == 'F' || style[0] == 'B') {
				// Try to parse as RGB
				if r, g, b, err := parseRGBHex(style[1:]); err == nil {
					if style[0] == 'F' {
						// Foreground RGB: 38;2;R;G;B
						params = append(params, "38", "2", fmt.Sprintf("%d", r), fmt.Sprintf("%d", g), fmt.Sprintf("%d", b))
					} else {
						// Background RGB: 48;2;R;G;B
						params = append(params, "48", "2", fmt.Sprintf("%d", r), fmt.Sprintf("%d", g), fmt.Sprintf("%d", b))
					}
					continue
				}
			}

			// Check for Indexed format: Fxxx or Bxxx (2-4 chars: F/B + 1-3 digits)
			if len(style) >= 2 && len(style) <= 4 && (style[0] == 'F' || style[0] == 'B') {
				// Try to parse as indexed color
				if index, err := strconv.Atoi(style[1:]); err == nil && index >= 0 && index <= 255 {
					if style[0] == 'F' {
						// Foreground Indexed: 38;5;n
						params = append(params, "38", "5", fmt.Sprintf("%d", index))
					} else {
						// Background Indexed: 48;5;n
						params = append(params, "48", "5", fmt.Sprintf("%d", index))
					}
					continue
				}
			}

			// Standard neotex code
			if sgrCode, ok := neotexToSGR[style]; ok {
				params = append(params, sgrCode)
			}
		}

		if len(params) > 0 {
			// Create SGR token
			token := types.Token{
				Type:       types.TokenSGR,
				Pos:        position,
				Raw:        entry,
				Parameters: params,
			}
			styleChanges[position] = append(styleChanges[position], token)
		}
	}

	return styleChanges
}

func (t *Tokenizer) calculateStats() {
	t.Stats.TotalTokens = len(t.Tokens)
	t.Stats.FileSize = int64(len(t.inputText) + len(t.inputSequence))

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

// GetStats retourne les statistiques de tokenization
func (t *Tokenizer) GetStats() types.TokenStats {
	return t.Stats
}
