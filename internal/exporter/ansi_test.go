package exporter

import (
	"strings"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestExportFlattenedANSIInline(t *testing.T) {
	tokens := []types.Token{
		{Type: types.TokenSGR, Parameters: []string{"31"}},
		{Type: types.TokenText, Value: "AB"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "CD"},
	}

	standard, err := ExportFlattenedANSI(2, 2, tokens, "utf8", false)
	if err != nil {
		t.Fatalf("unexpected standard export error: %v", err)
	}

	inline, err := ExportFlattenedANSIInline(2, 2, tokens, "utf8", false)
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
