package exporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// NeotexVersion is the current version of the neotex format
const NeotexVersion = "1.0.0" // x-release-please-version

// Neotex color codes indexed by ColorValue.Index (0-15)
// Index 0-7: normal colors (lowercase), Index 8-15: bright colors (uppercase)
var neotexFgColors = []string{
	"Fk", "Fr", "Fg", "Fy", "Fb", "Fm", "Fc", "Fw", // 0-7: normal
	"FK", "FR", "FG", "FY", "FB", "FM", "FC", "FW", // 8-15: bright
}

var neotexBgColors = []string{
	"Bk", "Br", "Bg", "By", "Bb", "Bm", "Bc", "Bw", // 0-7: normal
}

const neotexForbiddenLabelChars = " ;,:<>"

// ============================================================================
// EXPORTED
// ============================================================================

// HyperlinkToNeotex converts a Hyperlink to neotex format.
// Returns "HL:<url>" for hyperlink ON or "Hl" for hyperlink OFF.
func HyperlinkToNeotex(h *types.Hyperlink) string {
	if h == nil || h.URL == "" {
		return "Hl" // Hyperlink OFF
	}
	return "HL:<" + h.URL + ">" // Hyperlink ON with URL
}

// SGRToNeotex converts an types.SGR struct to neotex format strings
func SGRToNeotex(sgr *types.SGR) []string {
	codes := []string{}

	// Foreground color
	switch sgr.FgColor.Type {
	case types.ColorStandard:
		colorIndex := sgr.FgColor.Index
		// In neotex, bold is handled by color case (uppercase = bright)
		// So if bold is true and color < 8, we use the bright version
		if sgr.Bold && colorIndex < 8 {
			colorIndex += 8
		}
		if int(colorIndex) < len(neotexFgColors) {
			codes = append(codes, neotexFgColors[colorIndex])
		}

	case types.ColorRGB:
		// RGB format: FRRGGBB (F + 6 hex digits)
		codes = append(codes, fmt.Sprintf("F%02X%02X%02X", sgr.FgColor.R, sgr.FgColor.G, sgr.FgColor.B))

	case types.ColorIndexed:
		// Indexed format: Fxxx (F + 1-3 digits for index 0-255)
		codes = append(codes, fmt.Sprintf("F%d", sgr.FgColor.Index))
	}

	// Background color
	switch sgr.BgColor.Type {
	case types.ColorStandard:
		colorIndex := sgr.BgColor.Index
		if int(colorIndex) < len(neotexBgColors) {
			codes = append(codes, neotexBgColors[colorIndex])
		}

	case types.ColorRGB:
		// RGB format: BRRGGBB (B + 6 hex digits)
		codes = append(codes, fmt.Sprintf("B%02X%02X%02X", sgr.BgColor.R, sgr.BgColor.G, sgr.BgColor.B))

	case types.ColorIndexed:
		// Indexed format: Bxxx (B + 1-3 digits for index 0-255)
		codes = append(codes, fmt.Sprintf("B%d", sgr.BgColor.Index))
	}

	// Effects (excluding Bold, which is in color brightness)
	if sgr.Dim {
		codes = append(codes, "EM")
	}
	if sgr.Italic {
		codes = append(codes, "EI")
	}
	if sgr.Underline {
		codes = append(codes, "EU")
	}
	if sgr.Blink {
		codes = append(codes, "EB")
	}
	if sgr.Reverse {
		codes = append(codes, "ER")
	}

	// Hover link colors
	if !sgr.LinkFgColor.IsDefault() {
		codes = append(codes, linkHoverFgToNeotex(sgr)...)
	}
	if !sgr.LinkBgColor.IsDefault() {
		codes = append(codes, linkHoverBgToNeotex(sgr)...)
	}

	return codes
}

