package neotex

import (
	"bytes"
	"strconv"
	"strings"

	"splitans/types"
)

// NeopackMetadata contains metadata extracted from neopack format
type NeopackMetadata struct {
	Version      int               // Format version (!V1 = 1)
	TrimmedWidth int               // Trimmed width (!TW73/80 -> 73)
	Width        int               // Total width (!TW73/80 -> 80)
	NbLines      int               // Number of lines with content (!NL<n>)
	Extra        map[string]string // Other metadata (!key:value)
}

// ExtractMetadata extracts metadata from sequence lines
// Metadata entries start with '!' (e.g., !V1 for version)
func ExtractMetadata(seqLines []string) NeopackMetadata {
	meta := NeopackMetadata{
		Version: 0, // 0 means no version found (legacy format)
		Extra:   make(map[string]string),
	}

	for _, seqLine := range seqLines {
		entries := strings.Split(seqLine, ";")
		for _, entry := range entries {
			entry = strings.TrimSpace(entry)
			if !strings.HasPrefix(entry, "!") {
				continue
			}

			// Remove '!' prefix
			entry = entry[1:]

			// Check for version: V<number>
			if strings.HasPrefix(entry, "V") {
				if v, err := strconv.Atoi(entry[1:]); err == nil {
					meta.Version = v
				}
				continue
			}
			
			// Check trimmed width TW<trimmed>/<total> or TW<number>
			if strings.HasPrefix(entry, "TW") {
				twValue := entry[2:]
				if parts := strings.Split(twValue, "/"); len(parts) == 2 {
					// Format: TW73/80
					if v, err := strconv.Atoi(parts[0]); err == nil {
						meta.TrimmedWidth = v
					}
					if v, err := strconv.Atoi(parts[1]); err == nil {
						meta.Width = v
					}
				} 
				continue
			}

			// Check number of lines NL<number>
			if strings.HasPrefix(entry, "NL") {
				if v, err := strconv.Atoi(entry[2:]); err == nil {
					meta.NbLines = v
				}
				continue
			}

			// Other metadata: key:value
			if parts := strings.SplitN(entry, ":", 2); len(parts) == 2 {
				meta.Extra[parts[0]] = parts[1]
			}
		}
	}

	return meta
}

// ConvertNeotexToANSI converts neotex format (text + sequences) to raw ANSI format
// This allows reusing the existing ANSI tokenizer instead of duplicating parsing logic
// Tracks SGR state across lines for proper differential encoding
// Takes arrays of lines (without embedded \n) for cleaner processing
func ConvertNeotexToANSI(textLines []string, seqLines []string) []byte {
	var result bytes.Buffer
	currentSGR := types.NewSGR() // Track SGR state across lines

	for i, textLine := range textLines {
		var seqLine string
		if i < len(seqLines) {
			seqLine = seqLines[i]
		}

		ansiLine, newSGR := convertLineToANSI(textLine, seqLine, currentSGR)
		currentSGR = newSGR

		result.WriteString(ansiLine)

		// Add newline if not last line
		// if i < len(textLines)-1 {
		// 	result.WriteString("\n")
		// }
	}

	return result.Bytes()
}

// styleChange represents a style change at a specific position
type styleChange struct {
	position int
	codes    []string
}

// convertLineToANSI converts a single line of text with its sequences to ANSI
// Takes the current SGR state and returns the updated state after processing
func convertLineToANSI(textLine string, seqLine string, currentSGR *types.SGR) (string, *types.SGR) {
	if seqLine == "" {
		return textLine, currentSGR
	}

	styles := parseLineSequences(seqLine)
	if len(styles) == 0 {
		return textLine, currentSGR
	}

	// Build ANSI output by inserting escape sequences at the right positions
	var result bytes.Buffer
	textRunes := []rune(textLine)
	textPos := 0

	for _, style := range styles {
		// Write text before this position
		if style.position > textPos && style.position <= len(textRunes) {
			result.WriteString(string(textRunes[textPos:style.position]))
		}

		// Apply neotex codes to current SGR
		newSGR := currentSGR.Copy()
		for _, code := range style.codes {
			ApplyNeotexCode(code, newSGR)
		}

		// Generate differential ANSI sequence
		ansiSeq := newSGR.DiffToANSI(currentSGR, false, true)
		result.WriteString(ansiSeq)

		currentSGR = newSGR
		textPos = style.position
	}

	// Write remaining text
	if textPos < len(textRunes) {
		result.WriteString(string(textRunes[textPos:]))
	}

	return result.String(), currentSGR
}

// parseLineSequences parses sequences for a single line
// Returns a slice of styleChange in the order they appear (already sorted)
// Metadata entries starting with '!' are ignored (e.g., !V1 for version)
func parseLineSequences(seqLine string) []styleChange {
	var styles []styleChange
	if seqLine == "" {
		return styles
	}

	// Split by semicolons to get position entries
	entries := strings.Split(seqLine, ";")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Skip metadata entries (start with '!')
		if strings.HasPrefix(entry, "!") {
			continue
		}

		// Split position from styles: "14:Fr, EU"
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}

		position, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		// Convert 1-indexed (editor format) to 0-indexed (internal)
		position--

		// Parse styles separated by commas
		stylesStr := strings.TrimSpace(parts[1])
		styleList := strings.Split(stylesStr, ",")

		codes := make([]string, 0)
		for _, style := range styleList {
			style = strings.TrimSpace(style)
			if style != "" {
				codes = append(codes, style)
			}
		}

		if len(codes) > 0 {
			styles = append(styles, styleChange{
				position: position,
				codes:    codes,
			})
		}
	}

	return styles
}
