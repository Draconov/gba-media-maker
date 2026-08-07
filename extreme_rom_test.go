package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestExtremeROMWritesADPCMAndAdaptiveMetadata(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "video.raw")
	videoPath := filepath.Join(dir, "video.bin")
	indexPath := filepath.Join(dir, "video.idx")
	palettePath := filepath.Join(dir, "palette.bin")
	audioPath := filepath.Join(dir, "audio.bin")
	frames := make([]byte, frameBytes*4)
	for i := range frames {
		frames[i] = byte((i / frameBytes) * 7)
	}
	if err := os.WriteFile(rawPath, frames, 0644); err != nil {
		t.Fatal(err)
	}
	rawSize, storedSize, err := compressRawVideo(rawPath, videoPath, indexPath, "delta", 30, true, []int{0, 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(palettePath, make([]byte, 512), 0644); err != nil {
		t.Fatal(err)
	}
	pcm := make([]byte, audioRate)
	for i := range pcm {
		pcm[i] = byte(int8((i % 80) - 40))
	}
	encoded, info, err := encodeIMAADPCM(pcm, defaultADPCMBlockSamples)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(audioPath, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	clip := convertedClip{
		input:      ClipInput{Title: "EXTREME"},
		options:    ProjectOptions{VBlanks: 5, SeekSeconds: 5, Compression: "delta", AdaptiveKeyframes: true},
		frameCount: 4, paletteCount: 1, hasAudio: true, audioCodec: audioCodecADPCM,
		audioSampleCount: info.SampleCount, audioBlockSamples: info.BlockSamples, audioBlockBytes: info.BlockBytes,
		palette: palettePath, video: videoPath, videoIndex: indexPath, audio: audioPath,
		rawVideo: rawSize, storedVideo: storedSize,
	}
	output := filepath.Join(dir, "extreme.gba")
	result, err := assembleROM(ProjectOptions{OutputPath: output, VBlanks: 5, SeekSeconds: 5, Compression: "delta", RomTitle: "EXTREME", OutputMode: "rom"}, []convertedClip{clip}, output, func(int, string) {})
	if err != nil {
		t.Fatal(err)
	}
	if result.UnpaddedSize <= assetOffset {
		t.Fatalf("unexpected ROM size %d", result.UnpaddedSize)
	}
	rom, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	desc := rom[assetOffset : assetOffset+clipDescriptorSize]
	flags := binary.LittleEndian.Uint16(desc[50:52])
	if flags&0x10 == 0 || flags&0x20 == 0 {
		t.Fatalf("missing extreme flags: %#x", flags)
	}
	if got := binary.LittleEndian.Uint32(desc[80:84]); got != 2 {
		t.Fatalf("audio codec id %d", got)
	}
	if got := binary.LittleEndian.Uint32(desc[84:88]); got != uint32(len(pcm)) {
		t.Fatalf("sample count %d", got)
	}
	if got := binary.LittleEndian.Uint16(desc[56:58]); got != 0 {
		t.Fatalf("adaptive key interval %d", got)
	}
	seekOffset := binary.LittleEndian.Uint32(desc[32:36])
	if seekOffset == 0 {
		t.Fatal("missing seek table")
	}
	if got := binary.LittleEndian.Uint32(rom[seekOffset+4 : seekOffset+8]); got == 0 {
		t.Fatal("ADPCM seek table did not store sample positions")
	}
}
