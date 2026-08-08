package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

const (
	smartAnalysisVersion = 1
	smartScanFrames      = 120
	smartSampleCount     = 6
)

type SmartSample struct {
	Time       float64 `json:"time"`
	Kind       string  `json:"kind"`
	Motion     float64 `json:"motion"`
	Detail     float64 `json:"detail"`
	Brightness float64 `json:"brightness"`
	Colour     float64 `json:"colour"`
}

type SmartCandidate struct {
	ID                string  `json:"id"`
	Label             string  `json:"label"`
	VBlanks           int     `json:"vblanks"`
	FPS               float64 `json:"fps"`
	PaletteMode       string  `json:"paletteMode"`
	DitherMode        string  `json:"ditherMode"`
	AdaptiveKeyframes bool    `json:"adaptiveKeyframes"`
	AudioCodec        string  `json:"audioCodec"`
	EstimatedBytes    int64   `json:"estimatedBytes"`
	EstimatedMinBytes int64   `json:"estimatedMinBytes"`
	EstimatedMaxBytes int64   `json:"estimatedMaxBytes"`
	VisualQuality     int     `json:"visualQuality"`
	MotionQuality     int     `json:"motionQuality"`
	TemporalStability int     `json:"temporalStability"`
	AudioQuality      int     `json:"audioQuality"`
	FitsTarget        bool    `json:"fitsTarget"`
	Summary           string  `json:"summary"`
}

type SmartAnalysisResult struct {
	Version      int              `json:"version"`
	TargetBytes  int64            `json:"targetBytes"`
	Priority     string           `json:"priority"`
	Duration     float64          `json:"duration"`
	Confidence   string           `json:"confidence"`
	Samples      []SmartSample    `json:"samples"`
	Recommended  SmartCandidate   `json:"recommended"`
	Alternatives []SmartCandidate `json:"alternatives"`
	Candidates   []SmartCandidate `json:"candidates"`
	AnalyzedAt   string           `json:"analyzedAt"`
}

type smartFrameMetric struct {
	index      int
	time       float64
	motion     float64
	detail     float64
	brightness float64
	colour     float64
	scene      float64
}

func normalizedAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func analyseRGBFrame(frame, previous []byte) (motion, detail, brightness, colour, scene float64) {
	if len(frame) != frameBytes*3 {
		return
	}
	var lumaSum, colourSum, edgeSum, motionSum, scenePixels float64
	pixels := frameWidth * frameHeight
	for y := 0; y < frameHeight; y++ {
		for x := 0; x < frameWidth; x++ {
			i := (y*frameWidth + x) * 3
			r, g, b := float64(frame[i]), float64(frame[i+1]), float64(frame[i+2])
			l := (77*r + 150*g + 29*b) / 256
			lumaSum += l
			maxc := math.Max(r, math.Max(g, b))
			minc := math.Min(r, math.Min(g, b))
			colourSum += maxc - minc
			if x > 0 {
				j := i - 3
				ll := (77*float64(frame[j]) + 150*float64(frame[j+1]) + 29*float64(frame[j+2])) / 256
				edgeSum += normalizedAbs(l - ll)
			}
			if y > 0 {
				j := i - frameWidth*3
				ll := (77*float64(frame[j]) + 150*float64(frame[j+1]) + 29*float64(frame[j+2])) / 256
				edgeSum += normalizedAbs(l - ll)
			}
			if len(previous) == len(frame) {
				pr, pg, pb := float64(previous[i]), float64(previous[i+1]), float64(previous[i+2])
				pl := (77*pr + 150*pg + 29*pb) / 256
				d := normalizedAbs(l - pl)
				motionSum += d
				if d > 48 {
					scenePixels++
				}
			}
		}
	}
	brightness = lumaSum / float64(pixels) / 255
	colour = colourSum / float64(pixels) / 255
	detail = edgeSum / float64(pixels*2) / 64
	if detail > 1 {
		detail = 1
	}
	motion = motionSum / float64(pixels) / 64
	if motion > 1 {
		motion = 1
	}
	scene = scenePixels / float64(pixels)
	return
}

