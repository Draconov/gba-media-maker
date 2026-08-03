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
