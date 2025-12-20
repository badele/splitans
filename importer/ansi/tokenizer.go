package ansi

// Sources :
// - https://wezterm.org/escape-sequences.html#graphic-rendition-sgr
// - https://vt100.net/docs/vt510-rm/chapter4.html
// - https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// - https://ecma-international.org/wp-content/uploads/ECMA-48_5th_edition_june_1991.pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/badele/splitans/types"
)

type Tokenizer struct {
	input   []byte
	pos     int              // Position en octets dans input
	runePos int              // Position en runes (caractères Unicode)
	Tokens  []types.Token    `json:"tokens"`
	Stats   types.TokenStats `json:"stats"`
}

func NewANSITokenizer(input []byte) *Tokenizer {
	stats := types.TokenStats{
		TokensByType:        make(map[types.TokenType]int),
		SGRCodes:            make(map[string]int),
		CSISequences:        make(map[string]int),
		C0Codes:             make(map[byte]int),
		C1Codes:             make(map[string]int),
		FileSize:            int64(len(input)),
		ParsedPercent:       0.0,
		PosFirstBadSequence: 0,
	}

	return &Tokenizer{
		input:   input,
		pos:     0,
		runePos: 0,
		Tokens:  make([]types.Token, 0),
		Stats:   stats,
	}
}

func (t *Tokenizer) Tokenize() []types.Token {
	for t.pos < len(t.input) {
		t.nextToken()

		// Verify if parsing was interrupted by bad CSI
		if len(t.Tokens) > 0 && t.Tokens[len(t.Tokens)-1].Type == types.TokenCSIInterupted {
			t.Stats.ParsedPercent = float64(t.Stats.PosFirstBadSequence) / float64(t.Stats.FileSize) * 100
			return t.Tokens
		}
	}

	t.Stats.ParsedPercent = 100

	t.calculateStats()

	return t.Tokens
}

func (t *Tokenizer) nextToken() {
	if t.pos >= len(t.input) {
		return
	}

	c := t.input[t.pos]

	// C0 (0x00-0x1F)
	// not printable characters
	if c < 0x20 {
		if c == 0x1B { // ESC
			t.parseEscape(t.pos)
		} else if c == 0x1A {
			t.parseSauce(t.pos)
		} else {
			t.parseC0(t.pos, c)
		}
		return
	}

	t.parseText(t.pos, t.runePos)
}

func (t *Tokenizer) parseC0(start int, code byte) {
	token := types.Token{
		Type:   types.TokenC0,
		Pos:    t.runePos,
		Raw:    string(code),
		C0Code: code,
	}
	t.Tokens = append(t.Tokens, token)
	t.pos++
	t.runePos++ // 1 octet ASCII = 1 rune
}

func (t *Tokenizer) parseEscape(start int) {
	startRunePos := t.runePos
	startBytePos := t.pos
	t.pos++

	if t.pos >= len(t.input) {
		t.Tokens = append(t.Tokens, types.Token{
			Type: types.TokenEscape,
			Pos:  startRunePos,
			Raw:  string(t.input[startBytePos:t.pos]),
		})
		t.runePos += (t.pos - startBytePos)
		return
	}

	next := t.input[t.pos]

	if name, ok := C1Sequences[string(next)]; ok {
		t.pos++

		switch name {
		case "CSI":
			t.parseCSI(startBytePos, startRunePos)
		case "DCS":
			t.parseDCS(startBytePos, startRunePos)
		case "OSC":
			t.parseOSC(startBytePos, startRunePos)
		case "ST":
			t.Tokens = append(t.Tokens, types.Token{
				Type:   types.TokenC1,
				Pos:    startRunePos,
				Raw:    string(t.input[startBytePos:t.pos]),
				C1Code: name,
			})
			t.runePos += (t.pos - startBytePos)
		default:
			t.Tokens = append(t.Tokens, types.Token{
				Type:   types.TokenC1,
				Pos:    startRunePos,
				Raw:    string(t.input[startBytePos:t.pos]),
				C1Code: name,
			})
			t.runePos += (t.pos - startBytePos)
		}
		return
	}

	t.parseOtherEscape(startBytePos, startRunePos)
}

