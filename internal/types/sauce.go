package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

// SAUCE record constants
const (
	SauceRecordSize = 128 // Size of SAUCE record (without EOF marker)
	SauceTotalSize  = 129 // Total size including EOF marker
	SauceEOF        = 0x1A
	SauceID         = "SAUCE00"

	// DataType values
	SauceDataTypeNone       = 0
	SauceDataTypeCharacter  = 1 // Character-based (ANSI, ASCII, etc.)
	SauceDataTypeBitmap     = 2
	SauceDataTypeVector     = 3
	SauceDataTypeAudio      = 4
	SauceDataTypeBinaryText = 5
	SauceDataTypeXBin       = 6
	SauceDataTypeArchive    = 7
	SauceDataTypeExecutable = 8

	// FileType values for DataType=Character (1)
	SauceFileTypeASCII      = 0
	SauceFileTypeANSi       = 1
	SauceFileTypeANSiMation = 2
	SauceFileTypeRIPScript  = 3
	SauceFileTypePCBoard    = 4
	SauceFileTypeAvatar     = 5
	SauceFileTypeHTML       = 6
	SauceFileTypeSource     = 7
	SauceFileTypeTundraDraw = 8
)

// TFlags bit flags
const (
	SauceFlagNonBlink      = 0x01 // iCE colors (use background bright colors instead of blink)
	SauceFlagLetterSpacing = 0x06 // Bits 1-2: letter spacing
	SauceFlagAspectRatio   = 0x18 // Bits 3-4: aspect ratio
)

// ============================================================================
// EXPORTED
// ============================================================================

// Sauce represents a SAUCE (Standard Architecture for Universal Comment Extensions) record.
// SAUCE is a 128-byte metadata block appended to the end of ANSI art files.
type Sauce struct {
	Title    string    // 35 characters max - Title of the file
	Author   string    // 20 characters max - Author/creator name
	Group    string    // 20 characters max - Group/organization name
	Date     time.Time // Creation date (stored as YYYYMMDD)
	FileSize uint32    // Size of the file before SAUCE record
	DataType uint8     // Type of data (1 = Character for ANSI)
	FileType uint8     // File type (1 = ANSi for DataType=Character)
	TInfo1   uint16    // Width in characters
	TInfo2   uint16    // Height in lines
	TInfo3   uint16    // Reserved (0 for ANSI)
	TInfo4   uint16    // Reserved (0 for ANSI)
	Comments uint8     // Number of comment lines (0-255)
	TFlags   uint8     // Flags (iCE colors, letter spacing, aspect ratio)
	TInfoS   string    // 22 characters max - Font name/info
}

// NewSauce creates a new SAUCE record with default values for ANSI files.
// Width and height are stored in TInfo1 and TInfo2 respectively.
func NewSauce(width, height int) *Sauce {
	return &Sauce{
		Title:    "",
		Author:   "",
		Group:    "",
		Date:     time.Now(),
		FileSize: 0,
		DataType: SauceDataTypeCharacter,
		FileType: SauceFileTypeANSi,
		TInfo1:   uint16(width),
		TInfo2:   uint16(height),
		TInfo3:   0,
		TInfo4:   0,
		Comments: 0,
		TFlags:   0,
		TInfoS:   "",
	}
}

