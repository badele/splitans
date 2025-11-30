package exporter

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"splitans/tokenizer"
)

type TcellBuffer struct {
	screen       tcell.SimulationScreen
	style        tcell.Style
	cursorX      int
	cursorY      int
	width        int
	height       int
	decoder      *encoding.Decoder
	useCP437     bool
	debug        bool
	savedCursorX int
	savedCursorY int
}

func NewTcellBuffer(width, height int) (*TcellBuffer, error) {
	return NewTcellBufferWithEncoding(width, height, false)
}

func NewTcellBufferWithCP437(width, height int) (*TcellBuffer, error) {
	return NewTcellBufferWithEncoding(width, height, true)
}

func NewTcellBufferWithEncoding(width, height int, useCP437 bool) (*TcellBuffer, error) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("erreur initialisation écran: %w", err)
	}

	screen.SetSize(width, height)

	var decoder *encoding.Decoder
	if useCP437 {
		decoder = charmap.CodePage437.NewDecoder()
	}

	return &TcellBuffer{
		screen:   screen,
		style:    tcell.StyleDefault,
		cursorX:  0,
		cursorY:  0,
		width:    width,
		height:   height,
		decoder:  decoder,
		useCP437: useCP437,
	}, nil
}

func (tb *TcellBuffer) SetDebug(debug bool) {
	tb.debug = debug
}

func (tb *TcellBuffer) ApplyTokens(tokens []tokenizer.Token) error {
	for i, token := range tokens {
		if tb.debug {
			fmt.Fprintf(os.Stderr, "[Token %d] Type=%d Pos=%d Raw=%q Value=%q Params=%v\n",
				i, token.Type, token.Pos, token.Raw, token.Value, token.Parameters)
		}
		if err := tb.applyToken(token); err != nil {
			return err
		}
	}
	tb.screen.Show()
	return nil
}

func (tb *TcellBuffer) applyToken(token tokenizer.Token) error {
	switch token.Type {
	case tokenizer.TokenText:
		tb.writeText(token.Value)

	case tokenizer.TokenC0:
		tb.handleC0(token.C0Code)

	case tokenizer.TokenSGR:
		tb.handleSGR(token.Parameters)

	case tokenizer.TokenCSI:
		tb.handleCSI(token)
	}

	return nil
}

func (tb *TcellBuffer) writeText(text string) {
	if tb.debug {
		fmt.Fprintf(os.Stderr, "  [writeText] Cursor=(%d,%d) Text=%q (len=%d)\n",
			tb.cursorX, tb.cursorY, text, len(text))
	}

	// Convertir depuis CP437 si nécessaire
	if tb.useCP437 && tb.decoder != nil {
		// Convertir les bytes (qui sont en CP437) vers UTF-8
		srcBytes := []byte(text)
		dstBytes := make([]byte, len(srcBytes)*4) // UTF-8 peut prendre jusqu'à 4 bytes par caractère

		nDst, _, err := tb.decoder.Transform(dstBytes, srcBytes, true)
		if err == nil {
			text = string(dstBytes[:nDst])
			if tb.debug {
				fmt.Fprintf(os.Stderr, "  [writeText] Decoded=%q (len=%d)\n", text, len(text))
			}
		}
	}

	for _, r := range text {
		if tb.cursorX >= tb.width {
			// go to next line
			if tb.debug {
				fmt.Fprintf(os.Stderr, "  [writeText] WRAP! Cursor=(%d,%d) -> (0,%d)\n",
					tb.cursorX, tb.cursorY, tb.cursorY+1)
			}
			tb.cursorX = 0
			tb.cursorY++
			if tb.cursorY >= tb.height {
				tb.cursorY = tb.height - 1
			}
		}

		tb.screen.SetContent(tb.cursorX, tb.cursorY, r, nil, tb.style)
		tb.cursorX++
	}

	if tb.debug {
		fmt.Fprintf(os.Stderr, "  [writeText] After: Cursor=(%d,%d)\n", tb.cursorX, tb.cursorY)
	}
}

func (tb *TcellBuffer) handleC0(code byte) {
	switch code {
	case 0x09: // TAB
		// Avancer à la prochaine position de tabulation (multiple de 8)
		tb.cursorX = ((tb.cursorX / 8) + 1) * 8
		if tb.cursorX >= tb.width {
			tb.cursorX = 0
			tb.cursorY++
		}

	case 0x0A: // LF (Line Feed)
		tb.cursorY++
		if tb.cursorY >= tb.height {
			tb.cursorY = tb.height - 1
		}

	case 0x0D: // CR (Carriage Return)
		tb.cursorX = 0

	case 0x08: // BS (Backspace)
		if tb.cursorX > 0 {
			tb.cursorX--
		}
	}
}

