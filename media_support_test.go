package main

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireFFmpegForMedia(t *testing.T) string {
	t.Helper()
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg missing")
	}
	return ff
}

func makeAudioFixture(t *testing.T, ff, path string) {
	t.Helper()
	cmd := exec.Command(ff, "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=523:sample_rate=44100", "-t", "1.2", "-metadata", "title=Test Audio", "-metadata", "artist=GBA Media Maker", "-c:a", "pcm_s16le", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("audio fixture: %v\n%s", err, out)
	}
}

func makeImageFixture(t *testing.T, ff, path string) {
	t.Helper()
	cmd := exec.Command(ff, "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=royalblue:size=320x200", "-frames:v", "1", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("image fixture: %v\n%s", err, out)
	}
}

func descriptorAt(t *testing.T, rom []byte, index int) []byte {
	t.Helper()
	if len(rom) < metadataOffset+64 {
		t.Fatal("ROM too small")
	}
	table := int(binary.LittleEndian.Uint32(rom[metadataOffset+12 : metadataOffset+16]))
	size := int(binary.LittleEndian.Uint32(rom[metadataOffset+16 : metadataOffset+20]))
	if size != clipDescriptorSize {
		t.Fatalf("descriptor size=%d", size)
	}
	off := table + index*size
	if off < 0 || off+size > len(rom) {
		t.Fatalf("descriptor %d outside ROM", index)
	}
	return rom[off : off+size]
}

func TestInspectMediaRecognizesAudioAndImage(t *testing.T) {
	ff := requireFFmpegForMedia(t)
	dir := t.TempDir()
	wav := filepath.Join(dir, "song.wav")
	png := filepath.Join(dir, "picture.png")
	makeAudioFixture(t, ff, wav)
	makeImageFixture(t, ff, png)
	ai, err := inspectMedia(ff, wav)
	if err != nil {
		t.Fatal(err)
	}
	if ai.Kind != "audio" || ai.AudioStreams < 1 || ai.Duration <= 0 {
		t.Fatalf("bad audio info: %+v", ai)
	}
	ii, err := inspectMedia(ff, png)
	if err != nil {
		t.Fatal(err)
	}
	if ii.Kind != "image" || ii.Width != 320 || ii.Height != 200 {
		t.Fatalf("bad image info: %+v", ii)
	}
}

func TestMixedMediaROMUsesNativeAudioAndImageDescriptors(t *testing.T) {
	ff := requireFFmpegForMedia(t)
	dir := t.TempDir()
	wav := filepath.Join(dir, "song.wav")
	png := filepath.Join(dir, "picture.png")
	video := filepath.Join(dir, "video.mkv")
	makeAudioFixture(t, ff, wav)
	makeImageFixture(t, ff, png)
	makeFixture(t, ff, video, 0.8, "20")
	out := filepath.Join(dir, "mixed.gba")
	_, err := convertProject(ProjectOptions{
		Inputs: []ClipInput{
			{InputPath: video, Name: "video.mkv", Title: "VIDEO", MediaKind: "video"},
			{InputPath: wav, Name: "song.wav", Title: "AUDIO", MediaKind: "audio"},
			{InputPath: png, Name: "picture.png", Title: "IMAGE", MediaKind: "image", ImageSeconds: 3},
		},
		OutputPath: out, FFmpegPath: ff, Speed: 1, VBlanks: 6, FitMode: "fit", AudioMode: "mix", Volume: 1,
		RomTitle: "MEDIA TEST", SeekSeconds: 5, Limiter: true, Resume: true, Compression: "delta",
		PaletteMode: "shared", DitherMode: "ordered", OutputMode: "menu", KeyInterval: 30, AudioCodec: "pcm",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rom, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	videoD := descriptorAt(t, rom, 0)
	audioD := descriptorAt(t, rom, 1)
	imageD := descriptorAt(t, rom, 2)
	vflags := binary.LittleEndian.Uint16(videoD[50:52])
	aflags := binary.LittleEndian.Uint16(audioD[50:52])
	iflags := binary.LittleEndian.Uint16(imageD[50:52])
	if vflags&64 != 0 || vflags&128 != 0 {
		t.Fatalf("video media flags=%#x", vflags)
	}
	if aflags&64 == 0 || aflags&1 == 0 || aflags&4 != 0 {
		t.Fatalf("audio flags=%#x", aflags)
	}
	if iflags&128 == 0 || iflags&1 != 0 || iflags&4 != 0 {
		t.Fatalf("image flags=%#x", iflags)
	}
	for name, d := range map[string][]byte{"audio": audioD, "image": imageD} {
		if got := binary.LittleEndian.Uint32(d[4:8]); got != nativeImageBytes {
			t.Fatalf("%s frame bytes=%d", name, got)
		}
		if got := binary.LittleEndian.Uint16(d[46:48]); got != nativeImageWidth {
			t.Fatalf("%s width=%d", name, got)
		}
		if got := binary.LittleEndian.Uint16(d[48:50]); got != nativeImageHeight {
			t.Fatalf("%s height=%d", name, got)
		}
	}
	if got := binary.LittleEndian.Uint32(imageD[84:88]); got != 3000 {
		t.Fatalf("image duration ms=%d", got)
	}
}

func TestAudioMediaCannotBeSilencedByLegacyNoAudioSetting(t *testing.T) {
	ff := requireFFmpegForMedia(t)
	dir := t.TempDir()
	wav := filepath.Join(dir, "song.wav")
	makeAudioFixture(t, ff, wav)
	out := filepath.Join(dir, "audio.gba")
	_, err := convertProject(ProjectOptions{
		Inputs:     []ClipInput{{InputPath: wav, Name: "song.wav", Title: "AUDIO", MediaKind: "audio"}},
		OutputPath: out, FFmpegPath: ff, Speed: 1, VBlanks: 6, FitMode: "fit", AudioMode: "none", Volume: 1,
		RomTitle: "AUDIO TEST", SeekSeconds: 5, Limiter: true, Resume: true, Compression: "delta",
		PaletteMode: "shared", DitherMode: "ordered", OutputMode: "rom", KeyInterval: 30, AudioCodec: "pcm",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rom, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	flags := binary.LittleEndian.Uint16(descriptorAt(t, rom, 0)[50:52])
	if flags&64 == 0 || flags&1 == 0 {
		t.Fatalf("audio entry was emitted without audio: flags=%#x", flags)
	}
}