// DiffSGRToNeotex generates minimal neotex codes to transition from previous to current SGR state
func DiffSGRToNeotex(current, previous *types.SGR) []string {
	// If previous is nil, return full state
	if previous == nil {
		return SGRToNeotex(current)
	}

	// If equal, no codes needed
	if current.Equals(previous) {
		return nil
	}

	// If current is default state, return reset
	if current.Equals(types.NewSGR()) {
		return []string{"R0"}
	}

	// Check if we need a reset (attribute turned off or bright->normal transition)
	needsReset := false
	if previous.FgColor.Type == types.ColorStandard && current.FgColor.Type == types.ColorStandard {
		if previous.FgColor.Index >= 8 && current.FgColor.Index < 8 {
			needsReset = true
		}
	}
	// Previous was bright FG, current is different type or normal
	if previous.FgColor.Type == types.ColorStandard && previous.FgColor.Index >= 8 {
		if current.FgColor.Type != types.ColorStandard || current.FgColor.Index < 8 {
			needsReset = true
		}
	}
	// Check for bright->normal BG color transition
	if previous.BgColor.Type == types.ColorStandard && current.BgColor.Type == types.ColorStandard {
		if previous.BgColor.Index >= 8 && current.BgColor.Index < 8 {
			needsReset = true
		}
	}
	if previous.BgColor.Type == types.ColorStandard && previous.BgColor.Index >= 8 {
		if current.BgColor.Type != types.ColorStandard || current.BgColor.Index < 8 {
			needsReset = true
		}
	}
	// Check for attribute turned off
	if previous.Dim && !current.Dim {
		needsReset = true
	}
	if previous.Italic && !current.Italic {
		needsReset = true
	}
	if previous.Underline && !current.Underline {
		needsReset = true
	}
	if previous.Blink && !current.Blink {
		needsReset = true
	}
	if previous.Reverse && !current.Reverse {
		needsReset = true
	}

	// If reset needed, return R0 + full current state
	if needsReset {
		codes := []string{"R0"}

		// Rebuild state without hover codes (to avoid re-émission si inchangés)
		// Foreground
		if !current.FgColor.IsDefault() {
			codes = append(codes, fgColorToNeotex(current)...)
		}
		// Background
		if !current.BgColor.IsDefault() {
			codes = append(codes, bgColorToNeotex(current)...)
		}
		// Effects
		if current.Dim {
			codes = append(codes, "EM")
		}
		if current.Italic {
			codes = append(codes, "EI")
		}
		if current.Underline {
			codes = append(codes, "EU")
		}
		if current.Blink {
			codes = append(codes, "EB")
		}
		if current.Reverse {
			codes = append(codes, "ER")
		}

		// Hover colors seulement si elles ont changé
		if current.LinkFgColor != previous.LinkFgColor {
			codes = append(codes, linkHoverFgToNeotex(current)...)
		}
		if current.LinkBgColor != previous.LinkBgColor {
			codes = append(codes, linkHoverBgToNeotex(current)...)
		}

		return codes
	}

	var codes []string

	// Handle effects with ON codes only (OFF cases handled by reset above)
	if current.Dim && !previous.Dim {
		codes = append(codes, "EM")
	}

	if current.Italic && !previous.Italic {
		codes = append(codes, "EI")
	}

	if current.Underline && !previous.Underline {
		codes = append(codes, "EU")
	}

	if current.Blink && !previous.Blink {
		codes = append(codes, "EB")
	}

	if current.Reverse && !previous.Reverse {
		codes = append(codes, "ER")
	}

	// Handle foreground color (including bold which affects brightness)
	// We need to check both FgColor and Bold changes since Bold affects color brightness
	fgChanged := current.FgColor != previous.FgColor
	boldChanged := current.Bold != previous.Bold
	if fgChanged || (boldChanged && current.FgColor.Type == types.ColorStandard) {
		codes = append(codes, fgColorToNeotex(current)...)
	}

	// Handle background color
	if current.BgColor != previous.BgColor {
		codes = append(codes, bgColorToNeotex(current)...)
	}

	// Handle link hover colors (HF/HB) - not reset by R0
	if current.LinkFgColor != previous.LinkFgColor {
		codes = append(codes, linkHoverFgToNeotex(current)...)
	}
	if current.LinkBgColor != previous.LinkBgColor {
		codes = append(codes, linkHoverBgToNeotex(current)...)
	}

	return codes
}

// ExportToNeotex exports processor.VirtualTerminal buffer to neotex format with differential encoding.
// Returns (text, sequences) where:
// - text is the plain text content
// - sequences is the neotex format sequences with positions (per line)
// Uses differential encoding to minimize the number of codes by only outputting changes.
func ExportToNeotex(vt *processor.VirtualTerminal) (string, string) {
	return exportToNeotex(vt, false, false)
}

// ExportToInlineNeotex exports the buffer to neotex format, flattening all lines into one.
func ExportToInlineNeotex(vt *processor.VirtualTerminal) (string, string) {
	return exportToNeotex(vt, true, false)
}

