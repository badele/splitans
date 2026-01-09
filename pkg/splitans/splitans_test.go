package splitans

import "testing"

func TestNormalizeANSIUTF8Input_RemovesCarriageReturn(t *testing.T) {
	input := []byte("foo\r\nbar\r\nbaz")
	got := NormalizeANSIUTF8Input(input, 80)
	want := []byte("foo\nbar\nbaz")

	if string(got) != string(want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeANSIUTF8Input_KeepsDataWhenWidthZero(t *testing.T) {
	input := []byte("foo\rbar")
	got := NormalizeANSIUTF8Input(input, 0)

	if string(got) != string(input) {
		t.Fatalf("expected original data untouched, got %q", got)
	}
}

func TestNormalizeANSIUTF8Input_PreservesUnicode(t *testing.T) {
	input := []byte("é\rà\n∞")
	got := NormalizeANSIUTF8Input(input, 40)
	want := []byte("éà\n∞")

	if string(got) != string(want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
