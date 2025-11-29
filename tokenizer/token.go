package tokenizer

import (
	"encoding/json"
	"fmt"
)

/////////////////////////////////////////////////////////////////////////////
// TOKEN
/////////////////////////////////////////////////////////////////////////////

type TokenType int

const (
	TokenText TokenType = iota
	TokenC0
	TokenC1
	TokenCSI
	TokenCSIInterupted
	TokenSGR
	TokenDCS
	TokenOSC
	TokenEscape
	TokenUnknown
)

func (t TokenType) String() string {
	switch t {
	case TokenText:
		return "TokenText"
	case TokenC0:
		return "TokenC0"
	case TokenC1:
		return "TokenC1"
	case TokenCSI:
		return "TokenCSI"
	case TokenCSIInterupted:
		return "TokenCSIInterupted"
	case TokenSGR:
		return "TokenSGR"
	case TokenDCS:
		return "TokenDCS"
	case TokenOSC:
		return "TokenOSC"
	case TokenEscape:
		return "TokenEscape"
	case TokenUnknown:
		return "TokenUnknown"
	default:
		return fmt.Sprintf("TokenType(%d)", t)
	}
}

func (t TokenType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TokenType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "TokenText":
		*t = TokenText
	case "TokenC0":
		*t = TokenC0
	case "TokenC1":
		*t = TokenC1
	case "TokenCSI":
		*t = TokenCSI
	case "TokenCSIInterupted":
		*t = TokenCSIInterupted
	case "TokenSGR":
		*t = TokenSGR
	case "TokenDCS":
		*t = TokenDCS
	case "TokenOSC":
		*t = TokenOSC
	case "TokenEscape":
		*t = TokenEscape
	case "TokenUnknown":
		*t = TokenUnknown
	default:
		return fmt.Errorf("unknown TokenType: %s", s)
	}

	return nil
}

type Token struct {
	Type          TokenType `json:"type"`
	Pos           int       `json:"pos"`
	Raw           string    `json:"raw"`
	Value         string    `json:"value,omitempty"`
	Parameters    []string  `json:"parameters,omitempty"`
	C0Code        byte      `json:"c0_code,omitempty"`
	C1Code        string    `json:"c1_code,omitempty"`
	CSINotation   string    `json:"csi_notation,omitempty"`
	Signification string    `json:"signification,omitempty"`
}

// C0 control codes names
var C0Names = map[byte]string{
	0x00: "NUL",
	0x01: "SOH",
	0x02: "STX",
	0x03: "ETX",
	0x04: "EOT",
	0x05: "ENQ",
	0x06: "ACK",
	0x07: "BEL",
	0x08: "BS",
	0x09: "HT",
	0x0A: "LF",
	0x0B: "VT",
	0x0C: "FF",
	0x0D: "CR",
	0x0E: "SO",
	0x0F: "SI",
	0x10: "DLE",
	0x11: "DC1",
	0x12: "DC2",
	0x13: "DC3",
	0x14: "DC4",
	0x15: "NAK",
	0x16: "SYN",
	0x17: "ETB",
	0x18: "CAN",
	0x19: "EM",
	0x1A: "SUB",
	0x1B: "ESC",
	0x1C: "FS",
	0x1D: "GS",
	0x1E: "RS",
	0x1F: "US",
}

// C1 control codes (7-bit representation)
var C1Sequences = map[string]string{
	"D":  "IND", // Index
	"E":  "NEL", // Next Line
	"H":  "HTS", // Horizontal Tab Set
	"M":  "RI",  // Reverse Index
	"P":  "DCS", // Device Control String
	"[":  "CSI", // Control Sequence Introducer
	"\\": "ST",  // String Terminator
	"]":  "OSC", // Operating System Command
}

// SGR codes descriptions
var SGRCodes = map[int]string{
	0:   "Reset",
	1:   "Bold",
	2:   "Dim",
	3:   "Italic",
	4:   "Underline",
	5:   "Blink",
	6:   "RapidBlink",
	7:   "Inverse",
	8:   "Invisible",
	9:   "StrikeThrough",
	21:  "DoubleUnderline",
	22:  "NormalIntensity",
	23:  "ItalicOff",
	24:  "UnderlineOff",
	25:  "BlinkOff",
	27:  "InverseOff",
	28:  "InvisibleOff",
	29:  "StrikeThroughOff",
	30:  "ForegroundBlack",
	31:  "ForegroundRed",
	32:  "ForegroundGreen",
	33:  "ForegroundYellow",
	34:  "ForegroundBlue",
	35:  "ForegroundMagenta",
	36:  "ForegroundCyan",
	37:  "ForegroundWhite",
	39:  "ForegroundDefault",
	40:  "BackgroundBlack",
	41:  "BackgroundRed",
	42:  "BackgroundGreen",
	43:  "BackgroundYellow",
	44:  "BackgroundBlue",
	45:  "BackgroundMagenta",
	46:  "BackgroundCyan",
	47:  "BackgroundWhite",
	49:  "BackgroundDefault",
	53:  "OverlineOn",
	55:  "OverlineOff",
	59:  "UnderlineColorDefault",
	90:  "ForegroundBrightBlack",
	91:  "ForegroundBrightRed",
	92:  "ForegroundBrightGreen",
	93:  "ForegroundBrightYellow",
	94:  "ForegroundBrightBlue",
	95:  "ForegroundBrightMagenta",
	96:  "ForegroundBrightCyan",
	97:  "ForegroundBrightWhite",
	100: "BackgroundBrightBlack",
	101: "BackgroundBrightRed",
	102: "BackgroundBrightGreen",
	103: "BackgroundBrightYellow",
	104: "BackgroundBrightBlue",
	105: "BackgroundBrightMagenta",
	106: "BackgroundBrightCyan",
	107: "BackgroundBrightWhite",
}

// ED codes descriptions
var EDCodes = map[int]string{
	0: "EraseBelow",
	1: "EraseAbove",
	2: "EraseAll",
}

func (t Token) String() string {
	switch t.Type {
	case TokenText:
		return "TEXT: " + t.Value
	case TokenC0:
		if name, ok := C0Names[t.C0Code]; ok {
			return "C0: " + name
		}
		return "C0: unknown"
	case TokenC1:
		return "C1: " + t.C1Code
	case TokenCSI:
		return "CSI: " + " Notation:" + t.CSINotation
	case TokenSGR:
		return "SGR: " + " Notation:" + t.CSINotation
	case TokenDCS:
		return "DCS: " + t.Raw
	case TokenOSC:
		return "OSC: " + t.Raw
	case TokenEscape:
		return "ESC: " + t.Raw
	default:
		return "UNKNOWN"
	}
}
