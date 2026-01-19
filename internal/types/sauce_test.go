package types

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestNewSauce(t *testing.T) {
	sauce := NewSauce(80, 25)

	if sauce.TInfo1 != 80 {
		t.Errorf("Expected width 80, got %d", sauce.TInfo1)
	}
	if sauce.TInfo2 != 25 {
		t.Errorf("Expected height 25, got %d", sauce.TInfo2)
	}
	if sauce.DataType != SauceDataTypeCharacter {
		t.Errorf("Expected DataType %d, got %d", SauceDataTypeCharacter, sauce.DataType)
	}
	if sauce.FileType != SauceFileTypeANSi {
		t.Errorf("Expected FileType %d, got %d", SauceFileTypeANSi, sauce.FileType)
	}
}

func TestSauceToBytes_Size(t *testing.T) {
	sauce := NewSauce(80, 25)
	bytes := sauce.ToBytes()

	if len(bytes) != SauceTotalSize {
		t.Errorf("Expected SAUCE total size %d, got %d", SauceTotalSize, len(bytes))
	}
}

func TestSauceToBytes_EOFMarker(t *testing.T) {
	sauce := NewSauce(80, 25)
	bytes := sauce.ToBytes()

	if bytes[0] != SauceEOF {
		t.Errorf("Expected EOF marker 0x%02X at offset 0, got 0x%02X", SauceEOF, bytes[0])
	}
}

func TestSauceToBytes_ID(t *testing.T) {
	sauce := NewSauce(80, 25)
	bytes := sauce.ToBytes()

	id := string(bytes[1:8])
	if id != SauceID {
		t.Errorf("Expected SAUCE ID '%s', got '%s'", SauceID, id)
	}
}

func TestSauceToBytes_Dimensions(t *testing.T) {
	tests := []struct {
		width  int
		height int
	}{
		{80, 25},
		{132, 50},
		{40, 100},
		{160, 60},
		{1, 1},
		{65535, 65535}, // Max uint16
	}

	for _, tc := range tests {
		sauce := NewSauce(tc.width, tc.height)
		bytes := sauce.ToBytes()

		gotWidth := binary.LittleEndian.Uint16(bytes[97:99])
		gotHeight := binary.LittleEndian.Uint16(bytes[99:101])

		if gotWidth != uint16(tc.width) {
			t.Errorf("Width: expected %d, got %d", tc.width, gotWidth)
		}
		if gotHeight != uint16(tc.height) {
			t.Errorf("Height: expected %d, got %d", tc.height, gotHeight)
		}
	}
}

func TestSauceToBytes_Title(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.Title = "Test Title"
	bytes := sauce.ToBytes()

	title := string(bytes[8:43])
	expected := "Test Title                         " // 35 chars, space-padded

	if title != expected {
		t.Errorf("Title: expected '%s', got '%s'", expected, title)
	}
}

func TestSauceToBytes_TitleTruncation(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.Title = "This is a very long title that exceeds 35 characters limit"
	bytes := sauce.ToBytes()

	title := string(bytes[8:43])
	expected := "This is a very long title that exce" // Truncated to 35 chars

	if title != expected {
		t.Errorf("Title: expected '%s', got '%s'", expected, title)
	}
}

func TestSauceToBytes_Author(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.Author = "TestAuthor"
	bytes := sauce.ToBytes()

	author := string(bytes[43:63])
	expected := "TestAuthor          " // 20 chars, space-padded

	if author != expected {
		t.Errorf("Author: expected '%s', got '%s'", expected, author)
	}
}

func TestSauceToBytes_Group(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.Group = "TestGroup"
	bytes := sauce.ToBytes()

	group := string(bytes[63:83])
	expected := "TestGroup           " // 20 chars, space-padded

	if group != expected {
		t.Errorf("Group: expected '%s', got '%s'", expected, group)
	}
}

func TestSauceToBytes_Date(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.Date = time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	bytes := sauce.ToBytes()

	date := string(bytes[83:91])
	expected := "20240315"

	if date != expected {
		t.Errorf("Date: expected '%s', got '%s'", expected, date)
	}
}

func TestSauceToBytes_FileSize(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.FileSize = 12345

	bytes := sauce.ToBytes()
	gotSize := binary.LittleEndian.Uint32(bytes[91:95])

	if gotSize != 12345 {
		t.Errorf("FileSize: expected 12345, got %d", gotSize)
	}
}

