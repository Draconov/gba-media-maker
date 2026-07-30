package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ffmpegDownloadURL = "https://github.com/shaka-project/static-ffmpeg-binaries/releases/download/n8.1.2-1/ffmpeg-win-x64.exe"
	ffmpegSHA256      = "4044b3924c977ad31229d504c5d5b8685f9553124fbaff6e9c99048b42830341"
	maxUploadBytes    = int64(8 * 1024 * 1024 * 1024)
)

type appState struct {
	mu sync.Mutex

	token      string
	sessionDir string
	ffmpegPath string

	engineStatus   string // idle, ready, downloading, error
	engineProgress int
	engineMessage  string

	videoPath     string
	videoName     string
	mediaInfo     *MediaInfo
	inspectStatus string // idle, waiting, inspecting, ready, error
	inspectError  string

	converting      bool
	progress        int
	progressMessage string
	convertError    string
	result          *ConvertResult
	downloadName    string

	lastHeartbeat time.Time
	shutdownOnce  sync.Once
	shutdown      func()
}

type publicState struct {
	EngineStatus    string         `json:"engineStatus"`
	EngineProgress  int            `json:"engineProgress"`
	EngineMessage   string         `json:"engineMessage"`
	VideoName       string         `json:"videoName"`
	InspectStatus   string         `json:"inspectStatus"`
	InspectError    string         `json:"inspectError"`
	Media           *MediaInfo     `json:"media,omitempty"`
	Converting      bool           `json:"converting"`
	Progress        int            `json:"progress"`
	ProgressMessage string         `json:"progressMessage"`
	ConvertError    string         `json:"convertError"`
	Result          *ConvertResult `json:"result,omitempty"`
	DownloadName    string         `json:"downloadName"`
}

type convertRequest struct {
	Start    string  `json:"start"`
	End      string  `json:"end"`
	Speed    float64 `json:"speed"`
	FPS      string  `json:"fps"`
	Fit      string  `json:"fit"`
	Audio    string  `json:"audio"`
	Volume   float64 `json:"volume"`
	Loop     bool    `json:"loop"`
	RomTitle string  `json:"romTitle"`
}

func appDirectory() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func logDiagnostic(section string, recovered any) {
	path := filepath.Join(appDirectory(), "GBA Video Maker.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s: %v\n%s\n", time.Now().Format(time.RFC3339), section, recovered, debug.Stack())
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func locatePortableFFmpeg() string {
	candidates := []string{
		filepath.Join(appDirectory(), "ffmpeg.exe"),
		filepath.Join(appDirectory(), "tools", "ffmpeg.exe"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Size() > 1_000_000 {
			return p
		}
	}
	return ""
}

func newAppState(token, sessionDir string) *appState {
	s := &appState{token: token, sessionDir: sessionDir, lastHeartbeat: time.Now(), inspectStatus: "idle"}
	if ff := locatePortableFFmpeg(); ff != "" {
		s.ffmpegPath = ff
		s.engineStatus = "ready"
		s.engineProgress = 100
		s.engineMessage = "Conversion engine ready"
	} else {
		s.engineStatus = "idle"
		s.engineMessage = "The portable conversion engine will be prepared when a video is selected."
	}
	return s
}

func (s *appState) snapshot() publicState {
	s.mu.Lock()
	defer s.mu.Unlock()
	var media *MediaInfo
	if s.mediaInfo != nil {
		copy := *s.mediaInfo
		media = &copy
	}
	var result *ConvertResult
	if s.result != nil {
		copy := *s.result
		result = &copy
	}
	return publicState{
		EngineStatus: s.engineStatus, EngineProgress: s.engineProgress, EngineMessage: s.engineMessage,
		VideoName: s.videoName, InspectStatus: s.inspectStatus, InspectError: s.inspectError, Media: media,
		Converting: s.converting, Progress: s.progress, ProgressMessage: s.progressMessage,
		ConvertError: s.convertError, Result: result, DownloadName: s.downloadName,
	}
}

func (s *appState) touch() {
	s.mu.Lock()
	s.lastHeartbeat = time.Now()
	s.mu.Unlock()
}

func (s *appState) startEngineDownload() {
	s.mu.Lock()
	if s.engineStatus == "ready" || (s.engineStatus == "downloading" && s.ffmpegPath != "") {
		s.mu.Unlock()
		return
	}
	s.engineStatus = "downloading"
	s.engineProgress = 0
	s.engineMessage = "Preparing the portable conversion engine…"
	// A sentinel prevents duplicate download goroutines.
	s.ffmpegPath = "__downloading__"
	s.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDiagnostic("engine download panic", r)
				s.mu.Lock()
				s.engineStatus = "error"
				s.engineMessage = "Could not prepare FFmpeg. Click Retry."
				s.ffmpegPath = ""
				s.mu.Unlock()
			}
		}()
		target := filepath.Join(appDirectory(), "ffmpeg.exe")
		temp := target + ".download"
		_ = os.Remove(temp)
		err := downloadFileWithProgress(ffmpegDownloadURL, temp, func(done, total int64) {
			p := 0
			if total > 0 {
				p = int(done * 100 / total)
			}
			if p > 99 {
				p = 99
			}
			s.mu.Lock()
			s.engineProgress = p
			s.engineMessage = fmt.Sprintf("Preparing conversion engine… %d%%", p)
			s.mu.Unlock()
		})
		if err == nil {
			err = verifySHA256(temp, ffmpegSHA256)
		}
		if err == nil {
			err = os.Rename(temp, target)
		}
		if err != nil {
			_ = os.Remove(temp)
			s.mu.Lock()
			s.engineStatus = "error"
			s.engineMessage = "Engine download failed: " + err.Error()
			s.ffmpegPath = ""
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		s.ffmpegPath = target
		s.engineStatus = "ready"
		s.engineProgress = 100
		s.engineMessage = "Conversion engine ready"
		pending := s.videoPath != "" && s.inspectStatus == "waiting"
		s.mu.Unlock()
		if pending {
			s.startInspection()
		}
	}()
}

