package exporter

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"os"
	"path/filepath"
	"strings"

	"splitans/tokenizer"
)

type MetadataToken struct {
	Type          string   `json:"type"`
	Pos           int      `json:"pos"`
	TextPos       int      `json:"text_pos"`
	Raw           string   `json:"raw,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
	C0Code        *byte    `json:"c0_code,omitempty"`
	C0Name        string   `json:"c0_name,omitempty"`
	C1Code        string   `json:"c1_code,omitempty"`
	CSINotation   string   `json:"csi_notation,omitempty"`
	Signification string   `json:"signification,omitempty"`
	SGRMeaning    []string `json:"sgr_meaning,omitempty"`
}

type MetadataFile struct {
	Version  string          `json:"version"`
	TextFile string          `json:"text_file"`
	Tokens   []MetadataToken `json:"tokens"`
}

func GetPlainText(tokens []tokenizer.Token) (string, error) {
	buffer, err := NewTcellBufferWithEncoding(80, 1000, true)
	if err != nil {
		return "", fmt.Errorf("error creating buffer: %w", err)
	}
	defer buffer.Close()

	if err := buffer.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	return buffer.GetPlainText(), nil
}

func GetPlainTextSequence(tokens []tokenizer.Token) (string, error) {
	outputSequences := []string{}

	buffer, err := NewTcellBufferWithEncoding(80, 1000, true)
	if err != nil {
		return "", fmt.Errorf("error creating buffer: %w", err)
	}
	defer buffer.Close()

	if err := buffer.ApplyTokens(tokens); err != nil {
		return "", fmt.Errorf("error applying tokens: %w", err)
	}

	// Track the global style state across all lines
	globalStyle := tcell.StyleDefault

	// Analyze each line for style changes
	for y := 0; y < buffer.height; y++ {
		var sequences []string
		var lineHasContent bool

		// Check if line has any content
		for x := 0; x < buffer.width; x++ {
			mainc, _, _, _ := buffer.screen.GetContent(x, y)
			if mainc != 0 && mainc != ' ' {
				lineHasContent = true
				break
			}
		}

		if !lineHasContent {
			continue
		}

		// Scan the line for style changes on non-empty characters only
		for x := 0; x < buffer.width; x++ {
			mainc, _, style, _ := buffer.screen.GetContent(x, y)

			// Skip empty cells completely - we only care about style changes on visible characters
			if mainc == 0 || mainc == ' ' {
				continue
			}

			// Detect style change only on visible characters
			if style != globalStyle {
				diff := styleDiff(globalStyle, style)
				if diff != "" {
					sequences = append(sequences, fmt.Sprintf("%d:%s", x, diff))
				}
				globalStyle = style
			}
		}

		outputSequences = append(outputSequences, strings.Join(sequences, "; "))
	}

	return strings.Join(outputSequences, "\n"), nil
}

// - .ant : plain text content
// - .anc : plain sequence content
func ExportToMultipleFile(basePath string, plainText string, plainSequence string) error {
	basePath = strings.TrimSuffix(basePath, filepath.Ext(basePath))

	antPath := basePath + ".ant"
	ancPath := basePath + ".anc"

	textFile, err := os.Create(antPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .ant: %w", err)
	}
	defer textFile.Close()

	sequenceFile, err := os.Create(ancPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .anc: %w", err)
	}
	defer sequenceFile.Close()

	_, err = textFile.WriteString(plainText)
	if err != nil {
		return fmt.Errorf("erreur écriture dans .ant: %w", err)
	}

	_, err = sequenceFile.WriteString(plainSequence)
	if err != nil {
		return fmt.Errorf("erreur écriture dans .anc: %w", err)
	}

	return nil
}

func getTokenTypeName(tokenType tokenizer.TokenType) string {
	switch tokenType {
	case tokenizer.TokenText:
		return "TEXT"
	case tokenizer.TokenC0:
		return "C0"
	case tokenizer.TokenC1:
		return "C1"
	case tokenizer.TokenCSI:
		return "CSI"
	case tokenizer.TokenCSIInterupted:
		return "CSI_INTERRUPTED"
	case tokenizer.TokenSGR:
		return "SGR"
	case tokenizer.TokenDCS:
		return "DCS"
	case tokenizer.TokenOSC:
		return "OSC"
	case tokenizer.TokenEscape:
		return "ESCAPE"
	case tokenizer.TokenUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}