func extractSmartScanContext(parent context.Context, ffmpegPath, input string, start, end float64, output string) (int, error) {
	duration := end - start
	if duration <= 0 {
		return 0, errors.New("analysis range is empty")
	}
	count := smartScanFrames
	if duration < 60 {
		count = int(math.Max(18, math.Ceil(duration*2)))
	}
	fps := float64(count) / duration
	if fps > 2 {
		fps = 2
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprintf("%.6f", start), "-i", input, "-t", fmt.Sprintf("%.6f", duration), "-vf", fmt.Sprintf("fps=%.8f,scale=%d:%d:flags=area", fps, frameWidth, frameHeight), "-pix_fmt", "rgb24", "-f", "rawvideo", output}
	out, err := runCommandContext(ctx, ffmpegPath, args...)
	if err != nil {
		return 0, ffmpegVideoError("FFmpeg smart scan failed", out)
	}
	st, err := os.Stat(output)
	if err != nil {
		return 0, err
	}
	frameSize := int64(frameBytes * 3)
	if st.Size() < frameSize || st.Size()%frameSize != 0 {
		return 0, errors.New("smart scan produced an incomplete frame stream")
	}
	return int(st.Size() / frameSize), nil
}

func extractSmartScan(ffmpegPath, input string, start, end float64, output string) (int, error) {
	return extractSmartScanContext(context.Background(), ffmpegPath, input, start, end, output)
}

func readSmartMetrics(path string, frameCount int, start, end float64) ([]smartFrameMetric, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReaderSize(f, frameBytes*6)
	frame := make([]byte, frameBytes*3)
	previous := make([]byte, frameBytes*3)
	metrics := make([]smartFrameMetric, 0, frameCount)
	for i := 0; i < frameCount; i++ {
		if _, err := io.ReadFull(reader, frame); err != nil {
			return nil, err
		}
		motion, detail, brightness, colour, scene := analyseRGBFrame(frame, func() []byte {
			if i == 0 {
				return nil
			}
			return previous
		}())
		t := start
		if frameCount > 1 {
			t += float64(i) * (end - start) / float64(frameCount-1)
		}
		metrics = append(metrics, smartFrameMetric{index: i, time: t, motion: motion, detail: detail, brightness: brightness, colour: colour, scene: scene})
		copy(previous, frame)
	}
	return metrics, nil
}

func metricScore(m smartFrameMetric, kind string) float64 {
	switch kind {
	case "Fast motion":
		return m.motion*0.75 + m.scene*0.25
	case "High detail":
		return m.detail*0.8 + m.colour*0.2
	case "Dark scene":
		return (1-m.brightness)*0.8 + m.detail*0.2
	case "Bright / colourful":
		return m.brightness*0.35 + m.colour*0.65
	case "Scene transition":
		return m.scene*0.7 + m.motion*0.3
	case "Low motion":
		return (1-m.motion)*0.7 + m.detail*0.3
	default:
		return 1 - math.Abs(m.motion-0.35)
	}
}

func selectSmartSamples(metrics []smartFrameMetric) []SmartSample {
	if len(metrics) == 0 {
		return nil
	}
	kinds := []string{"Typical scene", "Fast motion", "High detail", "Dark scene", "Bright / colourful", "Scene transition", "Low motion"}
	used := map[int]bool{}
	var selected []SmartSample
	minimumGap := len(metrics) / 12
	if minimumGap < 2 {
		minimumGap = 2
	}
	for _, kind := range kinds {
		bestIndex, bestScore := -1, -1.0
		for i, metric := range metrics {
			near := false
			for index := range used {
				if int(math.Abs(float64(index-i))) < minimumGap {
					near = true
					break
				}
			}
			if near {
				continue
			}
			score := metricScore(metric, kind)
			if score > bestScore {
				bestIndex, bestScore = i, score
			}
		}
		if bestIndex >= 0 {
			m := metrics[bestIndex]
			selected = append(selected, SmartSample{Time: m.time, Kind: kind, Motion: m.motion, Detail: m.detail, Brightness: m.brightness, Colour: m.colour})
			used[bestIndex] = true
		}
		if len(selected) >= smartSampleCount {
			break
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Time < selected[j].Time })
	return selected
}

