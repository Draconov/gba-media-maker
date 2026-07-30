package main

import (
	"bytes"
	"encoding/json"
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
	go func() {
		done <- runWebApp(func(url string) error {
			urlCh <- url
			return nil
		})
	}()

	var url string
	select {
	case url = <-urlCh:
	case <-time.After(3 * time.Second):
		t.Fatal("app server did not start promptly")
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	page, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || !bytes.Contains(page, []byte("Drag and drop a video here")) {
		t.Fatalf("unexpected opening page: status=%d", resp.StatusCode)
	}

	closeURL := strings.TrimSuffix(url, "/") + "/api/close"
	req, err := http.NewRequest(http.MethodPost, closeURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("close status = %d", resp.StatusCode)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("app server did not shut down")
	}
}

func TestRenderPageEmbedsSessionToken(t *testing.T) {
	page, err := renderPage("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(page, []byte(`const TOKEN="abc123";`)) {
		t.Fatalf("session token was not rendered safely: %s", page[:min(len(page), 300)])
	}
	if bytes.Contains(bytes.ToLower(page), []byte("portable app •")) {
		t.Fatal("removed footer text returned")
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := map[string]string{
		"hello-world.mp4": "HELLO WORLD",
		"":                "GBA VIDEO",
		"123456789012345": "123456789012",
		"кіт":             "GBA VIDEO",
	}
	for input, want := range tests {
		if got := normalizeTitle(input); got != want {
			t.Fatalf("normalizeTitle(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestVerifySHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.bin")
	if err := os.WriteFile(path, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	const sum = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if err := verifySHA256(path, sum); err != nil {
		t.Fatalf("valid hash rejected: %v", err)
	}
	if err := verifySHA256(path, strings.Repeat("0", 64)); err == nil {
		t.Fatal("invalid hash accepted")
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timed out waiting for state")
}

func postJSON(t *testing.T, url string, value any) *http.Response {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestHTTPUploadInspectConvertDownload(t *testing.T) {
	ffmpeg := commandExists("ffmpeg")
	if ffmpeg == "" {
		t.Skip("ffmpeg is not installed in the test environment")
	}
	input := filepath.Join(t.TempDir(), "fixture.mkv")
	cmd := exec.Command(
		ffmpeg, "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100",
		"-t", "1.5", "-c:v", "ffv1", "-c:a", "pcm_s16le", "-shortest", input,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("could not create integration fixture: %v\n%s", err, output)
	}

	state := &appState{
		token: "testtoken", sessionDir: t.TempDir(), ffmpegPath: ffmpeg,
		engineStatus: "ready", engineProgress: 100, engineMessage: "ready",
		inspectStatus: "idle", lastHeartbeat: time.Now(), shutdown: func() {},
	}
	page, err := renderPage(state.token)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(state.routes(page))
	defer server.Close()
	base := server.URL + "/" + state.token + "/api"

	var upload bytes.Buffer
	mw := multipart.NewWriter(&upload)
	part, err := mw.CreateFormFile("video", filepath.Base(input))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, src); err != nil {
		src.Close()
		t.Fatal(err)
	}
	src.Close()
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, base+"/upload", &upload)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload status %d: %s", resp.StatusCode, b)
	}

	waitFor(t, 15*time.Second, func() bool { return state.snapshot().InspectStatus == "ready" })
	snap := state.snapshot()
	if snap.Media == nil || snap.Media.Width != 160 || snap.Media.Height != 120 {
		t.Fatalf("unexpected media info: %+v", snap.Media)
	}

	conv := convertRequest{Start: "0", End: "1.2", Speed: 1, FPS: "classic", Fit: "crop", Audio: "mix", Volume: 100, RomTitle: "WEB TEST"}
	resp = postJSON(t, base+"/convert", conv)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("convert status %d: %s", resp.StatusCode, body)
	}

	waitFor(t, 45*time.Second, func() bool {
		s := state.snapshot()
		return !s.Converting && (s.Result != nil || s.ConvertError != "")
	})
	snap = state.snapshot()
	if snap.ConvertError != "" {
		t.Fatalf("conversion failed: %s", snap.ConvertError)
	}
	if snap.Result == nil {
		t.Fatal("conversion returned no result")
	}

	resp, err = http.Get(base + "/download")
	if err != nil {
		t.Fatal(err)
	}
	rom, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("download status %d", resp.StatusCode)
	}
	if len(rom) < 1<<20 || len(rom) > romLimit || len(rom)&(len(rom)-1) != 0 {
		t.Fatalf("invalid ROM size: %d", len(rom))
	}
	if len(rom) <= 0xBD {
		t.Fatal("ROM is too short")
	}
	var check byte
	for i := 0xA0; i <= 0xBC; i++ {
		check -= rom[i]
	}
	check -= 0x19
	if rom[0xBD] != check {
		t.Fatalf("bad GBA header checksum: got %02x want %02x", rom[0xBD], check)
	}
	if string(rom[metadataOffset:metadataOffset+4]) != "GBV2" {
		t.Fatalf("missing GBV2 metadata marker")
	}

	flags := uint16(rom[metadataOffset+6]) | uint16(rom[metadataOffset+7])<<8
	if flags&1 == 0 {
		t.Fatalf("audio flag was not set in ROM metadata")
	}
}
