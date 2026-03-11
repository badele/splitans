package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/kong"

	"github.com/badele/splitans/internal/exporter"
	exporterhtml "github.com/badele/splitans/internal/exporter/html"
	"github.com/badele/splitans/internal/importer/neotex"
	"github.com/badele/splitans/internal/types"
	"github.com/badele/splitans/pkg/splitans"
)

// ============================================================================
// EXPORTED
// ============================================================================

type CLI struct {
	File string `arg:"" optional:"" type:"path" help:"ANSI file to process (reads from stdin if not specified)"`

	Input struct {
		Iformat   string `short:"f" default:"ansi" enum:"ansi,json, neotex" help:"Input format: ansi, json, neotex"`
		Iencoding string `short:"e" default:"utf8" enum:"cp437,cp850,utf8,iso-8859-1" help:"Input encoding: cp437, cp850, utf8, iso-8859-1"`
	} `embed:"" prefix:"" group:"Input options:"`

	Output struct {
		Oformat           string `short:"F" default:"neotex" enum:"ansi,json,neotex,plaintext,table,stats,html,html-pack" help:"Output format: ansi, json, neotex, plaintext, table, stats, html, html-pack"`
		Oencoding         string `short:"E" default:"utf8" enum:"cp437,cp850,utf8,iso-8859-1" help:"Output encoding: cp437, cp850, utf8, iso-8859-1"`
		Width             int    `short:"W" default:"80" help:"Width text to specified width"`
		Lines             int    `short:"N" default:"25" help:"Nb lines text"`
		Crop              string `short:"C" help:"Crop region: x,y:x1,y1 (1-indexed start:end coordinates)"`
		Inline            bool   `short:"I" help:"Flatten output on a single line (neotex, ansi, plaintext)"`
		KeepTrailingLines bool   `short:"K" help:"Preserve trailing empty lines in ansi/neotex output"`
		VGA               bool   `short:"V" help:"Use true VGA colors (not affected by terminal themes)"`
		Legacy            bool   `short:"L" help:"Use ANSI 1990 legacy mode (no bright backgrounds)"`
		Sauce             bool   `short:"S" help:"Include SAUCE metadata in output (ANSI: binary record, Neotex: labels)"`
		Delay             string `short:"D" help:"Output delay: <duration>[:c|:l] (example: 50ms:c, 120ms:l)"`
		DelayDeprecated   string `short:"d" hidden:"" help:"Deprecated alias for -D"`
	} `embed:"" prefix:"" group:"Output options:"`
}

// ============================================================================
// PRIVATE
// ============================================================================

func ConcatenateTextAndSequence(leftText, rightText string, leftWidth int, separator string) string {
	leftLines := strings.Split(leftText, "\n")
	rightLines := strings.Split(rightText, "\n")

	result := []string{}
	numLines := len(leftLines)

	for i := 0; i < numLines; i++ {
		if i < len(leftLines) {
			leftLine := leftLines[i]
			rightLine := ""
			if i < len(rightLines) {
				rightLine = rightLines[i]
			}

			if len(leftLine) < leftWidth {
				break
			}

			result = append(result, fmt.Sprintf("%s%s%s", leftLine, separator, rightLine))
		}
	}

	return strings.Join(result, "\n")
}

type delayMode int

const (
	delayNone delayMode = iota
	delayChar
	delayLine
)

type delayChange struct {
	line     int
	duration time.Duration
	mode     delayMode
}

