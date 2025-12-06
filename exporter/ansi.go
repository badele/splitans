package exporter

import (
	"fmt"

	"splitans/processor"
	"splitans/types"
)

func ExportFlattenedANSI(width int, tokens []types.Token, outputEncoding string, useVGAColors bool) (string, error) {
	vt := processor.NewVirtualTerminal(width, 1000, outputEncoding, useVGAColors)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	return vt.ExportFlattenedANSI(), nil
}
