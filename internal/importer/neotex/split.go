package neotex

import "strings"

// splitNeotexEntries splits a sequence line on semicolons, ignoring separators
// inside angle brackets (protected metadata values).
func splitNeotexEntries(seqLine string) []string {
	if seqLine == "" {
		return nil
	}

	entries := make([]string, 0)
	var buf strings.Builder
	inAngle := false

	for _, r := range seqLine {
		switch r {
		case '<':
			inAngle = true
			buf.WriteRune(r)
		case '>':
			inAngle = false
			buf.WriteRune(r)
		case ';':
			if inAngle {
				buf.WriteRune(r)
				continue
			}
			entry := strings.TrimSpace(buf.String())
			if entry != "" {
				entries = append(entries, entry)
			}
			buf.Reset()
		default:
			buf.WriteRune(r)
		}
	}

	entry := strings.TrimSpace(buf.String())
	if entry != "" {
		entries = append(entries, entry)
	}

	return entries
}
