package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
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

const maxUploadBytes = int64(8 * 1024 * 1024 * 1024)

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

	token       string
	sessionDir  string
	ffmpegPath  string
	allowedHost string

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

func locateFFmpeg() string {
	for _, candidate := range []string{
		filepath.Join(appDirectory(), "ffmpeg.exe"),
		filepath.Join(appDirectory(), "tools", "ffmpeg.exe"),
	} {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() && st.Size() > 1_000_000 {
			return candidate
		}
	}
	return commandExists("ffmpeg")
}

func newAppState(token, sessionDir string) *appState {
	s := &appState{token: token, sessionDir: sessionDir, lastHeartbeat: time.Now(), inspectStatus: "idle"}
	if ff := locateFFmpeg(); ff != "" {
		s.ffmpegPath, s.engineStatus, s.engineProgress, s.engineMessage = ff, "ready", 100, "Conversion engine ready"
	} else {
		s.engineStatus = "missing"
		s.engineMessage = "FFmpeg is missing. Place ffmpeg.exe beside GBA Video Maker.exe, then click Check again."
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

func (s *appState) refreshEngine() {
	ff := locateFFmpeg()
	s.mu.Lock()
	if ff == "" {
		s.ffmpegPath = ""
		s.engineStatus = "missing"
		s.engineProgress = 0
		s.engineMessage = "FFmpeg is missing. Place ffmpeg.exe beside GBA Video Maker.exe, then click Check again."
		s.mu.Unlock()
		return
	}
	s.ffmpegPath = ff
	s.engineStatus = "ready"
	s.engineProgress = 100
	s.engineMessage = "Conversion engine ready"
	pending := len(s.videos) > 0 && s.inspectStatus != "ready"
	s.mu.Unlock()
	if pending {
		s.startInspection()
	}
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
		s.refreshEngine()
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

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; connect-src 'self'; frame-ancestors 'none'; form-action 'none'; img-src 'self' blob: data:; media-src 'self' blob:; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *appState) localRequestGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.allowedHost != "" && r.Host != s.allowedHost {
			http.Error(w, "invalid local host", http.StatusForbidden)
			return
		}
		if r.RemoteAddr != "" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				ip := net.ParseIP(host)
				if ip == nil || !ip.IsLoopback() {
					http.Error(w, "local access only", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *appState) apiRequestGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.allowedHost != "" && r.Method != http.MethodGet && r.Method != http.MethodHead {
			provided := r.Header.Get("X-GBA-Token")
			if len(provided) != len(s.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) != 1 {
				errorJSON(w, http.StatusForbidden, errors.New("invalid session token"))
				return
			}
		}
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
	mux.HandleFunc(prefix+"/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(appCSS)
	})
	mux.HandleFunc(prefix+"/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(appJS)
	})
	api := http.NewServeMux()
	api.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) { s.touch(); jsonResponse(w, 200, s.snapshot()) })
	api.HandleFunc("/heartbeat", func(w http.ResponseWriter, r *http.Request) { s.touch(); w.WriteHeader(204) })
	api.HandleFunc("/engine/retry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		s.refreshEngine()
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
	mux.Handle(prefix+"/api/", http.StripPrefix(prefix+"/api", s.apiRequestGuard(api)))
	return securityHeaders(recovery(s.localRequestGuard(mux)))
}

func renderPage(token string) ([]byte, error) {
	if token == "" {
		return nil, errors.New("session token is empty")
	}
	page := strings.Replace(appHTML, "__SESSION_TOKEN__", html.EscapeString(token), 1)
	if page == appHTML {
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
	state.allowedHost = listener.Addr().String()
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