func (t *Tokenizer) parseSauce(start int) {
	t.pos++

	t.Tokens = append(t.Tokens, types.Token{
		Type: types.TokenSauce,
		Pos:  t.pos,
		Raw:  string(t.input[t.pos:]),
	})

	t.pos = len(t.input)
	t.runePos = t.pos
}

func (t *Tokenizer) parseCSI(startBytePos int, startRunePos int) {
	params := t.collectParams()

	if t.pos >= len(t.input) {
		t.Tokens = append(t.Tokens, types.Token{
			Type: types.TokenCSI,
			Pos:  startRunePos,
			Raw:  string(t.input[startBytePos:t.pos]),
		})
		t.runePos += (t.pos - startBytePos)
		return
	}

	final := t.input[t.pos]
	t.pos++

	token := types.Token{
		Type:       types.TokenCSI,
		Pos:        startRunePos,
		Raw:        string(t.input[startBytePos:t.pos]),
		Parameters: params,
	}

	// if final is C0 control character, the sequence is invalid/interrupted
	if final < 0x20 {
		token.Type = types.TokenCSIInterupted
		token.CSINotation = fmt.Sprintf("CSI interrupted by C0 control (0x%02X)", final)
		t.Tokens = append(t.Tokens, token)
		t.Stats.PosFirstBadSequence = int64(t.pos)
		t.runePos += (t.pos - startBytePos)
		return
	}

	// Detect final parameter
	switch final {
	case 'A':
		{
			token.CSINotation = "CSI Ps A"
			number := 1
			if len(params) > 0 {
				number = ParseNumberParam(params[0], 1)
			}
			token.Signification = fmt.Sprintf("Cursor Up %d times", number)
		}
	case 'B':
		{
			token.CSINotation = "CSI Ps B"
			number := 1
			if len(params) > 0 {
				number = ParseNumberParam(params[0], 1)
			}
			token.Signification = fmt.Sprintf("Cursor Down %d times", number)
		}
	case 'C':
		{
			token.CSINotation = "CSI Ps C"
			number := 1
			if len(params) > 0 {
				number = ParseNumberParam(params[0], 1)
			}
			token.Signification = fmt.Sprintf("Cursor Right %d times", number)
		}
	case 'D':
		{
			token.CSINotation = "CSI Ps D"
			number := 1
			if len(params) > 0 {
				number = ParseNumberParam(params[0], 1)
			}
			token.Signification = fmt.Sprintf("Cursor Left %d times", number)
		}
	case 'H':
		// ESC [ H 	Moves the cursor to line 1, column 1 (Home).
		// ESC [ 6 H 	Moves the cursor to line 6, column 1.
		// ESC [ ; 12 H 	Moves the cursor to line 1, column 12.
		// ESC [ 6 ; 12 H 	Moves the cursor to line 6, column 12.
		// ESC [ 99 ; 99 H 	Moves the cursor to end of Page.
		{
			token.CSINotation = "CSI Ps H"
			numbers := ParseDoubleNumbersParam(params, []int{1, 1})
			token.Signification = fmt.Sprintf("Cursor Position %d", numbers)
		}
	case 'J':
		{
			token.CSINotation = "CSI Ps J"
			token.Signification = strings.Join(ParseEDParams(params), ", ")
		}
	case 's':
		{
			token.CSINotation = "CSI s"
			token.Signification = "Save Cursor Position"
		}
	case 'u':
		{
			token.CSINotation = "CSI u"
			token.Signification = "Restore Cursor Position"
		}
	case 'm':
		{
			token.Type = types.TokenSGR
			token.CSINotation = "CSI Ps... m"
		}
	default:
		{
			token.Type = types.TokenUnknown
			token.CSINotation = ""
		}
	}

	t.Tokens = append(t.Tokens, token)
	t.runePos += (t.pos - startBytePos)
}

