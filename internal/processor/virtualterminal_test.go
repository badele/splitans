package processor

import (
	"strings"
	"testing"

	"github.com/badele/splitans/internal/types"
)

func TestWriteTextDoesNotInsertExtraBlankLineOnExactWidth(t *testing.T) {
	vt := NewVirtualTerminal(3, 10, "utf8", false, false)

	tokens := []types.Token{
		{Type: types.TokenText, Value: "abc"},
		{Type: types.TokenC0, C0Code: 0x0A}, // LF
		{Type: types.TokenText, Value: "def"},
	}

	if err := vt.ApplyTokens(tokens); err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}

	lines := vt.ExportSplitTextAndSequences()
	if got := len(lines); got != 2 {
		t.Fatalf("expected 2 lines, got %d", got)
	}

	if strings.TrimRight(lines[0].Text, " ") != "abc" {
		t.Fatalf("expected first line 'abc', got %q", lines[0].Text)
	}

	if strings.TrimRight(lines[1].Text, " ") != "def" {
		t.Fatalf("expected second line 'def', got %q", lines[1].Text)
	}
}

func TestPaste_BasicCopy(t *testing.T) {
	// Create a 10x5 workspace
	workspace := NewVirtualTerminal(10, 5, "utf8", false, false)

	// Create a 3x2 source with content
	source := NewVirtualTerminal(3, 2, "utf8", false, false)
	source.ApplyTokens([]types.Token{
		{Type: types.TokenText, Value: "ABC"},
		{Type: types.TokenC0, C0Code: 0x0A}, // LF
		{Type: types.TokenText, Value: "DEF"},
	})

	// Paste at origin
	if err := workspace.Paste(source, 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check content
	buf := workspace.GetBuffer()
	if buf[0][0].Char != 'A' || buf[0][1].Char != 'B' || buf[0][2].Char != 'C' {
		t.Errorf("first row mismatch: got %c%c%c", buf[0][0].Char, buf[0][1].Char, buf[0][2].Char)
	}
	if buf[1][0].Char != 'D' || buf[1][1].Char != 'E' || buf[1][2].Char != 'F' {
		t.Errorf("second row mismatch: got %c%c%c", buf[1][0].Char, buf[1][1].Char, buf[1][2].Char)
	}
}

func TestPaste_WithOffset(t *testing.T) {
	// Create a 10x5 workspace
	workspace := NewVirtualTerminal(10, 5, "utf8", false, false)

	// Create a 2x2 source
	source := NewVirtualTerminal(2, 2, "utf8", false, false)
	source.ApplyTokens([]types.Token{
		{Type: types.TokenText, Value: "XY"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "ZW"},
	})

	// Paste at offset (3, 2)
	if err := workspace.Paste(source, 3, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check content at offset position
	buf := workspace.GetBuffer()
	if buf[2][3].Char != 'X' || buf[2][4].Char != 'Y' {
		t.Errorf("row 2 mismatch at offset: got %c%c", buf[2][3].Char, buf[2][4].Char)
	}
	if buf[3][3].Char != 'Z' || buf[3][4].Char != 'W' {
		t.Errorf("row 3 mismatch at offset: got %c%c", buf[3][3].Char, buf[3][4].Char)
	}

	// Check that position (0,0) is still empty
	if buf[0][0].Char != 0 {
		t.Errorf("origin should be empty, got %c", buf[0][0].Char)
	}
}

func TestPaste_Clipping(t *testing.T) {
	// Create a 5x3 workspace
	workspace := NewVirtualTerminal(5, 3, "utf8", false, false)

	// Create a 4x4 source (larger than workspace when pasted at offset)
	source := NewVirtualTerminal(4, 4, "utf8", false, false)
	source.ApplyTokens([]types.Token{
		{Type: types.TokenText, Value: "1234"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "5678"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "9ABC"},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "DEFG"},
	})

	// Paste at (2, 1) - should clip horizontally and vertically
	if err := workspace.Paste(source, 2, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf := workspace.GetBuffer()
	// Row 1: should have "123" at positions 2,3,4 (clipped 4th char)
	if buf[1][2].Char != '1' || buf[1][3].Char != '2' || buf[1][4].Char != '3' {
		t.Errorf("row 1 clipping mismatch: got %c%c%c", buf[1][2].Char, buf[1][3].Char, buf[1][4].Char)
	}
	// Row 2: should have "567" (clipped)
	if buf[2][2].Char != '5' || buf[2][3].Char != '6' || buf[2][4].Char != '7' {
		t.Errorf("row 2 clipping mismatch: got %c%c%c", buf[2][2].Char, buf[2][3].Char, buf[2][4].Char)
	}
	// Row 3 and 4 should not exist (workspace is only 3 rows)
}

func TestPaste_NegativePosition(t *testing.T) {
	workspace := NewVirtualTerminal(10, 10, "utf8", false, false)
	source := NewVirtualTerminal(2, 2, "utf8", false, false)

	// Test negative X
	if err := workspace.Paste(source, -1, 0); err == nil {
		t.Error("expected error for negative X, got nil")
	}

	// Test negative Y
	if err := workspace.Paste(source, 0, -1); err == nil {
		t.Error("expected error for negative Y, got nil")
	}

	// Test both negative
	if err := workspace.Paste(source, -1, -1); err == nil {
		t.Error("expected error for both negative, got nil")
	}
}

func TestPaste_PreservesStyles(t *testing.T) {
	workspace := NewVirtualTerminal(10, 5, "utf8", false, false)

	// Create source with colored text
	source := NewVirtualTerminal(3, 1, "utf8", false, false)
	source.ApplyTokens([]types.Token{
		{Type: types.TokenSGR, Parameters: []string{"31"}}, // Red foreground
		{Type: types.TokenText, Value: "RED"},
	})

	// Paste
	if err := workspace.Paste(source, 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that style is preserved
	buf := workspace.GetBuffer()
	if buf[0][0].SGR == nil {
		t.Fatal("SGR should not be nil")
	}
	if buf[0][0].SGR.FgColor.Type != types.ColorStandard || buf[0][0].SGR.FgColor.Index != 1 {
		t.Errorf("expected red foreground (standard 1), got type=%d index=%d",
			buf[0][0].SGR.FgColor.Type, buf[0][0].SGR.FgColor.Index)
	}
}

func TestPaste_Transparency(t *testing.T) {
	// Create a 10x3 workspace with background content
	workspace := NewVirtualTerminal(10, 3, "utf8", false, false)
	workspace.ApplyTokens([]types.Token{
		{Type: types.TokenText, Value: "=========="},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "=========="},
		{Type: types.TokenC0, C0Code: 0x0A},
		{Type: types.TokenText, Value: "=========="},
	})

	// Create a 6x1 source with partial content using cursor positioning
	// Move cursor to column 3 (1-indexed), write "AB", leaving columns 1-2 and 5-6 as 0
	source := NewVirtualTerminal(6, 1, "utf8", false, false)
	source.ApplyTokens([]types.Token{
		{Type: types.TokenCSI, Raw: "\x1b[3G", Parameters: []string{"3"}}, // CHA: move to column 3
		{Type: types.TokenText, Value: "AB"},
	})

	// Verify source has transparent cells (Char == 0) at positions 0, 1, 4, 5
	srcBuf := source.GetBuffer()
	if srcBuf[0][0].Char != 0 || srcBuf[0][1].Char != 0 {
		t.Fatalf("source positions 0,1 should be 0 (transparent), got %d, %d", srcBuf[0][0].Char, srcBuf[0][1].Char)
	}
	if srcBuf[0][2].Char != 'A' || srcBuf[0][3].Char != 'B' {
		t.Fatalf("source positions 2,3 should be AB, got %c, %c", srcBuf[0][2].Char, srcBuf[0][3].Char)
	}
	if srcBuf[0][4].Char != 0 || srcBuf[0][5].Char != 0 {
		t.Fatalf("source positions 4,5 should be 0 (transparent), got %d, %d", srcBuf[0][4].Char, srcBuf[0][5].Char)
	}

	// Paste at (0, 1)
	if err := workspace.Paste(source, 0, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf := workspace.GetBuffer()
	// Row 1 should be: "==AB======" - only AB copied, rest preserved as '='
	// Source positions 0,1 are transparent (0), so workspace positions 0,1 stay '='
	// Source positions 2,3 have A,B, so workspace positions 2,3 get A,B
	// Source positions 4,5 are transparent (0), so workspace positions 4,5 stay '='
	expected := "==AB======"
	for i, ch := range expected {
		if buf[1][i].Char != ch {
			t.Errorf("position %d: expected %c, got %c", i, ch, buf[1][i].Char)
		}
	}

	// Row 0 and 2 should be unchanged
	for i := 0; i < 10; i++ {
		if buf[0][i].Char != '=' {
			t.Errorf("row 0, position %d: expected '=', got %c", i, buf[0][i].Char)
		}
		if buf[2][i].Char != '=' {
			t.Errorf("row 2, position %d: expected '=', got %c", i, buf[2][i].Char)
		}
	}
}

func TestGetContentBounds(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		tokens   []types.Token
		expected ContentBounds
	}{
		{
			name:   "Empty buffer",
			width:  10,
			height: 5,
			tokens: []types.Token{},
			expected: ContentBounds{
				Empty: true,
			},
		},
		{
			name:   "Content at origin",
			width:  10,
			height: 5,
			tokens: []types.Token{
				{Type: types.TokenText, Value: "ABC"},
			},
			expected: ContentBounds{
				MinX: 0, MaxX: 2, MinY: 0, MaxY: 0,
				Width: 3, Height: 1, Empty: false,
			},
		},
		{
			name:   "Content with offset",
			width:  10,
			height: 5,
			tokens: []types.Token{
				{Type: types.TokenCSI, Raw: "\x1b[2;3H", Parameters: []string{"2", "3"}}, // Move to row 2, col 3
				{Type: types.TokenText, Value: "XY"},
			},
			expected: ContentBounds{
				MinX: 2, MaxX: 3, MinY: 1, MaxY: 1,
				Width: 2, Height: 1, Empty: false,
			},
		},
		{
			name:   "Multi-line content",
			width:  10,
			height: 5,
			tokens: []types.Token{
				{Type: types.TokenCSI, Raw: "\x1b[2;4H", Parameters: []string{"2", "4"}}, // row 2, col 4 -> (3, 1) 0-indexed
				{Type: types.TokenText, Value: "AB"},                                     // positions 3,4 on row 1
				{Type: types.TokenCSI, Raw: "\x1b[4;3H", Parameters: []string{"4", "3"}}, // row 4, col 3 -> (2, 3) 0-indexed
				{Type: types.TokenText, Value: "CDEF"},                                   // positions 2,3,4,5 on row 3
			},
			expected: ContentBounds{
				MinX: 2, MaxX: 5, MinY: 1, MaxY: 3,
				Width: 4, Height: 3, Empty: false,
			},
		},
		{
			name:   "Only default spaces (ignored)",
			width:  10,
			height: 5,
			tokens: []types.Token{
				{Type: types.TokenText, Value: "   "},
			},
			expected: ContentBounds{Empty: true},
		},
		{
			name:   "Count styled spaces",
			width:  10,
			height: 5,
			tokens: []types.Token{
				{Type: types.TokenText, Value: " "},
				{Type: types.TokenText, Value: "A"},
				{Type: types.TokenText, Value: " "},
				{Type: types.TokenSGR, Parameters: []string{"41"}},
				{Type: types.TokenText, Value: " "},
				{Type: types.TokenSGR, Parameters: []string{"0"}},
			},
			expected: ContentBounds{
				MinX: 1, MaxX: 3, MinY: 0, MaxY: 0,
				Width: 3, Height: 1, Empty: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := NewVirtualTerminal(tt.width, tt.height, "utf8", false, false)
			vt.ApplyTokens(tt.tokens)

			bounds := vt.GetContentBounds()

			if bounds.Empty != tt.expected.Empty {
				t.Errorf("Empty: expected %v, got %v", tt.expected.Empty, bounds.Empty)
			}
			if !tt.expected.Empty {
				if bounds.MinX != tt.expected.MinX {
					t.Errorf("MinX: expected %d, got %d", tt.expected.MinX, bounds.MinX)
				}
				if bounds.MaxX != tt.expected.MaxX {
					t.Errorf("MaxX: expected %d, got %d", tt.expected.MaxX, bounds.MaxX)
				}
				if bounds.MinY != tt.expected.MinY {
					t.Errorf("MinY: expected %d, got %d", tt.expected.MinY, bounds.MinY)
				}
				if bounds.MaxY != tt.expected.MaxY {
					t.Errorf("MaxY: expected %d, got %d", tt.expected.MaxY, bounds.MaxY)
				}
				if bounds.Width != tt.expected.Width {
					t.Errorf("Width: expected %d, got %d", tt.expected.Width, bounds.Width)
				}
				if bounds.Height != tt.expected.Height {
					t.Errorf("Height: expected %d, got %d", tt.expected.Height, bounds.Height)
				}
			}
		})
	}
}
