package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/badele/splitans/internal/exporter"
	"github.com/badele/splitans/internal/types"
	"github.com/badele/splitans/pkg/splitans"
)

type CLI struct {
	File string `arg:"" optional:"" type:"path" help:"ANSI file to process (reads from stdin if not specified)"`

	Input struct {
		Iformat   string `short:"f" default:"ansi" enum:"ansi,json, neotex" help:"Input format: ansi, json, neotex"`
		Iencoding string `short:"e" default:"utf8" enum:"cp437,cp850,utf8,iso-8859-1" help:"Input encoding: cp437, cp850, utf8, iso-8859-1"`
	} `embed:"" prefix:"" group:"Input options:"`

	Output struct {
		Oformat   string `short:"F" default:"neotex" enum:"ansi,json,neotex,plaintext,table,stats" help:"Output format: ansi, json, neotex, plaintext, table, stats"`
		Oencoding string `short:"E" default:"utf8" enum:"cp437,cp850,utf8,iso-8859-1" help:"Output encoding: cp437, cp850, utf8, iso-8859-1"`
		Save      string `short:"S" type:"path" help:"Save to file (for -oformat option (neotex)"`
		Width     int    `short:"W" default:80 help:"Width text to specified width"`
		Lines     int    `short:"L" default:1000 help:"Nb lines text"`
		VGA       bool   `short:"v" help:"Use true VGA colors (not affected by terminal themes)"`
	} `embed:"" prefix:"" group:"Output options:"`

	Debug struct {
		Debug bool `short:"d" help:"Enable debug mode (displays cursor positions)"`
	} `embed:"" prefix:"" group:"Debug options:"`
}

func ConcatenateTextAndSequence(left, right string, leftWidth int, separator string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

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
		tok = splitans.NewANSITokenizer(data)
		tokens = tok.Tokenize()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ANSI parse error: %v\n", err)
			os.Exit(1)
		}

	case "neotex":
		decodedWidth, tok = splitans.NewNeotexTokenizer(data, cli.Output.Width)
		tokens = tok.Tokenize()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Neotex parse error: %v\n", err)
			os.Exit(1)
		}

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

	// Validate --write option usage
	if cli.Output.Save != "" && cli.Output.Oformat != "neotex" {
		fmt.Fprintf(os.Stderr, "Error: --write option can only be used with --oformat=neotex\n")
		os.Exit(1)
	}

	// if cli.Output.Oformat == "neotex" && cli.Output.Save == "" {
	// 	fmt.Fprintf(os.Stderr, "Error: --oformat=neotex requires --save option to specify output file (.neot and .neos)\n")
	// 	os.Exit(1)
	// }

	// Validate output encoding for neotex (must be utf8)
	if cli.Output.Oformat == "neotex" && cli.Output.Oencoding != "utf8" {
		fmt.Fprintf(os.Stderr, "Error: --oformat=%s requires --Oencoding=utf8 (neotex is always UTF-8)\n", cli.Output.Oformat)
		os.Exit(1)
	}

	/////////////////////////////////////////////////////////////////////////////
	// Write Output format file
	/////////////////////////////////////////////////////////////////////////////
	switch cli.Output.Oformat {
	case "ansi":
		var ansiOutput string
		var err error

		// if cli.Input.Iformat == "ansi" {
		// 	ansiOutput, err = exporter.ExportPassthroughANSI(tokens)
		// } else {
		// 	ansiOutput, err = exporter.ExportFlattenedANSI(cli.Output.Width, tokens, cli.Output.Oencoding, cli.Output.VGA)
		// }
		ansiOutput, err = exporter.ExportFlattenedANSI(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding, cli.Output.VGA)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting to ANSI: %v\n", err)
			os.Exit(1)
		}

		// Convert to output encoding if needed
		outputBytes, err := splitans.ConvertToEncoding([]byte(ansiOutput), cli.Output.Oencoding)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting to output encoding: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(string(outputBytes))
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
		plainText, sequenceText, err := exporter.ExportFlattenedNeotex(cli.Output.Width, cli.Output.Lines, tokens)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating neotex format: %v\n", err)
			os.Exit(1)
		}

		// metadatas := neotex.ExtractMetadata(strings.Split(sequenceText, "\n"))

		combined := ConcatenateTextAndSequence(plainText, sequenceText, cli.Output.Width, " | ")
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
		plainText, err := exporter.ExportFlattenedText(cli.Output.Width, cli.Output.Lines, tokens, cli.Output.Oencoding)
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
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported output format: %s\n", cli.Output.Oformat)
		os.Exit(1)
	}
}
