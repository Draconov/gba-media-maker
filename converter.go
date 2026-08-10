package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	romLimit               = 32 * 1024 * 1024
	romMinSize             = 1 * 1024 * 1024
	metadataOffset         = 0x7FC0
	assetOffset            = 0x8000
	clipDescriptorSize     = 96
	frameWidth             = 120
	frameHeight            = 80
	frameBytes             = frameWidth * frameHeight
	audioRate              = 16384
	videoPaletteColors     = 250
	nativeImageWidth       = 240
	nativeImageHeight      = 160
	nativeImageBytes       = nativeImageWidth * nativeImageHeight * 2
	gbaRefresh             = 59.727500569606
	defaultLongSplitBudget = 31 * 1024 * 1024
)

//go:embed assets/player_stub.bin
var playerStub []byte

var nintendoLogo = []byte{
	0x24, 0xFF, 0xAE, 0x51, 0x69, 0x9A, 0xA2, 0x21, 0x3D, 0x84, 0x82, 0x0A, 0x84, 0xE4, 0x09, 0xAD,
	0x11, 0x24, 0x8B, 0x98, 0xC0, 0x81, 0x7F, 0x21, 0xA3, 0x52, 0xBE, 0x19, 0x93, 0x09, 0xCE, 0x20,
	0x10, 0x46, 0x4A, 0x4A, 0xF8, 0x27, 0x31, 0xEC, 0x58, 0xC7, 0xE8, 0x33, 0x82, 0xE3, 0xCE, 0xBF,
	0x85, 0xF4, 0xDF, 0x94, 0xCE, 0x4B, 0x09, 0xC1, 0x94, 0x56, 0x8A, 0xC0, 0x13, 0x72, 0xA7, 0xFC,
	0x9F, 0x84, 0x4D, 0x73, 0xA3, 0xCA, 0x9A, 0x61, 0x58, 0x97, 0xA3, 0x27, 0xFC, 0x03, 0x98, 0x76,
	0x23, 0x1D, 0xC7, 0x61, 0x03, 0x04, 0xAE, 0x56, 0xBF, 0x38, 0x84, 0x00, 0x40, 0xA7, 0x0E, 0xFD,
	0xFF, 0x52, 0xFE, 0x03, 0x6F, 0x95, 0x30, 0xF1, 0x97, 0xFB, 0xC0, 0x85, 0x60, 0xD6, 0x80, 0x25,
	0xA9, 0x63, 0xBE, 0x03, 0x01, 0x4E, 0x38, 0xE2, 0xF9, 0xA2, 0x34, 0xFF, 0xBB, 0x3E, 0x03, 0x44,
	0x78, 0x00, 0x90, 0xCB, 0x88, 0x11, 0x3A, 0x94, 0x65, 0xC0, 0x7C, 0x63, 0x87, 0xF0, 0x3C, 0xAF,
	0xD6, 0x25, 0xE4, 0x8B, 0x38, 0x0A, 0xAC, 0x72, 0x21, 0xD4, 0xF8, 0x07,
}

type AudioTrackInfo struct {
	Index       int    `json:"index"`
	StreamIndex int    `json:"streamIndex,omitempty"`
	Language    string `json:"language,omitempty"`
	Title       string `json:"title,omitempty"`
	Codec       string `json:"codec,omitempty"`
	Channels    int    `json:"channels,omitempty"`
	Default     bool   `json:"default,omitempty"`
}

type MediaInfo struct {
	Kind          string           `json:"kind"` // video, audio, image
	Duration      float64          `json:"duration"`
	Width         int              `json:"width"`
	Height        int              `json:"height"`
	FPS           float64          `json:"fps"`
	Title         string           `json:"title,omitempty"`
	Artist        string           `json:"artist,omitempty"`
	Album         string           `json:"album,omitempty"`
	AudioStreams  int              `json:"audioStreams"`
	AudioChannels int              `json:"audioChannels"`
	AudioTracks   []AudioTrackInfo `json:"audioTracks,omitempty"`
	Chapters      []float64        `json:"chapters,omitempty"`
}

type ClipInput struct {
	InputPath string
	Name      string
	Title     string

	// Custom selects the per-clip overrides below. When false, the project
	// defaults in ProjectOptions are used.
	Custom       bool
	Start        float64
	End          float64
	Speed        float64
	FitMode      string
	AudioMode    string
	AudioTrack   int
	Volume       float64
	Loop         bool
	PaletteMode  string
	DitherMode   string
	MediaKind    string
	ImageSeconds float64
}

type ProjectOptions struct {
	Inputs                 []ClipInput
	OutputPath             string
	FFmpegPath             string
	Start                  float64
	End                    float64
	Speed                  float64
	VBlanks                int
	FitMode                string
	AudioMode              string
	AudioTrack             int
	Volume                 float64
	Loop                   bool
	RomTitle               string
	SeekSeconds            int
	Normalize              bool
	Limiter                bool
	Resume                 bool
	Compression            string // none, delta
	PaletteMode            string // shared, scene
	DitherMode             string // off, ordered, error
	OutputMode             string // rom, playlist, menu, batch; longsplit is internal
	KeyInterval            int
	SplitBudgetMiB         int
	MaxPartMinutes         float64
	ChapterAware           bool
	PartTitleScreens       bool // legacy title-card toggle kept for project compatibility
	ResumeLongSplit        bool
	TitleScreenPart        int    // legacy metadata fallback
	TitleScreenName        string // legacy metadata fallback
	TitleCards             *TitleCardProjectSettings
	TitleCard              *TitleCardSettings // resolved settings for one generated ROM part
	TitleCardAsset         []byte             // prepared native 240×160 Mode 3 image and timing header
	MenuTheme              *MenuThemeOptions
	Preset                 string // best, balanced, long, small, extreme, custom
	AudioCodec             string // pcm, adpcm, auto
	ExtremeOptimization    bool
	AdaptiveKeyframes      bool
	EnhancedSceneDetection bool
	SmartTargetMiB         int
	SmartPriority          string
}

// ConvertOptions keeps the single-video command-line/test API convenient.
type ConvertOptions struct {
	InputPath              string
	OutputPath             string
	FFmpegPath             string
	Start                  float64
	End                    float64
	Speed                  float64
	VBlanks                int
	FitMode                string
	AudioMode              string
	AudioTrack             int
	Volume                 float64
	Loop                   bool
	RomTitle               string
	SeekSeconds            int
	Normalize              bool
	Limiter                bool
	Resume                 bool
	Compression            string
	PaletteMode            string
	DitherMode             string
	KeyInterval            int
	SplitBudgetMiB         int
	MaxPartMinutes         float64
	ChapterAware           bool
	PartTitleScreens       bool
	ResumeLongSplit        bool
	TitleScreenPart        int
	TitleScreenName        string
	TitleCards             *TitleCardProjectSettings
	Preset                 string
	AudioCodec             string
	ExtremeOptimization    bool
	AdaptiveKeyframes      bool
	EnhancedSceneDetection bool
	SmartTargetMiB         int
	SmartPriority          string
}

type ConvertResult struct {
	OutputPath        string  `json:"outputPath"`
	FrameCount        int     `json:"frameCount"`
	FPS               float64 `json:"fps"`
	UnpaddedSize      int64   `json:"unpaddedSize"`
	PaddedSize        int64   `json:"paddedSize"`
	ClipCount         int     `json:"clipCount"`
	CompressedBytes   int64   `json:"compressedBytes"`
	UncompressedBytes int64   `json:"uncompressedBytes"`
	OutputKind        string  `json:"outputKind"`
	AutoSplit         bool    `json:"autoSplit,omitempty"`
	EstimatedParts    int     `json:"estimatedParts,omitempty"`
}

type ProgressFunc func(percent int, status string)

var (
	durationPattern      = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)
	dimensionsPattern    = regexp.MustCompile(`(?:^|[^0-9])(\d{2,5})x(\d{2,5})(?:[^0-9]|$)`)
	fpsPattern           = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s+fps`)
	channelsPattern      = regexp.MustCompile(`\b(\d+)\.(\d+)\b`)
	chapterPattern       = regexp.MustCompile(`Chapter #[^:]+:[^:]+: start ([0-9]+(?:\.[0-9]+)?), end ([0-9]+(?:\.[0-9]+)?)`)
	streamAudioPattern   = regexp.MustCompile(`Stream #\d+:(\d+)(?:\[[^\]]+\])?(?:\(([^)]+)\))?: Audio:\s*([^,]+)(.*)$`)
	streamAnyPattern     = regexp.MustCompile(`Stream #\d+:\d+`)
	metadataTitlePattern = regexp.MustCompile(`^\s*title\s*:\s*(.+?)\s*$`)
)

func parseAudioChannels(line string) int {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "mono") {
		return 1
	}
	if strings.Contains(lower, "stereo") {
		return 2
	}
	if match := channelsPattern.FindStringSubmatch(lower); match != nil {
		a, _ := strconv.Atoi(match[1])
		b, _ := strconv.Atoi(match[2])
		return a + b
	}
	return 0
}

func isStillImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp", ".tga", ".tif", ".tiff":
		return true
	}
	return false
}

func isAudioFirstPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".flac", ".wav", ".ogg", ".opus", ".m4a", ".aac", ".wma", ".aiff", ".ape":
		return true
	}
	return false
}

func parseMediaInfoForPath(text, path string) (MediaInfo, error) {
	duration := 0.0
	if dm := durationPattern.FindStringSubmatch(text); dm != nil {
		h, _ := strconv.Atoi(dm[1])
		m, _ := strconv.Atoi(dm[2])
		seconds, _ := strconv.ParseFloat(dm[3], 64)
		duration = float64(h*3600+m*60) + seconds
	}
	var videoLine string
	var audioTracks []AudioTrackInfo
	currentAudio := -1
	beforeStreams := true
	var metaTitle, metaArtist, metaAlbum string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, " Video:") && videoLine == "" {
			videoLine = line
		}
		if strings.HasPrefix(trimmed, "Stream #") {
			beforeStreams = false
		}
		if beforeStreams {
			lower := strings.ToLower(trimmed)
			if i := strings.Index(lower, ":"); i > 0 {
				key, val := strings.TrimSpace(lower[:i]), strings.TrimSpace(trimmed[i+1:])
				switch key {
				case "title":
					metaTitle = val
				case "artist":
					metaArtist = val
				case "album":
					metaAlbum = val
				}
			}
		}
		if match := streamAudioPattern.FindStringSubmatch(trimmed); match != nil {
			streamIndex, _ := strconv.Atoi(match[1])
			track := AudioTrackInfo{Index: len(audioTracks), StreamIndex: streamIndex, Language: strings.TrimSpace(match[2]), Codec: strings.TrimSpace(match[3]), Channels: parseAudioChannels(match[4]), Default: strings.Contains(strings.ToLower(match[4]), "(default)")}
			audioTracks = append(audioTracks, track)
			currentAudio = len(audioTracks) - 1
			continue
		}
		if streamAnyPattern.MatchString(trimmed) {
			currentAudio = -1
			continue
		}
		if currentAudio >= 0 {
			if match := metadataTitlePattern.FindStringSubmatch(line); match != nil {
				audioTracks[currentAudio].Title = strings.TrimSpace(match[1])
			}
		}
	}
	width, height, fps := 0, 0, 0.0
	if videoLine != "" {
		if dims := dimensionsPattern.FindStringSubmatch(videoLine); dims != nil {
			width, _ = strconv.Atoi(dims[1])
			height, _ = strconv.Atoi(dims[2])
		}
		if fm := fpsPattern.FindStringSubmatch(videoLine); fm != nil {
			fps, _ = strconv.ParseFloat(fm[1], 64)
		}
	}
	kind := "video"
	if isAudioFirstPath(path) && len(audioTracks) > 0 {
		kind = "audio"
	} else if isStillImagePath(path) {
		kind = "image"
	} else if videoLine == "" && len(audioTracks) > 0 {
		kind = "audio"
	} else if videoLine == "" {
		return MediaInfo{}, errors.New("could not find a supported video, audio, or image stream")
	}
	if kind == "video" && duration <= 0 {
		return MediaInfo{}, errors.New("could not read video duration")
	}
	if kind == "audio" && duration <= 0 {
		return MediaInfo{}, errors.New("could not read audio duration")
	}
	if kind == "image" {
		duration = 0
		fps = 0
	}
	var chapters []float64
	for _, match := range chapterPattern.FindAllStringSubmatch(text, -1) {
		if v, e := strconv.ParseFloat(match[1], 64); e == nil && v > 0 {
			chapters = append(chapters, v)
		}
	}
	sort.Float64s(chapters)
	audioChannels := 0
	if len(audioTracks) > 0 {
		audioChannels = audioTracks[0].Channels
	}
	return MediaInfo{Kind: kind, Duration: duration, Width: width, Height: height, FPS: fps, Title: metaTitle, Artist: metaArtist, Album: metaAlbum, AudioStreams: len(audioTracks), AudioChannels: audioChannels, AudioTracks: audioTracks, Chapters: chapters}, nil
}

