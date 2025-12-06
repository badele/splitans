package exporter

import (
	"fmt"

	"splitans/processor"
	"splitans/types"
)

// ExportFlattenedText exports tokens to flattened plain text without styles
// using a virtual terminal buffer to resolve cursor positioning
func ExportFlattenedText(width int, tokens []types.Token, outputEncoding string) (string, error) {
	vt := processor.NewVirtualTerminal(width, 1000, outputEncoding, false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	return vt.ExportPlainText(), nil
}
