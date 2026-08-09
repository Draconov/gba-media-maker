package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
)

const (
	menuBackgroundWidth       = 120
	menuBackgroundHeight      = 80
	menuBackgroundMaxFrames   = 16
	menuBackgroundMinDuration = 1.0
	menuBackgroundMaxDuration = 32.0
)

// menuBackgroundSampling chooses no more than 16 frames while respecting the
// MTH1 player's minimum 6-VBlank frame interval. Short clips therefore use
// fewer frames instead of being slowed down at playback time.
func menuBackgroundSampling(duration float64) (frameCount int, fps float64, frameVBlanks int, err error) {
	if !isFinitePositive(duration) || duration < menuBackgroundMinDuration || duration > menuBackgroundMaxDuration {
		return 0, 0, 0, fmt.Errorf("menu background duration must be between %.0f and %.0f seconds", menuBackgroundMinDuration, menuBackgroundMaxDuration)
	}
	frameCount = int(math.Round(duration * (gbaRefresh / 6.0)))
	if frameCount < 1 {
		frameCount = 1
	}
	if frameCount > menuBackgroundMaxFrames {
		frameCount = menuBackgroundMaxFrames
	}
	fps = float64(frameCount) / duration
	frameVBlanks = int(math.Round(gbaRefresh / fps))
	if frameVBlanks < 6 {
		frameVBlanks = 6
	}
	if frameVBlanks > 120 {
		frameVBlanks = 120
	}
	return frameCount, fps, frameVBlanks, nil
}

func isFinitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

// extractMenuBackgroundVideoFrames decodes a selected region to tightly packed
// 120x80 RGB24 frames. The browser UI performs the existing indexed-palette
// quantization, so image/GIF/video backgrounds share exactly the same MTH1
// palette and preview path.
func extractMenuBackgroundVideoFrames(ctx context.Context, ffmpegPath, input string, start, duration float64, outputPath string) ([]byte, int, int, error) {
	if ffmpegPath == "" {
		return nil, 0, 0, errors.New("FFmpeg is not available")
	}
	if start < 0 || math.IsNaN(start) || math.IsInf(start, 0) {
		return nil, 0, 0, errors.New("menu background start time is invalid")
	}
	wanted, fps, frameVBlanks, err := menuBackgroundSampling(duration)
	if err != nil {
		return nil, 0, 0, err
	}
	filter := fmt.Sprintf("fps=%.8f,scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", fps, menuBackgroundWidth, menuBackgroundHeight, menuBackgroundWidth, menuBackgroundHeight)
	output, runErr := runCommandContext(ctx, ffmpegPath,
		"-y", "-hide_banner", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.6f", start),
		"-i", input,
		"-t", fmt.Sprintf("%.6f", duration),
		"-an",
		"-vf", filter,
		"-frames:v", fmt.Sprintf("%d", wanted),
		"-pix_fmt", "rgb24",
		"-f", "rawvideo",
		outputPath,
	)
	if runErr != nil {
		return nil, 0, 0, ffmpegVideoError("FFmpeg could not decode the menu background video", output)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, 0, 0, err
	}
	frameBytes := menuBackgroundWidth * menuBackgroundHeight * 3
	if len(data) < frameBytes || len(data)%frameBytes != 0 {
		return nil, 0, 0, errors.New("menu background video produced no complete frames")
	}
	count := len(data) / frameBytes
	if count > menuBackgroundMaxFrames {
		count = menuBackgroundMaxFrames
		data = data[:count*frameBytes]
	}
	return data, count, frameVBlanks, nil
}