// parseMediaInfo keeps the legacy parser contract used by tests and helpers.
func parseMediaInfo(text string) (MediaInfo, error) { return parseMediaInfoForPath(text, "video.mp4") }

func inspectMedia(ffmpegPath, path string) (MediaInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, _ := runCommandContext(ctx, ffmpegPath, "-hide_banner", "-i", path)
	return parseMediaInfoForPath(string(output), path)
}

func parseTime(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("time is empty")
	}
	parts := strings.Split(value, ":")
	nums := make([]float64, len(parts))
	for i, p := range parts {
		n, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid time: %s", value)
		}
		nums[i] = n
	}
	var result float64
	switch len(nums) {
	case 1:
		result = nums[0]
	case 2:
		result = nums[0]*60 + nums[1]
	case 3:
		result = nums[0]*3600 + nums[1]*60 + nums[2]
	default:
		return 0, fmt.Errorf("invalid time: %s", value)
	}
	if result < 0 {
		return 0, errors.New("time cannot be negative")
	}
	return result, nil
}

func nextPowerOfTwo(v int64) int64 {
	p := int64(1 << 20)
	for p < v {
		p <<= 1
	}
	return p
}

func makeVideoFilter(fitMode string, speed, fps float64) string {
	var scale string
	switch fitMode {
	case "crop":
		scale = fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", frameWidth, frameHeight, frameWidth, frameHeight)
	case "stretch":
		scale = fmt.Sprintf("scale=%d:%d", frameWidth, frameHeight)
	default:
		scale = fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black", frameWidth, frameHeight, frameWidth, frameHeight)
	}
	return fmt.Sprintf("setpts=PTS/%.8f,%s,fps=%.10f,format=rgb24", speed, scale, fps)
}

func makePreviewFilter(fitMode string) string {
	var scale string
	switch fitMode {
	case "crop":
		scale = "scale=240:160:force_original_aspect_ratio=increase,crop=240:160"
	case "stretch":
		scale = "scale=240:160"
	default:
		scale = "scale=240:160:force_original_aspect_ratio=decrease,pad=240:160:(ow-iw)/2:(oh-ih)/2:black"
	}
	return scale + ",format=rgb24"
}

func buildAtempo(speed float64) []float64 {
	var factors []float64
	remaining := speed
	for remaining > 2.000000001 {
		factors = append(factors, 2)
		remaining /= 2
	}
	for remaining < 0.499999999 {
		factors = append(factors, .5)
		remaining /= .5
	}
	return append(factors, remaining)
}

func selectedAudioTrack(info MediaInfo, index int) (AudioTrackInfo, bool) {
	if index < 0 || index >= len(info.AudioTracks) {
		return AudioTrackInfo{}, false
	}
	return info.AudioTracks[index], true
}

func selectedAudioChannels(info MediaInfo, index int) int {
	if track, ok := selectedAudioTrack(info, index); ok && track.Channels > 0 {
		return track.Channels
	}
	return info.AudioChannels
}

func audioMapSpecifier(index int) string {
	if index < 0 {
		index = 0
	}
	return fmt.Sprintf("0:a:%d", index)
}

func audioFilters(opt ProjectOptions, info MediaInfo) []string {
	var filters []string
	switch opt.AudioMode {
	case "left":
		filters = append(filters, "pan=mono|c0=c0")
	case "right":
		if selectedAudioChannels(info, opt.AudioTrack) <= 1 {
			filters = append(filters, "pan=mono|c0=c0")
		} else {
			filters = append(filters, "pan=mono|c0=c1")
		}
	}
	// Anchor output to timestamp zero and compensate small timestamp gaps or
	// overlaps before speed and volume processing.
	filters = append(filters, "aresample=16384:async=1:first_pts=0")
	for _, f := range buildAtempo(opt.Speed) {
		filters = append(filters, fmt.Sprintf("atempo=%.8f", f))
	}
	if math.Abs(opt.Volume-1) > 0.000001 {
		filters = append(filters, fmt.Sprintf("volume=%.6f", opt.Volume))
	}
	if opt.Normalize {
		filters = append(filters, "loudnorm=I=-16:LRA=11:TP=-1.5")
	}
	if opt.Limiter {
		filters = append(filters, "alimiter=limit=0.95:attack=5:release=50")
	}
	return filters
}

func ffmpegVideoError(prefix string, output []byte) error {
	detail := strings.TrimSpace(string(output))
	lower := strings.ToLower(detail)
	if strings.Contains(lower, "hardware accelerated av1 decoding") ||
		(strings.Contains(lower, "decoder: function not implemented") && strings.Contains(lower, "/av1")) {
		return fmt.Errorf("%s:\nThis FFmpeg build cannot decode AV1 in software. Use an FFmpeg build with libdav1d or libaom AV1 decoding support.\n\nFFmpeg details:\n%s", prefix, detail)
	}
	return fmt.Errorf("%s:\n%s", prefix, detail)
}

func extractFrames(opt ProjectOptions, input string, duration float64, path string, progress ProgressFunc) error {
	fps := gbaRefresh / float64(opt.VBlanks)
	vf := makeVideoFilter(opt.FitMode, opt.Speed, fps)
	output, err := runCommand(opt.FFmpegPath, "-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", opt.Start), "-i", input, "-t", fmt.Sprintf("%.6f", duration), "-an", "-vf", vf, "-pix_fmt", "rgb24", "-f", "rawvideo", path)
	if err != nil {
		return ffmpegVideoError("FFmpeg could not convert the video", output)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() < frameBytes*3 {
		return errors.New("converted video contains no usable frames")
	}
	return nil
}

func rgb24ToRGB555(src []byte) ([]byte, error) {
	if len(src) != nativeImageWidth*nativeImageHeight*3 {
		return nil, fmt.Errorf("native image has %d bytes, expected %d", len(src), nativeImageWidth*nativeImageHeight*3)
	}
	out := make([]byte, nativeImageBytes)
	for i, j := 0, 0; i < len(src); i, j = i+3, j+2 {
		r := uint16(src[i] >> 3)
		g := uint16(src[i+1] >> 3)
		b := uint16(src[i+2] >> 3)
		v := r | (g << 5) | (b << 10)
		out[j] = byte(v)
		out[j+1] = byte(v >> 8)
	}
	return out, nil
}

func nativeImageFilter(fitMode string) string {
	switch fitMode {
	case "crop":
		return "scale=240:160:force_original_aspect_ratio=increase,crop=240:160,format=rgb24"
	case "stretch":
		return "scale=240:160,format=rgb24"
	default:
		return "scale=240:160:force_original_aspect_ratio=decrease,pad=240:160:(ow-iw)/2:(oh-ih)/2:black,format=rgb24"
	}
}

