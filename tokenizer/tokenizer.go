package tokenizer

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
)

type Tokenizer struct {
	input               []byte
	pos                 int
	Tokens              []Token `json:"tokens"`
	FileSize            int64   `json:"file_size"`
	PosFirstBadSequence int64   `json:"pos_first_bad_sequence"`
	ParsedPercent       float64 `json:"parsed_percent"`
}

func NewTokenizer(input []byte) *Tokenizer {
	return &Tokenizer{
		input:               input,
		pos:                 0,
		Tokens:              make([]Token, 0),
		FileSize:            int64(len(input)),
		PosFirstBadSequence: 0,
		ParsedPercent:       0.0,
	}
}

func (t *Tokenizer) Tokenize() []Token {
	for t.pos < len(t.input) {
		t.nextToken()

		// Verify if parsing was interrupted by bad CSI
		if len(t.Tokens) > 0 && t.Tokens[len(t.Tokens)-1].Type == TokenCSIInterupted {
			t.ParsedPercent = float64(t.PosFirstBadSequence) / float64(t.FileSize) * 100
			return t.Tokens
		}
	}

	t.ParsedPercent = 100
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
		} else {
			t.parseC0(t.pos, c)
		}
		return
	}

	t.parseText(t.pos)
}

func (t *Tokenizer) parseC0(start int, code byte) {
	token := Token{
		Type:   TokenC0,
		Pos:    start,
		Raw:    string(code),
		C0Code: code,
	}
	t.Tokens = append(t.Tokens, token)
	t.pos++
}

func (t *Tokenizer) parseEscape(start int) {
	t.pos++ 

	if t.pos >= len(t.input) {
		t.Tokens = append(t.Tokens, Token{
			Type: TokenEscape,
			Pos:  start,
			Raw:  string(t.input[start:t.pos]),
		})
		return
	}

	next := t.input[t.pos]

	if name, ok := C1Sequences[string(next)]; ok {
		t.pos++

		switch name {
		case "CSI":
			t.parseCSI(start)
		case "DCS":
			t.parseDCS(start)
		case "OSC":
			t.parseOSC(start)
		case "ST":
			t.Tokens = append(t.Tokens, Token{
				Type:   TokenC1,
				Pos:    start,
				Raw:    string(t.input[start:t.pos]),
				C1Code: name,
			})
		default:
			t.Tokens = append(t.Tokens, Token{
				Type:   TokenC1,
				Pos:    start,
				Raw:    string(t.input[start:t.pos]),
				C1Code: name,
			})
		}
		return
	}

	t.parseOtherEscape(start)
}

func (t *Tokenizer) parseCSI(start int) {
	params := t.collectParams()

	if t.pos >= len(t.input) {
		t.Tokens = append(t.Tokens, Token{
			Type: TokenCSI,
			Pos:  start,
			Raw:  string(t.input[start:t.pos]),
		})
		return
	}

	final := t.input[t.pos]
	t.pos++

	token := Token{
		Type:       TokenCSI,
		Pos:        start,
		Raw:        string(t.input[start:t.pos]),
		Parameters: params,
	}

	// if final is C0 control character, the sequence is invalid/interrupted
	if final < 0x20 {
		token.Type = TokenCSIInterupted
		token.CSINotation = fmt.Sprintf("CSI interrupted by C0 control (0x%02X)", final)
		t.Tokens = append(t.Tokens, token)
		t.PosFirstBadSequence = int64(t.pos)
		return
	}

	// Detect final parameter
	switch final {
	case 'A':
		{
			token.CSINotation = "CSI Ps A"
			if len(params) > 0 {
				number := ParseNumberParam(params[0], 1)
				token.Signification = fmt.Sprintf("Cursor Up %d times", number)
			}
		}
	case 'B':
		{
			token.CSINotation = "CSI Ps B"
			if len(params) > 0 {
				number := ParseNumberParam(params[0], 1)
				token.Signification = fmt.Sprintf("Cursor Down %d times", number)
			}
		}
	case 'C':
		{
			token.CSINotation = "CSI Ps C"
			if len(params) > 0 {
				number := ParseNumberParam(params[0], 1)
				token.Signification = fmt.Sprintf("Cursor Forward %d times", number)
			}
		}
	case 'D':
		{
			token.CSINotation = "CSI Ps D"
			if len(params) > 0 {
				number := ParseNumberParam(params[0], 1)
				token.Signification = fmt.Sprintf("Cursor Backward %d times", number)
			}
		}
	case 'H':
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
			token.Type = TokenSGR
			token.CSINotation = "CSI Ps... m"
		}
	default:
		{
			token.Type = TokenUnknown
			token.CSINotation = ""
		}
	}

	t.Tokens = append(t.Tokens, token)
}

func (t *Tokenizer) parseDCS(start int) {
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

	t.Tokens = append(t.Tokens, Token{
		Type:  TokenDCS,
		Pos:   start,
		Raw:   string(t.input[start:t.pos]),
		Value: string(data),
	})
}

func (t *Tokenizer) parseOSC(start int) {
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

	t.Tokens = append(t.Tokens, Token{
		Type:       TokenOSC,
		Pos:        start,
		Raw:        string(t.input[start:t.pos]),
		Value:      string(data),
		Parameters: params,
	})
}

func (t *Tokenizer) parseOtherEscape(start int) {
	// ESC c, ESC 7, ESC 8, ESC =, ESC >, ESC (0, ESC (B, ESC #8
	if t.pos >= len(t.input) {
		t.Tokens = append(t.Tokens, Token{
			Type: TokenEscape,
			Pos:  start,
			Raw:  string(t.input[start:t.pos]),
		})
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

	t.Tokens = append(t.Tokens, Token{
		Type: TokenEscape,
		Pos:  start,
		Raw:  string(t.input[start:t.pos]),
	})
}

func (t *Tokenizer) collectParams() []string {
	params := make([]string, 0)
	var current bytes.Buffer

	for t.pos < len(t.input) {
		b := t.input[t.pos]

		if (b >= '0' && b <= '9') || b == ';' || b == ':' {
			if b == ';' || b == ':' {
				if current.Len() > 0 || len(params) > 0 {
					params = append(params, current.String())
					current.Reset()
				}
				t.pos++
			} else {
				current.WriteByte(b)
				t.pos++
			}
		} else if b == '?' || b == '>' || b == '!' || b == '$' || b == '\'' || b == '"' || b == ' ' {
			// Intermediate bytes, on les ignore pour l'instant
			t.pos++
		} else {
			// C'est le byte final ou un caractère non valide
			break
		}
	}

	if current.Len() > 0 {
		params = append(params, current.String())
	}

	return params
}

func (t *Tokenizer) parseText(start int) {
	for t.pos < len(t.input) {
		b := t.input[t.pos]

		if b < 0x20 {
			break
		}

		_, size := utf8.DecodeRune(t.input[t.pos:])
		t.pos += size
	}

	if t.pos > start {
		text := string(t.input[start:t.pos])
		t.Tokens = append(t.Tokens, Token{
			Type:  TokenText,
			Pos:   start,
			Raw:   text,
			Value: text,
		})
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
