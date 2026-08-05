package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunWebAppLifecycle(t *testing.T) {
	urlCh := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- runWebApp(func(url string) error { urlCh <- url; return nil }) }()
	var url string
	select {
	case url = <-urlCh:
	case <-time.After(3 * time.Second):
		t.Fatal("app server did not start")
	}
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytes.Contains(page, []byte("Drag and drop videos here")) {
		t.Fatalf("unexpected page status=%d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, strings.TrimSuffix(url, "/")+"/api/close", nil)
	parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
	req.Header.Set("X-GBA-Token", parts[len(parts)-1])
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("app did not stop")
	}
}

func TestRenderPageEmbedsSessionToken(t *testing.T) {
	page, err := renderPage("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(page, []byte(`name="gbavm-session-token" content="abc123"`)) {
		t.Fatal("token not embedded")
	}
	if !bytes.Contains(page, []byte("GBA Video Maker 0.10.0")) {
		t.Fatal("version missing")
	}
	for _, want := range []string{"./icon.png", "./style.css", "./menu-themes.js", "./app.js", "Smooth — 14.93 fps", "End (blank = full video)", "Optimize to fit 32 MiB", "Fit with bars", "Single ROM", "Menu design", "Blue Wave — animated", "Custom image or GIF"} {
		if !bytes.Contains(page, []byte(want)) {
			t.Fatalf("page is missing %q", want)
		}
	}
	if bytes.Contains(page, []byte("🎞️")) || bytes.Contains(page, []byte("class=\"brand\"")) {
		t.Fatal("removed welcome branding returned")
	}
	if !bytes.Contains(appCSS, []byte("prefers-color-scheme:dark")) {
		t.Fatal("system dark-mode stylesheet missing")
	}
	if !bytes.Contains(appJS, []byte("gbavm-session-token")) {
		t.Fatal("external application script missing")
	}
}

func TestLocalServerSecurityGuards(t *testing.T) {
	state := &appState{token: "securetoken", allowedHost: "127.0.0.1:45678", lastHeartbeat: time.Now(), shutdown: func() {}}
	page, err := renderPage(state.token)
	if err != nil {
		t.Fatal(err)
	}
	handler := state.routes(page)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:45678/securetoken/", nil)
	req.Host = state.allowedHost
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid local request returned %d", rec.Code)
	}
	if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers missing")
	}

	req = httptest.NewRequest(http.MethodGet, "http://evil.example/securetoken/", nil)
	req.Host = "evil.example"
	req.RemoteAddr = "127.0.0.1:54321"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("bad host returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:45678/securetoken/api/heartbeat", nil)
	req.Host = state.allowedHost
	req.RemoteAddr = "127.0.0.1:54321"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing token returned %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:45678/securetoken/api/heartbeat", nil)
	req.Host = state.allowedHost
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-GBA-Token", state.token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("valid token returned %d", rec.Code)
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := map[string]string{"hello-world.mp4": "HELLO WORLD", "": "GBA VIDEO", "123456789012345": "123456789012", "кіт": "GBA VIDEO"}
	for in, want := range tests {
		if got := normalizeTitle(in); got != want {
			t.Fatalf("%q => %q want %q", in, got, want)
		}
	}
}

func waitFor(t *testing.T, d time.Duration, fn func() bool) {
	t.Helper()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if fn() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timed out")
}
func postJSON(t *testing.T, url string, v any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(v)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func makeFixture(t *testing.T, ffmpeg, path string, duration float64, color string) {
	t.Helper()
	cmd := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=160x120:rate=15", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", strconvFloat(duration), "-vf", "hue=h="+color, "-c:v", "ffv1", "-c:a", "pcm_s16le", "-shortest", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v\n%s", err, out)
	}
}
func strconvFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", v), "0"), ".")
}

func uploadFiles(t *testing.T, url string, paths []string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range paths {
		part, err := mw.CreateFormFile("video", filepath.Base(p))
		if err != nil {
			t.Fatal(err)
		}
		f, _ := os.Open(p)
		io.Copy(part, f)
		f.Close()
	}
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 202 {
		t.Fatalf("upload %d %s", resp.StatusCode, b)
	}
}

