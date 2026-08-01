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

type uploadedVideo struct {
	Path   string
	Name   string
	Info   *MediaInfo
	Status string
	Error  string
}

type publicVideo struct {
	Name   string     `json:"name"`
	Info   *MediaInfo `json:"info,omitempty"`
	Status string     `json:"status"`
	Error  string     `json:"error"`
}

type appState struct {
	mu sync.Mutex

	token      string
	sessionDir string
	ffmpegPath string

	engineStatus   string
	engineProgress int
	engineMessage  string

	videos        []uploadedVideo
	inspectStatus string
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
	VideoName       string         `json:"videoName"`
	Media           *MediaInfo     `json:"media,omitempty"`
	EngineStatus    string         `json:"engineStatus"`
	EngineProgress  int            `json:"engineProgress"`
	EngineMessage   string         `json:"engineMessage"`
	Videos          []publicVideo  `json:"videos"`
	InspectStatus   string         `json:"inspectStatus"`
	InspectError    string         `json:"inspectError"`
	Converting      bool           `json:"converting"`
	Progress        int            `json:"progress"`
	ProgressMessage string         `json:"progressMessage"`
	ConvertError    string         `json:"convertError"`
	Result          *ConvertResult `json:"result,omitempty"`
	DownloadName    string         `json:"downloadName"`
}

type removeVideoRequest struct {
	Index int `json:"index"`
}

type convertRequest struct {
	Start       string  `json:"start"`
	End         string  `json:"end"`
	Speed       float64 `json:"speed"`
	FPS         string  `json:"fps"`
	Fit         string  `json:"fit"`
	Audio       string  `json:"audio"`
	Volume      float64 `json:"volume"`
	Loop        bool    `json:"loop"`
	RomTitle    string  `json:"romTitle"`
	SeekSeconds int     `json:"seekSeconds"`
	Normalize   bool    `json:"normalize"`
	Limiter     bool    `json:"limiter"`
	Resume      bool    `json:"resume"`
	Compression string  `json:"compression"`
	PaletteMode string  `json:"paletteMode"`
	DitherMode  string  `json:"ditherMode"`
	OutputMode  string  `json:"outputMode"`
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
	for _, p := range []string{filepath.Join(appDirectory(), "ffmpeg.exe"), filepath.Join(appDirectory(), "tools", "ffmpeg.exe")} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Size() > 1_000_000 {
			return p
		}
	}
	return ""
}

func newAppState(token, sessionDir string) *appState {
	s := &appState{token: token, sessionDir: sessionDir, lastHeartbeat: time.Now(), inspectStatus: "idle"}
	if ff := locatePortableFFmpeg(); ff != "" {
		s.ffmpegPath, s.engineStatus, s.engineProgress, s.engineMessage = ff, "ready", 100, "Conversion engine ready"
	} else {
		s.engineStatus = "idle"
		s.engineMessage = "The portable conversion engine will be prepared when videos are selected."
	}
	return s
}

func (s *appState) snapshot() publicState {
	s.mu.Lock()
	defer s.mu.Unlock()
	videos := make([]publicVideo, len(s.videos))
	for i, v := range s.videos {
		var info *MediaInfo
		if v.Info != nil {
			c := *v.Info
			info = &c
		}
		videos[i] = publicVideo{Name: v.Name, Info: info, Status: v.Status, Error: v.Error}
	}
	var result *ConvertResult
	if s.result != nil {
		c := *s.result
		result = &c
	}
	var firstName string
	var firstInfo *MediaInfo
	if len(videos) > 0 {
		firstName = videos[0].Name
		firstInfo = videos[0].Info
	}
	return publicState{
		VideoName: firstName, Media: firstInfo,
		EngineStatus: s.engineStatus, EngineProgress: s.engineProgress, EngineMessage: s.engineMessage,
		Videos: videos, InspectStatus: s.inspectStatus, InspectError: s.inspectError,
		Converting: s.converting, Progress: s.progress, ProgressMessage: s.progressMessage,
		ConvertError: s.convertError, Result: result, DownloadName: s.downloadName,
	}
}

func (s *appState) touch() { s.mu.Lock(); s.lastHeartbeat = time.Now(); s.mu.Unlock() }

func downloadFileWithProgress(url, path string, progress func(done, total int64)) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "GBA-Video-Maker/0.8.0")
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
	defer f.Close()
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
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expected) {
		return errors.New("safety check failed (SHA-256 mismatch)")
	}
	return nil
}

func (s *appState) startEngineDownload() {
	s.mu.Lock()
	if s.engineStatus == "ready" || s.engineStatus == "downloading" {
		s.mu.Unlock()
		return
	}
	s.engineStatus, s.engineProgress, s.engineMessage, s.ffmpegPath = "downloading", 0, "Preparing the portable conversion engine…", "__downloading__"
	s.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDiagnostic("engine download panic", r)
				s.mu.Lock()
				s.engineStatus, s.engineMessage, s.ffmpegPath = "error", "Could not prepare FFmpeg. Click Retry.", ""
				s.mu.Unlock()
			}
		}()
		target, temp := filepath.Join(appDirectory(), "ffmpeg.exe"), filepath.Join(appDirectory(), "ffmpeg.exe.download")
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
			s.engineProgress, s.engineMessage = p, fmt.Sprintf("Preparing conversion engine… %d%%", p)
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
			s.engineStatus, s.engineMessage, s.ffmpegPath = "error", "Engine download failed: "+err.Error(), ""
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		s.ffmpegPath, s.engineStatus, s.engineProgress, s.engineMessage = target, "ready", 100, "Conversion engine ready"
		pending := len(s.videos) > 0
		s.mu.Unlock()
		if pending {
			s.startInspection()
		}
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

