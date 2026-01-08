package neotex

import (
	"reflect"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestSplitNeotexFormat(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		data         []byte
		expectedText []string
		expectedSeq  []string
	}{
		{
			name:         "Single line",
			width:        5,
			data:         []byte("Hello | 1:Fr"),
			expectedText: []string{"Hello"},
			expectedSeq:  []string{"1:Fr"},
		},
		{
			name:         "Multiple lines",
			width:        5,
			data:         []byte("Hello | 1:Fr\nWorld | 1:Fg"),
			expectedText: []string{"Hello", "World"},
			expectedSeq:  []string{"1:Fr", "1:Fg"},
		},
		{
			name:         "Unicode text",
			width:        9,
			data:         []byte("Héllo àüé | 1:Fr"),
			expectedText: []string{"Héllo àüé"},
			expectedSeq:  []string{"1:Fr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, textLines, seqLines := SplitNeotexFormat(tt.width, tt.data)
			if !reflect.DeepEqual(textLines, tt.expectedText) {
				t.Errorf("Text lines: expected %v, got %v", tt.expectedText, textLines)
			}
			if !reflect.DeepEqual(seqLines, tt.expectedSeq) {
				t.Errorf("Seq lines: expected %v, got %v", tt.expectedSeq, seqLines)
			}
		})
	}
}

func TestApplyNeotexCode(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		checkFn   func(*types.SGR) bool
		checkDesc string
	}{
		// Foreground colors (lowercase = standard 0-7)
		{
			name: "Foreground Black",
			code: "Fk",
			checkFn: func(s *types.SGR) bool {
				return s.FgColor.Type == types.ColorStandard && s.FgColor.Index == 0
			},
			checkDesc: "FgColor should be standard index 0",
		},
		{
			name: "Foreground Red",
			code: "Fr",
			checkFn: func(s *types.SGR) bool {
				return s.FgColor.Type == types.ColorStandard && s.FgColor.Index == 1
			},
			checkDesc: "FgColor should be standard index 1",
		},
		{
			name: "Foreground Green",
			code: "Fg",
			checkFn: func(s *types.SGR) bool {
				return s.FgColor.Type == types.ColorStandard && s.FgColor.Index == 2
			},
			checkDesc: "FgColor should be standard index 2",
		},
		// Foreground colors (uppercase = bright 8-15)
		{
			name: "Foreground Bright Red",
			code: "FR",
			checkFn: func(s *types.SGR) bool {
				return s.FgColor.Type == types.ColorStandard && s.FgColor.Index == 9
			},
			checkDesc: "FgColor should be standard index 9 (bright red)",
		},
		// Background colors
		{
			name: "Background Black",
			code: "Bk",
			checkFn: func(s *types.SGR) bool {
				return s.BgColor.Type == types.ColorStandard && s.BgColor.Index == 0
			},
			checkDesc: "BgColor should be standard index 0",
		},
		{
			name: "Background Red",
			code: "Br",
			checkFn: func(s *types.SGR) bool {
				return s.BgColor.Type == types.ColorStandard && s.BgColor.Index == 1
			},
			checkDesc: "BgColor should be standard index 1",
		},
		// Effects
		{
			name: "Dim ON",
			code: "EM",
			checkFn: func(s *types.SGR) bool {
				return s.Dim == true
			},
			checkDesc: "Dim should be true",
		},
		{
			name: "Dim OFF",
			code: "Em",
			checkFn: func(s *types.SGR) bool {
				return s.Dim == false
			},
			checkDesc: "Dim should be false",
		},
		{
			name: "Italic ON",
			code: "EI",
			checkFn: func(s *types.SGR) bool {
				return s.Italic == true
			},
			checkDesc: "Italic should be true",
		},
		{
			name: "Underline ON",
			code: "EU",
			checkFn: func(s *types.SGR) bool {
				return s.Underline == true
			},
			checkDesc: "Underline should be true",
		},
		{
			name: "Blink ON",
			code: "EB",
			checkFn: func(s *types.SGR) bool {
				return s.Blink == true
			},
			checkDesc: "Blink should be true",
		},
		{
			name: "Reverse ON",
			code: "ER",
			checkFn: func(s *types.SGR) bool {
				return s.Reverse == true
			},
			checkDesc: "Reverse should be true",
		},
		// RGB colors
		{
			name: "Foreground RGB",
			code: "FFF0080",
			checkFn: func(s *types.SGR) bool {
				return s.FgColor.Type == types.ColorRGB &&
					s.FgColor.R == 255 && s.FgColor.G == 0 && s.FgColor.B == 128
			},
			checkDesc: "FgColor should be RGB(255, 0, 128)",
		},
		{
			name: "Background RGB",
			code: "B00FF00",
			checkFn: func(s *types.SGR) bool {
				return s.BgColor.Type == types.ColorRGB &&
					s.BgColor.R == 0 && s.BgColor.G == 255 && s.BgColor.B == 0
			},
			checkDesc: "BgColor should be RGB(0, 255, 0)",
		},
		// Indexed colors
		{
			name: "Foreground Indexed",
			code: "F123",
			checkFn: func(s *types.SGR) bool {
				return s.FgColor.Type == types.ColorIndexed && s.FgColor.Index == 123
			},
			checkDesc: "FgColor should be indexed 123",
		},
		{
			name: "Background Indexed",
			code: "B200",
			checkFn: func(s *types.SGR) bool {
				return s.BgColor.Type == types.ColorIndexed && s.BgColor.Index == 200
			},
			checkDesc: "BgColor should be indexed 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sgr := types.NewSGR()
			ApplyNeotexCode(tt.code, sgr)
			if !tt.checkFn(sgr) {
				t.Errorf("ApplyNeotexCode(%q): %s", tt.code, tt.checkDesc)
			}
		})
	}
}

