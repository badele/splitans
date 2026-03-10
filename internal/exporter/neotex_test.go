package exporter

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/badele/splitans/internal/importer/ansi"
	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

func TestHyperlinkToNeotex(t *testing.T) {
	tests := []struct {
		name     string
		input    *types.Hyperlink
		expected string
	}{
		{
			name:     "Nil hyperlink",
			input:    nil,
			expected: "Hl",
		},
		{
			name:     "Empty URL",
			input:    &types.Hyperlink{URL: ""},
			expected: "Hl",
		},
		{
			name:     "Simple URL",
			input:    types.NewHyperlink("https://example.com"),
			expected: "HL:<https://example.com>",
		},
		{
			name:     "Complex URL",
			input:    types.NewHyperlink("https://example.com/path?query=1&foo=bar"),
			expected: "HL:<https://example.com/path?query=1&foo=bar>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HyperlinkToNeotex(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExportToNeotexWithHyperlinks(t *testing.T) {
	vt := processor.NewVirtualTerminal(20, 1, "utf8", false, false)

	// Create tokens with a hyperlink
	tokens := []types.Token{
		{Type: types.TokenText, Value: "Click "},
		{Type: types.TokenOSC, Hyperlink: types.NewHyperlink("https://example.com")},
		{Type: types.TokenText, Value: "here"},
		{Type: types.TokenOSC, Hyperlink: &types.Hyperlink{URL: ""}},
		{Type: types.TokenText, Value: " now"},
	}

	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	_, sequences, err := ExportToNeotex(vt)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	// Should contain hyperlink codes
	if !strings.Contains(sequences, "HL:<https://example.com>") {
		t.Errorf("Expected sequences to contain HL:<https://example.com>, got %q", sequences)
	}
	if !strings.Contains(sequences, "Hl") {
		t.Errorf("Expected sequences to contain Hl, got %q", sequences)
	}
}

func TestHyperlinkRoundTrip(t *testing.T) {
	// Create ANSI input with hyperlinks
	ansiInput := "Click \x1b]8;;https://example.com\x1b\\here\x1b]8;;\x1b\\ to continue"

	// Parse the ANSI input
	tokenizer := ansi.NewANSITokenizer([]byte(ansiInput))
	tokens := tokenizer.Tokenize()

	// Apply to virtual terminal
	vt := processor.NewVirtualTerminal(20, 1, "utf8", false, false)
	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("Failed to apply tokens: %v", err)
	}

	// Export to neotext
	text, sequences, err := ExportToNeotex(vt)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	// Verify neotext output contains hyperlink codes
	if !strings.Contains(sequences, "HL:<https://example.com>") {
		t.Errorf("Expected sequences to contain HL:<https://example.com>, got %q", sequences)
	}
	if !strings.Contains(sequences, "Hl") {
		t.Errorf("Expected sequences to contain Hl, got %q", sequences)
	}

	// Verify text contains the expected content
	if !strings.Contains(text, "Click ") || !strings.Contains(text, "here") {
		t.Errorf("Expected text to contain 'Click ' and 'here', got %q", text)
	}

	// Export back to ANSI
	ansiOutput := vt.ExportFlattenedANSI()

	// Verify ANSI output contains OSC 8 sequences
	if !strings.Contains(ansiOutput, "\x1b]8;;https://example.com\x1b\\") {
		t.Errorf("Expected ANSI output to contain hyperlink ON sequence")
	}
	if !strings.Contains(ansiOutput, "\x1b]8;;\x1b\\") {
		t.Errorf("Expected ANSI output to contain hyperlink OFF sequence")
	}
}

func TestExportToInlineNeotex(t *testing.T) {
	vt := processor.NewVirtualTerminal(4, 4, "utf8", false, false)

	tokens := []types.Token{
		{Type: types.TokenSGR, Parameters: []string{"31"}},
		{Type: types.TokenText, Value: "ABCD"},
		{Type: types.TokenSGR, Parameters: []string{"32"}},
		{Type: types.TokenText, Value: "EF"},
	}

	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	text, sequences, err := ExportToInlineNeotex(vt)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	if text != "ABCDEF  " {
		t.Fatalf("unexpected inline text: got %q", text)
	}

	expectedSequences := fmt.Sprintf("!V%s; !W6/8; !N1/1; 1:Fr, Bk; 5:Fg; 7:R0", NeotexVersion)
	if sequences != expectedSequences {
		t.Fatalf("unexpected inline sequences: got %q, want %q", sequences, expectedSequences)
	}
}

func TestExportFlattenedNeotexPreservesTrailingLines(t *testing.T) {
	tokens := []types.Token{
		{Type: types.TokenText, Value: "A"},
	}

	text, sequences, _, err := ExportFlattenedNeotex(2, 3, tokens, nil, true)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	if got := len(strings.Split(text, "\n")); got != 3 {
		t.Fatalf("expected 3 lines in text, got %d", got)
	}
	if !strings.Contains(sequences, "!N1/3") {
		t.Fatalf("expected sequences to include !N1/3, got %q", sequences)
	}
}

func TestExportToNeotexPalettizesRGB(t *testing.T) {
	vt := processor.NewVirtualTerminal(6, 1, "utf8", false, false)

	tokens := []types.Token{
		{Type: types.TokenSGR, Parameters: []string{"38", "2", "255", "0", "128"}},
		{Type: types.TokenText, Value: "A"},
		{Type: types.TokenSGR, Parameters: []string{"48", "2", "0", "255", "0"}},
		{Type: types.TokenText, Value: "B"},
		{Type: types.TokenSGR, Parameters: []string{"38", "2", "255", "0", "128"}},
		{Type: types.TokenText, Value: "C"},
	}

	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	_, sequences, err := ExportToNeotex(vt)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	if !strings.Contains(sequences, "!P1=#FF0080") {
		t.Fatalf("expected palette entry for first color, got %q", sequences)
	}
	if !strings.Contains(sequences, "!P2=#00FF00") {
		t.Fatalf("expected palette entry for second color, got %q", sequences)
	}
	if !strings.Contains(sequences, "FP1") {
		t.Fatalf("expected foreground palette code, got %q", sequences)
	}
	if !strings.Contains(sequences, "BP2") {
		t.Fatalf("expected background palette code, got %q", sequences)
	}

	paletteIdx := strings.Index(sequences, "!P1=")
	posIdx := strings.Index(sequences, "1:")
	if paletteIdx == -1 || posIdx == -1 || paletteIdx > posIdx {
		t.Fatalf("expected palette metadata before position entries, got %q", sequences)
	}

	rawRGB := regexp.MustCompile(`(?:^|[,:;\s])([FB][0-9A-Fa-f]{6})`)
	if rawRGB.MatchString(sequences) {
		t.Fatalf("expected RGB codes to be palettized, got %q", sequences)
	}
}

func TestPaletizeNeotexSequencesRejectsPaletteZero(t *testing.T) {
	sequences := "!P0=#FF00FF; 1:FP1"
	_, err := paletizeNeotexSequences(sequences)
	if err == nil {
		t.Fatalf("expected palette index 0 error")
	}
}

func TestFormatNeotexLabel(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
		wantErr  bool
	}{
		{name: "short form", key: "ST", value: "Title", expected: "!STTitle"},
		{name: "protected with space", key: "ST", value: "Hello World", expected: "!ST<Hello World>"},
		{name: "protected with punctuation", key: "SA", value: "Doe;Smith", expected: "!SA<Doe;Smith>"},
		{name: "forbidden angle bracket", key: "ST", value: "Bad<Title", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatNeotexLabel(tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Fatalf("unexpected label: got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSauceToNeotexLabelsProtectedAndRejected(t *testing.T) {
	sauce := &types.Sauce{Title: "Hello World", Author: "Jane"}
	labels, err := sauceToNeotexLabels(sauce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(labels, ";")
	if !strings.Contains(joined, "!ST<Hello World>") {
		t.Fatalf("expected protected title label, got %q", joined)
	}

	badSauce := &types.Sauce{Title: "Bad<Title"}
	if _, err := sauceToNeotexLabels(badSauce); err == nil {
		t.Fatalf("expected error for forbidden angle bracket")
	}
}

func TestSauceToNeotexLabelsDefaultDate(t *testing.T) {
	sauce := &types.Sauce{}
	now := time.Now()
	labels, err := sauceToNeotexLabels(sauce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	joined := strings.Join(labels, ";")
	if !strings.Contains(joined, "!SD"+now.Format("20060102")) &&
		!strings.Contains(joined, "!SD"+after.Format("20060102")) {
		t.Fatalf("expected default date label, got %q", joined)
	}
}