func decodeCompressedFrame(rom []byte, desc []byte, frame int) []byte {
	video := int(binary.LittleEndian.Uint32(desc[8:12]))
	idx := int(binary.LittleEndian.Uint32(desc[12:16]))
	get := func(n int) int { return video + int(binary.LittleEndian.Uint32(rom[idx+n*4:idx+n*4+4])) }
	base := frame
	for base > 0 {
		r := get(base)
		if binary.LittleEndian.Uint32(rom[r:r+4]) == 0 {
			break
		}
		base--
	}
	out := make([]byte, frameBytes)
	r := get(base)
	copy(out, rom[r+8:r+8+frameBytes])
	for base < frame {
		base++
		r = get(base)
		typ := binary.LittleEndian.Uint32(rom[r : r+4])
		size := int(binary.LittleEndian.Uint32(rom[r+4 : r+8]))
		if typ == 0 {
			copy(out, rom[r+8:r+8+frameBytes])
			continue
		}
		p := r + 8
		pos := 0
		end := p + size
		for p+4 <= end && pos < frameBytes {
			skip := int(binary.LittleEndian.Uint16(rom[p : p+2]))
			run := int(binary.LittleEndian.Uint16(rom[p+2 : p+4]))
			p += 4
			pos += skip
			copy(out[pos:pos+run], rom[p:p+run])
			pos += run
			p += run
		}
	}
	return out
}

func TestHTTPUploadInspectConvertDownloadV5(t *testing.T) {
	ff := commandExists("ffmpeg")
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	input := filepath.Join(t.TempDir(), "fixture.mkv")
	makeFixture(t, ff, input, 1.5, "20")
	state := &appState{token: "testtoken", sessionDir: t.TempDir(), ffmpegPath: ff, engineStatus: "ready", engineProgress: 100, engineMessage: "ready", inspectStatus: "idle", lastHeartbeat: time.Now(), shutdown: func() {}}
	page, _ := renderPage(state.token)
	server := httptest.NewServer(state.routes(page))
	defer server.Close()
	base := server.URL + "/" + state.token + "/api"
	uploadFiles(t, base+"/upload", []string{input})
	waitFor(t, 15*time.Second, func() bool { return state.snapshot().InspectStatus == "ready" })
	snap := state.snapshot()
	if len(snap.Videos) != 1 || snap.Videos[0].Info.Width != 160 {
		t.Fatalf("bad media %+v", snap.Videos)
	}
	resp, err := http.Get(server.URL + "/" + state.token + "/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	icon, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(icon) < 1000 {
		t.Fatalf("bad app icon response status=%d bytes=%d", resp.StatusCode, len(icon))
	}
	for _, asset := range []string{"style.css", "menu-themes.js", "app.js"} {
		resp, err = http.Get(server.URL + "/" + state.token + "/" + asset)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || len(body) < 100 {
			t.Fatalf("bad static asset %s status=%d bytes=%d", asset, resp.StatusCode, len(body))
		}
	}
	resp, err = http.Get(base + "/preview?index=0&time=1.5&fit=crop")
	if err != nil {
		t.Fatal(err)
	}
	preview, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(preview) < 100 {
		t.Fatalf("end preview failed status=%d bytes=%d body=%s", resp.StatusCode, len(preview), preview)
	}
	conv := convertRequest{Start: "0", End: "1.2", Speed: 1, FPS: "classic", Fit: "crop", Audio: "mix", Volume: 100, RomTitle: "WEB TEST", SeekSeconds: 5, Normalize: true, Limiter: true, Resume: true, Compression: "delta", PaletteMode: "shared", DitherMode: "ordered", OutputMode: "rom"}
	resp = postJSON(t, base+"/convert", conv)
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 202 {
		t.Fatalf("convert %d %s", resp.StatusCode, b)
	}
	waitFor(t, 60*time.Second, func() bool { s := state.snapshot(); return !s.Converting && (s.Result != nil || s.ConvertError != "") })
	snap = state.snapshot()
	if snap.ConvertError != "" {
		t.Fatal(snap.ConvertError)
	}
	resp, err = http.Get(base + "/download")
	if err != nil {
		t.Fatal(err)
	}
	rom, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(rom) < 1<<20 || len(rom) > romLimit || len(rom)&(len(rom)-1) != 0 {
		t.Fatalf("bad ROM size %d", len(rom))
	}
	if string(rom[metadataOffset:metadataOffset+4]) != "GBV5" {
		t.Fatal("GBV5 missing")
	}
	if binary.LittleEndian.Uint16(rom[metadataOffset+4:]) != 5 {
		t.Fatal("version")
	}
	if binary.LittleEndian.Uint16(rom[metadataOffset+8:]) != 1 {
		t.Fatal("clip count")
	}
	if binary.LittleEndian.Uint16(rom[metadataOffset+6:])&1 == 0 {
		t.Fatal("resume flag")
	}
	clipOff := int(binary.LittleEndian.Uint32(rom[metadataOffset+12:]))
	desc := rom[clipOff : clipOff+clipDescriptorSize]
	frames := int(binary.LittleEndian.Uint32(desc[0:4]))
	flags := binary.LittleEndian.Uint16(desc[50:52])
	if frames < 2 || flags&1 == 0 || flags&4 == 0 {
		t.Fatalf("bad desc frames=%d flags=%x", frames, flags)
	}
	if binary.LittleEndian.Uint16(desc[52:54]) != 5 || binary.LittleEndian.Uint32(desc[40:44]) != 50 {
		t.Fatal("seek metadata")
	}
	pal := int(binary.LittleEndian.Uint32(desc[24:28]))
	want := []uint16{0, 0x18C6, 0x7FFF, 0x037F, 0x001F, 0x03E0}
	for i, w := range want {
		got := binary.LittleEndian.Uint16(rom[pal+(250+i)*2:])
		if got != w {
			t.Fatalf("palette %d=%x", i, got)
		}
	}
	for _, f := range []int{0, frames / 2, frames - 1} {
		pix := decodeCompressedFrame(rom, desc, f)
		if len(pix) != frameBytes {
			t.Fatal("decode size")
		}
		for _, v := range pix {
			if v >= videoPaletteColors {
				t.Fatalf("reserved palette index %d", v)
			}
		}
	}
}