func TestApplyNeotexCodeReset(t *testing.T) {
	sgr := types.NewSGR()
	// Set some values
	ApplyNeotexCode("Fr", sgr)
	ApplyNeotexCode("EU", sgr)

	// Verify they are set
	if sgr.FgColor.Index != 1 {
		t.Error("FgColor should be red before reset")
	}
	if !sgr.Underline {
		t.Error("Underline should be true before reset")
	}

	// Reset
	ApplyNeotexCode("R0", sgr)

	// Verify reset
	if sgr.FgColor.Type != types.ColorStandard || sgr.FgColor.Index != 7 {
		t.Errorf("FgColor should be default (7) after reset, got %d", sgr.FgColor.Index)
	}
	if sgr.Underline {
		t.Error("Underline should be false after reset")
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		seqLines []string
		expected NeotexMetadata
	}{
		{
			name:     "Version only",
			seqLines: []string{"!V1"},
			expected: NeotexMetadata{
				Version: 1,
				Extra:   make(map[string]string),
			},
		},
		{
			name:     "Trimmed width with total",
			seqLines: []string{"!TW73/80"},
			expected: NeotexMetadata{
				TrimmedWidth: 73,
				Width:        80,
				Extra:        make(map[string]string),
			},
		},
		{
			name:     "Number of lines",
			seqLines: []string{"!NL42"},
			expected: NeotexMetadata{
				NbLines: 42,
				Extra:   make(map[string]string),
			},
		},
		{
			name:     "Multiple metadata",
			seqLines: []string{"!V1; !TW73/80; !NL42"},
			expected: NeotexMetadata{
				Version:      1,
				TrimmedWidth: 73,
				Width:        80,
				NbLines:      42,
				Extra:        make(map[string]string),
			},
		},
		{
			name:     "Mixed with sequences",
			seqLines: []string{"1:Fr; !V1", "2:Fg"},
			expected: NeotexMetadata{
				Version: 1,
				Extra:   make(map[string]string),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := ExtractMetadata(tt.seqLines)
			if meta.Version != tt.expected.Version {
				t.Errorf("Version: expected %d, got %d", tt.expected.Version, meta.Version)
			}
			if meta.TrimmedWidth != tt.expected.TrimmedWidth {
				t.Errorf("TrimmedWidth: expected %d, got %d", tt.expected.TrimmedWidth, meta.TrimmedWidth)
			}
			if meta.Width != tt.expected.Width {
				t.Errorf("Width: expected %d, got %d", tt.expected.Width, meta.Width)
			}
			if meta.NbLines != tt.expected.NbLines {
				t.Errorf("NbLines: expected %d, got %d", tt.expected.NbLines, meta.NbLines)
			}
		})
	}
}

