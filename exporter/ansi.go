package exporter

import (
	"fmt"

	"github.com/badele/splitans/processor"
	"github.com/badele/splitans/types"
)

func ExportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool) (string, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, useVGAColors)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	return vt.ExportFlattenedANSI(), nil
}