func downloadFileWithProgress(url, path string, progress func(done, total int64)) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "GBA-Video-Maker/0.5.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download returned %s", resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 256*1024)
	var done int64
	last := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			done += int64(n)
			if time.Since(last) > 150*time.Millisecond {
				progress(done, resp.ContentLength)
				last = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	progress(done, resp.ContentLength)
	return f.Sync()
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("safety check failed (SHA-256 mismatch)")
	}
	return nil
}

func (s *appState) setUploaded(path, name string) {
	s.mu.Lock()
	old := s.videoPath
	s.videoPath = path
	s.videoName = name
	s.mediaInfo = nil
	s.inspectError = ""
	s.convertError = ""
	s.result = nil
	s.downloadName = ""
	ready := s.engineStatus == "ready"
	if ready {
		s.inspectStatus = "inspecting"
	} else {
		s.inspectStatus = "waiting"
	}
	s.mu.Unlock()
	if old != "" && old != path {
		_ = os.Remove(old)
	}
	if ready {
		s.startInspection()
	} else {
		s.startEngineDownload()
	}
}

func (s *appState) startInspection() {
	s.mu.Lock()
	if s.videoPath == "" || s.engineStatus != "ready" || s.ffmpegPath == "" || s.ffmpegPath == "__downloading__" {
		s.mu.Unlock()
		return
	}
	path, ffmpeg := s.videoPath, s.ffmpegPath
	s.inspectStatus = "inspecting"
	s.inspectError = ""
	s.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDiagnostic("inspection panic", r)
				s.mu.Lock()
				s.inspectStatus = "error"
				s.inspectError = "The video inspector stopped unexpectedly. See GBA Video Maker.log."
				s.mu.Unlock()
			}
		}()
		info, err := inspectMedia(ffmpeg, path)
		s.mu.Lock()
		defer s.mu.Unlock()
		if path != s.videoPath {
			return
		}
		if err != nil {
			s.inspectStatus = "error"
			s.inspectError = err.Error()
			return
		}
		s.mediaInfo = &info
		s.inspectStatus = "ready"
	}()
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return "video"
	}
	var b strings.Builder
	for _, r := range name {
		if r < 32 || strings.ContainsRune(`<>:"/\\|?*`, r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "video"
	}
	runes := []rune(out)
	if len(runes) > 180 {
		out = string(runes[:180])
	}
	return out
}

func saveMultipartVideo(r *http.Request, dir string) (string, string, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return "", "", fmt.Errorf("invalid upload: %w", err)
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", err
		}
		if part.FileName() == "" {
			_ = part.Close()
			continue
		}
		original := sanitizeFilename(part.FileName())
		ext := filepath.Ext(original)
		f, err := os.CreateTemp(dir, "video-*"+ext)
		if err != nil {
			_ = part.Close()
			return "", "", err
		}
		path := f.Name()
		_, copyErr := io.Copy(f, part)
		closeErr := f.Close()
		_ = part.Close()
		if copyErr != nil {
			_ = os.Remove(path)
			return "", "", copyErr
		}
		if closeErr != nil {
			_ = os.Remove(path)
			return "", "", closeErr
		}
		return path, original, nil
	}
	return "", "", errors.New("no video file was provided")
}

