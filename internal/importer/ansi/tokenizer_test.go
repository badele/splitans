package ansi

import (
	"reflect"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestTokenizeText(t *testing.T) {
	input := []byte("Hello World")
	tokenizer := NewANSITokenizer(input)
	tokens := tokenizer.Tokenize()

	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token, got %d", len(tokens))
	}

	if tokens[0].Type != types.TokenText {
		t.Errorf("Expected types.TokenText, got %v", tokens[0].Type)
	}

	if tokens[0].Value != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", tokens[0].Value)
	}
}

func TestTokenizeC0(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected byte
	}{
		{"LF", []byte{0x0A}, 0x0A},
		{"CR", []byte{0x0D}, 0x0D},
		{"BEL", []byte{0x07}, 0x07},
		{"HT", []byte{0x09}, 0x09},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer(tt.input)
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			if tokens[0].Type != types.TokenC0 {
				t.Errorf("Expected types.TokenC0, got %v", tokens[0].Type)
			}

			if tokens[0].C0Code != tt.expected {
				t.Errorf("Expected code 0x%02X, got 0x%02X", tt.expected, tokens[0].C0Code)
			}
		})
	}
}

func TestTokenizeSGR(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedParams []string
	}{
		{"Reset", "\x1b[0m", []string{"0"}},
		{"Bold", "\x1b[1m", []string{"1"}},
		{"Red", "\x1b[31m", []string{"31"}},
		{"Multiple", "\x1b[1;4;31m", []string{"1", "4", "31"}},
		{"Palette", "\x1b[38;5;123m", []string{"38", "5", "123"}},
		{"RGB", "\x1b[38;2;255;100;50m", []string{"38", "2", "255", "100", "50"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			if tokens[0].Type != types.TokenSGR {
				t.Errorf("Expected types.TokenSGR, got %v", tokens[0].Type)
			}

			if !reflect.DeepEqual(tokens[0].Parameters, tt.expectedParams) {
				t.Errorf("Expected params %v, got %v", tt.expectedParams, tokens[0].Parameters)
			}
		})
	}
}

func TestTokenizeCSI(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedParams []string
	}{
		{"CursorPos", "\x1b[10;5H", []string{"10", "5"}},
		{"CursorUp", "\x1b[5A", []string{"5"}},
		{"EraseDisplay", "\x1b[2J", []string{"2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			if tokens[0].Type != types.TokenCSI {
				t.Errorf("Expected types.TokenCSI, got %v", tokens[0].Type)
			}

			if !reflect.DeepEqual(tokens[0].Parameters, tt.expectedParams) {
				t.Errorf("Expected params %v, got %v", tt.expectedParams, tokens[0].Parameters)
			}
		})
	}
}

func TestTokenizeOSC(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedParams []string
	}{
		{"WindowTitle", "\x1b]2;My Title\x07", []string{"2", "My Title"}},
		{"IconTitle", "\x1b]1;Icon\x1b\\", []string{"1", "Icon"}},
		{"Both", "\x1b]0;Title\x07", []string{"0", "Title"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			if tokens[0].Type != types.TokenOSC {
				t.Errorf("Expected types.TokenOSC, got %v", tokens[0].Type)
			}

			if !reflect.DeepEqual(tokens[0].Parameters, tt.expectedParams) {
				t.Errorf("Expected params %v, got %v", tt.expectedParams, tokens[0].Parameters)
			}
		})
	}
}

func TestTokenizeDCS(t *testing.T) {
	input := "\x1bP1$qm\x1b\\"
	tokenizer := NewANSITokenizer([]byte(input))
	tokens := tokenizer.Tokenize()

	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token, got %d", len(tokens))
	}

	if tokens[0].Type != types.TokenDCS {
		t.Errorf("Expected types.TokenDCS, got %v", tokens[0].Type)
	}

	if tokens[0].Value != "1$qm" {
		t.Errorf("Expected value '1$qm', got %q", tokens[0].Value)
	}
}

