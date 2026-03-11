package types

import "testing"

func TestParseNeotexColor(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		isForeground bool
		want         ColorValue
	}{
		{
			name:         "foreground standard",
			code:         "Fk",
			isForeground: true,
			want:         ColorValue{Type: ColorStandard, Index: 0},
		},
		{
			name:         "foreground bright",
			code:         "FR",
			isForeground: true,
			want:         ColorValue{Type: ColorStandard, Index: 9},
		},
		{
			name:         "background standard",
			code:         "Bk",
			isForeground: false,
			want:         ColorValue{Type: ColorStandard, Index: 0},
		},
		{
			name:         "background bright",
			code:         "BR",
			isForeground: false,
			want:         ColorValue{Type: ColorStandard, Index: 9},
		},
		{
			name:         "foreground indexed",
			code:         "F123",
			isForeground: true,
			want:         ColorValue{Type: ColorIndexed, Index: 123},
		},
		{
			name:         "background rgb",
			code:         "B00FF00",
			isForeground: false,
			want:         ColorValue{Type: ColorRGB, R: 0, G: 255, B: 0},
		},
		{
			name:         "foreground default",
			code:         "FD",
			isForeground: true,
			want:         ColorValue{Type: ColorStandard, Index: 7},
		},
		{
			name:         "background default",
			code:         "BD",
			isForeground: false,
			want:         ColorValue{Type: ColorStandard, Index: 0},
		},
		{
			name:         "foreground no prefix",
			code:         "k",
			isForeground: true,
			want:         ColorValue{Type: ColorStandard, Index: 0},
		},
		{
			name:         "background no prefix rgb",
			code:         "00FF00",
			isForeground: false,
			want:         ColorValue{Type: ColorRGB, R: 0, G: 255, B: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNeotexColor(tt.code, tt.isForeground)
			if err != nil {
				t.Fatalf("ParseNeotexColor(%q) error: %v", tt.code, err)
			}
			if got != tt.want {
				t.Fatalf("ParseNeotexColor(%q) = %+v, want %+v", tt.code, got, tt.want)
			}
		})
	}
}

func TestParseNeotexColorErrors(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		isForeground bool
	}{
		{
			name:         "prefix mismatch",
			code:         "BD",
			isForeground: true,
		},
		{
			name:         "empty code",
			code:         "",
			isForeground: true,
		},
		{
			name:         "indexed out of range",
			code:         "F999",
			isForeground: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseNeotexColor(tt.code, tt.isForeground)
			if err == nil {
				t.Fatalf("ParseNeotexColor(%q) expected error", tt.code)
			}
		})
	}
}

func TestParseNeotexColorCode(t *testing.T) {
	isFg, color, ok, err := ParseNeotexColorCode("Fk")
	if err != nil || !ok || !isFg || color != (ColorValue{Type: ColorStandard, Index: 0}) {
		t.Fatalf("ParseNeotexColorCode(Fk) unexpected result: %v %v %v %+v", ok, err, isFg, color)
	}

	isFg, color, ok, err = ParseNeotexColorCode("B123")
	if err != nil || !ok || isFg || color != (ColorValue{Type: ColorIndexed, Index: 123}) {
		t.Fatalf("ParseNeotexColorCode(B123) unexpected result: %v %v %v %+v", ok, err, isFg, color)
	}

	_, _, ok, err = ParseNeotexColorCode("EU")
	if err != nil || ok {
		t.Fatalf("ParseNeotexColorCode(EU) expected ok=false, got ok=%v err=%v", ok, err)
	}

	_, _, ok, err = ParseNeotexColorCode("F999")
	if err == nil || !ok {
		t.Fatalf("ParseNeotexColorCode(F999) expected error with ok=true")
	}
}