func (s *appState) buildOptions(req convertRequest) (ConvertOptions, error) {
	s.mu.Lock()
	path, ffmpeg := s.videoPath, s.ffmpegPath
	var info *MediaInfo
	if s.mediaInfo != nil {
		c := *s.mediaInfo
		info = &c
	}
	s.mu.Unlock()
	if info == nil || path == "" {
		return ConvertOptions{}, errors.New("video is not ready")
	}
	if ffmpeg == "" || ffmpeg == "__downloading__" {
		return ConvertOptions{}, errors.New("conversion engine is not ready")
	}
	start, err := parseTime(req.Start)
	if err != nil {
		return ConvertOptions{}, err
	}
	end := info.Duration
	if strings.TrimSpace(req.End) != "" {
		end, err = parseTime(req.End)
		if err != nil {
			return ConvertOptions{}, err
		}
	}
	if start < 0 || start >= info.Duration {
		return ConvertOptions{}, errors.New("start time is outside the video")
	}
	if end > info.Duration {
		end = info.Duration
	}
	if end <= start {
		return ConvertOptions{}, errors.New("end time must be after start time")
	}
	if req.Speed < 0.5 || req.Speed > 3 {
		return ConvertOptions{}, errors.New("speed must be between 0.50 and 3.00")
	}
	if req.Volume < 0 || req.Volume > 200 {
		return ConvertOptions{}, errors.New("volume must be between 0 and 200")
	}
	vblanksByFPS := map[string]int{"smooth": 4, "balanced": 5, "classic": 6, "compact": 8}
	vblanks, ok := vblanksByFPS[req.FPS]
	if !ok {
		return ConvertOptions{}, errors.New("invalid frame-rate preset")
	}
	if req.Fit != "crop" && req.Fit != "fit" && req.Fit != "stretch" {
		return ConvertOptions{}, errors.New("invalid screen framing")
	}
	if req.Audio != "mix" && req.Audio != "left" && req.Audio != "right" && req.Audio != "none" {
		return ConvertOptions{}, errors.New("invalid audio mode")
	}
	base := strings.TrimSuffix(sanitizeFilename(s.videoName), filepath.Ext(s.videoName))
	if base == "" {
		base = "video"
	}
	output := filepath.Join(s.sessionDir, base+"_GBA.gba")
	return ConvertOptions{InputPath: path, OutputPath: output, FFmpegPath: ffmpeg, Start: start, End: end, Speed: req.Speed, VBlanks: vblanks, FitMode: req.Fit, AudioMode: req.Audio, Volume: req.Volume / 100, Loop: req.Loop, RomTitle: normalizeTitle(req.RomTitle)}, nil
}

func normalizeTitle(base string) string {
	title := strings.ToUpper(base)
	var clean strings.Builder
	for _, r := range title {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			clean.WriteRune(r)
		} else {
			clean.WriteRune(' ')
		}
	}
	title = strings.Join(strings.Fields(clean.String()), " ")
	if len(title) > 12 {
		title = title[:12]
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "GBA VIDEO"
	}
	return title
}

func (s *appState) startConversion(req convertRequest) error {
	opt, err := s.buildOptions(req)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.converting {
		s.mu.Unlock()
		return errors.New("a conversion is already running")
	}
	s.converting = true
	s.progress = 1
	s.progressMessage = "Starting conversion…"
	s.convertError = ""
	s.result = nil
	s.downloadName = ""
	s.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDiagnostic("conversion panic", r)
				s.mu.Lock()
				s.converting = false
				s.convertError = "The converter stopped unexpectedly. See GBA Video Maker.log."
				s.mu.Unlock()
			}
		}()
		result, err := convertVideo(opt, func(p int, msg string) {
			s.mu.Lock()
			s.progress = p
			s.progressMessage = msg
			s.mu.Unlock()
		})
		s.mu.Lock()
		defer s.mu.Unlock()
		s.converting = false
		if err != nil {
			s.convertError = err.Error()
			return
		}
		s.result = &result
		s.progress = 100
		s.progressMessage = "ROM created successfully"
		s.downloadName = filepath.Base(result.OutputPath)
	}()
	return nil
}

