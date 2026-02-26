package neotex

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/badele/splitans/internal/exporter"
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
			_, textLines, seqLines, err := SplitNeotexFormat(tt.width, tt.data)
			if err != nil {
				t.Fatalf("SplitNeotexFormat failed: %v", err)
			}
			if !reflect.DeepEqual(textLines, tt.expectedText) {
				t.Errorf("Text lines: expected %v, got %v", tt.expectedText, textLines)
			}
			if !reflect.DeepEqual(seqLines, tt.expectedSeq) {
				t.Errorf("Seq lines: expected %v, got %v", tt.expectedSeq, seqLines)
			}
		})
	}
}

func TestSplitNeotexFormatRejectsInvalidLines(t *testing.T) {
	tests := []struct {
		name  string
		width int
		data  []byte
	}{
		{
			name:  "Line too short",
			width: 5,
			data:  []byte("Hi |"),
		},
		{
			name:  "Separator at wrong position",
			width: 5,
			data:  []byte("Hello| 1:Fr"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := SplitNeotexFormat(tt.width, tt.data)
			if err == nil {
				t.Fatal("Expected error, got nil")
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
	versionMajor, versionMinor, versionPatch, ok := parseNeotexVersion(exporter.NeotexVersion)
	if !ok {
		t.Fatalf("invalid NeotexVersion %q", exporter.NeotexVersion)
	}
	semverVersion := fmt.Sprintf("%d.23", versionMajor)

	tests := []struct {
		name     string
		seqLines []string
		expected NeotexMetadata
	}{
		{
			name:     "Version only",
			seqLines: []string{fmt.Sprintf("!V%s", exporter.NeotexVersion)},
			expected: NeotexMetadata{
				VersionRaw:   exporter.NeotexVersion,
				Version:      versionMajor,
				VersionMajor: versionMajor,
				VersionMinor: versionMinor,
				VersionPatch: versionPatch,
				Extra:        make(map[string]string),
			},
		},
		{
			name:     "Semver version",
			seqLines: []string{fmt.Sprintf("!V%s", semverVersion)},
			expected: NeotexMetadata{
				VersionRaw:   semverVersion,
				Version:      versionMajor,
				VersionMajor: versionMajor,
				VersionMinor: 23,
				VersionPatch: 0,
				Extra:        make(map[string]string),
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
			seqLines: []string{fmt.Sprintf("!V%s; !TW73/80; !NL42", exporter.NeotexVersion)},
			expected: NeotexMetadata{
				VersionRaw:   exporter.NeotexVersion,
				Version:      versionMajor,
				VersionMajor: versionMajor,
				VersionMinor: versionMinor,
				VersionPatch: versionPatch,
				TrimmedWidth: 73,
				Width:        80,
				NbLines:      42,
				Extra:        make(map[string]string),
			},
		},
		{
			name:     "Mixed with sequences",
			seqLines: []string{fmt.Sprintf("1:Fr; !V%s", exporter.NeotexVersion), "2:Fg"},
			expected: NeotexMetadata{
				VersionRaw:   exporter.NeotexVersion,
				Version:      versionMajor,
				VersionMajor: versionMajor,
				VersionMinor: versionMinor,
				VersionPatch: versionPatch,
				Extra:        make(map[string]string),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := ExtractMetadata(tt.seqLines)
			if err != nil {
				t.Fatalf("ExtractMetadata failed: %v", err)
			}
			if meta.VersionRaw != tt.expected.VersionRaw {
				t.Errorf("VersionRaw: expected %q, got %q", tt.expected.VersionRaw, meta.VersionRaw)
			}
			if meta.Version != tt.expected.Version {
				t.Errorf("Version: expected %d, got %d", tt.expected.Version, meta.Version)
			}
			if meta.VersionMajor != tt.expected.VersionMajor {
				t.Errorf("VersionMajor: expected %d, got %d", tt.expected.VersionMajor, meta.VersionMajor)
			}
			if meta.VersionMinor != tt.expected.VersionMinor {
				t.Errorf("VersionMinor: expected %d, got %d", tt.expected.VersionMinor, meta.VersionMinor)
			}
			if meta.VersionPatch != tt.expected.VersionPatch {
				t.Errorf("VersionPatch: expected %d, got %d", tt.expected.VersionPatch, meta.VersionPatch)
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

func TestExtractMetadataPalette(t *testing.T) {
	seqLines := []string{"!P2=FF0080; 1:Fr"}
	meta, err := ExtractMetadata(seqLines)
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}
	color, ok := meta.Palette[2]
	if !ok {
		t.Fatalf("expected palette index 2 to be parsed")
	}
	if color.R != 0xFF || color.G != 0x00 || color.B != 0x80 {
		t.Fatalf("unexpected palette color: %d %d %d", color.R, color.G, color.B)
	}
}

func TestExtractMetadataPaletteInvalidEntry(t *testing.T) {
	seqLines := []string{"!P2<FF0080>; 1:Fr"}
	_, err := ExtractMetadata(seqLines)
	if err == nil {
		t.Fatal("expected invalid palette entry to return error")
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
			meta, err := ExtractMetadata(tt.seqLines)
			if err != nil {
				t.Fatalf("ExtractMetadata failed: %v", err)
			}
			result, err := ConvertNeotexToANSI(tt.textLines, tt.seqLines, meta.Palette)
			if err != nil {
				t.Fatalf("ConvertNeotexToANSI failed: %v", err)
			}
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
	_, tokenizer, err := NewNeotexTokenizer(data, 5)
	if err != nil {
		t.Fatalf("NewNeotexTokenizer failed: %v", err)
	}

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
	_, tokenizer, err := NewNeotexTokenizer(data, 8)
	if err != nil {
		t.Fatalf("NewNeotexTokenizer failed: %v", err)
	}

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
	_, tokenizer, err := NewNeotexTokenizer(data, 5)
	if err != nil {
		t.Fatalf("NewNeotexTokenizer failed: %v", err)
	}
	tokenizer.Tokenize()

	stats := tokenizer.GetStats()
	if stats.TotalTokens == 0 {
		t.Error("Expected TotalTokens to be set after tokenization")
	}
}

func TestParseLineSequences(t *testing.T) {
	tests := []struct {
		name      string
		seqLine   string
		expected  []styleChange
		expectErr bool
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
			seqLine: fmt.Sprintf("!V%s.23; 1:Fr", exporter.NeotexVersion),
			expected: []styleChange{
				{position: 0, codes: []string{"Fr"}},
			},
		},
		{
			name:    "Skip protected metadata",
			seqLine: "!ST<Hello;World>; 1:Fr",
			expected: []styleChange{
				{position: 0, codes: []string{"Fr"}},
			},
		},
		{
			name:      "Non increasing positions",
			seqLine:   "5:Fr; 4:Fg",
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Duplicate positions",
			seqLine:   "3:Fr; 3:Fg",
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Reject bright background",
			seqLine:   "1:BR",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineSequences(tt.seqLine, map[int]types.ColorValue{})
			if tt.expectErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
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

func TestParseLineSequencesPalette(t *testing.T) {
	palette := map[int]types.ColorValue{
		2: {Type: types.ColorRGB, R: 0xFF, G: 0x00, B: 0x80},
	}
	result, err := parseLineSequences("1:FP2", palette)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 style change, got %d", len(result))
	}
	if len(result[0].codes) != 1 || result[0].codes[0] != "FFF0080" {
		t.Fatalf("Unexpected palette conversion: %v", result[0].codes)
	}

	_, err = parseLineSequences("1:FP9", map[int]types.ColorValue{})
	if err == nil {
		t.Fatal("Expected undefined palette index to return error")
	}
}

func TestApplyNeotexHyperlinkCode(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectURL     string
		expectOff     bool
		expectMatched bool
	}{
		{
			name:          "Hyperlink ON with URL",
			code:          "HL:<https://example.com>",
			expectURL:     "https://example.com",
			expectOff:     false,
			expectMatched: true,
		},
		{
			name:          "Hyperlink ON with complex URL",
			code:          "HL:<https://example.com/path?query=1&foo=bar>",
			expectURL:     "https://example.com/path?query=1&foo=bar",
			expectOff:     false,
			expectMatched: true,
		},
		{
			name:          "Hyperlink OFF",
			code:          "Hl",
			expectURL:     "",
			expectOff:     true,
			expectMatched: true,
		},
		{
			name:          "Not a hyperlink code",
			code:          "Fr",
			expectURL:     "",
			expectOff:     false,
			expectMatched: false,
		},
		{
			name:          "Not a hyperlink code - reset",
			code:          "R0",
			expectURL:     "",
			expectOff:     false,
			expectMatched: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hyperlink, matched := ApplyNeotexHyperlinkCode(tt.code)

			if matched != tt.expectMatched {
				t.Errorf("Expected matched=%v, got %v", tt.expectMatched, matched)
			}

			if tt.expectMatched {
				if tt.expectOff {
					if hyperlink != nil {
						t.Errorf("Expected nil hyperlink for OFF, got %v", hyperlink)
					}
				} else {
					if hyperlink == nil {
						t.Fatal("Expected non-nil hyperlink")
					}
					if hyperlink.URL != tt.expectURL {
						t.Errorf("Expected URL %q, got %q", tt.expectURL, hyperlink.URL)
					}
				}
			}
		})
	}
}

func TestExtractMetadataSauceShortAndProtected(t *testing.T) {
	seqLines := []string{"!ST<Hello;World>; !SAJane; !SG<ACME, Inc.>; 1:Fr"}
	meta, err := ExtractMetadata(seqLines)
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}
	if meta.Sauce == nil {
		t.Fatalf("expected SAUCE metadata to be parsed")
	}
	if meta.Sauce.Title != "Hello;World" {
		t.Fatalf("unexpected title: got %q", meta.Sauce.Title)
	}
	if meta.Sauce.Author != "Jane" {
		t.Fatalf("unexpected author: got %q", meta.Sauce.Author)
	}
	if meta.Sauce.Group != "ACME, Inc." {
		t.Fatalf("unexpected group: got %q", meta.Sauce.Group)
	}
}

func TestExtractMetadataRejectsAngleBrackets(t *testing.T) {
	seqLines := []string{"!STBad>Title; !SAJane"}
	meta, err := ExtractMetadata(seqLines)
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}
	if meta.Sauce == nil {
		t.Fatalf("expected SAUCE metadata to be parsed")
	}
	if meta.Sauce.Title != "" {
		t.Fatalf("expected title to be rejected, got %q", meta.Sauce.Title)
	}
	if meta.Sauce.Author != "Jane" {
		t.Fatalf("unexpected author: got %q", meta.Sauce.Author)
	}
}

func TestParseLineSequencesWithHyperlinks(t *testing.T) {
	tests := []struct {
		name                 string
		seqLine              string
		expectedCount        int
		expectedHasHyperlink bool
		expectedHyperlinkOff bool
		expectedURL          string
		expectedHyperlinkPos int
		expectedSGRCodeCount int
		expectErr            bool
	}{
		{
			name:                 "Hyperlink ON only",
			seqLine:              "9:HL:<https://example.com>",
			expectedCount:        1,
			expectedHasHyperlink: true,
			expectedURL:          "https://example.com",
			expectedHyperlinkPos: 8,
			expectedSGRCodeCount: 0,
		},
		{
			name:                 "Hyperlink OFF only",
			seqLine:              "12:Hl",
			expectedCount:        1,
			expectedHasHyperlink: false,
			expectedHyperlinkOff: true,
			expectedHyperlinkPos: 11,
			expectedSGRCodeCount: 0,
		},
		{
			name:                 "Mixed SGR and Hyperlink",
			seqLine:              "9:Fr, HL:<https://example.com>; 12:Hl",
			expectedCount:        2,
			expectedHasHyperlink: true,
			expectedURL:          "https://example.com",
			expectedHyperlinkPos: 8,
			expectedSGRCodeCount: 1,
		},
		{
			name:      "Non increasing positions",
			seqLine:   "5:HL:<https://example.com>; 3:Hl",
			expectErr: true,
		},
		{
			name:      "Duplicate positions",
			seqLine:   "5:Fr; 5:Hl",
			expectErr: true,
		},
		{
			name:      "Reject bright background",
			seqLine:   "2:BR",
			expectErr: true,
		},
		{
			name:      "Reject bright hover background",
			seqLine:   "2:HBK",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineSequencesWithHyperlinks(tt.seqLine, map[int]types.ColorValue{})
			if tt.expectErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Fatalf("Expected %d style changes, got %d", tt.expectedCount, len(result))
			}

			if tt.expectedCount > 0 {
				first := result[0]
				if first.position != tt.expectedHyperlinkPos {
					t.Errorf("Expected position %d, got %d", tt.expectedHyperlinkPos, first.position)
				}
				if tt.expectedHasHyperlink {
					if first.hyperlink == nil {
						t.Error("Expected hyperlink to be non-nil")
					} else if first.hyperlink.URL != tt.expectedURL {
						t.Errorf("Expected URL %q, got %q", tt.expectedURL, first.hyperlink.URL)
					}
				}
				if tt.expectedHyperlinkOff && !first.hasHyperlinkOff {
					t.Error("Expected hasHyperlinkOff to be true")
				}
				if len(first.codes) != tt.expectedSGRCodeCount {
					t.Errorf("Expected %d SGR codes, got %d", tt.expectedSGRCodeCount, len(first.codes))
				}
			}
		})
	}
}