func extractNativeImage(ffmpegPath, input, fitMode, path string) error {
	rawPath := path + ".rgb24"
	defer os.Remove(rawPath)
	output, err := runCommand(ffmpegPath, "-y", "-hide_banner", "-loglevel", "error", "-i", input, "-frames:v", "1", "-an", "-vf", nativeImageFilter(fitMode), "-pix_fmt", "rgb24", "-f", "rawvideo", rawPath)
	if err != nil {
		return ffmpegVideoError("FFmpeg could not convert the image", output)
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		return err
	}
	data, err := rgb24ToRGB555(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func fallbackAudioArtwork(path string) error {
	data := make([]byte, nativeImageBytes)
	for y := 0; y < nativeImageHeight; y++ {
		for x := 0; x < nativeImageWidth; x++ {
			// Dark blue/black GBA-safe background with a subtle centered panel.
			r, g, b := uint16(2), uint16(4), uint16(8)
			if x > 28 && x < 212 && y > 18 && y < 142 {
				r, g, b = 3, 7, 13
			}
			if x > 56 && x < 184 && y > 42 && y < 118 {
				r, g, b = 5, 10, 18
			}
			v := r | (g << 5) | (b << 10)
			i := (y*nativeImageWidth + x) * 2
			data[i] = byte(v)
			data[i+1] = byte(v >> 8)
		}
	}
	return os.WriteFile(path, data, 0644)
}

func extractAudioArtwork(ffmpegPath, input, path string) error {
	if err := extractNativeImage(ffmpegPath, input, "fit", path); err == nil {
		return nil
	}
	return fallbackAudioArtwork(path)
}

type colorPoint struct {
	index   int
	count   uint64
	r, g, b int
}
type colorBox struct {
	points                             []colorPoint
	total                              uint64
	minR, maxR, minG, maxG, minB, maxB int
}
type rgb5 struct{ r, g, b int }

func newColorBox(points []colorPoint) colorBox {
	b := colorBox{points: points, minR: 31, minG: 31, minB: 31}
	for _, p := range points {
		b.total += p.count
		if p.r < b.minR {
			b.minR = p.r
		}
		if p.r > b.maxR {
			b.maxR = p.r
		}
		if p.g < b.minG {
			b.minG = p.g
		}
		if p.g > b.maxG {
			b.maxG = p.g
		}
		if p.b < b.minB {
			b.minB = p.b
		}
		if p.b > b.maxB {
			b.maxB = p.b
		}
	}
	return b
}
func (b colorBox) score() uint64 {
	r := b.maxR - b.minR
	g := b.maxG - b.minG
	bl := b.maxB - b.minB
	rng := r
	if g > rng {
		rng = g
	}
	if bl > rng {
		rng = bl
	}
	return uint64(rng+1) * b.total
}
func splitColorBox(b colorBox) (colorBox, colorBox, bool) {
	if len(b.points) < 2 {
		return b, colorBox{}, false
	}
	ch := 0
	rr := b.maxR - b.minR
	gr := b.maxG - b.minG
	br := b.maxB - b.minB
	if gr > rr && gr >= br {
		ch = 1
	} else if br > rr && br > gr {
		ch = 2
	}
	pts := append([]colorPoint(nil), b.points...)
	sort.Slice(pts, func(i, j int) bool {
		switch ch {
		case 1:
			if pts[i].g != pts[j].g {
				return pts[i].g < pts[j].g
			}
		case 2:
			if pts[i].b != pts[j].b {
				return pts[i].b < pts[j].b
			}
		default:
			if pts[i].r != pts[j].r {
				return pts[i].r < pts[j].r
			}
		}
		return pts[i].index < pts[j].index
	})
	half := b.total / 2
	var acc uint64
	split := 1
	for i, p := range pts {
		acc += p.count
		if acc >= half && i+1 < len(pts) {
			split = i + 1
			break
		}
	}
	if split <= 0 || split >= len(pts) {
		split = len(pts) / 2
	}
	return newColorBox(pts[:split]), newColorBox(pts[split:]), true
}

func quantizePalette(hist []uint64) []rgb5 {
	points := make([]colorPoint, 0, 32768)
	for idx, count := range hist {
		if count > 0 {
			points = append(points, colorPoint{idx, count, idx & 31, (idx >> 5) & 31, (idx >> 10) & 31})
		}
	}
	if len(points) == 0 {
		points = append(points, colorPoint{0, 1, 0, 0, 0})
	}
	boxes := []colorBox{newColorBox(points)}
	for len(boxes) < videoPaletteColors {
		best := -1
		var score uint64
		for i, b := range boxes {
			if len(b.points) < 2 {
				continue
			}
			if s := b.score(); best < 0 || s > score {
				best = i
				score = s
			}
		}
		if best < 0 {
			break
		}
		a, b, ok := splitColorBox(boxes[best])
		if !ok {
			break
		}
		boxes[best] = a
		boxes = append(boxes, b)
	}
	palette := make([]rgb5, 256)
	for i, b := range boxes {
		var rs, gs, bs, total uint64
		for _, p := range b.points {
			rs += uint64(p.r) * p.count
			gs += uint64(p.g) * p.count
			bs += uint64(p.b) * p.count
			total += p.count
		}
		if total > 0 {
			palette[i] = rgb5{int((rs + total/2) / total), int((gs + total/2) / total), int((bs + total/2) / total)}
		}
	}
	palette[250] = rgb5{0, 0, 0}
	palette[251] = rgb5{6, 6, 6}
	palette[252] = rgb5{31, 31, 31}
	palette[253] = rgb5{31, 27, 0}
	palette[254] = rgb5{31, 0, 0}
	palette[255] = rgb5{0, 31, 0}
	return palette
}

func paletteLookup(palette []rgb5) []byte {
	lookup := make([]byte, 32768)
	workers := runtime.GOMAXPROCS(0)
	if workers > 16 {
		workers = 16
	}
	if workers < 1 {
		workers = 1
	}
	chunk := (len(lookup) + workers - 1) / workers
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		start := worker * chunk
		end := start + chunk
		if end > len(lookup) {
			end = len(lookup)
		}
		if start >= end {
			break
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for idx := start; idx < end; idx++ {
				r, g, b := idx&31, (idx>>5)&31, (idx>>10)&31
				best, dist := 0, math.MaxInt
				for j, p := range palette[:videoPaletteColors] {
					dr, dg, db := r-p.r, g-p.g, b-p.b
					d := dr*dr + dg*dg + db*db
					if d < dist {
						best, dist = j, d
						if d == 0 {
							break
						}
					}
				}
				lookup[idx] = byte(best)
			}
		}(start, end)
	}
	wg.Wait()
	return lookup
}
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func quantizeFrame(src, dst []byte, palette []rgb5, lookup []byte, mode string, cur, next []int) {
	if mode == "error" {
		clear(cur)
		clear(next)
		for y := 0; y < frameHeight; y++ {
			clear(next)
			for x := 0; x < frameWidth; x++ {
				i := (y*frameWidth + x) * 3
				e := (x + 1) * 3
				r := clamp(int(src[i])+cur[e]/16, 0, 255)
				g := clamp(int(src[i+1])+cur[e+1]/16, 0, 255)
				b := clamp(int(src[i+2])+cur[e+2]/16, 0, 255)
				r5 := (r*31 + 127) / 255
				g5 := (g*31 + 127) / 255
				b5 := (b*31 + 127) / 255
				idx := lookup[r5|(g5<<5)|(b5<<10)]
				dst[y*frameWidth+x] = idx
				p := palette[idx]
				er := r - p.r*255/31
				eg := g - p.g*255/31
				eb := b - p.b*255/31
				cur[e+3] += er * 7
				cur[e+4] += eg * 7
				cur[e+5] += eb * 7
				next[e-3] += er * 3
				next[e-2] += eg * 3
				next[e-1] += eb * 3
				next[e] += er * 5
				next[e+1] += eg * 5
				next[e+2] += eb * 5
				next[e+3] += er
				next[e+4] += eg
				next[e+5] += eb
			}
			cur, next = next, cur
		}
		return
	}
	bayer := [16]int{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}
	for y := 0; y < frameHeight; y++ {
		for x := 0; x < frameWidth; x++ {
			i := (y*frameWidth + x) * 3
			r := (int(src[i])*31 + 127) / 255
			g := (int(src[i+1])*31 + 127) / 255
			b := (int(src[i+2])*31 + 127) / 255
			if mode == "ordered" {
				d := (bayer[(y&3)*4+(x&3)] - 7) / 4
				r = clamp(r+d, 0, 31)
				g = clamp(g+d, 0, 31)
				b = clamp(b+d, 0, 31)
			}
			dst[y*frameWidth+x] = lookup[r|(g<<5)|(b<<10)]
		}
	}
}

func detectSceneStartsLegacy(framesPath string, frameCount int, rgbFrameBytes int64) ([]int, error) {
	if frameCount <= 1 {
		return []int{0}, nil
	}
	f, err := os.Open(framesPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, rgbFrameBytes)
	sigLen := ((frameHeight-1-4)/8 + 1) * ((frameWidth-1-4)/8 + 1) * 3
	sig := make([]byte, sigLen)
	previous := make([]byte, sigLen)
	havePrevious := false
	starts := []int{0}
	lastStart := 0
	for frame := 0; frame < frameCount; frame++ {
		if _, err := io.ReadFull(f, buf); err != nil {
			return nil, err
		}
		pos := 0
		for y := 4; y < frameHeight; y += 8 {
			for x := 4; x < frameWidth; x += 8 {
				i := (y*frameWidth + x) * 3
				sig[pos], sig[pos+1], sig[pos+2] = buf[i], buf[i+1], buf[i+2]
				pos += 3
			}
		}
		if havePrevious {
			diff := 0
			for i := range sig {
				d := int(sig[i]) - int(previous[i])
				if d < 0 {
					d = -d
				}
				diff += d
			}
			avg := diff / len(sig)
			if (frame-lastStart >= 10 && avg >= 42) || frame-lastStart >= 120 {
				starts = append(starts, frame)
				lastStart = frame
			}
		}
		copy(previous, sig)
		havePrevious = true
	}
	return starts, nil
}

func buildPalettesAndRawVideo(framesPath, palettePath, paletteIndexPath, videoPath, paletteMode, ditherMode string, enhancedSceneDetection bool, progress ProgressFunc) (int, int, []int, error) {
	st, err := os.Stat(framesPath)
	if err != nil {
		return 0, 0, nil, err
	}
	rgbFrameBytes := int64(frameBytes * 3)
	if st.Size()%rgbFrameBytes != 0 {
		return 0, 0, nil, errors.New("FFmpeg produced an incomplete frame stream")
	}
	frameCount := int(st.Size() / rgbFrameBytes)
	if frameCount < 1 {
		return 0, 0, nil, errors.New("no frames were produced")
	}

	sceneStarts := []int{0}
	if paletteMode == "scene" {
		progress(18, "Detecting scene changes…")
		if enhancedSceneDetection {
			sceneStarts, err = detectSceneStartsEnhanced(framesPath, frameCount)
		} else {
			sceneStarts, err = detectSceneStartsLegacy(framesPath, frameCount, rgbFrameBytes)
		}
		if err != nil {
			return 0, 0, nil, err
		}
	}
	sceneCount := len(sceneStarts)
	frameScene := make([]int, frameCount)
	for scene, start := range sceneStarts {
		end := frameCount
		if scene+1 < sceneCount {
			end = sceneStarts[scene+1]
		}
		for frame := start; frame < end; frame++ {
			frameScene[frame] = scene
		}
	}

	f, err := os.Open(framesPath)
	if err != nil {
		return 0, 0, nil, err
	}
	defer f.Close()
	frameBuf := make([]byte, rgbFrameBytes)
	palettes := make([][]rgb5, sceneCount)
	lookups := make([][]byte, sceneCount)
	for scene, start := range sceneStarts {
		progress(20+int(20*float64(scene)/float64(sceneCount)), fmt.Sprintf("Building palette %d of %d…", scene+1, sceneCount))
		end := frameCount
		if scene+1 < sceneCount {
			end = sceneStarts[scene+1]
		}
		samples := end - start
		if samples > 60 {
			samples = 60
		}
		hist := make([]uint64, 32768)
		for n := 0; n < samples; n++ {
			idx := start
			if samples > 1 {
				idx = start + int(math.Round(float64(n)*float64(end-start-1)/float64(samples-1)))
			}
			if _, err := f.Seek(int64(idx)*rgbFrameBytes, io.SeekStart); err != nil {
				return 0, 0, nil, err
			}
			if _, err := io.ReadFull(f, frameBuf); err != nil {
				return 0, 0, nil, err
			}
			for i := 0; i < len(frameBuf); i += 3 {
				r := (int(frameBuf[i])*31 + 127) / 255
				g := (int(frameBuf[i+1])*31 + 127) / 255
				b := (int(frameBuf[i+2])*31 + 127) / 255
				hist[r|(g<<5)|(b<<10)]++
			}
		}
		palettes[scene] = quantizePalette(hist)
		lookups[scene] = paletteLookup(palettes[scene])
	}

	paletteBytes := make([]byte, sceneCount*256*2)
	palettePos := 0
	for _, pal := range palettes {
		for _, colour := range pal {
			binary.LittleEndian.PutUint16(paletteBytes[palettePos:palettePos+2], uint16(colour.r|(colour.g<<5)|(colour.b<<10)))
			palettePos += 2
		}
	}
	if err := os.WriteFile(palettePath, paletteBytes, 0644); err != nil {
		return 0, 0, nil, err
	}

	if sceneCount > 1 {
		paletteIndexes := make([]byte, frameCount*2)
		for frame, scene := range frameScene {
			binary.LittleEndian.PutUint16(paletteIndexes[frame*2:frame*2+2], uint16(scene))
		}
		if err := os.WriteFile(paletteIndexPath, paletteIndexes, 0644); err != nil {
			return 0, 0, nil, err
		}
	} else if err := os.WriteFile(paletteIndexPath, nil, 0644); err != nil {
		return 0, 0, nil, err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, 0, nil, err
	}
	out, err := os.Create(videoPath)
	if err != nil {
		return 0, 0, nil, err
	}
	defer out.Close()
	reader := bufio.NewReaderSize(f, int(rgbFrameBytes)*2)
	writer := bufio.NewWriterSize(out, frameBytes*64)
	workers := runtime.GOMAXPROCS(0)
	if workers > 16 {
		workers = 16
	}
	if workers > frameCount {
		workers = frameCount
	}
	if workers < 1 {
		workers = 1
	}
	rgbBuffers := make([][]byte, workers)
	indexBuffers := make([][]byte, workers)
	errorCur := make([][]int, workers)
	errorNext := make([][]int, workers)
	for slot := 0; slot < workers; slot++ {
		rgbBuffers[slot] = make([]byte, rgbFrameBytes)
		indexBuffers[slot] = make([]byte, frameBytes)
		errorCur[slot] = make([]int, (frameWidth+2)*3)
		errorNext[slot] = make([]int, (frameWidth+2)*3)
	}
	for base := 0; base < frameCount; base += workers {
		count := workers
		if remaining := frameCount - base; remaining < count {
			count = remaining
		}
		for slot := 0; slot < count; slot++ {
			if _, err := io.ReadFull(reader, rgbBuffers[slot]); err != nil {
				return 0, 0, nil, err
			}
		}
		var quantizeWG sync.WaitGroup
		quantizeWG.Add(count)
		for slot := 0; slot < count; slot++ {
			frame := base + slot
			scene := frameScene[frame]
			go func(slot, scene int) {
				defer quantizeWG.Done()
				quantizeFrame(rgbBuffers[slot], indexBuffers[slot], palettes[scene], lookups[scene], ditherMode, errorCur[slot], errorNext[slot])
			}(slot, scene)
		}
		quantizeWG.Wait()
		for slot := 0; slot < count; slot++ {
			frame := base + slot
			if _, err := writer.Write(indexBuffers[slot]); err != nil {
				return 0, 0, nil, err
			}
			if frame%20 == 0 || frame+1 == frameCount {
				progress(40+int(25*float64(frame+1)/float64(frameCount)), fmt.Sprintf("Converting frame %d of %d…", frame+1, frameCount))
			}
		}
	}
	if err := writer.Flush(); err != nil {
		return 0, 0, nil, err
	}
	return frameCount, sceneCount, sceneStarts, nil
}

func writeRecord(w io.Writer, typ uint32, payload []byte) (int, error) {
	var header [8]byte
	binary.LittleEndian.PutUint32(header[0:4], typ)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return 0, err
	}
	if _, err := w.Write(payload); err != nil {
		return 8, err
	}
	total := 8 + len(payload)
	padding := (-total) & 3
	if padding != 0 {
		var zeros [3]byte
		if _, err := w.Write(zeros[:padding]); err != nil {
			return total, err
		}
		total += padding
	}
	return total, nil
}

func appendUint16LE(dst []byte, value int) []byte {
	return append(dst, byte(value), byte(value>>8))
}

func encodeDelta(prev, curr []byte) []byte {
	out := make([]byte, 0, len(curr)/2)
	pos := 0
	for pos < len(curr) {
		skip := 0
		for pos+skip < len(curr) && prev[pos+skip] == curr[pos+skip] && skip < 65535 {
			skip++
		}
		pos += skip
		if pos >= len(curr) {
			out = appendUint16LE(out, skip)
			out = appendUint16LE(out, 0)
			break
		}
		start := pos
		unchanged := 0
		for pos < len(curr) && pos-start < 65535 {
			if prev[pos] == curr[pos] {
				unchanged++
			} else {
				unchanged = 0
			}
			pos++
			if unchanged >= 4 {
				pos -= unchanged
				break
			}
		}
		run := pos - start
		out = appendUint16LE(out, skip)
		out = appendUint16LE(out, run)
		out = append(out, curr[start:pos]...)
	}
	return out
}