func (tb *TcellBuffer) handleSGR(params []string) {
	if len(params) == 0 {
		params = []string{"0"}
	}

	for i := 0; i < len(params); i++ {
		param, _ := strconv.Atoi(params[i])

		switch param {
		case 0: // Reset
			tb.style = tcell.StyleDefault

		case 1: // Bold
			tb.style = tb.style.Bold(true)

		case 2: // Dim
			tb.style = tb.style.Dim(true)

		case 3: // Italic
			tb.style = tb.style.Italic(true)

		case 4: // Underline
			tb.style = tb.style.Underline(true)

		case 5, 6: // Blink
			tb.style = tb.style.Blink(true)

		case 7: // Reverse
			tb.style = tb.style.Reverse(true)

		case 22: // Normal intensity
			tb.style = tb.style.Bold(false).Dim(false)

		case 23: // Italic off
			tb.style = tb.style.Italic(false)

		case 24: // Underline off
			tb.style = tb.style.Underline(false)

		case 25: // Blink off
			tb.style = tb.style.Blink(false)

		case 27: // Reverse off
			tb.style = tb.style.Reverse(false)

		// Foreground colors (30-37)
		case 30:
			tb.style = tb.style.Foreground(tcell.ColorBlack)
		case 31:
			tb.style = tb.style.Foreground(tcell.ColorMaroon)
		case 32:
			tb.style = tb.style.Foreground(tcell.ColorGreen)
		case 33:
			tb.style = tb.style.Foreground(tcell.ColorOlive)
		case 34:
			tb.style = tb.style.Foreground(tcell.ColorNavy)
		case 35:
			tb.style = tb.style.Foreground(tcell.ColorPurple)
		case 36:
			tb.style = tb.style.Foreground(tcell.ColorTeal)
		case 37:
			tb.style = tb.style.Foreground(tcell.ColorSilver)

		case 39: // Default foreground
			tb.style = tb.style.Foreground(tcell.ColorDefault)

		// Background colors (40-47)
		case 40:
			tb.style = tb.style.Background(tcell.ColorBlack)
		case 41:
			tb.style = tb.style.Background(tcell.ColorMaroon)
		case 42:
			tb.style = tb.style.Background(tcell.ColorGreen)
		case 43:
			tb.style = tb.style.Background(tcell.ColorOlive)
		case 44:
			tb.style = tb.style.Background(tcell.ColorNavy)
		case 45:
			tb.style = tb.style.Background(tcell.ColorPurple)
		case 46:
			tb.style = tb.style.Background(tcell.ColorTeal)
		case 47:
			tb.style = tb.style.Background(tcell.ColorSilver)

		case 49: // Default background
			tb.style = tb.style.Background(tcell.ColorDefault)

		// Bright foreground colors (90-97)
		case 90:
			tb.style = tb.style.Foreground(tcell.ColorGray)
		case 91:
			tb.style = tb.style.Foreground(tcell.ColorRed)
		case 92:
			tb.style = tb.style.Foreground(tcell.ColorLime)
		case 93:
			tb.style = tb.style.Foreground(tcell.ColorYellow)
		case 94:
			tb.style = tb.style.Foreground(tcell.ColorBlue)
		case 95:
			tb.style = tb.style.Foreground(tcell.ColorFuchsia)
		case 96:
			tb.style = tb.style.Foreground(tcell.ColorAqua)
		case 97:
			tb.style = tb.style.Foreground(tcell.ColorWhite)

		// Bright background colors (100-107)
		case 100:
			tb.style = tb.style.Background(tcell.ColorGray)
		case 101:
			tb.style = tb.style.Background(tcell.ColorRed)
		case 102:
			tb.style = tb.style.Background(tcell.ColorLime)
		case 103:
			tb.style = tb.style.Background(tcell.ColorYellow)
		case 104:
			tb.style = tb.style.Background(tcell.ColorBlue)
		case 105:
			tb.style = tb.style.Background(tcell.ColorFuchsia)
		case 106:
			tb.style = tb.style.Background(tcell.ColorAqua)
		case 107:
			tb.style = tb.style.Background(tcell.ColorWhite)

		// 256 colors et RGB (38;5;n et 48;5;n)
		case 38, 48:
			if i+2 < len(params) && params[i+1] == "5" {
				colorIndex, _ := strconv.Atoi(params[i+2])
				color := tcell.Color(colorIndex)
				if param == 38 {
					tb.style = tb.style.Foreground(color)
				} else {
					tb.style = tb.style.Background(color)
				}
				i += 2
			}
		}
	}
}

