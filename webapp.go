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
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxUploadBytes = int64(8 * 1024 * 1024 * 1024)

type uploadedVideo struct {
	ID         string
	Path       string
	SourcePath string
	Name       string
	Info       *MediaInfo
	Status     string
	Error      string
	Owned      bool
}

type publicVideo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Info        *MediaInfo `json:"info,omitempty"`
	Status      string     `json:"status"`
	Error       string     `json:"error"`
	NeedsRelink bool       `json:"needsRelink"`
}

type appState struct {
	mu        sync.Mutex
	previewMu sync.Mutex

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

type reorderVideoRequest struct {
	IDs []string `json:"ids"`
}

type clipSettingsRequest struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	UseProject         bool    `json:"useProject"`
	Start              string  `json:"start"`
	End                string  `json:"end"`
	Speed              float64 `json:"speed"`
	Fit                string  `json:"fit"`
	Audio              string  `json:"audio"`
	AudioTrack         int     `json:"audioTrack"`
	Volume             float64 `json:"volume"`
	Loop               bool    `json:"loop"`
	PaletteMode        string  `json:"paletteMode"`
	DitherMode         string  `json:"ditherMode"`
	ImageSeconds       float64 `json:"imageSeconds,omitempty"`
	MusicTitle         string  `json:"musicTitle,omitempty"`
	MusicArtist        string  `json:"musicArtist,omitempty"`
	MusicArtworkMode   string  `json:"musicArtworkMode,omitempty"`
	MusicArtworkPreset string  `json:"musicArtworkPreset,omitempty"`
	MusicArtworkCustom string  `json:"musicArtworkCustom,omitempty"`
	MusicSeekSeconds   int     `json:"musicSeekSeconds,omitempty"`
}

type convertRequest struct {
	Preset             string                    `json:"preset"`
	AudioQuality       string                    `json:"audioQuality"`
	SmartTargetMiB     int                       `json:"smartTargetMiB"`
	SmartPriority      string                    `json:"smartPriority"`
	Start              string                    `json:"start"`
	End                string                    `json:"end"`
	Speed              float64                   `json:"speed"`
	FPS                string                    `json:"fps"`
	Fit                string                    `json:"fit"`
	Audio              string                    `json:"audio"`
	Volume             float64                   `json:"volume"`
	Loop               bool                      `json:"loop"`
	RomTitle           string                    `json:"romTitle"`
	SeekSeconds        int                       `json:"seekSeconds"`
	Normalize          bool                      `json:"normalize"`
	Limiter            bool                      `json:"limiter"`
	Resume             bool                      `json:"resume"`
	Compression        string                    `json:"compression"`
	PaletteMode        string                    `json:"paletteMode"`
	DitherMode         string                    `json:"ditherMode"`
	ImageSeconds       float64                   `json:"imageSeconds,omitempty"`
	OutputMode         string                    `json:"outputMode"`
	SplitVideo         bool                      `json:"splitVideo"`
	SplitBudgetMiB     int                       `json:"splitBudgetMiB"`
	MaxPartDuration    string                    `json:"maxPartDuration"`
	MaxPartMinutes     float64                   `json:"maxPartMinutes,omitempty"` // legacy project files
	ChapterAware       bool                      `json:"chapterAware"`
	PartTitleScreens   bool                      `json:"partTitleScreens"` // legacy mirror of titleCards.enabled
	ResumeLongSplit    bool                      `json:"resumeLongSplit"`
	TitleCards         *TitleCardProjectSettings `json:"titleCards,omitempty"`
	MenuBackground     string                    `json:"menuBackground"`
	MenuUIColor        string                    `json:"menuUIColor"`
	MenuSelectionColor string                    `json:"menuSelectionColor"`
	MenuOutline        bool                      `json:"menuOutline"`
	MenuOutlineColor   string                    `json:"menuOutlineColor"`
	MenuTheme          *MenuThemeOptions         `json:"menuTheme,omitempty"`
	Clips              []clipSettingsRequest     `json:"clips"`
}

type projectClip struct {
	Path         string              `json:"path"`
	Name         string              `json:"name"`
	Size         int64               `json:"size,omitempty"`
	LastModified int64               `json:"lastModified,omitempty"`
	Settings     clipSettingsRequest `json:"settings"`
}

type projectDocument struct {
	Format     string         `json:"format"`
	Version    int            `json:"version"`
	AppVersion string         `json:"appVersion"`
	Settings   convertRequest `json:"settings"`
	Clips      []projectClip  `json:"clips"`
}

type projectOpenResponse struct {
	Cancelled bool           `json:"cancelled"`
	Settings  convertRequest `json:"settings,omitempty"`
}

type languagePreferenceRequest struct {
	Language string `json:"language"`
}

type appPreferences struct {
	Language string `json:"language,omitempty"`
}

type localeManifest struct {
	Fallback  string `json:"fallback"`
	Languages []struct {
		Code     string `json:"code"`
		File     string `json:"file"`
		FlagFile string `json:"flagFile"`
	} `json:"languages"`
}

var (
	localeManifestOnce sync.Once
	localeManifestData localeManifest
	localeManifestErr  error
)

func loadLocaleManifest() (localeManifest, error) {
	localeManifestOnce.Do(func() {
		data, err := localeFS.ReadFile("locales/index.json")
		if err != nil {
			localeManifestErr = err
			return
		}
		localeManifestErr = json.Unmarshal(data, &localeManifestData)
	})
	return localeManifestData, localeManifestErr
}