func compressRawVideo(rawPath, streamPath, indexPath, mode string, keyInterval int, adaptive bool, sceneStarts []int) (int64, int64, error) {
	st, err := os.Stat(rawPath)
	if err != nil {
		return 0, 0, err
	}
	rawSize := st.Size()
	if mode != "delta" {
		if err := copyFile(rawPath, streamPath); err != nil {
			return 0, 0, err
		}
		if err := os.WriteFile(indexPath, nil, 0644); err != nil {
			return 0, 0, err
		}
		return rawSize, rawSize, nil
	}
	if keyInterval < 1 {
		keyInterval = 30
	}
	in, err := os.Open(rawPath)
	if err != nil {
		return 0, 0, err
	}
	defer in.Close()
	out, err := os.Create(streamPath)
	if err != nil {
		return 0, 0, err
	}
	defer out.Close()
	idxFile, err := os.Create(indexPath)
	if err != nil {
		return 0, 0, err
	}
	defer idxFile.Close()
	idxWriter := bufio.NewWriterSize(idxFile, 64*1024)
	frames := int(rawSize / frameBytes)
	prev := make([]byte, frameBytes)
	cur := make([]byte, frameBytes)
	sceneSet := make(map[int]bool, len(sceneStarts))
	for _, frame := range sceneStarts {
		if frame > 0 {
			sceneSet[frame] = true
		}
	}
	var offset uint32
	lastKey := -keyInterval
	changeBudget := 0
	minAdaptiveInterval := 8
	maxAdaptiveInterval := keyInterval * 3
	if maxAdaptiveInterval < 60 {
		maxAdaptiveInterval = 60
	}
	if maxAdaptiveInterval > 150 {
		maxAdaptiveInterval = 150
	}
	for frame := 0; frame < frames; frame++ {
		if _, err := io.ReadFull(in, cur); err != nil {
			return 0, 0, err
		}
		var indexEntry [4]byte
		binary.LittleEndian.PutUint32(indexEntry[:], offset)
		if _, err := idxWriter.Write(indexEntry[:]); err != nil {
			return 0, 0, err
		}
		typ := uint32(0)
		payload := cur
		if frame > 0 {
			delta := encodeDelta(prev, cur)
			distance := frame - lastKey
			forceKey := false
			if adaptive {
				changed := 0
				for i := range cur {
					if cur[i] != prev[i] {
						changed++
					}
				}
				changeBudget += changed
				forceKey = sceneSet[frame] || distance >= maxAdaptiveInterval || (distance >= minAdaptiveInterval && len(delta) > frameBytes*82/100) || (distance >= keyInterval && changeBudget > frameBytes*5)
			} else {
				forceKey = frame%keyInterval == 0
			}
			if !forceKey && len(delta)+8 < len(cur)+8 {
				typ = 1
				payload = delta
			} else {
				lastKey = frame
				changeBudget = 0
			}
		} else {
			lastKey = 0
		}
		n, err := writeRecord(out, typ, payload)
		if err != nil {
			return 0, 0, err
		}
		offset += uint32(n)
		prev, cur = cur, prev
	}
	if err := idxWriter.Flush(); err != nil {
		return 0, 0, err
	}
	return rawSize, int64(offset), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func extractAudio(opt ProjectOptions, info MediaInfo, input string, duration float64, frameCount int, audioPath string) (bool, string, adpcmInfo, error) {
	if opt.AudioMode == "none" || info.AudioStreams == 0 {
		return false, audioCodecPCM, adpcmInfo{}, os.WriteFile(audioPath, nil, 0644)
	}
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", opt.Start), "-i", input, "-t", fmt.Sprintf("%.6f", duration), "-map", audioMapSpecifier(opt.AudioTrack), "-vn"}
	filters := audioFilters(opt, info)
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}
	args = append(args, "-ac", "1", "-ar", strconv.Itoa(audioRate), "-f", "s8", audioPath)
	output, err := runCommand(opt.FFmpegPath, args...)
	if err != nil {
		return false, audioCodecPCM, adpcmInfo{}, fmt.Errorf("FFmpeg could not convert audio:\n%s", strings.TrimSpace(string(output)))
	}
	display := float64(frameCount*opt.VBlanks) / gbaRefresh
	required := int64(math.Ceil(display * audioRate))
	aligned := (required + 15) / 16 * 16
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		return false, audioCodecPCM, adpcmInfo{}, err
	}
	if int64(len(audio)) < aligned {
		audio = append(audio, make([]byte, aligned-int64(len(audio)))...)
	} else {
		audio = audio[:aligned]
	}
	targetMiB := opt.SmartTargetMiB
	if targetMiB < 1 || targetMiB > 32 {
		targetMiB = 32
	}
	codec := resolveAudioCodec(opt.AudioCodec, opt.ExtremeOptimization, int64(len(audio)), int64(targetMiB)*1024*1024)
	if codec == audioCodecADPCM {
		encoded, adpcm, err := encodeIMAADPCM(audio, defaultADPCMBlockSamples)
		if err != nil {
			return false, codec, adpcmInfo{}, err
		}
		return true, codec, adpcm, os.WriteFile(audioPath, encoded, 0644)
	}
	pcmInfo := adpcmInfo{SampleCount: len(audio)}
	return true, audioCodecPCM, pcmInfo, os.WriteFile(audioPath, audio, 0644)
}

func safeRomTitle(value string) []byte {
	return safeGBAHeaderTitle(value)
}

func safeTitleScreenName(value string) []byte {
	return encodeGBATextFixed(value, 24)
}

func patchGBAHeader(rom []byte, title string) {
	binary.LittleEndian.PutUint32(rom[0:4], 0xEA00002E)
	copy(rom[4:0xA0], nintendoLogo)
	copy(rom[0xA0:0xAC], safeRomTitle(title))
	copy(rom[0xAC:0xB0], []byte("GM05"))
	copy(rom[0xB0:0xB2], []byte("01"))
	rom[0xB2] = 0x96
	for i := 0xB3; i < 0xBD; i++ {
		rom[i] = 0
	}
	sum := 0
	for _, v := range rom[0xA0:0xBD] {
		sum += int(v)
	}
	rom[0xBD] = byte((-0x19 - sum) & 0xFF)
	rom[0xBE] = 0
	rom[0xBF] = 0
}
func appendAligned(rom []byte, data []byte) []byte {
	rom = append(rom, data...)
	for len(rom)%4 != 0 {
		rom = append(rom, 0)
	}
	return rom
}

type convertedClip struct {
	input                                                ClipInput
	options                                              ProjectOptions
	info                                                 MediaInfo
	mediaKind                                            string
	artist, album                                        string
	mediaMetadata                                        bool
	imageSeconds                                         float64
	frameCount, paletteCount                             int
	hasAudio                                             bool
	audioCodec                                           string
	audioSampleCount, audioBlockSamples, audioBlockBytes int
	palette, paletteIndex, video, videoIndex, audio      string
	rawVideo, storedVideo                                int64
	duration                                             float64
}

func optionsForClip(project ProjectOptions, input ClipInput) ProjectOptions {
	clip := project
	clip.AudioTrack = input.AudioTrack
	if !input.Custom {
		return clip
	}
	clip.Start = input.Start
	clip.End = input.End
	clip.Speed = input.Speed
	clip.FitMode = input.FitMode
	clip.AudioMode = input.AudioMode
	clip.Volume = input.Volume
	clip.Loop = input.Loop
	clip.PaletteMode = input.PaletteMode
	clip.DitherMode = input.DitherMode
	return clip
}

func validateClipSettings(opt ProjectOptions, label string) error {
	if opt.Speed < .5 || opt.Speed > 3 {
		return fmt.Errorf("%s: speed must be between 0.5 and 3.0", label)
	}
	if opt.Volume < 0 || opt.Volume > 2 {
		return fmt.Errorf("%s: volume must be between 0 and 200 percent", label)
	}
	if opt.FitMode != "fit" && opt.FitMode != "crop" && opt.FitMode != "stretch" {
		return fmt.Errorf("%s: invalid screen framing", label)
	}
	if opt.AudioMode != "mix" && opt.AudioMode != "left" && opt.AudioMode != "right" && opt.AudioMode != "none" {
		return fmt.Errorf("%s: invalid audio mode", label)
	}
	if opt.PaletteMode != "shared" && opt.PaletteMode != "scene" {
		return fmt.Errorf("%s: invalid palette mode", label)
	}
	if opt.DitherMode != "off" && opt.DitherMode != "ordered" && opt.DitherMode != "error" {
		return fmt.Errorf("%s: invalid dithering mode", label)
	}
	return nil
}

