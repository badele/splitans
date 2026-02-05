package exporter

import (
	"fmt"
	"strings"
	"testing"

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
	vt := processor.NewVirtualTerminal(20, 1, "utf8", false)

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

	_, sequences := ExportToNeotex(vt)

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
	vt := processor.NewVirtualTerminal(20, 1, "utf8", false)
	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("Failed to apply tokens: %v", err)
	}

	// Export to neotext
	text, sequences := ExportToNeotex(vt)

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
	vt := processor.NewVirtualTerminal(4, 4, "utf8", false)

	tokens := []types.Token{
		{Type: types.TokenSGR, Parameters: []string{"31"}},
		{Type: types.TokenText, Value: "ABCD"},
		{Type: types.TokenSGR, Parameters: []string{"32"}},
		{Type: types.TokenText, Value: "EF"},
	}

	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	text, sequences := ExportToInlineNeotex(vt)

	if text != "ABCDEF  " {
		t.Fatalf("unexpected inline text: got %q", text)
	}

	expectedSequences := fmt.Sprintf("!V%s; !TW6/8; !NL1; 1:Fr, Bk; 5:Fg; 7:R0", NeotexVersion)
	if sequences != expectedSequences {
		t.Fatalf("unexpected inline sequences: got %q, want %q", sequences, expectedSequences)
	}
}
