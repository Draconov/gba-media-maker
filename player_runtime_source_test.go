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
		"intadvance=frame+1<clip->frame_count&&next_frame_valid",
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
		"intadvance=frame+1<clip->frame_count&&next_frame_valid",
		"state!=PLAYBACK_RESUME_ARMED",
		"audio_start_for_frame(clip,resume_frame,1,ui)",
		"wait_vblank()",
		"if(advance)",
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
		"ui->mute_timer=mute_hold",
		"ui->volume_timer=AUDIO_VOLUME_HOLD_VBLANKS",
		"#defineAUDIO_HUD_HOLD_VBLANKS24u",
		"#defineAUDIO_VOLUME_HOLD_VBLANKS24u",
		"#defineAUDIO_SEEK_HOLD_VBLANKS24u",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("audio Now Playing UI missing %q", want)
		}
	}
}

func TestPlayerUsesSimplifiedV013Controls(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"#defineSEEK_REPEAT_VBLANKS18u",
		"cycle_hud(structPlayerUI*ui)",
		"held_seek_action(u16now,u16pressed,intpaused,structPlayerUI*ui)",
		"common_combo_action(u16now,u16pressed,intcan_change,intaudio_controls,u16mute_hold,structPlayerUI*ui)",
		"(now&(KEY_START|KEY_SELECT))==(KEY_START|KEY_SELECT)",
		"if(can_change&&(pressed&KEY_L)&&!(now&KEY_R))returnACTION_PREV_CLIP",
		"if(can_change&&(pressed&KEY_R)&&!(now&KEY_L))returnACTION_NEXT_CLIP",
		"if(has_audio&&(pressed&KEY_UP))",
		"if(has_audio&&(pressed&KEY_DOWN))",
		"PLAY_RESULT_NEXT_CLIP_DIRECT",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("v0.13 control mapping missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"SHOULDER_COMBO_GRACE_VBLANKS",
		"quick_toggle_hud",
		"hud_combo_latched",
		"clip_combo_latched",
		"shoulder_pending_direction",
		"SELECT+L/R",
		"L+RQUICKHUD",
	} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("removed control overlap is still present: %q", forbidden)
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

func TestVideoInputIsPolledBeforeFrameDeadline(t *testing.T) {
	src := compactSource(playerSource(t))
	start := strings.Index(src, "staticintwait_frame_period(")
	if start < 0 {
		t.Fatal("wait_frame_period missing")
	}
	end := strings.Index(src[start:], "staticinttick_ui_timers(")
	if end < 0 {
		t.Fatal("could not isolate wait_frame_period")
	}
	fn := src[start : start+end]
	keyPos := strings.Index(fn, "now=keys_down()")
	deadlinePos := strings.Index(fn, "if(playback_timer_read()>=deadline)returnACTION_NONE")
	if keyPos < 0 || deadlinePos < 0 {
		t.Fatal("wait_frame_period missing key/deadline checks")
	}
	if keyPos > deadlinePos {
		t.Fatal("video input must be polled before returning for an elapsed frame deadline")
	}
}

func TestShouldersNavigateWithoutSelectOrHudCombo(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"if(can_change&&(pressed&KEY_L)&&!(now&KEY_R))returnACTION_PREV_CLIP",
		"if(can_change&&(pressed&KEY_R)&&!(now&KEY_L))returnACTION_NEXT_CLIP",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("direct shoulder navigation missing %q", want)
		}
	}
	for _, forbidden := range []string{"if(can_change&&(now&KEY_SELECT))", "quick_toggle_hud", "SHOULDER_COMBO_GRACE_VBLANKS"} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("obsolete shoulder combo behavior remains: %q", forbidden)
		}
	}
}

func TestManualImagesDoNotPauseAndImagesHaveNoAudioControls(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"action=common_combo_action(now,p,m->clip_count>1,0,HUD_HOLD_VBLANKS,ui)",
		"if(limit&&(p&KEY_A))",
		"show_help_screen(&help_page,is_menu_mode(m),m->clip_count>1,limit>0,0,0)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("image control cleanup missing %q", want)
		}
	}
}

func TestSilentVideoHasNoMuteOrVolumeControls(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"action=common_combo_action(now,pressed,can_change,has_audio,HUD_HOLD_VBLANKS,ui)",
		"if(has_audio&&(pressed&KEY_UP))",
		"if(has_audio&&(pressed&KEY_DOWN))",
		"if(has_audio&&ui->mute_timer)mute_badge4",
		"if(has_audio&&ui->volume_timer)volume_badge4",
		"show_help_screen(&displayed_page,is_menu_mode(meta),meta->clip_count>1,1,1,has_audio)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("silent-video audio-control guard missing %q", want)
		}
	}
}

func TestPlaybackHUDRestoresVideoAndAudioFeedback(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"frame6(fr,f+1u)",
		"text4n(d,51,69,fr,6,UI_WHITE)",
		"seek_badge4(d,ui->seek_direction",
		"seek_badge3(VRAM0,ui->seek_direction",
		"ui->paused_ui=1;returnACTION_UI_REFRESH",
		"if((ui->hud_timer||ui->paused_ui)&&mode<2)mode=2",
		"ui->mute_timer=mute_hold",
		"ui->volume_timer=AUDIO_VOLUME_HOLD_VBLANKS",
		"start_audio_seek_feedback(ui,forward?1:-1)",
		"bitmap3(d,cx*2u,66,seek_arrow_rows,7,0)",
		"if(c->flags&CLIP_FLAG_MEDIA_AUDIO){ui->hud_mode=2;ui->hud_last_visible=2",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("playback HUD feedback missing %q", want)
		}
	}
}

