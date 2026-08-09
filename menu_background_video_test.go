package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMenuBackgroundSampling(t *testing.T) {
	count, fps, vblanks, err := menuBackgroundSampling(4)
	if err != nil {
		t.Fatal(err)
	}
	if count != 16 || vblanks != 15 {
		t.Fatalf("4-second background sampling = %d frames, %.4f fps, %d VBlanks; want 16 frames and 15 VBlanks", count, fps, vblanks)
	}
	count, _, vblanks, err = menuBackgroundSampling(1)
	if err != nil {
		t.Fatal(err)
	}
	if count != 10 || vblanks != 6 {
		t.Fatalf("1-second background sampling = %d frames, %d VBlanks; want 10 frames and 6 VBlanks", count, vblanks)
	}
	for _, duration := range []float64{0.5, 33} {
		if _, _, _, err := menuBackgroundSampling(duration); err == nil {
			t.Fatalf("duration %.2f should be rejected", duration)
		}
	}
}

func TestExtractMenuBackgroundVideoFrames(t *testing.T) {
	ffmpeg := commandExists("ffmpeg")
	if ffmpeg == "" {
		t.Skip("ffmpeg is not available")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "menu-source.mp4")
	if output, err := runCommand(ffmpeg,
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=320x180:rate=30",
		"-t", "2", "-an", "-c:v", "mpeg4", input,
	); err != nil {
		t.Skipf("could not generate test video with this ffmpeg: %v\n%s", err, output)
	}
	output := filepath.Join(dir, "menu.rgb")
	data, count, vblanks, err := extractMenuBackgroundVideoFrames(context.Background(), ffmpeg, input, 0, 2, output)
	if err != nil {
		t.Fatal(err)
	}
	if count != 16 {
		t.Fatalf("decoded frame count = %d; want 16", count)
	}
	if vblanks != 7 {
		t.Fatalf("frame interval = %d VBlanks; want 7", vblanks)
	}
	wantBytes := count * menuBackgroundWidth * menuBackgroundHeight * 3
	if len(data) != wantBytes {
		t.Fatalf("raw frame bytes = %d; want %d", len(data), wantBytes)
	}
	if st, err := os.Stat(output); err != nil || st.Size() != int64(wantBytes) {
		t.Fatalf("raw output size mismatch: stat=%v err=%v", st, err)
	}
}
