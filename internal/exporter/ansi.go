package exporter

import (
	"fmt"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// ExportFlattenedANSI exports tokens to flattened ANSI format
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop
func ExportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, crop *types.CropRegion) (string, int, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, false, crop)
}

// ExportFlattenedANSIInline flattens ANSI output on a single line.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop
func ExportFlattenedANSIInline(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, crop *types.CropRegion) (string, int, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, true, crop)
}

func exportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, inline bool, crop *types.CropRegion) (string, int, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, useVGAColors)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", 0, fmt.Errorf("error applying tokens: %w", err)
	}

	// Apply crop if specified
	if crop != nil {
		vt = vt.Crop(crop.X, crop.Y, crop.Width, crop.Height)
		if vt == nil {
			return "", 0, fmt.Errorf("invalid crop region")
		}
	}

	effectiveWidth := vt.GetWidth()

	if inline {
		return vt.ExportFlattenedANSIInline(), effectiveWidth, nil
	}

	return vt.ExportFlattenedANSI(), effectiveWidth, nil
}