func parseDelayMode(mode string) (delayMode, error) {
	switch strings.ToLower(mode) {
	case "c", "char", "character", "chars":
		return delayChar, nil
	case "l", "line", "lines":
		return delayLine, nil
	case "":
		return delayChar, nil
	default:
		return delayNone, fmt.Errorf("unknown delay mode %q", mode)
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func parseDelayDuration(value string) (time.Duration, error) {
	if value == "" {
		return 0, fmt.Errorf("delay requires a duration")
	}
	if isDigits(value) {
		value += "ms"
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	if duration < 0 {
		return 0, fmt.Errorf("delay must be non-negative")
	}
	return duration, nil
}

func parseDelaySpec(spec string) (time.Duration, delayMode, error) {
	if spec == "" {
		return 0, delayNone, nil
	}
	parts := strings.SplitN(spec, ":", 2)
	duration, err := parseDelayDuration(parts[0])
	if err != nil {
		return 0, delayNone, err
	}
	mode := ""
	if len(parts) == 2 {
		mode = parts[1]
	}
	parsedMode, err := parseDelayMode(mode)
	if err != nil {
		return 0, delayNone, err
	}
	return duration, parsedMode, nil
}

func buildDelaySchedule(meta *neotex.NeotexMetadata) []delayChange {
	if meta == nil || len(meta.DelayChanges) == 0 {
		return nil
	}

	schedule := make([]delayChange, 0, len(meta.DelayChanges))
	for _, change := range meta.DelayChanges {
		mode := delayNone
		switch change.Mode {
		case neotex.NeotexDelayChar:
			mode = delayChar
		case neotex.NeotexDelayLine:
			mode = delayLine
		}
		duration := change.Duration
		if duration <= 0 {
			duration = 0
			mode = delayNone
		}
		schedule = append(schedule, delayChange{line: change.Line, duration: duration, mode: mode})
	}

	return schedule
}

func scanAnsiSequence(content string, start int) int {
	length := len(content)
	if start+1 >= length {
		return start + 1
	}
	next := content[start+1]
	switch next {
	case '[':
		for i := start + 2; i < length; i++ {
			b := content[i]
			if b >= 0x40 && b <= 0x7E {
				return i + 1
			}
		}
	case ']':
		for i := start + 2; i < length; i++ {
			if content[i] == 0x07 {
				return i + 1
			}
			if content[i] == 0x1b && i+1 < length && content[i+1] == '\\' {
				return i + 2
			}
		}
	case 'P', 'X', '^', '_':
		for i := start + 2; i < length; i++ {
			if content[i] == 0x1b && i+1 < length && content[i+1] == '\\' {
				return i + 2
			}
		}
	default:
		if start+2 <= length {
			return start + 2
		}
		return length
	}
	return length
}

func writeWithDelay(writer io.Writer, content string, delay time.Duration, mode delayMode) error {
	if delay <= 0 || mode == delayNone {
		_, err := writer.Write([]byte(content))
		return err
	}
	switch mode {
	case delayLine:
		parts := strings.SplitAfter(content, "\n")
		for _, part := range parts {
			if part == "" {
				continue
			}
			if _, err := writer.Write([]byte(part)); err != nil {
				return err
			}
			if strings.HasSuffix(part, "\n") {
				time.Sleep(delay)
			}
		}
		return nil
	case delayChar:
		for i := 0; i < len(content); {
			if content[i] == 0x1b {
				end := scanAnsiSequence(content, i)
				if _, err := writer.Write([]byte(content[i:end])); err != nil {
					return err
				}
				i = end
				continue
			}
			r, size := utf8.DecodeRuneInString(content[i:])
			if r == utf8.RuneError && size == 1 {
				size = 1
			}
			if _, err := writer.Write([]byte(content[i : i+size])); err != nil {
				return err
			}
			i += size
			if r != ' ' && r != 0x00 && r != '\u2800' {
				time.Sleep(delay)
			}
		}
		return nil
	default:
		_, err := writer.Write([]byte(content))
		return err
	}
}

func writeWithDelaySchedule(writer io.Writer, content string, schedule []delayChange) error {
	if len(schedule) == 0 {
		_, err := writer.Write([]byte(content))
		return err
	}

	lines := strings.SplitAfter(content, "\n")
	currentDuration := time.Duration(0)
	currentMode := delayNone
	scheduleIdx := 0

	for lineIdx, line := range lines {
		if line == "" {
			continue
		}
		for scheduleIdx < len(schedule) && schedule[scheduleIdx].line == lineIdx {
			currentDuration = schedule[scheduleIdx].duration
			currentMode = schedule[scheduleIdx].mode
			scheduleIdx++
		}
		if err := writeWithDelay(writer, line, currentDuration, currentMode); err != nil {
			return err
		}
	}

	return nil
}

func writeOutputWithDelay(writer io.Writer, content string, delay time.Duration, mode delayMode, schedule []delayChange) error {
	if len(schedule) > 0 {
		return writeWithDelaySchedule(writer, content, schedule)
	}
	return writeWithDelay(writer, content, delay, mode)
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("splitans"),
		kong.Description("ANSI art file processor - displays plain text content by default.\nUse output redirection to save to file: splitans file.ans > output.txt"),
		kong.UsageOnError(),
	)

	var data []byte
	var err error
	var filename string
	var encoding string
	decodedWidth := 0
	var delayDuration time.Duration
	var delayMode delayMode
	var delaySchedule []delayChange

	/////////////////////////////////////////////////////////////////////////////
	// Parse argument file or stdin
	/////////////////////////////////////////////////////////////////////////////
	// Read from stdin if no file argument is provided
	if cli.File == "" {
		// Check if stdin is a pipe or has data
		stat, err := os.Stdin.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking stdin: %v\n", err)
			os.Exit(1)
		}

		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Reading from pipe
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				os.Exit(1)
			}
			filename = "stdin"
		} else {
			// No pipe and no file argument - show help
			_ = ctx.PrintUsage(false)
			os.Exit(0)
		}
	} else {
		// Read from file
		filename = cli.File
		data, err = os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	// Convert encoding to UTF-8
	encoding = cli.Input.Iencoding
	switch cli.Input.Iformat {
	case "neotex":
		if cli.Input.Iencoding != "utf8" {
			fmt.Fprintf(os.Stderr, "Error: --iformat=%s requires --Iencoding=utf8 (neotex is always UTF-8)\n", cli.Input.Iencoding)
			os.Exit(1)
		}
		encoding = "utf8"
	}

	data, err = splitans.ConvertToUTF8(data, encoding)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Encoding conversion error: %v\n", err)
		os.Exit(1)
	}

	var tokens []types.Token
	var tok types.TokenizerWithStats

	/////////////////////////////////////////////////////////////////////////////
	// Read Input format file
	/////////////////////////////////////////////////////////////////////////////
	switch cli.Input.Iformat {
	case "ansi":
		// Pass source encoding to tokenizer for SAUCE text field conversion
		tok = splitans.NewANSITokenizerWithEncoding(data, encoding)
		tokens = tok.Tokenize()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ANSI parse error: %v\n", err)
			os.Exit(1)
		}

	case "neotex":
		decodedWidth, tok, err = splitans.NewNeotexTokenizer(data, cli.Output.Width, cli.Output.Legacy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Neotex parse error: %v\n", err)
			os.Exit(1)
		}
		tokens = tok.Tokenize()

	// case "neotex":
	// 	tok = neotex.NewTokenizer(textData, seqData)
	// 	tokens = tok.Tokenize()
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Neotex parse error: %v\n", err)
	// 		os.Exit(1)
	// 	}

	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", cli.Input.Iformat)
		os.Exit(1)
	}

	delaySpec := cli.Output.Delay
	if cli.Output.Delay != "" && cli.Output.DelayDeprecated != "" {
		fmt.Fprintln(os.Stderr, "Warning: -d is deprecated and ignored because -D is set.")
	}
	if delaySpec == "" && cli.Output.DelayDeprecated != "" {
		delaySpec = cli.Output.DelayDeprecated
		fmt.Fprintln(os.Stderr, "Warning: -d is deprecated; use -D instead.")
	}
	if delaySpec != "" {
		delayDuration, delayMode, err = parseDelaySpec(delaySpec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing delay: %v\n", err)
			os.Exit(1)
		}
	} else if provider, ok := tok.(interface{ Metadata() *neotex.NeotexMetadata }); ok {
		if meta := provider.Metadata(); meta != nil && meta.DelayExplicit {
			delaySchedule = buildDelaySchedule(meta)
			if len(delaySchedule) == 0 {
				delayDuration = meta.DelayDuration
				switch meta.DelayMode {
				case neotex.NeotexDelayLine:
					delayMode = delayLine
				case neotex.NeotexDelayChar:
					delayMode = delayChar
				default:
					delayMode = delayNone
				}
				if delayDuration <= 0 {
					delayDuration = 0
					delayMode = delayNone
				}
			}
		}
	}

	if decodedWidth > 0 {
		cli.Output.Width = decodedWidth
	}
	if cli.Output.Lines == 25 {
		if provider, ok := tok.(interface{ LineCount() int }); ok {
			if lineCount := provider.LineCount(); lineCount > 0 {
				cli.Output.Lines = lineCount
			}
		}
	}

	// After tokenizing, check for SAUCE dimensions
	// Priority: SAUCE > CLI defaults (user can still override with explicit -W/-L)
	for _, token := range tokens {
		if token.Type == types.TokenSauce && token.Sauce != nil {
			// Use SAUCE width if CLI is at default value
			if token.Sauce.TInfo1 > 0 && cli.Output.Width == 80 {
				cli.Output.Width = int(token.Sauce.TInfo1)
			}
			// Use SAUCE height if CLI is at default value
			if token.Sauce.TInfo2 > 0 && cli.Output.Lines == 25 {
				cli.Output.Lines = int(token.Sauce.TInfo2)
			}
			break
		}
	}

	// Validate output encoding for neotex (must be utf8)
	if cli.Output.Oformat == "neotex" && cli.Output.Oencoding != "utf8" {
		fmt.Fprintf(os.Stderr, "Error: --oformat=%s requires --Oencoding=utf8 (neotex is always UTF-8)\n", cli.Output.Oformat)
		os.Exit(1)
	}

	// Parse crop region if specified
	cropRegion, err := types.ParseCropRegion(cli.Output.Crop)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing crop region: %v\n", err)
		os.Exit(1)
	}

	/////////////////////////////////////////////////////////////////////////////
	// Write Output format file
	/////////////////////////////////////////////////////////////////////////////
	switch cli.Output.Oformat {
	case "ansi":
		var ansiOutput string
		var err error

		// Create or reuse SAUCE record if requested
		var sauce *types.Sauce
		if cli.Output.Sauce {
			for _, token := range tokens {
				if token.Type == types.TokenSauce && token.Sauce != nil {
					sauce = token.Sauce
					break
				}
			}
			if sauce == nil {
				sauce = types.NewSauce(cli.Output.Width, cli.Output.Lines)
			}
		}

		if cli.Output.Inline {
			ansiOutput, _, err = exporter.ExportFlattenedANSIInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cli.Output.VGA, cli.Output.Legacy, cropRegion, sauce, cli.Output.KeepTrailingLines)
		} else {
			ansiOutput, _, err = exporter.ExportFlattenedANSIWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cli.Output.VGA, cli.Output.Legacy, cropRegion, sauce, cli.Output.KeepTrailingLines)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting to ANSI: %v\n", err)
			os.Exit(1)
		}

		// Convert to output encoding if needed (but not SAUCE bytes which are binary)
		if sauce != nil {
			// When SAUCE is present, convert only the ANSI part, keep SAUCE binary
			ansiLen := len(ansiOutput) - types.SauceTotalSize
			ansiPart := ansiOutput[:ansiLen]
			saucePart := ansiOutput[ansiLen:]

			outputBytes, err := splitans.ConvertToEncoding([]byte(ansiPart), cli.Output.Oencoding)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting to output encoding: %v\n", err)
				os.Exit(1)
			}

			output := string(outputBytes) + saucePart
			if err := writeOutputWithDelay(os.Stdout, output, delayDuration, delayMode, delaySchedule); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
				os.Exit(1)
			}
		} else {
			outputBytes, err := splitans.ConvertToEncoding([]byte(ansiOutput), cli.Output.Oencoding)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting to output encoding: %v\n", err)
				os.Exit(1)
			}
			if err := writeOutputWithDelay(os.Stdout, string(outputBytes), delayDuration, delayMode, delaySchedule); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
				os.Exit(1)
			}
		}
	// case "neotex":
	// 	// Neotex format is always UTF-8 (outputEncoding parameter is ignored by ExportFlattenedNeotex)
	// 	plainText, sequenceText, err := exporter.ExportFlattenedNeotex(cli.Output.Width, cli.Output.Lines, tokens)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error generating neotex format: %v\n", err)
	// 		os.Exit(1)
	// 	}
	//
	// 	if err := exporter.ExportToNeotexFile(cli.Output.Save, plainText, sequenceText); err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error exporting to neotex file: %v\n", err)
	// 		os.Exit(1)
	// 	}
	case "neotex":
		// Neotex format is always UTF-8 (outputEncoding parameter is ignored by ExportFlattenedNeotex)
		var plainText, sequenceText string
		var effectiveWidth int

		// Always extract SAUCE from tokens for neotex output
		var sauce *types.Sauce
		for _, token := range tokens {
			if token.Type == types.TokenSauce && token.Sauce != nil {
				sauce = token.Sauce
				break
			}
		}
		// If -S flag is set and no SAUCE found, create one with dimensions
		if cli.Output.Sauce && sauce == nil {
			sauce = types.NewSauce(cli.Output.Width, cli.Output.Lines)
		}

		if cli.Output.Inline {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce, cli.Output.KeepTrailingLines)
		} else {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce, cli.Output.KeepTrailingLines)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating neotex format: %v\n", err)
			os.Exit(1)
		}

		combined := ConcatenateTextAndSequence(plainText, sequenceText, effectiveWidth, " | ")
		if err := writeOutputWithDelay(os.Stdout, combined+"\n", delayDuration, delayMode, delaySchedule); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}

	case "json":
		exporter.TokensJSON(tok)
	case "stats":
		exporter.DisplayStats(tok)
	case "table":
		stats := tok.GetStats()
		if stats.PosFirstBadSequence > 0 {
			fmt.Printf("=== Parsing file: %s ===\n\n", filename)
		}

		fmt.Printf("=== %% Parsed %f  ===\n", stats.ParsedPercent)

		if err := exporter.ExportTokensToTable(tokens, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying table: %v\n", err)
			os.Exit(1)
		}
	case "plaintext":
		var plainText string
		if cli.Output.Inline {
			plainText, _, err = exporter.ExportFlattenedTextInline(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cropRegion, cli.Output.KeepTrailingLines)
		} else {
			plainText, _, err = exporter.ExportFlattenedText(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cropRegion, cli.Output.KeepTrailingLines)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying plain text: %v\n", err)
			os.Exit(1)
		}

		// Convert to output encoding if needed
		outputBytes, err := splitans.ConvertToEncoding([]byte(plainText), cli.Output.Oencoding)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to output encoding: %v\n", err)
			os.Exit(1)
		}

		// Replace null bytes (0x0) with spaces (0x20)
		for i, b := range outputBytes {
			if b == 0x0 {
				outputBytes[i] = 0x20
			}
		}

		if err := writeOutputWithDelay(os.Stdout, string(outputBytes)+"\n", delayDuration, delayMode, delaySchedule); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}

	case "html":
		var plainText, sequenceText string
		var effectiveWidth int

		var sauce *types.Sauce
		for _, token := range tokens {
			if token.Type == types.TokenSauce && token.Sauce != nil {
				sauce = token.Sauce
				break
			}
		}
		if cli.Output.Sauce && sauce == nil {
			sauce = types.NewSauce(cli.Output.Width, cli.Output.Lines)
		}

		if cli.Output.Inline {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce, cli.Output.KeepTrailingLines)
		} else {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce, cli.Output.KeepTrailingLines)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating neotex format for HTML: %v\n", err)
			os.Exit(1)
		}

		neotexContent := ConcatenateTextAndSequence(plainText, sequenceText, effectiveWidth, " | ")
		htmlOutput, err := exporterhtml.ExportHTML(neotexContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating HTML export: %v\n", err)
			os.Exit(1)
		}

		if err := writeOutputWithDelay(os.Stdout, htmlOutput, delayDuration, delayMode, delaySchedule); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}

	case "html-pack":
		var plainText, sequenceText string
		var effectiveWidth int

		var sauce *types.Sauce
		for _, token := range tokens {
			if token.Type == types.TokenSauce && token.Sauce != nil {
				sauce = token.Sauce
				break
			}
		}
		if cli.Output.Sauce && sauce == nil {
			sauce = types.NewSauce(cli.Output.Width, cli.Output.Lines)
		}

		if cli.Output.Inline {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce, cli.Output.KeepTrailingLines)
		} else {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce, cli.Output.KeepTrailingLines)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating neotex format for HTML pack: %v\n", err)
			os.Exit(1)
		}

		neotexContent := ConcatenateTextAndSequence(plainText, sequenceText, effectiveWidth, " | ")
		pack, err := exporterhtml.ExportHTMLPack(neotexContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating HTML pack: %v\n", err)
			os.Exit(1)
		}

		outputDir := "exported-html"
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", outputDir, err)
			os.Exit(1)
		}

		if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(pack.HTML), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing index.html: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "style.css"), []byte(pack.CSS), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing style.css: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "app.js"), []byte(pack.JS), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing app.js: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(outputDir, pack.FontName), pack.FontData, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", pack.FontName, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Generated %s/index.html, style.css, app.js, %s\n", outputDir, pack.FontName)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported output format: %s\n", cli.Output.Oformat)
		os.Exit(1)
	}
}