func averageSmartMetrics(metrics []smartFrameMetric) (motion, detail, colour, variability float64) {
	if len(metrics) == 0 {
		return .35, .4, .35, .2
	}
	var brightnessMean float64
	for _, m := range metrics {
		motion += m.motion
		detail += m.detail
		colour += m.colour
		brightnessMean += m.brightness
	}
	n := float64(len(metrics))
	motion, detail, colour, brightnessMean = motion/n, detail/n, colour/n, brightnessMean/n
	for _, m := range metrics {
		variability += math.Abs(m.brightness-brightnessMean) + math.Abs(m.motion-motion)
	}
	variability /= 2 * n
	return
}

func candidateQuality(vblanks int, palette, dither, codec string, motion, detail, colour float64) (visual, motionScore, stability, audio int) {
	fpsScore := map[int]float64{4: 1, 5: .84, 6: .70, 8: .52}[vblanks]
	paletteScore := .82
	if palette == "scene" {
		paletteScore = .96
	}
	ditherScore := .84
	switch dither {
	case "error":
		ditherScore = .98 - motion*.2
	case "ordered":
		ditherScore = .92
	case "off":
		ditherScore = .76 + (1-detail)*.12
	}
	visualF := 58 + 24*paletteScore + 15*ditherScore + 8*(1-detail*0.15)
	motionF := 48 + 47*fpsScore - 10*motion*(1-fpsScore)
	stabilityF := 92 - 22*motion
	if dither == "error" {
		stabilityF -= 14 * motion
	}
	if palette == "scene" {
		stabilityF += 4 * colour
	}
	audioF := 100.0
	if codec == audioCodecADPCM {
		audioF = 88
	}
	return clampInt(int(math.Round(visualF)), 0, 100), clampInt(int(math.Round(motionF)), 0, 100), clampInt(int(math.Round(stabilityF)), 0, 100), int(audioF)
}

func estimateSmartCandidate(duration float64, hasAudio bool, candidate SmartCandidate, target int64, motion, detail, colour, variability float64) SmartCandidate {
	fps := gbaRefresh / float64(candidate.VBlanks)
	frames := math.Max(1, math.Ceil(duration*fps))
	compression := .22 + .58*motion + .12*detail
	if candidate.DitherMode == "error" {
		compression += .13
	} else if candidate.DitherMode == "off" {
		compression -= .05
	}
	if candidate.PaletteMode == "scene" {
		compression -= .04 * colour
	}
	if candidate.AdaptiveKeyframes {
		compression *= .88 + .08*motion
	}
	compression = math.Max(.16, math.Min(.94, compression))
	video := frames * frameBytes * compression
	indexes := frames * 8
	palettes := 512.0
	if candidate.PaletteMode == "scene" {
		scenes := math.Max(1, math.Ceil(duration/18*(.7+variability*2)))
		palettes = scenes*512 + frames*2
	}
	audioBytes := 0.0
	if hasAudio {
		audioBytes = duration * audioRate
		if candidate.AudioCodec == audioCodecADPCM {
			audioBytes = float64(adpcmHeaderBytes) + math.Ceil(audioBytes/defaultADPCMBlockSamples)*float64(4+(defaultADPCMBlockSamples-1+1)/2)
		}
		audioBytes += frames * 4
	}
	total := float64(assetOffset+clipDescriptorSize+4096) + video + indexes + palettes + audioBytes
	uncertainty := .06 + variability*.12
	candidate.EstimatedBytes = int64(math.Ceil(total))
	candidate.EstimatedMinBytes = int64(math.Floor(total * (1 - uncertainty)))
	candidate.EstimatedMaxBytes = int64(math.Ceil(total * (1 + uncertainty)))
	candidate.FitsTarget = candidate.EstimatedMaxBytes <= target
	candidate.FPS = fps
	candidate.VisualQuality, candidate.MotionQuality, candidate.TemporalStability, candidate.AudioQuality = candidateQuality(candidate.VBlanks, candidate.PaletteMode, candidate.DitherMode, candidate.AudioCodec, motion, detail, colour)
	return candidate
}