func validateProject(opt ProjectOptions) error {
	if len(opt.Inputs) == 0 {
		return errors.New("at least one input media file is required")
	}
	if opt.OutputPath == "" || opt.FFmpegPath == "" {
		return errors.New("output and FFmpeg paths are required")
	}
	if err := validateClipSettings(opt, "project defaults"); err != nil {
		return err
	}
	if opt.VBlanks != 4 && opt.VBlanks != 5 && opt.VBlanks != 6 && opt.VBlanks != 8 {
		return errors.New("invalid frame rate")
	}
	if opt.SeekSeconds != 3 && opt.SeekSeconds != 5 && opt.SeekSeconds != 10 && opt.SeekSeconds != 15 {
		return errors.New("seek step must be 3, 5, 10 or 15 seconds")
	}
	if opt.Compression != "none" && opt.Compression != "delta" {
		return errors.New("invalid compression mode")
	}
	if opt.AudioCodec != "" && opt.AudioCodec != audioCodecPCM && opt.AudioCodec != audioCodecADPCM && opt.AudioCodec != audioCodecAuto {
		return errors.New("invalid audio quality")
	}
	if !opt.ExtremeOptimization && (opt.AudioCodec == audioCodecADPCM || opt.AudioCodec == audioCodecAuto || opt.AdaptiveKeyframes || opt.EnhancedSceneDetection) {
		return errors.New("experimental audio and adaptive encoding require the Extreme optimization preset")
	}
	if opt.OutputMode != "rom" && opt.OutputMode != "playlist" && opt.OutputMode != "menu" && opt.OutputMode != "batch" && opt.OutputMode != "longsplit" {
		return errors.New("invalid output mode")
	}
	if opt.OutputMode == "longsplit" && len(opt.Inputs) != 1 {
		return errors.New("automatic long-video splitting requires exactly one input video")
	}
	if err := validateTitleCardProject(opt.TitleCards); err != nil {
		return err
	}
	if opt.OutputMode == "menu" && len(opt.Inputs) > 1 && opt.MenuTheme != nil {
		if err := opt.MenuTheme.validate(); err != nil {
			return fmt.Errorf("menu theme: %w", err)
		}
	}
	for _, input := range opt.Inputs {
		if input.Custom {
			if err := validateClipSettings(optionsForClip(opt, input), input.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertClip(project ProjectOptions, input ClipInput, tempDir string, index, total int, progress ProgressFunc) (convertedClip, error) {
	opt := optionsForClip(project, input)
	info, err := inspectMedia(opt.FFmpegPath, input.InputPath)
	if err != nil {
		return convertedClip{}, fmt.Errorf("%s: %w", input.Name, err)
	}
	kind := info.Kind
	if kind == "" {
		kind = "video"
	}
	if input.MediaKind != "" {
		kind = input.MediaKind
	}
	// Audio media must always retain an audio stream. Treat the legacy
	// "none" setting as the default mono mix instead of generating a silent
	// audio entry that the GBA runtime cannot meaningfully play.
	if kind == "audio" && info.AudioStreams > 0 && opt.AudioMode == "none" {
		opt.AudioMode = "mix"
	}
	if opt.AudioMode != "none" && info.AudioStreams > 0 && (opt.AudioTrack < 0 || opt.AudioTrack >= info.AudioStreams) {
		return convertedClip{}, fmt.Errorf("%s: selected audio track %d is not available", input.Name, opt.AudioTrack+1)
	}
	end := opt.End
	if kind == "image" {
		end = 0
		opt.Start = 0
	} else {
		if end <= 0 || end > info.Duration {
			end = info.Duration
		}
		if opt.Start < 0 || opt.Start >= end {
			return convertedClip{}, fmt.Errorf("%s: start time must be before end time", input.Name)
		}
	}
	duration := 0.0
	if kind != "image" {
		duration = end - opt.Start
	}
	prefix := filepath.Join(tempDir, fmt.Sprintf("clip-%03d", index))
	framesPath := prefix + ".rgb"
	palettePath := prefix + ".pal"
	paletteIndexPath := prefix + ".pidx"
	rawVideoPath := prefix + ".raw"
	videoPath := prefix + ".video"
	videoIndexPath := prefix + ".vidx"
	audioPath := prefix + ".audio"
	base := index * 80 / total
	span := 80 / total
	local := func(p int, msg string) { progress(base+p*span/100, fmt.Sprintf("%s — %s", input.Name, msg)) }
	if kind == "image" {
		local(20, "converting native 240×160 image")
		if err := extractNativeImage(opt.FFmpegPath, input.InputPath, opt.FitMode, videoPath); err != nil {
			return convertedClip{}, err
		}
		_ = os.WriteFile(palettePath, nil, 0644)
		_ = os.WriteFile(paletteIndexPath, nil, 0644)
		_ = os.WriteFile(videoIndexPath, nil, 0644)
		_ = os.WriteFile(audioPath, nil, 0644)
		seconds := input.ImageSeconds
		if seconds < 0 {
			seconds = 0
		}
		return convertedClip{input: input, options: opt, info: info, mediaKind: "image", imageSeconds: seconds, frameCount: 1, paletteCount: 0, video: videoPath, palette: palettePath, paletteIndex: paletteIndexPath, videoIndex: videoIndexPath, audio: audioPath, rawVideo: nativeImageBytes, storedVideo: nativeImageBytes, duration: seconds}, nil
	}
	if kind == "audio" {
		local(20, "preparing album artwork")
		if err := extractAudioArtwork(opt.FFmpegPath, input.InputPath, videoPath); err != nil {
			return convertedClip{}, err
		}
		_ = os.WriteFile(palettePath, nil, 0644)
		_ = os.WriteFile(paletteIndexPath, nil, 0644)
		_ = os.WriteFile(videoIndexPath, nil, 0644)
		display := duration / opt.Speed
		frameCount := int(math.Ceil(display * gbaRefresh / float64(opt.VBlanks)))
		if frameCount < 1 {
			frameCount = 1
		}
		local(60, "converting audio")
		hasAudio, audioCodec, audioInfo, err := extractAudio(opt, info, input.InputPath, duration, frameCount, audioPath)
		if err != nil {
			return convertedClip{}, err
		}
		return convertedClip{input: input, options: opt, info: info, mediaKind: "audio", artist: info.Artist, album: info.Album, mediaMetadata: strings.TrimSpace(info.Artist) != "" || strings.TrimSpace(info.Album) != "", frameCount: frameCount, paletteCount: 0, hasAudio: hasAudio, audioCodec: audioCodec, audioSampleCount: audioInfo.SampleCount, audioBlockSamples: audioInfo.BlockSamples, audioBlockBytes: audioInfo.BlockBytes, palette: palettePath, paletteIndex: paletteIndexPath, video: videoPath, videoIndex: videoIndexPath, audio: audioPath, rawVideo: nativeImageBytes, storedVideo: nativeImageBytes, duration: duration}, nil
	}
	local(5, "extracting frames")
	if err := extractFrames(opt, input.InputPath, duration, framesPath, local); err != nil {
		return convertedClip{}, err
	}
	frameCount, paletteCount, sceneStarts, err := buildPalettesAndRawVideo(framesPath, palettePath, paletteIndexPath, rawVideoPath, opt.PaletteMode, opt.DitherMode, opt.EnhancedSceneDetection, local)
	if err != nil {
		return convertedClip{}, err
	}
	local(72, "compressing video")
	raw, stored, err := compressRawVideo(rawVideoPath, videoPath, videoIndexPath, opt.Compression, opt.KeyInterval, opt.AdaptiveKeyframes, sceneStarts)
	if err != nil {
		return convertedClip{}, err
	}
	local(82, "converting audio")
	hasAudio, audioCodec, audioInfo, err := extractAudio(opt, info, input.InputPath, duration, frameCount, audioPath)
	if err != nil {
		return convertedClip{}, err
	}
	return convertedClip{input: input, options: opt, info: info, mediaKind: "video", frameCount: frameCount, paletteCount: paletteCount, hasAudio: hasAudio, audioCodec: audioCodec, audioSampleCount: audioInfo.SampleCount, audioBlockSamples: audioInfo.BlockSamples, audioBlockBytes: audioInfo.BlockBytes, palette: palettePath, paletteIndex: paletteIndexPath, video: videoPath, videoIndex: videoIndexPath, audio: audioPath, rawVideo: raw, storedVideo: stored, duration: duration}, nil
}

func appendFile(rom []byte, path string) ([]byte, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	off := len(rom)
	return appendAligned(rom, data), off, nil
}

func writeClipDescriptor(dst []byte, c convertedClip, offsets map[string]int) {
	opt := c.options
	flags := uint16(0)
	if c.hasAudio {
		flags |= 1
	}
	if opt.Loop {
		flags |= 2
	}
	if (c.mediaKind == "" || c.mediaKind == "video") && opt.Compression == "delta" {
		flags |= 4
	}
	if c.paletteCount > 1 {
		flags |= 8
	}
	if c.audioCodec == audioCodecADPCM {
		flags |= 16
	}
	if opt.AdaptiveKeyframes && (c.mediaKind == "" || c.mediaKind == "video") {
		flags |= 32
	}
	if c.mediaKind == "audio" {
		flags |= 64
	}
	if c.mediaKind == "image" {
		flags |= 128
	}
	if c.mediaMetadata {
		flags |= 256
	}
	seekFrames := int(math.Round(float64(opt.SeekSeconds) * gbaRefresh / float64(opt.VBlanks)))
	if seekFrames < 1 {
		seekFrames = 1
	}
	binary.LittleEndian.PutUint32(dst[0:4], uint32(c.frameCount))
	frameSize := uint32(frameBytes)
	width, height := uint16(frameWidth), uint16(frameHeight)
	if c.mediaKind == "audio" || c.mediaKind == "image" {
		frameSize = nativeImageBytes
		width = nativeImageWidth
		height = nativeImageHeight
	}
	binary.LittleEndian.PutUint32(dst[4:8], frameSize)
	binary.LittleEndian.PutUint32(dst[8:12], uint32(offsets["video"]))
	binary.LittleEndian.PutUint32(dst[12:16], uint32(offsets["videoIndex"]))
	binary.LittleEndian.PutUint32(dst[16:20], uint32(offsets["audio"]))
	binary.LittleEndian.PutUint32(dst[20:24], uint32(offsets["audioSize"]))
	binary.LittleEndian.PutUint32(dst[24:28], uint32(offsets["palette"]))
	binary.LittleEndian.PutUint32(dst[28:32], uint32(offsets["paletteIndex"]))
	binary.LittleEndian.PutUint32(dst[32:36], uint32(offsets["seek"]))
	binary.LittleEndian.PutUint32(dst[36:40], audioRate)
	binary.LittleEndian.PutUint32(dst[40:44], uint32(seekFrames))
	binary.LittleEndian.PutUint16(dst[44:46], uint16(opt.VBlanks))
	binary.LittleEndian.PutUint16(dst[46:48], width)
	binary.LittleEndian.PutUint16(dst[48:50], height)
	binary.LittleEndian.PutUint16(dst[50:52], flags)
	binary.LittleEndian.PutUint16(dst[52:54], uint16(opt.SeekSeconds))
	binary.LittleEndian.PutUint16(dst[54:56], uint16(c.paletteCount))
	keyInterval := opt.KeyInterval
	if opt.AdaptiveKeyframes {
		keyInterval = 0
	}
	binary.LittleEndian.PutUint16(dst[56:58], uint16(keyInterval))
	copy(dst[60:72], encodeGBATextFixed(c.input.Title, 12))
	binary.LittleEndian.PutUint32(dst[72:76], uint32(c.rawVideo))
	binary.LittleEndian.PutUint32(dst[76:80], uint32(c.storedVideo))
	audioCodecID := uint32(0)
	if c.hasAudio {
		audioCodecID = 1
		if c.audioCodec == audioCodecADPCM {
			audioCodecID = 2
		}
	}
	binary.LittleEndian.PutUint32(dst[80:84], audioCodecID)
	auxCount := uint32(c.audioSampleCount)
	if c.mediaKind == "image" {
		auxCount = uint32(math.Round(c.imageSeconds * 1000))
	}
	binary.LittleEndian.PutUint32(dst[84:88], auxCount)
	binary.LittleEndian.PutUint32(dst[88:92], uint32(c.audioBlockSamples))
	binary.LittleEndian.PutUint32(dst[92:96], uint32(c.audioBlockBytes))
}

const mediaMetadataMagic = 0x31444d4d // "MMD1"
const mediaMetadataSize = 44

func encodeMediaMetadata(artist, album string) []byte {
	b := make([]byte, mediaMetadataSize)
	binary.LittleEndian.PutUint32(b[0:4], mediaMetadataMagic)
	copy(b[4:24], encodeGBATextFixed(artist, 20))
	copy(b[24:44], encodeGBATextFixed(album, 20))
	return b
}

func assembleROM(opt ProjectOptions, clips []convertedClip, output string, progress ProgressFunc) (ConvertResult, error) {
	if len(playerStub) != assetOffset {
		return ConvertResult{}, fmt.Errorf("player template size is %d, expected %d", len(playerStub), assetOffset)
	}
	rom := append([]byte(nil), playerStub...)
	clipTableOffset := len(rom)
	rom = append(rom, make([]byte, len(clips)*clipDescriptorSize)...)
	menuThemeOffset := 0
	if opt.OutputMode == "menu" && len(clips) > 1 && opt.MenuTheme != nil {
		var err error
		rom, menuThemeOffset, err = appendMenuTheme(rom, opt.MenuTheme)
		if err != nil {
			return ConvertResult{}, fmt.Errorf("menu theme: %w", err)
		}
	}
	titleCardOffset := 0
	if len(opt.TitleCardAsset) > 0 {
		titleCardOffset = len(rom)
		rom = appendAligned(rom, cloneTitleCardAsset(opt.TitleCardAsset))
	}
	var totalFrames int
	var rawVideo, storedVideo int64
	for i, c := range clips {
		offsets := map[string]int{}
		var err error
		if c.paletteCount > 0 {
			rom, offsets["palette"], err = appendFile(rom, c.palette)
			if err != nil {
				return ConvertResult{}, err
			}
		}
		if c.paletteCount > 1 {
			rom, offsets["paletteIndex"], err = appendFile(rom, c.paletteIndex)
			if err != nil {
				return ConvertResult{}, err
			}
		}
		if c.mediaKind == "video" && c.options.Compression == "delta" {
			rom, offsets["videoIndex"], err = appendFile(rom, c.videoIndex)
			if err != nil {
				return ConvertResult{}, err
			}
		}
		if c.mediaKind == "audio" && c.mediaMetadata {
			offsets["videoIndex"] = len(rom)
			rom = appendAligned(rom, encodeMediaMetadata(c.artist, c.album))
		}
		rom, offsets["video"], err = appendFile(rom, c.video)
		if err != nil {
			return ConvertResult{}, err
		}
		if c.hasAudio {
			audioData, err := os.ReadFile(c.audio)
			if err != nil {
				return ConvertResult{}, err
			}
			offsets["seek"] = len(rom)
			seek := make([]byte, c.frameCount*4)
			for frame := 0; frame < c.frameCount; frame++ {
				sample := int64(math.Floor(float64(frame*c.options.VBlanks) * audioRate / gbaRefresh))
				if c.audioSampleCount > 0 && sample >= int64(c.audioSampleCount) {
					sample = int64(c.audioSampleCount - 1)
				}
				value := sample
				if c.audioCodec != audioCodecADPCM {
					value &= ^int64(3)
					if len(audioData) >= 4 && value > int64(len(audioData)-4) {
						value = int64(len(audioData)-4) &^ 3
					}
				}
				binary.LittleEndian.PutUint32(seek[frame*4:frame*4+4], uint32(value))
			}
			rom = appendAligned(rom, seek)
			offsets["audio"] = len(rom)
			offsets["audioSize"] = len(audioData)
			rom = appendAligned(rom, audioData)
		}
		writeClipDescriptor(rom[clipTableOffset+i*clipDescriptorSize:clipTableOffset+(i+1)*clipDescriptorSize], c, offsets)
		totalFrames += c.frameCount
		rawVideo += c.rawVideo
		storedVideo += c.storedVideo
	}
	unpadded := int64(len(rom))
	if unpadded > romLimit {
		return ConvertResult{}, fmt.Errorf("conversion needs %.2f MiB, exceeding the 32 MiB GBA limit", float64(unpadded)/1048576)
	}
	meta := make([]byte, 64)
	copy(meta[0:4], []byte("GBV5"))
	binary.LittleEndian.PutUint16(meta[4:6], 5)
	flags := uint16(0)
	if opt.Resume {
		flags |= 0x0001
	}
	if opt.OutputMode == "playlist" {
		flags |= 0x0002
	}
	if opt.TitleScreenPart > 0 || titleCardOffset > 0 {
		flags |= 0x0004
	}
	binary.LittleEndian.PutUint16(meta[6:8], flags)
	binary.LittleEndian.PutUint16(meta[8:10], uint16(len(clips)))
	binary.LittleEndian.PutUint32(meta[12:16], uint32(clipTableOffset))
	binary.LittleEndian.PutUint32(meta[16:20], clipDescriptorSize)
	if opt.TitleScreenPart > 0 {
		binary.LittleEndian.PutUint32(meta[20:24], uint32(opt.TitleScreenPart))
		copy(meta[24:48], safeTitleScreenName(opt.TitleScreenName))
	}
	if menuThemeOffset > 0 {
		binary.LittleEndian.PutUint32(meta[48:52], uint32(menuThemeOffset))
	}
	if titleCardOffset > 0 {
		binary.LittleEndian.PutUint32(meta[52:56], uint32(titleCardOffset))
	}
	copy(rom[metadataOffset:metadataOffset+64], meta)
	patchGBAHeader(rom, opt.RomTitle)
	padded := nextPowerOfTwo(unpadded)
	if padded < romMinSize {
		padded = romMinSize
	}
	if padded > romLimit {
		return ConvertResult{}, errors.New("next cartridge size exceeds 32 MiB")
	}
	rom = append(rom, bytes.Repeat([]byte{0xFF}, int(padded-int64(len(rom))))...)
	progress(96, "Writing the finished ROM…")
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return ConvertResult{}, err
	}
	tmp := output + ".part"
	if err := os.WriteFile(tmp, rom, 0644); err != nil {
		return ConvertResult{}, err
	}
	_ = os.Remove(output)
	if err := os.Rename(tmp, output); err != nil {
		return ConvertResult{}, err
	}
	progress(100, "Done — your GBA ROM is ready.")
	return ConvertResult{OutputPath: output, FrameCount: totalFrames, FPS: gbaRefresh / float64(opt.VBlanks), UnpaddedSize: unpadded, PaddedSize: padded, ClipCount: len(clips), CompressedBytes: storedVideo, UncompressedBytes: rawVideo, OutputKind: "rom"}, nil
}

func formatSplitTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	totalMilliseconds := int64(math.Round(seconds * 1000))
	hours := totalMilliseconds / 3600000
	minutes := (totalMilliseconds / 60000) % 60
	wholeSeconds := (totalMilliseconds / 1000) % 60
	milliseconds := totalMilliseconds % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, wholeSeconds, milliseconds)
}

func splitROMTitle(title string, part int) string {
	base := strings.TrimSpace(title)
	if base == "" {
		base = "GBA VIDEO"
	}
	suffix := fmt.Sprintf(" P%02d", part)
	limit := 12 - len(suffix)
	if limit < 1 {
		limit = 1
	}
	if len(base) > limit {
		base = base[:limit]
	}
	return strings.TrimSpace(base) + suffix
}

func romSizeFromError(err error) (int64, bool) {
	if err == nil {
		return 0, false
	}
	text := err.Error()
	if !strings.Contains(text, "32 MiB") && !strings.Contains(text, "cartridge size") {
		return 0, false
	}
	const prefix = "conversion needs "
	start := strings.Index(text, prefix)
	if start < 0 {
		return 0, true
	}
	start += len(prefix)
	end := strings.Index(text[start:], " MiB")
	if end < 0 {
		return 0, true
	}
	value, parseErr := strconv.ParseFloat(strings.TrimSpace(text[start:start+end]), 64)
	if parseErr != nil {
		return 0, true
	}
	return int64(value * 1048576), true
}

func splitBudgetBytes(opt ProjectOptions) int64 {
	mib := opt.SplitBudgetMiB
	if mib == 0 {
		return defaultLongSplitBudget
	}
	if mib < 1 {
		mib = 1
	}
	if mib > 32 {
		mib = 32
	}
	return int64(mib) * 1024 * 1024
}

func splitSourceRange(opt ProjectOptions, info MediaInfo) (float64, float64, error) {
	start := opt.Start
	end := opt.End
	if end <= 0 || end > info.Duration {
		end = info.Duration
	}
	if start < 0 || start >= end {
		return 0, 0, errors.New("start time must be before end time")
	}
	return start, end, nil
}

func estimateLongSplitParts(opt ProjectOptions, info MediaInfo, budget int64) int {
	start, end, err := splitSourceRange(opt, info)
	if err != nil || budget <= int64(assetOffset+clipDescriptorSize+4096) {
		return 1
	}
	displayDuration := (end - start) / math.Max(opt.Speed, 0.01)
	fps := gbaRefresh / float64(opt.VBlanks)
	frames := math.Max(1, math.Ceil(displayDuration*fps))
	compressionFactor := 1.0
	if opt.Compression == "delta" {
		compressionFactor = 0.68
	}
	videoBytes := frames * frameBytes * compressionFactor
	paletteCount := 1.0
	if opt.PaletteMode == "scene" {
		paletteCount = math.Max(1, math.Ceil(frames/60))
	}
	paletteBytes := paletteCount * 512
	if paletteCount > 1 {
		paletteBytes += frames * 2
	}
	indexBytes := 0.0
	if opt.Compression == "delta" {
		indexBytes = frames * 8
	}
	audioBytes := 0.0
	if opt.AudioMode != "none" && info.AudioStreams > 0 {
		audioBytes = displayDuration*audioRate + frames*4
		if opt.ExtremeOptimization && (opt.AudioCodec == audioCodecADPCM || opt.AudioCodec == audioCodecAuto) {
			audioBytes = float64(adpcmHeaderBytes) + math.Ceil(displayDuration*audioRate/defaultADPCMBlockSamples)*float64(4+(defaultADPCMBlockSamples-1+1)/2) + frames*4
		}
	}
	perPartOverhead := int64(assetOffset + clipDescriptorSize + 512)
	if opt.TitleCards != nil && opt.TitleCards.Enabled {
		perPartOverhead += int64(titleCardHeaderSize + titleCardPixelBytes)
	}
	estimatedBytes := float64(perPartOverhead) + videoBytes + paletteBytes + indexBytes + audioBytes
	usable := float64(budget - perPartOverhead)
	if usable < 1 {
		usable = 1
	}
	parts := int(math.Ceil(math.Max(1, estimatedBytes-float64(perPartOverhead)) / usable))
	if opt.MaxPartMinutes > 0 {
		byDuration := int(math.Ceil((end - start) / (opt.MaxPartMinutes * 60)))
		if byDuration > parts {
			parts = byDuration
		}
	}
	if parts < 1 {
		parts = 1
	}
	return parts
}

func formatProgressTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int64(math.Round(seconds))
	if total >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", total/3600, (total/60)%60, total%60)
	}
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