func TestTokenizeC1(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedCode string
	}{
		{"NEL", "\x1bE", "NEL"},
		{"IND", "\x1bD", "IND"},
		{"RI", "\x1bM", "RI"},
		{"HTS", "\x1bH", "HTS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			if tokens[0].Type != types.TokenC1 {
				t.Errorf("Expected types.TokenC1, got %v", tokens[0].Type)
			}

			if tokens[0].C1Code != tt.expectedCode {
				t.Errorf("Expected code %q, got %q", tt.expectedCode, tokens[0].C1Code)
			}
		})
	}
}

func TestTokenizeMixed(t *testing.T) {
	input := "Hello \x1b[31mRed\x1b[0m World"
	tokenizer := NewANSITokenizer([]byte(input))
	tokens := tokenizer.Tokenize()

	if len(tokens) != 5 {
		t.Fatalf("Expected 5 tokens, got %d", len(tokens))
	}

	// Token 1: "Hello "
	if tokens[0].Type != types.TokenText || tokens[0].Value != "Hello " {
		t.Errorf("Token 1: expected text 'Hello ', got %v", tokens[0])
	}

	// Token 2: SGR [31m
	if tokens[1].Type != types.TokenSGR {
		t.Errorf("Token 2: expected SGR, got %v", tokens[1].Type)
	}

	// Token 3: "Red"
	if tokens[2].Type != types.TokenText || tokens[2].Value != "Red" {
		t.Errorf("Token 3: expected text 'Red', got %v", tokens[2])
	}

	// Token 4: SGR [0m
	if tokens[3].Type != types.TokenSGR {
		t.Errorf("Token 4: expected SGR, got %v", tokens[3].Type)
	}

	// Token 5: " World"
	if tokens[4].Type != types.TokenText || tokens[4].Value != " World" {
		t.Errorf("Token 5: expected text ' World', got %v", tokens[4])
	}
}