func (t *Tokenizer) parseDCS(startBytePos int, startRunePos int) {
	data := make([]byte, 0)
	for t.pos < len(t.input) {
		if t.input[t.pos] == 0x1B && t.pos+1 < len(t.input) && t.input[t.pos+1] == '\\' {
			// Trouvé ESC \
			t.pos += 2
			break
		}
		if t.input[t.pos] == 0x9C {
			// Trouvé ST (8-bit)
			t.pos++
			break
		}
		data = append(data, t.input[t.pos])
		t.pos++
	}

	t.Tokens = append(t.Tokens, types.Token{
		Type:  types.TokenDCS,
		Pos:   startRunePos,
		Raw:   string(t.input[startBytePos:t.pos]),
		Value: string(data),
	})
	t.runePos += (t.pos - startBytePos)
}

func (t *Tokenizer) parseOSC(startBytePos int, startRunePos int) {
	data := make([]byte, 0)
	for t.pos < len(t.input) {
		if t.input[t.pos] == 0x07 { // BEL
			t.pos++
			break
		}
		if t.input[t.pos] == 0x1B && t.pos+1 < len(t.input) && t.input[t.pos+1] == '\\' {
			t.pos += 2
			break
		}
		if t.input[t.pos] == 0x9C {
			t.pos++
			break
		}
		data = append(data, t.input[t.pos])
		t.pos++
	}

	parts := strings.SplitN(string(data), ";", 2)
	params := make([]string, 0)
	if len(parts) > 0 {
		params = append(params, parts[0])
		if len(parts) > 1 {
			params = append(params, parts[1])
		}
	}

	t.Tokens = append(t.Tokens, types.Token{
		Type:       types.TokenOSC,
		Pos:        startRunePos,
		Raw:        string(t.input[startBytePos:t.pos]),
		Value:      string(data),
		Parameters: params,
	})
	t.runePos += (t.pos - startBytePos)
}

func (t *Tokenizer) parseOtherEscape(startBytePos int, startRunePos int) {
	// ESC c, ESC 7, ESC 8, ESC =, ESC >, ESC (0, ESC (B, ESC #8
	if t.pos >= len(t.input) {
		t.Tokens = append(t.Tokens, types.Token{
			Type: types.TokenEscape,
			Pos:  startRunePos,
			Raw:  string(t.input[startBytePos:t.pos]),
		})
		t.runePos += (t.pos - startBytePos) // ASCII: 1 byte = 1 rune
		return
	}

	next := t.input[t.pos]
	t.pos++

	// Two characters
	if next == '(' || next == ')' || next == '#' {
		if t.pos < len(t.input) {
			t.pos++
		}
	}

	t.Tokens = append(t.Tokens, types.Token{
		Type: types.TokenEscape,
		Pos:  startRunePos,
		Raw:  string(t.input[startBytePos:t.pos]),
	})
	t.runePos += (t.pos - startBytePos) // ASCII: 1 byte = 1 rune
}

func (t *Tokenizer) collectParams() []string {
	// [] == ESC [ H
	// [6,1] == ESC [ 6 H
	// [1,12]  == ESC [ ; 12 H
	// [6,12] == ESC [ 6 ; 12 H
	params := make([]string, 0)
	var current bytes.Buffer

	for t.pos < len(t.input) {
		b := t.input[t.pos]

		if (b >= '0' && b <= '9') || b == ';' || b == ':' {
			if b == ';' || b == ':' {
				// Always append current param (even if empty) when separator is found
				params = append(params, current.String())
				current.Reset()
				t.pos++
			} else {
				current.WriteByte(b)
				t.pos++
			}
		} else if b == '?' || b == '>' || b == '!' || b == '$' || b == '\'' || b == '"' || b == ' ' {
			// Ignore these characters in parameters
			t.pos++
		} else {
			// CSI or SGR Final byte or invalid character
			break
		}
	}

	if current.Len() > 0 {
		params = append(params, current.String())
	}

	return params
}

