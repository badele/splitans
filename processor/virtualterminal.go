package processor

import (
	"fmt"
	"strconv"
	"strings"

	"splitans/types"
)

///////////////////////////////////////////////////////////////////////////////
// Virtual Terminal
///////////////////////////////////////////////////////////////////////////////

type Cell struct {
	Char rune
	SGR  *types.SGR
}

type VirtualTerminal struct {
	buffer         [][]Cell
	width          int
	height         int
	cursorX        int
	cursorY        int
	currentSGR     *types.SGR
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
		currentSGR:     defaultSGR,
		outputEncoding: outputEncoding,
		useVGAColors:   useVGAColors,
		debugCursor:    false,
		debugSGR:       false,
	}
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

			// Wrap to next line if we've reached the end
			if vt.cursorX >= vt.width {
				vt.cursorX = 0
				vt.cursorY++
				if vt.cursorY >= vt.height {
					vt.cursorY = vt.height - 1
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

	switch code {
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

}

func (vt *VirtualTerminal) handleSGR(params []string) {
	if vt.debugSGR {
		fmt.Printf("\nBefore handleSGR Current SGR: '%v', New params: %v\n", vt.currentSGR, params)
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

	case 'C': // Cursor Forward
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorX += n
		if vt.cursorX >= vt.width {
			vt.cursorX = vt.width - 1
		}

	case 'D': // Cursor Backward
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		vt.cursorX -= n
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}

	case 'H', 'f': // Cursor Position
		if vt.debugCursor {

			fmt.Printf("Before CSI Cursor Position with params: %v, Cusor at (%d, %d) \n", token.Parameters, vt.cursorX, vt.cursorY)
		}
		row, col := 0, 0
		if len(token.Parameters) > 0 {
			row, _ = strconv.Atoi(token.Parameters[0])
		}
		if len(token.Parameters) > 1 {
			col, _ = strconv.Atoi(token.Parameters[1])
		}
		vt.cursorY = max(row, 1) - 1
		vt.cursorX = max(col, 1) - 1
		if vt.debugCursor {
			fmt.Printf("After CSI Cursor Position with params: %v, Cusor at (%d, %d) \n", token.Parameters, vt.cursorX, vt.cursorY)
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

// ExportFlattenedANSI exports the buffer with ANSI codes
// Uses ExportSplitTextAndSequences and applies SGR codes at the appropriate positions
func (vt *VirtualTerminal) ExportFlattenedANSI() string {
	lines := vt.ExportSplitTextAndSequences()
	var builder strings.Builder

	for _, line := range lines {
		var lineBuilder strings.Builder
		textRunes := []rune(line.Text)

		seqIndex := 0
		for i, r := range textRunes {
			// Check if there's a SGR change at this position
			if seqIndex < len(line.Sequences) && line.Sequences[seqIndex].Position == i {
				// Always reset first, then apply new style
				// This ensures attributes that turn from true to false are properly reset
				lineBuilder.WriteString("\x1b[0m")
				if !line.Sequences[seqIndex].SGR.Equals(types.NewSGR()) {
					lineBuilder.WriteString(line.Sequences[seqIndex].SGR.ToANSI(vt.useVGAColors))
				}
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

	// Reset at the end
	builder.WriteString("\x1b[0m")

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

	for y := 0; y < vt.height; y++ {
		// Check if line has content
		hasContent := false
		for x := 0; x < vt.width; x++ {
			if vt.buffer[y][x].Char != 0x0 {
				hasContent = true
				break
			}
		}

		if !hasContent {
			continue
		}

		line := types.LineWithSequences{
			Text:      "",
			Sequences: []types.SGRChange{},
		}

		currentSGR := types.NewSGR()
		charPosition := 0

		for x := 0; x < vt.width; x++ {
			cell := vt.buffer[y][x]

			// Detect SGR change
			if !cell.SGR.Equals(currentSGR) {
				line.Sequences = append(line.Sequences, types.SGRChange{
					Position: charPosition,
					SGR:      cell.SGR.Copy(),
				})
				currentSGR = cell.SGR.Copy()
			}

			// Add character to text (replace 0x0 with space)
			char := cell.Char
			if char == 0x0 {
				char = ' '
			}
			line.Text += string(char)
			charPosition++
		}

		result = append(result, line)
	}

	return result
}