func scoreCandidate(candidate SmartCandidate, priority string, target int64) float64 {
	quality := float64(candidate.VisualQuality)*.45 + float64(candidate.MotionQuality)*.3 + float64(candidate.TemporalStability)*.2 + float64(candidate.AudioQuality)*.05
	fitPenalty := 0.0
	if candidate.EstimatedMaxBytes > target {
		fitPenalty = 80 + float64(candidate.EstimatedMaxBytes-target)/float64(target)*100
	}
	headroom := float64(target-candidate.EstimatedBytes) / float64(target)
	switch priority {
	case "longest":
		quality = quality*.55 + headroom*50
	case "motion":
		quality = float64(candidate.MotionQuality)*.55 + float64(candidate.VisualQuality)*.25 + float64(candidate.TemporalStability)*.2
	case "detail":
		quality = float64(candidate.VisualQuality)*.65 + float64(candidate.MotionQuality)*.2 + float64(candidate.TemporalStability)*.15
	case "quality":
		quality = quality * 1.08
	default:
		quality += math.Min(.08, math.Max(0, headroom)) * 20
	}
	return quality - fitPenalty
}

func buildSmartCandidates(duration float64, hasAudio bool, requestedAudio string, target int64, priority string, motion, detail, colour, variability float64) []SmartCandidate {
	base := []SmartCandidate{
		{ID: "sharp", Label: "Sharper", VBlanks: 4, PaletteMode: "scene", DitherMode: "error", AdaptiveKeyframes: true, AudioCodec: audioCodecPCM, Summary: "Preserves fine detail and colour at a higher storage cost."},
		{ID: "balanced", Label: "Recommended", VBlanks: 5, PaletteMode: "scene", DitherMode: "ordered", AdaptiveKeyframes: true, AudioCodec: audioCodecPCM, Summary: "Balances detail, motion and temporal stability."},
		{ID: "stable", Label: "Stable", VBlanks: 5, PaletteMode: "shared", DitherMode: "ordered", AdaptiveKeyframes: true, AudioCodec: audioCodecPCM, Summary: "Uses a stable palette to reduce flicker and delta noise."},
		{ID: "compact", Label: "Smaller", VBlanks: 6, PaletteMode: "shared", DitherMode: "ordered", AdaptiveKeyframes: true, AudioCodec: audioCodecPCM, Summary: "Trades some motion smoothness for additional ROM space."},
		{ID: "longest", Label: "Longest", VBlanks: 8, PaletteMode: "shared", DitherMode: "off", AdaptiveKeyframes: true, AudioCodec: audioCodecADPCM, Summary: "Prioritizes maximum duration with compact audio."},
		{ID: "smooth", Label: "Smoother", VBlanks: 4, PaletteMode: "shared", DitherMode: "ordered", AdaptiveKeyframes: true, AudioCodec: audioCodecADPCM, Summary: "Keeps a higher frame rate while recovering space from audio."},
	}
	for i := range base {
		switch requestedAudio {
		case audioCodecPCM:
			base[i].AudioCodec = audioCodecPCM
		case audioCodecADPCM:
			base[i].AudioCodec = audioCodecADPCM
		case audioCodecAuto:
			// Keep both modes in the candidate set so the results explain the trade-off.
		default:
			base[i].AudioCodec = audioCodecPCM
		}
		base[i] = estimateSmartCandidate(duration, hasAudio, base[i], target, motion, detail, colour, variability)
	}
	sort.SliceStable(base, func(i, j int) bool {
		return scoreCandidate(base[i], priority, target) > scoreCandidate(base[j], priority, target)
	})
	return base
}