// ExportFlattenedNeotex exports tokens to neotex format (always UTF-8)
// Returns (text, sequences, effectiveWidth, error) where effectiveWidth is the VT width after crop
func ExportFlattenedNeotex(width, nblines int, tokens []types.Token, crop *types.CropRegion, keepTrailing bool) (string, string, int, error) {
	return exportFlattenedNeotex(width, nblines, tokens, false, crop, keepTrailing)
}

// ExportFlattenedNeotexInline exports tokens to inline neotex format (always UTF-8)
// Returns (text, sequences, effectiveWidth, error) where effectiveWidth is the VT width after crop
func ExportFlattenedNeotexInline(width, nblines int, tokens []types.Token, crop *types.CropRegion, keepTrailing bool) (string, string, int, error) {
	return exportFlattenedNeotex(width, nblines, tokens, true, crop, keepTrailing)
}

// ExportFlattenedNeotexWithSauce exports tokens to neotex format with SAUCE metadata on the last line.
// Returns (text, sequences, effectiveWidth, error) where effectiveWidth is the VT width after crop.
// If sauce is nil, behaves identically to ExportFlattenedNeotex.
func ExportFlattenedNeotexWithSauce(width, nblines int, tokens []types.Token, crop *types.CropRegion, sauce *types.Sauce, keepTrailing bool) (string, string, int, error) {
	return exportFlattenedNeotexWithSauce(width, nblines, tokens, false, crop, sauce, keepTrailing)
}

// ExportFlattenedNeotexInlineWithSauce exports tokens to inline neotex format with SAUCE metadata.
// Returns (text, sequences, effectiveWidth, error) where effectiveWidth is the VT width after crop.
// If sauce is nil, behaves identically to ExportFlattenedNeotexInline.
func ExportFlattenedNeotexInlineWithSauce(width, nblines int, tokens []types.Token, crop *types.CropRegion, sauce *types.Sauce, keepTrailing bool) (string, string, int, error) {
	return exportFlattenedNeotexWithSauce(width, nblines, tokens, true, crop, sauce, keepTrailing)
}

// formatNeotexLabel chooses between the short form (!KEYvalue) and protected form
// (!KEY<value>) depending on the content. Forbidden characters for the short form
// are space, ';', ',', ':', '<', '>'. Angle brackets are rejected outright.
func formatNeotexLabel(key, value string) (string, error) {
	if strings.ContainsAny(value, "<>") {
		return "", fmt.Errorf("neotex label %s contains forbidden angle bracket", key)
	}

	if strings.ContainsAny(value, neotexForbiddenLabelChars) {
		return fmt.Sprintf("!%s<%s>", key, value), nil
	}

	return fmt.Sprintf("!%s%s", key, value), nil
}

