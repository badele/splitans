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

// ExportFlattenedANSIWithSauce exports tokens to flattened ANSI format with SAUCE metadata appended.
// If sauce is nil, behaves identically to ExportFlattenedANSI.
// When sauce is provided, the actual content dimensions are calculated and stored in the SAUCE record.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop.
func ExportFlattenedANSIWithSauce(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, crop *types.CropRegion, sauce *types.Sauce) (string, int, error) {
	return exportFlattenedANSIWithSauce(width, nblines, tokens, outputEncoding, useVGAColors, false, crop, sauce)
}

// ExportFlattenedANSIInlineWithSauce exports tokens to single-line ANSI format with SAUCE metadata appended.
// If sauce is nil, behaves identically to ExportFlattenedANSIInline.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop.
func ExportFlattenedANSIInlineWithSauce(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, crop *types.CropRegion, sauce *types.Sauce) (string, int, error) {
	return exportFlattenedANSIWithSauce(width, nblines, tokens, outputEncoding, useVGAColors, true, crop, sauce)
}

func exportFlattenedANSIWithSauce(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, inline bool, crop *types.CropRegion, sauce *types.Sauce) (string, int, error) {
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

	var ansiOutput string
	if inline {
		ansiOutput = vt.ExportFlattenedANSIInline()
	} else {
		ansiOutput = vt.ExportFlattenedANSI()
	}

	// If no SAUCE record requested, return the ANSI output as-is
	if sauce == nil {
		return ansiOutput, effectiveWidth, nil
	}

	// Get content bounds to determine actual dimensions
	bounds := vt.GetContentBounds()
	if !bounds.Empty {
		sauce.SetDimensions(bounds.Width, bounds.Height)
	} else {
		sauce.SetDimensions(effectiveWidth, 0)
	}

	// Set file size (size of ANSI content before SAUCE)
	sauce.FileSize = uint32(len(ansiOutput))

	// Append SAUCE record to output
	sauceBytes := sauce.ToBytes()
	return ansiOutput + string(sauceBytes), effectiveWidth, nil
}
