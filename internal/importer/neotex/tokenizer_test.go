package neotex

import (
	"fmt"
	"reflect"
	"testing"
	"time"

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
			seqLines: []string{"!W73/80"},
			expected: NeotexMetadata{
				TrimmedWidth: 73,
				Width:        80,
				Extra:        make(map[string]string),
			},
		},
		{
			name:     "Number of lines",
			seqLines: []string{"!N42/100"},
			expected: NeotexMetadata{
				NbLines: 42,
				Lines:   100,
				Extra:   make(map[string]string),
			},
		},
		{
			name:     "Multiple metadata",
			seqLines: []string{fmt.Sprintf("!V%s; !W73/80; !N42/100", exporter.NeotexVersion)},
			expected: NeotexMetadata{
				VersionRaw:   exporter.NeotexVersion,
				Version:      versionMajor,
				VersionMajor: versionMajor,
				VersionMinor: versionMinor,
				VersionPatch: versionPatch,
				TrimmedWidth: 73,
				Width:        80,
				NbLines:      42,
				Lines:        100,
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
			if meta.Lines != tt.expected.Lines {
				t.Errorf("Lines: expected %d, got %d", tt.expected.Lines, meta.Lines)
			}
		})
	}
}

func TestExtractMetadataDelaySequences(t *testing.T) {
	tests := []struct {
		name           string
		seqLines       []string
		expectExplicit bool
		expectDuration time.Duration
		expectMode     string
		expectLines    []int
		expectColumns  []int
	}{
		{
			name:           "Delay char numeric",
			seqLines:       []string{"1:DC20"},
			expectExplicit: true,
			expectDuration: 20 * time.Millisecond,
			expectMode:     NeotexDelayChar,
			expectLines:    []int{0},
			expectColumns:  []int{0},
		},
		{
			name:           "Delay line duration",
			seqLines:       []string{"1:DL120ms"},
			expectExplicit: true,
			expectDuration: 120 * time.Millisecond,
			expectMode:     NeotexDelayLine,
			expectLines:    []int{0},
			expectColumns:  []int{0},
		},
		{
			name:           "Delay disabled",
			seqLines:       []string{"1:DC0"},
			expectExplicit: true,
			expectDuration: 0,
			expectMode:     "",
			expectLines:    []int{0},
			expectColumns:  []int{0},
		},
		{
			name:           "Delay override",
			seqLines:       []string{"1:DC20", "1:DL30"},
			expectExplicit: true,
			expectDuration: 30 * time.Millisecond,
			expectMode:     NeotexDelayLine,
			expectLines:    []int{0, 1},
			expectColumns:  []int{0, 0},
		},
		{
			name:           "Delay both zero",
			seqLines:       []string{"1:DC0; 4:DL0"},
			expectExplicit: true,
			expectDuration: 0,
			expectMode:     "",
			expectLines:    []int{0, 0},
			expectColumns:  []int{0, 3},
		},
		{
			name:           "Delay not set",
			seqLines:       []string{"1:Fr"},
			expectExplicit: false,
			expectDuration: 0,
			expectMode:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := ExtractMetadata(tt.seqLines)
			if err != nil {
				t.Fatalf("ExtractMetadata failed: %v", err)
			}
			if meta.DelayExplicit != tt.expectExplicit {
				t.Fatalf("DelayExplicit: expected %v, got %v", tt.expectExplicit, meta.DelayExplicit)
			}
			if meta.DelayDuration != tt.expectDuration {
				t.Fatalf("DelayDuration: expected %v, got %v", tt.expectDuration, meta.DelayDuration)
			}
			if meta.DelayMode != tt.expectMode {
				t.Fatalf("DelayMode: expected %q, got %q", tt.expectMode, meta.DelayMode)
			}
			if tt.expectLines != nil {
				if len(meta.DelayChanges) != len(tt.expectLines) {
					t.Fatalf("DelayChanges: expected %d entries, got %d", len(tt.expectLines), len(meta.DelayChanges))
				}
				for i, expectedLine := range tt.expectLines {
					if meta.DelayChanges[i].Line != expectedLine {
						t.Fatalf("DelayChanges[%d].Line: expected %d, got %d", i, expectedLine, meta.DelayChanges[i].Line)
					}
					if meta.DelayChanges[i].Column != tt.expectColumns[i] {
						t.Fatalf("DelayChanges[%d].Column: expected %d, got %d", i, tt.expectColumns[i], meta.DelayChanges[i].Column)
					}
				}
			} else if tt.expectExplicit && len(meta.DelayChanges) == 0 {
				t.Fatalf("DelayChanges: expected entries, got none")
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

func TestExtractMetadataPaletteWithHash(t *testing.T) {
	seqLines := []string{"!P2=#FF0080; 1:Fr"}
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

func TestExtractMetadataPaletteWithHashBrackets(t *testing.T) {
	seqLines := []string{"!P2=<#FF0080>; 1:Fr"}
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

func TestTokenizerLineCountFromMetadata(t *testing.T) {
	data := []byte("Hello | !V1; !N2/4")
	_, tokenizer, err := NewNeotexTokenizer(data, 5, false)
	if err != nil {
		t.Fatalf("NewNeotexTokenizer failed: %v", err)
	}
	_ = tokenizer.Tokenize()
	if got := tokenizer.LineCount(); got != 4 {
		t.Fatalf("LineCount: expected 4, got %d", got)
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
		{
			name:      "Clear screen control",
			textLines: []string{"Hello"},
			seqLines:  []string{"1:CS"},
			contains:  []string{"Hello", "\x1b[2J"},
		},
		{
			name:      "Go home control",
			textLines: []string{"Hello"},
			seqLines:  []string{"1:GH"},
			contains:  []string{"Hello", "\x1b[H"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := ExtractMetadata(tt.seqLines)
			if err != nil {
				t.Fatalf("ExtractMetadata failed: %v", err)
			}
			result, err := ConvertNeotexToANSI(tt.textLines, tt.seqLines, meta.Palette, false)
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

func TestConvertNeotexToANSI_BrightBackgroundLegacyMode(t *testing.T) {
	textLines := []string{"X"}
	seqLines := []string{"1:BR"}
	meta, err := ExtractMetadata(seqLines)
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}

	modern, err := ConvertNeotexToANSI(textLines, seqLines, meta.Palette, false)
	if err != nil {
		t.Fatalf("ConvertNeotexToANSI modern failed: %v", err)
	}
	if !contains(string(modern), "\x1b[101m") {
		t.Fatalf("expected modern output to contain 101m, got %q", string(modern))
	}

	legacy, err := ConvertNeotexToANSI(textLines, seqLines, meta.Palette, true)
	if err != nil {
		t.Fatalf("ConvertNeotexToANSI legacy failed: %v", err)
	}
	if !contains(string(legacy), "\x1b[5;41m") {
		t.Fatalf("expected legacy output to contain 5;41m, got %q", string(legacy))
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
	_, tokenizer, err := NewNeotexTokenizer(data, 5, false)
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
	_, tokenizer, err := NewNeotexTokenizer(data, 8, false)
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
	_, tokenizer, err := NewNeotexTokenizer(data, 5, false)
	if err != nil {
		t.Fatalf("NewNeotexTokenizer failed: %v", err)
	}
	tokenizer.Tokenize()

	stats := tokenizer.GetStats()
	if stats.TotalTokens == 0 {
		t.Error("Expected TotalTokens to be set after tokenization")
	}
}

func TestParseLineSequenceOps(t *testing.T) {
	tests := []struct {
		name      string
		seqLine   string
		expected  []sequenceOp
		expectErr bool
	}{
		{
			name:     "Empty",
			seqLine:  "",
			expected: []sequenceOp{},
		},
		{
			name:     "Single style",
			seqLine:  "1:Fr",
			expected: []sequenceOp{{position: 0, code: "Fr"}},
		},
		{
			name:    "Multiple styles same position",
			seqLine: "1:Fr, EU",
			expected: []sequenceOp{
				{position: 0, code: "Fr"},
				{position: 0, code: "EU"},
			},
		},
		{
			name:    "Multiple positions",
			seqLine: "1:Fr; 5:Fg",
			expected: []sequenceOp{
				{position: 0, code: "Fr"},
				{position: 4, code: "Fg"},
			},
		},
		{
			name:    "Control codes",
			seqLine: "1:CS, GH",
			expected: []sequenceOp{
				{position: 0, code: "CS"},
				{position: 0, code: "GH"},
			},
		},
		{
			name:     "Skip metadata",
			seqLine:  fmt.Sprintf("!V%s.23; 1:Fr", exporter.NeotexVersion),
			expected: []sequenceOp{{position: 0, code: "Fr"}},
		},
		{
			name:     "Skip protected metadata",
			seqLine:  "!ST<Hello;World>; 1:Fr",
			expected: []sequenceOp{{position: 0, code: "Fr"}},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineSequenceOps(tt.seqLine, map[int]types.ColorValue{})
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
				t.Fatalf("Expected %d sequence ops, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if result[i].position != expected.position {
					t.Errorf("Position %d: expected %d, got %d", i, expected.position, result[i].position)
				}
				if result[i].code != expected.code {
					t.Errorf("Code %d: expected %q, got %q", i, expected.code, result[i].code)
				}
			}
		})
	}
}

func TestParseLineSequencesPalette(t *testing.T) {
	palette := map[int]types.ColorValue{
		2: {Type: types.ColorRGB, R: 0xFF, G: 0x00, B: 0x80},
	}
	result, err := parseLineSequenceOps("1:FP2", palette)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 style change, got %d", len(result))
	}
	if result[0].code != "FFF0080" {
		t.Fatalf("Unexpected palette conversion: %v", result[0].code)
	}

	_, err = parseLineSequenceOps("1:FP9", map[int]types.ColorValue{})
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

func TestParseLineSequenceOpsHyperlinks(t *testing.T) {
	tests := []struct {
		name      string
		seqLine   string
		expected  []sequenceOp
		expectErr bool
	}{
		{
			name:     "Hyperlink ON only",
			seqLine:  "9:HL:<https://example.com>",
			expected: []sequenceOp{{position: 8, code: "HL:<https://example.com>"}},
		},
		{
			name:    "Control codes with hyperlink",
			seqLine: "1:CS, GH, HL:<https://example.com>",
			expected: []sequenceOp{
				{position: 0, code: "CS"},
				{position: 0, code: "GH"},
				{position: 0, code: "HL:<https://example.com>"},
			},
		},
		{
			name:     "Hyperlink OFF only",
			seqLine:  "12:Hl",
			expected: []sequenceOp{{position: 11, code: "Hl"}},
		},
		{
			name:    "Mixed SGR and Hyperlink",
			seqLine: "9:Fr, HL:<https://example.com>; 12:Hl",
			expected: []sequenceOp{
				{position: 8, code: "Fr"},
				{position: 8, code: "HL:<https://example.com>"},
				{position: 11, code: "Hl"},
			},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineSequenceOps(tt.seqLine, map[int]types.ColorValue{})
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
				t.Fatalf("Expected %d sequence ops, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if result[i].position != expected.position {
					t.Errorf("Position %d: expected %d, got %d", i, expected.position, result[i].position)
				}
				if result[i].code != expected.code {
					t.Errorf("Code %d: expected %q, got %q", i, expected.code, result[i].code)
				}
			}
		})
	}
}

func TestStripSequenceComment(t *testing.T) {
	tests := []struct {
		name         string
		seqLine      string
		expectedLine string
		expectedNote string
	}{
		{
			name:         "No comment",
			seqLine:      "1:Fr, EU",
			expectedLine: "1:Fr, EU",
			expectedNote: "",
		},
		{
			name:         "Trailing comment",
			seqLine:      "1:Fr # comment",
			expectedLine: "1:Fr",
			expectedNote: "# comment",
		},
		{
			name:         "Comment without semicolon",
			seqLine:      "!V0.14.0; 1:Fr #note",
			expectedLine: "!V0.14.0; 1:Fr",
			expectedNote: "#note",
		},
		{
			name:         "Hash inside hyperlink",
			seqLine:      "1:HL:<https://example.com/#frag>; 2:Fr # end",
			expectedLine: "1:HL:<https://example.com/#frag>; 2:Fr",
			expectedNote: "# end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clean, note := stripSequenceComment(tt.seqLine)
			if clean != tt.expectedLine {
				t.Fatalf("expected cleaned %q, got %q", tt.expectedLine, clean)
			}
			if note != tt.expectedNote {
				t.Fatalf("expected comment %q, got %q", tt.expectedNote, note)
			}
		})
	}
}
