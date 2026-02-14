package processor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/badele/splitans/internal/types"
)

///////////////////////////////////////////////////////////////////////////////
// Virtual Terminal
///////////////////////////////////////////////////////////////////////////////

type VirtualTerminal struct {
	buffer           [][]types.Cell
	width            int
	height           int
	cursorX          int
	cursorY          int
	currentSGR       *types.SGR
	currentHyperlink *types.Hyperlink // Current active hyperlink (OSC 8)
	savedCursorX     int
	savedCursorY     int
	outputEncoding   string
	useVGAColors     bool
	debugCursor      bool
	debugSGR         bool
	lastWrapped      bool
	// Ignore CR/LF tokens that immediately follow a soft wrap at width.
	ignoreWrapCRLF bool
}

// ContentBounds represents the bounding box of actual content in the buffer.
type ContentBounds struct {
	MinX   int  // First column with content (0-indexed)
	MaxX   int  // Last column with content (0-indexed)
	MinY   int  // First row with content (0-indexed)
	MaxY   int  // Last row with content (0-indexed)
	Width  int  // MaxX - MinX + 1
	Height int  // MaxY - MinY + 1
	Empty  bool // True if no content found
}

// ============================================================================
// EXPORTED
// ============================================================================

func NewVirtualTerminal(width, height int, outputEncoding string, useVGAColors bool) *VirtualTerminal {
	defaultSGR := types.NewSGR()
	buffer := make([][]types.Cell, height)
	for i := range buffer {
		buffer[i] = make([]types.Cell, width)
		for j := range buffer[i] {
			buffer[i][j] = types.NewCell()
		}
	}

	return &VirtualTerminal{
		buffer:         buffer,
		width:          width,
		height:         height,
		cursorX:        0,
		cursorY:        0,
		currentSGR:     defaultSGR,
		outputEncoding: outputEncoding,
		useVGAColors:   useVGAColors,
		debugCursor:    false,
		debugSGR:       false,
		lastWrapped:    false,
		ignoreWrapCRLF: true,
	}
}

func (vt *VirtualTerminal) GetWidth() int {
	return vt.width
}

func (vt *VirtualTerminal) GetMaxCursorX() int {
	bounds := vt.GetContentBounds()
	if bounds.Empty {
		return 0
	}
	return bounds.MaxX
}

func (vt *VirtualTerminal) GetMaxCursorY() int {
	bounds := vt.GetContentBounds()
	if bounds.Empty {
		return 0
	}
	return bounds.MaxY
}

// ApplyTokens applies ANSI tokens to the virtual terminal
func (vt *VirtualTerminal) ApplyTokens(tokens []types.Token) error {
	for _, token := range tokens {
		if err := vt.applyToken(token); err != nil {
			return err
		}
	}
	return nil
}

// ExportFlattenedANSI exports the buffer with optimized ANSI codes using differential encoding.
// Uses ExportSplitTextAndSequences and applies minimal SGR codes at the appropriate positions.
// The legacyMode ensures ANSI 1990 compatibility by using reset+rebuild
// when attributes need to be turned OFF, rather than using codes like [22m, [23m, etc.
func (vt *VirtualTerminal) ExportFlattenedANSI() string {
	return vt.exportFlattenedANSI(false)
}

// ExportFlattenedANSIInline exports the buffer as a single line with minimal ANSI codes.
func (vt *VirtualTerminal) ExportFlattenedANSIInline() string {
	return vt.exportFlattenedANSI(true)
}

// ExportPlainText exports the buffer as plain text without ANSI codes
// Uses ExportSplitTextAndSequences and extracts only the text part
func (vt *VirtualTerminal) ExportPlainText() string {
	return vt.exportPlainText(false)
}

// ExportPlainTextInline exports the buffer as plain text without newlines.
func (vt *VirtualTerminal) ExportPlainTextInline() string {
	return vt.exportPlainText(true)
}

