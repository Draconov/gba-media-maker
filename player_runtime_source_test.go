package main

import (
	"bytes"
	"os"
	"testing"
)

func TestPlayerPausePreservesPreparedBackBuffer(t *testing.T) {
	src, err := os.ReadFile("player/main.c")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("if (*paused) {\n                playback_timer_pause();\n                if (has_audio) audio_pause();")
	if !bytes.Contains(src, want) {
		t.Fatal("pause path must stop clocks without leaving the frame-wait loop")
	}
	if !bytes.Contains(src, []byte("return ACTION_RESUME_PENDING;")) {
		t.Fatal("unpause must use the dedicated resume-pending action")
	}
}

func TestPlayerResumeUsesDedicatedLightPath(t *testing.T) {
	src, err := os.ReadFile("player/main.c")
	if err != nil {
		t.Fatal(err)
	}
	start := bytes.Index(src, []byte("if (action==ACTION_RESUME_PENDING)"))
	if start < 0 {
		t.Fatal("dedicated ACTION_RESUME_PENDING branch missing")
	}
	end := bytes.Index(src[start:], []byte("if (action==ACTION_UI_REFRESH)"))
	if end < 0 {
		t.Fatal("generic UI refresh branch missing after dedicated resume path")
	}
	branch := src[start : start+end]
	ordered := [][]byte{
		[]byte("audio_start_for_frame(clip,resume_frame,1,ui);"),
		[]byte("wait_vblank();"),
		[]byte("show_rendered_page(&displayed_page,palette_for_frame(clip,resume_frame));"),
		[]byte("playback_timer_reset();"),
		[]byte("if (has_audio) audio_resume();"),
	}
	position := 0
	for _, want := range ordered {
		index := bytes.Index(branch[position:], want)
		if index < 0 {
			t.Fatalf("light resume path missing or misordered %q", want)
		}
		position += index + len(want)
	}
	if bytes.Contains(branch, []byte("render_and_show(")) || bytes.Contains(branch, []byte("load_next_pixels(")) {
		t.Fatal("dedicated resume path must not redraw or decode a frame")
	}
}

func TestPlayerSlashGlyphSourceUsesCompactFiveStepPattern(t *testing.T) {
	src, err := os.ReadFile("player/main.c")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(src, []byte("case '/': return 0x12A4u;")) {
		t.Fatal("player slash glyph must use the 0x12A4 compact five-step pattern")
	}
}

func TestPauseButtonUsesDedicatedReleaseLatch(t *testing.T) {
	src, err := os.ReadFile("player/main.c")
	if err != nil {
		t.Fatal(err)
	}
	checks := [][]byte{
		[]byte("int pause_button_latched;"),
		[]byte("if ((now & KEY_A) == 0u) ui->pause_button_latched = 0;"),
		[]byte("if ((pressed & KEY_A) && !ui->pause_button_latched)"),
		[]byte("ui->pause_button_latched = 1;"),
		[]byte("ui->pause_button_latched = (previous_keys & KEY_A) != 0u;"),
	}
	for _, want := range checks {
		if !bytes.Contains(src, want) {
			t.Fatalf("pause edge-trigger guard missing %q", want)
		}
	}
}