func normalizeAppLanguage(value string) string {
	code := strings.ToLower(strings.TrimSpace(value))
	if split := strings.IndexAny(code, "-_"); split >= 0 {
		code = code[:split]
	}
	manifest, err := loadLocaleManifest()
	if err != nil {
		return ""
	}
	for _, language := range manifest.Languages {
		if strings.EqualFold(language.Code, code) {
			return strings.ToLower(language.Code)
		}
	}
	return ""
}

func localeAssetAllowed(name string) bool {
	if name == "index.json" {
		return true
	}
	manifest, err := loadLocaleManifest()
	if err != nil {
		return false
	}
	for _, language := range manifest.Languages {
		if language.File == name || language.FlagFile == name {
			return true
		}
	}
	return false
}

func preferencesFilePath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "GBA Media Maker", "settings.json")
	}
	return filepath.Join(appDirectory(), "GBA Media Maker.settings.json")
}

func loadLanguagePreference() string {
	data, err := os.ReadFile(preferencesFilePath())
	if err != nil {
		return ""
	}
	var prefs appPreferences
	if json.Unmarshal(data, &prefs) != nil {
		return ""
	}
	return normalizeAppLanguage(prefs.Language)
}

func saveLanguagePreference(language string) error {
	language = normalizeAppLanguage(language)
	if language == "" {
		return errors.New("unsupported interface language")
	}
	path := preferencesFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(appPreferences{Language: language}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func appDirectory() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func logDiagnostic(section string, recovered any) {
	path := diagnosticLogPath()
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

func newVideoID() string {
	if token, err := randomToken(); err == nil && len(token) >= 16 {
		return token[:16]
	}
	return fmt.Sprintf("clip-%d", time.Now().UnixNano())
}

func locateFFmpeg() string {
	candidates := make([]string, 0, 12)
	for _, name := range []string{"ffmpeg.exe", "ffmpeg"} {
		candidates = append(candidates,
			filepath.Join(appDirectory(), name),
			filepath.Join(appDirectory(), "tools", name),
		)
	}
	candidates = append(candidates, extraFFmpegCandidates()...)
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if candidate == "." || seen[candidate] {
			continue
		}
		seen[candidate] = true
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
		s.engineMessage = "FFmpeg is missing. Place FFmpeg beside GBA Media Maker or install it on PATH, then click Check again."
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
		videos[i] = publicVideo{ID: v.ID, Name: v.Name, Info: info, Status: v.Status, Error: v.Error, NeedsRelink: v.SourcePath == "" || v.Status == "missing"}
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

// ensurePreview serializes desktop preview generation and publishes the file
// atomically. Several UI refreshes may ask for the same frame at once; only the
// first request is allowed to start FFmpeg, while the others reuse its result.
func (s *appState) ensurePreview(ctx context.Context, ffmpegPath, input string, timeSec float64, fit, outPath string) error {
	if st, err := os.Stat(outPath); err == nil && st.Size() > 0 {
		return nil
	}
	s.previewMu.Lock()
	defer s.previewMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if st, err := os.Stat(outPath); err == nil && st.Size() > 0 {
		return nil
	}
	tempPath := fmt.Sprintf("%s.%d.png", outPath, time.Now().UnixNano())
	defer os.Remove(tempPath)
	if err := generatePreviewContext(ctx, ffmpegPath, input, timeSec, fit, tempPath); err != nil {
		return err
	}
	if st, err := os.Stat(tempPath); err != nil || st.Size() == 0 {
		return errors.New("FFmpeg returned no preview frame")
	}
	_ = os.Remove(outPath)
	return os.Rename(tempPath, outPath)
}

func (s *appState) ensureAudioPreview(ctx context.Context, ffmpegPath, input, fit, fallbackPreset, outPath string) error {
	if st, err := os.Stat(outPath); err == nil && st.Size() > 0 {
		return nil
	}
	s.previewMu.Lock()
	defer s.previewMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if st, err := os.Stat(outPath); err == nil && st.Size() > 0 {
		return nil
	}
	tempPath := fmt.Sprintf("%s.%d.png", outPath, time.Now().UnixNano())
	defer os.Remove(tempPath)
	ctx2, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, artErr := runCommandContext(ctx2, ffmpegPath, "-y", "-hide_banner", "-loglevel", "error", "-threads", "1", "-i", input, "-map", "0:v:0", "-frames:v", "1", "-an", "-sn", "-dn", "-vf", makePreviewFilter(fit), "-threads", "1", "-f", "image2", tempPath)
	if artErr != nil {
		_ = os.Remove(tempPath)
		data, fallbackErr := audioArtworkPresetPNG(fallbackPreset)
		if fallbackErr != nil {
			return fallbackErr
		}
		artErr = os.WriteFile(tempPath, data, 0644)
	}
	if artErr != nil {
		return artErr
	}
	if st, err := os.Stat(tempPath); err != nil || st.Size() == 0 {
		return errors.New("FFmpeg returned no artwork preview")
	}
	_ = os.Remove(outPath)
	return os.Rename(tempPath, outPath)
}

func (s *appState) refreshEngine() {
	ff := locateFFmpeg()
	s.mu.Lock()
	if ff == "" {
		s.ffmpegPath = ""
		s.engineStatus = "missing"
		s.engineProgress = 0
		s.engineMessage = "FFmpeg is missing. Place FFmpeg beside GBA Media Maker or install it on PATH, then click Check again."
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
		videos = append(videos, uploadedVideo{ID: newVideoID(), Path: path, Name: name, Status: "waiting", Owned: true})
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
	if victim.Owned {
		_ = os.Remove(victim.Path)
	}
	return nil
}

func (s *appState) addUploaded(videos []uploadedVideo, appendMode bool) {
	for i := range videos {
		if videos[i].ID == "" {
			videos[i].ID = newVideoID()
		}
	}
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
			if v.Owned {
				_ = os.Remove(v.Path)
			}
		}
	}
	if s.engineStatus == "ready" {
		s.startInspection()
	} else {
		s.refreshEngine()
	}
}

func videosFromPaths(paths []string) ([]uploadedVideo, error) {
	videos := make([]uploadedVideo, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			return nil, fmt.Errorf("cannot open %s", filepath.Base(path))
		}
		videos = append(videos, uploadedVideo{
			ID: newVideoID(), Path: path, SourcePath: path,
			Name: sanitizeFilename(filepath.Base(path)), Status: "waiting",
		})
	}
	if len(videos) == 0 {
		return nil, errors.New("no videos were selected")
	}
	return videos, nil
}

func (s *appState) addVideoPaths(paths []string, appendMode bool) error {
	videos, err := videosFromPaths(paths)
	if err != nil {
		return err
	}
	s.addUploaded(videos, appendMode)
	return nil
}

func (s *appState) reorderVideos(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.converting {
		return errors.New("wait for the current conversion to finish")
	}
	if len(ids) != len(s.videos) {
		return errors.New("video order is incomplete")
	}
	byID := make(map[string]uploadedVideo, len(s.videos))
	for _, video := range s.videos {
		byID[video.ID] = video
	}
	ordered := make([]uploadedVideo, 0, len(ids))
	for _, id := range ids {
		video, ok := byID[id]
		if !ok {
			return errors.New("video order contains an unknown clip")
		}
		ordered = append(ordered, video)
		delete(byID, id)
	}
	if len(byID) != 0 {
		return errors.New("video order contains duplicates")
	}
	s.videos = ordered
	s.result, s.convertError, s.downloadName = nil, "", ""
	return nil
}

func (s *appState) relinkVideo(index int) (bool, error) {
	paths, err := openFilesDialog("Relink media", mediaDialogFilter, false)
	if err != nil {
		if errors.Is(err, errDialogCancelled) {
			return true, nil
		}
		return false, err
	}
	if len(paths) != 1 {
		return false, errors.New("select one replacement video")
	}
	path := filepath.Clean(paths[0])
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false, errors.New("the selected replacement file cannot be opened")
	}
	s.mu.Lock()
	if index < 0 || index >= len(s.videos) {
		s.mu.Unlock()
		return false, errors.New("video index out of range")
	}
	old := s.videos[index]
	s.videos[index].Path = path
	s.videos[index].SourcePath = path
	s.videos[index].Name = sanitizeFilename(filepath.Base(path))
	s.videos[index].Info = nil
	s.videos[index].Status = "waiting"
	s.videos[index].Error = ""
	s.videos[index].Owned = false
	s.inspectStatus = "inspecting"
	s.result, s.convertError, s.downloadName = nil, "", ""
	s.mu.Unlock()
	if old.Owned {
		_ = os.Remove(old.Path)
	}
	s.startInspection()
	return false, nil
}

func clipSettingsByID(clips []clipSettingsRequest) map[string]clipSettingsRequest {
	out := make(map[string]clipSettingsRequest, len(clips))
	for _, clip := range clips {
		out[clip.ID] = clip
	}
	return out
}

func (s *appState) saveProject(req convertRequest) (bool, string, error) {
	s.mu.Lock()
	videos := append([]uploadedVideo(nil), s.videos...)
	s.mu.Unlock()
	if len(videos) == 0 {
		return false, "", errors.New("there is no project to save")
	}
	settings := clipSettingsByID(req.Clips)
	doc := projectDocument{Format: "gba-media-maker-project", Version: 2, AppVersion: appVersion, Settings: req}
	doc.Settings.Clips = nil
	for _, video := range videos {
		path := video.SourcePath
		if path == "" {
			return false, "", fmt.Errorf("%s was imported through drag and drop; use its Relink button before saving the project", video.Name)
		}
		st, statErr := os.Stat(path)
		if statErr != nil || st.IsDir() {
			return false, "", fmt.Errorf("%s needs to be relinked before saving", video.Name)
		}
		clip, ok := settings[video.ID]
		if !ok {
			return false, "", fmt.Errorf("settings for %s are missing", video.Name)
		}
		clip.ID = ""
		doc.Clips = append(doc.Clips, projectClip{
			Path: path, Name: video.Name, Size: st.Size(), LastModified: st.ModTime().UnixMilli(), Settings: clip,
		})
	}
	path, err := saveFileDialog("Save GBA Media Maker project", projectDialogFilter, "gbamedia", "My Project.gbamedia")
	if err != nil {
		if errors.Is(err, errDialogCancelled) {
			return true, "", nil
		}
		return false, "", err
	}
	if ext := strings.ToLower(filepath.Ext(path)); ext != ".gbamedia" && ext != ".gbavideo" {
		path += ".gbamedia"
	}
	projectDir := filepath.Dir(path)
	for i := range doc.Clips {
		if rel, relErr := filepath.Rel(projectDir, doc.Clips[i].Path); relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			doc.Clips[i].Path = rel
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return false, "", err
	}
	return false, path, nil
}

func (s *appState) openProject() (projectOpenResponse, error) {
	paths, err := openFilesDialog("Open GBA Media Maker project", projectDialogFilter, false)
	if err != nil {
		if errors.Is(err, errDialogCancelled) {
			return projectOpenResponse{Cancelled: true}, nil
		}
		return projectOpenResponse{}, err
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return projectOpenResponse{}, err
	}
	if len(data) > 32<<20 {
		return projectOpenResponse{}, errors.New("project file is too large")
	}
	var doc projectDocument
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return projectOpenResponse{}, fmt.Errorf("invalid project file: %w", err)
	}
	if !((doc.Format == "gba-video-maker-project" && doc.Version == 1) || (doc.Format == "gba-media-maker-project" && doc.Version == 2)) || len(doc.Clips) == 0 {
		return projectOpenResponse{}, errors.New("unsupported or empty project file")
	}
	videos := make([]uploadedVideo, 0, len(doc.Clips))
	settings := doc.Settings
	settings.Clips = make([]clipSettingsRequest, 0, len(doc.Clips))
	projectDir := filepath.Dir(paths[0])
	for _, saved := range doc.Clips {
		id := newVideoID()
		rawPath := strings.TrimSpace(saved.Path)
		path := ""
		status := "waiting"
		var clipErr string
		if rawPath != "" {
			path = filepath.Clean(rawPath)
			if !filepath.IsAbs(path) {
				path = filepath.Join(projectDir, path)
			}
			if st, statErr := os.Stat(path); statErr != nil || st.IsDir() {
				status = "missing"
				clipErr = "Source file is missing. Relink this clip."
			}
		} else {
			status = "missing"
			clipErr = "Source file needs to be relinked on this computer."
		}
		name := sanitizeFilename(saved.Name)
		if name == "video" && path != "" {
			name = sanitizeFilename(filepath.Base(path))
		}
		videos = append(videos, uploadedVideo{ID: id, Path: path, SourcePath: path, Name: name, Status: status, Error: clipErr})
		clip := saved.Settings
		clip.ID = id
		settings.Clips = append(settings.Clips, clip)
	}
	s.mu.Lock()
	if s.converting {
		s.mu.Unlock()
		return projectOpenResponse{}, errors.New("wait for the current conversion to finish")
	}
	old := s.videos
	s.videos = videos
	s.result, s.convertError, s.downloadName = nil, "", ""
	s.inspectStatus, s.inspectError = "inspecting", ""
	s.mu.Unlock()
	for _, video := range old {
		if video.Owned {
			_ = os.Remove(video.Path)
		}
	}
	s.startInspection()
	return projectOpenResponse{Settings: settings}, nil
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
				s.inspectStatus, s.inspectError = "error", "Media inspection stopped unexpectedly."
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
			if strings.TrimSpace(path) == "" {
				continue
			}
			if st, statErr := os.Stat(path); statErr != nil || st.IsDir() {
				s.mu.Lock()
				if i < len(s.videos) && s.videos[i].Path == path {
					s.videos[i].Status = "missing"
					s.videos[i].Error = "Source file is missing. Relink this clip."
				}
				s.mu.Unlock()
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
		allSettled := len(s.videos) > 0
		hasFatalError := false
		for _, v := range s.videos {
			if v.Status == "inspecting" || v.Status == "waiting" {
				allSettled = false
			}
			if v.Status == "error" {
				hasFatalError = true
				if v.Error != "" {
					s.inspectError = v.Name + ": " + v.Error
				}
			}
		}
		if allSettled && !hasFatalError {
			s.inspectStatus = "ready"
		} else if hasFatalError {
			s.inspectStatus = "error"
		}
		s.mu.Unlock()
	}()
}

