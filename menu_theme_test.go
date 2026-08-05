package main

import (
	"encoding/binary"
	"testing"
)

func TestAppendMenuTheme(t *testing.T) {
	palette := make([]byte, 512)
	frame := make([]byte, frameBytes)
	theme := &MenuThemeOptions{
		ID: "test", Kind: "frames", Palette: palette, Frames: [][]byte{frame, frame},
		FrameVBlanks: 12, UIColor: 0x7fff, SelectedColor: 0x037f,
		Outline: true, OutlineColor: 0,
	}
	rom := make([]byte, assetOffset)
	out, offset, err := appendMenuTheme(rom, theme)
	if err != nil {
		t.Fatal(err)
	}
	if offset != assetOffset {
		t.Fatalf("theme offset=%#x", offset)
	}
	h := out[offset : offset+menuThemeHeaderSize]
	if got := binary.LittleEndian.Uint32(h[:4]); got != menuThemeMagic {
		t.Fatalf("magic=%#x", got)
	}
	if got := binary.LittleEndian.Uint16(h[6:8]); got != menuThemeFrames {
		t.Fatalf("kind=%d", got)
	}
	if got := binary.LittleEndian.Uint16(h[16:18]); got != 2 {
		t.Fatalf("frames=%d", got)
	}
	if got := binary.LittleEndian.Uint16(h[18:20]); got != 12 {
		t.Fatalf("vblanks=%d", got)
	}
	if got := binary.LittleEndian.Uint16(h[20:22]); got != 1 {
		t.Fatalf("flags=%d", got)
	}
	if len(out) < assetOffset+menuThemeHeaderSize+512+frameBytes*2 {
		t.Fatal("theme data was not aligned/appended")
	}
}

func TestMenuThemeRejectsReservedPixels(t *testing.T) {
	palette := make([]byte, 512)
	frame := make([]byte, frameBytes)
	frame[10] = 250
	theme := &MenuThemeOptions{Kind: "static", Palette: palette, Frames: [][]byte{frame}}
	if err := theme.validate(); err == nil {
		t.Fatal("expected reserved palette index error")
	}
}
