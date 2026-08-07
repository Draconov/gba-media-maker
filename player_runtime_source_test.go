package main

import (
	"bytes"
	"os"
	"testing"
)

func playerSource(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile("player/main.c")
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestPlayerUsesExplicitResumeStateMachine(t *testing.T) {
	src := playerSource(t)
	checks := [][]byte{
		[]byte("enum PlaybackState"),
		[]byte("PLAYBACK_RUNNING = 0"),
		[]byte("PLAYBACK_PAUSED = 1"),
		[]byte("PLAYBACK_RESUME_ARMED = 2"),
		[]byte("*state = PLAYBACK_PAUSED;"),
		[]byte("*state = PLAYBACK_RESUME_ARMED;"),
		[]byte("state = PLAYBACK_RUNNING;"),
	}
	for _, want := range checks {
		if !bytes.Contains(src, want) {
			t.Fatalf("playback state machine missing %q", want)
		}
	}
}

func TestPlayerPausePreservesPreparedFrontAndBackBuffers(t *testing.T) {
	src := playerSource(t)
	start := bytes.Index(src, []byte("if (*state == PLAYBACK_RUNNING)"))
	if start < 0 {
		t.Fatal("RUNNING -> PAUSED transition missing")
	}
	end := bytes.Index(src[start:], []byte("if (*state == PLAYBACK_PAUSED)"))
	if end < 0 {
		t.Fatal("PAUSED -> RESUME_ARMED transition missing")
	}
	branch := src[start : start+end]
	for _, forbidden := range [][]byte{
		[]byte("render_and_show("),
		[]byte("load_next_pixels("),
		[]byte("show_rendered_page("),
	} {
		if bytes.Contains(branch, forbidden) {
			t.Fatalf("pause transition must not touch prepared frame buffers: %q", forbidden)
		}
	}
	if !bytes.Contains(branch, []byte("playback_timer_pause();")) || !bytes.Contains(branch, []byte("audio_pause();")) {
		t.Fatal("pause transition must freeze playback/audio clocks")
	}
}

func TestPlayerTracksPreparedNextFrameValidity(t *testing.T) {
	src := playerSource(t)
	checks := [][]byte{
		[]byte("int next_frame_valid = 0;"),
		[]byte("if (has_next && !next_frame_valid)"),
		[]byte("next_frame_valid = 1;"),
		[]byte("if (has_next && !next_frame_valid) { state = PLAYBACK_PAUSED; continue; }"),
		[]byte("++frame; next_frame_valid = 0; playback_clock_advance(&clock);"),
	}
	for _, want := range checks {
		if !bytes.Contains(src, want) {
			t.Fatalf("prepared-frame validity guard missing %q", want)
		}
	}
}

func TestPlayerResumeArmedUsesOnlyPreparedFrame(t *testing.T) {
	src := playerSource(t)
	start := bytes.Index(src, []byte("if (action==ACTION_RESUME_PENDING)"))
	if start < 0 {
		t.Fatal("dedicated resume branch missing")
	}
	end := bytes.Index(src[start:], []byte("if (action==ACTION_UI_REFRESH)"))
	if end < 0 {
		t.Fatal("generic UI refresh branch missing after resume branch")
	}
	branch := src[start : start+end]
	ordered := [][]byte{
		[]byte("state != PLAYBACK_RESUME_ARMED"),
		[]byte("audio_start_for_frame(clip,resume_frame,1,ui);"),
		[]byte("wait_vblank();"),
		[]byte("show_rendered_page(&displayed_page,palette_for_frame(clip,resume_frame));"),
		[]byte("playback_timer_reset();"),
		[]byte("if (has_audio) audio_resume();"),
		[]byte("state = PLAYBACK_RUNNING;"),
	}
	position := 0
	for _, want := range ordered {
		index := bytes.Index(branch[position:], want)
		if index < 0 {
			t.Fatalf("resume-armed path missing or misordered %q", want)
		}
		position += index + len(want)
	}
	for _, forbidden := range [][]byte{
		[]byte("render_and_show("),
		[]byte("load_next_pixels("),
		[]byte("render_frame_with_ui("),
	} {
		if bytes.Contains(branch, forbidden) {
			t.Fatalf("resume-armed path must not redraw/decode: %q", forbidden)
		}
	}
}

func TestPlayerInvalidatesPreparedFrameOnlyForExplicitFrameChangingPaths(t *testing.T) {
	src := playerSource(t)
	if bytes.Count(src, []byte("next_frame_valid = 0;")) < 5 {
		t.Fatal("expected explicit next-frame invalidation after redraw, help, frame-step, seek, and consumption")
	}
	checks := [][]byte{
		[]byte("render_and_show(current,frame,&displayed_page,clip,ui);\n                next_frame_valid = 0;"),
		[]byte("next_frame_valid = 0; state = PLAYBACK_PAUSED;"),
	}
	for _, want := range checks {
		if !bytes.Contains(src, want) {
			t.Fatalf("explicit invalidation path missing %q", want)
		}
	}
}

func TestPauseButtonUsesDedicatedReleaseLatch(t *testing.T) {
	src := playerSource(t)
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

func TestPlayerSlashGlyphSourceUsesCompactFiveStepPattern(t *testing.T) {
	src := playerSource(t)
	if !bytes.Contains(src, []byte("case '/': return 0x12A4u;")) {
		t.Fatal("player slash glyph must use the 0x12A4 compact five-step pattern")
	}
}
