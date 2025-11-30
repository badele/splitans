package exporter

import (
	"fmt"
	"io"
	"strings"

	"splitans/tokenizer"
)

func ExportTokensToTable(tokens []tokenizer.Token, writer io.Writer) error {
	fmt.Fprintln(writer, "\n┌─────────┬────────┬──────────────────────────────────────┬──────────────────────────────────────┬─────────────────┬──────────────────────────────────────┐")
	fmt.Fprintf(writer, "│ %-7s │ %-6s │ %-36s │ %-36s │ %-15s │ %-36s │\n", "Token", "Pos", "CSISignification", "Signification", "Paramètres", "Raw/Texte")
	fmt.Fprintln(writer, "├─────────┼────────┼──────────────────────────────────────┼──────────────────────────────────────┼─────────────────┼──────────────────────────────────────┤")

	for i, token := range tokens {
		var csiSignification, signification, params, rawOrText string

		switch token.Type {
		case tokenizer.TokenText:
			csiSignification = "-"
			signification = "TEXT"
			params = "-"
			rawOrText = truncate(token.Value, 36)

		case tokenizer.TokenSGR:
			meanings := tokenizer.ParseSGRParams(token.Parameters)
			csiSignification = truncate(token.CSINotation, 36)
			signification = truncate(strings.Join(meanings, ", "), 36)
			params = truncate(fmt.Sprintf("%v", token.Parameters), 15)
			rawOrText = truncate(token.Raw, 36)

		case tokenizer.TokenCSI:
			csiSignification = truncate(token.CSINotation, 36)
			signification = truncate(token.Signification, 36)
			params = truncate(fmt.Sprintf("%v", token.Parameters), 15)
			rawOrText = truncate(token.Raw, 36)

		case tokenizer.TokenOSC:
			csiSignification = "-"
			signification = truncate(token.Signification, 36)
			params = truncate(fmt.Sprintf("%v", token.Parameters), 15)
			rawOrText = truncate(token.Raw, 36)

		case tokenizer.TokenDCS:
			csiSignification = "-"
			signification = "DCS"
			params = "-"
			rawOrText = truncate(token.Raw, 36)

		case tokenizer.TokenC0:
			csiSignification = "-"
			if name, ok := tokenizer.C0Names[token.C0Code]; ok {
				signification = name
			} else {
				signification = "C0: unknown"
			}
			params = fmt.Sprintf("0x%02X", token.C0Code)
			rawOrText = truncate(token.Raw, 36)

		case tokenizer.TokenC1:
			csiSignification = "-"
			signification = fmt.Sprintf("C1: %s", token.C1Code)
			params = "-"
			rawOrText = truncate(token.Raw, 36)

		case tokenizer.TokenCSIInterupted:
			csiSignification = truncate(token.CSINotation, 36)
			signification = "CSI INTERRUPTED"
			params = truncate(fmt.Sprintf("%v", token.Parameters), 15)
			rawOrText = truncate(token.Raw, 36)

		default:
			csiSignification = "-"
			signification = "UNKNOWN"
			if token.CSINotation != "" {
				csiSignification = truncate(token.CSINotation, 36)
				signification = "-"
			}
			params = "-"
			rawOrText = truncate(token.Raw, 36)
		}

		fmt.Fprintf(writer, "│ %-7d │ %-6d │ %-36s │ %-36s │ %-15s │ %-36s │\n",
			i+1, token.Pos, csiSignification, signification, params, rawOrText)
	}

	fmt.Fprintln(writer, "└─────────┴────────┴──────────────────────────────────────┴──────────────────────────────────────┴─────────────────┴──────────────────────────────────────┘")

	return nil
}

func truncate(s string, maxLen int) string {
	s = fmt.Sprintf("%q", s)

	// Remove quote added by %q
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
