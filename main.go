package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"splitans/exporter"
	"splitans/tokenizer"
)

func ConcatenateTextAndSequence(left, right string, leftWidth int, separator string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	result := []string{}
	numLines := len(leftLines)

	// var result strings.Builder

	for i := 0; i < numLines; i++ {
		if i < len(leftLines) {
			leftLine := leftLines[i]
			rightLine := ""
			if i < len(rightLines) {
				rightLine = rightLines[i]
			}
			result = append(result, fmt.Sprintf("%s%s%s", leftLine, separator, rightLine))
		}
	}

	return strings.Join(result, "\n")
}

func main() {
	// Flags
	jsonOutput := flag.Bool("json", false, "")
	flag.BoolVar(jsonOutput, "j", false, "")

	multiFormatMode := flag.Bool("multiformat", false, "")
	flag.BoolVar(multiFormatMode, "m", false, "")

	writePath := flag.String("write", "", "")
	flag.StringVar(writePath, "w", "", "")

	debugMode := flag.Bool("debug", false, "")
	flag.BoolVar(debugMode, "d", false, "")

	statsMode := flag.Bool("stats", false, "")
	flag.BoolVar(statsMode, "s", false, "")

	tableOutput := flag.Bool("table", false, "")
	flag.BoolVar(tableOutput, "t", false, "")

	// Customize help message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [file.ans]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "If no file is specified, reads from stdin (pipe).\n")
		fmt.Fprintf(os.Stderr, "Default behavior: displays plain text content to stdout.\n")
		fmt.Fprintf(os.Stderr, "Use output redirection to save to file: %s file.ans > output.txt\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -t, --table\n")
		fmt.Fprintf(os.Stderr, "        Display tokens in table format\n")
		fmt.Fprintf(os.Stderr, "  -j, --json\n")
		fmt.Fprintf(os.Stderr, "        Display tokens in JSON format\n")
		fmt.Fprintf(os.Stderr, "  -s, --stats\n")
		fmt.Fprintf(os.Stderr, "        Display usage statistics for characters and sequences\n")
		fmt.Fprintf(os.Stderr, "  -m, --multiformat\n")
		fmt.Fprintf(os.Stderr, "        Export to stdout or .ant and .anc files (with -s param)\n")
		fmt.Fprintf(os.Stderr, "  -w, --write\n")
		fmt.Fprintf(os.Stderr, "        Write the multiformat to file .ant and .anc files (with -m param)\n")
		fmt.Fprintf(os.Stderr, "  -d, --debug\n")
		fmt.Fprintf(os.Stderr, "        Enable debug mode (displays cursor positions)\n")
	}

	flag.Parse()

	args := flag.Args()

	var data []byte
	var err error
	var filename string

	// Read from stdin if no file argument is provided
	if len(args) == 0 {
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
			// No pipe and no file argument
			flag.Usage()
			os.Exit(1)
		}
	} else {
		// Read from file
		filename = args[0]
		data, err = os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	tok := tokenizer.NewTokenizer(data)
	tokens := tok.Tokenize()

	// Display statistics
	if *statsMode {
		exporter.DisplayStats(tok)
		return
	}

	// Export to JSON
	if *jsonOutput {
		exporter.DisplayTokensJSON(tok)
		return
	}

	// Display table
	if *tableOutput {
		if tok.Stats.PosFirstBadSequence > 0 {
			fmt.Printf("=== Parsing file: %s ===\n\n", filename)
		}

		// fmt.Printf("=== file size: %d bytes ===\n", tok.FileSize)
		fmt.Printf("=== %% Parsed %f  ===\n", tok.Stats.ParsedPercent)

		if err := exporter.ExportTokensToTable(tokens, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying table: %v\n", err)
			os.Exit(1)
		}
		return
	}

	plainText, err := exporter.GetPlainText(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error displaying plain text: %v\n", err)
		os.Exit(1)
	}

	if *multiFormatMode {
		sequenceText, err := exporter.GetPlainTextSequence(tokens)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying sequence text: %v\n", err)
			os.Exit(1)
		}

		if *writePath != "" {
			if err := exporter.ExportToMultipleFile(*writePath, plainText, sequenceText); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting to multifile: %v\n", err)
				os.Exit(1)
			}

		} else {
			combined := ConcatenateTextAndSequence(plainText, sequenceText, 80, " | ")
			fmt.Println(combined)
		}
	} else {
		fmt.Println(plainText)
		return
	}
}
