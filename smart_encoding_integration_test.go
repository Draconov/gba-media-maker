package main

import (
	"context"
	"os"
	"testing"
)

// TestSmartEncodingWithFFmpeg exercises the real bounded FFmpeg scan when a
// fixture is supplied by release validation. Ordinary contributors can run the
// unit suite without shipping a media file.
func TestSmartEncodingWithFFmpeg(t *testing.T) {
	input := os.Getenv("GBAVM_TEST_VIDEO")
	if input == "" {
		t.Skip("set GBAVM_TEST_VIDEO to run the FFmpeg integration test")
	}
	ffmpeg := commandExists("ffmpeg")
	if ffmpeg == "" {
		t.Skip("ffmpeg is not installed")
	}
	info, err := inspectMedia(ffmpeg, input)
	if err != nil {
		t.Fatal(err)
	}
	result, err := AnalyzeExtremeEncodingContext(context.Background(), ffmpeg, input, info, ProjectOptions{
		VBlanks: 6, AudioMode: "mix", Compression: "delta", PaletteMode: "shared",
		DitherMode: "ordered", KeyInterval: 30, ExtremeOptimization: true,
		AdaptiveKeyframes: true, EnhancedSceneDetection: true, AudioCodec: audioCodecAuto,
	}, 32, "balanced", audioCodecAuto, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Samples) < 3 {
		t.Fatalf("selected %d representative samples; want at least 3", len(result.Samples))
	}
	if len(result.Candidates) < 4 || result.Recommended.ID == "" {
		t.Fatalf("incomplete recommendation: %+v", result)
	}
	if result.Recommended.EstimatedMaxBytes <= 0 || result.Recommended.VisualQuality <= 0 {
		t.Fatalf("invalid recommendation metrics: %+v", result.Recommended)
	}
}
