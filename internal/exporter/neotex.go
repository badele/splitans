package exporter

import (
	"fmt"
	"strings"

	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// NeotexVersion is the current version of the neotex format
const NeotexVersion = 1

// Neotex color codes indexed by ColorValue.Index (0-15)
// Index 0-7: normal colors (lowercase), Index 8-15: bright colors (uppercase)
var neotexFgColors = []string{
	"Fk", "Fr", "Fg", "Fy", "Fb", "Fm", "Fc", "Fw", // 0-7: normal
	"FK", "FR", "FG", "FY", "FB", "FM", "FC", "FW", // 8-15: bright
}

var neotexBgColors = []string{
	"Bk", "Br", "Bg", "By", "Bb", "Bm", "Bc", "Bw", // 0-7: normal
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

	return codes
}

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
		// Add back all active attributes from current state
		fullCodes := SGRToNeotex(current)
		for _, c := range fullCodes {
			if c != "R0" {
				codes = append(codes, c)
			}
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

	return codes
}

func flattenLinesWithSequences(lines []types.LineWithSequences) []types.LineWithSequences {
	if len(lines) <= 1 {
		return lines
	}

	totalSeqs := 0
	for _, line := range lines {
		totalSeqs += len(line.Sequences)
	}

	var textBuilder strings.Builder
	flattenedSeqs := make([]types.SGRSequence, 0, totalSeqs)

	offset := 0
	for _, line := range lines {
		textBuilder.WriteString(line.Text)

		for _, seq := range line.Sequences {
			flattenedSeqs = append(flattenedSeqs, types.SGRSequence{
				Position: seq.Position + offset,
				SGR:      seq.SGR.Copy(),
			})
		}

		offset += len([]rune(line.Text))
	}

	return []types.LineWithSequences{{
		Text:      textBuilder.String(),
		Sequences: flattenedSeqs,
	}}
}

// ExportToNeotex exports processor.VirtualTerminal buffer to neotex format with differential encoding.
// Returns (text, sequences) where:
// - text is the plain text content
// - sequences is the neotex format sequences with positions (per line)
// Uses differential encoding to minimize the number of codes by only outputting changes.
func ExportToNeotex(vt *processor.VirtualTerminal) (string, string) {
	return exportToNeotex(vt, false)
}

// ExportToInlineNeotex exports the buffer to neotex format, flattening all lines into one.
func ExportToInlineNeotex(vt *processor.VirtualTerminal) (string, string) {
	return exportToNeotex(vt, true)
}

func exportToNeotex(vt *processor.VirtualTerminal, inline bool) (string, string) {
	lines := vt.ExportSplitTextAndSequences()

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

	textWidth := vt.GetWidth()
	maxWidth := vt.GetMaxCursorX() + 1
	lineCount := len(lines)

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
	}

	for lineIdx, line := range lines {
		// Add text
		textBuilder.WriteString(line.Text)
		if lineIdx < len(lines)-1 {
			textBuilder.WriteString("\n")
		}

		// Add sequences for this line (positions are relative to the line)
		var lineSeqs []string

		// Add version metadata on the first line
		if lineIdx == 0 {
			lineSeqs = append(lineSeqs, fmt.Sprintf("!V%d", NeotexVersion))
			lineSeqs = append(lineSeqs, fmt.Sprintf("!TW%d/%d", maxWidth, textWidth))
			lineSeqs = append(lineSeqs, fmt.Sprintf("!NL%d", lineCount))
		}

		for _, sgrChange := range line.Sequences {
			// Generate differential neotex codes
			neotexCodes := DiffSGRToNeotex(sgrChange.SGR, previousSGR)
			if len(neotexCodes) > 0 {
				// if firstLine {
				// 	neotexCodes = append([]string{"R0"}, neotexCodes...)
				// 	firstLine = false
				// }
				// Use position relative to the current line (1-indexed for editor compatibility)
				seqStr := fmt.Sprintf("%d:%s", sgrChange.Position+1, strings.Join(neotexCodes, ", "))
				lineSeqs = append(lineSeqs, seqStr)
			}
			// Update previous state
			previousSGR = sgrChange.SGR.Copy()
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

// ExportFlattenedNeotex exports tokens to neotex format (always UTF-8)
func ExportFlattenedNeotex(width, nblines int, tokens []types.Token) (string, string, error) {
	return exportFlattenedNeotex(width, nblines, tokens, false)
}

// ExportFlattenedNeotexInline exports tokens to inline neotex format (always UTF-8)
func ExportFlattenedNeotexInline(width, nblines int, tokens []types.Token) (string, string, error) {
	return exportFlattenedNeotex(width, nblines, tokens, true)
}

func exportFlattenedNeotex(width, nblines int, tokens []types.Token, inline bool) (string, string, error) {
	vt := processor.NewVirtualTerminal(width, nblines, "utf8", false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", "", fmt.Errorf("error applying tokens: %w", err)
	}

	var text, sequences string
	if inline {
		text, sequences = ExportToInlineNeotex(vt)
	} else {
		text, sequences = ExportToNeotex(vt)
	}

	return text, sequences, nil
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