func chapterSplitEnd(chapters []float64, cursor, candidateEnd, finalEnd float64) float64 {
	if candidateEnd >= finalEnd-0.0005 || len(chapters) == 0 {
		return candidateEnd
	}
	best := 0.0
	for _, chapter := range chapters {
		if chapter <= cursor+2 {
			continue
		}
		if chapter > candidateEnd+0.0005 {
			break
		}
		best = chapter
	}
	if best == 0 {
		return candidateEnd
	}
	threshold := math.Max(30, (candidateEnd-cursor)*0.25)
	if candidateEnd-best <= threshold {
		return best
	}
	return candidateEnd
}

type splitPartRecord struct {
	FileName          string  `json:"fileName"`
	Start             float64 `json:"start"`
	End               float64 `json:"end"`
	FrameCount        int     `json:"frameCount"`
	UnpaddedSize      int64   `json:"unpaddedSize"`
	PaddedSize        int64   `json:"paddedSize"`
	CompressedBytes   int64   `json:"compressedBytes"`
	UncompressedBytes int64   `json:"uncompressedBytes"`
}

type splitRecoveryState struct {
	Version        int               `json:"version"`
	Fingerprint    string            `json:"fingerprint"`
	SourceName     string            `json:"sourceName"`
	Start          float64           `json:"start"`
	End            float64           `json:"end"`
	Cursor         float64           `json:"cursor"`
	NextPart       int               `json:"nextPart"`
	EstimatedParts int               `json:"estimatedParts"`
	Parts          []splitPartRecord `json:"parts"`
}

func splitRecoveryRoot() string {
	if override := strings.TrimSpace(os.Getenv("GBA_VIDEO_MAKER_RECOVERY_DIR")); override != "" {
		return override
	}
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, "GBA Media Maker", "long-video-recovery")
	}
	return filepath.Join(os.TempDir(), "GBA Media Maker", "long-video-recovery")
}

func splitRecoveryIdentity(opt ProjectOptions, input ClipInput, info MediaInfo, start, end float64, budget int64) (string, error) {
	st, err := os.Stat(input.InputPath)
	if err != nil {
		return "", err
	}
	identity := struct {
		Path             string
		Name             string
		Size             int64
		Modified         int64
		Duration         float64
		Start            float64
		End              float64
		Speed            float64
		VBlanks          int
		FitMode          string
		AudioMode        string
		AudioTrack       int
		Volume           float64
		Normalize        bool
		Limiter          bool
		Compression      string
		PaletteMode      string
		DitherMode       string
		Budget           int64
		MaxPartMinutes   float64
		ChapterAware     bool
		PartTitleScreens bool
		PlayerTemplate   string
	}{
		Path: input.InputPath, Name: input.Name, Size: st.Size(), Modified: st.ModTime().UnixNano(), Duration: info.Duration,
		Start: start, End: end, Speed: opt.Speed, VBlanks: opt.VBlanks, FitMode: opt.FitMode,
		AudioMode: opt.AudioMode, AudioTrack: opt.AudioTrack, Volume: opt.Volume, Normalize: opt.Normalize, Limiter: opt.Limiter,
		Compression: opt.Compression, PaletteMode: opt.PaletteMode, DitherMode: opt.DitherMode,
		Budget: budget, MaxPartMinutes: opt.MaxPartMinutes, ChapterAware: opt.ChapterAware, PartTitleScreens: opt.PartTitleScreens,
		PlayerTemplate: fmt.Sprintf("%x", sha256.Sum256(playerStub)),
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:16]), nil
}

func saveSplitRecovery(path string, state splitRecoveryState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

func loadSplitRecovery(path, fingerprint string) (splitRecoveryState, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return splitRecoveryState{}, false
	}
	var state splitRecoveryState
	if json.Unmarshal(data, &state) != nil || state.Version != 1 || state.Fingerprint != fingerprint || state.NextPart < 1 {
		return splitRecoveryState{}, false
	}
	return state, true
}

func splitFailure(err error, recoveryDir string, completed int) error {
	if completed > 0 {
		return fmt.Errorf("%w\n\n%d completed ROM part(s) were kept. Start the same conversion again to resume.\nRecovery folder: %s", err, completed, recoveryDir)
	}
	return err
}