func saveMultipartVideos(r *http.Request, dir string) ([]uploadedVideo, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("invalid upload: %w", err)
	}
	var videos []uploadedVideo
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.FileName() == "" {
			part.Close()
			continue
		}
		name := sanitizeFilename(part.FileName())
		f, err := os.CreateTemp(dir, "video-*"+filepath.Ext(name))
		if err != nil {
			part.Close()
			return nil, err
		}
		path := f.Name()
		_, copyErr := io.Copy(f, part)
		closeErr := f.Close()
		part.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(path)
			if copyErr != nil {
				return nil, copyErr
			}
			return nil, closeErr
		}
		videos = append(videos, uploadedVideo{Path: path, Name: name, Status: "waiting"})
	}
	if len(videos) == 0 {
		return nil, errors.New("no video files were provided")
	}
	return videos, nil
}

func (s *appState) removeVideo(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.converting {
		return errors.New("wait for the current conversion to finish")
	}
	if s.inspectStatus != "ready" {
		return errors.New("wait until the videos finish loading")
	}
	if index < 0 || index >= len(s.videos) {
		return errors.New("video index out of range")
	}
	victim := s.videos[index]
	s.videos = append(s.videos[:index], s.videos[index+1:]...)
	s.result, s.convertError, s.downloadName = nil, "", ""
	if len(s.videos) == 0 {
		s.inspectStatus = "waiting"
	}
	_ = os.Remove(victim.Path)
	return nil
}

func (s *appState) addUploaded(videos []uploadedVideo, appendMode bool) {
	s.mu.Lock()
	old := s.videos
	if appendMode {
		s.videos = append(s.videos, videos...)
	} else {
		s.videos = videos
	}
	s.result, s.convertError, s.downloadName = nil, "", ""
	if s.engineStatus == "ready" {
		s.inspectStatus = "inspecting"
	} else {
		s.inspectStatus = "waiting"
	}
	s.mu.Unlock()
	if !appendMode {
		for _, v := range old {
			_ = os.Remove(v.Path)
		}
	}
	if s.engineStatus == "ready" {
		s.startInspection()
	} else {
		s.startEngineDownload()
	}
}

func (s *appState) startInspection() {
	s.mu.Lock()
	if s.engineStatus != "ready" || s.ffmpegPath == "" || len(s.videos) == 0 {
		s.mu.Unlock()
		return
	}
	ff := s.ffmpegPath
	for i := range s.videos {
		if s.videos[i].Info == nil {
			s.videos[i].Status = "inspecting"
			s.videos[i].Error = ""
		}
	}
	s.inspectStatus, s.inspectError = "inspecting", ""
	s.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDiagnostic("inspection panic", r)
				s.mu.Lock()
				s.inspectStatus, s.inspectError = "error", "Video inspection stopped unexpectedly."
				s.mu.Unlock()
			}
		}()
		s.mu.Lock()
		count := len(s.videos)
		s.mu.Unlock()
		for i := 0; i < count; i++ {
			s.mu.Lock()
			if i >= len(s.videos) {
				s.mu.Unlock()
				return
			}
			path := s.videos[i].Path
			ready := s.videos[i].Info != nil
			s.mu.Unlock()
			if ready {
				continue
			}
			info, err := inspectMedia(ff, path)
			s.mu.Lock()
			if i < len(s.videos) && s.videos[i].Path == path {
				if err != nil {
					s.videos[i].Status, s.videos[i].Error = "error", err.Error()
				} else {
					s.videos[i].Info, s.videos[i].Status = &info, "ready"
				}
			}
			s.mu.Unlock()
		}
		s.mu.Lock()
		allReady := len(s.videos) > 0
		for _, v := range s.videos {
			if v.Status != "ready" {
				allReady = false
				if v.Error != "" {
					s.inspectError = v.Name + ": " + v.Error
				}
			}
		}
		if allReady {
			s.inspectStatus = "ready"
		} else {
			s.inspectStatus = "error"
		}
		s.mu.Unlock()
	}()
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

