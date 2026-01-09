package exporter

import (
	"testing"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

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

	expectedSequences := "!V1; !TW8/8; !NL1; 1:Fr, Bk; 5:Fg; 7:R0"
	if sequences != expectedSequences {
		t.Fatalf("unexpected inline sequences: got %q, want %q", sequences, expectedSequences)
	}
}
