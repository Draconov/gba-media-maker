package main

import (
	"strings"
	"testing"
)

func TestFFmpegVideoErrorExplainsMissingSoftwareAV1Decoder(t *testing.T) {
	raw := []byte(`[av1 @ 00000211edc03580] Your platform doesn't support hardware accelerated AV1 decoding.
[av1 @ 00000211edc03580] Failed to get pixel format.
[vist#0:0/av1 @ 00000211edd5df40] [dec:av1 @ 00000211edd64900] Error submitting packet to decoder: Function not implemented`)
	err := ffmpegVideoError("FFmpeg could not convert the video", raw)
	if err == nil {
		t.Fatal("expected an error")
	}
	message := err.Error()
	for _, want := range []string{"cannot decode AV1 in software", "libdav1d", "libaom", "FFmpeg details"} {
		if !strings.Contains(message, want) {
			t.Fatalf("AV1 decoder error does not contain %q:\n%s", want, message)
		}
	}
}

func TestFFmpegVideoErrorLeavesOtherErrorsGeneric(t *testing.T) {
	err := ffmpegVideoError("Preview failed", []byte("Invalid data found when processing input"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "libdav1d") || strings.Contains(err.Error(), "libaom") {
		t.Fatalf("non-AV1 error received AV1-specific advice: %s", err)
	}
}
