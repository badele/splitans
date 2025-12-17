package exporter

import (
	"strings"

	"splitans/types"
)

// ExportPassthroughANSI reconstructs ANSI output directly from tokens
func ExportPassthroughANSI(tokens []types.Token) (string, error) {
	var result strings.Builder

	for _, token := range tokens {
		switch token.Type {
		case types.TokenText:
			result.WriteString(token.Value)

		case types.TokenSGR, types.TokenCSI, types.TokenC0, types.TokenC1,
			types.TokenEscape, types.TokenDCS, types.TokenOSC:
			// Reconstruit la séquence originale telle quelle
			result.WriteString(token.Raw)

		case types.TokenUnknown, types.TokenCSIInterupted:
			// Garde tel quel (pour debug/compatibilité)
			result.WriteString(token.Raw)
		}
	}

	return result.String(), nil
}