func TestMultiClipMenuAndBatch(t *testing.T) {
	ff := commandExists("ffmpeg")
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mkv")
	b := filepath.Join(dir, "b.mkv")
	makeFixture(t, ff, a, .7, "10")
	makeFixture(t, ff, b, .7, "120")
	base := ProjectOptions{Inputs: []ClipInput{{InputPath: a, Name: "A.mkv", Title: "CAT"}, {InputPath: b, Name: "B.mkv", Title: "INTRO"}}, FFmpegPath: ff, Start: 0, End: .6, Speed: 1, VBlanks: 8, FitMode: "fit", AudioMode: "none", Volume: 1, RomTitle: "COLLECTION", SeekSeconds: 10, Resume: true, Compression: "delta", PaletteMode: "scene", DitherMode: "off", KeyInterval: 15}
	base.OutputPath = filepath.Join(dir, "menu.gba")
	base.OutputMode = "menu"
	base.MenuTheme = &MenuThemeOptions{
		ID: "test-menu", Kind: "frames", Palette: make([]byte, 512),
		Frames:       [][]byte{make([]byte, frameBytes), make([]byte, frameBytes)},
		FrameVBlanks: 12, UIColor: 0x7FFF, SelectedColor: 0x037F,
		Outline: true, OutlineColor: 0,
	}
	res, err := convertProject(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	rom, _ := os.ReadFile(res.OutputPath)
	if binary.LittleEndian.Uint16(rom[metadataOffset+8:]) != 2 {
		t.Fatal("menu clip count")
	}
	if binary.LittleEndian.Uint16(rom[metadataOffset+6:])&0x0004 != 0 {
		t.Fatal("obsolete menu preview flag present")
	}
	themeOffset := int(binary.LittleEndian.Uint32(rom[metadataOffset+48:]))
	if themeOffset <= assetOffset || binary.LittleEndian.Uint32(rom[themeOffset:]) != menuThemeMagic {
		t.Fatalf("menu theme was not embedded at %#x", themeOffset)
	}
	if binary.LittleEndian.Uint16(rom[themeOffset+18:]) != 12 || binary.LittleEndian.Uint16(rom[themeOffset+20:]) != 1 {
		t.Fatal("menu animation or outline metadata missing")
	}
	base.OutputPath = filepath.Join(dir, "batch.zip")
	base.OutputMode = "batch"
	res, err = convertProject(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 2 {
		t.Fatalf("batch files=%d", len(zr.File))
	}
}

func TestPerClipOverrides(t *testing.T) {
	ff := commandExists("ffmpeg")
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	dir := t.TempDir()
	a := filepath.Join(dir, "default.mkv")
	b := filepath.Join(dir, "custom.mkv")
	makeFixture(t, ff, a, 1.0, "15")
	makeFixture(t, ff, b, 1.0, "90")
	opt := ProjectOptions{
		Inputs: []ClipInput{
			{InputPath: a, Name: "default.mkv", Title: "DEFAULT"},
			{InputPath: b, Name: "custom.mkv", Title: "CUSTOM", Custom: true, Start: .1, End: .8, Speed: 2, FitMode: "crop", AudioMode: "none", Volume: 1, Loop: true, PaletteMode: "shared", DitherMode: "off"},
		},
		OutputPath: filepath.Join(dir, "per-clip.gba"), FFmpegPath: ff,
		Start: 0, End: .9, Speed: 1, VBlanks: 6, FitMode: "fit", AudioMode: "mix", Volume: 1,
		RomTitle: "PER CLIP", SeekSeconds: 5, Compression: "delta", PaletteMode: "shared", DitherMode: "ordered", OutputMode: "menu", KeyInterval: 30,
	}
	if _, err := convertProject(opt, nil); err != nil {
		t.Fatal(err)
	}
	rom, err := os.ReadFile(opt.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	table := int(binary.LittleEndian.Uint32(rom[metadataOffset+12:]))
	first := rom[table : table+clipDescriptorSize]
	second := rom[table+clipDescriptorSize : table+2*clipDescriptorSize]
	firstFlags := binary.LittleEndian.Uint16(first[50:52])
	secondFlags := binary.LittleEndian.Uint16(second[50:52])
	if firstFlags&CLIP_FLAG_AUDIO_TEST == 0 {
		t.Fatal("default clip audio was lost")
	}
	if secondFlags&CLIP_FLAG_AUDIO_TEST != 0 || secondFlags&CLIP_FLAG_LOOP_TEST == 0 {
		t.Fatalf("custom flags=%x", secondFlags)
	}
	firstFrames := binary.LittleEndian.Uint32(first[0:4])
	secondFrames := binary.LittleEndian.Uint32(second[0:4])
	if secondFrames >= firstFrames {
		t.Fatalf("custom speed/trim not applied: first=%d second=%d", firstFrames, secondFrames)
	}
	if got := strings.TrimSpace(strings.TrimRight(string(second[60:72]), "\x00")); got != "CUSTOM" {
		t.Fatalf("custom title=%q", got)
	}
}

const (
	CLIP_FLAG_AUDIO_TEST = 0x0001
	CLIP_FLAG_LOOP_TEST  = 0x0002
)

func TestReorderVideos(t *testing.T) {
	state := &appState{videos: []uploadedVideo{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}}
	if err := state.reorderVideos([]string{"c", "a", "b"}); err != nil {
		t.Fatal(err)
	}
	if got := state.videos[0].ID + state.videos[1].ID + state.videos[2].ID; got != "cab" {
		t.Fatalf("order=%s", got)
	}
	if err := state.reorderVideos([]string{"a", "a", "b"}); err == nil {
		t.Fatal("duplicate order accepted")
	}
}

func TestAutomaticLongVideoSplit(t *testing.T) {
	ff := commandExists("ffmpeg")
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "long-test.mkv")
	makeFixture(t, ff, input, 8.0, "45")
	output := filepath.Join(dir, "long-test.zip")
	opt := ProjectOptions{
		Inputs:     []ClipInput{{InputPath: input, Name: "MY_VIDEO.mp4", Title: "MY VIDEO"}},
		OutputPath: output, FFmpegPath: ff, Start: 0, End: 8, Speed: 1,
		VBlanks: 8, FitMode: "fit", AudioMode: "none", Volume: 1,
		RomTitle: "MY VIDEO", SeekSeconds: 5, Compression: "none",
		PaletteMode: "shared", DitherMode: "off", OutputMode: "longsplit", KeyInterval: 30,
	}
	res, err := convertLongVideoSplitWithBudget(opt, 260*1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OutputKind != "zip" || res.ClipCount < 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
	zr, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	romCount := 0
	var manifest string
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".gba") {
			romCount++
			want := fmt.Sprintf("MY_VIDEO_PART_%02d.gba", romCount)
			if f.Name != want {
				t.Fatalf("part name=%q want %q", f.Name, want)
			}
			if f.UncompressedSize64 > romLimit {
				t.Fatalf("part exceeds ROM limit: %d", f.UncompressedSize64)
			}
		}
		if f.Name == "PARTS.txt" {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, _ := io.ReadAll(r)
			r.Close()
			manifest = string(b)
		}
	}
	if romCount != res.ClipCount || manifest == "" {
		t.Fatalf("roms=%d result=%d manifest=%q", romCount, res.ClipCount, manifest)
	}
	if !strings.Contains(manifest, "00:00:00.000") || !strings.Contains(manifest, "00:00:08.000") {
		t.Fatalf("manifest timestamps missing:\n%s", manifest)
	}
}

func TestBuildOptionsUsesNormalSingleROMModeForAutomaticSplitting(t *testing.T) {
	dir := t.TempDir()
	state := &appState{
		sessionDir: dir,
		ffmpegPath: "ffmpeg",
		videos: []uploadedVideo{{
			ID: "one", Path: filepath.Join(dir, "movie.mp4"), Name: "movie.mp4",
			Info:   &MediaInfo{Duration: 3000, Width: 1920, Height: 1080, FPS: 30, AudioStreams: 1, AudioChannels: 2},
			Status: "ready",
		}},
	}
	req := convertRequest{
		Start: "0", Speed: 1, FPS: "compact", Fit: "fit", Audio: "mix", Volume: 100,
		RomTitle: "MY VIDEO", SeekSeconds: 5, Compression: "delta",
		PaletteMode: "shared", DitherMode: "ordered", OutputMode: "rom",
	}
	opt, _, err := state.buildOptions(req)
	if err != nil {
		t.Fatal(err)
	}
	if opt.OutputMode != "rom" || filepath.Ext(opt.OutputPath) != ".gba" {
		t.Fatalf("mode=%q output=%q", opt.OutputMode, opt.OutputPath)
	}
}

func TestSingleROMAutomaticallySplitsWhenBudgetIsExceeded(t *testing.T) {
	ff := commandExists("ffmpeg")
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "automatic-test.mkv")
	makeFixture(t, ff, input, 8.0, "45")
	output := filepath.Join(dir, "automatic-test.gba")
	opt := ProjectOptions{
		Inputs:     []ClipInput{{InputPath: input, Name: "MY_VIDEO.mp4", Title: "MY VIDEO"}},
		OutputPath: output, FFmpegPath: ff, Start: 0, End: 8, Speed: 1,
		VBlanks: 8, FitMode: "fit", AudioMode: "none", Volume: 1,
		RomTitle: "MY VIDEO", SeekSeconds: 5, Compression: "none",
		PaletteMode: "shared", DitherMode: "off", OutputMode: "rom", KeyInterval: 30,
	}
	res, err := convertProjectWithAutoSplitBudget(opt, 260*1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.AutoSplit || res.OutputKind != "zip" || res.ClipCount < 2 {
		t.Fatalf("unexpected automatic result: %+v", res)
	}
	if filepath.Ext(res.OutputPath) != ".zip" || !strings.Contains(filepath.Base(res.OutputPath), "_PARTS") {
		t.Fatalf("automatic output path=%q", res.OutputPath)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("oversized temporary ROM was not removed: %v", err)
	}
}

func TestLongVideoControlsAndProgressUIAreEmbedded(t *testing.T) {
	page, err := renderPage("controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`id="splitVideo"`, `id="longSplitControls"`, `id="splitBudget"`, `id="maxPartDuration"`,
		`id="chapterAware"`, `id="partTitleScreens"`, `id="resumeLongSplit"`, "Estimated output:",
	} {
		if !bytes.Contains(page, []byte(want)) && !bytes.Contains(appJS, []byte(want)) {
			t.Fatalf("long-video UI is missing %q", want)
		}
	}
	if !bytes.Contains(appCSS, []byte("white-space:pre-line")) {
		t.Fatal("multi-line progress styling is missing")
	}
}

func TestBuildOptionsUsesCleanExportNameAndSplitSettings(t *testing.T) {
	dir := t.TempDir()
	state := &appState{
		sessionDir: dir,
		ffmpegPath: "ffmpeg",
		videos: []uploadedVideo{{
			ID: "one", Path: filepath.Join(dir, "movie.mp4"), Name: "movie.mp4",
			Info: &MediaInfo{Duration: 3000, Width: 1920, Height: 1080, FPS: 30, AudioStreams: 1}, Status: "ready",
		}},
	}
	req := convertRequest{
		Start: "0", Speed: 1, FPS: "compact", Fit: "fit", Audio: "mix", Volume: 100,
		RomTitle: "MOVIE", SeekSeconds: 5, Compression: "delta", PaletteMode: "shared", DitherMode: "ordered",
		OutputMode: "rom", SplitVideo: true, SplitBudgetMiB: 20, MaxPartDuration: "1:05", ChapterAware: true,
		PartTitleScreens: true, ResumeLongSplit: true,
	}
	opt, _, err := state.buildOptions(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Base(opt.OutputPath); got != "movie.gba" {
		t.Fatalf("export name=%q, want movie.gba", got)
	}
	if opt.SplitBudgetMiB != 20 || opt.MaxPartMinutes < 1.0832 || opt.MaxPartMinutes > 1.0834 || !opt.ChapterAware || !opt.PartTitleScreens || !opt.ResumeLongSplit {
		t.Fatalf("split settings were not preserved: %+v", opt)
	}
}

func TestBuildOptionsIgnoresHiddenManualSplitRules(t *testing.T) {
	dir := t.TempDir()
	state := &appState{
		sessionDir: dir,
		ffmpegPath: "ffmpeg",
		videos: []uploadedVideo{{
			ID: "one", Path: filepath.Join(dir, "movie.mp4"), Name: "movie.mp4",
			Info: &MediaInfo{Duration: 3000, Width: 1920, Height: 1080, FPS: 30, AudioStreams: 1}, Status: "ready",
		}},
	}
	req := convertRequest{
		Start: "0", Speed: 1, FPS: "compact", Fit: "fit", Audio: "mix", Volume: 100,
		RomTitle: "MOVIE", SeekSeconds: 5, Compression: "delta", PaletteMode: "shared", DitherMode: "ordered",
		OutputMode: "rom", SplitVideo: false, SplitBudgetMiB: 20, MaxPartDuration: "not-used",
		ChapterAware: false, PartTitleScreens: false, ResumeLongSplit: false,
	}
	opt, _, err := state.buildOptions(req)
	if err != nil {
		t.Fatal(err)
	}
	if opt.SplitBudgetMiB != 32 || opt.MaxPartMinutes != 0 {
		t.Fatalf("hidden manual rules leaked into normal Single ROM mode: %+v", opt)
	}
	if !opt.ChapterAware || !opt.PartTitleScreens || !opt.ResumeLongSplit {
		t.Fatalf("automatic overflow split defaults were not restored: %+v", opt)
	}
}

func TestParseMaximumPartDuration(t *testing.T) {
	seconds, err := parseMaximumPartDuration("1:05", 0)
	if err != nil || seconds != 65 {
		t.Fatalf("1:05 parsed as %v, %v", seconds, err)
	}
	for _, value := range []string{"1:5", "1:60", "1", "-1:05"} {
		if _, err := parseMaximumPartDuration(value, 0); err == nil {
			t.Fatalf("invalid duration %q was accepted", value)
		}
	}
}

func TestLongSplitEstimateUsesSizeAndDurationLimits(t *testing.T) {
	opt := ProjectOptions{
		Start: 0, End: 50 * 60, Speed: 1, VBlanks: 8, AudioMode: "mix",
		Compression: "delta", PaletteMode: "shared", MaxPartMinutes: 8,
	}
	info := MediaInfo{Duration: 50 * 60, AudioStreams: 1}
	parts := estimateLongSplitParts(opt, info, 20*1024*1024)
	if parts < 7 {
		t.Fatalf("estimated parts=%d, want at least 7 for a 50-minute source capped at 8 minutes", parts)
	}
}

func TestChapterAwareSplitPrefersNearbyBoundary(t *testing.T) {
	chapters := []float64{120, 300, 470, 720}
	if got := chapterSplitEnd(chapters, 300, 500, 900); got != 470 {
		t.Fatalf("chapter split=%v, want 470", got)
	}
	if got := chapterSplitEnd(chapters, 300, 700, 900); got != 700 {
		t.Fatalf("distant chapter should not be forced, got %v", got)
	}
}

func TestPartTitleScreenMetadata(t *testing.T) {
	ff := commandExists("ffmpeg")
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "title-screen.mkv")
	makeFixture(t, ff, input, .6, "35")
	output := filepath.Join(dir, "title-screen.gba")
	opt := ProjectOptions{
		Inputs:     []ClipInput{{InputPath: input, Name: "MOVIE.mp4", Title: "MOVIE"}},
		OutputPath: output, FFmpegPath: ff, Start: 0, End: .5, Speed: 1, VBlanks: 8,
		FitMode: "fit", AudioMode: "none", Volume: 1, RomTitle: "MOVIE P02", SeekSeconds: 5,
		Compression: "delta", PaletteMode: "shared", DitherMode: "off", OutputMode: "rom",
		KeyInterval: 30, TitleScreenPart: 2, TitleScreenName: "My Long Movie File",
	}
	if _, err := convertProjectExact(opt, nil); err != nil {
		t.Fatal(err)
	}
	rom, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	flags := binary.LittleEndian.Uint16(rom[metadataOffset+6:])
	part := binary.LittleEndian.Uint32(rom[metadataOffset+20:])
	name := strings.TrimRight(string(rom[metadataOffset+24:metadataOffset+48]), "\x00")
	if flags&0x0004 == 0 || part != 2 || name != "MY LONG MOVIE FILE" {
		t.Fatalf("title-screen metadata flags=%04x part=%d name=%q", flags, part, name)
	}
}

func TestSplitRecoveryStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	want := splitRecoveryState{
		Version: 1, Fingerprint: "abc", SourceName: "movie.mp4", Start: 0, End: 100,
		Cursor: 25, NextPart: 2, EstimatedParts: 4,
		Parts: []splitPartRecord{{FileName: "movie_PART_01.gba", Start: 0, End: 25, FrameCount: 100}},
	}
	if err := saveSplitRecovery(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok := loadSplitRecovery(path, "abc")
	if !ok || got.NextPart != 2 || len(got.Parts) != 1 || got.Parts[0].End != 25 {
		t.Fatalf("recovery state did not round-trip: ok=%v state=%+v", ok, got)
	}
}
