package main

import "testing"

func benchmarkPalette() []rgb5 {
	p := make([]rgb5, 256)
	for i := 0; i < videoPaletteColors; i++ {
		p[i] = rgb5{r: i & 31, g: (i * 7) & 31, b: (i * 13) & 31}
	}
	return p
}

func BenchmarkPaletteLookup(b *testing.B) {
	p := benchmarkPalette()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = paletteLookup(p)
	}
}

func BenchmarkEncodeDelta(b *testing.B) {
	prev := make([]byte, frameBytes)
	curr := make([]byte, frameBytes)
	for i := range curr {
		prev[i] = byte(i % 251)
		curr[i] = prev[i]
		if i%17 == 0 {
			curr[i] ^= 0x3f
		}
	}
	b.ReportAllocs()
	b.SetBytes(frameBytes)
	for i := 0; i < b.N; i++ {
		_ = encodeDelta(prev, curr)
	}
}

func BenchmarkEncodeTileDelta(b *testing.B) {
	prev := make([]byte, frameBytes)
	curr := make([]byte, frameBytes)
	copy(curr, prev)
	for tile := 0; tile < 20; tile++ {
		tx := (tile % (frameWidth / 8)) * 8
		ty := (tile / (frameWidth / 8)) * 8
		for row := 0; row < 8; row++ {
			for col := 0; col < 8; col++ {
				curr[(ty+row)*frameWidth+tx+col] = byte(tile + 1)
			}
		}
	}
	b.ReportAllocs()
	b.SetBytes(frameBytes)
	for i := 0; i < b.N; i++ {
		_ = encodeTileDelta(prev, curr, frameWidth, frameHeight)
	}
}

func BenchmarkEncodeIMAADPCM(b *testing.B) {
	pcm := make([]byte, audioRate)
	for i := range pcm {
		pcm[i] = byte(int8((i % 255) - 127))
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(pcm)))
	for i := 0; i < b.N; i++ {
		_ = encodeIMAADPCM(pcm)
	}
}
