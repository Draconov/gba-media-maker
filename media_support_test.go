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

func makeGIFFixture(t *testing.T, ff, path string) {
	t.Helper()
	cmd := exec.Command(ff, "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc=size=96x64:rate=10", "-t", "0.8", "-vf", "fps=10,scale=96:64:flags=neighbor", "-loop", "0", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gif fixture: %v\n%s", err, out)
	}
}

func TestGIFIsVideoAndLoopsAutomatically(t *testing.T) {
	ff := requireFFmpegForMedia(t)
	dir := t.TempDir()
	gif := filepath.Join(dir, "animation.gif")
	makeGIFFixture(t, ff, gif)
	info, err := inspectMedia(ff, gif)
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != "video" || info.Duration <= 0 || info.Width != 96 || info.Height != 64 {
		t.Fatalf("bad GIF info: %+v", info)
	}
	out := filepath.Join(dir, "gif.gba")
	_, err = convertProject(ProjectOptions{
		Inputs:     []ClipInput{{InputPath: gif, Name: "animation.gif", Title: "GIF TEST", MediaKind: info.Kind}},
		OutputPath: out, FFmpegPath: ff, Speed: 1, VBlanks: 6, FitMode: "fit", AudioMode: "none", Volume: 1,
		RomTitle: "GIF TEST", SeekSeconds: 5, Limiter: true, Resume: true, Compression: "delta",
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
	if flags&2 == 0 {
		t.Fatalf("GIF clip loop flag not set: flags=%#x", flags)
	}
	if flags&128 != 0 {
		t.Fatalf("GIF was encoded as static image: flags=%#x", flags)
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

func TestAudioMetadataV2StoresLongSongTitleAndArtist(t *testing.T) {
	title := "A MUCH LONGER SONG TITLE HERE"
	artist := "FIRST ARTIST AND SECOND ART"
	b := encodeMediaMetadata(title, artist, "ALBUM")
	if len(b) != 80 {
		t.Fatalf("metadata size=%d", len(b))
	}
	if got := binary.LittleEndian.Uint32(b[0:4]); got != mediaMetadataMagic {
		t.Fatalf("metadata magic=%#x", got)
	}
	wantTitle := encodeGBATextFixed(title, 28)
	wantArtist := encodeGBATextFixed(artist, 28)
	if string(b[4:32]) != string(wantTitle) {
		t.Fatalf("title field was not preserved at 28 chars")
	}
	if string(b[32:60]) != string(wantArtist) {
		t.Fatalf("artist field was not preserved at 28 chars")
	}
}

func TestAudioMusicSettingsOverrideSourceMetadata(t *testing.T) {
	ff := requireFFmpegForMedia(t)
	dir := t.TempDir()
	wav := filepath.Join(dir, "song.wav")
	makeAudioFixture(t, ff, wav)
	out := filepath.Join(dir, "custom-metadata.gba")
	wantTitle := "CUSTOM SONG TITLE 123456789"
	wantArtist := "CUSTOM ARTIST AND FRIENDS"
	_, err := convertProject(ProjectOptions{
		Inputs: []ClipInput{{
			InputPath: wav, Name: "song.wav", Title: "MENU TITLE", MediaKind: "audio",
			MusicTitle: wantTitle, MusicArtist: wantArtist,
		}},
		OutputPath: out, FFmpegPath: ff, Speed: 1, VBlanks: 6, FitMode: "fit", AudioMode: "mix", Volume: 1,
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
	d := descriptorAt(t, rom, 0)
	metaOff := int(binary.LittleEndian.Uint32(d[12:16]))
	if metaOff <= 0 || metaOff+mediaMetadataSize > len(rom) {
		t.Fatalf("metadata offset=%d outside ROM", metaOff)
	}
	meta := rom[metaOff : metaOff+mediaMetadataSize]
	if got := binary.LittleEndian.Uint32(meta[:4]); got != mediaMetadataMagic {
		t.Fatalf("metadata magic=%#x", got)
	}
	if string(meta[4:32]) != string(encodeGBATextFixed(wantTitle, 28)) {
		t.Fatal("custom song title was not written into ROM metadata")
	}
	if string(meta[32:60]) != string(encodeGBATextFixed(wantArtist, 28)) {
		t.Fatal("custom artist was not written into ROM metadata")
	}
}
