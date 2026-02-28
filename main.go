package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/badele/splitans/internal/exporter"
	exporterhtml "github.com/badele/splitans/internal/exporter/html"
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
		Oformat   string `short:"F" default:"neotex" enum:"ansi,json,neotex,plaintext,table,stats,html,html-pack" help:"Output format: ansi, json, neotex, plaintext, table, stats, html, html-pack"`
		Oencoding string `short:"E" default:"utf8" enum:"cp437,cp850,utf8,iso-8859-1" help:"Output encoding: cp437, cp850, utf8, iso-8859-1"`
		Width     int    `short:"W" default:"80" help:"Width text to specified width"`
		Lines     int    `short:"N" default:"1000" help:"Nb lines text"`
		Crop      string `short:"C" help:"Crop region: x,y:x1,y1 (1-indexed start:end coordinates)"`
		Inline    bool   `short:"I" help:"Flatten output on a single line (neotex, ansi, plaintext)"`
		VGA       bool   `short:"v" help:"Use true VGA colors (not affected by terminal themes)"`
		Legacy    bool   `short:"L" help:"Use ANSI 1990 legacy mode (no bright backgrounds)"`
		Sauce     bool   `short:"S" help:"Include SAUCE metadata in output (ANSI: binary record, Neotex: labels)"`
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

	if decodedWidth > 0 {
		cli.Output.Width = decodedWidth
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
			if token.Sauce.TInfo2 > 0 && cli.Output.Lines == 1000 {
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
			ansiOutput, _, err = exporter.ExportFlattenedANSIInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cli.Output.VGA, cli.Output.Legacy, cropRegion, sauce)
		} else {
			ansiOutput, _, err = exporter.ExportFlattenedANSIWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cli.Output.VGA, cli.Output.Legacy, cropRegion, sauce)
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

			fmt.Print(string(outputBytes) + saucePart)
		} else {
			outputBytes, err := splitans.ConvertToEncoding([]byte(ansiOutput), cli.Output.Oencoding)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting to output encoding: %v\n", err)
				os.Exit(1)
			}

			fmt.Print(string(outputBytes))
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
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce)
		} else {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating neotex format: %v\n", err)
			os.Exit(1)
		}

		combined := ConcatenateTextAndSequence(plainText, sequenceText, effectiveWidth, " | ")
		fmt.Println(combined)

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
			plainText, _, err = exporter.ExportFlattenedTextInline(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cropRegion)
		} else {
			plainText, _, err = exporter.ExportFlattenedText(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cropRegion)
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

		fmt.Println(string(outputBytes))

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
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce)
		} else {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce)
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

		fmt.Print(htmlOutput)

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
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexInlineWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce)
		} else {
			plainText, sequenceText, effectiveWidth, err = exporter.ExportFlattenedNeotexWithSauce(cli.Output.Width, cli.Output.Lines, tokens, cropRegion, sauce)
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
