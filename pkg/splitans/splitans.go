// Package splitans provides a public API for parsing and exporting ANSI art files.
//
// This package provides functions to:
//   - Convert between character encodings (CP437, CP850, ISO-8859-1, UTF-8)
//   - Tokenize ANSI and Neotex format files
//   - Export to various formats (ANSI, plain text, Neotex)
//   - Process tokens through a virtual terminal
//
// Example usage:
//
//	import "github.com/badele/splitans/pkg/splitans"
//
//	data, _ := os.ReadFile("art.ans")
//	utf8Data, _ := splitans.ConvertToUTF8(data, "cp437")
//	tokenizer := splitans.NewANSITokenizer(utf8Data)
//	tokens := tokenizer.Tokenize()
//	output, _ := splitans.ExportFlattenedANSI(80, 25, tokens, "utf8", true)
package splitans

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/badele/splitans/internal/exporter"
	"github.com/badele/splitans/internal/importer/ansi"
	"github.com/badele/splitans/internal/importer/neotex"
	"github.com/badele/splitans/internal/processor"
	"github.com/badele/splitans/internal/types"
)

// Type aliases for public API
type (
	// Token represents a parsed ANSI token (text, control code, escape sequence, etc.)
	Token = types.Token

	// TokenType represents the type of a token
	TokenType = types.TokenType

	// TokenStats contains statistics about parsed tokens
	TokenStats = types.TokenStats

	// SGR represents Select Graphic Rendition attributes (colors, styles)
	SGR = types.SGR

	// ColorValue represents a color (standard, indexed, or RGB)
	ColorValue = types.ColorValue

	// ColorType represents the type of color encoding
	ColorType = types.ColorType

	// Tokenizer is the interface for all tokenizers
	Tokenizer = types.Tokenizer

	// TokenizerWithStats is a tokenizer that also provides statistics
	TokenizerWithStats = types.TokenizerWithStats

	// VirtualTerminal provides a virtual terminal buffer for processing tokens
	VirtualTerminal = processor.VirtualTerminal

	// ANSITokenizer is the tokenizer for ANSI format files
	ANSITokenizer = ansi.Tokenizer

	// NeotexTokenizer is the tokenizer for Neotex format files
	NeotexTokenizer = neotex.Tokenizer
)

// Token type constants
const (
	TokenText          = types.TokenText
	TokenC0            = types.TokenC0
	TokenC1            = types.TokenC1
	TokenCSI           = types.TokenCSI
	TokenCSIInterupted = types.TokenCSIInterupted
	TokenSGR           = types.TokenSGR
	TokenDCS           = types.TokenDCS
	TokenOSC           = types.TokenOSC
	TokenEscape        = types.TokenEscape
	TokenSauce         = types.TokenSauce
	TokenUnknown       = types.TokenUnknown
)

// Color type constants
const (
	ColorDefault  = types.ColorDefault
	ColorStandard = types.ColorStandard
	ColorIndexed  = types.ColorIndexed
	ColorRGB      = types.ColorRGB
)

// VGAPalette contains the 16 standard VGA colors
var VGAPalette = types.VGAPalette

// C0Names maps C0 control codes to their names
var C0Names = types.C0Names

// UTF-8 BOM (Byte Order Mark) sequence
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// stripUTF8BOM removes the UTF-8 BOM if present at the beginning of the data
func stripUTF8BOM(data []byte) []byte {
	if len(data) >= 3 && bytes.Equal(data[:3], utf8BOM) {
		return data[3:]
	}
	return data
}