func TestParseSGRParams(t *testing.T) {
	tests := []struct {
		name     string
		params   []string
		expected []string
	}{
		{
			name:     "Reset",
			params:   []string{"0"},
			expected: []string{"Reset"},
		},
		{
			name:     "Bold Red",
			params:   []string{"1", "31"},
			expected: []string{"Bold", "ForegroundRed"},
		},
		{
			name:     "Palette",
			params:   []string{"38", "5", "123"},
			expected: []string{"Foreground Palette Index: 123"},
		},
		{
			name:     "RGB",
			params:   []string{"38", "2", "255", "100", "50"},
			expected: []string{"Foreground RGB: 255,100,50"},
		},
		{
			name:     "Background RGB",
			params:   []string{"48", "2", "128", "64", "32"},
			expected: []string{"Background RGB: 128,64,32"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSGRParams(tt.params)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseEDParams(t *testing.T) {
	tests := []struct {
		name     string
		params   []string
		expected []string
	}{
		{
			name:     "Default (EraseBelow)",
			params:   []string{""},
			expected: []string{"EraseBelow"},
		},
		{
			name:     "EraseBelow",
			params:   []string{"0"},
			expected: []string{"EraseBelow"},
		},
		{
			name:     "EraseAbove",
			params:   []string{"1"},
			expected: []string{"EraseAbove"},
		},
		{
			name:     "EraseAll",
			params:   []string{"2"},
			expected: []string{"EraseAll"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseEDParams(tt.params)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseNumberParam(t *testing.T) {
	tests := []struct {
		name         string
		param        string
		defaultValue int
		expected     int
	}{
		{"Empty string returns default", "", 10, 10},
		{"Valid number", "5", 10, 5},
		{"Invalid number returns default", "abc", 10, 10},
		{"Zero", "0", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseNumberParam(tt.param, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseDoubleNumbersParam(t *testing.T) {
	tests := []struct {
		name         string
		params       []string
		defaultValue []int
		expected     []int
	}{
		{
			name:         "Two valid numbers",
			params:       []string{"10", "5"},
			defaultValue: []int{1, 1},
			expected:     []int{10, 5},
		},
		{
			name:         "Empty params returns default",
			params:       []string{},
			defaultValue: []int{1, 1},
			expected:     []int{1, 1},
		},
		{
			name:         "Invalid number returns default",
			params:       []string{"abc", "5"},
			defaultValue: []int{1, 1},
			expected:     []int{1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDoubleNumbersParam(tt.params, tt.defaultValue)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCSIWithSignification(t *testing.T) {
	tests := []struct {
		name                  string
		input                 string
		expectedType          types.TokenType
		expectedNotation      string
		expectedSignification string
	}{
		{
			name:                  "Cursor Up",
			input:                 "\x1b[5A",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI Ps A",
			expectedSignification: "Cursor Up 5 times",
		},
		{
			name:                  "Cursor Down",
			input:                 "\x1b[3B",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI Ps B",
			expectedSignification: "Cursor Down 3 times",
		},
		{
			name:                  "Cursor Right",
			input:                 "\x1b[2C",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI Ps C",
			expectedSignification: "Cursor Right 2 times",
		},
		{
			name:                  "Cursor Left",
			input:                 "\x1b[4D",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI Ps D",
			expectedSignification: "Cursor Left 4 times",
		},
		{
			name:                  "Erase Display",
			input:                 "\x1b[2J",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI Ps J",
			expectedSignification: "EraseAll",
		},
		{
			name:                  "Save Cursor Position",
			input:                 "\x1b[s",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI s",
			expectedSignification: "Save Cursor Position",
		},
		{
			name:                  "Restore Cursor Position",
			input:                 "\x1b[u",
			expectedType:          types.TokenCSI,
			expectedNotation:      "CSI u",
			expectedSignification: "Restore Cursor Position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			token := tokens[0]

			if token.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, token.Type)
			}

			if token.CSINotation != tt.expectedNotation {
				t.Errorf("Expected notation %q, got %q", tt.expectedNotation, token.CSINotation)
			}

			if token.Signification != tt.expectedSignification {
				t.Errorf("Expected signification %q, got %q", tt.expectedSignification, token.Signification)
			}
		})
	}
}

func TestTokenUnknown(t *testing.T) {
	// Test d'une séquence CSI non reconnue
	input := "\x1b[99Z"
	tokenizer := NewANSITokenizer([]byte(input))
	tokens := tokenizer.Tokenize()

	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token, got %d", len(tokens))
	}

	if tokens[0].Type != types.TokenUnknown {
		t.Errorf("Expected types.TokenUnknown, got %v", tokens[0].Type)
	}
}

func TestCSIWithoutParameters(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedType     types.TokenType
		expectedNotation string
	}{
		{
			name:             "Cursor Up without params",
			input:            "\x1b[A",
			expectedType:     types.TokenCSI,
			expectedNotation: "CSI Ps A",
		},
		{
			name:             "Cursor Down without params",
			input:            "\x1b[B",
			expectedType:     types.TokenCSI,
			expectedNotation: "CSI Ps B",
		},
		{
			name:             "Cursor Right without params",
			input:            "\x1b[C",
			expectedType:     types.TokenCSI,
			expectedNotation: "CSI Ps C",
		},
		{
			name:             "Cursor Left without params",
			input:            "\x1b[D",
			expectedType:     types.TokenCSI,
			expectedNotation: "CSI Ps D",
		},
		{
			name:             "Erase Display without params",
			input:            "\x1b[J",
			expectedType:     types.TokenCSI,
			expectedNotation: "CSI Ps J",
		},
		{
			name:             "Cursor Position without params",
			input:            "\x1b[H",
			expectedType:     types.TokenCSI,
			expectedNotation: "CSI Ps H",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			token := tokens[0]

			if token.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, token.Type)
			}

			if token.CSINotation != tt.expectedNotation {
				t.Errorf("Expected notation %q, got %q", tt.expectedNotation, token.CSINotation)
			}

			// Vérifie que ça ne panic pas et que la signification est vide ou par défaut
			// (pas de panic = test réussi)
		})
	}
}

func TestTokenCSIInterrupted(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		c0Char   byte
		expected string
	}{
		{
			name:     "CSI interrupted by LF",
			input:    "\x1b[5\x0A",
			c0Char:   0x0A,
			expected: "CSI interrupted by C0 control (0x0A)",
		},
		{
			name:     "CSI interrupted by CR",
			input:    "\x1b[10;5\x0D",
			c0Char:   0x0D,
			expected: "CSI interrupted by C0 control (0x0D)",
		},
		{
			name:     "CSI interrupted by BEL",
			input:    "\x1b[2\x07",
			c0Char:   0x07,
			expected: "CSI interrupted by C0 control (0x07)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewANSITokenizer([]byte(tt.input))
			tokens := tokenizer.Tokenize()

			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}

			token := tokens[0]

			if token.Type != types.TokenCSIInterupted {
				t.Errorf("Expected types.TokenCSIInterupted, got %v", token.Type)
			}

			if token.CSINotation != tt.expected {
				t.Errorf("Expected notation %q, got %q", tt.expected, token.CSINotation)
			}

			if tokenizer.Stats.PosFirstBadSequence == 0 {
				t.Error("Expected PosFirstBadSequence to be set")
			}

			if tokenizer.Stats.ParsedPercent == 0 {
				t.Error("Expected ParsedPercent to be calculated")
			}
		})
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tokenType types.TokenType
		expected  string
	}{
		{types.TokenText, "TokenText"},
		{types.TokenC0, "TokenC0"},
		{types.TokenC1, "TokenC1"},
		{types.TokenCSI, "TokenCSI"},
		{types.TokenCSIInterupted, "TokenCSIInterupted"},
		{types.TokenSGR, "TokenSGR"},
		{types.TokenDCS, "TokenDCS"},
		{types.TokenOSC, "TokenOSC"},
		{types.TokenEscape, "TokenEscape"},
		{types.TokenUnknown, "TokenUnknown"},
		{types.TokenType(999), "TokenType(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.tokenType.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTokenString(t *testing.T) {
	tests := []struct {
		name     string
		token    types.Token
		expected string
	}{
		{
			name:     "Text token",
			token:    types.Token{Type: types.TokenText, Value: "Hello"},
			expected: "TEXT: Hello",
		},
		{
			name:     "C0 token - LF",
			token:    types.Token{Type: types.TokenC0, C0Code: 0x0A},
			expected: "C0: LF",
		},
		{
			name:     "C0 token - unknown",
			token:    types.Token{Type: types.TokenC0, C0Code: 0xFF},
			expected: "C0: unknown",
		},
		{
			name:     "C1 token",
			token:    types.Token{Type: types.TokenC1, C1Code: "NEL"},
			expected: "C1: NEL",
		},
		{
			name:     "CSI token",
			token:    types.Token{Type: types.TokenCSI, CSINotation: "CSI Ps A"},
			expected: "CSI:  Notation:CSI Ps A",
		},
		{
			name:     "SGR token",
			token:    types.Token{Type: types.TokenSGR, CSINotation: "CSI Ps... m"},
			expected: "SGR:  Notation:CSI Ps... m",
		},
		{
			name:     "DCS token",
			token:    types.Token{Type: types.TokenDCS, Raw: "\x1bP1$qm\x1b\\"},
			expected: "DCS: \x1bP1$qm\x1b\\",
		},
		{
			name:     "OSC token",
			token:    types.Token{Type: types.TokenOSC, Raw: "\x1b]2;Title\x07"},
			expected: "OSC: \x1b]2;Title\x07",
		},
		{
			name:     "Escape token",
			token:    types.Token{Type: types.TokenEscape, Raw: "\x1bc"},
			expected: "ESC: \x1bc",
		},
		{
			name:     "Unknown token",
			token:    types.Token{Type: types.TokenUnknown},
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTokenTypeJSON(t *testing.T) {
	tests := []struct {
		name      string
		tokenType types.TokenType
		expected  string
	}{
		{"TokenText", types.TokenText, `"TokenText"`},
		{"TokenC0", types.TokenC0, `"TokenC0"`},
		{"TokenCSI", types.TokenCSI, `"TokenCSI"`},
		{"TokenSGR", types.TokenSGR, `"TokenSGR"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.tokenType.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(data))
			}

			var tokenType types.TokenType
			err = tokenType.UnmarshalJSON(data)
			if err != nil {
				t.Fatalf("UnmarshalJSON error: %v", err)
			}

			if tokenType != tt.tokenType {
				t.Errorf("Expected %v, got %v", tt.tokenType, tokenType)
			}
		})
	}
}

func TestTokenTypeUnmarshalJSON_Invalid(t *testing.T) {
	var tokenType types.TokenType
	err := tokenType.UnmarshalJSON([]byte(`"InvalidType"`))
	if err == nil {
		t.Error("Expected error for invalid token type")
	}

	err = tokenType.UnmarshalJSON([]byte(`invalid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
