package types

import (
	"encoding/json"
	"fmt"
)

// ============================================================================
// EXPORTED
// ============================================================================

/////////////////////////////////////////////////////////////////////////////
// TOKEN TYPE
/////////////////////////////////////////////////////////////////////////////

type TokenType int

const (
	TokenText TokenType = iota
	TokenC0
	TokenC1
	TokenCSI
	TokenCSIInterupted
	TokenSGR
	TokenHoverFg
	TokenHoverBg
	TokenDCS
	TokenOSC
	TokenEscape
	TokenSauce
	TokenSequenceComment
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
	case TokenHoverFg:
		return "TokenHoverFg"
	case TokenHoverBg:
		return "TokenHoverBg"
	case TokenDCS:
		return "TokenDCS"
	case TokenOSC:
		return "TokenOSC"
	case TokenEscape:
		return "TokenEscape"
	case TokenSauce:
		return "TokenSauce"
	case TokenSequenceComment:
		return "TokenSequenceComment"
	case TokenUnknown:
		return "TokenUnknown"
	default:
		return fmt.Sprintf("TokenType(%d)", t)
	}
}

/////////////////////////////////////////////////////////////////////////
// SEQUENCE SCOPE
/////////////////////////////////////////////////////////////////////////

type SequenceScope int

const (
	ScopeVT SequenceScope = iota
	ScopeEX
	ScopeVX
)

func (s SequenceScope) String() string {
	switch s {
	case ScopeVT:
		return "VT"
	case ScopeEX:
		return "EX"
	case ScopeVX:
		return "VX"
	default:
		return "VT"
	}
}

func (s SequenceScope) Normalize() SequenceScope {
	switch s {
	case ScopeVT, ScopeEX, ScopeVX:
		return s
	default:
		return ScopeVT
	}
}

func (s SequenceScope) IncludesVT() bool {
	n := s.Normalize()
	return n == ScopeVT || n == ScopeVX
}

func (s SequenceScope) IncludesExport() bool {
	n := s.Normalize()
	return n == ScopeEX || n == ScopeVX
}

func ParseSequenceScope(value string) (SequenceScope, bool) {
	switch value {
	case "VT":
		return ScopeVT, true
	case "EX":
		return ScopeEX, true
	case "VX":
		return ScopeVX, true
	default:
		return ScopeVT, false
	}
}

func (s SequenceScope) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SequenceScope) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	parsed, ok := ParseSequenceScope(value)
	if !ok {
		return fmt.Errorf("unknown SequenceScope: %s", value)
	}
	*s = parsed
	return nil
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
	case "TokenHoverFg":
		*t = TokenHoverFg
	case "TokenHoverBg":
		*t = TokenHoverBg
	case "TokenDCS":
		*t = TokenDCS
	case "TokenOSC":
		*t = TokenOSC
	case "TokenEscape":
		*t = TokenEscape
	case "TokenSauce":
		*t = TokenSauce
	case "TokenSequenceComment":
		*t = TokenSequenceComment
	case "TokenUnknown":
		*t = TokenUnknown
	default:
		return fmt.Errorf("unknown TokenType: %s", s)
	}

	return nil
}

/////////////////////////////////////////////////////////////////////////////
// TOKEN
/////////////////////////////////////////////////////////////////////////////

type Token struct {
	Type          TokenType     `json:"type"`
	Pos           int           `json:"pos"`
	Scope         SequenceScope `json:"scope,omitempty"`
	Line          int           `json:"line,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	Raw           string        `json:"raw"`
	Value         string        `json:"value,omitempty"`
	Parameters    []string      `json:"parameters,omitempty"`
	C0Code        byte          `json:"c0_code,omitempty"`
	C1Code        string        `json:"c1_code,omitempty"`
	CSINotation   string        `json:"csi_notation,omitempty"`
	Signification string        `json:"signification,omitempty"`
	Sauce         *Sauce        `json:"sauce,omitempty"`
	Hyperlink     *Hyperlink    `json:"hyperlink,omitempty"` // OSC 8 hyperlink
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
	case TokenSequenceComment:
		return "COMMENT: " + t.Comment
	default:
		return "UNKNOWN"
	}
}

/////////////////////////////////////////////////////////////////////////////
// TOKEN STATS
/////////////////////////////////////////////////////////////////////////////

type TokenStats struct {
	TotalTokens         int               `json:"total_tokens"`
	TokensByType        map[TokenType]int `json:"tokens_by_type"`
	SGRCodes            map[string]int    `json:"sgr_codes"`
	CSISequences        map[string]int    `json:"csi_sequences"`
	C0Codes             map[byte]int      `json:"c0_codes"`
	C1Codes             map[string]int    `json:"c1_codes"`
	TotalTextLength     int               `json:"total_text_length"`
	FileSize            int64             `json:"file_size"`
	ParsedPercent       float64           `json:"parsed_percent"`
	PosFirstBadSequence int64             `json:"pos_first_bad_sequence"`
}