func normalizeTitle(base string) string {
	title := sanitizeGBAText(base, 12)
	if title == "" {
		title = "GBA VIDEO"
	}
	return title
}

func parseOptionalEnd(value string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return parseTime(value)
}

func validateWebClipSettings(label string, start, end, speed, volume float64, fit, audio, palette, dither string, duration float64) error {
	if start < 0 || start >= duration {
		return fmt.Errorf("%s: start time is outside the video", label)
	}
	if end > 0 && (end <= start || end > duration+0.001) {
		return fmt.Errorf("%s: end time must be after start and inside the video", label)
	}
	if speed < .5 || speed > 3 {
		return fmt.Errorf("%s: speed must be between 0.50 and 3.00", label)
	}
	if volume < 0 || volume > 200 {
		return fmt.Errorf("%s: volume must be between 0 and 200", label)
	}
	if fit != "crop" && fit != "fit" && fit != "stretch" {
		return fmt.Errorf("%s: invalid screen framing", label)
	}
	if audio != "mix" && audio != "left" && audio != "right" && audio != "none" {
		return fmt.Errorf("%s: invalid audio mode", label)
	}
	if palette != "shared" && palette != "scene" {
		return fmt.Errorf("%s: invalid palette mode", label)
	}
	if dither != "off" && dither != "ordered" && dither != "error" {
		return fmt.Errorf("%s: invalid dithering mode", label)
	}
	return nil
}

