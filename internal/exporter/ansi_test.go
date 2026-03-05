package exporter

import (
	"strings"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestExportANSI_NoHyperlinksForCP437(t *testing.T) {
	// Create tokens with hyperlink
	tokens := []types.Token{
		{Type: types.TokenText, Value: "Click "},
		{Type: types.TokenOSC, Hyperlink: &types.Hyperlink{URL: "https://example.com"}},
		{Type: types.TokenText, Value: "here"},
		{Type: types.TokenOSC, Hyperlink: &types.Hyperlink{URL: ""}}, // Close hyperlink
		{Type: types.TokenText, Value: " to continue"},
	}

	// Test UTF-8 export: should contain OSC 8 hyperlink sequences
	utf8Output, _, err := ExportFlattenedANSI(80, 1, tokens, "utf8", false, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected utf8 export error: %v", err)
	}
	if !strings.Contains(utf8Output, "\x1b]8;") {
		t.Errorf("UTF-8 output should contain OSC 8 hyperlink sequences, got %q", utf8Output)
	}
	if !strings.Contains(utf8Output, "https://example.com") {
		t.Errorf("UTF-8 output should contain the hyperlink URL, got %q", utf8Output)
	}

	// Test CP437 export: should NOT contain OSC 8 hyperlink sequences
	cp437Output, _, err := ExportFlattenedANSI(80, 1, tokens, "cp437", false, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected cp437 export error: %v", err)
	}
	if strings.Contains(cp437Output, "\x1b]8;") {
		t.Errorf("CP437 output should NOT contain OSC 8 hyperlink sequences, got %q", cp437Output)
	}
	if strings.Contains(cp437Output, "https://example.com") {
		t.Errorf("CP437 output should NOT contain the hyperlink URL, got %q", cp437Output)
	}

	// Verify the text content is still present in both outputs
	if !strings.Contains(utf8Output, "Click ") || !strings.Contains(utf8Output, "here") {
		t.Errorf("UTF-8 output missing expected text content")
	}
	if !strings.Contains(cp437Output, "Click ") || !strings.Contains(cp437Output, "here") {
		t.Errorf("CP437 output missing expected text content")
	}
}

func TestExportFlattenedANSIInline(t *testing.T) {
	tokens := []types.Token{
		{Type: types.TokenSGR, Parameters: []string{"31"}},
		{Type: types.TokenText, Value: "AB"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "CD"},
	}

	standard, _, err := ExportFlattenedANSI(2, 2, tokens, "utf8", false, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected standard export error: %v", err)
	}

	inline, _, err := ExportFlattenedANSIInline(2, 2, tokens, "utf8", false, false, nil, false)
	if err != nil {
		t.Fatalf("unexpected inline export error: %v", err)
	}

	if strings.Contains(inline, "\n") {
		t.Fatalf("inline output should not contain newline, got %q", inline)
	}

	if strings.ReplaceAll(standard, "\n", "") != inline {
		t.Fatalf("inline output should equal standard output without newlines")
	}
}

func TestExportFlattenedANSIPreservesTrailingLines(t *testing.T) {
	tokens := []types.Token{
		{Type: types.TokenText, Value: "A"},
	}

	output, _, err := ExportFlattenedANSI(2, 3, tokens, "utf8", false, false, nil, true)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	if got := strings.Count(output, "\n"); got != 3 {
		t.Fatalf("expected 3 newlines for trailing lines, got %d", got)
	}
}
