package exporter

import (
	"fmt"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// ExportFlattenedANSI exports tokens to flattened ANSI format.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop.
func ExportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, legacyMode bool, crop *types.CropRegion, keepTrailing bool) (string, int, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, legacyMode, false, crop, keepTrailing)
}

// ExportFlattenedANSIInline flattens ANSI output on a single line.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop.
func ExportFlattenedANSIInline(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, legacyMode bool, crop *types.CropRegion, keepTrailing bool) (string, int, error) {
	return exportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors, legacyMode, true, crop, keepTrailing)
}

func exportFlattenedANSI(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, legacyMode bool, inline bool, crop *types.CropRegion, keepTrailing bool) (string, int, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, useVGAColors, legacyMode)

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
	if keepTrailing {
		return vt.ExportFlattenedANSIWithTrailing(), effectiveWidth, nil
	}

	return vt.ExportFlattenedANSI(), effectiveWidth, nil
}

// ExportFlattenedANSIWithSauce exports tokens to flattened ANSI format with SAUCE metadata appended.
// If sauce is nil, behaves identically to ExportFlattenedANSI.
// When dimensions are missing in the SAUCE record, content bounds are used to fill them.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop.
func ExportFlattenedANSIWithSauce(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, legacyMode bool, crop *types.CropRegion, sauce *types.Sauce, keepTrailing bool) (string, int, error) {
	return exportFlattenedANSIWithSauce(width, nblines, tokens, outputEncoding, useVGAColors, legacyMode, false, crop, sauce, keepTrailing)
}

// ExportFlattenedANSIInlineWithSauce exports tokens to single-line ANSI format with SAUCE metadata appended.
// If sauce is nil, behaves identically to ExportFlattenedANSIInline.
// Returns (output, effectiveWidth, error) where effectiveWidth is the VT width after crop.
func ExportFlattenedANSIInlineWithSauce(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, legacyMode bool, crop *types.CropRegion, sauce *types.Sauce, keepTrailing bool) (string, int, error) {
	return exportFlattenedANSIWithSauce(width, nblines, tokens, outputEncoding, useVGAColors, legacyMode, true, crop, sauce, keepTrailing)
}

func exportFlattenedANSIWithSauce(width, nblines int, tokens []types.Token, outputEncoding string, useVGAColors bool, legacyMode bool, inline bool, crop *types.CropRegion, sauce *types.Sauce, keepTrailing bool) (string, int, error) {
	vt := processor.NewVirtualTerminal(width, nblines, outputEncoding, useVGAColors, legacyMode)

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
	} else if keepTrailing {
		ansiOutput = vt.ExportFlattenedANSIWithTrailing()
	} else {
		ansiOutput = vt.ExportFlattenedANSI()
	}

	// If no SAUCE record requested, return the ANSI output as-is
	if sauce == nil {
		return ansiOutput, effectiveWidth, nil
	}

	// Fill missing dimensions from content bounds when not provided
	if sauce.TInfo1 == 0 || sauce.TInfo2 == 0 {
		bounds := vt.GetContentBounds()
		if bounds.Empty {
			if sauce.TInfo1 == 0 {
				sauce.TInfo1 = uint16(effectiveWidth)
			}
			if sauce.TInfo2 == 0 {
				sauce.TInfo2 = 0
			}
		} else {
			if sauce.TInfo1 == 0 {
				sauce.TInfo1 = uint16(bounds.Width)
			}
			if sauce.TInfo2 == 0 {
				sauce.TInfo2 = uint16(bounds.Height)
			}
		}
	}

	// Set file size (size of ANSI content before SAUCE)
	sauce.FileSize = uint32(len(ansiOutput))

	// Append SAUCE record to output
	sauceBytes := sauce.ToBytes()
	return ansiOutput + string(sauceBytes), effectiveWidth, nil
}