func (s *appState) buildOptions(req convertRequest) (ProjectOptions, []MediaInfo, error) {
	s.mu.Lock()
	ff := s.ffmpegPath
	videos := append([]uploadedVideo(nil), s.videos...)
	s.mu.Unlock()
	if len(videos) == 0 {
		return ProjectOptions{}, nil, errors.New("videos are not ready")
	}
	if ff == "" || ff == "__downloading__" {
		return ProjectOptions{}, nil, errors.New("conversion engine is not ready")
	}
	start, err := parseTime(req.Start)
	if err != nil {
		return ProjectOptions{}, nil, err
	}
	end := 0.0
	if strings.TrimSpace(req.End) != "" {
		end, err = parseTime(req.End)
		if err != nil {
			return ProjectOptions{}, nil, err
		}
	}
	if req.Speed < .5 || req.Speed > 3 {
		return ProjectOptions{}, nil, errors.New("speed must be between 0.50 and 3.00")
	}
	if req.Volume < 0 || req.Volume > 200 {
		return ProjectOptions{}, nil, errors.New("volume must be between 0 and 200")
	}
	vblanks, ok := map[string]int{"smooth": 4, "balanced": 5, "classic": 6, "compact": 8}[req.FPS]
	if !ok {
		return ProjectOptions{}, nil, errors.New("invalid frame-rate preset")
	}
	if req.Fit != "crop" && req.Fit != "fit" && req.Fit != "stretch" {
		return ProjectOptions{}, nil, errors.New("invalid screen framing")
	}
	if req.Audio != "mix" && req.Audio != "left" && req.Audio != "right" && req.Audio != "none" {
		return ProjectOptions{}, nil, errors.New("invalid audio mode")
	}
	var inputs []ClipInput
	var infos []MediaInfo
	for _, v := range videos {
		if v.Info == nil || v.Status != "ready" {
			return ProjectOptions{}, nil, fmt.Errorf("%s is not ready", v.Name)
		}
		if start >= v.Info.Duration {
			return ProjectOptions{}, nil, fmt.Errorf("start time is outside %s", v.Name)
		}
		inputs = append(inputs, ClipInput{InputPath: v.Path, Name: v.Name, Title: normalizeTitle(strings.TrimSuffix(v.Name, filepath.Ext(v.Name)))})
		infos = append(infos, *v.Info)
	}
	if req.SeekSeconds == 0 {
		req.SeekSeconds = 5
	}
	if req.Compression == "" {
		req.Compression = "delta"
	}
	if req.PaletteMode == "" {
		req.PaletteMode = "shared"
	}
	if req.DitherMode == "" {
		req.DitherMode = "ordered"
	}
	mode := req.OutputMode
	if len(inputs) == 1 {
		mode = "rom"
	}
	if mode != "rom" && mode != "playlist" && mode != "menu" && mode != "batch" {
		mode = "rom"
	}
	base := strings.TrimSuffix(sanitizeFilename(videos[0].Name), filepath.Ext(videos[0].Name))
	if len(inputs) > 1 {
		base = "GBA_Video_Collection"
	}
	ext := ".gba"
	if mode == "batch" {
		ext = ".zip"
	}
	output := filepath.Join(s.sessionDir, base+"_v0.8.0"+ext)
	romTitle := normalizeTitle(req.RomTitle)
	if strings.TrimSpace(req.RomTitle) == "" {
		romTitle = normalizeTitle(base)
	}
	opt := ProjectOptions{Inputs: inputs, OutputPath: output, FFmpegPath: ff, Start: start, End: end, Speed: req.Speed, VBlanks: vblanks, FitMode: req.Fit, AudioMode: req.Audio, Volume: req.Volume / 100, Loop: req.Loop, RomTitle: romTitle, SeekSeconds: req.SeekSeconds, Normalize: req.Normalize, Limiter: req.Limiter, Resume: req.Resume, Compression: req.Compression, PaletteMode: req.PaletteMode, DitherMode: req.DitherMode, OutputMode: mode, KeyInterval: 30}
	return opt, infos, validateProject(opt)
}

func (s *appState) startConversion(req convertRequest) error {
	opt, _, err := s.buildOptions(req)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.converting {
		s.mu.Unlock()
		return errors.New("a conversion is already running")
	}
	s.converting, s.progress, s.progressMessage, s.convertError, s.result, s.downloadName = true, 1, "Starting conversion…", "", nil, ""
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
		result, err := convertProject(opt, func(p int, msg string) { s.mu.Lock(); s.progress, s.progressMessage = p, msg; s.mu.Unlock() })
		s.mu.Lock()
		defer s.mu.Unlock()
		s.converting = false
		if err != nil {
			s.convertError = err.Error()
			return
		}
		s.result = &result
		s.progress = 100
		s.progressMessage = "Output created successfully"
		s.downloadName = filepath.Base(result.OutputPath)
	}()
	return nil
}