func (s *appState) resetVideo() {
	s.mu.Lock()
	oldVideo := s.videoPath
	oldResult := ""
	if s.result != nil {
		oldResult = s.result.OutputPath
	}
	s.videoPath = ""
	s.videoName = ""
	s.mediaInfo = nil
	s.inspectStatus = "idle"
	s.inspectError = ""
	s.convertError = ""
	s.result = nil
	s.downloadName = ""
	s.progress = 0
	s.progressMessage = ""
	s.mu.Unlock()
	if oldVideo != "" {
		_ = os.Remove(oldVideo)
	}
	if oldResult != "" {
		_ = os.Remove(oldResult)
	}
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func errorJSON(w http.ResponseWriter, status int, err error) {
	jsonResponse(w, status, map[string]string{"error": err.Error()})
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				logDiagnostic("HTTP handler panic", x)
				errorJSON(w, 500, errors.New("internal error; see GBA Video Maker.log"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *appState) routes(page []byte) http.Handler {
	mux := http.NewServeMux()
	prefix := "/" + s.token
	mux.HandleFunc(prefix+"/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != prefix+"/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(page)
	})
	api := http.NewServeMux()
	api.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		s.touch()
		jsonResponse(w, 200, s.snapshot())
	})
	api.HandleFunc("/heartbeat", func(w http.ResponseWriter, r *http.Request) { s.touch(); w.WriteHeader(204) })
	api.HandleFunc("/engine/retry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		s.startEngineDownload()
		jsonResponse(w, 202, map[string]bool{"ok": true})
	})
	api.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		s.mu.Lock()
		busy := s.converting
		s.mu.Unlock()
		if busy {
			errorJSON(w, http.StatusConflict, errors.New("wait for the current conversion to finish"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		path, name, err := saveMultipartVideo(r, s.sessionDir)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		s.setUploaded(path, name)
		jsonResponse(w, 202, map[string]string{"name": name})
	})
	api.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var req convertRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			errorJSON(w, 400, fmt.Errorf("invalid settings: %w", err))
			return
		}
		if err := s.startConversion(req); err != nil {
			errorJSON(w, 400, err)
			return
		}
		jsonResponse(w, 202, map[string]bool{"ok": true})
	})
	api.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		s.resetVideo()
		w.WriteHeader(204)
	})
	api.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		var path, name string
		if s.result != nil {
			path = s.result.OutputPath
			name = s.downloadName
		}
		s.mu.Unlock()
		if path == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, path)
	})
	api.HandleFunc("/close-intent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		go func() {
			time.Sleep(5 * time.Second)
			s.mu.Lock()
			idle := time.Since(s.lastHeartbeat)
			busy := s.converting
			s.mu.Unlock()
			if idle > 4*time.Second && !busy {
				s.shutdownOnce.Do(s.shutdown)
			}
		}()
	})
	api.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204); go s.shutdownOnce.Do(s.shutdown) })
	mux.Handle(prefix+"/api/", http.StripPrefix(prefix+"/api", api))
	return recovery(mux)
}

func renderPage(token string) ([]byte, error) {
	if token == "" {
		return nil, errors.New("session token is empty")
	}
	page := strings.Replace(indexHTML, "__SESSION_TOKEN__", strconv.Quote(token), 1)
	if page == indexHTML {
		return nil, errors.New("session token placeholder is missing")
	}
	return []byte(page), nil
}

