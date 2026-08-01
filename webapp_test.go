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
	if !bytes.Contains(page, []byte(`const TOKEN="abc123"`)) {
		t.Fatal("token not embedded")
	}
	if !bytes.Contains(page, []byte("GBA Video Maker 0.8.0")) {
		t.Fatal("version missing")
	}
	for _, want := range []string{"./icon.png", "Smooth — 14.93 fps", "End (blank = full video)", "prefers-color-scheme:dark"} {
		if !bytes.Contains(page, []byte(want)) {
			t.Fatalf("page is missing %q", want)
		}
	}
	if bytes.Contains(page, []byte("🎞️")) || bytes.Contains(page, []byte("class=\"brand\"")) {
		t.Fatal("removed welcome branding returned")
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

func TestVerifySHA256(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x")
	os.WriteFile(p, []byte("abc"), 0644)
	const sum = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if err := verifySHA256(p, sum); err != nil {
		t.Fatal(err)
	}
	if verifySHA256(p, strings.Repeat("0", 64)) == nil {
		t.Fatal("bad hash accepted")
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
	base := ProjectOptions{Inputs: []ClipInput{{a, "A.mkv", "CAT"}, {b, "B.mkv", "INTRO"}}, FFmpegPath: ff, Start: 0, End: .6, Speed: 1, VBlanks: 8, FitMode: "fit", AudioMode: "none", Volume: 1, RomTitle: "COLLECTION", SeekSeconds: 10, Resume: true, Compression: "delta", PaletteMode: "scene", DitherMode: "off", KeyInterval: 15}
	base.OutputPath = filepath.Join(dir, "menu.gba")
	base.OutputMode = "menu"
	res, err := convertProject(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	rom, _ := os.ReadFile(res.OutputPath)
	if binary.LittleEndian.Uint16(rom[metadataOffset+8:]) != 2 {
		t.Fatal("menu clip count")
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
