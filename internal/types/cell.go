package types

// Hyperlink represents an OSC 8 hyperlink with URL and optional parameters.
// OSC 8 format: ESC ] 8 ; params ; URL ST text ESC ] 8 ; ; ST
type Hyperlink struct {
	URL     string            // The target URL
	Params  map[string]string // Optional parameters (e.g., id=xyz)
	HoverFg *ColorValue       // Optional hover foreground color
	HoverBg *ColorValue       // Optional hover background color
}

// NewHyperlink creates a new Hyperlink with the given URL.
func NewHyperlink(url string) *Hyperlink {
	return &Hyperlink{
		URL:    url,
		Params: make(map[string]string),
	}
}

// Copy returns a deep copy of the Hyperlink.
func (h *Hyperlink) Copy() *Hyperlink {
	if h == nil {
		return nil
	}
	params := make(map[string]string)
	for k, v := range h.Params {
		params[k] = v
	}
	var fgCopy *ColorValue
	var bgCopy *ColorValue
	if h.HoverFg != nil {
		fg := *h.HoverFg
		fgCopy = &fg
	}
	if h.HoverBg != nil {
		bg := *h.HoverBg
		bgCopy = &bg
	}
	return &Hyperlink{
		URL:     h.URL,
		Params:  params,
		HoverFg: fgCopy,
		HoverBg: bgCopy,
	}
}

// Equals checks if two Hyperlinks are equal.
func (h *Hyperlink) Equals(other *Hyperlink) bool {
	if h == nil && other == nil {
		return true
	}
	if h == nil || other == nil {
		return false
	}
	if h.URL != other.URL {
		return false
	}
	if len(h.Params) != len(other.Params) {
		return false
	}
	for k, v := range h.Params {
		if other.Params[k] != v {
			return false
		}
	}
	if (h.HoverFg == nil) != (other.HoverFg == nil) {
		return false
	}
	if h.HoverFg != nil && *h.HoverFg != *other.HoverFg {
		return false
	}
	if (h.HoverBg == nil) != (other.HoverBg == nil) {
		return false
	}
	if h.HoverBg != nil && *h.HoverBg != *other.HoverBg {
		return false
	}
	return true
}

// Cell represents a single character in the virtual terminal buffer
// with its associated SGR (Select Graphic Rendition) styling.
type Cell struct {
	Char      rune
	SGR       *SGR
	Hyperlink *Hyperlink // Optional hyperlink (OSC 8)
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
	var hyperlinkCopy *Hyperlink
	if c.Hyperlink != nil {
		hyperlinkCopy = c.Hyperlink.Copy()
	}
	return Cell{
		Char:      c.Char,
		SGR:       sgrCopy,
		Hyperlink: hyperlinkCopy,
	}
}