func TestVideoStartsWithHUDHidden(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"REG_WAITCNT=0x4317;ui.volume_level=2;ui.hud_mode=0;ui.hud_last_visible=2",
		"if(c->flags&CLIP_FLAG_MEDIA_AUDIO){ui->hud_mode=2;ui->hud_last_visible=2",
		"ui->hud_mode=0;ui->hud_last_visible=2;if(!(c->flags&CLIP_FLAG_AUDIO))",
		"render_pixels_rows(p,d,mode==2?67u:80u)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("video hidden-HUD startup behavior missing %q", want)
		}
	}
}

func TestAudioClockUsesIncrementalHUDRefresh(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"native_audio_clock(conststructClipDescriptor*c,u32frame,conststructPlayerUI*ui)",
		"if(sec!=lastsec){native_audio_clock(c,f,ui);lastsec=sec;}",
		"rect3(VRAM0,8,144,38,10,0)",
		"rect3(VRAM0,8,156,224,4,0x2108)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("incremental audio HUD refresh missing %q", want)
		}
	}
	if strings.Contains(src, "if((f&15)==0)native_refresh_audio_ui") {
		t.Fatal("audio playback must not periodically redraw the complete Now Playing HUD")
	}
}

func TestAudioTemporaryFeedbackUsesDirtyRegions(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"native_refresh_audio_badges(conststructClipDescriptor*c,conststructPlayerUI*ui)",
		"native_restore_audio_badges(c);native_draw_audio_badges(c,ui)",
		"native_audio_clock(c,f,ui);native_refresh_audio_badges(c,ui)",
		"ui->volume_timer=AUDIO_VOLUME_HOLD_VBLANKS;audio_apply_state(ui);native_refresh_audio_badges(c,ui)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("audio dirty-region feedback missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"ui->volume_timer=AUDIO_VOLUME_HOLD_VBLANKS;audio_apply_state(ui);native_refresh_audio_ui",
		"start_audio_seek_feedback(ui,forward?1:-1);f=target;base_sample=seek_value(c,f);paused_sample=base_sample;audio_start_at(c,base_sample,paused,ui);playback_timer_reset();if(paused)playback_timer_pause();native_refresh_audio_ui",
		"if(has_audio&&!next_frame_valid&&frame+2<clip->frame_count)",
	} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("unexpected broad/video-side behavior remains: %q", forbidden)
		}
	}
}

func TestPlayerRestoresFastGamePakWaitState(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"#defineREG_WAITCNTREG16(0x04000204)",
		"REG_WAITCNT=0x4317",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("Game Pak timing restore missing %q", want)
		}
	}
}

func TestVideoPCMHasHardDescriptorEndGuard(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"staticu32pcm_guard_ticks;staticintpcm_guard_active",
		"u32start=v&~3u;u32end=c->audio_sample_count?c->audio_sample_count:c->audio_size",
		"pcm_guard_ticks=end-start;pcm_guard_active=1",
		"if(pcm_guard_active&&e>=pcm_guard_ticks){REG_TM0CNT_H=0;REG_DMA1CNT_H=0;REG_SOUNDCNT_H=0x0800;pcm_guard_active=0;}",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("PCM end guard missing %q", want)
		}
	}
}

func TestDirectSoundStateMatchesStableVideoPlayer(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"u16v=reset?0x0800u:0u",
		"v|=0x0300u",
		"if(ui->volume_level>=2)v|=0x0004u",
		"REG_SOUNDCNT_H=sound_control(ui,0)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("stable Direct Sound control missing %q", want)
		}
	}
}

func TestFullVideoHUDSkipsOnlyCoveredRows(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"render_pixels_rows(constu8*s,volatileu16*d,u32rows)",
		"render_pixels_rows(p,d,mode==2?67u:80u)",
		"if((ui->hud_timer||ui->paused_ui)&&mode<2)mode=2",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("covered-row render optimization missing %q", want)
		}
	}
	if strings.Contains(src, "target=frame+1;structPlaybackClocktarget_clock") || strings.Contains(src, "frame+2<clip->frame_count") {
		t.Fatal("video frame dropping/catch-up must not be present")
	}
}

func TestPlayerBuildUsesSizeStableARMFlags(t *testing.T) {
	for _, path := range []string{"player/build.sh", "player/build.ps1"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		for _, want := range []string{"-fomit-frame-pointer", "-enable-machine-outliner", "-machine-outliner-reruns=3"} {
			if !strings.Contains(src, want) {
				t.Fatalf("%s missing player-size build flag %q", path, want)
			}
		}
	}
}

func TestV0131ImageHUDHasOnlyHiddenAndShownStates(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"if(c->flags&CLIP_FLAG_MEDIA_IMAGE){ui->hud_mode=0;ui->hud_last_visible=1",
		"if(action==ACTION_UI_REFRESH&&ui->hud_mode==2)ui->hud_mode=0",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("v0.13.1 image HUD behavior missing %q", want)
		}
	}
}

func TestV0131AudioHUDKeepsCoverVisible(t *testing.T) {
	src := compactSource(playerSource(t))
	for _, want := range []string{
		"staticvoiddim3(volatileu16*d,u32y,u32h)",
		"if(mode==2){dim3(VRAM0,104,56)",
		"elsedim3(VRAM0,140,20)",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("v0.13.1 cover-visible audio HUD missing %q", want)
		}
	}
	if strings.Contains(src, "if(mode==2){rect3(VRAM0,0,104,240,56,0)") {
		t.Fatal("full audio HUD must not replace the lower cover with a solid black panel")
	}
}