func (tb *TcellBuffer) handleCSI(token tokenizer.Token) {
	if len(token.Raw) == 0 {
		return
	}

	lastChar := token.Raw[len(token.Raw)-1]

	switch lastChar {
	case 'H', 'f': // Cursor Position
		row, col := 1, 1
		if len(token.Parameters) > 0 {
			row, _ = strconv.Atoi(token.Parameters[0])
		}
		if len(token.Parameters) > 1 {
			col, _ = strconv.Atoi(token.Parameters[1])
		}
		oldX, oldY := tb.cursorX, tb.cursorY
		tb.cursorY = row - 1
		tb.cursorX = col - 1
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI H] Cursor Position: (%d,%d) -> (%d,%d) [row=%d,col=%d]\n",
				oldX, oldY, tb.cursorX, tb.cursorY, row, col)
		}

	case 'A': // Cursor Up
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		oldY := tb.cursorY
		tb.cursorY -= n
		if tb.cursorY < 0 {
			tb.cursorY = 0
		}
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI A] Cursor Up %d: y=%d -> %d\n", n, oldY, tb.cursorY)
		}

	case 'B': // Cursor Down
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		oldY := tb.cursorY
		tb.cursorY += n
		if tb.cursorY >= tb.height {
			tb.cursorY = tb.height - 1
		}
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI B] Cursor Down %d: y=%d -> %d\n", n, oldY, tb.cursorY)
		}

	case 'C': // Cursor Forward
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		oldX := tb.cursorX
		tb.cursorX += n
		if tb.cursorX >= tb.width {
			tb.cursorX = tb.width - 1
		}
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI C] Cursor Forward %d: x=%d -> %d\n", n, oldX, tb.cursorX)
		}

	case 'D': // Cursor Backward
		n := 1
		if len(token.Parameters) > 0 {
			n, _ = strconv.Atoi(token.Parameters[0])
		}
		oldX := tb.cursorX
		tb.cursorX -= n
		if tb.cursorX < 0 {
			tb.cursorX = 0
		}
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI D] Cursor Backward %d: x=%d -> %d\n", n, oldX, tb.cursorX)
		}

	case 'J': // Erase Display
		mode := 0
		if len(token.Parameters) > 0 {
			mode, _ = strconv.Atoi(token.Parameters[0])
		}
		tb.eraseDisplay(mode)

	case 'K': // Erase Line
		mode := 0
		if len(token.Parameters) > 0 {
			mode, _ = strconv.Atoi(token.Parameters[0])
		}
		tb.eraseLine(mode)

	case 's': // Save Cursor Position
		tb.savedCursorX = tb.cursorX
		tb.savedCursorY = tb.cursorY
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI s] Save Cursor: (%d,%d)\n", tb.savedCursorX, tb.savedCursorY)
		}

	case 'u': // Restore Cursor Position
		oldX, oldY := tb.cursorX, tb.cursorY
		tb.cursorX = tb.savedCursorX
		tb.cursorY = tb.savedCursorY
		if tb.debug {
			fmt.Fprintf(os.Stderr, "  [CSI u] Restore Cursor: (%d,%d) -> (%d,%d)\n",
				oldX, oldY, tb.cursorX, tb.cursorY)
		}

	default:
		fmt.Fprintf(os.Stderr, "  [WARNING] Unsupported CSI sequence: %q (command: %c, params: %v)\n",
			token.Raw, lastChar, token.Parameters)
	}
}

func (tb *TcellBuffer) eraseDisplay(mode int) {
	switch mode {
	case 0: // Clear from cursor to end of screen
		for y := tb.cursorY; y < tb.height; y++ {
			for x := 0; x < tb.width; x++ {
				if y == tb.cursorY && x < tb.cursorX {
					continue
				}
				tb.screen.SetContent(x, y, ' ', nil, tb.style)
			}
		}
	case 1: // Clear from beginning of screen to cursor
		for y := 0; y <= tb.cursorY; y++ {
			for x := 0; x < tb.width; x++ {
				if y == tb.cursorY && x > tb.cursorX {
					break
				}
				tb.screen.SetContent(x, y, ' ', nil, tb.style)
			}
		}
	case 2: // CLear entire screen
		tb.screen.Clear()
	}
}