func AnalyzeExtremeEncodingContext(ctx context.Context, ffmpegPath, input string, info MediaInfo, opt ProjectOptions, targetMiB int, priority, requestedAudio, tempDir string) (SmartAnalysisResult, error) {
	if ffmpegPath == "" || input == "" {
		return SmartAnalysisResult{}, errors.New("FFmpeg and an input video are required")
	}
	start, end, err := splitSourceRange(opt, info)
	if err != nil {
		return SmartAnalysisResult{}, err
	}
	displayDuration := (end - start) / math.Max(.01, opt.Speed)
	if targetMiB < 1 || targetMiB > 32 {
		targetMiB = 32
	}
	if priority == "" {
		priority = "balanced"
	}
	if requestedAudio == "" {
		requestedAudio = audioCodecAuto
	}
	if requestedAudio != audioCodecPCM && requestedAudio != audioCodecADPCM && requestedAudio != audioCodecAuto {
		return SmartAnalysisResult{}, errors.New("invalid audio quality")
	}
	if tempDir == "" {
		tempDir, err = os.MkdirTemp("", "gbavm-smart-")
		if err != nil {
			return SmartAnalysisResult{}, err
		}
		defer os.RemoveAll(tempDir)
	}
	scanPath := filepath.Join(tempDir, "smart-scan.rgb")
	frameCount, err := extractSmartScanContext(ctx, ffmpegPath, input, start, end, scanPath)
	if err != nil {
		return SmartAnalysisResult{}, err
	}
	metrics, err := readSmartMetrics(scanPath, frameCount, start, end)
	if err != nil {
		return SmartAnalysisResult{}, err
	}
	motion, detail, colour, variability := averageSmartMetrics(metrics)
	target := int64(targetMiB) * 1024 * 1024
	candidates := buildSmartCandidates(displayDuration, info.AudioStreams > 0 && opt.AudioMode != "none", requestedAudio, target, priority, motion, detail, colour, variability)
	if len(candidates) == 0 {
		return SmartAnalysisResult{}, errors.New("no smart encoding candidates were produced")
	}
	recommended := candidates[0]
	alternatives := append([]SmartCandidate(nil), candidates[1:]...)
	if len(alternatives) > 3 {
		alternatives = alternatives[:3]
	}
	confidence := "medium"
	if len(metrics) >= 80 && variability < .28 {
		confidence = "high"
	} else if len(metrics) < 30 || variability > .45 {
		confidence = "low"
	}
	return SmartAnalysisResult{
		Version: smartAnalysisVersion, TargetBytes: target, Priority: priority, Duration: displayDuration,
		Confidence: confidence, Samples: selectSmartSamples(metrics), Recommended: recommended,
		Alternatives: alternatives, Candidates: candidates, AnalyzedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func AnalyzeExtremeEncoding(ffmpegPath, input string, info MediaInfo, opt ProjectOptions, targetMiB int, priority, requestedAudio, tempDir string) (SmartAnalysisResult, error) {
	return AnalyzeExtremeEncodingContext(context.Background(), ffmpegPath, input, info, opt, targetMiB, priority, requestedAudio, tempDir)
}

func writeSmartAnalysis(path string, result SmartAnalysisResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func parseSmartTarget(value string) int {
	target, _ := strconv.Atoi(value)
	if target < 1 || target > 32 {
		return 32
	}
	return target
}

func detectSceneStartsEnhanced(framesPath string, frameCount int) ([]int, error) {
	if frameCount <= 1 {
		return []int{0}, nil
	}
	f, err := os.Open(framesPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReaderSize(f, frameBytes*6)
	frame := make([]byte, frameBytes*3)
	previous := make([]byte, frameBytes*3)
	metrics := make([]smartFrameMetric, frameCount)
	for i := 0; i < frameCount; i++ {
		if _, err := io.ReadFull(reader, frame); err != nil {
			return nil, err
		}
		motion, detail, brightness, colour, scene := analyseRGBFrame(frame, func() []byte {
			if i == 0 {
				return nil
			}
			return previous
		}())
		metrics[i] = smartFrameMetric{index: i, motion: motion, detail: detail, brightness: brightness, colour: colour, scene: scene}
		copy(previous, frame)
	}
	starts := []int{0}
	lastStart := 0
	for i := 1; i < frameCount; i++ {
		m := metrics[i]
		prev := metrics[i-1]
		nextScene := 0.0
		if i+1 < frameCount {
			nextScene = metrics[i+1].scene
		}
		brightnessJump := math.Abs(m.brightness - prev.brightness)
		hardCut := m.scene >= .34 && m.motion >= .42 && (nextScene >= .08 || brightnessJump < .55)
		fadeBoundary := (m.brightness < .07 && prev.brightness >= .16) || (prev.brightness < .07 && m.brightness >= .16)
		flash := brightnessJump > .58 && nextScene < .08
		if flash {
			hardCut = false
		}
		gap := i - lastStart
		if gap >= 10 && (hardCut || fadeBoundary || gap >= 180) {
			starts = append(starts, i)
			lastStart = i
		}
	}
	return starts, nil
}
