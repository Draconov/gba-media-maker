package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
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
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	appName        = "GBA Video Maker"
	appVersion     = "0.5.0 Portable"
	romLimit       = 32 * 1024 * 1024
	romMinSize     = 1 * 1024 * 1024
	metadataOffset = 0x1000
	assetOffset    = 0x2000
	frameWidth     = 120
	frameHeight    = 80
	frameBytes     = frameWidth * frameHeight
	audioRate      = 16384
	gbaRefresh     = 59.727500569606
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
	Duration      float64
	Width         int
	Height        int
	FPS           float64
	AudioStreams  int
	AudioChannels int
}

type ConvertOptions struct {
	InputPath  string
	OutputPath string
	FFmpegPath string
	Start      float64
	End        float64 // 0 means end of source
	Speed      float64
	VBlanks    int
	FitMode    string // fit, crop, stretch
	AudioMode  string // mix, left, right, none
	Volume     float64
	Loop       bool
	RomTitle   string
}

type ConvertResult struct {
	OutputPath   string
	FrameCount   int
	FPS          float64
	UnpaddedSize int64
	PaddedSize   int64
}

type ProgressFunc func(percent int, status string)

func inspectMedia(ffmpegPath, path string) (MediaInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := runCommandContext(ctx, ffmpegPath, "-hide_banner", "-i", path)
	text := string(output)
	// ffmpeg returns a non-zero status for probe-only invocations. Parse output anyway.
	_ = err

	durRE := regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)
	dm := durRE.FindStringSubmatch(text)
	if dm == nil {
		return MediaInfo{}, errors.New("could not read video duration")
	}
	h, _ := strconv.Atoi(dm[1])
	m, _ := strconv.Atoi(dm[2])
	s, _ := strconv.ParseFloat(dm[3], 64)

	var videoLine string
	audioStreams := 0
	audioChannels := 0
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
				} else {
					layoutRE := regexp.MustCompile(`\b(\d+)\.(\d+)\b`)
					lm := layoutRE.FindStringSubmatch(lower)
					if lm != nil {
						a, _ := strconv.Atoi(lm[1])
						b, _ := strconv.Atoi(lm[2])
						audioChannels = a + b
					}
				}
			}
		}
	}
	if videoLine == "" {
		return MediaInfo{}, errors.New("could not find a video stream")
	}
	dimRE := regexp.MustCompile(`(?:^|[^0-9])(\d{2,5})x(\d{2,5})(?:[^0-9]|$)`)
	dims := dimRE.FindStringSubmatch(videoLine)
	if dims == nil {
		return MediaInfo{}, errors.New("could not read video dimensions")
	}
	w, _ := strconv.Atoi(dims[1])
	hgt, _ := strconv.Atoi(dims[2])
	fps := 0.0
	fpsRE := regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s+fps`)
	fm := fpsRE.FindStringSubmatch(videoLine)
	if fm != nil {
		fps, _ = strconv.ParseFloat(fm[1], 64)
	}
	return MediaInfo{Duration: float64(h*3600+m*60) + s, Width: w, Height: hgt, FPS: fps, AudioStreams: audioStreams, AudioChannels: audioChannels}, nil
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

func formatTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := int(seconds / 3600)
	minutes := int(math.Mod(seconds, 3600) / 60)
	secs := math.Mod(seconds, 60)
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%05.2f", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%05.2f", minutes, secs)
}

func nextPowerOfTwo(v int64) int64 {
	p := int64(1 << 20)
	for p < v {
		p <<= 1
	}
	return p
}

func estimateROM(duration, speed float64, vblanks int, withAudio bool) (raw, padded int64, frames int) {
	fps := gbaRefresh / float64(vblanks)
	frames = int(math.Ceil((duration / speed) * fps))
	if frames < 1 {
		frames = 1
	}
	displaySeconds := float64(frames*vblanks) / gbaRefresh
	audioBytes := int64(0)
	if withAudio {
		audioBytes = int64(math.Ceil(displaySeconds*audioRate/16.0) * 16)
	}
	raw = assetOffset + 512 + int64(frames*frameBytes) + audioBytes
	padded = nextPowerOfTwo(raw)
	return
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

func buildAtempo(speed float64) []float64 {
	factors := []float64{}
	remaining := speed
	for remaining > 2.000000001 {
		factors = append(factors, 2.0)
		remaining /= 2.0
	}
	for remaining < 0.499999999 {
		factors = append(factors, 0.5)
		remaining /= 0.5
	}
	factors = append(factors, remaining)
	return factors
}

func extractFrames(opt ConvertOptions, duration float64, path string, progress ProgressFunc) error {
	fps := gbaRefresh / float64(opt.VBlanks)
	vf := makeVideoFilter(opt.FitMode, opt.Speed, fps)
	progress(6, "Extracting and resizing video frames…")
	output, err := runCommand(opt.FFmpegPath,
		"-y", "-hide_banner", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.6f", opt.Start), "-i", opt.InputPath,
		"-t", fmt.Sprintf("%.6f", duration), "-an", "-vf", vf,
		"-pix_fmt", "rgb24", "-f", "rawvideo", path,
	)
	if err != nil {
		return fmt.Errorf("FFmpeg could not convert the video:\n%s", strings.TrimSpace(string(output)))
	}
	stat, err := os.Stat(path)
	if err != nil || stat.Size() < frameBytes*3 {
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
	points     []colorPoint
	total      uint64
	minR, maxR int
	minG, maxG int
	minB, maxB int
}

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
	channel := 0
	rRange := b.maxR - b.minR
	gRange := b.maxG - b.minG
	bRange := b.maxB - b.minB
	if gRange > rRange && gRange >= bRange {
		channel = 1
	} else if bRange > rRange && bRange > gRange {
		channel = 2
	}
	pts := append([]colorPoint(nil), b.points...)
	sort.Slice(pts, func(i, j int) bool {
		switch channel {
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
	var accum uint64
	split := 1
	for i, p := range pts {
		accum += p.count
		if accum >= half && i+1 < len(pts) {
			split = i + 1
			break
		}
	}
	if split <= 0 || split >= len(pts) {
		split = len(pts) / 2
	}
	return newColorBox(pts[:split]), newColorBox(pts[split:]), true
}

func buildPaletteAndVideo(framesPath, palettePath, videoPath string, progress ProgressFunc) (int, error) {
	stat, err := os.Stat(framesPath)
	if err != nil {
		return 0, err
	}
	rgbFrameBytes := int64(frameBytes * 3)
	if stat.Size()%rgbFrameBytes != 0 {
		return 0, errors.New("FFmpeg produced an incomplete frame stream")
	}
	frameCount := int(stat.Size() / rgbFrameBytes)
	if frameCount < 1 {
		return 0, errors.New("no frames were produced")
	}

	progress(28, "Choosing a shared 256-colour GBA palette…")
	f, err := os.Open(framesPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	hist := make([]uint64, 32768)
	sampleCount := frameCount
	if sampleCount > 120 {
		sampleCount = 120
	}
	frameBuf := make([]byte, rgbFrameBytes)
	used := map[int]bool{}
	for n := 0; n < sampleCount; n++ {
		idx := 0
		if sampleCount > 1 {
			idx = int(math.Round(float64(n) * float64(frameCount-1) / float64(sampleCount-1)))
		}
		if used[idx] {
			continue
		}
		used[idx] = true
		if _, err := f.Seek(int64(idx)*rgbFrameBytes, io.SeekStart); err != nil {
			return 0, err
		}
		if _, err := io.ReadFull(f, frameBuf); err != nil {
			return 0, err
		}
		for i := 0; i < len(frameBuf); i += 3 {
			r := (int(frameBuf[i])*31 + 127) / 255
			g := (int(frameBuf[i+1])*31 + 127) / 255
			b := (int(frameBuf[i+2])*31 + 127) / 255
			hist[r|(g<<5)|(b<<10)]++
		}
	}

	points := make([]colorPoint, 0, 32768)
	for idx, count := range hist {
		if count == 0 {
			continue
		}
		points = append(points, colorPoint{index: idx, count: count, r: idx & 31, g: (idx >> 5) & 31, b: (idx >> 10) & 31})
	}
	if len(points) == 0 {
		return 0, errors.New("could not build a color palette")
	}
	boxes := []colorBox{newColorBox(points)}
	for len(boxes) < 256 {
		best := -1
		var bestScore uint64
		for i, b := range boxes {
			if len(b.points) < 2 {
				continue
			}
			if s := b.score(); best < 0 || s > bestScore {
				best, bestScore = i, s
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

	type rgb5 struct{ r, g, b int }
	palette := make([]rgb5, 256)
	for i, box := range boxes {
		var rs, gs, bs, total uint64
		for _, p := range box.points {
			rs += uint64(p.r) * p.count
			gs += uint64(p.g) * p.count
			bs += uint64(p.b) * p.count
			total += p.count
		}
		if total > 0 {
			palette[i] = rgb5{int((rs + total/2) / total), int((gs + total/2) / total), int((bs + total/2) / total)}
		}
	}

	palFile, err := os.Create(palettePath)
	if err != nil {
		return 0, err
	}
	for _, p := range palette {
		value := uint16(p.r | (p.g << 5) | (p.b << 10))
		if err := binary.Write(palFile, binary.LittleEndian, value); err != nil {
			palFile.Close()
			return 0, err
		}
	}
	if err := palFile.Close(); err != nil {
		return 0, err
	}

	progress(40, "Building the RGB555 palette lookup…")
	lookup := make([]byte, 32768)
	for idx := 0; idx < 32768; idx++ {
		r, g, b := idx&31, (idx>>5)&31, (idx>>10)&31
		best := 0
		bestDist := math.MaxInt
		for j, p := range palette {
			dr, dg, db := r-p.r, g-p.g, b-p.b
			dist := dr*dr + dg*dg + db*db
			if dist < bestDist {
				best, bestDist = j, dist
				if dist == 0 {
					break
				}
			}
		}
		lookup[idx] = byte(best)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	out, err := os.Create(videoPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	reader := bufio.NewReaderSize(f, int(rgbFrameBytes)*2)
	writer := bufio.NewWriterSize(out, frameBytes*64)
	defer writer.Flush()
	indexBuf := make([]byte, frameBytes)
	progress(50, "Converting frames to GBA pixels…")
	for frame := 0; frame < frameCount; frame++ {
		if _, err := io.ReadFull(reader, frameBuf); err != nil {
			return 0, err
		}
		px := 0
		for i := 0; i < len(frameBuf); i += 3 {
			r := (int(frameBuf[i])*31 + 127) / 255
			g := (int(frameBuf[i+1])*31 + 127) / 255
			b := (int(frameBuf[i+2])*31 + 127) / 255
			indexBuf[px] = lookup[r|(g<<5)|(b<<10)]
			px++
		}
		if _, err := writer.Write(indexBuf); err != nil {
			return 0, err
		}
		if frame%20 == 0 || frame+1 == frameCount {
			pct := 50 + int(25*float64(frame+1)/float64(frameCount))
			progress(pct, fmt.Sprintf("Converting frame %d of %d…", frame+1, frameCount))
		}
	}
	if err := writer.Flush(); err != nil {
		return 0, err
	}
	return frameCount, nil
}

func extractAudio(opt ConvertOptions, info MediaInfo, duration float64, frameCount int, audioPath string, progress ProgressFunc) (bool, error) {
	if opt.AudioMode == "none" || info.AudioStreams == 0 {
		return false, os.WriteFile(audioPath, nil, 0644)
	}
	filters := []string{}
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
	for _, factor := range buildAtempo(opt.Speed) {
		filters = append(filters, fmt.Sprintf("atempo=%.8f", factor))
	}
	if math.Abs(opt.Volume-1.0) > 0.000001 {
		filters = append(filters, fmt.Sprintf("volume=%.6f", opt.Volume))
	}
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.6f", opt.Start), "-i", opt.InputPath,
		"-t", fmt.Sprintf("%.6f", duration), "-map", "0:a:0", "-vn",
	}
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}
	args = append(args, "-ac", "1", "-ar", strconv.Itoa(audioRate), "-f", "s8", audioPath)
	progress(78, "Converting audio to 16,384 Hz mono…")
	output, err := runCommand(opt.FFmpegPath, args...)
	if err != nil {
		return false, fmt.Errorf("FFmpeg could not convert the audio:\n%s", strings.TrimSpace(string(output)))
	}
	displaySeconds := float64(frameCount*opt.VBlanks) / gbaRefresh
	required := int64(math.Ceil(displaySeconds * audioRate))
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
	if err := os.WriteFile(audioPath, audio, 0644); err != nil {
		return false, err
	}
	return true, nil
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
	copy(rom[0x004:0x0A0], nintendoLogo)
	copy(rom[0x0A0:0x0AC], safeRomTitle(title))
	copy(rom[0x0AC:0x0B0], []byte("GV02"))
	copy(rom[0x0B0:0x0B2], []byte("01"))
	rom[0x0B2] = 0x96
	for i := 0x0B3; i < 0x0BD; i++ {
		rom[i] = 0
	}
	sum := 0
	for _, v := range rom[0x0A0:0x0BD] {
		sum += int(v)
	}
	rom[0x0BD] = byte((-0x19 - sum) & 0xFF)
	rom[0x0BE], rom[0x0BF] = 0, 0
}

func appendAligned(rom []byte, data []byte) []byte {
	rom = append(rom, data...)
	for len(rom)%4 != 0 {
		rom = append(rom, 0)
	}
	return rom
}

func assembleROM(opt ConvertOptions, frameCount int, palettePath, videoPath, audioPath string, hasAudio bool, progress ProgressFunc) (int64, int64, error) {
	if len(playerStub) != assetOffset {
		return 0, 0, errors.New("bundled player template is corrupted")
	}
	palette, err := os.ReadFile(palettePath)
	if err != nil {
		return 0, 0, err
	}
	video, err := os.ReadFile(videoPath)
	if err != nil {
		return 0, 0, err
	}
	audio := []byte{}
	if hasAudio {
		audio, err = os.ReadFile(audioPath)
		if err != nil {
			return 0, 0, err
		}
	}
	if len(palette) != 512 || len(video) != frameCount*frameBytes {
		return 0, 0, errors.New("converted asset sizes are inconsistent")
	}

	rom := append([]byte(nil), playerStub...)
	paletteOffset := len(rom)
	rom = appendAligned(rom, palette)
	videoOffset := len(rom)
	rom = appendAligned(rom, video)
	audioOffset := len(rom)
	rom = append(rom, audio...)
	unpadded := int64(len(rom))
	if unpadded > romLimit {
		return 0, 0, fmt.Errorf("conversion needs %.2f MiB, exceeding the 32 MiB GBA limit", float64(unpadded)/1048576)
	}
	flags := uint16(0)
	if hasAudio {
		flags |= 1
	}
	if opt.Loop {
		flags |= 2
	}

	metadata := &bytes.Buffer{}
	fields := []any{
		uint32(0x32564247), uint16(2), flags,
		uint32(frameCount), uint32(frameBytes), uint32(videoOffset), uint32(audioOffset), uint32(len(audio)), uint32(paletteOffset), uint32(audioRate),
		uint16(opt.VBlanks), uint16(frameWidth), uint16(frameHeight), uint16(0),
		uint32(len(audio)), uint32(0), uint32(0), uint32(0), uint32(0),
	}
	for _, field := range fields {
		if err := binary.Write(metadata, binary.LittleEndian, field); err != nil {
			return 0, 0, err
		}
	}
	if metadata.Len() != 64 {
		return 0, 0, fmt.Errorf("metadata has wrong size: %d", metadata.Len())
	}
	copy(rom[metadataOffset:metadataOffset+64], metadata.Bytes())
	patchGBAHeader(rom, opt.RomTitle)

	padded := nextPowerOfTwo(unpadded)
	if padded < romMinSize {
		padded = romMinSize
	}
	if padded > romLimit {
		return 0, 0, errors.New("next cartridge size exceeds 32 MiB")
	}
	rom = append(rom, bytes.Repeat([]byte{0xFF}, int(padded-int64(len(rom))))...)

	progress(96, "Writing the finished ROM…")
	if err := os.MkdirAll(filepath.Dir(opt.OutputPath), 0755); err != nil {
		return 0, 0, err
	}
	temp := opt.OutputPath + ".part"
	if err := os.WriteFile(temp, rom, 0644); err != nil {
		return 0, 0, err
	}
	_ = os.Remove(opt.OutputPath)
	if err := os.Rename(temp, opt.OutputPath); err != nil {
		return 0, 0, err
	}
	progress(100, "Done — your GBA ROM is ready.")
	return unpadded, padded, nil
}

func validateOptions(opt ConvertOptions, info MediaInfo) (float64, error) {
	if opt.InputPath == "" || opt.OutputPath == "" {
		return 0, errors.New("input and output files are required")
	}
	if opt.Speed < 0.5 || opt.Speed > 3.0 {
		return 0, errors.New("speed must be between 0.5 and 3.0")
	}
	if opt.Volume < 0 || opt.Volume > 2.0 {
		return 0, errors.New("volume must be between 0 and 200 percent")
	}
	if opt.VBlanks != 4 && opt.VBlanks != 5 && opt.VBlanks != 6 && opt.VBlanks != 8 {
		return 0, errors.New("invalid frame-rate preset")
	}
	end := opt.End
	if end <= 0 || end > info.Duration {
		end = info.Duration
	}
	if opt.Start < 0 || opt.Start >= end {
		return 0, errors.New("start time must be before end time")
	}
	return end - opt.Start, nil
}

func convertVideo(opt ConvertOptions, progress ProgressFunc) (ConvertResult, error) {
	if progress == nil {
		progress = func(int, string) {}
	}
	info, err := inspectMedia(opt.FFmpegPath, opt.InputPath)
	if err != nil {
		return ConvertResult{}, err
	}
	duration, err := validateOptions(opt, info)
	if err != nil {
		return ConvertResult{}, err
	}
	_, _, framesEstimate := estimateROM(duration, opt.Speed, opt.VBlanks, opt.AudioMode != "none" && info.AudioStreams > 0)
	if framesEstimate < 1 {
		return ConvertResult{}, errors.New("clip is too short")
	}

	tempDir, err := os.MkdirTemp("", "gba-video-maker-")
	if err != nil {
		return ConvertResult{}, err
	}
	defer os.RemoveAll(tempDir)
	framesPath := filepath.Join(tempDir, "frames.rgb")
	palettePath := filepath.Join(tempDir, "palette.bin")
	videoPath := filepath.Join(tempDir, "video.bin")
	audioPath := filepath.Join(tempDir, "audio.s8")

	if err := extractFrames(opt, duration, framesPath, progress); err != nil {
		return ConvertResult{}, err
	}
	frameCount, err := buildPaletteAndVideo(framesPath, palettePath, videoPath, progress)
	if err != nil {
		return ConvertResult{}, err
	}
	hasAudio, err := extractAudio(opt, info, duration, frameCount, audioPath, progress)
	if err != nil {
		return ConvertResult{}, err
	}
	unpadded, padded, err := assembleROM(opt, frameCount, palettePath, videoPath, audioPath, hasAudio, progress)
	if err != nil {
		return ConvertResult{}, err
	}
	return ConvertResult{OutputPath: opt.OutputPath, FrameCount: frameCount, FPS: gbaRefresh / float64(opt.VBlanks), UnpaddedSize: unpadded, PaddedSize: padded}, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func commandExists(name string) string {
	p, _ := exec.LookPath(name)
	return p
}