func (tb *TcellBuffer) eraseLine(mode int) {
	switch mode {
	case 0: // clear from cursor to end of line
		for x := tb.cursorX; x < tb.width; x++ {
			tb.screen.SetContent(x, tb.cursorY, ' ', nil, tb.style)
		}
	case 1: // Clear from beginning of line to cursor
		for x := 0; x <= tb.cursorX; x++ {
			tb.screen.SetContent(x, tb.cursorY, ' ', nil, tb.style)
		}
	case 2: // Cear entire line
		for x := 0; x < tb.width; x++ {
			tb.screen.SetContent(x, tb.cursorY, ' ', nil, tb.style)
		}
	}
}

func (tb *TcellBuffer) GetPlainText() string {
	var builder strings.Builder

	for y := 0; y < tb.height; y++ {
		lineHasContent := false
		lineText := ""

		for x := 0; x < tb.width; x++ {
			mainc, _, _, _ := tb.screen.GetContent(x, y)
			if mainc != 0 && mainc != ' ' {
				lineHasContent = true
			}
			// Convert 0 (empty cell) to space to avoid null bytes in output
			if mainc == 0 {
				lineText += " "
			} else {
				lineText += string(mainc)
			}
		}

		if lineHasContent {
			// lineText = strings.TrimRight(lineText, " ")
			builder.WriteString(lineText)
			if y < tb.height-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
	// return strings.TrimRight(builder.String(), "\n")
}

func (tb *TcellBuffer) GetActualWidth() int {
	maxWidth := 0

	for y := 0; y < tb.height; y++ {
		for x := tb.width - 1; x >= 0; x-- {
			mainc, _, _, _ := tb.screen.GetContent(x, y)
			if mainc != 0 && mainc != ' ' {
				if x+1 > maxWidth {
					maxWidth = x + 1
				}
				break
			}
		}
	}

	return maxWidth
}

func (tb *TcellBuffer) GetActualHeight() int {
	for y := tb.height - 1; y >= 0; y-- {
		for x := 0; x < tb.width; x++ {
			mainc, _, _, _ := tb.screen.GetContent(x, y)
			if mainc != 0 && mainc != ' ' {
				return y + 1
			}
		}
	}
	return 0
}

func (tb *TcellBuffer) GetDimensions() (int, int) {
	return tb.width, tb.height
}

func (tb *TcellBuffer) Close() {
	tb.screen.Fini()
}

func ExportToPlainText(tokens []tokenizer.Token, outputPath string) error {
	return ExportToPlainTextWithCP437(tokens, outputPath, false)
}

func ExportToPlainTextWithCP437Support(tokens []tokenizer.Token, outputPath string) error {
	return ExportToPlainTextWithCP437(tokens, outputPath, true)
}

func ExportToPlainTextWithCP437(tokens []tokenizer.Token, outputPath string, useCP437 bool) error {
	buffer, err := NewTcellBufferWithEncoding(80, 1000, useCP437)
	if err != nil {
		return fmt.Errorf("erreur création buffer: %w", err)
	}
	defer buffer.Close()

	if err := buffer.ApplyTokens(tokens); err != nil {
		return fmt.Errorf("erreur application tokens: %w", err)
	}

	plainText := buffer.GetPlainText()

	if err := os.WriteFile(outputPath, []byte(plainText), 0644); err != nil {
		return fmt.Errorf("erreur écriture fichier: %w", err)
	}

	return nil
}

func ExportToPlainTextWithInfo(tokens []tokenizer.Token, outputPath string) (width, height int, err error) {
	return ExportToPlainTextWithInfoAndCP437(tokens, outputPath, true) // Désactivé temporairement pour debug
}

func ExportToPlainTextWithInfoAndCP437(tokens []tokenizer.Token, outputPath string, useCP437 bool) (width, height int, err error) {
	return ExportToPlainTextWithInfoAndDebug(tokens, outputPath, useCP437, false)
}

func ExportToPlainTextWithInfoAndDebug(tokens []tokenizer.Token, outputPath string, useCP437, debug bool) (width, height int, err error) {
	buffer, err := NewTcellBufferWithEncoding(80, 1000, useCP437)
	if err != nil {
		return 0, 0, fmt.Errorf("erreur création buffer: %w", err)
	}
	defer buffer.Close()

	buffer.SetDebug(debug)

	if err := buffer.ApplyTokens(tokens); err != nil {
		return 0, 0, fmt.Errorf("erreur application tokens: %w", err)
	}

	width = buffer.GetActualWidth()
	height = buffer.GetActualHeight()

	plainText := buffer.GetPlainText()

	if err := os.WriteFile(outputPath, []byte(plainText), 0644); err != nil {
		return 0, 0, fmt.Errorf("erreur écriture fichier: %w", err)
	}

	return width, height, nil
}

// DisplayPlainText displays the plain text content to stdout
func DisplayPlainText(tokens []tokenizer.Token) error {
	buffer, err := NewTcellBufferWithEncoding(80, 1000, true)
	if err != nil {
		return fmt.Errorf("error creating buffer: %w", err)
	}
	defer buffer.Close()

	if err := buffer.ApplyTokens(tokens); err != nil {
		return fmt.Errorf("error applying tokens: %w", err)
	}

	plainText := buffer.GetPlainText()
	fmt.Println(plainText)

	return nil
}

// styleDiff compares two styles and returns only the changes
func styleDiff(oldStyle, newStyle tcell.Style) string {
	var changes []string

	oldFg, oldBg, oldAttrs := oldStyle.Decompose()
	newFg, newBg, newAttrs := newStyle.Decompose()

	// Check if this is a complete reset
	if newFg == tcell.ColorDefault && newBg == tcell.ColorDefault && newAttrs == tcell.AttrNone {
		// Only return RESET if it's different from the old style
		if oldFg != tcell.ColorDefault || oldBg != tcell.ColorDefault || oldAttrs != tcell.AttrNone {
			return "RESET"
		}
		return "" // No change, already in default state
	}

	// Foreground color changed
	if newFg != oldFg {
		if newFg == tcell.ColorDefault {
			changes = append(changes, "FGD")
		} else {
			changes = append(changes, fmt.Sprintf("FG%s", colorToString(newFg)))
		}
	}

	// Background color changed
	if newBg != oldBg {
		if newBg == tcell.ColorDefault {
			changes = append(changes, "BGD")
		} else {
			changes = append(changes, fmt.Sprintf("BG%s", colorToString(newBg)))
		}
	}

	// Check attribute changes
	type attrInfo struct {
		mask tcell.AttrMask
		name string
	}

	attrs := []attrInfo{
		{tcell.AttrBold, "BD"},
		{tcell.AttrDim, "DM"},
		{tcell.AttrItalic, "IC"},
		{tcell.AttrUnderline, "UE"},
		{tcell.AttrBlink, "BK"},
		{tcell.AttrReverse, "RE"},
	}

	for _, attr := range attrs {
		oldHas := (oldAttrs & attr.mask) != 0
		newHas := (newAttrs & attr.mask) != 0

		if oldHas != newHas {
			if newHas {
				changes = append(changes, fmt.Sprintf("%s1", attr.name))
			} else {
				changes = append(changes, fmt.Sprintf("%s0", attr.name))
			}
		}
	}

	if len(changes) == 0 {
		return "" // No changes
	}

	return strings.Join(changes, ", ")
}

// colorToString converts a tcell.Color to a readable string
func colorToString(color tcell.Color) string {
	colorNames := map[tcell.Color]string{
		tcell.ColorBlack:   "BK",
		tcell.ColorMaroon:  "MN",
		tcell.ColorGreen:   "GN",
		tcell.ColorOlive:   "OE",
		tcell.ColorNavy:    "NY",
		tcell.ColorPurple:  "PE",
		tcell.ColorTeal:    "TL",
		tcell.ColorSilver:  "SR",
		tcell.ColorGray:    "GY",
		tcell.ColorRed:     "RD",
		tcell.ColorLime:    "LE",
		tcell.ColorYellow:  "YW",
		tcell.ColorBlue:    "BE",
		tcell.ColorFuchsia: "FA",
		tcell.ColorAqua:    "AA",
		tcell.ColorWhite:   "WE",
	}

	if name, ok := colorNames[color]; ok {
		return name
	}

	// For 256 colors or RGB, return the numeric value
	return fmt.Sprintf("%d", color)
}
