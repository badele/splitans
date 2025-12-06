package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"splitans/processor"
	"splitans/types"
)

var sgrToNeotex = map[string]string{
	// Foreground colors (normal)
	"30": "Fk", "31": "Fr", "32": "Fg", "33": "Fy",
	"34": "Fb", "35": "Fm", "36": "Fc", "37": "Fw",
	// Foreground colors (bright)
	"90": "FK", "91": "FR", "92": "FG", "93": "FY",
	"94": "FB", "95": "FM", "96": "FC", "97": "FW",
	"39": "FD", // Foreground Default

	// Background colors (normal)
	"40": "Bk", "41": "Br", "42": "Bg", "43": "By",
	"44": "Bb", "45": "Bm", "46": "Bc", "47": "Bw",
	// Background colors (bright)
	"100": "BK", "101": "BR", "102": "BG", "103": "BY",
	"104": "BB", "105": "BM", "106": "BC", "107": "BW",
	"49": "BD", // Background Default

	// Effects (uppercase = ON, lowercase = OFF)
	"2": "EM", "22": "Em", // Dim
	"3": "EI", "23": "Ei", // Italic
	"4": "EU", "24": "Eu", // Underline
	"5": "EB", "25": "Eb", // Blink
	"7": "ER", "27": "Er", // Reverse

	// Special
	"0": "R0", // Reset
}

// SGRToNeotex converts an types.SGR struct to neotex format strings
func SGRToNeotex(sgr *types.SGR) []string {
	codes := []string{}

	// Handle reset
	if sgr.Equals(types.NewSGR()) {
		return []string{"R0"}
	}

	// Foreground color
	if !sgr.FgColor.IsDefault() {
		switch sgr.FgColor.Type {

		case types.ColorStandard:
			{
				colorIndex := sgr.FgColor.Index
				// In neotex, bold is handled by color case (uppercase = bright)
				// So if bold is true and color < 8, we use the bright version
				if sgr.Bold && colorIndex < 8 {
					colorIndex += 8
				}

				var code string
				if colorIndex < 8 {
					code = fmt.Sprintf("%d", 30+colorIndex)
				} else {
					code = fmt.Sprintf("%d", 82+colorIndex)
				}

				if neotex, ok := sgrToNeotex[code]; ok {
					codes = append(codes, neotex)
				}
			}

		case types.ColorRGB:
			{
				// RGB format: FRRGGBB (F + 6 hex digits)
				neotexCode := fmt.Sprintf("F%02X%02X%02X", sgr.FgColor.R, sgr.FgColor.G, sgr.FgColor.B)
				codes = append(codes, neotexCode)
			}

		case types.ColorIndexed:
			{
				// Indexed format: Fxxx (F + 1-3 digits for index 0-255)
				neotexCode := fmt.Sprintf("F%d", sgr.FgColor.Index)
				codes = append(codes, neotexCode)
			}
		}
	}

	// Background color
	if !sgr.BgColor.IsDefault() {
		switch sgr.BgColor.Type {

		case types.ColorStandard:
			{
				colorIndex := sgr.BgColor.Index
				var code string
				if colorIndex < 8 {
					code = fmt.Sprintf("%d", 40+colorIndex)
				} else {
					code = fmt.Sprintf("%d", 92+colorIndex)
				}

				if neotex, ok := sgrToNeotex[code]; ok {
					codes = append(codes, neotex)
				}
			}

		case types.ColorRGB:
			{
				// RGB format: BRRGGBB (B + 6 hex digits)
				neotexCode := fmt.Sprintf("B%02X%02X%02X", sgr.BgColor.R, sgr.BgColor.G, sgr.BgColor.B)
				codes = append(codes, neotexCode)
			}

		case types.ColorIndexed:
			{
				// Indexed format: Bxxx (B + 1-3 digits for index 0-255)
				neotexCode := fmt.Sprintf("B%d", sgr.BgColor.Index)
				codes = append(codes, neotexCode)
			}
		}
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

// ExportToNeotex exports processor.VirtualTerminal buffer to neotex format
// Returns (text, sequences) where:
// - text is the plain text content
// - sequences is the neotex format sequences with positions (per line)
func ExportToNeotex(vt *processor.VirtualTerminal) (string, string) {
	lines := vt.ExportSplitTextAndSequences()

	var textBuilder strings.Builder
	var seqBuilder strings.Builder

	for lineIdx, line := range lines {
		// Add text
		textBuilder.WriteString(line.Text)
		if lineIdx < len(lines)-1 {
			textBuilder.WriteString("\n")
		}

		// Add sequences for this line (positions are relative to the line)
		var lineSeqs []string
		for _, sgrChange := range line.Sequences {
			// Convert types.SGR to neotex codes
			neotexCodes := SGRToNeotex(sgrChange.SGR)
			if len(neotexCodes) > 0 {
				// Use position relative to the current line
				seqStr := fmt.Sprintf("%d:%s", sgrChange.Position, strings.Join(neotexCodes, ", "))
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

func ExportFlattenedNeotex(width int, tokens []types.Token, outputEncoding string) (string, string, error) {
	vt := processor.NewVirtualTerminal(width, 1000, outputEncoding, false)

	if err := vt.ApplyTokens(tokens); err != nil {
		return "", "", fmt.Errorf("error applying tokens: %w", err)
	}

	text, sequences := ExportToNeotex(vt)
	return text, sequences, nil
}

func ExportToNeotexFile(basePath string, plainText string, plainSequence string) error {
	basePath = strings.TrimSuffix(basePath, filepath.Ext(basePath))

	neotPath := basePath + ".neot"
	neosPath := basePath + ".neos"

	textFile, err := os.Create(neotPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .neot: %w", err)
	}
	defer textFile.Close()

	sequenceFile, err := os.Create(neosPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .neos: %w", err)
	}
	defer sequenceFile.Close()

	_, err = textFile.WriteString(plainText)
	if err != nil {
		return fmt.Errorf("erreur écriture dans .neot: %w", err)
	}

	_, err = sequenceFile.WriteString(plainSequence)
	if err != nil {
		return fmt.Errorf("erreur écriture dans .neos: %w", err)
	}

	return nil
}

// - .neot : plain text content
// - .neos : plain sequence content
// - .neop : plain neotex packed (text + sequence)
// - .neoi : project information
func ExportToNeopackedFile(basePath string, plainText string, plainSequence string) error {
	basePath = strings.TrimSuffix(basePath, filepath.Ext(basePath))

	neotPath := basePath + ".neot"
	neosPath := basePath + ".neos"

	textFile, err := os.Create(neotPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .neot: %w", err)
	}
	defer textFile.Close()

	sequenceFile, err := os.Create(neosPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .neos: %w", err)
	}
	defer sequenceFile.Close()

	_, err = textFile.WriteString(plainText)
	if err != nil {
		return fmt.Errorf("erreur écriture dans .neot: %w", err)
	}

	_, err = sequenceFile.WriteString(plainSequence)
	if err != nil {
		return fmt.Errorf("erreur écriture dans .neos: %w", err)
	}

	return nil
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
