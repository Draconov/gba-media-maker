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

func TestAudioNowPlayingUILayoutAndFeedback(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"MEDIA_META_MAGIC_V2",
		"media_title_limit(c),0x7FFF",
		"media_artist_limit(c),0x5294",
		"text3n(VRAM0,8,144,cur,5,0x7FFF)",
		"rect3(VRAM0,8,156,224,4,0x2108)",
		"rect3(VRAM0,8,156,w,4,0x03FF)",
		"mute_badge3(VRAM0,ui->muted)",
		"volume_badge3(VRAM0,ui->volume_level)",
		"ui->mute_timer=HUD_HOLD_VBLANKS",
		"ui->volume_timer=VOLUME_HOLD_VBLANKS",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("audio Now Playing UI missing %q", want)
		}
	}
}

func TestPlayerUsesUnifiedV013Controls(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"#defineSEEK_REPEAT_VBLANKS18u",
		"#defineSHOULDER_COMBO_GRACE_VBLANKS2u",
		"cycle_hud(structPlayerUI*ui)",
		"quick_toggle_hud(structPlayerUI*ui)",
		"held_seek_action(u16now,u16pressed,intpaused,structPlayerUI*ui)",
		"common_combo_action(u16now,u16pressed,intcan_change,intplaylist,structPlayerUI*ui)",
		"(now&(KEY_START|KEY_SELECT))==(KEY_START|KEY_SELECT)",
		"(now&(KEY_L|KEY_R))==(KEY_L|KEY_R)",
		"returncan_change?(dir<0?ACTION_PREV_CLIP:ACTION_NEXT_CLIP):ACTION_NONE",
		"ui->volume_level<2",
		"ui->volume_level>0",
		"PLAY_RESULT_NEXT_CLIP_DIRECT",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("v0.13 control mapping missing %q", want)
		}
	}
}

func TestPlayerMenuUsesStableColumnNavigation(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"#defineMENU_ROWS10u",
		"menu_column_count",
		"page_size=cols*MENU_ROWS",
		"for(col=0;col<cols;col++)",
		"for(row=0;row<MENU_ROWS;row++)",
		"menu_move_up",
		"menu_move_down",
		"if(p&KEY_LEFT)",
		"if(p&KEY_RIGHT)",
		"if(p&KEY_A)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("stable column menu navigation missing %q", want)
		}
	}
}

func TestPlayerMenuRestoresStableThemeRuntime(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"#include\"menu_background_data.h\"",
		"MENU_THEME_SHIMMER",
		"MENU_THEME_FRAMES",
		"step_menu_shimmer",
		"menu_arrow_init",
		"MENU_ARROW_BLINK_VBLANKS",
		"active_menu_outline",
		"draw_menu_char",
		"SELECTMEDIA",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("stable menu runtime missing %q", want)
		}
	}
}

func TestRestartClearsResumeOutsideMenu(t *testing.T) {
	src := compactSource(playerSource(t))
	if !strings.Contains(src, "if(is_menu_mode(meta)){save_position(meta,clip_index,frame);returnPLAY_RESULT_RETURN_MENU;}clear_position(meta,clip_index);returnPLAY_RESULT_RESTART_CURRENT") {
		t.Fatal("video B restart must clear the resume position outside menu ROMs")
	}
	if !strings.Contains(src, "if(is_menu_mode(m)){save_position(m,idx,f);returnPLAY_RESULT_RETURN_MENU;}clear_position(m,idx);returnPLAY_RESULT_RESTART_CURRENT") {
		t.Fatal("audio B restart must clear the resume position outside menu ROMs")
	}
}
