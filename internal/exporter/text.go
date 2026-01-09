package exporter

import (
	"fmt"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// ExportFlattenedText exports tokens to flattened plain text without styles
// using a virtual terminal buffer to resolve cursor positioning
func ExportFlattenedText(width, nblines int, tokens []types.Token, outputEncoding string) (string, error) {
	return exportFlattenedText(width, nblines, tokens, outputEncoding, false)
}

// ExportFlattenedTextInline exports tokens to flattened plain text on a single line.
func ExportFlattenedTextInline(width, nblines int, tokens []types.Token, outputEncoding string) (string, error) {
	return exportFlattenedText(width, nblines, tokens, outputEncoding, true)
}

func exportFlattenedText(width, nblines int, tokens []types.Token, outputEncoding string, inline bool) (string, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	if inline {
		return vt.ExportPlainTextInline(), nil
	}

	return vt.ExportPlainText(), nil
}