// ConvertToUTF8 converts byte data from a source encoding to UTF-8.
// Supported encodings: "utf8", "cp437", "cp850", "iso-8859-1"
// The UTF-8 BOM (Byte Order Mark) is automatically stripped if present.
func ConvertToUTF8(data []byte, sourceEncoding string) ([]byte, error) {
	if sourceEncoding == "utf8" {
		return stripUTF8BOM(data), nil
	}

	var decoder *encoding.Decoder

	switch sourceEncoding {
	case "cp437":
		decoder = charmap.CodePage437.NewDecoder()
	case "cp850":
		decoder = charmap.CodePage850.NewDecoder()
	case "iso-8859-1":
		decoder = charmap.ISO8859_1.NewDecoder()
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", sourceEncoding)
	}

	reader := transform.NewReader(bytes.NewReader(data), decoder)
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("encoding conversion error: %w", err)
	}

	// Strip BOM if present after conversion
	return stripUTF8BOM(utf8Data), nil
}

// ConvertToEncoding converts UTF-8 data to the target encoding.
// Supported encodings: "utf8", "cp437", "cp850", "iso-8859-1"
func ConvertToEncoding(data []byte, targetEncoding string) ([]byte, error) {
	if targetEncoding == "utf8" {
		return data, nil
	}

	var encoder *encoding.Encoder

	switch targetEncoding {
	case "cp437":
		encoder = charmap.CodePage437.NewEncoder()
	case "cp850":
		encoder = charmap.CodePage850.NewEncoder()
	case "iso-8859-1":
		encoder = charmap.ISO8859_1.NewEncoder()
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", targetEncoding)
	}

	reader := transform.NewReader(bytes.NewReader(data), encoder)
	encodedData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("encoding conversion error: %w", err)
	}

	return encodedData, nil
}

// NewANSITokenizer creates a new tokenizer for ANSI format data.
// The input should be UTF-8 encoded (use ConvertToUTF8 if needed).
func NewANSITokenizer(input []byte) *ANSITokenizer {
	return ansi.NewANSITokenizer(input)
}

// NewNeotexTokenizer creates a new tokenizer for Neotex format data.
// The width parameter specifies the expected line width.
// Returns the parsed width (overrides when !TWxx/yy is present) and the tokenizer.
func NewNeotexTokenizer(data []byte, width int) (int, *NeotexTokenizer) {
	return neotex.NewNeotexTokenizer(data, width)
}

// NewVirtualTerminal creates a new virtual terminal with the specified dimensions.
// outputEncoding specifies the output encoding ("utf8", "cp437", "cp850", "iso-8859-1").
// useVGAColors enables true VGA colors (not affected by terminal themes).
func NewVirtualTerminal(width, height int, outputEncoding string, useVGAColors bool) *VirtualTerminal {
	return processor.NewVirtualTerminal(width, height, outputEncoding, useVGAColors)
}

// NewSGR creates a new SGR with default values.
func NewSGR() *SGR {
	return types.NewSGR()
}

// ExportFlattenedANSI exports tokens to a flattened ANSI string.
// This processes tokens through a virtual terminal to resolve cursor positioning
// and produces clean ANSI output.
func ExportFlattenedANSI(width, nblines int, tokens []Token, outputEncoding string, useVGAColors bool) (string, error) {
	return exporter.ExportFlattenedANSI(width, nblines, tokens, outputEncoding, useVGAColors)
}

// ExportFlattenedText exports tokens to plain text without ANSI codes.
// This processes tokens through a virtual terminal and outputs only the text content.
func ExportFlattenedText(width, nblines int, tokens []Token, outputEncoding string) (string, error) {
	return exporter.ExportFlattenedText(width, nblines, tokens, outputEncoding)
}

// ExportFlattenedNeotex exports tokens to Neotex format.
// Returns (text, sequences, error) where:
//   - text is the plain text content
//   - sequences is the neotex format sequences with positions
func ExportFlattenedNeotex(width, nblines int, tokens []Token) (string, string, error) {
	return exporter.ExportFlattenedNeotex(width, nblines, tokens)
}

// SGRToNeotex converts an SGR struct to neotex format strings.
func SGRToNeotex(sgr *SGR) []string {
	return exporter.SGRToNeotex(sgr)
}

// DiffSGRToNeotex generates minimal neotex codes to transition between SGR states.
func DiffSGRToNeotex(current, previous *SGR) []string {
	return exporter.DiffSGRToNeotex(current, previous)
}
