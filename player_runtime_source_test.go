package main

import (
	"bytes"
	"os"
	"strings"
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

func compactSource(src []byte) string {
	return strings.Join(strings.Fields(string(src)), "")
}

func TestPlayerUsesExplicitResumeStateMachine(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"enumPlaybackState{PLAYBACK_RUNNING=0,PLAYBACK_PAUSED=1,PLAYBACK_RESUME_ARMED=2}",
		"*state=PLAYBACK_PAUSED",
		"*state=PLAYBACK_RESUME_ARMED",
		"state=PLAYBACK_RUNNING",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("playback state machine missing %q", want)
		}
	}
}

func TestPlayerPausePreservesPreparedBuffers(t *testing.T) {
	src := compactSource(playerSource(t))
	start := strings.Index(src, "if(*state==PLAYBACK_RUNNING){")
	if start < 0 {
		t.Fatal("RUNNING -> PAUSED transition missing")
	}
	end := strings.Index(src[start:], "elseif(*state==PLAYBACK_PAUSED)")
	if end < 0 {
		t.Fatal("PAUSED -> RESUME_ARMED transition missing")
	}
	branch := src[start : start+end]
	for _, forbidden := range []string{"render_and_show(", "load_next_pixels(", "show_rendered_page("} {
		if strings.Contains(branch, forbidden) {
			t.Fatalf("pause transition touches prepared frame buffers: %q", forbidden)
		}
	}
	if !strings.Contains(branch, "playback_timer_pause()") || !strings.Contains(branch, "audio_pause()") {
		t.Fatal("pause transition must freeze playback/audio clocks")
	}
}

func TestPlayerTracksPreparedNextFrameValidity(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"intnext_frame_valid=0",
		"if(has_next&&!next_frame_valid)",
		"next_frame_valid=1",
		"if(frame+1<clip->frame_count&&!next_frame_valid){state=PLAYBACK_PAUSED;continue;}",
		"++frame;next_frame_valid=0;playback_clock_advance(&clock)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("prepared-frame guard missing %q", want)
		}
	}
}

func TestPlayerResumeArmedUsesPreparedFrame(t *testing.T) {
	src := compactSource(playerSource(t))
	start := strings.Index(src, "if(action==ACTION_RESUME_PENDING){")
	if start < 0 {
		t.Fatal("dedicated resume branch missing")
	}
	end := strings.Index(src[start:], "if(action==ACTION_UI_REFRESH){")
	if end < 0 {
		t.Fatal("UI refresh branch missing after resume branch")
	}
	branch := src[start : start+end]
	ordered := []string{
		"state!=PLAYBACK_RESUME_ARMED",
		"audio_start_for_frame(clip,resume_frame,1,ui)",
		"wait_vblank()",
		"show_rendered_page(&displayed_page,palette_for_frame(clip,resume_frame))",
		"playback_timer_reset()",
		"audio_resume()",
		"state=PLAYBACK_RUNNING",
	}
	pos := 0
	for _, want := range ordered {
		i := strings.Index(branch[pos:], want)
		if i < 0 {
			t.Fatalf("resume path missing/misordered %q", want)
		}
		pos += i + len(want)
	}
}

func TestPauseButtonUsesDedicatedReleaseLatch(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"pause_button_latched",
		"if((now&KEY_A)==0)ui->pause_button_latched=0",
		"if((pressed&KEY_A)&&!ui->pause_button_latched)",
		"ui->pause_button_latched=1",
		"ui->pause_button_latched=(previous_keys&KEY_A)!=0u",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("pause edge-trigger guard missing %q", want)
		}
	}
}

func TestPlayerHasFirstClassMediaRuntime(t *testing.T) {
	src := playerSource(t)
	for _, want := range [][]byte{
		[]byte("CLIP_FLAG_MEDIA_AUDIO"), []byte("CLIP_FLAG_MEDIA_IMAGE"), []byte("CLIP_FLAG_MEDIA_META"),
		[]byte("MODE3"), []byte("play_audio"), []byte("play_image"),
	} {
		if !bytes.Contains(src, want) {
			t.Fatalf("media runtime missing %q", want)
		}
	}
}

func TestPlayerSlashGlyphSourceUsesCompactFiveStepPattern(t *testing.T) {
	if !bytes.Contains(playerSource(t), []byte("case '/': return 0x12A4u;")) {
		t.Fatal("player slash glyph must use the 0x12A4 compact five-step pattern")
	}
}
