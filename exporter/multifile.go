package exporter

import (
	"encoding/json"
	"fmt"
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

// ExportToMultifile exporte les tokens vers deux fichiers :
// - .ant : contient le texte pur
// - .anc : contient les métadonnées (couleurs, curseur, etc.) au format JSON
func ExportToMultifile(tokens []tokenizer.Token, basePath string) error {
	// Remove ext if exists
	basePath = strings.TrimSuffix(basePath, filepath.Ext(basePath))

	antPath := basePath + ".ant"
	ancPath := basePath + ".anc"

	textFile, err := os.Create(antPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .ant: %w", err)
	}
	defer textFile.Close()

	metadata := MetadataFile{
		Version:  "1.0",
		TextFile: filepath.Base(antPath),
		Tokens:   make([]MetadataToken, 0),
	}

	textPos := 0
	for _, token := range tokens {
		switch token.Type {
		case tokenizer.TokenText:
			n, err := textFile.WriteString(token.Value)
			if err != nil {
				return fmt.Errorf("erreur écriture dans .ant: %w", err)
			}
			textPos += n

		default:
			metaToken := MetadataToken{
				Type:          getTokenTypeName(token.Type),
				Pos:           token.Pos,
				TextPos:       textPos,
				Raw:           token.Raw,
				Parameters:    token.Parameters,
				CSINotation:   token.CSINotation,
				Signification: token.Signification,
			}

			switch token.Type {
			case tokenizer.TokenC0:
				code := token.C0Code
				metaToken.C0Code = &code
				if name, ok := tokenizer.C0Names[token.C0Code]; ok {
					metaToken.C0Name = name
				}

			case tokenizer.TokenC1:
				metaToken.C1Code = token.C1Code

			case tokenizer.TokenSGR:
				metaToken.SGRMeaning = tokenizer.ParseSGRParams(token.Parameters)
			}

			metadata.Tokens = append(metadata.Tokens, metaToken)
		}
	}

	metaFile, err := os.Create(ancPath)
	if err != nil {
		return fmt.Errorf("erreur création fichier .anc: %w", err)
	}
	defer metaFile.Close()

	encoder := json.NewEncoder(metaFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("erreur écriture métadonnées JSON: %w", err)
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
