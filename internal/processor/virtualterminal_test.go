package processor

import (
	"strings"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestWriteTextDoesNotInsertExtraBlankLineOnExactWidth(t *testing.T) {
	vt := NewVirtualTerminal(3, 10, "utf8", false)

	tokens := []types.Token{
		{Type: types.TokenText, Value: "abc"},
		{Type: types.TokenC0, C0Code: 0x0A}, // LF
		{Type: types.TokenText, Value: "def"},
	}

	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	lines := vt.ExportSplitTextAndSequences()
	if got := len(lines); got != 2 {
		t.Fatalf("expected 2 lines, got %d", got)
	}

	if strings.TrimRight(lines[0].Text, " ") != "abc" {
		t.Fatalf("expected first line 'abc', got %q", lines[0].Text)
	}

	if strings.TrimRight(lines[1].Text, " ") != "def" {
		t.Fatalf("expected second line 'def', got %q", lines[1].Text)
	}
}