func (s *appState) resetVideos() {
	s.mu.Lock()
	old := s.videos
	result := ""
	if s.result != nil {
		result = s.result.OutputPath
	}
	s.videos = nil
	s.inspectStatus = "idle"
	s.inspectError = ""
	s.convertError = ""
	s.result = nil
	s.downloadName = ""
	s.progress = 0
	s.progressMessage = ""
	s.mu.Unlock()
	for _, v := range old {
		_ = os.Remove(v.Path)
	}
	if result != "" {
		_ = os.Remove(result)
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
	mux.HandleFunc(prefix+"/icon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(appIconPNG)
	})
	api := http.NewServeMux()
	api.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) { s.touch(); jsonResponse(w, 200, s.snapshot()) })
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
			errorJSON(w, 409, errors.New("wait for the current conversion to finish"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		videos, err := saveMultipartVideos(r, s.sessionDir)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		appendMode := r.URL.Query().Get("append") == "1"
		s.addUploaded(videos, appendMode)
		jsonResponse(w, 202, map[string]int{"count": len(videos)})
	})
	api.HandleFunc("/video/remove", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var req removeVideoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorJSON(w, 400, errors.New("invalid remove request"))
			return
		}
		if err := s.removeVideo(req.Index); err != nil {
			errorJSON(w, 409, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"ok": true})
	})
	api.HandleFunc("/preview", func(w http.ResponseWriter, r *http.Request) {
		idx, err := strconv.Atoi(r.URL.Query().Get("index"))
		if err != nil {
			idx = 0
		}
		timeSec, _ := strconv.ParseFloat(r.URL.Query().Get("time"), 64)
		fit := r.URL.Query().Get("fit")
		if fit != "crop" && fit != "stretch" {
			fit = "fit"
		}
		s.mu.Lock()
		if idx < 0 || idx >= len(s.videos) || s.videos[idx].Info == nil {
			s.mu.Unlock()
			http.NotFound(w, r)
			return
		}
		video := s.videos[idx]
		ff := s.ffmpegPath
		s.mu.Unlock()
		if timeSec < 0 {
			timeSec = 0
		}
		lastFrameTime := video.Info.Duration
		if video.Info.FPS > 0 {
			lastFrameTime -= 1.0 / video.Info.FPS
		} else {
			lastFrameTime -= 0.04
		}
		if lastFrameTime < 0 {
			lastFrameTime = 0
		}
		if timeSec >= video.Info.Duration {
			timeSec = lastFrameTime
		}
		frameStep := 0.04
		if video.Info.FPS > 0 {
			frameStep = 1.0 / video.Info.FPS
		}
		attempts := []float64{timeSec}
		if timeSec > 0 {
			attempts = append(attempts, timeSec-frameStep, timeSec-frameStep*2, timeSec-0.25, 0)
		}
		var out string
		var previewErr error
		for _, candidate := range attempts {
			if candidate < 0 {
				candidate = 0
			}
			out = filepath.Join(s.sessionDir, fmt.Sprintf("preview-%d-%d-%s.png", idx, int(candidate*1000), fit))
			if st, err := os.Stat(out); err == nil && st.Size() > 0 {
				previewErr = nil
				break
			}
			_ = os.Remove(out)
			previewErr = generatePreview(ff, video.Path, candidate, fit, out)
			if previewErr == nil {
				if st, err := os.Stat(out); err == nil && st.Size() > 0 {
					break
				}
				previewErr = errors.New("FFmpeg returned no preview frame")
			}
		}
		if previewErr != nil {
			errorJSON(w, 500, previewErr)
			return
		}
		data, err := os.ReadFile(out)
		if err != nil {
			errorJSON(w, 500, err)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})
	api.HandleFunc("/audio-preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		idx, _ := strconv.Atoi(r.URL.Query().Get("index"))
		var req convertRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		if err := dec.Decode(&req); err != nil {
			errorJSON(w, 400, err)
			return
		}
		opt, infos, err := s.buildOptions(req)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		if idx < 0 || idx >= len(opt.Inputs) {
			idx = 0
		}
		out := filepath.Join(s.sessionDir, fmt.Sprintf("audio-preview-%d.wav", time.Now().UnixNano()))
		defer os.Remove(out)
		if err := generateAudioPreview(opt, infos[idx], opt.Inputs[idx].InputPath, out); err != nil {
			errorJSON(w, 400, err)
			return
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, out)
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
		s.resetVideos()
		w.WriteHeader(204)
	})
	api.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		path, name := "", ""
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
		w.WriteHeader(204)
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
		os.RemoveAll(sessionDir)
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.RemoveAll(sessionDir)
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
		os.RemoveAll(sessionDir)
		return err
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-shutdownCh:
			os.RemoveAll(sessionDir)
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
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="theme-color" media="(prefers-color-scheme: dark)" content="#0c0f14"><meta name="theme-color" media="(prefers-color-scheme: light)" content="#f4f7fb"><link rel="icon" type="image/png" href="./icon.png"><title>GBA Video Maker 0.8.0</title>
<style>
:root{color-scheme:light dark;--bg:#f4f7fb;--body-top:#ffffff;--panel:#ffffff;--panel2:#eef3f8;--line:#c9d4e1;--text:#17202b;--muted:#5c6b7a;--accent:#d8bd00;--danger:#c33c4d;--green:#24874a;--button:#e8eef5;--button-line:#b8c5d4;--input:#ffffff;--label:#334252;--bar-bg:#dbe3ec;--active:#dbe7f5;--pill:#dce6f1;--pill-text:#34485c;--drag:#fff9cf;--shadow:#23364a26;--error-bg:#fff0f1;--error-line:#e1a7ae;--error-text:#7b1f2b}
@media(prefers-color-scheme:dark){:root{--bg:#0c0f14;--body-top:#1a2230;--panel:#151a22;--panel2:#1b222d;--line:#2a3442;--text:#eef3f8;--muted:#9ba8b7;--accent:#f5d90a;--danger:#ff6c6c;--green:#62d38a;--button:#222b37;--button-line:#3b4758;--input:#10151c;--label:#c5cfdb;--bar-bg:#252e3b;--active:#283344;--pill:#263142;--pill-text:#c9d5e2;--drag:#1c2230;--shadow:#0008;--error-bg:#401f25;--error-line:#73333d;--error-text:#ffd1d1}}
*{box-sizing:border-box}html,body{min-height:100%;height:100%}body{margin:0;background:radial-gradient(circle at top,var(--body-top),var(--bg) 55%);color:var(--text);font:14px/1.35 system-ui,Segoe UI,sans-serif}.hidden{display:none!important}.shell{width:100%;min-height:100vh;padding:22px;display:flex;flex-direction:column}.top{width:min(1180px,100%);margin:0 auto 16px;display:flex;align-items:center;justify-content:flex-end}.card{background:color-mix(in srgb,var(--panel) 96%,transparent);border:1px solid var(--line);border-radius:15px;box-shadow:0 18px 60px var(--shadow)}.welcome{width:min(1180px,100%);flex:1;min-height:260px;margin:auto;display:grid;place-items:center;text-align:center;padding:40px;cursor:pointer;border:2px dashed var(--button-line)}.welcome.drag{border-color:var(--accent);background:var(--drag)}.welcome h1{font-size:30px;margin:8px}.welcome p{color:var(--muted)}.btn{border:1px solid var(--button-line);background:var(--button);color:var(--text);border-radius:9px;padding:10px 15px;font-weight:700;cursor:pointer}.btn:hover{filter:brightness(1.08)}.btn.primary{background:var(--accent);color:#111;border-color:var(--accent)}.btn:disabled{opacity:.5;cursor:not-allowed}.loading{width:min(1180px,100%);margin:auto;padding:50px;text-align:center}.bar{height:10px;border-radius:9px;background:var(--bar-bg);overflow:hidden}.bar i{display:block;height:100%;width:0;background:var(--accent);transition:.2s}.editor{width:min(1180px,100%);margin:0 auto;display:grid;grid-template-columns:390px 1fr;gap:16px}.left,.right{padding:16px}.preview{aspect-ratio:3/2;background:#000;border:1px solid var(--line);border-radius:10px;display:grid;place-items:center;overflow:hidden}.preview img{width:100%;height:100%;image-rendering:pixelated;object-fit:contain}.preview img:not([src]){display:none}.preview-controls{display:flex;gap:8px;margin-top:10px}.clips{margin-top:14px;border:1px solid var(--line);border-radius:10px;max-height:190px;overflow:auto}.clip{display:flex;align-items:center;justify-content:space-between;gap:10px;padding:9px 11px;border-bottom:1px solid var(--line);cursor:pointer}.clip:last-child{border:0}.clip.active{background:var(--active)}.clip small{display:block;color:var(--muted)}.clip-info{min-width:0;flex:1}.clip-info b,.clip-info small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.clip-remove{flex:0 0 auto;width:24px;height:24px;border-radius:7px;border:1px solid color-mix(in srgb,#ff5a5a 70%,var(--button-line));background:color-mix(in srgb,#ff4b4b 18%,transparent);color:#ff6b6b;font-weight:900;cursor:pointer;line-height:1;padding:0}.clip-remove:hover{background:color-mix(in srgb,#ff4b4b 30%,transparent);color:#fff}.section{border:1px solid var(--line);background:var(--panel2);border-radius:11px;padding:13px;margin-bottom:11px}.section h3{font-size:14px;margin:0 0 10px;color:var(--accent)}.fields{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px}.field{display:flex;flex-direction:column;gap:5px}.field.full{grid-column:1/-1}.field.two{grid-column:span 2}label{color:var(--label);font-size:12px;font-weight:650}input,select{width:100%;background:var(--input);color:var(--text);border:1px solid var(--button-line);border-radius:7px;padding:8px}input[type=checkbox]{width:auto;accent-color:var(--accent)}.check{display:flex;align-items:center;gap:8px;padding-top:22px}.tiny{font-size:12px;color:var(--muted);margin:8px 0 0}.bottom{display:flex;align-items:center;gap:12px;justify-content:space-between}.estimate{color:var(--text);line-height:1.5}.progress-wrap{margin-top:14px}.status{display:flex;justify-content:space-between;margin-bottom:6px}.error{color:var(--error-text);background:var(--error-bg);border:1px solid var(--error-line);padding:10px;border-radius:8px;margin-top:10px}.done{margin-top:12px}.audio-row{display:flex;gap:8px;align-items:center}.audio-row audio{height:34px;flex:1}.pill{display:inline-block;padding:3px 7px;border-radius:999px;background:var(--pill);color:var(--pill-text);font-size:11px}@media(max-width:900px){.editor{grid-template-columns:1fr}.fields{grid-template-columns:1fr 1fr}}
</style></head><body><div class="shell"><div id="topBar" class="top hidden"><button id="resetTop" class="btn">Start over</button></div>
<input id="picker" type="file" accept="video/*,.mkv,.webm,.avi,.mov,.mp4" multiple class="hidden">
<section id="welcome" class="card welcome" tabindex="0"><div><h1>Drag and drop videos here</h1><p>Choose one video, several videos for one combined ROM, or a batch ZIP.</p><button class="btn primary">Choose videos</button></div></section>
<section id="loading" class="card loading hidden"><h2 id="loadingTitle">Opening videos…</h2><p id="loadingText">Reading video information.</p><div class="bar"><i id="loadingFill"></i></div><div id="engineError" class="error hidden"></div><button id="retryEngine" class="btn hidden">Retry engine download</button></section>
<section id="editor" class="editor hidden"><div class="card left"><div class="preview"><img id="previewImage" alt=""></div><div class="preview-controls"><button id="previewStart" class="btn">Start frame</button><button id="previewEnd" class="btn">End frame</button><button id="addVideos" class="btn">Add videos</button></div><div id="clips" class="clips"></div><p id="clipInfo" class="tiny"></p></div>
<div class="card right"><section class="section"><h3>Quality preset</h3><div class="fields"><div class="field full"><select id="preset"><option value="best">Best quality</option><option value="balanced" selected>Balanced</option><option value="long">Long video</option><option value="small">Smallest ROM</option><option value="custom">Custom</option></select></div></div></section>
<section class="section"><h3>Video</h3><div class="fields"><div class="field"><label>Start</label><input id="start" value="0:00"></div><div class="field"><label>End (blank = full video)</label><input id="end"></div><div class="field"><label>Speed</label><input id="speed" type="number" min="0.5" max="3" step="0.05" value="1"></div><div class="field"><label>Frame rate</label><select id="fps"><option value="smooth">Smooth — 14.93 fps</option><option value="balanced" selected>Balanced — 11.95 fps</option><option value="classic">Classic — 9.95 fps</option><option value="compact">Compact — 7.47 fps</option></select></div><div class="field"><label>Screen framing</label><select id="fit"><option value="fit">Fit with bars</option><option value="crop" selected>Crop to fill</option><option value="stretch">Stretch</option></select></div><div class="field"><label>Seek step</label><select id="seekSeconds"><option>3</option><option selected>5</option><option>10</option><option>15</option></select></div></div></section>
<section class="section"><h3>Colour and compression</h3><div class="fields"><div class="field"><label>Palette</label><select id="paletteMode"><option value="shared" selected>Shared palette</option><option value="scene">Per-scene palette</option></select></div><div class="field"><label>Dithering</label><select id="ditherMode"><option value="off">Off</option><option value="ordered" selected>Ordered</option><option value="error">Error diffusion</option></select></div><div class="field"><label>Video compression</label><select id="compression"><option value="delta" selected>Delta + keyframes</option><option value="none">Uncompressed</option></select></div></div></section>
<section class="section"><h3>Audio</h3><div class="fields"><div class="field"><label>Channel</label><select id="audio"><option value="mix" selected>Mix to mono</option><option value="left">Left channel</option><option value="right">Right channel</option><option value="none">No audio</option></select></div><div class="field"><label>Volume %</label><input id="volume" type="number" min="0" max="200" step="5" value="100"></div><label class="check"><input id="normalize" type="checkbox"> Normalize quiet audio</label><label class="check"><input id="limiter" type="checkbox" checked> Limiter</label><div class="field two"><label>Preview selected channel</label><div class="audio-row"><button id="audioPreview" class="btn">Create preview</button><audio id="audioPlayer" controls></audio></div></div></div></section>
<section class="section"><h3>ROM and playback</h3><div class="fields"><div class="field two"><label>ROM title</label><input id="romTitle" maxlength="12" value=""></div><div class="field"><label>Output</label><select id="outputMode"><option value="playlist">One ROM — play clips in order</option><option value="menu">One ROM — clip menu</option><option value="batch">Separate ROMs in ZIP</option></select></div><label class="check"><input id="loop" type="checkbox"> Loop playback</label><label class="check"><input id="resume" type="checkbox" checked> Save/resume position</label></div><p class="tiny">Controls: A pause; B restart (or return to the clip menu in menu ROMs); L/R seek and hold; Left/Right seek while playing or step frames while paused; Up/Down volume; SELECT mute; START cycles HUD; L+R toggles HUD; START+SELECT opens control help.</p></section>
<div class="bottom"><div id="estimate" class="estimate"></div><button id="convert" class="btn primary">Create output</button></div><div id="progressWrap" class="progress-wrap hidden"><div class="status"><span id="progressText"></span><span id="progressPct"></span></div><div class="bar"><i id="progressFill"></i></div></div><div id="convertError" class="error hidden"></div><div id="done" class="done hidden"><button id="download" class="btn primary">Download output</button></div></div></section></div>
<script>
const TOKEN=__SESSION_TOKEN__,BASE='/'+TOKEN+'/api',$=id=>document.getElementById(id);let state=null,pollTimer=null,uploadBusy=false,selected=0,previewMode='start',audioURL='',lastPreviewKey='',romTitleAuto=true;
function headers(x={}){return Object.assign({'X-GBA-Token':TOKEN},x)}function show(id){['welcome','loading','editor'].forEach(x=>$(x).classList.toggle('hidden',x!==id));$('topBar').classList.toggle('hidden',id!=='editor')}
async function api(path,opt={}){opt.headers=headers(opt.headers||{});let r=await fetch(BASE+path,opt);if(!r.ok){let x;try{x=await r.json()}catch{x={error:await r.text()}}throw Error(x.error||'Request failed')};return r.status===204?null:r.json()}
function fmt(sec){sec=Math.max(0,+sec||0);let m=Math.floor(sec/60),s=Math.floor(sec%60);return m+':'+String(s).padStart(2,'0')}function parseClock(s){if(String(s).trim()==='')return 0;let a=String(s).split(':').map(Number);if(a.some(x=>!isFinite(x)))return NaN;return a.length===3?a[0]*3600+a[1]*60+a[2]:a.length===2?a[0]*60+a[1]:a[0]}
function titleName(n){return(n.replace(/\.[^.]+$/,'').toUpperCase().replace(/[^A-Z0-9 ]/g,' ').replace(/\s+/g,' ').trim()||'GBA VIDEO').slice(0,12)}
function choose(append=false){$('picker').dataset.append=append?'1':'0';$('picker').click()}$('welcome').onclick=()=>choose(false);$('welcome').onkeydown=e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();choose(false)}};$('addVideos').onclick=()=>choose(true);$('picker').onchange=()=>{let f=[...$('picker').files];if(f.length)upload(f,$('picker').dataset.append==='1');$('picker').value=''};
for(const ev of ['dragenter','dragover'])document.addEventListener(ev,e=>{e.preventDefault();$('welcome').classList.add('drag')});for(const ev of ['dragleave','drop'])document.addEventListener(ev,e=>{e.preventDefault();$('welcome').classList.remove('drag')});document.addEventListener('drop',e=>{let f=[...(e.dataTransfer?.files||[])];if(f.length)upload(f,state&&state.videos?.length)});
function upload(files,append){if(uploadBusy)return;if(!append)romTitleAuto=true;uploadBusy=true;show('loading');$('loadingTitle').textContent='Loading '+files.length+' video'+(files.length===1?'':'s')+'…';$('loadingText').textContent='Copying files into the portable workspace.';let form=new FormData();files.forEach(f=>form.append('video',f,f.name));let x=new XMLHttpRequest();x.open('POST',BASE+'/upload?append='+(append?1:0));x.setRequestHeader('X-GBA-Token',TOKEN);x.upload.onprogress=e=>{if(e.lengthComputable)$('loadingFill').style.width=Math.round(e.loaded/e.total*100)+'%'};x.onload=()=>{uploadBusy=false;if(x.status<300){lastPreviewKey='';$('previewImage').removeAttribute('src');poll()}else{showLoadError('Upload failed')}};x.onerror=()=>{uploadBusy=false;showLoadError('Upload failed')};x.send(form)}
function showLoadError(m){show('loading');$('loadingTitle').textContent='Could not open videos';$('loadingText').textContent=m}
async function poll(){try{state=await api('/state');render()}catch(e){console.error(e)}clearTimeout(pollTimer);pollTimer=setTimeout(poll,500)}
function render(){if(!state)return;if(!state.videos||!state.videos.length){show('welcome');$('clipInfo').textContent='';return}if(state.inspectStatus==='waiting'||state.inspectStatus==='inspecting'){show('loading');$('loadingTitle').textContent=state.inspectStatus==='waiting'?'Preparing the app…':'Opening videos…';$('loadingText').textContent=state.inspectStatus==='waiting'?(state.engineMessage||'Preparing FFmpeg'):'Reading duration, dimensions and audio streams.';$('loadingFill').style.width=(state.engineProgress||0)+'%'}if(state.engineStatus==='error'){show('loading');$('engineError').textContent=state.engineMessage;$('engineError').classList.remove('hidden');$('retryEngine').classList.remove('hidden')}if(state.inspectStatus==='error'){show('loading');showLoadError(state.inspectError||'A video could not be inspected.')}if(state.inspectStatus==='ready'){show('editor');if(selected>=state.videos.length)selected=0;renderClips();updatePreview();estimate()}
let ids=['preset','start','end','speed','fps','fit','seekSeconds','paletteMode','ditherMode','compression','audio','volume','normalize','limiter','romTitle','outputMode','loop','resume'];ids.forEach(id=>$(id).disabled=!!state.converting);$('convert').disabled=!!state.converting;
if(state.converting){$('progressWrap').classList.remove('hidden');$('done').classList.add('hidden');$('convertError').classList.add('hidden');$('progressText').textContent=state.progressMessage;$('progressPct').textContent=state.progress+'%';$('progressFill').style.width=state.progress+'%'}else if(state.result){$('progressWrap').classList.remove('hidden');$('progressText').textContent='Output created successfully';$('progressPct').textContent='100%';$('progressFill').style.width='100%';$('done').classList.remove('hidden')}else if(state.convertError){$('convertError').textContent=state.convertError;$('convertError').classList.remove('hidden')}}
async function removeVideo(i){try{await api('/video/remove',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({index:i})});lastPreviewKey='';$('previewImage').removeAttribute('src');selected=Math.max(0,Math.min(selected,state.videos.length-2));await poll();if(!state?.videos?.length){selected=0;$('clipInfo').textContent='';show('welcome')}}catch(e){alert(e.message)}}
function renderClips(){let h='';state.videos.forEach((v,i)=>{let inf=v.info;h+='<div class="clip '+(i===selected?'active':'')+'" data-i="'+i+'"><div class="clip-info"><b>'+(i+1)+'. '+escapeHTML(v.name)+'</b><small>'+(inf?(inf.width+'×'+inf.height+' • '+fmt(inf.duration)+(inf.audioStreams?' • audio':' • silent')):v.status)+'</small></div><button class="clip-remove" type="button" data-remove="'+i+'" title="Remove this video" aria-label="Remove '+escapeHTML(v.name)+'">×</button></div>'});$('clips').innerHTML=h;[...$('clips').querySelectorAll('.clip')].forEach(el=>el.onclick=e=>{if(e.target.closest('.clip-remove'))return;selected=+el.dataset.i;renderClips();updatePreview()});[...$('clips').querySelectorAll('.clip-remove')].forEach(el=>el.onclick=e=>{e.stopPropagation();removeVideo(+el.dataset.remove)});let v=state.videos[selected];if(v?.info)$('clipInfo').textContent='Previewing '+v.name+' • '+v.info.fps.toFixed(2)+' source fps';else $('clipInfo').textContent='';if(romTitleAuto&&state.videos[0])$('romTitle').value=titleName(state.videos[0].name);let modeSel=$('outputMode'),current=modeSel.value;if(state.videos.length===1){modeSel.innerHTML='<option value="rom">Single ROM</option>';modeSel.value='rom'}else{modeSel.innerHTML='<option value="playlist">One ROM — play clips in order</option><option value="menu">One ROM — clip menu</option><option value="batch">Separate ROMs in ZIP</option>';modeSel.value=['playlist','menu','batch'].includes(current)?current:'playlist'}}
function escapeHTML(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
let previewTimer;function updatePreview(){if(!state?.videos?.[selected]?.info)return;clearTimeout(previewTimer);previewTimer=setTimeout(()=>{let v=state.videos[selected],t=previewMode==='end'?parseClock($('end').value):parseClock($('start').value);if(previewMode==='end'&&(!$('end').value.trim()||!isFinite(t)))t=v.info.duration;t=Math.min(Math.max(0,t),v.info.duration);let key=[selected,t.toFixed(3),$('fit').value,previewMode].join('|');if(key===lastPreviewKey&&$('previewImage').src)return;lastPreviewKey=key;let img=$('previewImage');img.onerror=()=>{img.removeAttribute('src')};img.src=BASE+'/preview?index='+selected+'&time='+encodeURIComponent(t)+'&fit='+$('fit').value+'&key='+encodeURIComponent(key)},160)}$('previewStart').onclick=()=>{previewMode='start';lastPreviewKey='';updatePreview()};$('previewEnd').onclick=()=>{previewMode='end';lastPreviewKey='';updatePreview()};
function values(){return{start:$('start').value,end:$('end').value,speed:+$('speed').value,fps:$('fps').value,fit:$('fit').value,audio:$('audio').value,volume:+$('volume').value,loop:$('loop').checked,romTitle:$('romTitle').value,seekSeconds:+$('seekSeconds').value,normalize:$('normalize').checked,limiter:$('limiter').checked,resume:$('resume').checked,compression:$('compression').value,paletteMode:$('paletteMode').value,ditherMode:$('ditherMode').value,outputMode:$('outputMode').value}}
const presets={best:{fps:'smooth',audio:'mix',paletteMode:'scene',ditherMode:'error',compression:'delta',normalize:true,limiter:true},balanced:{fps:'balanced',audio:'mix',paletteMode:'shared',ditherMode:'ordered',compression:'delta',normalize:false,limiter:true},long:{fps:'compact',audio:'mix',paletteMode:'shared',ditherMode:'ordered',compression:'delta',normalize:false,limiter:true},small:{fps:'compact',audio:'none',paletteMode:'shared',ditherMode:'off',compression:'delta',normalize:false,limiter:false}};
function applyPreset(){let p=presets[$('preset').value];if(!p)return;Object.entries(p).forEach(([k,v])=>{let e=$(k);if(e.type==='checkbox')e.checked=v;else e.value=v});estimate();updatePreview()}$('preset').onchange=applyPreset;
function markCustom(e){if(e.target.id!=='preset')$('preset').value='custom'}
function estimate(){if(!state?.videos?.length)return;let v=values(),start=parseClock(v.start),endText=v.end.trim(),vb={smooth:4,balanced:5,classic:6,compact:8}[v.fps],fps=59.727500569606/vb,totalFrames=0,raw=16384+state.videos.length*96,totalDur=0;for(let x of state.videos){if(!x.info)continue;let end=endText?Math.min(parseClock(endText),x.info.duration):x.info.duration;if(!isFinite(start)||!isFinite(end)||end<=start){$('estimate').textContent='Check trim settings.';return}let d=(end-start)/v.speed,f=Math.max(1,Math.ceil(d*fps));totalFrames+=f;totalDur+=end-start;let pals=v.paletteMode==='scene'?Math.ceil(f/60):1;raw+=pals*512+(pals>1?f*2:0)+f*9600+(v.compression==='delta'?f*16:0);if(v.audio!=='none'&&x.info.audioStreams)raw+=f*4+Math.ceil((f*vb/59.7275)*16384/16)*16}let p=1048576;while(p<raw)p*=2;let bps=fps*9600+(v.audio!=='none'?16384+fps*4:0),limit=(33554432-16384)/bps*v.speed;$('estimate').innerHTML=(raw>33554432?'<b style="color:#ff8d8d">Worst-case estimate exceeds 32 MiB</b>':'Estimated cartridge: <b>'+(p/1048576)+' MiB</b>')+'<br>Worst-case data: '+(raw/1048576).toFixed(2)+' MiB • '+totalFrames+' frames • '+fps.toFixed(2)+' fps<br>Approximate single-clip duration limit: '+fmt(limit)+' <span class="pill">delta compression may improve it</span>'}
for(let id of ['start','end','speed','fps','fit','seekSeconds','paletteMode','ditherMode','compression','audio','volume','normalize','limiter'])$(id).addEventListener('input',e=>{markCustom(e);estimate();if(['start','end','fit'].includes(id))updatePreview()});$('romTitle').addEventListener('input',()=>{romTitleAuto=false});
$('audioPreview').onclick=async()=>{try{$('audioPreview').disabled=true;let r=await fetch(BASE+'/audio-preview?index='+selected,{method:'POST',headers:headers({'Content-Type':'application/json'}),body:JSON.stringify(values())});if(!r.ok){let x=await r.json();throw Error(x.error)}let b=await r.blob();if(audioURL)URL.revokeObjectURL(audioURL);audioURL=URL.createObjectURL(b);$('audioPlayer').src=audioURL;$('audioPlayer').play()}catch(e){alert(e.message)}finally{$('audioPreview').disabled=false}};
$('convert').onclick=async()=>{try{await api('/convert',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(values())});poll()}catch(e){$('convertError').textContent=e.message;$('convertError').classList.remove('hidden')}};$('download').onclick=()=>{let a=document.createElement('a');a.href=BASE+'/download';a.download=state.downloadName||'GBA_Video_Maker_output';a.click()};$('retryEngine').onclick=()=>api('/engine/retry',{method:'POST'});$('resetTop').onclick=async()=>{await api('/reset',{method:'POST'});state=null;selected=0;lastPreviewKey='';romTitleAuto=true;$('romTitle').value='';show('welcome')};setInterval(()=>fetch(BASE+'/heartbeat',{method:'POST',headers:headers(),keepalive:true}).catch(()=>{}),5000);window.addEventListener('pagehide',()=>fetch(BASE+'/close-intent',{method:'POST',headers:headers(),keepalive:true}).catch(()=>{}));poll();
</script></body></html>`