func convertLongVideoSplitWithBudget(opt ProjectOptions, budget int64, progress ProgressFunc) (ConvertResult, error) {
	if progress == nil {
		progress = func(int, string) {}
	}
	if len(opt.Inputs) != 1 {
		return ConvertResult{}, errors.New("automatic long-video splitting requires exactly one input video")
	}
	if budget <= int64(assetOffset+clipDescriptorSize+4096) || budget > romLimit {
		return ConvertResult{}, errors.New("invalid automatic split size budget")
	}

	input := opt.Inputs[0]
	effective := optionsForClip(opt, input)
	info, err := inspectMedia(effective.FFmpegPath, input.InputPath)
	if err != nil {
		return ConvertResult{}, fmt.Errorf("%s: %w", input.Name, err)
	}
	start, end, err := splitSourceRange(effective, info)
	if err != nil {
		return ConvertResult{}, fmt.Errorf("%s: %w", input.Name, err)
	}
	if err := os.MkdirAll(filepath.Dir(opt.OutputPath), 0755); err != nil {
		return ConvertResult{}, err
	}

	estimatedParts := estimateLongSplitParts(effective, info, budget)
	persistentRecovery := opt.ResumeLongSplit
	fingerprint, err := splitRecoveryIdentity(effective, input, info, start, end, budget)
	if err != nil {
		return ConvertResult{}, err
	}
	recoveryDir := filepath.Join(splitRecoveryRoot(), fingerprint)
	if !persistentRecovery {
		recoveryDir, err = os.MkdirTemp("", "gba-media-maker-longsplit-")
		if err != nil {
			return ConvertResult{}, err
		}
		defer os.RemoveAll(recoveryDir)
	}
	statePath := filepath.Join(recoveryDir, "state.json")
	state, resumed := splitRecoveryState{}, false
	if persistentRecovery {
		state, resumed = loadSplitRecovery(statePath, fingerprint)
	}
	if resumed {
		for _, record := range state.Parts {
			if st, statErr := os.Stat(filepath.Join(recoveryDir, record.FileName)); statErr != nil || st.IsDir() {
				resumed = false
				break
			}
		}
	}
	if !resumed {
		_ = os.RemoveAll(recoveryDir)
		if err := os.MkdirAll(recoveryDir, 0755); err != nil {
			return ConvertResult{}, err
		}
		state = splitRecoveryState{Version: 1, Fingerprint: fingerprint, SourceName: input.Name, Start: start, End: end, Cursor: start, NextPart: 1, EstimatedParts: estimatedParts}
		if err := saveSplitRecovery(statePath, state); err != nil {
			return ConvertResult{}, err
		}
	} else {
		if state.EstimatedParts > estimatedParts {
			estimatedParts = state.EstimatedParts
		}
		progress(1, fmt.Sprintf("Resuming after %d completed part(s)…", len(state.Parts)))
	}
	preserveFailure := func(failure error) error {
		if persistentRecovery {
			return splitFailure(failure, recoveryDir, len(state.Parts))
		}
		return failure
	}

	baseName := strings.TrimSuffix(sanitizeFilename(input.Name), filepath.Ext(input.Name))
	if strings.TrimSpace(baseName) == "" {
		baseName = "MY_VIDEO"
	}
	totalDuration := end - start
	cursor := state.Cursor
	part := state.NextPart
	guessSeconds := math.Min(8*60, end-cursor)
	if len(state.Parts) > 0 {
		last := state.Parts[len(state.Parts)-1]
		guessSeconds = last.End - last.Start
		if last.UnpaddedSize > 0 {
			guessSeconds *= float64(budget) / float64(last.UnpaddedSize) * 0.94
		}
	}
	maxPartSeconds := 0.0
	if opt.MaxPartMinutes > 0 {
		maxPartSeconds = opt.MaxPartMinutes * 60
	}
	if maxPartSeconds > 0 && guessSeconds > maxPartSeconds {
		guessSeconds = maxPartSeconds
	}
	reportedProgress := 0
	report := func(value int, message string) {
		if value < reportedProgress {
			value = reportedProgress
		}
		if value > 99 {
			value = 99
		}
		reportedProgress = value
		progress(value, message)
	}
	progressMessage := func(partNumber int, sourcePosition float64, detail string) string {
		return fmt.Sprintf("Part %d of approximately %d\nSource position: %s / %s\n%s", partNumber, estimatedParts, formatProgressTimestamp(sourcePosition-start), formatProgressTimestamp(totalDuration), detail)
	}

	for cursor < end-0.0005 {
		remaining := end - cursor
		candidateSeconds := math.Min(guessSeconds, remaining)
		if maxPartSeconds > 0 && candidateSeconds > maxPartSeconds {
			candidateSeconds = maxPartSeconds
		}
		if candidateSeconds < 0.1 {
			candidateSeconds = remaining
		}
		var accepted ConvertResult
		var acceptedPath string
		var acceptedEnd float64

		for attempt := 1; attempt <= 8; attempt++ {
			if candidateSeconds > remaining {
				candidateSeconds = remaining
			}
			if maxPartSeconds > 0 && candidateSeconds > maxPartSeconds {
				candidateSeconds = maxPartSeconds
			}
			if candidateSeconds < 2 && remaining > 2 {
				candidateSeconds = 2
			}
			candidateEnd := cursor + candidateSeconds
			if candidateEnd > end {
				candidateEnd = end
			}
			if opt.ChapterAware {
				candidateEnd = chapterSplitEnd(info.Chapters, cursor, candidateEnd, end)
			}
			actualSeconds := candidateEnd - cursor
			if actualSeconds <= 0 {
				candidateEnd = math.Min(end, cursor+candidateSeconds)
				actualSeconds = candidateEnd - cursor
			}
			partName := fmt.Sprintf("%s_PART_%02d.gba", baseName, part)
			partPath := filepath.Join(recoveryDir, partName)
			_ = os.Remove(partPath)

			partOpt := effective
			partOpt.Inputs = []ClipInput{{InputPath: input.InputPath, Name: input.Name, Title: input.Title}}
			partOpt.OutputPath = partPath
			partOpt.Start = cursor
			partOpt.End = candidateEnd
			partOpt.OutputMode = "rom"
			partOpt.RomTitle = splitROMTitle(opt.RomTitle, part)
			partOpt.Loop = false
			if settings, enabled := resolveTitleCardSettings(opt.TitleCards, input.Name, part); enabled {
				partOpt.TitleCard = &settings
				partOpt.TitleScreenPart = part
				partOpt.TitleScreenName = strings.TrimSuffix(input.Name, filepath.Ext(input.Name))
			} else if opt.PartTitleScreens {
				// Legacy projects still receive the old text-only title screen.
				partOpt.TitleScreenPart = part
				partOpt.TitleScreenName = strings.TrimSuffix(input.Name, filepath.Ext(input.Name))
			}

			partResult, convertErr := convertProjectExact(partOpt, func(p int, msg string) {
				sourcePosition := cursor + actualSeconds*float64(p)/100
				fraction := (sourcePosition - start) / totalDuration
				mapped := 2 + int(fraction*90)
				report(mapped, progressMessage(part, sourcePosition, msg))
			})
			if convertErr != nil {
				needed, sizeErr := romSizeFromError(convertErr)
				if !sizeErr {
					return ConvertResult{}, preserveFailure(fmt.Errorf("part %02d: %w", part, convertErr))
				}
				ratio := 0.72
				if needed > 0 {
					ratio = float64(budget) / float64(needed) * 0.96
				}
				if ratio > 0.9 {
					ratio = 0.9
				}
				if ratio < 0.25 {
					ratio = 0.25
				}
				candidateSeconds = actualSeconds * ratio
				if candidateSeconds < 0.5 {
					return ConvertResult{}, preserveFailure(fmt.Errorf("part %02d cannot fit within the selected ROM size even at the minimum duration", part))
				}
				report(reportedProgress, progressMessage(part, cursor, "Part was too large; retrying a shorter segment…"))
				continue
			}

			if partResult.UnpaddedSize > budget {
				ratio := float64(budget) / float64(partResult.UnpaddedSize) * 0.97
				if ratio > 0.92 {
					ratio = 0.92
				}
				candidateSeconds = actualSeconds * ratio
				_ = os.Remove(partPath)
				report(reportedProgress, progressMessage(part, cursor, "Part is above the selected safety size; retrying…"))
				continue
			}

			atFinalEnd := candidateEnd >= end-0.0005
			atDurationLimit := maxPartSeconds > 0 && actualSeconds >= maxPartSeconds-0.0005
			if !atFinalEnd && !atDurationLimit && partResult.UnpaddedSize < budget*88/100 && attempt < 8 {
				proposed := actualSeconds * float64(budget) / float64(partResult.UnpaddedSize) * 0.96
				if proposed > actualSeconds*2 {
					proposed = actualSeconds * 2
				}
				if proposed > remaining {
					proposed = remaining
				}
				if maxPartSeconds > 0 && proposed > maxPartSeconds {
					proposed = maxPartSeconds
				}
				if proposed > actualSeconds+math.Max(1, actualSeconds*0.03) {
					candidateSeconds = proposed
					_ = os.Remove(partPath)
					report(reportedProgress, progressMessage(part, cursor, "Part has room; extending it before finalizing…"))
					continue
				}
			}

			accepted = partResult
			acceptedPath = partPath
			acceptedEnd = candidateEnd
			break
		}

		if acceptedPath == "" {
			return ConvertResult{}, preserveFailure(fmt.Errorf("could not find a safe size for part %02d", part))
		}
		record := splitPartRecord{
			FileName: filepath.Base(acceptedPath), Start: cursor, End: acceptedEnd, FrameCount: accepted.FrameCount,
			UnpaddedSize: accepted.UnpaddedSize, PaddedSize: accepted.PaddedSize,
			CompressedBytes: accepted.CompressedBytes, UncompressedBytes: accepted.UncompressedBytes,
		}
		state.Parts = append(state.Parts, record)
		acceptedDuration := acceptedEnd - cursor
		cursor = acceptedEnd
		state.Cursor = cursor
		state.NextPart = part + 1
		if len(state.Parts) > 0 && cursor < end {
			averageDuration := (cursor - start) / float64(len(state.Parts))
			if averageDuration > 0 {
				dynamicEstimate := len(state.Parts) + int(math.Ceil((end-cursor)/averageDuration))
				if maxPartSeconds > 0 {
					minimumByDuration := len(state.Parts) + int(math.Ceil((end-cursor)/maxPartSeconds))
					if minimumByDuration > dynamicEstimate {
						dynamicEstimate = minimumByDuration
					}
				}
				if dynamicEstimate < len(state.Parts) {
					dynamicEstimate = len(state.Parts)
				}
				estimatedParts = dynamicEstimate
			}
		}
		state.EstimatedParts = estimatedParts
		if err := saveSplitRecovery(statePath, state); err != nil {
			return ConvertResult{}, preserveFailure(err)
		}
		remaining = end - cursor
		if remaining > 0 {
			guessSeconds = acceptedDuration
			if accepted.UnpaddedSize > 0 {
				guessSeconds *= float64(budget) / float64(accepted.UnpaddedSize) * 0.94
			}
			if guessSeconds < 2 {
				guessSeconds = 2
			}
			if guessSeconds > remaining {
				guessSeconds = remaining
			}
			if maxPartSeconds > 0 && guessSeconds > maxPartSeconds {
				guessSeconds = maxPartSeconds
			}
		}
		part++
		report(2+int((cursor-start)/totalDuration*90), progressMessage(part, cursor, fmt.Sprintf("Completed part %02d", part-1)))
	}

	var manifest strings.Builder
	manifest.WriteString("GBA Media Maker automatic long-video split\n")
	manifest.WriteString("Source: " + input.Name + "\n")
	fmt.Fprintf(&manifest, "Target data size per ROM: %.0f MiB\n", float64(budget)/1048576)
	if maxPartSeconds > 0 {
		fmt.Fprintf(&manifest, "Maximum source duration per ROM: %s\n", formatProgressTimestamp(maxPartSeconds))
	}
	manifest.WriteString("Each part continues at the exact source timestamp where the previous part ended.\n\n")
	var totalFrames int
	var totalPadded, totalUnpadded, totalCompressed, totalRaw int64
	for _, record := range state.Parts {
		fmt.Fprintf(&manifest, "%s  %s - %s  data %.2f MiB  cartridge %.0f MiB\n",
			record.FileName, formatSplitTimestamp(record.Start), formatSplitTimestamp(record.End),
			float64(record.UnpaddedSize)/1048576, float64(record.PaddedSize)/1048576)
		totalFrames += record.FrameCount
		totalPadded += record.PaddedSize
		totalUnpadded += record.UnpaddedSize
		totalCompressed += record.CompressedBytes
		totalRaw += record.UncompressedBytes
	}
	fmt.Fprintf(&manifest, "\nParts: %d\nTotal unpadded ROM data: %.2f MiB\nTotal padded cartridge data: %.2f MiB\n", len(state.Parts), float64(totalUnpadded)/1048576, float64(totalPadded)/1048576)

	zipTemp := opt.OutputPath + ".part"
	_ = os.Remove(zipTemp)
	zipFile, err := os.Create(zipTemp)
	if err != nil {
		return ConvertResult{}, preserveFailure(err)
	}
	zw := zip.NewWriter(zipFile)
	zipOK := false
	defer func() {
		if !zipOK {
			_ = zw.Close()
			_ = zipFile.Close()
			_ = os.Remove(zipTemp)
		}
	}()
	for _, record := range state.Parts {
		data, readErr := os.ReadFile(filepath.Join(recoveryDir, record.FileName))
		if readErr != nil {
			return ConvertResult{}, preserveFailure(readErr)
		}
		w, createErr := zw.Create(record.FileName)
		if createErr != nil {
			return ConvertResult{}, preserveFailure(createErr)
		}
		if _, writeErr := w.Write(data); writeErr != nil {
			return ConvertResult{}, preserveFailure(writeErr)
		}
	}
	manifestWriter, err := zw.Create("PARTS.txt")
	if err != nil {
		return ConvertResult{}, preserveFailure(err)
	}
	if _, err := io.WriteString(manifestWriter, manifest.String()); err != nil {
		return ConvertResult{}, preserveFailure(err)
	}
	progress(95, "Finishing the numbered ZIP…")
	if err := zw.Close(); err != nil {
		return ConvertResult{}, preserveFailure(err)
	}
	if err := zipFile.Close(); err != nil {
		return ConvertResult{}, preserveFailure(err)
	}
	zipOK = true
	_ = os.Remove(opt.OutputPath)
	if err := os.Rename(zipTemp, opt.OutputPath); err != nil {
		return ConvertResult{}, preserveFailure(err)
	}
	st, err := os.Stat(opt.OutputPath)
	if err != nil {
		return ConvertResult{}, err
	}
	partCount := len(state.Parts)
	_ = os.RemoveAll(recoveryDir)
	progress(100, fmt.Sprintf("Done — %d numbered ROM parts are ready.", partCount))
	return ConvertResult{
		OutputPath: opt.OutputPath, FrameCount: totalFrames,
		UnpaddedSize: st.Size(), PaddedSize: st.Size(), ClipCount: partCount,
		CompressedBytes: totalCompressed, UncompressedBytes: totalRaw,
		OutputKind: "zip", EstimatedParts: estimatedParts,
	}, nil
}

