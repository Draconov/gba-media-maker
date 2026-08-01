package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
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
	romLimit           = 32 * 1024 * 1024
	romMinSize         = 1 * 1024 * 1024
	metadataOffset     = 0x3F00
	assetOffset        = 0x4000
	clipDescriptorSize = 96
	frameWidth         = 120
	frameHeight        = 80
	frameBytes         = frameWidth * frameHeight
	audioRate          = 16384
	videoPaletteColors = 250
	gbaRefresh         = 59.727500569606
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

type MediaInfo struct {
	Duration      float64 `json:"duration"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	FPS           float64 `json:"fps"`
	AudioStreams  int     `json:"audioStreams"`
	AudioChannels int     `json:"audioChannels"`
}

type ClipInput struct {
	InputPath string
	Name      string
	Title     string

	// Custom selects the per-clip overrides below. When false, the project
	// defaults in ProjectOptions are used.
	Custom      bool
	Start       float64
	End         float64
	Speed       float64
	FitMode     string
	AudioMode   string
	Volume      float64
	Loop        bool
	PaletteMode string
	DitherMode  string
}

type ProjectOptions struct {
	Inputs      []ClipInput
	OutputPath  string
	FFmpegPath  string
	Start       float64
	End         float64
	Speed       float64
	VBlanks     int
	FitMode     string
	AudioMode   string
	Volume      float64
	Loop        bool
	RomTitle    string
	SeekSeconds int
	Normalize   bool
	Limiter     bool
	Resume      bool
	Compression string // none, delta
	PaletteMode string // shared, scene
	DitherMode  string // off, ordered, error
	OutputMode  string // rom, playlist, menu, batch
	MenuPreview bool
	KeyInterval int
}

// ConvertOptions keeps the single-video command-line/test API convenient.
type ConvertOptions struct {
	InputPath   string
	OutputPath  string
	FFmpegPath  string
	Start       float64
	End         float64
	Speed       float64
	VBlanks     int
	FitMode     string
	AudioMode   string
	Volume      float64
	Loop        bool
	RomTitle    string
	SeekSeconds int
	Normalize   bool
	Limiter     bool
	Resume      bool
	Compression string
	PaletteMode string
	DitherMode  string
	KeyInterval int
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
}

type ProgressFunc func(percent int, status string)

var (
	durationPattern   = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)
	dimensionsPattern = regexp.MustCompile(`(?:^|[^0-9])(\d{2,5})x(\d{2,5})(?:[^0-9]|$)`)
	fpsPattern        = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s+fps`)
	channelsPattern   = regexp.MustCompile(`\b(\d+)\.(\d+)\b`)
)

