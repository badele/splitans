package neotex

import (
	"bytes"
	"fmt"
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

	TrimmedWidth int               // Trimmed width (!W73/80 -> 73)
	Width        int               // Total width (!W73/80 -> 80)
	NbLines      int               // Trimmed lines (!N10/25 -> 10)
	Lines        int               // Total lines (!N10/25 -> 25)
	Extra        map[string]string // Other metadata (!key:value)
	Sauce        *types.Sauce      // SAUCE metadata if present (!SAUCE, !ST, !SA, etc.)
	Palette      map[int]types.ColorValue
}

func validateNeotexLabelValue(key, value string) error {
	if strings.ContainsAny(value, "<>") {
		return fmt.Errorf("neotex label %s contains forbidden angle bracket", key)
	}
	return nil
}

// parseNeotexLabel extracts a label value for a known key (short or protected form).
func parseNeotexLabel(token, key string) (string, bool, error) {
	prefix := "!" + key
	if !strings.HasPrefix(token, prefix) {
		return "", false, nil
	}

	rest := token[len(prefix):]
	if strings.HasPrefix(rest, "=") {
		rest = rest[1:]
	}
	if rest == "" {
		return "", true, nil
	}

	if strings.HasPrefix(rest, "<") {
		if !strings.HasSuffix(rest, ">") {
			return "", true, fmt.Errorf("neotex label %s missing closing bracket", key)
		}
		value := rest[1 : len(rest)-1]
		if err := validateNeotexLabelValue(key, value); err != nil {
			return "", true, err
		}
		return value, true, nil
	}

	if err := validateNeotexLabelValue(key, rest); err != nil {
		return "", true, err
	}

	return rest, true, nil
}

// parseProtectedMetadata extracts key/value from a protected metadata entry (!KEY<value>).
func parseProtectedMetadata(token string) (string, string, bool, error) {
	if !strings.HasPrefix(token, "!") {
		return "", "", false, nil
	}
	body := token[1:]
	idx := strings.Index(body, "<")
	if idx <= 0 {
		return "", "", false, nil
	}
	if !strings.HasSuffix(body, ">") {
		return "", "", true, fmt.Errorf("neotex metadata %s missing closing bracket", body[:idx])
	}
	key := body[:idx]
	key = strings.TrimSuffix(key, "=")
	value := body[idx+1 : len(body)-1]
	if err := validateNeotexLabelValue(key, value); err != nil {
		return "", "", true, err
	}
	return key, value, true, nil
}

func parsePaletteEntry(token string) (int, types.ColorValue, bool, error) {
	if !strings.HasPrefix(token, "!P") {
		return 0, types.ColorValue{}, false, nil
	}
	body := token[2:]
	if body == "" {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry missing index")
	}
	parts := strings.SplitN(body, "=", 2)
	if len(parts) != 2 {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry missing '='")
	}
	idxStr := strings.TrimSpace(parts[0])
	if idxStr == "" {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry missing index")
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry invalid index: %s", idxStr)
	}
	value := strings.TrimSpace(parts[1])
	if strings.HasPrefix(value, "<") {
		if !strings.HasSuffix(value, ">") {
			return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry missing closing bracket")
		}
		value = value[1 : len(value)-1]
	}
	if strings.ContainsAny(value, "<>") {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry has invalid brackets")
	}
	if len(value) != 6 {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry must be 6 hex digits")
	}
	r, g, b, err := parseRGBHex(value)
	if err != nil {
		return 0, types.ColorValue{}, true, fmt.Errorf("neotex palette entry invalid hex")
	}
	return idx, types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}, true, nil
}

