package exporter

import (
	"fmt"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

func ExportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, crop *types.CropRegion) (string, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, false, crop)
}

// ExportFlattenedANSIInline flattens ANSI output on a single line.
func ExportFlattenedANSIInline(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, crop *types.CropRegion) (string, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, true, crop)
}

func exportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, inline bool, crop *types.CropRegion) (string, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, useVGAColors)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	// Apply crop if specified
	if crop != nil {
		vt = vt.Crop(crop.X, crop.Y, crop.Width, crop.Height)
		if vt == nil {
			return "", fmt.Errorf("invalid crop region")
		}
	}

	if inline {
		return vt.ExportFlattenedANSIInline(), nil
	}

	return vt.ExportFlattenedANSI(), nil
}
