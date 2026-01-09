package exporter

import (
	"fmt"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

func ExportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool) (string, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, false)
}

// ExportFlattenedANSIInline flattens ANSI output on a single line.
func ExportFlattenedANSIInline(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool) (string, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, true)
}

func exportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, inline bool) (string, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, useVGAColors)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	if inline {
		return vt.ExportFlattenedANSIInline(), nil
	}

	return vt.ExportFlattenedANSI(), nil
}