func (t *Tokenizer) parseText(startByte int, startRune int) {
	for t.pos < len(t.input) {
		b := t.input[t.pos]

		if b < 0x20 {
			break
		}

		_, size := utf8.DecodeRune(t.input[t.pos:])
		t.pos += size
		t.runePos++ // Incrémente la position en runes
	}

	if t.pos > startByte {
		text := string(t.input[startByte:t.pos])
		t.Tokens = append(t.Tokens, types.Token{
			Type:  types.TokenText,
			Pos:   startRune, // Utilise la position en runes
			Raw:   text,
			Value: text,
		})
	}
}

func (t *Tokenizer) calculateStats() {
	t.Stats.TotalTokens = len(t.Tokens)

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

func ParseSGRParams(params []string) []string {
	result := make([]string, 0)

	const defaultCode = 0
	for i := 0; i < len(params); i++ {
		if params[i] == "" {
			if name, ok := SGRCodes[defaultCode]; ok {
				result = append(result, name)
				continue
			}
		}

		code, err := strconv.Atoi(params[i])
		if err != nil {
			result = append(result, "Invalid: "+params[i])
			continue
		}

		// Gestion des codes étendus (38, 48, 58 pour les couleurs)
		if (code == 38 || code == 48 || code == 58) && i+2 < len(params) {
			mode, _ := strconv.Atoi(params[i+1])
			if mode == 5 && i+2 < len(params) {
				// Palette index
				prefix := "Foreground"
				if code == 48 {
					prefix = "Background"
				} else if code == 58 {
					prefix = "Underline"
				}
				result = append(result, prefix+" Palette Index: "+params[i+2])
				i += 2
				continue
			} else if (mode == 2 || mode == 6) && i+4 < len(params) {
				// RGB ou RGBA
				prefix := "Foreground"
				if code == 48 {
					prefix = "Background"
				} else if code == 58 {
					prefix = "Underline"
				}
				colorType := "RGB"
				if mode == 6 && i+5 < len(params) {
					colorType = "RGBA"
					result = append(result, prefix+" "+colorType+": "+params[i+2]+","+params[i+3]+","+params[i+4]+","+params[i+5])
					i += 5
				} else {
					result = append(result, prefix+" "+colorType+": "+params[i+2]+","+params[i+3]+","+params[i+4])
					i += 4
				}
				continue
			}
		}

		if name, ok := SGRCodes[code]; ok {
			result = append(result, name)
		} else {
			result = append(result, "Unknown: "+strconv.Itoa(code))
		}
	}

	return result
}

func ParseEDParams(params []string) []string {
	result := make([]string, 0)

	const defaultCode = 0
	for i := 0; i < len(params); i++ {

		if params[i] == "" {
			if name, ok := EDCodes[defaultCode]; ok {
				result = append(result, name)
				continue
			}
		}

		code, err := strconv.Atoi(params[i])
		if err != nil {
			result = append(result, "Invalid: "+params[i])
			continue
		}

		if name, ok := EDCodes[code]; ok {
			result = append(result, name)
		} else {
			result = append(result, "Unknown: "+strconv.Itoa(code))
		}
	}

	return result
}

func ParseNumberParam(param string, defaultValue int) int {
	if param == "" {
		return defaultValue
	}

	num, err := strconv.Atoi(param)
	if err != nil {
		return defaultValue
	}
	return num
}

func ParseDoubleNumbersParam(params []string, defaultValue []int) []int {
	result := defaultValue

	for i := 0; i < len(params); i++ {
		num, err := strconv.Atoi(params[i])
		if err != nil {
			return defaultValue
		}

		result[i] = num
	}

	return result
}

// GetStats retourne les statistiques de tokenization
func (t *Tokenizer) GetStats() types.TokenStats {
	return t.Stats
}