func inspectMedia(ffmpegPath, path string) (MediaInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, _ := runCommandContext(ctx, ffmpegPath, "-hide_banner", "-i", path)
	text := string(output)
	dm := durationPattern.FindStringSubmatch(text)
	if dm == nil {
		return MediaInfo{}, errors.New("could not read video duration")
	}
	h, _ := strconv.Atoi(dm[1])
	m, _ := strconv.Atoi(dm[2])
	s, _ := strconv.ParseFloat(dm[3], 64)
	var videoLine string
	audioStreams, audioChannels := 0, 0
	for _, line := range strings.Split(text, "\n") {
		if videoLine == "" && strings.Contains(line, " Video:") {
			videoLine = line
		}
		if strings.Contains(line, " Audio:") {
			audioStreams++
			lower := strings.ToLower(line)
			if audioChannels == 0 {
				if strings.Contains(lower, "mono") {
					audioChannels = 1
				} else if strings.Contains(lower, "stereo") {
					audioChannels = 2
				} else if lm := channelsPattern.FindStringSubmatch(lower); lm != nil {
					a, _ := strconv.Atoi(lm[1])
					b, _ := strconv.Atoi(lm[2])
					audioChannels = a + b
				}
			}
		}
	}
	if videoLine == "" {
		return MediaInfo{}, errors.New("could not find a video stream")
	}
	dims := dimensionsPattern.FindStringSubmatch(videoLine)
	if dims == nil {
		return MediaInfo{}, errors.New("could not read video dimensions")
	}
	w, _ := strconv.Atoi(dims[1])
	hg, _ := strconv.Atoi(dims[2])
	fps := 0.0
	if fm := fpsPattern.FindStringSubmatch(videoLine); fm != nil {
		fps, _ = strconv.ParseFloat(fm[1], 64)
	}
	return MediaInfo{Duration: float64(h*3600+m*60) + s, Width: w, Height: hg, FPS: fps, AudioStreams: audioStreams, AudioChannels: audioChannels}, nil
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

func audioFilters(opt ProjectOptions, info MediaInfo) []string {
	var filters []string
	switch opt.AudioMode {
	case "left":
		filters = append(filters, "pan=mono|c0=c0")
	case "right":
		if info.AudioChannels <= 1 {
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

func extractFrames(opt ProjectOptions, input string, duration float64, path string, progress ProgressFunc) error {
	fps := gbaRefresh / float64(opt.VBlanks)
	vf := makeVideoFilter(opt.FitMode, opt.Speed, fps)
	output, err := runCommand(opt.FFmpegPath, "-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", opt.Start), "-i", input, "-t", fmt.Sprintf("%.6f", duration), "-an", "-vf", vf, "-pix_fmt", "rgb24", "-f", "rawvideo", path)
	if err != nil {
		return fmt.Errorf("FFmpeg could not convert the video:\n%s", strings.TrimSpace(string(output)))
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() < frameBytes*3 {
		return errors.New("converted video contains no usable frames")
	}
	return nil
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

func detectSceneStarts(framesPath string, frameCount int, rgbFrameBytes int64) ([]int, error) {
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

func buildPalettesAndRawVideo(framesPath, palettePath, paletteIndexPath, videoPath, paletteMode, ditherMode string, progress ProgressFunc) (int, int, error) {
	st, err := os.Stat(framesPath)
	if err != nil {
		return 0, 0, err
	}
	rgbFrameBytes := int64(frameBytes * 3)
	if st.Size()%rgbFrameBytes != 0 {
		return 0, 0, errors.New("FFmpeg produced an incomplete frame stream")
	}
	frameCount := int(st.Size() / rgbFrameBytes)
	if frameCount < 1 {
		return 0, 0, errors.New("no frames were produced")
	}

	sceneStarts := []int{0}
	if paletteMode == "scene" {
		progress(18, "Detecting scene changes…")
		sceneStarts, err = detectSceneStarts(framesPath, frameCount, rgbFrameBytes)
		if err != nil {
			return 0, 0, err
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
		return 0, 0, err
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
				return 0, 0, err
			}
			if _, err := io.ReadFull(f, frameBuf); err != nil {
				return 0, 0, err
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
		return 0, 0, err
	}

	if sceneCount > 1 {
		paletteIndexes := make([]byte, frameCount*2)
		for frame, scene := range frameScene {
			binary.LittleEndian.PutUint16(paletteIndexes[frame*2:frame*2+2], uint16(scene))
		}
		if err := os.WriteFile(paletteIndexPath, paletteIndexes, 0644); err != nil {
			return 0, 0, err
		}
	} else if err := os.WriteFile(paletteIndexPath, nil, 0644); err != nil {
		return 0, 0, err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, 0, err
	}
	out, err := os.Create(videoPath)
	if err != nil {
		return 0, 0, err
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
				return 0, 0, err
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
				return 0, 0, err
			}
			if frame%20 == 0 || frame+1 == frameCount {
				progress(40+int(25*float64(frame+1)/float64(frameCount)), fmt.Sprintf("Converting frame %d of %d…", frame+1, frameCount))
			}
		}
	}
	if err := writer.Flush(); err != nil {
		return 0, 0, err
	}
	return frameCount, sceneCount, nil
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

func compressRawVideo(rawPath, streamPath, indexPath, mode string, keyInterval int) (int64, int64, error) {
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
	var offset uint32
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
		if frame > 0 && frame%keyInterval != 0 {
			delta := encodeDelta(prev, cur)
			if len(delta)+8 < len(cur)+8 {
				typ = 1
				payload = delta
			}
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

func extractAudio(opt ProjectOptions, info MediaInfo, input string, duration float64, frameCount int, audioPath string) (bool, error) {
	if opt.AudioMode == "none" || info.AudioStreams == 0 {
		return false, os.WriteFile(audioPath, nil, 0644)
	}
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", opt.Start), "-i", input, "-t", fmt.Sprintf("%.6f", duration), "-map", "0:a:0", "-vn"}
	filters := audioFilters(opt, info)
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}
	args = append(args, "-ac", "1", "-ar", strconv.Itoa(audioRate), "-f", "s8", audioPath)
	output, err := runCommand(opt.FFmpegPath, args...)
	if err != nil {
		return false, fmt.Errorf("FFmpeg could not convert audio:\n%s", strings.TrimSpace(string(output)))
	}
	display := float64(frameCount*opt.VBlanks) / gbaRefresh
	required := int64(math.Ceil(display * audioRate))
	aligned := (required + 15) / 16 * 16
	audio, err := os.ReadFile(audioPath)
	if err != nil {
		return false, err
	}
	if int64(len(audio)) < aligned {
		audio = append(audio, make([]byte, aligned-int64(len(audio)))...)
	} else {
		audio = audio[:aligned]
	}
	return true, os.WriteFile(audioPath, audio, 0644)
}

func safeRomTitle(value string) []byte {
	value = strings.ToUpper(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		s = "GBA VIDEO"
	}
	raw := []byte(s)
	if len(raw) > 12 {
		raw = raw[:12]
	}
	out := bytes.Repeat([]byte{' '}, 12)
	copy(out, raw)
	return out
}
func patchGBAHeader(rom []byte, title string) {
	binary.LittleEndian.PutUint32(rom[0:4], 0xEA00002E)
	copy(rom[4:0xA0], nintendoLogo)
	copy(rom[0xA0:0xAC], safeRomTitle(title))
	copy(rom[0xAC:0xB0], []byte("GV05"))
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
	input                                           ClipInput
	options                                         ProjectOptions
	info                                            MediaInfo
	frameCount, paletteCount                        int
	hasAudio                                        bool
	palette, paletteIndex, video, videoIndex, audio string
	rawVideo, storedVideo                           int64
	duration                                        float64
}

func optionsForClip(project ProjectOptions, input ClipInput) ProjectOptions {
	if !input.Custom {
		return project
	}
	clip := project
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
		return errors.New("at least one input video is required")
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
	end := opt.End
	if end <= 0 || end > info.Duration {
		end = info.Duration
	}
	if opt.Start < 0 || opt.Start >= end {
		return convertedClip{}, fmt.Errorf("%s: start time must be before end time", input.Name)
	}
	duration := end - opt.Start
	prefix := filepath.Join(tempDir, fmt.Sprintf("clip-%03d", index))
	framesPath := prefix + ".rgb"
	palettePath := prefix + ".pal"
	paletteIndexPath := prefix + ".pidx"
	rawVideoPath := prefix + ".raw"
	videoPath := prefix + ".video"
	videoIndexPath := prefix + ".vidx"
	audioPath := prefix + ".s8"
	base := index * 80 / total
	span := 80 / total
	local := func(p int, msg string) { progress(base+p*span/100, fmt.Sprintf("%s — %s", input.Name, msg)) }
	local(5, "extracting frames")
	if err := extractFrames(opt, input.InputPath, duration, framesPath, local); err != nil {
		return convertedClip{}, err
	}
	frameCount, paletteCount, err := buildPalettesAndRawVideo(framesPath, palettePath, paletteIndexPath, rawVideoPath, opt.PaletteMode, opt.DitherMode, local)
	if err != nil {
		return convertedClip{}, err
	}
	local(72, "compressing video")
	raw, stored, err := compressRawVideo(rawVideoPath, videoPath, videoIndexPath, opt.Compression, opt.KeyInterval)
	if err != nil {
		return convertedClip{}, err
	}
	local(82, "converting audio")
	hasAudio, err := extractAudio(opt, info, input.InputPath, duration, frameCount, audioPath)
	if err != nil {
		return convertedClip{}, err
	}
	return convertedClip{input: input, options: opt, info: info, frameCount: frameCount, paletteCount: paletteCount, hasAudio: hasAudio, palette: palettePath, paletteIndex: paletteIndexPath, video: videoPath, videoIndex: videoIndexPath, audio: audioPath, rawVideo: raw, storedVideo: stored, duration: duration}, nil
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
	if opt.Compression == "delta" {
		flags |= 4
	}
	if c.paletteCount > 1 {
		flags |= 8
	}
	seekFrames := int(math.Round(float64(opt.SeekSeconds) * gbaRefresh / float64(opt.VBlanks)))
	if seekFrames < 1 {
		seekFrames = 1
	}
	binary.LittleEndian.PutUint32(dst[0:4], uint32(c.frameCount))
	binary.LittleEndian.PutUint32(dst[4:8], frameBytes)
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
	binary.LittleEndian.PutUint16(dst[46:48], frameWidth)
	binary.LittleEndian.PutUint16(dst[48:50], frameHeight)
	binary.LittleEndian.PutUint16(dst[50:52], flags)
	binary.LittleEndian.PutUint16(dst[52:54], uint16(opt.SeekSeconds))
	binary.LittleEndian.PutUint16(dst[54:56], uint16(c.paletteCount))
	binary.LittleEndian.PutUint16(dst[56:58], uint16(opt.KeyInterval))
	copy(dst[60:72], safeRomTitle(c.input.Title))
	binary.LittleEndian.PutUint32(dst[72:76], uint32(c.rawVideo))
	binary.LittleEndian.PutUint32(dst[76:80], uint32(c.storedVideo))
}

func assembleROM(opt ProjectOptions, clips []convertedClip, output string, progress ProgressFunc) (ConvertResult, error) {
	if len(playerStub) != assetOffset {
		return ConvertResult{}, fmt.Errorf("player template size is %d, expected %d", len(playerStub), assetOffset)
	}
	rom := append([]byte(nil), playerStub...)
	clipTableOffset := len(rom)
	rom = append(rom, make([]byte, len(clips)*clipDescriptorSize)...)
	var totalFrames int
	var rawVideo, storedVideo int64
	for i, c := range clips {
		offsets := map[string]int{}
		var err error
		rom, offsets["palette"], err = appendFile(rom, c.palette)
		if err != nil {
			return ConvertResult{}, err
		}
		if c.paletteCount > 1 {
			rom, offsets["paletteIndex"], err = appendFile(rom, c.paletteIndex)
			if err != nil {
				return ConvertResult{}, err
			}
		}
		if c.options.Compression == "delta" {
			rom, offsets["videoIndex"], err = appendFile(rom, c.videoIndex)
			if err != nil {
				return ConvertResult{}, err
			}
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
				off := int64(math.Floor(float64(frame*c.options.VBlanks)*audioRate/gbaRefresh)) &^ 3
				if len(audioData) >= 4 && off > int64(len(audioData)-4) {
					off = int64(len(audioData)-4) &^ 3
				}
				binary.LittleEndian.PutUint32(seek[frame*4:frame*4+4], uint32(off))
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
	if opt.MenuPreview && opt.OutputMode == "menu" {
		flags |= 0x0004
	}
	binary.LittleEndian.PutUint16(meta[6:8], flags)
	binary.LittleEndian.PutUint16(meta[8:10], uint16(len(clips)))
	binary.LittleEndian.PutUint32(meta[12:16], uint32(clipTableOffset))
	binary.LittleEndian.PutUint32(meta[16:20], clipDescriptorSize)
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

func convertProject(opt ProjectOptions, progress ProgressFunc) (ConvertResult, error) {
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
	tempDir, err := os.MkdirTemp("", "gba-video-maker-v090-")
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
			res, err := convertProject(single, func(p int, msg string) {
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
	return assembleROM(opt, clips, opt.OutputPath, progress)
}

func convertVideo(opt ConvertOptions, progress ProgressFunc) (ConvertResult, error) {
	title := opt.RomTitle
	if title == "" {
		title = "GBA VIDEO"
	}
	return convertProject(ProjectOptions{Inputs: []ClipInput{{InputPath: opt.InputPath, Name: filepath.Base(opt.InputPath), Title: title}}, OutputPath: opt.OutputPath, FFmpegPath: opt.FFmpegPath, Start: opt.Start, End: opt.End, Speed: opt.Speed, VBlanks: opt.VBlanks, FitMode: opt.FitMode, AudioMode: opt.AudioMode, Volume: opt.Volume, Loop: opt.Loop, RomTitle: title, SeekSeconds: opt.SeekSeconds, Normalize: opt.Normalize, Limiter: opt.Limiter, Resume: opt.Resume, Compression: opt.Compression, PaletteMode: opt.PaletteMode, DitherMode: opt.DitherMode, OutputMode: "rom", KeyInterval: opt.KeyInterval}, progress)
}

func generatePreview(ffmpegPath, input string, timeSec float64, fitMode, outPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	output, err := runCommandContext(ctx, ffmpegPath, "-y", "-hide_banner", "-loglevel", "error", "-i", input, "-ss", fmt.Sprintf("%.6f", timeSec), "-frames:v", "1", "-vf", makePreviewFilter(fitMode), "-f", "image2", outPath)
	if err != nil {
		return fmt.Errorf("preview failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func generateAudioPreview(opt ProjectOptions, info MediaInfo, input, outPath string) error {
	if info.AudioStreams == 0 || opt.AudioMode == "none" {
		return errors.New("this video has no selected audio")
	}
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", opt.Start), "-i", input, "-t", "8", "-map", "0:a:0", "-vn"}
	filters := audioFilters(opt, info)
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}
	args = append(args, "-ac", "1", "-ar", "44100", "-c:a", "pcm_s16le", outPath)
	output, err := runCommand(opt.FFmpegPath, args...)
	if err != nil {
		return fmt.Errorf("audio preview failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func commandExists(name string) string { p, _ := exec.LookPath(name); return p }