// ExportSplitTextAndSequences exports the buffer as separate text and sequences
// Returns a slice of LineWithSequences, each containing the plain text and SGR changes
func (vt *VirtualTerminal) ExportSplitTextAndSequences() []types.LineWithSequences {
	result := []types.LineWithSequences{}
	var currentSGR *types.SGR = nil
	var currentHyperlink *types.Hyperlink = nil

	maxCursorY := 0
	for y := 0; y < vt.height; y++ {

		// Check if line has content
		for x := 0; x < vt.width; x++ {
			if vt.buffer[y][x].Char != 0x0 {
				maxCursorY = max(maxCursorY, y)
				break
			}
		}

		line := types.LineWithSequences{
			Text:               "",
			Sequences:          []types.SGRSequence{},
			HyperlinkSequences: []types.HyperlinkSequence{},
		}

		var textBuilder strings.Builder
		rowMaxX := -1
		for x := 0; x < vt.width; x++ {
			if vt.buffer[y][x].Char != 0x0 {
				rowMaxX = x
			}
		}
		if rowMaxX == -1 {
			rowMaxX = vt.width - 1
		}
		emittedTrailingChange := false

		for x := 0; x < vt.width; x++ {
			cell := vt.buffer[y][x]

			// fmt.Printf("Processing cell at (%d, %d): Char='%c' SGR='%v'\n", x, y, cell.Char, cell.SGR)

			// Detect SGR/Hyperlink changes only up to last content column
			if x <= rowMaxX {
				if !cell.SGR.Equals(currentSGR) {
					line.Sequences = append(line.Sequences, types.SGRSequence{
						Position: x,
						SGR:      cell.SGR.Copy(),
					})
					currentSGR = cell.SGR.Copy()
				}

				if !cell.Hyperlink.Equals(currentHyperlink) {
					var hyperlinkCopy *types.Hyperlink
					if cell.Hyperlink != nil {
						hyperlinkCopy = cell.Hyperlink.Copy()
					}
					line.HyperlinkSequences = append(line.HyperlinkSequences, types.HyperlinkSequence{
						Position:  x,
						Hyperlink: hyperlinkCopy,
					})
					currentHyperlink = hyperlinkCopy
				}
			} else {
				// Trailing area: emit a single SGR change if different, but keep currentSGR to preserve cross-line state
				if !emittedTrailingChange && !cell.SGR.Equals(currentSGR) {
					line.Sequences = append(line.Sequences, types.SGRSequence{
						Position: x,
						SGR:      cell.SGR.Copy(),
					})
					emittedTrailingChange = true
				}
			}

			// Add character to text (replace 0x0 with space)
			char := cell.Char
			if vt.outputEncoding == "utf8" && char == 0x0 {
				char = ' '
			}

			textBuilder.WriteRune(char)
		}

		line.Text = textBuilder.String()

		result = append(result, line)
	}

	return result[:maxCursorY+1]
}

///////////////////////////////////////////////////////////////////////////////
// Buffer Access and Manipulation
///////////////////////////////////////////////////////////////////////////////

// GetBuffer returns a deep copy of the buffer with all cells and their styles.
// Each cell contains the character and its complete SGR state.
// Useful for extracting raw buffer data for manipulation or cropping.
func (vt *VirtualTerminal) GetBuffer() [][]types.Cell {
	result := make([][]types.Cell, vt.height)
	for y := 0; y < vt.height; y++ {
		result[y] = make([]types.Cell, vt.width)
		for x := 0; x < vt.width; x++ {
			result[y][x] = vt.buffer[y][x].Copy()
		}
	}
	return result
}

// GetHeight returns the height of the virtual terminal buffer.
func (vt *VirtualTerminal) GetHeight() int {
	return vt.height
}

// GetContentBounds calculates the bounding box of actual content.
// Ignores null characters and default-style spaces (equivalent to R0).
// Returns ContentBounds with Empty=true if no content found.
func (vt *VirtualTerminal) GetContentBounds() ContentBounds {
	minX, minY := vt.width, vt.height
	maxX, maxY := -1, -1
	defaultSGR := types.NewSGR()

	for y := 0; y < vt.height; y++ {
		for x := 0; x < vt.width; x++ {
			cell := vt.buffer[y][x]
			if cell.Char == 0 {
				continue
			}
			if cell.Char == ' ' && cell.Hyperlink == nil {
				if cell.SGR == nil || cell.SGR.Equals(defaultSGR) {
					continue
				}
			}
			minX = min(minX, x)
			maxX = max(maxX, x)
			minY = min(minY, y)
			maxY = max(maxY, y)
		}
	}

	if maxX == -1 {
		return ContentBounds{Empty: true}
	}

	return ContentBounds{
		MinX:   minX,
		MaxX:   maxX,
		MinY:   minY,
		MaxY:   maxY,
		Width:  maxX - minX + 1,
		Height: maxY - minY + 1,
		Empty:  false,
	}
}

