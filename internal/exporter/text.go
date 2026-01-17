package exporter

import (
	"fmt"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// ExportFlattenedText exports tokens to flattened plain text without styles
// using a virtual terminal buffer to resolve cursor positioning
// Returns (text, effectiveWidth, error) where effectiveWidth is the VT width after crop
func ExportFlattenedText(width, nblines int, tokens []types.Token, outputEncoding string, crop *types.CropRegion) (string, int, error) {
	return exportFlattenedText(width, nblines, tokens, outputEncoding, false, crop)
}

// ExportFlattenedTextInline exports tokens to flattened plain text on a single line.
// Returns (text, effectiveWidth, error) where effectiveWidth is the VT width after crop
func ExportFlattenedTextInline(width, nblines int, tokens []types.Token, outputEncoding string, crop *types.CropRegion) (string, int, error) {
	return exportFlattenedText(width, nblines, tokens, outputEncoding, true, crop)
}

func exportFlattenedText(width, nblines int, tokens []types.Token, outputEncoding string, inline bool, crop *types.CropRegion) (string, int, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, false)

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
		return vt.ExportPlainTextInline(), effectiveWidth, nil
	}

	return vt.ExportPlainText(), effectiveWidth, nil
}
