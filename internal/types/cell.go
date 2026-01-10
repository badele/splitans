package types

// Cell represents a single character in the virtual terminal buffer
// with its associated SGR (Select Graphic Rendition) styling.
type Cell struct {
	Char rune
	SGR  *SGR
}

// NewCell creates a new empty cell with default SGR styling.
func NewCell() Cell {
	return Cell{Char: 0x0, SGR: NewSGR()}
}

// Copy returns a deep copy of the cell.
func (c Cell) Copy() Cell {
	var sgrCopy *SGR
	if c.SGR != nil {
		sgrCopy = c.SGR.Copy()
	}
	return Cell{
		Char: c.Char,
		SGR:  sgrCopy,
	}
}