func runWebApp(launch func(string) error) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	sessionDir, err := os.MkdirTemp("", "gba-video-maker-")
	if err != nil {
		return err
	}
	page, err := renderPage(token)
	if err != nil {
		_ = os.RemoveAll(sessionDir)
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = os.RemoveAll(sessionDir)
		return err
	}
	state := newAppState(token, sessionDir)
	server := &http.Server{Handler: state.routes(page), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 5 * time.Minute}
	shutdownCh := make(chan struct{})
	state.shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		close(shutdownCh)
	}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logDiagnostic("server error", err)
			state.shutdownOnce.Do(state.shutdown)
		}
	}()
	url := "http://" + listener.Addr().String() + "/" + token + "/"
	if err := launch(url); err != nil {
		state.shutdownOnce.Do(state.shutdown)
		_ = os.RemoveAll(sessionDir)
		return err
	}

	// Exit when the page has been gone for a while, even if the browser did not
	// deliver its close beacon. This prevents invisible orphan processes.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-shutdownCh:
			_ = os.RemoveAll(sessionDir)
			return nil
		case <-ticker.C:
			state.mu.Lock()
			idle := time.Since(state.lastHeartbeat)
			busy := state.converting
			state.mu.Unlock()
			if idle > 120*time.Second && !busy {
				state.shutdownOnce.Do(state.shutdown)
			}
		}
	}
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>GBA Video Maker</title>
<style>
:root{--bg:#f5f6fb;--card:#fff;--text:#191b24;--muted:#72778a;--line:#e1e3ec;--purple:#6f4df6;--purple2:#8d6cff;--soft:#f1edff;--danger:#d53b4d;--shadow:0 24px 70px rgba(37,27,84,.12)}
*{box-sizing:border-box}html,body{height:100%;margin:0;font-family:Inter,"Segoe UI",system-ui,sans-serif;color:var(--text);background:radial-gradient(circle at 50% -20%,#ebe6ff 0,transparent 45%),var(--bg)}
button,input,select{font:inherit}.app{min-height:100%;display:grid;place-items:center;padding:28px}.hidden{display:none!important}
.drop-card{width:min(720px,88vw);height:min(430px,70vh);min-height:340px;border:1px solid #d5d7e2;border-radius:26px;background:rgba(255,255,255,.92);box-shadow:var(--shadow);display:flex;flex-direction:column;align-items:center;justify-content:center;cursor:pointer;transition:.18s ease;outline:none;user-select:none}
.drop-card:hover,.drop-card:focus-visible,.drop-card.drag{transform:translateY(-2px);border-color:var(--purple);box-shadow:0 26px 80px rgba(79,54,175,.18)}
.plus{width:68px;height:68px;border-radius:22px;background:var(--soft);display:grid;place-items:center;color:var(--purple);font-size:48px;font-weight:300;line-height:1;margin-bottom:34px}.drop-card h1{font-size:30px;letter-spacing:-.6px;margin:0 24px 13px;text-align:center}.drop-card p{margin:0 24px;color:var(--purple);font-size:18px;text-align:center}
.panel{width:min(970px,96vw);background:rgba(255,255,255,.96);border:1px solid var(--line);border-radius:26px;box-shadow:var(--shadow);padding:30px}.loading{width:min(680px,92vw);text-align:center;padding:56px}.loading h2{font-size:27px;margin:0 0 10px}.loading p{color:var(--muted);margin:0 0 28px}.bar{height:12px;background:#ebeaf1;border-radius:999px;overflow:hidden}.bar>i{display:block;height:100%;width:12%;background:linear-gradient(90deg,var(--purple),var(--purple2));border-radius:999px;transition:width .25s}.spinner .bar>i{animation:slide 1.2s infinite;width:30%}@keyframes slide{from{transform:translateX(-100%)}to{transform:translateX(430%)}}
.header{display:flex;align-items:center;gap:18px;margin-bottom:24px}.file-icon{width:52px;height:52px;border-radius:17px;background:var(--soft);display:grid;place-items:center;color:var(--purple);font-weight:800}.file-head{min-width:0;flex:1}.file-head h2{margin:0 0 5px;font-size:23px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.file-head p{margin:0;color:var(--muted)}
.btn{border:0;border-radius:13px;padding:12px 18px;font-weight:650;cursor:pointer;transition:.15s}.btn:hover{transform:translateY(-1px)}.btn.secondary{background:#f0f1f6;color:#3f4352}.btn.primary{background:linear-gradient(135deg,var(--purple),var(--purple2));color:#fff;box-shadow:0 12px 26px rgba(111,77,246,.25);padding:15px 25px}.btn:disabled{opacity:.48;cursor:not-allowed;transform:none}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:18px}.section{border:1px solid var(--line);border-radius:18px;padding:20px;background:#fff}.section h3{margin:0 0 16px;font-size:16px}.fields{display:grid;grid-template-columns:repeat(2,1fr);gap:14px}.field.full{grid-column:1/-1}.field label{display:block;font-size:12px;font-weight:700;color:var(--muted);margin:0 0 7px;text-transform:uppercase;letter-spacing:.45px}.field input,.field select{width:100%;height:42px;border:1px solid #d9dbe5;border-radius:11px;background:#fafafe;padding:0 12px;outline:none}.field input:focus,.field select:focus{border-color:var(--purple);box-shadow:0 0 0 3px rgba(111,77,246,.1)}
.check{display:flex;align-items:center;gap:10px;margin-top:14px;color:#4b4f60}.check input{accent-color:var(--purple);width:17px;height:17px}.bottom{display:flex;gap:18px;align-items:center;margin-top:22px}.estimate{flex:1;color:var(--muted);font-size:14px}.progress-wrap{margin-top:18px}.status{display:flex;justify-content:space-between;gap:15px;color:var(--muted);font-size:13px;margin-bottom:8px}.error{color:var(--danger);white-space:pre-wrap;margin-top:14px;font-size:14px}.done{display:flex;gap:12px;justify-content:flex-end;margin-top:16px}
.engine-error{max-width:620px;margin:16px auto 0;color:var(--danger);text-align:center}.tiny{font-size:12px;color:var(--muted)}
@media(max-width:760px){.app{padding:14px}.panel{padding:20px}.grid{grid-template-columns:1fr}.header{align-items:flex-start;flex-wrap:wrap}.header .btn{margin-left:auto}.bottom{flex-direction:column;align-items:stretch}.btn.primary{width:100%}}
</style>
</head>
<body>
<div class="app">
  <section id="welcome" class="drop-card" role="button" tabindex="0" aria-label="Choose a video">
    <div class="plus">+</div><h1>Drag and drop a video here</h1><p>or click to open it from File Explorer</p>
    <input id="picker" type="file" accept="video/*,.mkv,.avi,.webm,.mov,.m4v,.wmv,.flv,.mpeg,.mpg,.ts" hidden>
  </section>

  <section id="loading" class="panel loading hidden">
    <h2 id="loadingTitle">Opening video…</h2><p id="loadingText">Reading duration, picture size and audio information.</p>
    <div id="loadingBar" class="bar"><i id="loadingFill"></i></div>
    <div id="engineError" class="engine-error hidden"></div>
    <button id="retryEngine" class="btn secondary hidden" style="margin-top:18px">Retry engine download</button>
  </section>

  <main id="editor" class="panel hidden">
    <div class="header"><div class="file-icon">GBA</div><div class="file-head"><h2 id="fileName"></h2><p id="fileInfo"></p></div><button id="change" class="btn secondary">Choose another video</button></div>
    <div class="grid">
      <section class="section"><h3>Trim & playback</h3><div class="fields">
        <div class="field"><label for="start">Start</label><input id="start" value="0:00"></div>
        <div class="field"><label for="end">End</label><input id="end"></div>
        <div class="field"><label for="speed">Playback speed</label><input id="speed" type="number" min="0.5" max="3" step="0.05" value="1"></div>
        <div class="field"><label for="fps">Frame rate</label><select id="fps"><option value="smooth">Smooth — 14.93 fps</option><option value="balanced">Balanced — 11.95 fps</option><option value="classic" selected>Classic — 9.95 fps</option><option value="compact">Compact — 7.47 fps</option></select></div>
      </div><label class="check"><input id="loop" type="checkbox"> Loop when playback ends</label></section>
      <section class="section"><h3>Picture</h3><div class="fields"><div class="field full"><label for="fit">Screen framing</label><select id="fit"><option value="crop" selected>Crop to fill — no black bars</option><option value="fit">Fit — keep full image with bars</option><option value="stretch">Stretch to fill</option></select></div></div><p class="tiny">The GBA plays a 120×80 image expanded to its 240×160 screen.</p></section>
      <section class="section"><h3>Audio</h3><div class="fields"><div class="field"><label for="audio">Channel</label><select id="audio"><option value="mix">Mix channels to mono</option><option value="left">Left channel only</option><option value="right">Right channel only</option><option value="none">No audio</option></select></div><div class="field"><label for="volume">Volume %</label><input id="volume" type="number" min="0" max="200" step="5" value="100"></div></div></section>
      <section class="section"><h3>ROM</h3><div class="fields"><div class="field full"><label for="romTitle">ROM title</label><input id="romTitle" maxlength="12" value="GBA VIDEO"></div></div><p class="tiny">Controls in the ROM: A pauses or resumes; START restarts.</p></section>
    </div>
    <div class="bottom"><div id="estimate" class="estimate"></div><button id="convert" class="btn primary">Create .gba ROM</button></div>
    <div id="progressWrap" class="progress-wrap hidden"><div class="status"><span id="progressText"></span><span id="progressPct"></span></div><div class="bar"><i id="progressFill"></i></div></div>
    <div id="convertError" class="error hidden"></div>
    <div id="done" class="done hidden"><button id="download" class="btn primary">Download .gba</button></div>
  </main>
</div>
<script>
const TOKEN=__SESSION_TOKEN__;
const BASE='/'+TOKEN+'/api';
const $=id=>document.getElementById(id);
let state=null, pollTimer=null, uploadBusy=false, lastReadyVideo='';
function headers(extra={}){return Object.assign({'X-GBA-Token':TOKEN},extra)}
function show(which){['welcome','loading','editor'].forEach(id=>$(id).classList.toggle('hidden',id!==which))}
function fmtTime(sec){sec=Math.max(0,Number(sec)||0);let h=Math.floor(sec/3600),m=Math.floor((sec%3600)/60),s=(sec%60).toFixed(2).padStart(5,'0');return h?(h+':'+String(m).padStart(2,'0')+':'+s):(m+':'+s)}
function titleFromName(name){let b=name.replace(/\.[^.]+$/,'').toUpperCase().replace(/[^A-Z0-9 ]/g,' ').replace(/\s+/g,' ').trim();return (b||'GBA VIDEO').slice(0,12)}
async function api(path,opt={}){opt.headers=headers(opt.headers||{});let r=await fetch(BASE+path,opt);if(!r.ok){let x;try{x=await r.json()}catch{x={error:await r.text()}}throw new Error(x.error||('Request failed ('+r.status+')'))}return r.status===204?null:r.json()}
function choose(){if(!uploadBusy)$('picker').click()}
$('welcome').addEventListener('click',choose);$('welcome').addEventListener('keydown',e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();choose()}});
for(const ev of ['dragenter','dragover'])document.addEventListener(ev,e=>{e.preventDefault();$('welcome').classList.add('drag')});
for(const ev of ['dragleave','drop'])document.addEventListener(ev,e=>{e.preventDefault();$('welcome').classList.remove('drag')});
document.addEventListener('drop',e=>{const f=e.dataTransfer&&e.dataTransfer.files&&e.dataTransfer.files[0];if(f)upload(f)});
$('picker').addEventListener('change',()=>{const f=$('picker').files&&$('picker').files[0];if(f)upload(f);$('picker').value='' });
function upload(file){if(uploadBusy)return;uploadBusy=true;show('loading');$('loadingTitle').textContent='Loading '+file.name+'…';$('loadingText').textContent='Copying the video into the portable workspace.';$('loadingBar').classList.remove('spinner');$('loadingFill').style.width='0%';$('engineError').classList.add('hidden');$('retryEngine').classList.add('hidden');let form=new FormData();form.append('video',file,file.name);let x=new XMLHttpRequest();x.open('POST',BASE+'/upload');x.setRequestHeader('X-GBA-Token',TOKEN);x.upload.onprogress=e=>{if(e.lengthComputable)$('loadingFill').style.width=Math.round(e.loaded/e.total*100)+'%'};x.onload=()=>{uploadBusy=false;if(x.status>=200&&x.status<300){$('loadingTitle').textContent='Opening video…';$('loadingText').textContent='Reading duration, picture size and audio information.';$('loadingBar').classList.add('spinner');poll()}else{let m='Upload failed';try{m=JSON.parse(x.responseText).error||m}catch{}showErrorLoading(m)}};x.onerror=()=>{uploadBusy=false;showErrorLoading('Upload failed. Please try again.')};x.send(form)}
function showErrorLoading(msg){show('loading');$('loadingBar').classList.remove('spinner');$('loadingFill').style.width='0%';$('loadingTitle').textContent='Could not open the video';$('loadingText').textContent=msg}
async function poll(){try{state=await api('/state');renderState()}catch(e){console.error(e)}clearTimeout(pollTimer);pollTimer=setTimeout(poll,500)}
function renderState(){if(!state)return;if(state.inspectStatus==='waiting'){show('loading');$('loadingTitle').textContent='Preparing the app…';$('loadingText').textContent=state.engineMessage||'Preparing the portable conversion engine.';$('loadingBar').classList.remove('spinner');$('loadingFill').style.width=(state.engineProgress||0)+'%'}
if(state.engineStatus==='error'&&state.inspectStatus!=='ready'){show('loading');$('loadingTitle').textContent='Conversion engine unavailable';$('loadingText').textContent='The one-time engine download did not finish.';$('engineError').textContent=state.engineMessage;$('engineError').classList.remove('hidden');$('retryEngine').classList.remove('hidden');$('loadingBar').classList.remove('spinner')}
if(state.inspectStatus==='inspecting'){show('loading');$('loadingTitle').textContent='Opening '+(state.videoName||'video')+'…';$('loadingText').textContent='Reading duration, picture size and audio information.';$('loadingBar').classList.add('spinner')}
if(state.inspectStatus==='error'){show('loading');$('loadingTitle').textContent='Could not open the video';$('loadingText').textContent=state.inspectError||'Unsupported or damaged file.';$('loadingBar').classList.remove('spinner');$('retryEngine').classList.add('hidden')}
if(state.inspectStatus==='ready'&&state.media){show('editor');if(lastReadyVideo!==state.videoName){lastReadyVideo=state.videoName;$('fileName').textContent=state.videoName;$('fileInfo').textContent=state.media.Width+'×'+state.media.Height+(state.media.FPS?(' • '+state.media.FPS.toFixed(2)+' fps'):'')+' • '+fmtTime(state.media.Duration)+' • '+(state.media.AudioStreams?(state.media.AudioStreams+' audio stream(s)'):'no audio');$('start').value='0:00';$('end').value=fmtTime(state.media.Duration);$('romTitle').value=titleFromName(state.videoName);if(!state.media.AudioStreams)$('audio').value='none';estimate()}}
$('convert').disabled=!!state.converting;for(const id of ['change','start','end','speed','fps','fit','audio','volume','romTitle','loop'])$(id).disabled=!!state.converting;
if(state.converting){$('progressWrap').classList.remove('hidden');$('done').classList.add('hidden');$('convertError').classList.add('hidden');$('progressText').textContent=state.progressMessage||'Converting…';$('progressPct').textContent=(state.progress||0)+'%';$('progressFill').style.width=(state.progress||0)+'%'}
else if(state.result){$('progressWrap').classList.remove('hidden');$('progressText').textContent='ROM created successfully';$('progressPct').textContent='100%';$('progressFill').style.width='100%';$('done').classList.remove('hidden');$('convertError').classList.add('hidden')}
else if(state.convertError){$('progressWrap').classList.add('hidden');$('done').classList.add('hidden');$('convertError').textContent=state.convertError;$('convertError').classList.remove('hidden')}
}
$('retryEngine').onclick=()=>api('/engine/retry',{method:'POST'}).catch(e=>showErrorLoading(e.message));
$('change').onclick=async()=>{await api('/reset',{method:'POST'});lastReadyVideo='';show('welcome')};
function values(){return{start:$('start').value,end:$('end').value,speed:Number($('speed').value),fps:$('fps').value,fit:$('fit').value,audio:$('audio').value,volume:Number($('volume').value),loop:$('loop').checked,romTitle:$('romTitle').value}}
function estimate(){if(!state||!state.media)return;let v=values(),start=parseClock(v.start),end=parseClock(v.end);if(!Number.isFinite(start)||!Number.isFinite(end)||end<=start||v.speed<.5||v.speed>3){$('estimate').textContent='Check the trim and speed settings.';return}end=Math.min(end,state.media.Duration);let vb={smooth:4,balanced:5,classic:6,compact:8}[v.fps],fps=59.727500569606/vb,frames=Math.max(1,Math.ceil(((end-start)/v.speed)*fps)),display=frames*vb/59.727500569606,audio=(v.audio!=='none'&&state.media.AudioStreams)?Math.ceil(display*16384/16)*16:0,raw=8192+512+frames*9600+audio,p=1048576;while(p<raw)p*=2;$('estimate').textContent=raw>33554432?('Too large: about '+(raw/1048576).toFixed(2)+' MiB. Shorten it, speed it up, lower FPS or disable audio.'):('Estimated output: '+(p/1048576)+' MiB cartridge • '+frames+' frames • '+fps.toFixed(2)+' fps')}
function parseClock(s){let a=String(s).trim().split(':').map(Number);if(a.some(x=>!Number.isFinite(x))||a.length<1||a.length>3)return NaN;if(a.length===1)return a[0];if(a.length===2)return a[0]*60+a[1];return a[0]*3600+a[1]*60+a[2]}
for(const id of ['start','end','speed','fps','fit','audio','volume'])$(id).addEventListener('input',estimate);
$('convert').onclick=async()=>{try{await api('/convert',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(values())});poll()}catch(e){$('convertError').textContent=e.message;$('convertError').classList.remove('hidden')}};
$('download').onclick=()=>{const a=document.createElement('a');a.href=BASE+'/download';a.download=state.downloadName||'video_GBA.gba';a.style.display='none';document.body.appendChild(a);a.click();a.remove()};
setInterval(()=>fetch(BASE+'/heartbeat',{method:'POST',headers:headers(),keepalive:true}).catch(()=>{}),5000);
window.addEventListener('pagehide',()=>fetch(BASE+'/close-intent',{method:'POST',headers:headers(),keepalive:true}).catch(()=>{}));
poll();
</script>
</body>
</html>`