func TestSauceToBytes_DataTypeAndFileType(t *testing.T) {
	sauce := NewSauce(80, 25)
	bytes := sauce.ToBytes()

	if bytes[95] != SauceDataTypeCharacter {
		t.Errorf("DataType: expected %d, got %d", SauceDataTypeCharacter, bytes[95])
	}
	if bytes[96] != SauceFileTypeANSi {
		t.Errorf("FileType: expected %d, got %d", SauceFileTypeANSi, bytes[96])
	}
}

func TestSauceToBytes_TInfoS(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.TInfoS = "IBM VGA"
	bytes := sauce.ToBytes()

	tinfos := bytes[107:129]

	// Check first 7 bytes match "IBM VGA"
	if string(tinfos[:7]) != "IBM VGA" {
		t.Errorf("TInfoS: expected 'IBM VGA', got '%s'", string(tinfos[:7]))
	}

	// Check remaining bytes are null
	for i := 7; i < 22; i++ {
		if tinfos[i] != 0 {
			t.Errorf("TInfoS: expected null at position %d, got 0x%02X", i, tinfos[i])
		}
	}
}

func TestSauceSetDimensions(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.SetDimensions(132, 50)

	if sauce.TInfo1 != 132 {
		t.Errorf("Expected width 132, got %d", sauce.TInfo1)
	}
	if sauce.TInfo2 != 50 {
		t.Errorf("Expected height 50, got %d", sauce.TInfo2)
	}
}

func TestSauceICEColors(t *testing.T) {
	sauce := NewSauce(80, 25)

	// Initially disabled
	if sauce.HasICEColors() {
		t.Error("Expected iCE colors to be disabled initially")
	}

	// Enable iCE colors
	sauce.SetICEColors(true)
	if !sauce.HasICEColors() {
		t.Error("Expected iCE colors to be enabled")
	}
	if sauce.TFlags&SauceFlagNonBlink == 0 {
		t.Error("Expected TFlags to have NonBlink bit set")
	}

	// Disable iCE colors
	sauce.SetICEColors(false)
	if sauce.HasICEColors() {
		t.Error("Expected iCE colors to be disabled")
	}
}

func TestSauceToBytes_TFlags(t *testing.T) {
	sauce := NewSauce(80, 25)
	sauce.SetICEColors(true)

	bytes := sauce.ToBytes()

	if bytes[106] != SauceFlagNonBlink {
		t.Errorf("TFlags: expected 0x%02X, got 0x%02X", SauceFlagNonBlink, bytes[106])
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"test", 10, "test      "},
		{"test", 4, "test"},
		{"test", 2, "te"},
		{"", 5, "     "},
	}

	for _, tc := range tests {
		result := padRight(tc.input, tc.length)
		if result != tc.expected {
			t.Errorf("padRight(%q, %d): expected %q, got %q", tc.input, tc.length, tc.expected, result)
		}
	}
}

func TestPadRightNull(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected []byte
	}{
		{"test", 8, []byte{'t', 'e', 's', 't', 0, 0, 0, 0}},
		{"test", 4, []byte{'t', 'e', 's', 't'}},
		{"test", 2, []byte{'t', 'e'}},
		{"", 3, []byte{0, 0, 0}},
	}

	for _, tc := range tests {
		result := []byte(padRightNull(tc.input, tc.length))
		if len(result) != len(tc.expected) {
			t.Errorf("padRightNull(%q, %d): expected len %d, got len %d", tc.input, tc.length, len(tc.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("padRightNull(%q, %d): at position %d, expected 0x%02X, got 0x%02X", tc.input, tc.length, i, tc.expected[i], result[i])
			}
		}
	}
}

