package exporter

import (
	"fmt"

	"splitans/processor"
	"splitans/types"
)

// ExportFlattenedText exports tokens to flattened plain text without styles
// using a virtual terminal buffer to resolve cursor positioning
func ExportFlattenedText(width, nblines int, tokens []types.Token, outputEncoding string) (string, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	return vt.ExportPlainText(), nil
}
