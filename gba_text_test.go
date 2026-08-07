package main

import (
	"bytes"
	"testing"
)

func TestGBATextSupportsUnifiedUkrainianAndRussianAlphabet(t *testing.T) {
	input := "АБВГҐДЕЄЁЖЗИІЇЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ"
	got := sanitizeGBAText(input, 0)
	if got != input {
		t.Fatalf("sanitized alphabet=%q", got)
	}
	if unsupported := unsupportedGBARunes(input); len(unsupported) != 0 {
		t.Fatalf("supported alphabet reported unsupported runes: %q", string(unsupported))
	}
}

func TestGBATextLowercaseAndTypographyNormalization(t *testing.T) {
	got := sanitizeGBAText("Привіт — п’ять… Ёжик", 0)
	want := "ПРИВІТ - П'ЯТЬ... ЁЖИК"
	if got != want {
		t.Fatalf("normalized=%q want %q", got, want)
	}
}

func TestGBATextUsesGlyphCountNotUTF8ByteCount(t *testing.T) {
	got := sanitizeGBAText("Українськийт", 12)
	if gbaTextLength(got) != 12 {
		t.Fatalf("glyph count=%d text=%q", gbaTextLength(got), got)
	}
	if len([]byte(got)) <= 12 {
		t.Fatalf("expected UTF-8 bytes to exceed glyph count: %d", len([]byte(got)))
	}
}

func TestGBATextRuntimeEncodingIsSingleBytePerGlyph(t *testing.T) {
	encoded := encodeGBATextFixed("АБВҐЄЇЁЯ№", 12)
	want := []byte{0x80, 0x81, 0x82, 0x84, 0x87, 0x8D, 0x88, 0xA4, 0xA5, 0, 0, 0}
	if !bytes.Equal(encoded, want) {
		t.Fatalf("encoded=% x want=% x", encoded, want)
	}
}

func TestGBATextUnsupportedCharactersAreReported(t *testing.T) {
	unsupported := unsupportedGBARunes("Відео 😀 日本")
	if len(unsupported) != 3 || unsupported[0] != '😀' || unsupported[1] != '日' || unsupported[2] != '本' {
		t.Fatalf("unsupported=%q", string(unsupported))
	}
}

func TestSafeGBAHeaderTitleTransliteratesCyrillic(t *testing.T) {
	got := string(safeGBAHeaderTitle("Моє відео"))
	if got != "MOYE VIDEO  " {
		t.Fatalf("header title=%q", got)
	}
}

func TestASCIIEncodingRemainsCompatible(t *testing.T) {
	encoded := encodeGBATextFixed("GBA VIDEO", 12)
	want := append([]byte("GBA VIDEO"), 0, 0, 0)
	if !bytes.Equal(encoded, want) {
		t.Fatalf("encoded=% x want=% x", encoded, want)
	}
}

func TestCyrillicLookalikesRemainDistinct(t *testing.T) {
	pairs := [][2]rune{{'И', 'Н'}, {'Й', 'А'}, {'Щ', 'Ц'}}
	for _, pair := range pairs {
		if gbaGlyphBits(pair[0]) == gbaGlyphBits(pair[1]) {
			t.Fatalf("glyphs %c and %c must remain visually distinct", pair[0], pair[1])
		}
	}
}

func TestGBASlashGlyphRemainsDiagonal(t *testing.T) {
	const want uint16 = 0x12A4
	if glyph := gbaGlyphs['/']; glyph != want {
		t.Fatalf("slash glyph = %#04x, want %#04x", glyph, want)
	}
	if gbaGlyphs['/'] == gbaGlyphs['|'] {
		t.Fatalf("slash glyph must not match vertical bar")
	}
}