// FromBytes parses a 129-byte slice (EOF + 128-byte record) into a Sauce struct.
// Returns an error if the data is invalid (wrong size, missing SAUCE00 signature).
func FromBytes(data []byte) (*Sauce, error) {
	if len(data) < SauceTotalSize {
		return nil, fmt.Errorf("SAUCE record too short: %d bytes (expected %d)", len(data), SauceTotalSize)
	}

	// Validate EOF marker
	if data[0] != SauceEOF {
		return nil, fmt.Errorf("invalid SAUCE: missing EOF marker (0x1A)")
	}

	// Validate SAUCE00 signature
	if string(data[1:8]) != SauceID {
		return nil, fmt.Errorf("invalid SAUCE: missing SAUCE00 signature")
	}

	sauce := &Sauce{
		Title:    strings.TrimRight(string(data[8:43]), " \x00"),
		Author:   strings.TrimRight(string(data[43:63]), " \x00"),
		Group:    strings.TrimRight(string(data[63:83]), " \x00"),
		Date:     parseSauceDate(string(data[83:91])),
		FileSize: binary.LittleEndian.Uint32(data[91:95]),
		DataType: data[95],
		FileType: data[96],
		TInfo1:   binary.LittleEndian.Uint16(data[97:99]),
		TInfo2:   binary.LittleEndian.Uint16(data[99:101]),
		TInfo3:   binary.LittleEndian.Uint16(data[101:103]),
		TInfo4:   binary.LittleEndian.Uint16(data[103:105]),
		Comments: data[105],
		TFlags:   data[106],
		TInfoS:   strings.TrimRight(string(data[107:129]), " \x00"),
	}

	return sauce, nil
}

// FromBytesWithEncoding parses a 129-byte slice and converts text fields from source encoding.
// This is used when the SAUCE record was extracted before encoding conversion to preserve
// binary fields (FileSize, TInfo1-4, etc.) while still converting text fields.
func FromBytesWithEncoding(data []byte, sourceEncoding string) (*Sauce, error) {
	if len(data) < SauceTotalSize {
		return nil, fmt.Errorf("SAUCE record too short: %d bytes (expected %d)", len(data), SauceTotalSize)
	}

	// Validate EOF marker
	if data[0] != SauceEOF {
		return nil, fmt.Errorf("invalid SAUCE: missing EOF marker (0x1A)")
	}

	// Validate SAUCE00 signature
	if string(data[1:8]) != SauceID {
		return nil, fmt.Errorf("invalid SAUCE: missing SAUCE00 signature")
	}

	sauce := &Sauce{
		Title:    convertField(data[8:43], sourceEncoding),
		Author:   convertField(data[43:63], sourceEncoding),
		Group:    convertField(data[63:83], sourceEncoding),
		Date:     parseSauceDate(string(data[83:91])),
		FileSize: binary.LittleEndian.Uint32(data[91:95]),
		DataType: data[95],
		FileType: data[96],
		TInfo1:   binary.LittleEndian.Uint16(data[97:99]),
		TInfo2:   binary.LittleEndian.Uint16(data[99:101]),
		TInfo3:   binary.LittleEndian.Uint16(data[101:103]),
		TInfo4:   binary.LittleEndian.Uint16(data[103:105]),
		Comments: data[105],
		TFlags:   data[106],
		TInfoS:   convertField(data[107:129], sourceEncoding),
	}

	return sauce, nil
}

