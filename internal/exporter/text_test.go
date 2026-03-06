package exporter

import (
	"strings"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestExportFlattenedTextInline(t *testing.T) {
	tokens := []types.Token{
		{Type: types.TokenText, Value: "AB"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "CD"},
	}

	standard, _, err := ExportFlattenedText(2, 2, tokens, "utf8", nil, false)
	if err != nil {
		t.Fatalf("unexpected standard export error: %v", err)
	}

	inline, _, err := ExportFlattenedTextInline(2, 2, tokens, "utf8", nil, false)
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

func TestExportFlattenedTextPreservesTrailingLines(t *testing.T) {
	tokens := []types.Token{
		{Type: types.TokenText, Value: "A"},
	}

	output, _, err := ExportFlattenedText(2, 3, tokens, "utf8", nil, true)
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	if got := strings.Count(output, "\n"); got != 3 {
		t.Fatalf("expected 3 newlines for trailing lines, got %d", got)
	}
}
