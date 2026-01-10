package types

import (
	"fmt"
	"strconv"
	"strings"
)

// CropRegion defines a rectangular region for cropping.
// Coordinates are 0-indexed.
type CropRegion struct {
	X      int // Start X (column)
	Y      int // Start Y (row)
	Width  int // Width of region
	Height int // Height of region
}

// ParseCropRegion parses a crop string in format "x,y:x1,y1" (start:end coordinates).
// Both coordinates are 0-indexed.
// Returns nil if the string is empty.
func ParseCropRegion(s string) (*CropRegion, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid crop format: expected 'x,y:x1,y1', got '%s'", s)
	}

	startParts := strings.Split(parts[0], ",")
	endParts := strings.Split(parts[1], ",")

	if len(startParts) != 2 || len(endParts) != 2 {
		return nil, fmt.Errorf("invalid crop format: expected 'x,y:x1,y1', got '%s'", s)
	}

	x, err := strconv.Atoi(strings.TrimSpace(startParts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid start X coordinate: %w", err)
	}

	y, err := strconv.Atoi(strings.TrimSpace(startParts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid start Y coordinate: %w", err)
	}

	x1, err := strconv.Atoi(strings.TrimSpace(endParts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid end X coordinate: %w", err)
	}

	y1, err := strconv.Atoi(strings.TrimSpace(endParts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid end Y coordinate: %w", err)
	}

	// Calculate width and height from start:end
	width := x1 - x
	height := y1 - y

	if width <= 0 {
		return nil, fmt.Errorf("invalid crop region: end X (%d) must be greater than start X (%d)", x1, x)
	}
	if height <= 0 {
		return nil, fmt.Errorf("invalid crop region: end Y (%d) must be greater than start Y (%d)", y1, y)
	}

	return &CropRegion{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}, nil
}

// IsSet returns true if the crop region is defined (not nil).
func (c *CropRegion) IsSet() bool {
	return c != nil
}