// NewVirtualTerminalFromCells creates a VirtualTerminal from a cell buffer.
// Useful for reconstructing a VT after cropping or manipulation.
func NewVirtualTerminalFromCells(cells [][]types.Cell, outputEncoding string, useVGAColors bool) *VirtualTerminal {
	if len(cells) == 0 {
		return NewVirtualTerminal(0, 0, outputEncoding, useVGAColors)
	}

	height := len(cells)
	width := len(cells[0])

	vt := NewVirtualTerminal(width, height, outputEncoding, useVGAColors)

	for y := 0; y < height; y++ {
		for x := 0; x < width && x < len(cells[y]); x++ {
			vt.buffer[y][x] = cells[y][x].Copy()
		}
	}

	return vt
}

// Crop extracts a rectangular region from the buffer and returns a new VirtualTerminal.
// Coordinates are 0-indexed. Returns nil if region is invalid.
// The cropped VT preserves all cell styles (colors, effects).
func (vt *VirtualTerminal) Crop(x, y, width, height int) *VirtualTerminal {
	// Validation
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return nil
	}
	if x >= vt.width || y >= vt.height {
		return nil
	}

	// Adjust if region exceeds bounds
	if x+width > vt.width {
		width = vt.width - x
	}
	if y+height > vt.height {
		height = vt.height - y
	}

	// Create the new buffer
	cells := make([][]types.Cell, height)
	for dy := 0; dy < height; dy++ {
		cells[dy] = make([]types.Cell, width)
		for dx := 0; dx < width; dx++ {
			cells[dy][dx] = vt.buffer[y+dy][x+dx].Copy()
		}
	}

	return NewVirtualTerminalFromCells(cells, vt.outputEncoding, vt.useVGAColors)
}

// Paste copies the content of source into the current VT at position (x, y).
// Cells with Char == 0 are treated as transparent and are not copied.
// If source extends beyond the destination bounds, it is clipped.
// Returns error if coordinates are negative.
func (vt *VirtualTerminal) Paste(source *VirtualTerminal, x, y int) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("invalid position: (%d, %d)", x, y)
	}

	for sy := 0; sy < source.height; sy++ {
		dy := y + sy
		if dy >= vt.height {
			break // Exceeded vertical bounds
		}
		for sx := 0; sx < source.width; sx++ {
			dx := x + sx
			if dx >= vt.width {
				break // Exceeded horizontal bounds
			}
			// Skip transparent cells (Char == 0)
			if source.buffer[sy][sx].Char == 0 {
				continue
			}
			vt.buffer[dy][dx] = source.buffer[sy][sx].Copy()
		}
	}

	return nil
}

// Fill sets all cells in the buffer to the specified character and SGR style.
// Useful for creating colored backgrounds before compositing.
func (vt *VirtualTerminal) Fill(char rune, sgr *types.SGR) {
	for y := 0; y < vt.height; y++ {
		for x := 0; x < vt.width; x++ {
			vt.buffer[y][x] = types.Cell{
				Char: char,
				SGR:  sgr.Copy(),
			}
		}
	}
}

// ============================================================================
// PRIVATE
// ============================================================================

func (vt *VirtualTerminal) applyToken(token types.Token) error {
	switch token.Type {
	case types.TokenText:
		vt.writeText(token.Value)

	case types.TokenC0:
		vt.handleC0(token.C0Code)

	case types.TokenSGR:
		vt.handleSGR(token.Parameters)

	case types.TokenHoverFg:
		if len(token.Parameters) >= 2 {
			switch token.Parameters[0] {
			case "std":
				idx, _ := strconv.Atoi(token.Parameters[1])
				vt.currentSGR.LinkFgColor = types.ColorValue{Type: types.ColorStandard, Index: uint8(idx)}
			case "idx":
				idx, _ := strconv.Atoi(token.Parameters[1])
				vt.currentSGR.LinkFgColor = types.ColorValue{Type: types.ColorIndexed, Index: uint8(idx)}
			case "rgb":
				if len(token.Parameters) == 4 {
					r := parseByteString(token.Parameters[1])
					g := parseByteString(token.Parameters[2])
					b := parseByteString(token.Parameters[3])
					vt.currentSGR.LinkFgColor = types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}
				}
			}
		}
	case types.TokenHoverBg:
		if len(token.Parameters) >= 2 {
			switch token.Parameters[0] {
			case "std":
				idx, _ := strconv.Atoi(token.Parameters[1])
				vt.currentSGR.LinkBgColor = types.ColorValue{Type: types.ColorStandard, Index: uint8(idx)}
			case "idx":
				idx, _ := strconv.Atoi(token.Parameters[1])
				vt.currentSGR.LinkBgColor = types.ColorValue{Type: types.ColorIndexed, Index: uint8(idx)}
			case "rgb":
				if len(token.Parameters) == 4 {
					r := parseByteString(token.Parameters[1])
					g := parseByteString(token.Parameters[2])
					b := parseByteString(token.Parameters[3])
					vt.currentSGR.LinkBgColor = types.ColorValue{Type: types.ColorRGB, R: r, G: g, B: b}
				}
			}
		}

	case types.TokenCSI:
		vt.handleCSI(token)

	case types.TokenOSC:
		vt.handleOSC(token)
	}

	return nil
}