func parseMaximumPartDuration(value string, legacyMinutes float64) (float64, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		if legacyMinutes > 0 {
			if legacyMinutes > 240 {
				return 0, errors.New("maximum part duration must not exceed 240:00")
			}
			return legacyMinutes * 60, nil
		}
		return 0, nil
	}
	if text == "0" {
		return 0, nil
	}
	parts := strings.Split(text, ":")
	if len(parts) != 2 {
		return 0, errors.New("maximum part duration must be 0 or MM:SS, for example 1:05")
	}
	minutes, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || minutes < 0 {
		return 0, errors.New("maximum part duration must be 0 or MM:SS, for example 1:05")
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || seconds < 0 || seconds > 59 || len(strings.TrimSpace(parts[1])) != 2 {
		return 0, errors.New("maximum part duration must be 0 or MM:SS, for example 1:05")
	}
	total := float64(minutes*60 + seconds)
	if total <= 0 || total > 240*60 {
		return 0, errors.New("maximum part duration must be between 0:01 and 240:00")
	}
	return total, nil
}

func (s *appState) buildOptions(req convertRequest) (ProjectOptions, []MediaInfo, error) {
	s.mu.Lock()
	ff := s.ffmpegPath
	videos := append([]uploadedVideo(nil), s.videos...)
	s.mu.Unlock()
	if len(videos) == 0 {
		return ProjectOptions{}, nil, errors.New("media files are not ready")
	}
	if ff == "" {
		return ProjectOptions{}, nil, errors.New("conversion engine is not ready")
	}
	start, err := parseTime(req.Start)
	if err != nil {
		return ProjectOptions{}, nil, err
	}
	end, err := parseOptionalEnd(req.End)
	if err != nil {
		return ProjectOptions{}, nil, err
	}
	if req.Fit == "" {
		req.Fit = "fit"
	}
	if req.Speed == 0 {
		req.Speed = 1
	}
	if req.Volume == 0 && req.Audio != "none" {
		// Zero is a valid explicit volume, so only apply the default when the
		// field was omitted together with the other project defaults.
		if req.Start == "" && req.End == "" && req.Fit == "" {
			req.Volume = 100
		}
	}
	vblanks, ok := map[string]int{"smooth": 4, "balanced": 5, "classic": 6, "compact": 8}[req.FPS]
	if !ok {
		return ProjectOptions{}, nil, errors.New("invalid frame-rate preset")
	}
	if req.PaletteMode == "" {
		req.PaletteMode = "shared"
	}
	if req.DitherMode == "" {
		req.DitherMode = "ordered"
	}
	if req.Audio == "" {
		req.Audio = "mix"
	}
	if req.Preset == "" {
		req.Preset = "custom"
	}
	extreme := req.Preset == "extreme"
	if req.AudioQuality == "" {
		req.AudioQuality = audioCodecPCM
	}
	if !extreme {
		req.AudioQuality = audioCodecPCM
	}
	if req.AudioQuality != audioCodecPCM && req.AudioQuality != audioCodecADPCM && req.AudioQuality != audioCodecAuto {
		return ProjectOptions{}, nil, errors.New("invalid audio quality")
	}
	if req.SmartTargetMiB < 1 || req.SmartTargetMiB > 32 {
		req.SmartTargetMiB = 32
	}
	if req.SmartPriority == "" {
		req.SmartPriority = "balanced"
	}
	settingsByID := clipSettingsByID(req.Clips)
	var inputs []ClipInput
	var infos []MediaInfo
	for _, v := range videos {
		if v.Info == nil || v.Status != "ready" {
			return ProjectOptions{}, nil, fmt.Errorf("%s is not ready; relink it if the source moved", v.Name)
		}
		clipReq, exists := settingsByID[v.ID]
		if !exists {
			clipReq = clipSettingsRequest{ID: v.ID, Title: normalizeTitle(strings.TrimSuffix(v.Name, filepath.Ext(v.Name))), UseProject: true}
		}
		title := normalizeTitle(clipReq.Title)
		if strings.TrimSpace(clipReq.Title) == "" {
			title = normalizeTitle(strings.TrimSuffix(v.Name, filepath.Ext(v.Name)))
		}
		imageSeconds := clipReq.ImageSeconds
		if clipReq.UseProject {
			imageSeconds = req.ImageSeconds
		}
		artworkMode := strings.ToLower(strings.TrimSpace(clipReq.MusicArtworkMode))
		if artworkMode == "" {
			artworkMode = "embedded"
		}
		if artworkMode != "default" && artworkMode != "embedded" && artworkMode != "custom" {
			return ProjectOptions{}, nil, fmt.Errorf("%s: invalid audio artwork mode", v.Name)
		}
		artworkPresetValue := strings.TrimSpace(clipReq.MusicArtworkPreset)
		if artworkPresetValue == "" {
			artworkPresetValue = automaticAudioArtworkPreset(v.Name)
		}
		artworkPreset := normalizeAudioArtworkPreset(artworkPresetValue)
		if v.Info.Kind == "audio" && artworkMode == "custom" {
			if _, customErr := decodeCustomAudioArtworkDataURL(clipReq.MusicArtworkCustom); customErr != nil {
				return ProjectOptions{}, nil, fmt.Errorf("%s: %w", v.Name, customErr)
			}
		}
		musicSeekSeconds := 0
		if v.Info.Kind == "audio" {
			musicSeekSeconds = clipReq.MusicSeekSeconds
			if musicSeekSeconds == 0 {
				musicSeekSeconds = req.SeekSeconds
				if musicSeekSeconds == 0 {
					musicSeekSeconds = 5
				}
			}
			if musicSeekSeconds != 3 && musicSeekSeconds != 5 && musicSeekSeconds != 10 && musicSeekSeconds != 15 {
				return ProjectOptions{}, nil, fmt.Errorf("%s: audio seek step must be 3, 5, 10 or 15 seconds", v.Name)
			}
		}
		input := ClipInput{InputPath: v.Path, Name: v.Name, Title: title, AudioTrack: clipReq.AudioTrack, MediaKind: v.Info.Kind, ImageSeconds: imageSeconds, MusicTitle: clipReq.MusicTitle, MusicArtist: clipReq.MusicArtist, MusicArtworkMode: artworkMode, MusicArtworkPreset: artworkPreset, MusicArtworkCustom: clipReq.MusicArtworkCustom, MusicSeekSeconds: musicSeekSeconds}
		if isAnimatedGIFPath(v.Path) {
			input.MediaKind = "video"
			input.Loop = true
		}
		if input.AudioTrack < 0 || input.AudioTrack >= v.Info.AudioStreams {
			if v.Info.AudioStreams > 0 && input.AudioTrack != 0 {
				return ProjectOptions{}, nil, fmt.Errorf("%s: selected audio track %d is not available", v.Name, input.AudioTrack+1)
			}
			input.AudioTrack = 0
		}
		if !clipReq.UseProject {
			clipStart, parseErr := parseTime(clipReq.Start)
			if parseErr != nil {
				return ProjectOptions{}, nil, fmt.Errorf("%s: %w", v.Name, parseErr)
			}
			clipEnd, parseErr := parseOptionalEnd(clipReq.End)
			if parseErr != nil {
				return ProjectOptions{}, nil, fmt.Errorf("%s: %w", v.Name, parseErr)
			}
			if v.Info.Kind != "image" {
				if err := validateWebClipSettings(v.Name, clipStart, clipEnd, clipReq.Speed, clipReq.Volume, clipReq.Fit, clipReq.Audio, clipReq.PaletteMode, clipReq.DitherMode, v.Info.Duration); err != nil {
					return ProjectOptions{}, nil, err
				}
			}
			input.Custom = true
			input.Start = clipStart
			input.End = clipEnd
			input.Speed = clipReq.Speed
			input.FitMode = clipReq.Fit
			input.AudioMode = clipReq.Audio
			input.Volume = clipReq.Volume / 100
			input.Loop = clipReq.Loop || isAnimatedGIFPath(v.Path)
			input.PaletteMode = clipReq.PaletteMode
			input.DitherMode = clipReq.DitherMode
		}
		inputs = append(inputs, input)
		infos = append(infos, *v.Info)
	}
	if len(infos) > 0 && infos[0].Kind != "image" {
		if err := validateWebClipSettings("project defaults", start, end, req.Speed, req.Volume, req.Fit, req.Audio, req.PaletteMode, req.DitherMode, infos[0].Duration); err != nil {
			// The default end may be longer than the first clip but will be capped
			// independently by the converter, so only enforce the shared basics.
			if !strings.Contains(err.Error(), "end time") {
				return ProjectOptions{}, nil, err
			}
		}
	}
	if req.SeekSeconds == 0 {
		req.SeekSeconds = 5
	}
	if req.Compression == "" {
		req.Compression = "delta"
	}
	if req.SplitBudgetMiB == 0 {
		req.SplitBudgetMiB = 31
	}
	if req.SplitBudgetMiB < 1 || req.SplitBudgetMiB > 32 {
		return ProjectOptions{}, nil, errors.New("split ROM size must be between 1 and 32 MiB")
	}
	maxPartSeconds := 0.0
	if req.SplitVideo {
		maxPartSeconds, err = parseMaximumPartDuration(req.MaxPartDuration, req.MaxPartMinutes)
		if err != nil {
			return ProjectOptions{}, nil, err
		}
	} else {
		// Normal Single ROM mode still falls back to automatic splitting at the
		// physical 32 MiB cartridge limit, but hidden manual rules do not apply.
		req.SplitBudgetMiB = 32
		req.ChapterAware = true
		req.ResumeLongSplit = true
	}
	if req.TitleCards == nil && req.PartTitleScreens {
		req.TitleCards = &TitleCardProjectSettings{
			Enabled: true, UseShared: true,
			Shared: defaultTitleCardSettings(videos[0].Name),
		}
	}
	if req.TitleCards != nil {
		req.PartTitleScreens = req.TitleCards.Enabled
	}
	mode := req.OutputMode
	if len(inputs) == 1 {
		// A one-item ROM opens the media directly; a menu would only add an
		// unnecessary extra click.
		mode = "rom"
	} else if mode == "batch" {
		// Separate-ROM export remains available for users who do not want a
		// collection ROM.
		mode = "batch"
	} else {
		// Every multi-item collection uses the media menu, regardless of
		// whether it contains mixed media, only videos, only music, or only
		// images. Legacy playlist/rom project values are upgraded here.
		mode = "menu"
	}
	base := strings.TrimSuffix(sanitizeFilename(videos[0].Name), filepath.Ext(videos[0].Name))
	if len(inputs) > 1 {
		base = "GBA_Media_Collection"
	}
	ext := ".gba"
	if mode == "batch" {
		ext = ".zip"
	}
	output := filepath.Join(s.sessionDir, base+ext)
	romTitle := normalizeTitle(req.RomTitle)
	if strings.TrimSpace(req.RomTitle) == "" {
		romTitle = normalizeTitle(base)
	}
	opt := ProjectOptions{
		Inputs: inputs, OutputPath: output, FFmpegPath: ff,
		Start: start, End: end, Speed: req.Speed, VBlanks: vblanks,
		FitMode: req.Fit, AudioMode: req.Audio, Volume: req.Volume / 100,
		Loop: req.Loop, RomTitle: romTitle, SeekSeconds: req.SeekSeconds,
		Normalize: req.Normalize, Limiter: req.Limiter, Resume: req.Resume,
		Compression: req.Compression, PaletteMode: req.PaletteMode,
		DitherMode: req.DitherMode, OutputMode: mode,
		KeyInterval: 30, SplitBudgetMiB: req.SplitBudgetMiB,
		MaxPartMinutes: maxPartSeconds / 60, ChapterAware: req.ChapterAware,
		PartTitleScreens: req.PartTitleScreens, ResumeLongSplit: req.ResumeLongSplit,
		TitleCards: req.TitleCards, MenuTheme: req.MenuTheme,
		Preset: req.Preset, AudioCodec: req.AudioQuality,
		ExtremeOptimization: extreme, AdaptiveKeyframes: extreme,
		EnhancedSceneDetection: extreme, SmartTargetMiB: req.SmartTargetMiB,
		SmartPriority: req.SmartPriority,
	}
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
				s.convertError = "The converter stopped unexpectedly. See GBA Media Maker.log."
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
		if result.AutoSplit {
			s.progressMessage = fmt.Sprintf("Video was automatically split into %d numbered ROMs", result.ClipCount)
		} else {
			s.progressMessage = "Output created successfully"
		}
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
		if v.Owned {
			_ = os.Remove(v.Path)
		}
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
				errorJSON(w, 500, errors.New("internal error; see GBA Media Maker.log"))
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
	mux.HandleFunc(prefix+"/audio-artwork/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, prefix+"/audio-artwork/"), ".png")
		if normalizeAudioArtworkPreset(name) != name {
			http.NotFound(w, r)
			return
		}
		data, err := audioArtworkPresetPNG(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	})
	mux.HandleFunc(prefix+"/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(appCSS)
	})
	mux.HandleFunc(prefix+"/i18n.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(appI18nJS)
	})
	mux.HandleFunc(prefix+"/locales/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, prefix+"/locales/")
		if filepath.Base(name) != name || !localeAssetAllowed(name) {
			http.NotFound(w, r)
			return
		}
		data, err := localeFS.ReadFile("locales/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if strings.EqualFold(filepath.Ext(name), ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})
	mux.HandleFunc(prefix+"/gba-text.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(gbaTextJS)
	})
	mux.HandleFunc(prefix+"/menu-themes.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(menuThemesJS)
	})
	mux.HandleFunc(prefix+"/title-cards.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(titleCardsJS)
	})
	mux.HandleFunc(prefix+"/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(appJS)
	})
	api := http.NewServeMux()
	api.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) { s.touch(); jsonResponse(w, 200, s.snapshot()) })
	api.HandleFunc("/heartbeat", func(w http.ResponseWriter, r *http.Request) { s.touch(); w.WriteHeader(204) })
	api.HandleFunc("/preferences/language", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req languagePreferenceRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid language preference"})
			return
		}
		if err := saveLanguagePreference(req.Language); err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
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
	api.HandleFunc("/menu-background/video", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		start, err := strconv.ParseFloat(r.URL.Query().Get("start"), 64)
		if err != nil || start < 0 {
			errorJSON(w, 400, errors.New("invalid menu background start time"))
			return
		}
		duration, err := strconv.ParseFloat(r.URL.Query().Get("duration"), 64)
		if err != nil {
			errorJSON(w, 400, errors.New("invalid menu background duration"))
			return
		}
		if _, _, _, err := menuBackgroundSampling(duration); err != nil {
			errorJSON(w, 400, err)
			return
		}
		s.mu.Lock()
		ff := s.ffmpegPath
		busy := s.converting
		s.mu.Unlock()
		if busy {
			errorJSON(w, 409, errors.New("wait for the current conversion to finish"))
			return
		}
		if ff == "" {
			errorJSON(w, 503, errors.New("FFmpeg is not available"))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			errorJSON(w, 400, fmt.Errorf("invalid menu background upload: %w", err))
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		source, header, err := r.FormFile("video")
		if err != nil {
			errorJSON(w, 400, errors.New("menu background video is missing"))
			return
		}
		defer source.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" || len(ext) > 10 {
			ext = ".video"
		}
		inputPath := filepath.Join(s.sessionDir, fmt.Sprintf("menu-background-%d%s", time.Now().UnixNano(), ext))
		outputPath := filepath.Join(s.sessionDir, fmt.Sprintf("menu-background-%d.rgb", time.Now().UnixNano()))
		defer os.Remove(inputPath)
		defer os.Remove(outputPath)
		destination, err := os.Create(inputPath)
		if err != nil {
			errorJSON(w, 500, err)
			return
		}
		_, copyErr := io.Copy(destination, source)
		closeErr := destination.Close()
		if copyErr != nil {
			errorJSON(w, 500, copyErr)
			return
		}
		if closeErr != nil {
			errorJSON(w, 500, closeErr)
			return
		}

		data, count, frameVBlanks, err := extractMenuBackgroundVideoFrames(r.Context(), ff, inputPath, start, duration, outputPath)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Menu-Frame-Count", strconv.Itoa(count))
		w.Header().Set("X-Menu-Frame-VBlanks", strconv.Itoa(frameVBlanks))
		w.WriteHeader(200)
		_, _ = w.Write(data)
	})
	api.HandleFunc("/dialog/videos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		paths, err := openFilesDialog("Choose source media", mediaDialogFilter, true)
		if err != nil {
			if errors.Is(err, errDialogCancelled) {
				jsonResponse(w, 200, map[string]bool{"cancelled": true})
				return
			}
			if errors.Is(err, errDialogUnsupported) {
				errorJSON(w, http.StatusNotImplemented, err)
				return
			}
			errorJSON(w, 500, err)
			return
		}
		if err := s.addVideoPaths(paths, r.URL.Query().Get("append") == "1"); err != nil {
			errorJSON(w, 400, err)
			return
		}
		jsonResponse(w, 202, map[string]int{"count": len(paths)})
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
	api.HandleFunc("/video/reorder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var req reorderVideoRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			errorJSON(w, 400, errors.New("invalid reorder request"))
			return
		}
		if err := s.reorderVideos(req.IDs); err != nil {
			errorJSON(w, 409, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"ok": true})
	})
	api.HandleFunc("/video/relink", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		idx, err := strconv.Atoi(r.URL.Query().Get("index"))
		if err != nil {
			errorJSON(w, 400, errors.New("invalid video index"))
			return
		}
		cancelled, err := s.relinkVideo(idx)
		if err != nil {
			errorJSON(w, 500, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"cancelled": cancelled})
	})
	api.HandleFunc("/project/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var req convertRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			errorJSON(w, 400, fmt.Errorf("invalid project settings: %w", err))
			return
		}
		cancelled, path, err := s.saveProject(req)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		jsonResponse(w, 200, map[string]any{"cancelled": cancelled, "path": path})
	})
	api.HandleFunc("/project/open", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		response, err := s.openProject()
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		jsonResponse(w, 200, response)
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
			fallbackPreset := normalizeAudioArtworkPreset(r.URL.Query().Get("artworkFallback"))
			cacheSuffix := fit
			if video.Info.Kind == "audio" {
				cacheSuffix += "-" + fallbackPreset
			}
			out = filepath.Join(s.sessionDir, fmt.Sprintf("preview-%d-%d-%s.png", idx, int(candidate*1000), cacheSuffix))
			if st, err := os.Stat(out); err == nil && st.Size() > 0 {
				previewErr = nil
				break
			}
			if video.Info.Kind == "audio" {
				previewErr = s.ensureAudioPreview(r.Context(), ff, video.Path, fit, fallbackPreset, out)
			} else {
				previewErr = s.ensurePreview(r.Context(), ff, video.Path, candidate, fit, out)
			}
			if previewErr == nil {
				break
			}
			if errors.Is(previewErr, context.Canceled) || errors.Is(previewErr, context.DeadlineExceeded) {
				return
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
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = w.Write(data)
	})
	api.HandleFunc("/smart-analyze", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var req convertRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			errorJSON(w, 400, fmt.Errorf("invalid settings: %w", err))
			return
		}
		if req.Preset != "extreme" {
			errorJSON(w, 400, errors.New("select Extreme optimization before analysis"))
			return
		}
		opt, infos, err := s.buildOptions(req)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		if len(opt.Inputs) != 1 || len(infos) != 1 {
			errorJSON(w, 400, errors.New("smart analysis currently requires exactly one source video"))
			return
		}
		tempDir, err := os.MkdirTemp(s.sessionDir, "smart-analysis-")
		if err != nil {
			errorJSON(w, 500, err)
			return
		}
		defer os.RemoveAll(tempDir)
		result, err := AnalyzeExtremeEncodingContext(r.Context(), opt.FFmpegPath, opt.Inputs[0].InputPath, infos[0], opt, req.SmartTargetMiB, req.SmartPriority, req.AudioQuality, tempDir)
		if err != nil {
			errorJSON(w, 400, err)
			return
		}
		jsonResponse(w, 200, result)
	})
	api.HandleFunc("/audio-preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		idx, _ := strconv.Atoi(r.URL.Query().Get("index"))
		var req convertRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20))
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
		clipOpt := optionsForClip(opt, opt.Inputs[idx])
		if err := generateAudioPreview(clipOpt, infos[idx], opt.Inputs[idx].InputPath, out); err != nil {
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
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20))
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
	versioned := strings.Replace(page, "__APP_VERSION__", html.EscapeString(appVersion), 1)
	if versioned == page {
		return nil, errors.New("application version placeholder is missing")
	}
	language := loadLanguagePreference()
	localized := strings.Replace(versioned, "__APP_LANGUAGE__", html.EscapeString(language), 1)
	if localized == versioned {
		return nil, errors.New("application language placeholder is missing")
	}
	flagMode := "emoji"
	if runtime.GOOS == "windows" {
		flagMode = "svg"
	}
	flagged := strings.Replace(localized, "__APP_FLAG_MODE__", flagMode, 1)
	if flagged == localized {
		return nil, errors.New("application flag mode placeholder is missing")
	}
	return []byte(flagged), nil
}

func runWebApp(launch func(string) error) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	sessionDir, err := os.MkdirTemp("", "gba-media-maker-")
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