func convertProjectExact(opt ProjectOptions, progress ProgressFunc) (ConvertResult, error) {
	if progress == nil {
		progress = func(int, string) {}
	}
	if opt.SeekSeconds == 0 {
		opt.SeekSeconds = 5
	}
	if opt.FitMode == "" {
		opt.FitMode = "fit"
	}
	if opt.Compression == "" {
		opt.Compression = "delta"
	}
	if opt.PaletteMode == "" {
		opt.PaletteMode = "shared"
	}
	if opt.DitherMode == "" {
		opt.DitherMode = "ordered"
	}
	if opt.OutputMode == "" {
		opt.OutputMode = "rom"
	}
	if opt.KeyInterval == 0 {
		opt.KeyInterval = 30
	}
	if err := validateProject(opt); err != nil {
		return ConvertResult{}, err
	}
	if opt.OutputMode == "longsplit" {
		return convertLongVideoSplitWithBudget(opt, splitBudgetBytes(opt), progress)
	}
	tempDir, err := os.MkdirTemp("", "gba-media-maker-")
	if err != nil {
		return ConvertResult{}, err
	}
	defer os.RemoveAll(tempDir)
	if opt.OutputMode == "batch" && len(opt.Inputs) > 1 {
		progress(1, "Creating batch ROMs…")
		zipFile, err := os.Create(opt.OutputPath + ".part")
		if err != nil {
			return ConvertResult{}, err
		}
		zw := zip.NewWriter(zipFile)
		var totalSize int64
		for i, input := range opt.Inputs {
			single := opt
			single.Inputs = []ClipInput{input}
			single.OutputMode = "rom"
			romPath := filepath.Join(tempDir, strings.TrimSuffix(sanitizeFilename(input.Name), filepath.Ext(input.Name))+"_GBA.gba")
			single.OutputPath = romPath
			res, err := convertProjectExact(single, func(p int, msg string) {
				progress((i*100+p)/len(opt.Inputs), fmt.Sprintf("%s — %s", input.Name, msg))
			})
			if err != nil {
				zw.Close()
				zipFile.Close()
				return ConvertResult{}, err
			}
			data, err := os.ReadFile(romPath)
			if err != nil {
				return ConvertResult{}, err
			}
			w, err := zw.Create(filepath.Base(romPath))
			if err != nil {
				return ConvertResult{}, err
			}
			if _, err := w.Write(data); err != nil {
				return ConvertResult{}, err
			}
			totalSize += res.PaddedSize
		}
		if err := zw.Close(); err != nil {
			return ConvertResult{}, err
		}
		if err := zipFile.Close(); err != nil {
			return ConvertResult{}, err
		}
		_ = os.Remove(opt.OutputPath)
		if err := os.Rename(opt.OutputPath+".part", opt.OutputPath); err != nil {
			return ConvertResult{}, err
		}
		st, _ := os.Stat(opt.OutputPath)
		progress(100, "Batch ZIP is ready.")
		return ConvertResult{OutputPath: opt.OutputPath, PaddedSize: st.Size(), UnpaddedSize: st.Size(), ClipCount: len(opt.Inputs), OutputKind: "zip", CompressedBytes: totalSize}, nil
	}
	clips := make([]convertedClip, 0, len(opt.Inputs))
	for i, input := range opt.Inputs {
		clip, err := convertClip(opt, input, tempDir, i, len(opt.Inputs), progress)
		if err != nil {
			return ConvertResult{}, err
		}
		clips = append(clips, clip)
	}
	if opt.TitleCard != nil && len(opt.Inputs) == 1 {
		progress(84, "Rendering native 240×160 title card…")
		asset, assetErr := prepareTitleCardAsset(opt, opt.Inputs[0], tempDir)
		if assetErr != nil {
			return ConvertResult{}, assetErr
		}
		opt.TitleCardAsset = asset
	}
	return assembleROM(opt, clips, opt.OutputPath, progress)
}

func automaticSplitOutputPath(output string) string {
	ext := filepath.Ext(output)
	base := strings.TrimSuffix(output, ext)
	if strings.TrimSpace(base) == "" {
		base = output
	}
	return base + "_PARTS.zip"
}

func projectNeedsAutomaticSplit(opt ProjectOptions, budget int64) (bool, error) {
	if len(opt.Inputs) != 1 || (opt.OutputMode != "" && opt.OutputMode != "rom") {
		return false, nil
	}
	input := opt.Inputs[0]
	effective := optionsForClip(opt, input)
	info, err := inspectMedia(effective.FFmpegPath, input.InputPath)
	if err != nil {
		return false, err
	}
	if info.Kind != "video" {
		return false, nil
	}
	end := effective.End
	if end <= 0 || end > info.Duration {
		end = info.Duration
	}
	if effective.Start < 0 || effective.Start >= end || effective.Speed <= 0 || effective.VBlanks <= 0 {
		return false, nil // normal validation will return the detailed error
	}
	if effective.MaxPartMinutes > 0 && end-effective.Start > effective.MaxPartMinutes*60+0.0005 {
		return true, nil
	}
	displaySeconds := (end - effective.Start) / effective.Speed
	fps := gbaRefresh / float64(effective.VBlanks)
	frameCount := int64(math.Ceil(displaySeconds * fps))
	minimum := int64(assetOffset + clipDescriptorSize + 512)
	if effective.AudioMode != "none" && info.AudioStreams > 0 {
		audioBytes := int64(math.Ceil(displaySeconds * audioRate))
		if effective.ExtremeOptimization && (effective.AudioCodec == audioCodecADPCM || effective.AudioCodec == audioCodecAuto) {
			blocks := int64(math.Ceil(float64(audioBytes) / defaultADPCMBlockSamples))
			audioBytes = adpcmHeaderBytes + blocks*int64(4+(defaultADPCMBlockSamples-1+1)/2)
		}
		minimum += audioBytes + frameCount*4
	}
	return minimum >= budget, nil
}

func convertProjectWithAutoSplitBudget(opt ProjectOptions, budget int64, progress ProgressFunc) (ConvertResult, error) {
	if progress == nil {
		progress = func(int, string) {}
	}
	eligible := false
	if len(opt.Inputs) == 1 && (opt.OutputMode == "" || opt.OutputMode == "rom") {
		kind := opt.Inputs[0].MediaKind
		if kind == "" {
			effective := optionsForClip(opt, opt.Inputs[0])
			if info, inspectErr := inspectMedia(effective.FFmpegPath, opt.Inputs[0].InputPath); inspectErr == nil {
				kind = info.Kind
			}
		}
		eligible = kind == "video"
	}
	if eligible {
		needsSplit, preflightErr := projectNeedsAutomaticSplit(opt, budget)
		if preflightErr == nil && needsSplit {
			split := opt
			split.OutputMode = "longsplit"
			split.OutputPath = automaticSplitOutputPath(opt.OutputPath)
			progress(1, "Video is too large for one cartridge; splitting it automatically…")
			result, err := convertLongVideoSplitWithBudget(split, budget, progress)
			result.AutoSplit = err == nil
			return result, err
		}
	}

	result, err := convertProjectExact(opt, progress)
	if err == nil {
		if !eligible || result.UnpaddedSize <= budget {
			return result, nil
		}
		_ = os.Remove(result.OutputPath)
	} else {
		if !eligible {
			return ConvertResult{}, err
		}
		if _, sizeErr := romSizeFromError(err); !sizeErr {
			return ConvertResult{}, err
		}
	}

	split := opt
	split.OutputMode = "longsplit"
	split.OutputPath = automaticSplitOutputPath(opt.OutputPath)
	progress(1, "One ROM would exceed the cartridge limit; splitting it automatically…")
	result, splitErr := convertLongVideoSplitWithBudget(split, budget, progress)
	result.AutoSplit = splitErr == nil
	return result, splitErr
}

func convertProject(opt ProjectOptions, progress ProgressFunc) (ConvertResult, error) {
	return convertProjectWithAutoSplitBudget(opt, splitBudgetBytes(opt), progress)
}

func convertVideo(opt ConvertOptions, progress ProgressFunc) (ConvertResult, error) {
	title := opt.RomTitle
	if title == "" {
		title = "GBA VIDEO"
	}
	return convertProject(ProjectOptions{
		Inputs:     []ClipInput{{InputPath: opt.InputPath, Name: filepath.Base(opt.InputPath), Title: title, AudioTrack: opt.AudioTrack}},
		OutputPath: opt.OutputPath, FFmpegPath: opt.FFmpegPath,
		Start: opt.Start, End: opt.End, Speed: opt.Speed, VBlanks: opt.VBlanks,
		FitMode: opt.FitMode, AudioMode: opt.AudioMode, AudioTrack: opt.AudioTrack, Volume: opt.Volume, Loop: opt.Loop,
		RomTitle: title, SeekSeconds: opt.SeekSeconds, Normalize: opt.Normalize, Limiter: opt.Limiter,
		Resume: opt.Resume, Compression: opt.Compression, PaletteMode: opt.PaletteMode,
		DitherMode: opt.DitherMode, OutputMode: "rom", KeyInterval: opt.KeyInterval,
		SplitBudgetMiB: opt.SplitBudgetMiB, MaxPartMinutes: opt.MaxPartMinutes,
		ChapterAware: opt.ChapterAware, PartTitleScreens: opt.PartTitleScreens,
		ResumeLongSplit: opt.ResumeLongSplit, TitleScreenPart: opt.TitleScreenPart,
		TitleScreenName: opt.TitleScreenName, TitleCards: opt.TitleCards,
		Preset: opt.Preset, AudioCodec: opt.AudioCodec,
		ExtremeOptimization: opt.ExtremeOptimization, AdaptiveKeyframes: opt.AdaptiveKeyframes,
		EnhancedSceneDetection: opt.EnhancedSceneDetection, SmartTargetMiB: opt.SmartTargetMiB,
		SmartPriority: opt.SmartPriority,
	}, progress)
}

func generatePreviewContext(parent context.Context, ffmpegPath, input string, timeSec float64, fitMode, outPath string) error {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	// Preview extraction is intentionally lightweight. Input-side seeking avoids
	// decoding a long video from the beginning, and one decoder thread prevents
	// a tiny 240×160 preview from taking over the whole CPU.
	output, err := runCommandContext(ctx, ffmpegPath,
		"-y", "-hide_banner", "-loglevel", "error",
		"-threads", "1", "-ss", fmt.Sprintf("%.6f", timeSec), "-i", input,
		"-map", "0:v:0", "-frames:v", "1", "-an", "-sn", "-dn",
		"-vf", makePreviewFilter(fitMode), "-threads", "1", "-f", "image2", outPath)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ffmpegVideoError("Preview failed", output)
	}
	return nil
}

func generatePreview(ffmpegPath, input string, timeSec float64, fitMode, outPath string) error {
	return generatePreviewContext(context.Background(), ffmpegPath, input, timeSec, fitMode, outPath)
}

func writePCM16WAVFromS8(path string, pcm []byte, sampleRate int) error {
	const channels = 1
	const bitsPerSample = 16
	dataBytes := len(pcm) * 2
	buf := bytes.NewBuffer(make([]byte, 0, 44+dataBytes))
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataBytes))
	buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*channels*bitsPerSample/8))
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels*bitsPerSample/8))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataBytes))
	for _, sample := range pcm {
		value := int16(int8(sample)) << 8
		_ = binary.Write(buf, binary.LittleEndian, value)
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func generateAudioPreview(opt ProjectOptions, info MediaInfo, input, outPath string) error {
	if info.AudioStreams == 0 || opt.AudioMode == "none" {
		return errors.New("this video has no selected audio")
	}
	rawPath := outPath + ".s8"
	defer os.Remove(rawPath)
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", opt.Start), "-i", input, "-t", "8", "-map", audioMapSpecifier(opt.AudioTrack), "-vn"}
	filters := audioFilters(opt, info)
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}
	args = append(args, "-ac", "1", "-ar", strconv.Itoa(audioRate), "-f", "s8", rawPath)
	output, err := runCommand(opt.FFmpegPath, args...)
	if err != nil {
		return fmt.Errorf("audio preview failed: %s", strings.TrimSpace(string(output)))
	}
	pcm, err := os.ReadFile(rawPath)
	if err != nil {
		return err
	}
	codec := resolveAudioCodec(opt.AudioCodec, opt.ExtremeOptimization, int64(len(pcm)), int64(max(1, opt.SmartTargetMiB))*1024*1024)
	if codec == audioCodecADPCM {
		encoded, _, err := encodeIMAADPCM(pcm, defaultADPCMBlockSamples)
		if err != nil {
			return err
		}
		pcm, _, err = decodeIMAADPCM(encoded)
		if err != nil {
			return err
		}
	}
	return writePCM16WAVFromS8(outPath, pcm, audioRate)
}

func commandExists(name string) string { p, _ := exec.LookPath(name); return p }