func TestConvertNeotexToANSI(t *testing.T) {
	tests := []struct {
		name      string
		textLines []string
		seqLines  []string
		contains  []string // Strings that should be in the output
	}{
		{
			name:      "Simple text no styles",
			textLines: []string{"Hello"},
			seqLines:  []string{""},
			contains:  []string{"Hello"},
		},
		{
			name:      "Text with foreground color",
			textLines: []string{"Hello"},
			seqLines:  []string{"1:Fr"},
			contains:  []string{"Hello", "\x1b["},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertNeotexToANSI(tt.textLines, tt.seqLines)
			resultStr := string(result)
			for _, s := range tt.contains {
				if !contains(resultStr, s) {
					t.Errorf("Expected output to contain %q, got %q", s, resultStr)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewNeotexTokenizer(t *testing.T) {
	// Test basic tokenizer creation
	data := []byte("Hello | 1:Fr")
	_, tokenizer := NewNeotexTokenizer(data, 5)

	if tokenizer == nil {
		t.Fatal("NewNeotexTokenizer returned nil")
	}

	tokens := tokenizer.Tokenize()
	if len(tokens) == 0 {
		t.Error("Expected at least one token")
	}
}

func TestTokenizerWithMultipleStyles(t *testing.T) {
	// Test with multiple style changes
	data := []byte("RedGreen | 1:Fr; 4:Fg")
	_, tokenizer := NewNeotexTokenizer(data, 8)

	tokens := tokenizer.Tokenize()

	// Should have tokens for text and SGR
	hasText := false
	hasSGR := false
	for _, token := range tokens {
		if token.Type == types.TokenText {
			hasText = true
		}
		if token.Type == types.TokenSGR {
			hasSGR = true
		}
	}

	if !hasText {
		t.Error("Expected at least one text token")
	}
	if !hasSGR {
		t.Error("Expected at least one SGR token")
	}
}

func TestTokenizerGetStats(t *testing.T) {
	data := []byte("Hello | 1:Fr")
	_, tokenizer := NewNeotexTokenizer(data, 5)
	tokenizer.Tokenize()

	stats := tokenizer.GetStats()
	if stats.TotalTokens == 0 {
		t.Error("Expected TotalTokens to be set after tokenization")
	}
}

func TestParseLineSequences(t *testing.T) {
	tests := []struct {
		name     string
		seqLine  string
		expected []styleChange
	}{
		{
			name:     "Empty",
			seqLine:  "",
			expected: []styleChange{},
		},
		{
			name:    "Single style",
			seqLine: "1:Fr",
			expected: []styleChange{
				{position: 0, codes: []string{"Fr"}},
			},
		},
		{
			name:    "Multiple styles same position",
			seqLine: "1:Fr, EU",
			expected: []styleChange{
				{position: 0, codes: []string{"Fr", "EU"}},
			},
		},
		{
			name:    "Multiple positions",
			seqLine: "1:Fr; 5:Fg",
			expected: []styleChange{
				{position: 0, codes: []string{"Fr"}},
				{position: 4, codes: []string{"Fg"}},
			},
		},
		{
			name:    "Skip metadata",
			seqLine: "!V1; 1:Fr",
			expected: []styleChange{
				{position: 0, codes: []string{"Fr"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLineSequences(tt.seqLine)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d style changes, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if result[i].position != expected.position {
					t.Errorf("Position %d: expected %d, got %d", i, expected.position, result[i].position)
				}
				if !reflect.DeepEqual(result[i].codes, expected.codes) {
					t.Errorf("Codes %d: expected %v, got %v", i, expected.codes, result[i].codes)
				}
			}
		})
	}
}