func parseByteString(s string) uint8 {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func (vt *VirtualTerminal) writeText(text string) {
	for _, r := range text {
		if vt.lastWrapped {
			vt.lastWrapped = false
		}

		if vt.debugCursor {
			fmt.Printf("\nBefore writeText Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
		}

		if vt.cursorY < vt.height {
			var hyperlinkCopy *types.Hyperlink
			if vt.currentHyperlink != nil {
				hyperlinkCopy = vt.currentHyperlink.Copy()
			}

			if vt.debugCursor {
				fmt.Printf("Writing char '%c' with SGR '%v'\n", r, vt.currentSGR)
			}

			vt.buffer[vt.cursorY][vt.cursorX] = types.Cell{
				Char:      r,
				SGR:       vt.currentSGR.Copy(),
				Hyperlink: hyperlinkCopy,
			}
			vt.cursorX++

			// Width to next line if we've reached the end
			if vt.cursorX >= vt.width {
				vt.cursorX = 0
				vt.cursorY++
				vt.lastWrapped = true

				if vt.debugCursor {
					fmt.Printf("Soft wrap occurred, moving to next line. Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
				}
			}

			if vt.debugCursor {
				fmt.Printf("After writeText Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
			}
		}
	}
}

func (vt *VirtualTerminal) handleC0(code byte) {
	if vt.debugCursor {
		fmt.Printf("\nBefore handleC0 Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
	}

	if vt.ignoreWrapCRLF && vt.lastWrapped {
		if code == 0x0D {
			return
		}
		if code == 0x0A {
			vt.lastWrapped = false
			return
		}
	}

	switch code {
	case 0x00: // NUL
		vt.cursorX++
		if vt.cursorX >= vt.width {
			vt.cursorX = 0
			vt.cursorY++
		}

	case 0x09: // TAB
		vt.cursorX = ((vt.cursorX / 8) + 1) * 8
		if vt.cursorX >= vt.width {
			vt.cursorX = 0
			vt.cursorY++
		}

	case 0x0A: // LF (Line Feed)
		vt.cursorY++
		if vt.cursorY >= vt.height {
			vt.cursorY = vt.height - 1
		}
		vt.cursorX = 0

	case 0x0D: // CR (Carriage Return)
		vt.cursorX = 0

	case 0x08: // BS (Backspace)
		if vt.cursorX > 0 {
			vt.cursorX--
		}
	}
	vt.lastWrapped = false

	if vt.debugCursor {
		fmt.Printf("\nAfter handleC0 Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
	}

	// vt.computeMaxCursorPosition()

}

func (vt *VirtualTerminal) handleOSC(token types.Token) {
	// Handle OSC 8 hyperlinks
	if token.Hyperlink != nil {
		if token.Hyperlink.URL == "" {
			// Empty URL means hyperlink OFF
			vt.currentHyperlink = nil
		} else {
			// Set new hyperlink
			vt.currentHyperlink = token.Hyperlink.Copy()
		}
	}
}

func (vt *VirtualTerminal) handleSGR(params []string) {
	if vt.debugSGR {
		fmt.Printf("\nBefore handleSGR Current SGR: '%v'\nNew params: %v\n", vt.currentSGR, params)
	}

	// Convert string params to int params
	intParams := make([]int, 0, len(params))
	for _, p := range params {
		if p == "" {
			intParams = append(intParams, 0)
		} else {
			val, err := strconv.Atoi(p)
			if err == nil {
				intParams = append(intParams, val)
			}
		}
	}

	// Apply parameters to current SGR
	if len(intParams) == 0 {
		vt.currentSGR.Reset()
	} else {
		vt.currentSGR.ApplyParams(intParams)
	}

	if vt.debugSGR {
		fmt.Printf("After handleSGR Current SGR: '%v'\n", vt.currentSGR)
	}
}

func (vt *VirtualTerminal) handleCSI(token types.Token) {
	if vt.debugCursor {
		fmt.Printf("\nBefore handleCSI Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
	}

	if len(token.Raw) == 0 {
		return
	}

	lastChar := token.Raw[len(token.Raw)-1]

	switch lastChar {
	case 'A': // Cursor Up
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorY = max(0, vt.cursorY-n)

	case 'B': // Cursor Down
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorY += n

	case 'C': // Cursor Right
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorX += n
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}

	case 'D': // Cursor Left
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorX -= n
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}

	case 'H', 'f': // Cursor Position
		// ESC [ H 	Moves the cursor to line 1, column 1 (Home).
		// ESC [ 6 H 	Moves the cursor to line 6, column 1.
		// ESC [ ; 12 H 	Moves the cursor to line 1, column 12.
		// ESC [ 6 ; 12 H 	Moves the cursor to line 6, column 12.
		// ESC [ 99 ; 99 H 	Moves the cursor to end of Page.

		if vt.debugCursor {
			fmt.Printf("Before CSI Cursor Position with params: %v, Cusor at (%d, %d) \n", token.Parameters, vt.cursorX, vt.cursorY)
		}

		row, col := 1, 1 // default 1,1 in ANSI

		// replace "" by default value (1)
		for i := 0; i < len(token.Parameters); i++ {
			if token.Parameters[i] == "" {
				token.Parameters[i] = "1"
			}
		}

		if len(token.Parameters) > 1 {
			row, _ = strconv.Atoi(token.Parameters[0])
			col, _ = strconv.Atoi(token.Parameters[1])
		} else if len(token.Parameters) > 0 {
			row, _ = strconv.Atoi(token.Parameters[0])
			col = 1
		}
		vt.cursorY = max(0, row-1)
		vt.cursorX = col - 1

		if vt.debugCursor {
			fmt.Printf("After CSI Cursor Position with params: %v, Cusor at (%d, %d) \n", token.Parameters, vt.cursorY, vt.cursorX)
		}
	case 'J': // Erase Display
		mode := 0
		if len(token.Parameters) > 0 {
			mode, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.eraseDisplay(mode)

	case 'K': // Erase Line
		mode := 0
		if len(token.Parameters) > 0 {
			mode, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.eraseLine(mode)

	case 'b': // Repeat previous character (REP)
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		if n < 1 {
			n = 1
		}

		srcX := vt.cursorX - 1
		srcY := vt.cursorY
		if srcX < 0 {
			if vt.cursorY == 0 {
				break
			}
			srcY = vt.cursorY - 1
			srcX = vt.width - 1
		}

		source := vt.buffer[srcY][srcX]
		if source.Char == 0x0 {
			break
		}

		for i := 0; i < n; i++ {
			if vt.cursorY >= vt.height {
				break
			}

			vt.buffer[vt.cursorY][vt.cursorX] = types.Cell{
				Char: source.Char,
				SGR:  source.SGR.Copy(),
			}
			vt.cursorX++

			if vt.cursorX >= vt.width {
				vt.cursorX = 0
				vt.cursorY++
			}
		}

	case 'G': // Cursor Horizontal Absolute (CHA)
		col := 1
		if len(token.Parameters) > 0 && token.Parameters[0] != "" {
			col, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorX = max(0, col-1)
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}

	case 's': // Save Cursor Position
		vt.savedCursorX = vt.cursorX
		vt.savedCursorY = vt.cursorY

	case 'u': // Restore Cursor Position
		vt.cursorX = vt.savedCursorX
		vt.cursorY = vt.savedCursorY
	}

	if vt.debugCursor {
		fmt.Printf("After handleCSI Cursor at (%d, %d) for char '%q' \n", vt.cursorX, vt.cursorY, lastChar)
	}

}

func (vt *VirtualTerminal) eraseDisplay(mode int) {
	switch mode {
	case 0: // Clear from cursor to end of screen
		for y := vt.cursorY; y < vt.height; y++ {
			for x := 0; x < vt.width; x++ {
				if y == vt.cursorY && x < vt.cursorX {
					continue
				}
				vt.buffer[y][x] = types.NewCell()
			}
		}
	case 1: // Clear from beginning of screen to cursor
		for y := 0; y <= vt.cursorY; y++ {
			for x := 0; x < vt.width; x++ {
				if y == vt.cursorY && x > vt.cursorX {
					break
				}
				vt.buffer[y][x] = types.NewCell()
			}
		}
	case 2: // Clear entire screen
		for y := 0; y < vt.height; y++ {
			for x := 0; x < vt.width; x++ {
				vt.buffer[y][x] = types.NewCell()
			}
		}
		vt.cursorX = 0
		vt.cursorY = 0
	}
}

func (vt *VirtualTerminal) eraseLine(mode int) {
	switch mode {
	case 0: // Clear from cursor to end of line
		for x := vt.cursorX; x < vt.width; x++ {
			vt.buffer[vt.cursorY][x] = types.NewCell()
		}
	case 1: // Clear from beginning of line to cursor
		for x := 0; x <= vt.cursorX; x++ {
			vt.buffer[vt.cursorY][x] = types.NewCell()
		}
	case 2: // Clear entire line
		for x := 0; x < vt.width; x++ {
			vt.buffer[vt.cursorY][x] = types.NewCell()
		}
	}
}

func (vt *VirtualTerminal) exportFlattenedANSI(inline bool) string {
	lines := vt.ExportSplitTextAndSequences()
	var builder strings.Builder

	// Track the current SGR state across all lines for differential encoding
	var currentSGR *types.SGR = nil
	var currentHyperlink *types.Hyperlink = nil

	for _, line := range lines {
		var lineBuilder strings.Builder
		textRunes := []rune(line.Text)

		seqIndex := 0
		hyperlinkSeqIndex := 0
		for i, r := range textRunes {
			// Check if there's a SGR change at this position
			if seqIndex < len(line.Sequences) && line.Sequences[seqIndex].Position == i {
				newSGR := line.Sequences[seqIndex].SGR

				// Generate differential ANSI sequence (legacyMode=true for ANSI 1990 compatibility)
				diffSequence := newSGR.DiffToANSI(currentSGR, vt.useVGAColors, true)
				if diffSequence != "" {
					lineBuilder.WriteString(diffSequence)
				}

				// Update current state
				currentSGR = newSGR.Copy()
				seqIndex++
			}

			// Check if there's a hyperlink change at this position
			// Skip hyperlink handling for CP437 encoding as OSC 8 was not supported in 1990s ANSI terminals
			if vt.outputEncoding != "cp437" {
				if hyperlinkSeqIndex < len(line.HyperlinkSequences) && line.HyperlinkSequences[hyperlinkSeqIndex].Position == i {
					newHyperlink := line.HyperlinkSequences[hyperlinkSeqIndex].Hyperlink

					// Generate OSC 8 sequence
					osc8Seq := hyperlinkToOSC8(newHyperlink)
					lineBuilder.WriteString(osc8Seq)

					// Update current state
					if newHyperlink != nil {
						currentHyperlink = newHyperlink.Copy()
					} else {
						currentHyperlink = nil
					}
					hyperlinkSeqIndex++
				}
			}

			lineBuilder.WriteRune(r)
		}

		lineText := lineBuilder.String()
		if vt.outputEncoding == "utf8" {
			lineText = strings.ReplaceAll(lineText, "\x00", " ")
		}

		builder.WriteString(lineText)

		if vt.outputEncoding == "utf8" && !inline {
			builder.WriteString("\n")
		}
	}

	// Reset hyperlink at the end if still active (skip for CP437)
	if vt.outputEncoding != "cp437" && currentHyperlink != nil {
		builder.WriteString("\x1b]8;;\x1b\\")
	}

	// Reset at the end only if not already at default state
	if !currentSGR.Equals(types.NewSGR()) {
		builder.WriteString("\x1b[0m")
	}

	return builder.String()
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

func (vt *VirtualTerminal) exportPlainText(inline bool) string {
	lines := vt.ExportSplitTextAndSequences()

	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString(line.Text)

		if !inline {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
