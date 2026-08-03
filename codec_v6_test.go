package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func decodeIMAForTest(data []byte, samples int) []byte {
	out := make([]byte, 0, samples)
	for block := 0; len(out) < samples; block++ {
		off := block * adpcmBlockBytes
		if off+adpcmBlockBytes > len(data) {
			break
		}
		predictor := int(int16(binary.LittleEndian.Uint16(data[off : off+2])))
		index := int(data[off+2])
		if index > 88 {
			index = 88
		}
		out = append(out, byte(int8(clampInt(predictor>>8, -128, 127))))
		for sample := 1; sample < adpcmBlockSamples && len(out) < samples; sample++ {
			packedIndex := sample - 1
			packed := data[off+4+packedIndex/2]
			nibble := int(packed & 15)
			if packedIndex&1 != 0 {
				nibble = int(packed >> 4)
			}
			step := imaStepTable[index]
			delta := step >> 3
			if nibble&4 != 0 {
				delta += step
			}
			if nibble&2 != 0 {
				delta += step >> 1
			}
			if nibble&1 != 0 {
				delta += step >> 2
			}
			if nibble&8 != 0 {
				predictor -= delta
			} else {
				predictor += delta
			}
			predictor = clampInt(predictor, -32768, 32767)
			index = clampInt(index+imaIndexTable[nibble], 0, 88)
			out = append(out, byte(int8(clampInt(predictor>>8, -128, 127))))
		}
	}
	return out
}

func TestIMAADPCMRoundTrip(t *testing.T) {
	pcm := make([]byte, adpcmBlockSamples*3)
	for i := range pcm {
		value := 75*math.Sin(2*math.Pi*440*float64(i)/audioRate) + 20*math.Sin(2*math.Pi*90*float64(i)/audioRate)
		pcm[i] = byte(int8(value))
	}
	encoded := encodeIMAADPCM(pcm)
	if len(encoded) != 3*adpcmBlockBytes {
		t.Fatalf("encoded bytes=%d", len(encoded))
	}
	decoded := decodeIMAForTest(encoded, len(pcm))
	if len(decoded) != len(pcm) {
		t.Fatalf("decoded samples=%d", len(decoded))
	}
	var absolute int64
	for i := range pcm {
		d := int(int8(pcm[i])) - int(int8(decoded[i]))
		if d < 0 {
			d = -d
		}
		absolute += int64(d)
	}
	mean := float64(absolute) / float64(len(pcm))
	if mean > 7.0 {
		t.Fatalf("ADPCM mean absolute error %.2f", mean)
	}
}

func applyTileDeltaForTest(prev, payload []byte, width, height int) []byte {
	out := append([]byte(nil), prev...)
	tileCols := (width + 7) / 8
	tileRows := (height + 7) / 8
	tileCount := tileCols * tileRows
	bitmapBytes := (tileCount + 7) / 8
	off := bitmapBytes
	for tile := 0; tile < tileCount; tile++ {
		if payload[tile>>3]&(1<<uint(tile&7)) == 0 {
			continue
		}
		tx, ty := (tile%tileCols)*8, (tile/tileCols)*8
		tileWidth, tileHeight := minInt(8, width-tx), minInt(8, height-ty)
		for row := 0; row < tileHeight; row++ {
			copy(out[(ty+row)*width+tx:(ty+row)*width+tx+tileWidth], payload[off:off+tileWidth])
			off += tileWidth
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestTileDeltaRoundTrip(t *testing.T) {
	prev := make([]byte, frameBytes)
	curr := append([]byte(nil), prev...)
	for y := 16; y < 24; y++ {
		for x := 32; x < 40; x++ {
			curr[y*frameWidth+x] = byte(x + y)
		}
	}
	payload := encodeTileDelta(prev, curr, frameWidth, frameHeight)
	if len(payload) != 19+64 {
		t.Fatalf("tile delta bytes=%d", len(payload))
	}
	got := applyTileDeltaForTest(prev, payload, frameWidth, frameHeight)
	for i := range curr {
		if got[i] != curr[i] {
			t.Fatalf("pixel %d mismatch", i)
		}
	}
}

func TestTileDeltaEnhancedResolutionEdgeTile(t *testing.T) {
	const width, height = 180, 120
	prev := make([]byte, width*height)
	curr := append([]byte(nil), prev...)
	for y := 112; y < height; y++ {
		for x := 176; x < width; x++ {
			curr[y*width+x] = byte(x + y)
		}
	}
	payload := encodeTileDelta(prev, curr, width, height)
	got := applyTileDeltaForTest(prev, payload, width, height)
	for i := range curr {
		if got[i] != curr[i] {
			t.Fatalf("pixel %d mismatch", i)
		}
	}
}

func TestResolutionDimensions(t *testing.T) {
	cases := []struct {
		mode          string
		width, height int
	}{{"efficient", 120, 80}, {"enhanced", 180, 120}, {"native", 240, 160}, {"", 120, 80}}
	for _, tc := range cases {
		w, h, err := resolutionDimensions(tc.mode)
		if err != nil || w != tc.width || h != tc.height {
			t.Fatalf("%q = %dx%d, %v", tc.mode, w, h, err)
		}
	}
	if _, _, err := resolutionDimensions("wrong"); err == nil {
		t.Fatal("invalid resolution accepted")
	}
}

func TestCompressRawVideoRejectsPartialFrame(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "partial.raw")
	if err := os.WriteFile(raw, make([]byte, 180*120+1), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := compressRawVideo(raw, filepath.Join(dir, "out.video"), filepath.Join(dir, "out.index"), "hybrid", 30, 180, 120)
	if err == nil {
		t.Fatal("partial frame was accepted")
	}
}

func TestHybridCodecChoosesRepeatAndTileDelta(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "frames.raw")
	stream := filepath.Join(dir, "frames.video")
	index := filepath.Join(dir, "frames.index")
	f0 := make([]byte, frameBytes)
	f1 := append([]byte(nil), f0...)
	f2 := append([]byte(nil), f1...)
	for y := 8; y < 16; y++ {
		for x := 8; x < 16; x++ {
			f2[y*frameWidth+x] = 42
		}
	}
	data := append(append(append([]byte{}, f0...), f1...), f2...)
	if err := os.WriteFile(raw, data, 0644); err != nil {
		t.Fatal(err)
	}
	_, stored, stats, err := compressRawVideo(raw, stream, index, "hybrid", 30, frameWidth, frameHeight)
	if err != nil {
		t.Fatal(err)
	}
	if stats.RawFrames != 1 || stats.RepeatFrames != 1 || stats.TileDeltaFrames != 1 {
		t.Fatalf("unexpected stats %+v", stats)
	}
	if stored >= int64(len(data)) {
		t.Fatalf("stored=%d raw=%d", stored, len(data))
	}
}

func TestGBV6PlayerTemplateLayout(t *testing.T) {
	if metadataOffset != 0xAF00 || assetOffset != 0xB000 {
		t.Fatalf("unexpected GBV6 layout metadata=%#x assets=%#x", metadataOffset, assetOffset)
	}
	if len(playerStub) != assetOffset {
		t.Fatalf("player template size=%d want=%d", len(playerStub), assetOffset)
	}
	if string(playerStub[metadataOffset:metadataOffset+4]) != "\x00\x00\x00\x00" {
		t.Fatal("metadata placeholder is not zeroed")
	}
}