// ExtractMetadata extracts metadata from sequence lines
// Metadata entries start with '!' (e.g., !V1 for version)
// Also extracts SAUCE metadata if present (!SAUCE, !ST, !SA, etc.)
func ExtractMetadata(seqLines []string) (NeotexMetadata, error) {
	meta := NeotexMetadata{
		Version: 0, // 0 means no version found (legacy format)
		Extra:   make(map[string]string),
		Palette: make(map[int]types.ColorValue),
	}

	// Collect all tokens for SAUCE parsing
	var allTokens []string

	for _, seqLine := range seqLines {
		entries := splitNeotexEntries(seqLine)
		for _, entry := range entries {
			entry = strings.TrimSpace(entry)
			if !strings.HasPrefix(entry, "!") {
				continue
			}

			// Collect token for SAUCE parsing (with '!' prefix)
			allTokens = append(allTokens, entry)

			if idx, color, ok, err := parsePaletteEntry(entry); ok {
				if err != nil {
					return meta, err
				}
				meta.Palette[idx] = color
				continue
			}

			// Remove '!' prefix for other metadata
			entryWithoutPrefix := entry[1:]

			// Check for version: V<semver>
			if versionStr, ok, err := parseNeotexLabel(entry, "V"); ok {
				if err != nil || versionStr == "" {
					continue
				}
				meta.VersionRaw = versionStr
				if major, minor, patch, ok := parseNeotexVersion(versionStr); ok {
					meta.VersionMajor = major
					meta.VersionMinor = minor
					meta.VersionPatch = patch
					meta.Version = major // legacy major-only field
				}
				continue
			}

			// Check trimmed width W<trimmed>/<total> or W<number>
			if wValue, ok, err := parseNeotexLabel(entry, "W"); ok {
				if err != nil || wValue == "" {
					continue
				}
				if parts := strings.Split(wValue, "/"); len(parts) == 2 {
					// Format: W73/80
					if v, err := strconv.Atoi(parts[0]); err == nil {
						meta.TrimmedWidth = v
					}
					if v, err := strconv.Atoi(parts[1]); err == nil {
						meta.Width = v
					}
				} else if v, err := strconv.Atoi(wValue); err == nil {
					meta.TrimmedWidth = v
					meta.Width = v
				}
				continue
			}

			// Check number of lines N<trimmed>/<total> or N<number>
			if nValue, ok, err := parseNeotexLabel(entry, "N"); ok {
				if err != nil || nValue == "" {
					continue
				}
				if parts := strings.Split(nValue, "/"); len(parts) == 2 {
					if v, err := strconv.Atoi(parts[0]); err == nil {
						meta.NbLines = v
					}
					if v, err := strconv.Atoi(parts[1]); err == nil {
						meta.Lines = v
					}
				} else if v, err := strconv.Atoi(nValue); err == nil {
					meta.NbLines = v
					meta.Lines = v
				}
				continue
			}

			if key, value, ok, err := parseProtectedMetadata(entry); ok {
				if err != nil {
					continue
				}
				meta.Extra[key] = value
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

	// Populate SAUCE dimensions from !W and !N if SAUCE exists
	if meta.Sauce != nil {
		if meta.Width > 0 {
			meta.Sauce.TInfo1 = uint16(meta.Width)
		}
		if meta.Lines > 0 {
			meta.Sauce.TInfo2 = uint16(meta.Lines)
		} else if meta.NbLines > 0 {
			meta.Sauce.TInfo2 = uint16(meta.NbLines)
		}
	}

	return meta, nil
}

// parseSauceLabels extracts SAUCE metadata from neotex labels.
// All SAUCE labels start with "!S" prefix (e.g., !ST, !SA, !SG, !SD, !SF, !SI).
// Note: !SW and !SH are still parsed for backward compatibility but dimensions
// are now taken from !W and !N metadata.
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

		if token == "!SI" {
			sauce.SetICEColors(true)
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "ST"); ok {
			if err == nil {
				sauce.Title = value
			}
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "SA"); ok {
			if err == nil {
				sauce.Author = value
			}
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "SG"); ok {
			if err == nil {
				sauce.Group = value
			}
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "SD"); ok {
			if err == nil {
				sauce.Date, _ = time.Parse("20060102", value)
			}
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "SW"); ok {
			if err == nil {
				if w, err := strconv.Atoi(value); err == nil {
					sauce.TInfo1 = uint16(w)
				}
			}
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "SH"); ok {
			if err == nil {
				if h, err := strconv.Atoi(value); err == nil {
					sauce.TInfo2 = uint16(h)
				}
			}
			continue
		}
		if value, ok, err := parseNeotexLabel(token, "SF"); ok {
			if err == nil {
				sauce.TInfoS = value
			}
			continue
		}
	}

	return sauce
}

// ConvertNeotexToANSI converts neotex format (text + sequences) to raw ANSI format.
// This allows reusing the existing ANSI tokenizer instead of duplicating parsing logic.
// Tracks SGR state across lines for proper differential encoding.
// Takes arrays of lines (without embedded \n) for cleaner processing.
func ConvertNeotexToANSI(textLines []string, seqLines []string, palette map[int]types.ColorValue, legacyMode bool) ([]byte, error) {
	var result bytes.Buffer
	currentSGR := types.NewSGR() // Track SGR state across lines

	for i, textLine := range textLines {
		var seqLine string
		if i < len(seqLines) {
			seqLine = seqLines[i]
		}

		ansiLine, newSGR, err := convertLineToANSI(textLine, seqLine, currentSGR, palette, legacyMode)
		if err != nil {
			return nil, fmt.Errorf("neotex line %d: %w", i+1, err)
		}
		currentSGR = newSGR

		result.WriteString(ansiLine)

		// Add newline if not last line
		// if i < len(textLines)-1 {
		// 	result.WriteString("\n")
		// }
	}

	return result.Bytes(), nil
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
func convertLineToANSI(textLine string, seqLine string, currentSGR *types.SGR, palette map[int]types.ColorValue, legacyMode bool) (string, *types.SGR, error) {
	return convertLineToANSIWithHyperlink(textLine, seqLine, currentSGR, nil, palette, legacyMode)
}

// convertLineToANSIWithHyperlink converts a single line of text with its sequences to ANSI
// Takes the current SGR and Hyperlink state and returns the updated states after processing
func convertLineToANSIWithHyperlink(textLine string, seqLine string, currentSGR *types.SGR, currentHyperlink *types.Hyperlink, palette map[int]types.ColorValue, legacyMode bool) (string, *types.SGR, error) {
	if seqLine == "" {
		return textLine, currentSGR, nil
	}

	styles, err := parseLineSequencesWithHyperlinks(seqLine, palette)
	if err != nil {
		return "", currentSGR, err
	}
	if len(styles) == 0 {
		return textLine, currentSGR, nil
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
		ansiSeq := newSGR.DiffToANSI(currentSGR, false, legacyMode)
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

	return result.String(), currentSGR, nil
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
func parseLineSequencesWithHyperlinks(seqLine string, palette map[int]types.ColorValue) ([]styleChangeWithHyperlink, error) {
	var styles []styleChangeWithHyperlink
	if seqLine == "" {
		return styles, nil
	}

	lastPosition := -1

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
	entries := splitNeotexEntries(seqLine)

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
		if position <= lastPosition {
			return nil, fmt.Errorf("sequence positions must be strictly increasing: %d <= %d", position+1, lastPosition+1)
		}
		lastPosition = position

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
			if strings.HasPrefix(style, "HFP") || strings.HasPrefix(style, "HBP") {
				isFg := strings.HasPrefix(style, "HFP")
				idxStr := strings.TrimPrefix(strings.TrimPrefix(style, "HFP"), "HBP")
				idx, err := strconv.Atoi(idxStr)
				if err != nil || idx < 0 {
					return nil, fmt.Errorf("invalid palette index %q", idxStr)
				}
				color, ok := palette[idx]
				if !ok {
					return nil, fmt.Errorf("undefined palette index %d", idx)
				}
				if isFg {
					hoverFg = &color
				} else {
					hoverBg = &color
				}
				continue
			}
			if strings.HasPrefix(style, "FP") || strings.HasPrefix(style, "BP") {
				idxStr := strings.TrimPrefix(strings.TrimPrefix(style, "FP"), "BP")
				idx, err := strconv.Atoi(idxStr)
				if err != nil || idx < 0 {
					return nil, fmt.Errorf("invalid palette index %q", idxStr)
				}
				color, ok := palette[idx]
				if !ok {
					return nil, fmt.Errorf("undefined palette index %d", idx)
				}
				prefix := "F"
				if strings.HasPrefix(style, "BP") {
					prefix = "B"
				}
				codes = append(codes, fmt.Sprintf("%s%02X%02X%02X", prefix, color.R, color.G, color.B))
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

	return styles, nil
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
func parseLineSequences(seqLine string, palette map[int]types.ColorValue) ([]styleChange, error) {
	var styles []styleChange
	if seqLine == "" {
		return styles, nil
	}

	lastPosition := -1

	// Split by semicolons to get position entries
	entries := splitNeotexEntries(seqLine)

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
		if position <= lastPosition {
			return nil, fmt.Errorf("sequence positions must be strictly increasing: %d <= %d", position+1, lastPosition+1)
		}
		lastPosition = position

		// Parse styles separated by commas
		stylesStr := strings.TrimSpace(parts[1])
		styleList := strings.Split(stylesStr, ",")

		codes := make([]string, 0)
		for _, style := range styleList {
			style = strings.TrimSpace(style)
			if style != "" {
				if strings.HasPrefix(style, "FP") || strings.HasPrefix(style, "BP") {
					idxStr := strings.TrimPrefix(strings.TrimPrefix(style, "FP"), "BP")
					idx, err := strconv.Atoi(idxStr)
					if err != nil || idx < 0 {
						return nil, fmt.Errorf("invalid palette index %q", idxStr)
					}
					color, ok := palette[idx]
					if !ok {
						return nil, fmt.Errorf("undefined palette index %d", idx)
					}
					prefix := "F"
					if strings.HasPrefix(style, "BP") {
						prefix = "B"
					}
					codes = append(codes, fmt.Sprintf("%s%02X%02X%02X", prefix, color.R, color.G, color.B))
					continue
				}
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

	return styles, nil
}