// ToBytes serializes the SAUCE record to a 129-byte slice (EOF + 128-byte record).
// Format:
//
//	Offset  Size  Field
//	0       1     EOF (0x1A)
//	1       5     "SAUCE"
//	6       2     "00"
//	8       35    Title
//	43      20    Author
//	63      20    Group
//	83      8     Date (YYYYMMDD)
//	91      4     FileSize (little-endian)
//	95      1     DataType
//	96      1     FileType
//	97      2     TInfo1/Width (little-endian)
//	99      2     TInfo2/Height (little-endian)
//	101     2     TInfo3 (little-endian)
//	103     2     TInfo4 (little-endian)
//	105     1     Comments
//	106     1     TFlags
//	107     22    TInfoS
func (s *Sauce) ToBytes() []byte {
	record := make([]byte, SauceTotalSize)

	// Offset 0: EOF marker
	record[0] = SauceEOF

	// Offset 1-7: SAUCE ID ("SAUCE00")
	copy(record[1:8], SauceID)

	// Offset 8-42: Title (35 bytes, space-padded)
	copy(record[8:43], padRight(s.Title, 35))

	// Offset 43-62: Author (20 bytes, space-padded)
	copy(record[43:63], padRight(s.Author, 20))

	// Offset 63-82: Group (20 bytes, space-padded)
	copy(record[63:83], padRight(s.Group, 20))

	// Offset 83-90: Date (8 bytes, YYYYMMDD format)
	date := s.Date
	if date.IsZero() {
		date = time.Now()
	}
	dateStr := date.Format("20060102")
	copy(record[83:91], dateStr)

	// Offset 91-94: FileSize (4 bytes, little-endian)
	binary.LittleEndian.PutUint32(record[91:95], s.FileSize)

	// Offset 95: DataType
	record[95] = s.DataType

	// Offset 96: FileType
	record[96] = s.FileType

	// Offset 97-98: TInfo1/Width (2 bytes, little-endian)
	binary.LittleEndian.PutUint16(record[97:99], s.TInfo1)

	// Offset 99-100: TInfo2/Height (2 bytes, little-endian)
	binary.LittleEndian.PutUint16(record[99:101], s.TInfo2)

	// Offset 101-102: TInfo3 (2 bytes, little-endian)
	binary.LittleEndian.PutUint16(record[101:103], s.TInfo3)

	// Offset 103-104: TInfo4 (2 bytes, little-endian)
	binary.LittleEndian.PutUint16(record[103:105], s.TInfo4)

	// Offset 105: Comments
	record[105] = s.Comments

	// Offset 106: TFlags
	record[106] = s.TFlags

	// Offset 107-128: TInfoS (22 bytes, null-padded)
	copy(record[107:129], padRightNull(s.TInfoS, 22))

	return record
}

// SetDimensions sets the width and height in the SAUCE record.
func (s *Sauce) SetDimensions(width, height int) {
	s.TInfo1 = uint16(width)
	s.TInfo2 = uint16(height)
}

// SetICEColors enables or disables iCE colors (non-blink mode).
func (s *Sauce) SetICEColors(enabled bool) {
	if enabled {
		s.TFlags |= SauceFlagNonBlink
	} else {
		s.TFlags &^= SauceFlagNonBlink
	}
}

// HasICEColors returns true if iCE colors are enabled.
func (s *Sauce) HasICEColors() bool {
	return s.TFlags&SauceFlagNonBlink != 0
}

// ============================================================================
// PRIVATE
// ============================================================================

// convertField converts a byte slice from source encoding to UTF-8.
// Used for SAUCE text fields (Title, Author, Group, TInfoS).
func convertField(data []byte, sourceEncoding string) string {
	trimmed := bytes.TrimRight(data, " \x00")
	if sourceEncoding == "utf8" || len(trimmed) == 0 {
		return string(trimmed)
	}

	var decoder interface {
		Bytes([]byte) ([]byte, error)
	}

	switch sourceEncoding {
	case "cp437":
		decoder = charmap.CodePage437.NewDecoder()
	case "cp850":
		decoder = charmap.CodePage850.NewDecoder()
	case "iso-8859-1":
		decoder = charmap.ISO8859_1.NewDecoder()
	default:
		return string(trimmed)
	}

	result, err := decoder.Bytes(trimmed)
	if err != nil {
		return string(trimmed)
	}
	return string(result)
}

// parseSauceDate parses YYYYMMDD format into time.Time.
// Returns zero time if the format is invalid.
func parseSauceDate(s string) time.Time {
	if len(s) != 8 {
		return time.Time{}
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// padRight pads a string to the specified length with spaces.
// If the string is longer than length, it is truncated.
func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	result := make([]byte, length)
	copy(result, s)
	for i := len(s); i < length; i++ {
		result[i] = ' '
	}
	return string(result)
}

// padRightNull pads a string to the specified length with null bytes.
// If the string is longer than length, it is truncated.
func padRightNull(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	result := make([]byte, length)
	copy(result, s)
	// remaining bytes are already 0 (null)
	return string(result)
}