func TestFromBytes_RoundTrip(t *testing.T) {
	// Create a SAUCE, serialize it, then parse it back
	original := NewSauce(80, 25)
	original.Title = "Test Art"
	original.Author = "Artist"
	original.Group = "MyGroup"
	original.Date = time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	original.FileSize = 12345
	original.TInfoS = "IBM VGA"
	original.SetICEColors(true)

	bytes := original.ToBytes()
	parsed, err := FromBytes(bytes)

	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	if parsed.Title != "Test Art" {
		t.Errorf("Title mismatch: got %q, want %q", parsed.Title, "Test Art")
	}
	if parsed.Author != "Artist" {
		t.Errorf("Author mismatch: got %q, want %q", parsed.Author, "Artist")
	}
	if parsed.Group != "MyGroup" {
		t.Errorf("Group mismatch: got %q, want %q", parsed.Group, "MyGroup")
	}
	if parsed.TInfo1 != 80 || parsed.TInfo2 != 25 {
		t.Errorf("Dimensions mismatch: got %dx%d, want 80x25", parsed.TInfo1, parsed.TInfo2)
	}
	if parsed.FileSize != 12345 {
		t.Errorf("FileSize mismatch: got %d, want 12345", parsed.FileSize)
	}
	if parsed.TInfoS != "IBM VGA" {
		t.Errorf("TInfoS mismatch: got %q, want %q", parsed.TInfoS, "IBM VGA")
	}
	if !parsed.HasICEColors() {
		t.Error("Expected iCE colors to be enabled")
	}
	if parsed.DataType != SauceDataTypeCharacter {
		t.Errorf("DataType mismatch: got %d, want %d", parsed.DataType, SauceDataTypeCharacter)
	}
	if parsed.FileType != SauceFileTypeANSi {
		t.Errorf("FileType mismatch: got %d, want %d", parsed.FileType, SauceFileTypeANSi)
	}

	// Check date
	expectedDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !parsed.Date.Equal(expectedDate) {
		t.Errorf("Date mismatch: got %v, want %v", parsed.Date, expectedDate)
	}
}

func TestFromBytes_InvalidData(t *testing.T) {
	// Test with data too short
	_, err := FromBytes([]byte{0x1A, 'S', 'A', 'U', 'C', 'E'})
	if err == nil {
		t.Error("Expected error for short data")
	}

	// Test without EOF marker
	data := make([]byte, SauceTotalSize)
	copy(data[1:8], "SAUCE00")
	data[0] = 0x00 // Not 0x1A
	_, err = FromBytes(data)
	if err == nil {
		t.Error("Expected error for missing EOF marker")
	}

	// Test without SAUCE00 signature
	data2 := make([]byte, SauceTotalSize)
	data2[0] = SauceEOF
	copy(data2[1:8], "INVALID")
	_, err = FromBytes(data2)
	if err == nil {
		t.Error("Expected error for missing SAUCE00 signature")
	}
}

func TestFromBytes_AllDimensions(t *testing.T) {
	tests := []struct {
		width  int
		height int
	}{
		{80, 25},
		{132, 50},
		{40, 100},
		{160, 60},
		{1, 1},
		{65535, 65535}, // Max uint16
	}

	for _, tc := range tests {
		original := NewSauce(tc.width, tc.height)
		bytes := original.ToBytes()
		parsed, err := FromBytes(bytes)

		if err != nil {
			t.Errorf("FromBytes failed for %dx%d: %v", tc.width, tc.height, err)
			continue
		}

		if parsed.TInfo1 != uint16(tc.width) || parsed.TInfo2 != uint16(tc.height) {
			t.Errorf("Dimensions mismatch: got %dx%d, want %dx%d",
				parsed.TInfo1, parsed.TInfo2, tc.width, tc.height)
		}
	}
}

func TestParseSauceDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"20240315", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"19991231", time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)},
		{"20000101", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"invalid", time.Time{}},   // Invalid format
		{"2024031", time.Time{}},   // Too short
		{"202403155", time.Time{}}, // Too long (but only first 8 chars would be parsed if len check passed)
	}

	for _, tc := range tests {
		result := parseSauceDate(tc.input)
		if !result.Equal(tc.expected) {
			t.Errorf("parseSauceDate(%q): got %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestFromBytes_EmptyFields(t *testing.T) {
	// Test with empty string fields (should be trimmed to "")
	original := NewSauce(80, 25)
	// Leave Title, Author, Group, TInfoS empty

	bytes := original.ToBytes()
	parsed, err := FromBytes(bytes)

	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	if parsed.Title != "" {
		t.Errorf("Title should be empty, got %q", parsed.Title)
	}
	if parsed.Author != "" {
		t.Errorf("Author should be empty, got %q", parsed.Author)
	}
	if parsed.Group != "" {
		t.Errorf("Group should be empty, got %q", parsed.Group)
	}
	if parsed.TInfoS != "" {
		t.Errorf("TInfoS should be empty, got %q", parsed.TInfoS)
	}
}
