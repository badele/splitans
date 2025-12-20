package processor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/badele/splitans/types"
)

///////////////////////////////////////////////////////////////////////////////
// Virtual Terminal
///////////////////////////////////////////////////////////////////////////////

type Cell struct {
	Char rune
	SGR  *types.SGR
}

type VirtualTerminal struct {
	buffer     [][]Cell
	width      int
	height     int
	cursorX    int
	cursorY    int
	maxCursorX int
	maxCursorY int
	currentSGR *types.SGR
	savedCursorX   int
	savedCursorY   int
	outputEncoding string
	useVGAColors   bool
	debugCursor    bool
	debugSGR       bool
}

func NewVirtualTerminal(width, height int, outputEncoding string, useVGAColors bool) *VirtualTerminal {
	defaultSGR := types.NewSGR()
	buffer := make([][]Cell, height)
	for i := range buffer {
		buffer[i] = make([]Cell, width)
		for j := range buffer[i] {
			buffer[i][j] = Cell{Char: 0x0, SGR: types.NewSGR()}
		}
	}

	return &VirtualTerminal{
		buffer:         buffer,
		width:          width,
		height:         height,
		cursorX:        0,
		cursorY:        0,
		maxCursorX:     0,
		maxCursorY:     0,
		currentSGR:     defaultSGR,
		outputEncoding: outputEncoding,
		useVGAColors:   useVGAColors,
		debugCursor:    false,
		debugSGR:       false,
	}
}
func (vt *VirtualTerminal) GetWidth() int {
	return vt.width
}

func (vt *VirtualTerminal) GetMaxCursorX() int {
	return vt.maxCursorX
}