// sauceToNeotexLabels converts SAUCE metadata to neotex label format.
// All SAUCE labels start with "!S" prefix, so no separate marker is needed.
// Note: Width and Height are not exported here as they use !W and !N metadata.
func sauceToNeotexLabels(sauce *types.Sauce) ([]string, error) {
	if sauce == nil {
		return nil, nil
	}

	var labels []string

	if sauce.Title != "" {
		label, err := formatNeotexLabel("ST", sauce.Title)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	if sauce.Author != "" {
		label, err := formatNeotexLabel("SA", sauce.Author)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	if sauce.Group != "" {
		label, err := formatNeotexLabel("SG", sauce.Group)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	date := sauce.Date
	if date.IsZero() {
		date = time.Now()
	}
	label, err := formatNeotexLabel("SD", date.Format("20060102"))
	if err != nil {
		return nil, err
	}
	labels = append(labels, label)
	if sauce.TInfoS != "" {
		label, err := formatNeotexLabel("SF", sauce.TInfoS)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	if sauce.HasICEColors() {
		labels = append(labels, "!SI")
	}

	return labels, nil
}

// ============================================================================
// PRIVATE
// ============================================================================

// fgColorToNeotex generates neotex code for foreground color
func fgColorToNeotex(sgr *types.SGR) []string {
	switch sgr.FgColor.Type {
	case types.ColorStandard:
		colorIndex := sgr.FgColor.Index
		if sgr.Bold && colorIndex < 8 {
			colorIndex += 8
		}
		if int(colorIndex) < len(neotexFgColors) {
			return []string{neotexFgColors[colorIndex]}
		}

	case types.ColorRGB:
		return []string{fmt.Sprintf("F%02X%02X%02X", sgr.FgColor.R, sgr.FgColor.G, sgr.FgColor.B)}

	case types.ColorIndexed:
		return []string{fmt.Sprintf("F%d", sgr.FgColor.Index)}
	}

	return nil
}

// bgColorToNeotex generates neotex code for background color
func bgColorToNeotex(sgr *types.SGR) []string {
	switch sgr.BgColor.Type {
	case types.ColorStandard:
		colorIndex := sgr.BgColor.Index
		if int(colorIndex) < len(neotexBgColors) {
			return []string{neotexBgColors[colorIndex]}
		}

	case types.ColorRGB:
		return []string{fmt.Sprintf("B%02X%02X%02X", sgr.BgColor.R, sgr.BgColor.G, sgr.BgColor.B)}

	case types.ColorIndexed:
		return []string{fmt.Sprintf("B%d", sgr.BgColor.Index)}
	}

	return nil
}

// linkHoverFgToNeotex generates neotex code for link hover foreground color
func linkHoverFgToNeotex(sgr *types.SGR) []string {
	switch sgr.LinkFgColor.Type {
	case types.ColorStandard:
		colorIndex := sgr.LinkFgColor.Index
		if int(colorIndex) < len(neotexFgColors) {
			return []string{"HF" + neotexFgColors[colorIndex][1:]}
		}
	case types.ColorRGB:
		return []string{fmt.Sprintf("HF%02X%02X%02X", sgr.LinkFgColor.R, sgr.LinkFgColor.G, sgr.LinkFgColor.B)}
	case types.ColorIndexed:
		return []string{fmt.Sprintf("HF%d", sgr.LinkFgColor.Index)}
	}
	return nil
}

// linkHoverBgToNeotex generates neotex code for link hover background color
func linkHoverBgToNeotex(sgr *types.SGR) []string {
	switch sgr.LinkBgColor.Type {
	case types.ColorStandard:
		colorIndex := sgr.LinkBgColor.Index
		if int(colorIndex) < len(neotexBgColors) {
			return []string{"HB" + neotexBgColors[colorIndex][1:]}
		}
	case types.ColorRGB:
		return []string{fmt.Sprintf("HB%02X%02X%02X", sgr.LinkBgColor.R, sgr.LinkBgColor.G, sgr.LinkBgColor.B)}
	case types.ColorIndexed:
		return []string{fmt.Sprintf("HB%d", sgr.LinkBgColor.Index)}
	}
	return nil
}

func flattenLinesWithSequences(lines []types.LineWithSequences) []types.LineWithSequences {
	if len(lines) <= 1 {
		return lines
	}

	totalSeqs := 0
	totalHyperlinkSeqs := 0
	for _, line := range lines {
		totalSeqs += len(line.Sequences)
		totalHyperlinkSeqs += len(line.HyperlinkSequences)
	}

	var textBuilder strings.Builder
	flattenedSeqs := make([]types.SGRSequence, 0, totalSeqs)
	flattenedHyperlinkSeqs := make([]types.HyperlinkSequence, 0, totalHyperlinkSeqs)

	offset := 0
	for _, line := range lines {
		textBuilder.WriteString(line.Text)

		for _, seq := range line.Sequences {
			flattenedSeqs = append(flattenedSeqs, types.SGRSequence{
				Position: seq.Position + offset,
				SGR:      seq.SGR.Copy(),
			})
		}

		for _, seq := range line.HyperlinkSequences {
			var hyperlinkCopy *types.Hyperlink
			if seq.Hyperlink != nil {
				hyperlinkCopy = seq.Hyperlink.Copy()
			}
			flattenedHyperlinkSeqs = append(flattenedHyperlinkSeqs, types.HyperlinkSequence{
				Position:  seq.Position + offset,
				Hyperlink: hyperlinkCopy,
			})
		}

		offset += len([]rune(line.Text))
	}

	return []types.LineWithSequences{{
		Text:               textBuilder.String(),
		Sequences:          flattenedSeqs,
		HyperlinkSequences: flattenedHyperlinkSeqs,
	}}
}

func exportToNeotex(vt *processor.VirtualTerminal, inline bool, keepTrailing bool) (string, string) {
	lines := vt.ExportSplitTextAndSequences(keepTrailing && !inline)
	buffer := vt.GetBuffer()

	if inline {
		lines = flattenLinesWithSequences(lines)
	}

	if len(lines) == 0 {
		return "", ""
	}

	var textBuilder strings.Builder
	var seqBuilder strings.Builder

	// Track previous SGR state across all lines for differential encoding
	var previousSGR *types.SGR = nil
	var previousHyperlink *types.Hyperlink = nil

	textWidth := vt.GetWidth()
	maxWidth := vt.GetMaxCursorX() + 1
	lineCount := vt.GetMaxCursorY() + 1
	totalLines := vt.GetHeight()

	if inline {
		textRunes := []rune(lines[0].Text)
		textWidth = len(textRunes)
		// Calculate true width by finding last non-space character
		maxWidth = textWidth
		for i := len(textRunes) - 1; i >= 0; i-- {
			if textRunes[i] != ' ' {
				maxWidth = i + 1
				break
			}
		}
		lineCount = 1
		totalLines = lineCount
	}

	defaultSGR := types.NewSGR()
	for lineIdx, line := range lines {
		rowMaxX := -1
		if inline {
			textRunes := []rune(line.Text)
			rowMaxX = len(textRunes) - 1
		} else if lineIdx < len(buffer) {
			for x := 0; x < len(buffer[lineIdx]); x++ {
				if buffer[lineIdx][x].Char != 0x0 {
					rowMaxX = x
				}
			}
		}
		if rowMaxX < 0 {
			rowMaxX = textWidth - 1
		}

		linePreviousSGR := previousSGR
		if linePreviousSGR != nil {
			linePreviousSGR = linePreviousSGR.Copy()
		}

		// Add text
		textBuilder.WriteString(line.Text)
		if lineIdx < len(lines)-1 {
			textBuilder.WriteString("\n")
		}

		// Add sequences for this line (positions are relative to the line)
		var lineSeqs []string

		// Add version metadata on the first line
		if lineIdx == 0 {
			lineSeqs = append(lineSeqs, fmt.Sprintf("!V%s", NeotexVersion))
			lineSeqs = append(lineSeqs, fmt.Sprintf("!W%d/%d", maxWidth, textWidth))
			lineSeqs = append(lineSeqs, fmt.Sprintf("!N%d/%d", lineCount, totalLines))
		}

		// Merge SGR and hyperlink sequences by position
		sgrIdx := 0
		hyperlinkIdx := 0

		// Process all positions that have either SGR or hyperlink changes
		for sgrIdx < len(line.Sequences) || hyperlinkIdx < len(line.HyperlinkSequences) {
			var sgrPos, hyperlinkPos int = -1, -1
			if sgrIdx < len(line.Sequences) {
				sgrPos = line.Sequences[sgrIdx].Position
			}
			if hyperlinkIdx < len(line.HyperlinkSequences) {
				hyperlinkPos = line.HyperlinkSequences[hyperlinkIdx].Position
			}

			// Determine which to process first (or both if same position)
			var codes []string
			var pos int

			if sgrPos >= 0 && (hyperlinkPos < 0 || sgrPos <= hyperlinkPos) {
				pos = sgrPos
				allowTrailingReset := pos == rowMaxX+1 && line.Sequences[sgrIdx].SGR.Equals(defaultSGR)
				if pos > rowMaxX && !allowTrailingReset {
					sgrIdx++
					if hyperlinkPos == pos {
						hyperlinkIdx++
					}
					continue
				}
				// Generate differential neotex codes for SGR
				neotexCodes := DiffSGRToNeotex(line.Sequences[sgrIdx].SGR, linePreviousSGR)
				codes = append(codes, neotexCodes...)
				linePreviousSGR = line.Sequences[sgrIdx].SGR.Copy()
				if pos <= rowMaxX || allowTrailingReset {
					previousSGR = line.Sequences[sgrIdx].SGR.Copy()
				}
				sgrIdx++

				// Check if there's also a hyperlink change at the same position
				if hyperlinkIdx < len(line.HyperlinkSequences) && line.HyperlinkSequences[hyperlinkIdx].Position == pos {
					newHyperlink := line.HyperlinkSequences[hyperlinkIdx].Hyperlink
					if !newHyperlink.Equals(previousHyperlink) {
						codes = append(codes, HyperlinkToNeotex(newHyperlink))
						if newHyperlink != nil {
							previousHyperlink = newHyperlink.Copy()
						} else {
							previousHyperlink = nil
						}
					}
					hyperlinkIdx++
				}
			} else if hyperlinkPos >= 0 {
				pos = hyperlinkPos
				allowTrailingReset := pos == rowMaxX+1 && line.HyperlinkSequences[hyperlinkIdx].Hyperlink == nil
				if pos > rowMaxX && !allowTrailingReset {
					hyperlinkIdx++
					continue
				}
				newHyperlink := line.HyperlinkSequences[hyperlinkIdx].Hyperlink
				if !newHyperlink.Equals(previousHyperlink) {
					codes = append(codes, HyperlinkToNeotex(newHyperlink))
					if newHyperlink != nil {
						previousHyperlink = newHyperlink.Copy()
					} else {
						previousHyperlink = nil
					}
				}
				hyperlinkIdx++
			} else {
				break
			}

			if len(codes) > 0 {
				// Use position relative to the current line (1-indexed for editor compatibility)
				seqStr := fmt.Sprintf("%d:%s", pos+1, strings.Join(codes, ", "))
				lineSeqs = append(lineSeqs, seqStr)
			}
		}

		// Add line sequences to builder
		if len(lineSeqs) > 0 {
			seqBuilder.WriteString(strings.Join(lineSeqs, "; "))
		}

		// Add newline if not last line
		if lineIdx < len(lines)-1 {
			seqBuilder.WriteString("\n")
		}
	}

	return textBuilder.String(), seqBuilder.String()
}

func exportFlattenedNeotex(width, nblines int, tokens []types.Token, inline bool, crop *types.CropRegion, keepTrailing bool) (string, string, int, error) {
	vt := processor.NewVirtualTerminal(width, nblines, "utf8", false, false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", "", 0, fmt.Errorf("error applying tokens: %w", err)
	}

	// Apply crop if specified
	if crop != nil {
		vt = vt.Crop(crop.X, crop.Y, crop.Width, crop.Height)
		if vt == nil {
			return "", "", 0, fmt.Errorf("invalid crop region")
		}
	}

	effectiveWidth := vt.GetWidth()

	var text, sequences string
	if inline {
		text, sequences = exportToNeotex(vt, true, false)
	} else {
		text, sequences = exportToNeotex(vt, false, keepTrailing)
	}

	return text, sequences, effectiveWidth, nil
}

func exportFlattenedNeotexWithSauce(width, nblines int, tokens []types.Token, inline bool, crop *types.CropRegion, sauce *types.Sauce, keepTrailing bool) (string, string, int, error) {
	// If no SAUCE, delegate to the regular export function
	if sauce == nil {
		return exportFlattenedNeotex(width, nblines, tokens, inline, crop, keepTrailing)
	}

	vt := processor.NewVirtualTerminal(width, nblines, "utf8", false, false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", "", 0, fmt.Errorf("error applying tokens: %w", err)
	}

	// Apply crop if specified
	if crop != nil {
		vt = vt.Crop(crop.X, crop.Y, crop.Width, crop.Height)
		if vt == nil {
			return "", "", 0, fmt.Errorf("invalid crop region")
		}
	}

	effectiveWidth := vt.GetWidth()

	var text, sequences string
	if inline {
		text, sequences = exportToNeotex(vt, true, false)
	} else {
		text, sequences = exportToNeotex(vt, false, keepTrailing)
	}

	// Append SAUCE line: empty text line + SAUCE labels
	sauceLabels, err := sauceToNeotexLabels(sauce)
	if err != nil {
		return "", "", 0, err
	}
	if len(sauceLabels) > 0 {
		// Add a new line with empty text (spaces to match width) and SAUCE labels
		text += "\n" + strings.Repeat(" ", effectiveWidth)
		sequences += "\n" + strings.Join(sauceLabels, "; ")
	}

	return text, sequences, effectiveWidth, nil
}

func getTokenTypeName(tokenType types.TokenType) string {
	switch tokenType {
	case types.TokenText:
		return "TEXT"
	case types.TokenC0:
		return "C0"
	case types.TokenC1:
		return "C1"
	case types.TokenCSI:
		return "CSI"
	case types.TokenCSIInterupted:
		return "CSI_INTERRUPTED"
	case types.TokenSGR:
		return "types.SGR"
	case types.TokenDCS:
		return "DCS"
	case types.TokenOSC:
		return "OSC"
	case types.TokenEscape:
		return "ESCAPE"
	case types.TokenUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}
