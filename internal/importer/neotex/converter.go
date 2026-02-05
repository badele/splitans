package neotex

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/badele/splitans/internal/types"
)

// NeotexMetadata contains metadata extracted from neotex format
type NeotexMetadata struct {
	// VersionRaw stores the raw version string after !V (e.g., "1.23")
	VersionRaw string
	// Version is kept for backward compatibility and mirrors VersionMajor.
	Version int
	// Parsed semantic version components (missing minor/patch default to 0)
	VersionMajor int
	VersionMinor int
	VersionPatch int

	TrimmedWidth int               // Trimmed width (!TW73/80 -> 73)
	Width        int               // Total width (!TW73/80 -> 80)
	NbLines      int               // Number of lines with content (!NL<n>)
	Extra        map[string]string // Other metadata (!key:value)
	Sauce        *types.Sauce      // SAUCE metadata if present (!SAUCE, !ST, !SA, etc.)
}

// ExtractMetadata extracts metadata from sequence lines
// Metadata entries start with '!' (e.g., !V1 for version)
// Also extracts SAUCE metadata if present (!SAUCE, !ST, !SA, etc.)
func ExtractMetadata(seqLines []string) NeotexMetadata {
	meta := NeotexMetadata{
		Version: 0, // 0 means no version found (legacy format)
		Extra:   make(map[string]string),
	}

	// Collect all tokens for SAUCE parsing
	var allTokens []string

	for _, seqLine := range seqLines {
		entries := strings.Split(seqLine, ";")
		for _, entry := range entries {
			entry = strings.TrimSpace(entry)
			if !strings.HasPrefix(entry, "!") {
				continue
			}

			// Collect token for SAUCE parsing (with '!' prefix)
			allTokens = append(allTokens, entry)

			// Remove '!' prefix for other metadata
			entryWithoutPrefix := entry[1:]

			// Check for version: V<semver>
			if strings.HasPrefix(entryWithoutPrefix, "V") {
				versionStr := entryWithoutPrefix[1:]
				meta.VersionRaw = versionStr
				if major, minor, patch, ok := parseNeotexVersion(versionStr); ok {
					meta.VersionMajor = major
					meta.VersionMinor = minor
					meta.VersionPatch = patch
					meta.Version = major // legacy major-only field
				}
				continue
			}

			// Check trimmed width TW<trimmed>/<total> or TW<number>
			if strings.HasPrefix(entryWithoutPrefix, "TW") {
				twValue := entryWithoutPrefix[2:]
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
			if strings.HasPrefix(entryWithoutPrefix, "NL") {
				if v, err := strconv.Atoi(entryWithoutPrefix[2:]); err == nil {
					meta.NbLines = v
				}
				continue
			}

			// Other metadata: key:value
			if parts := strings.SplitN(entryWithoutPrefix, ":", 2); len(parts) == 2 {
				meta.Extra[parts[0]] = parts[1]
			}
		}
	}

	// Parse SAUCE metadata from collected tokens
	meta.Sauce = parseSauceLabels(allTokens)

	// Populate SAUCE dimensions from !TW and !NL if SAUCE exists
	if meta.Sauce != nil {
		if meta.Width > 0 {
			meta.Sauce.TInfo1 = uint16(meta.Width)
		}
		if meta.NbLines > 0 {
			meta.Sauce.TInfo2 = uint16(meta.NbLines)
		}
	}

	return meta
}

// parseSauceLabels extracts SAUCE metadata from neotex labels.
// All SAUCE labels start with "!S" prefix (e.g., !ST, !SA, !SG, !SD, !SF, !SI).
// Note: !SW and !SH are still parsed for backward compatibility but dimensions
// are now taken from !TW and !NL metadata.
// Returns nil if no SAUCE labels are found.
func parseSauceLabels(tokens []string) *types.Sauce {
	var sauce *types.Sauce

	for _, token := range tokens {
		// All SAUCE labels start with "!S"
		if !strings.HasPrefix(token, "!S") {
			continue
		}

		// Create Sauce struct on first SAUCE label
		if sauce == nil {
			sauce = &types.Sauce{}
		}

		switch {
		case strings.HasPrefix(token, "!ST<"):
			sauce.Title = extractLabelValue(token)
		case strings.HasPrefix(token, "!SA<"):
			sauce.Author = extractLabelValue(token)
		case strings.HasPrefix(token, "!SG<"):
			sauce.Group = extractLabelValue(token)
		case strings.HasPrefix(token, "!SD<"):
			sauce.Date, _ = time.Parse("20060102", extractLabelValue(token))
		case strings.HasPrefix(token, "!SW<"):
			w, _ := strconv.Atoi(extractLabelValue(token))
			sauce.TInfo1 = uint16(w)
		case strings.HasPrefix(token, "!SH<"):
			h, _ := strconv.Atoi(extractLabelValue(token))
			sauce.TInfo2 = uint16(h)
		case strings.HasPrefix(token, "!SF<"):
			sauce.TInfoS = extractLabelValue(token)
		case token == "!SI":
			sauce.SetICEColors(true)
		}
	}

	return sauce
}

// extractLabelValue extracts the value between < and > from a label.
// e.g., "!ST<Fire Calendar 2025>" returns "Fire Calendar 2025"
func extractLabelValue(label string) string {
	start := strings.Index(label, "<")
	end := strings.LastIndex(label, ">")
	if start >= 0 && end > start {
		return label[start+1 : end]
	}
	return ""
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

// styleChangeWithHyperlink represents a style change that may include hyperlink changes
type styleChangeWithHyperlink struct {
	position        int
	codes           []string
	hyperlink       *types.Hyperlink
	hasHyperlinkOff bool
	hoverFg         *types.ColorValue
	hoverBg         *types.ColorValue
}

// convertLineToANSI converts a single line of text with its sequences to ANSI
// Takes the current SGR state and returns the updated state after processing
func convertLineToANSI(textLine string, seqLine string, currentSGR *types.SGR) (string, *types.SGR) {
	return convertLineToANSIWithHyperlink(textLine, seqLine, currentSGR, nil)
}

// convertLineToANSIWithHyperlink converts a single line of text with its sequences to ANSI
// Takes the current SGR and Hyperlink state and returns the updated states after processing
func convertLineToANSIWithHyperlink(textLine string, seqLine string, currentSGR *types.SGR, currentHyperlink *types.Hyperlink) (string, *types.SGR) {
	if seqLine == "" {
		return textLine, currentSGR
	}

	styles := parseLineSequencesWithHyperlinks(seqLine)
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

		// Generate differential ANSI sequence for SGR
		ansiSeq := newSGR.DiffToANSI(currentSGR, false, true)
		result.WriteString(ansiSeq)

		// Handle hyperlink changes
		if style.hasHyperlinkOff {
			// Generate hyperlink OFF sequence
			result.WriteString(hyperlinkToOSC8(nil))
			currentHyperlink = nil
		} else if style.hyperlink != nil {
			// Generate hyperlink ON sequence
			if style.hoverFg != nil {
				fgCopy := *style.hoverFg
				style.hyperlink.HoverFg = &fgCopy
			}
			if style.hoverBg != nil {
				bgCopy := *style.hoverBg
				style.hyperlink.HoverBg = &bgCopy
			}
			result.WriteString(hyperlinkToOSC8(style.hyperlink))
			currentHyperlink = style.hyperlink
		}

		currentSGR = newSGR
		textPos = style.position
	}

	// Write remaining text
	if textPos < len(textRunes) {
		result.WriteString(string(textRunes[textPos:]))
	}

	return result.String(), currentSGR
}

// hyperlinkToOSC8 converts a Hyperlink to an OSC 8 escape sequence.
// If hyperlink is nil, returns the "close hyperlink" sequence.
func hyperlinkToOSC8(h *types.Hyperlink) string {
	if h == nil || h.URL == "" {
		// Close hyperlink: ESC ] 8 ; ; ESC \
		return "\x1b]8;;\x1b\\"
	}

	// Build params string (key=value:key=value)
	var params string
	if len(h.Params) > 0 {
		var paramParts []string
		for k, v := range h.Params {
			paramParts = append(paramParts, k+"="+v)
		}
		params = strings.Join(paramParts, ":")
	}

	// Open hyperlink: ESC ] 8 ; params ; URL ESC \
	return "\x1b]8;" + params + ";" + h.URL + "\x1b\\"
}

// parseLineSequencesWithHyperlinks parses sequences for a single line, including hyperlinks.
// Returns a slice of styleChangeWithHyperlink in the order they appear.
// Metadata entries starting with '!' are ignored (e.g., !V1 for version)
func parseLineSequencesWithHyperlinks(seqLine string) []styleChangeWithHyperlink {
	var styles []styleChangeWithHyperlink
	if seqLine == "" {
		return styles
	}

	parseHoverColorValue := func(code string) *types.ColorValue {
		if len(code) == 0 {
			return nil
		}
		// RGB (6 hex)
		if len(code) == 6 {
			if r, g, b, err := parseRGBHex(code); err == nil {
				return &types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}
			}
		}
		// Indexed
		if idx, err := strconv.Atoi(code); err == nil && idx >= 0 && idx <= 255 {
			return &types.ColorValue{Type: types.ColorIndexed, Index: uint8(idx)}
		}
		// Standard single-letter (k r g y b m c w K R G Y B M C W)
		if len(code) == 1 {
			switch code {
			case "k":
				return &types.ColorValue{Type: types.ColorStandard, Index: 0}
			case "r":
				return &types.ColorValue{Type: types.ColorStandard, Index: 1}
			case "g":
				return &types.ColorValue{Type: types.ColorStandard, Index: 2}
			case "y":
				return &types.ColorValue{Type: types.ColorStandard, Index: 3}
			case "b":
				return &types.ColorValue{Type: types.ColorStandard, Index: 4}
			case "m":
				return &types.ColorValue{Type: types.ColorStandard, Index: 5}
			case "c":
				return &types.ColorValue{Type: types.ColorStandard, Index: 6}
			case "w":
				return &types.ColorValue{Type: types.ColorStandard, Index: 7}
			case "K":
				return &types.ColorValue{Type: types.ColorStandard, Index: 8}
			case "R":
				return &types.ColorValue{Type: types.ColorStandard, Index: 9}
			case "G":
				return &types.ColorValue{Type: types.ColorStandard, Index: 10}
			case "Y":
				return &types.ColorValue{Type: types.ColorStandard, Index: 11}
			case "B":
				return &types.ColorValue{Type: types.ColorStandard, Index: 12}
			case "M":
				return &types.ColorValue{Type: types.ColorStandard, Index: 13}
			case "C":
				return &types.ColorValue{Type: types.ColorStandard, Index: 14}
			case "W":
				return &types.ColorValue{Type: types.ColorStandard, Index: 15}
			}
		}
		return nil
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

		// Split position from styles: "14:Fr, EU, HL:<url>"
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

		var codes []string
		var hyperlink *types.Hyperlink
		hasHyperlinkOff := false
		var hoverFg *types.ColorValue
		var hoverBg *types.ColorValue

		for _, style := range styleList {
			style = strings.TrimSpace(style)
			if style == "" {
				continue
			}

			// Check if it's a hyperlink code
			if h, isHyperlink := ApplyNeotexHyperlinkCode(style); isHyperlink {
				if h == nil {
					hasHyperlinkOff = true
				} else {
					hyperlink = h
				}
				continue
			}

			// Hover colors HF/HB (reuse ApplyNeotexCode parsing rules)
			if strings.HasPrefix(style, "HF") || strings.HasPrefix(style, "HB") {
				if strings.HasPrefix(style, "HF") {
					if c := parseHoverColorValue(style[2:]); c != nil {
						hoverFg = c
						continue
					}
				}
				if strings.HasPrefix(style, "HB") {
					if c := parseHoverColorValue(style[2:]); c != nil {
						hoverBg = c
						continue
					}
				}
			}

			codes = append(codes, style)
		}

		if len(codes) > 0 || hyperlink != nil || hasHyperlinkOff || hoverFg != nil || hoverBg != nil {
			styles = append(styles, styleChangeWithHyperlink{
				position:        position,
				codes:           codes,
				hyperlink:       hyperlink,
				hasHyperlinkOff: hasHyperlinkOff,
				hoverFg:         hoverFg,
				hoverBg:         hoverBg,
			})
		}
	}

	return styles
}

// parseNeotexVersion parses a version string that may contain 1 to 3 numeric segments.
// Missing minor or patch segments default to 0 (e.g., "1" -> 1.0.0, "1.2" -> 1.2.0).
func parseNeotexVersion(raw string) (int, int, int, bool) {
	if raw == "" {
		return 0, 0, 0, false
	}

	parts := strings.Split(raw, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return 0, 0, 0, false
	}

	var nums [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0, 0, 0, false
		}
		nums[i] = n
	}

	return nums[0], nums[1], nums[2], true
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