func (vt *VirtualTerminal) GetMaxCursorY() int {
	return vt.maxCursorY
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

func (vt *VirtualTerminal) applyToken(token types.Token) error {
	switch token.Type {
	case types.TokenText:
		vt.writeText(token.Value)

	case types.TokenC0:
		vt.handleC0(token.C0Code)

	case types.TokenSGR:
		vt.handleSGR(token.Parameters)

	case types.TokenCSI:
		vt.handleCSI(token)
	}

	return nil
}


func (vt *VirtualTerminal) writeText(text string) {
	for _, r := range text {
		if vt.debugCursor {
			fmt.Printf("\nBefore writeText Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
		}

		if vt.cursorY < vt.height {
			vt.buffer[vt.cursorY][vt.cursorX] = Cell{
				Char: r,
				SGR:  vt.currentSGR.Copy(),
			}
			vt.cursorX++
			vt.maxCursorX = max(vt.maxCursorX, vt.cursorX)
			vt.maxCursorY = max(vt.maxCursorY, vt.cursorY)

			// Width to next line if we've reached the end
			if vt.cursorX >= vt.width {
				vt.cursorX = 0
				vt.cursorY++
				vt.maxCursorX = vt.width - 1
				vt.maxCursorY = max(vt.maxCursorY, vt.cursorY)
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

	switch code {
	case 0x00: // NUL
		vt.cursorX++
		if vt.cursorX >= vt.width {
			vt.cursorX = 0
			vt.cursorY++
			vt.maxCursorY = max(vt.maxCursorY, vt.cursorY)
		}

	case 0x09: // TAB
		vt.cursorX = ((vt.cursorX / 8) + 1) * 8
		if vt.cursorX >= vt.width {
			vt.cursorX = 0
			vt.cursorY++
			vt.maxCursorY = max(vt.maxCursorY, vt.cursorY)
		}

	case 0x0A: // LF (Line Feed)
		vt.cursorY++
		vt.maxCursorY = max(vt.maxCursorY, vt.cursorY)
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

	if vt.debugCursor {
		fmt.Printf("\nAfter handleC0 Cursor at (%d, %d)\n", vt.cursorX, vt.cursorY)
	}

	// vt.computeMaxCursorPosition()

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

			// vt.maxCursorX = vt.width - 1

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
				vt.buffer[y][x] = Cell{Char: 0x0, SGR: types.NewSGR()}
			}
		}
	case 1: // Clear from beginning of screen to cursor
		for y := 0; y <= vt.cursorY; y++ {
			for x := 0; x < vt.width; x++ {
				if y == vt.cursorY && x > vt.cursorX {
					break
				}
				vt.buffer[y][x] = Cell{Char: 0x0, SGR: types.NewSGR()}
			}
		}
	case 2: // Clear entire screen
		for y := 0; y < vt.height; y++ {
			for x := 0; x < vt.width; x++ {
				vt.buffer[y][x] = Cell{Char: 0x0, SGR: types.NewSGR()}
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
			vt.buffer[vt.cursorY][x] = Cell{Char: 0x0, SGR: types.NewSGR()}
		}
	case 1: // Clear from beginning of line to cursor
		for x := 0; x <= vt.cursorX; x++ {
			vt.buffer[vt.cursorY][x] = Cell{Char: 0x0, SGR: types.NewSGR()}
		}
	case 2: // Clear entire line
		for x := 0; x < vt.width; x++ {
			vt.buffer[vt.cursorY][x] = Cell{Char: 0x0, SGR: types.NewSGR()}
		}
	}
}

// ExportFlattenedANSI exports the buffer with optimized ANSI codes using differential encoding.
// Uses ExportSplitTextAndSequences and applies minimal SGR codes at the appropriate positions.
// The legacyMode ensures ANSI 1990 compatibility by using reset+rebuild
// when attributes need to be turned OFF, rather than using codes like [22m, [23m, etc.
func (vt *VirtualTerminal) ExportFlattenedANSI() string {
	lines := vt.ExportSplitTextAndSequences()
	var builder strings.Builder

	// Track the current SGR state across all lines for differential encoding
	var currentSGR *types.SGR = nil

	for _, line := range lines {
		var lineBuilder strings.Builder
		textRunes := []rune(line.Text)

		seqIndex := 0
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

			lineBuilder.WriteRune(r)
		}

		lineText := lineBuilder.String()
		if vt.outputEncoding == "utf8" {
			lineText = strings.ReplaceAll(lineText, "\x00", " ")
		}

		builder.WriteString(lineText)

		if vt.outputEncoding == "utf8" {
			builder.WriteString("\n")
		}
	}

	// Reset at the end only if not already at default state
	if !currentSGR.Equals(types.NewSGR()) {
		builder.WriteString("\x1b[0m")
	}

	return builder.String()
}

// ExportPlainText exports the buffer as plain text without ANSI codes
// Uses ExportSplitTextAndSequences and extracts only the text part
func (vt *VirtualTerminal) ExportPlainText() string {
	lines := vt.ExportSplitTextAndSequences()

	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString(line.Text)
		builder.WriteString("\n")
	}

	return builder.String()
}

// ExportSplitTextAndSequences exports the buffer as separate text and sequences
// Returns a slice of LineWithSequences, each containing the plain text and SGR changes
func (vt *VirtualTerminal) ExportSplitTextAndSequences() []types.LineWithSequences {
	result := []types.LineWithSequences{}
	var currentSGR *types.SGR = nil

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
			Text:      "",
			Sequences: []types.SGRSequence{},
		}

		var textBuilder strings.Builder

		for x := 0; x < vt.width; x++ {
			cell := vt.buffer[y][x]

			// fmt.Printf("Processing cell at (%d, %d): Char='%c' SGR='%v'\n", x, y, cell.Char, cell.SGR)

			// Detect SGR change
			if !cell.SGR.Equals(currentSGR) {
				line.Sequences = append(line.Sequences, types.SGRSequence{
					Position: x,
					SGR:      cell.SGR.Copy(),
				})
				currentSGR = cell.SGR.Copy()

				// fmt.Printf("  Detected SGR change at position %d: New SGR='%v'\n", x, cell.SGR)
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
