package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	titleCardWidth      = 240
	titleCardHeight     = 160
	titleCardPixelBytes = titleCardWidth * titleCardHeight * 2
	titleCardHeaderSize = 32
	titleCardMagic      = 0x31444354 // "TCD1"

	titleCardFlagWaitForA  = 1 << 0
	titleCardFlagAllowSkip = 1 << 1
	titleCardFlagFade      = 1 << 2
)

// TitleCardSettings describes one native 240×160 title card. Background frames
// are extracted relative to the beginning of the generated ROM part.
type TitleCardSettings struct {
	Title                string  `json:"title"`
	Subtitle             string  `json:"subtitle"`
	BackgroundMode       string  `json:"backgroundMode"` // part-first-frame, part-frame, solid
	FrameOffsetSeconds   float64 `json:"frameOffsetSeconds"`
	Darkness             int     `json:"darkness"` // 0..90 percent
	SolidColor           string  `json:"solidColor"`
	TextColor            string  `json:"textColor,omitempty"`    // legacy shared text colour
	OutlineColor         string  `json:"outlineColor,omitempty"` // legacy shared outline colour
	Alignment            string  `json:"alignment,omitempty"`    // legacy shared alignment
	TextSize             string  `json:"textSize,omitempty"`     // legacy shared size
	TitleTextColor       string  `json:"titleTextColor"`
	TitleOutlineColor    string  `json:"titleOutlineColor"`
	TitleAlignment       string  `json:"titleAlignment"` // left, center, right
	TitleTextSize        string  `json:"titleTextSize"`  // large, medium, small
	SubtitleTextColor    string  `json:"subtitleTextColor"`
	SubtitleOutlineColor string  `json:"subtitleOutlineColor"`
	SubtitleAlignment    string  `json:"subtitleAlignment"` // left, center, right
	SubtitleTextSize     string  `json:"subtitleTextSize"`  // large, medium, small
	DrawOutline          bool    `json:"drawOutline"`
	StartMode            string  `json:"startMode"` // button, timer
	DurationSeconds      float64 `json:"durationSeconds"`
	AllowSkip            bool    `json:"allowSkip"`
	Fade                 bool    `json:"fade"`
}

type TitleCardPartSettings struct {
	Part     int               `json:"part"`
	Settings TitleCardSettings `json:"settings"`
}

type TitleCardProjectSettings struct {
	Enabled   bool                    `json:"enabled"`
	UseShared bool                    `json:"useShared"`
	Shared    TitleCardSettings       `json:"shared"`
	Parts     []TitleCardPartSettings `json:"parts,omitempty"`
}

func defaultTitleCardSettings(sourceName string) TitleCardSettings {
	base := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	return TitleCardSettings{
		Title:                sanitizeTitleCardText(base, 36),
		Subtitle:             "Part {part}",
		BackgroundMode:       "part-first-frame",
		FrameOffsetSeconds:   0,
		Darkness:             50,
		SolidColor:           "#000000",
		TextColor:            "#FFFFFF",
		OutlineColor:         "#000000",
		Alignment:            "center",
		TextSize:             "large",
		TitleTextColor:       "#FFFFFF",
		TitleOutlineColor:    "#000000",
		TitleAlignment:       "center",
		TitleTextSize:        "large",
		SubtitleTextColor:    "#FFFFFF",
		SubtitleOutlineColor: "#000000",
		SubtitleAlignment:    "center",
		SubtitleTextSize:     "small",
		DrawOutline:          true,
		StartMode:            "button",
		DurationSeconds:      3,
		AllowSkip:            true,
		Fade:                 true,
	}
}

func normalizeTitleCardSettings(value TitleCardSettings, sourceName string, part int) TitleCardSettings {
	defaults := defaultTitleCardSettings(sourceName)
	if strings.TrimSpace(value.Title) == "" {
		value.Title = defaults.Title
	}
	value.Title = sanitizeTitleCardText(value.Title, 36)
	value.Subtitle = sanitizeTitleCardText(strings.ReplaceAll(value.Subtitle, "{part}", strconv.Itoa(part)), 40)
	if value.BackgroundMode == "" {
		value.BackgroundMode = defaults.BackgroundMode
	}
	if value.BackgroundMode != "part-first-frame" && value.BackgroundMode != "part-frame" && value.BackgroundMode != "solid" {
		value.BackgroundMode = "part-first-frame"
	}
	if value.FrameOffsetSeconds < 0 || math.IsNaN(value.FrameOffsetSeconds) || math.IsInf(value.FrameOffsetSeconds, 0) {
		value.FrameOffsetSeconds = 0
	}
	if value.Darkness < 0 {
		value.Darkness = 0
	}
	if value.Darkness > 90 {
		value.Darkness = 90
	}
	if strings.TrimSpace(value.SolidColor) == "" {
		value.SolidColor = defaults.SolidColor
	}
	if strings.TrimSpace(value.TextColor) == "" {
		value.TextColor = defaults.TextColor
	}
	if strings.TrimSpace(value.OutlineColor) == "" {
		value.OutlineColor = defaults.OutlineColor
	}
	if value.Alignment != "left" && value.Alignment != "right" {
		value.Alignment = "center"
	}
	if value.TextSize != "medium" && value.TextSize != "small" {
		value.TextSize = "large"
	}
	// v0.12.1 adds independent title/subtitle typography while preserving
	// v0.12.0 project files that only stored the shared legacy fields above.
	if strings.TrimSpace(value.TitleTextColor) == "" {
		value.TitleTextColor = value.TextColor
	}
	if strings.TrimSpace(value.TitleOutlineColor) == "" {
		value.TitleOutlineColor = value.OutlineColor
	}
	if strings.TrimSpace(value.SubtitleTextColor) == "" {
		value.SubtitleTextColor = value.TextColor
	}
	if strings.TrimSpace(value.SubtitleOutlineColor) == "" {
		value.SubtitleOutlineColor = value.OutlineColor
	}
	if value.TitleAlignment != "left" && value.TitleAlignment != "right" {
		value.TitleAlignment = value.Alignment
	}
	if value.SubtitleAlignment != "left" && value.SubtitleAlignment != "right" {
		value.SubtitleAlignment = value.Alignment
	}
	if value.TitleTextSize != "medium" && value.TitleTextSize != "small" && value.TitleTextSize != "large" {
		value.TitleTextSize = value.TextSize
	}
	if value.SubtitleTextSize != "medium" && value.SubtitleTextSize != "small" && value.SubtitleTextSize != "large" {
		switch value.TextSize {
		case "large":
			value.SubtitleTextSize = "medium"
		default:
			value.SubtitleTextSize = "small"
		}
	}
	if value.StartMode != "timer" {
		value.StartMode = "button"
	}
	if value.DurationSeconds <= 0 || math.IsNaN(value.DurationSeconds) || math.IsInf(value.DurationSeconds, 0) {
		value.DurationSeconds = defaults.DurationSeconds
	}
	if value.DurationSeconds > 60 {
		value.DurationSeconds = 60
	}
	return value
}

func resolveTitleCardSettings(project *TitleCardProjectSettings, sourceName string, part int) (TitleCardSettings, bool) {
	if project == nil || !project.Enabled || part <= 0 {
		return TitleCardSettings{}, false
	}
	selected := project.Shared
	if !project.UseShared {
		for _, candidate := range project.Parts {
			if candidate.Part == part {
				selected = candidate.Settings
				break
			}
		}
	}
	return normalizeTitleCardSettings(selected, sourceName, part), true
}

func sanitizeTitleCardText(value string, maximum int) string {
	return sanitizeGBAText(value, maximum)
}

func titleCardHexRGB(value, fallback string) (uint8, uint8, uint8) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "#"))
	if len(value) != 6 {
		value = strings.TrimPrefix(fallback, "#")
	}
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		parsed, _ = strconv.ParseUint(strings.TrimPrefix(fallback, "#"), 16, 32)
	}
	return uint8(parsed >> 16), uint8(parsed >> 8), uint8(parsed)
}

func titleCardRGB555(r, g, b uint8) uint16 {
	r5 := (uint16(r)*31 + 127) / 255
	g5 := (uint16(g)*31 + 127) / 255
	b5 := (uint16(b)*31 + 127) / 255
	return r5 | g5<<5 | b5<<10
}

func titleCardWrap(text string, maxChars, maxLines int) []string {
	text = strings.TrimSpace(text)
	if text == "" || maxLines <= 0 {
		return nil
	}
	words := strings.Fields(text)
	lines := make([]string, 0, maxLines)
	current := ""
	for _, word := range words {
		wordRunes := []rune(word)
		for len(wordRunes) > maxChars {
			if current != "" {
				lines = append(lines, current)
				current = ""
				if len(lines) >= maxLines {
					return lines
				}
			}
			lines = append(lines, string(wordRunes[:maxChars]))
			wordRunes = wordRunes[maxChars:]
			if len(lines) >= maxLines {
				return lines
			}
		}
		word = string(wordRunes)
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if gbaTextLength(candidate) <= maxChars {
			current = candidate
		} else {
			lines = append(lines, current)
			if len(lines) >= maxLines {
				return lines
			}
			current = word
		}
	}
	if current != "" && len(lines) < maxLines {
		lines = append(lines, current)
	}
	return lines
}

func titleCardTextX(line string, scale int, alignment string) int {
	width := 0
	length := gbaTextLength(line)
	if length > 0 {
		width = length*4*scale - scale
	}
	switch alignment {
	case "left":
		return 12
	case "right":
		return titleCardWidth - 12 - width
	default:
		return (titleCardWidth - width) / 2
	}
}

func titleCardSetPixel(pixels []uint16, x, y int, colour uint16) {
	if x < 0 || y < 0 || x >= titleCardWidth || y >= titleCardHeight {
		return
	}
	pixels[y*titleCardWidth+x] = colour
}

func titleCardDrawGlyph(pixels []uint16, x, y, scale int, glyph uint16, colour uint16) {
	for row := 0; row < 5; row++ {
		for col := 0; col < 3; col++ {
			bit := 14 - (row*3 + col)
			if glyph&(1<<bit) == 0 {
				continue
			}
			for yy := 0; yy < scale; yy++ {
				for xx := 0; xx < scale; xx++ {
					titleCardSetPixel(pixels, x+col*scale+xx, y+row*scale+yy, colour)
				}
			}
		}
	}
}

func titleCardDrawLine(pixels []uint16, line string, y, scale int, alignment string, colour, outline uint16, drawOutline bool) {
	runes := []rune(line)
	x := titleCardTextX(line, scale, alignment)
	if drawOutline {
		radius := 1
		if scale >= 3 {
			radius = 2
		}
		offsets := [][2]int{{-radius, 0}, {radius, 0}, {0, -radius}, {0, radius}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}}
		for _, offset := range offsets {
			for index, r := range runes {
				titleCardDrawGlyph(pixels, x+index*4*scale+offset[0], y+offset[1], scale, gbaGlyphBits(r), outline)
			}
		}
	}
	for index, r := range runes {
		titleCardDrawGlyph(pixels, x+index*4*scale, y, scale, gbaGlyphBits(r), colour)
	}
}

func titleCardTextStyle(size string) (scale, maxChars, lineHeight int) {
	switch size {
	case "medium":
		return 3, 19, 18
	case "small":
		return 2, 29, 12
	default:
		return 4, 14, 24
	}
}

func titleCardTypographyGap(titleSize, subtitleSize string) int {
	if titleSize == "large" || subtitleSize == "large" {
		return 10
	}
	if titleSize == "medium" || subtitleSize == "medium" {
		return 8
	}
	return 6
}

func renderTitleCardPixels(backgroundRGB []byte, settings TitleCardSettings) ([]byte, error) {
	if len(backgroundRGB) != titleCardWidth*titleCardHeight*3 {
		return nil, fmt.Errorf("title-card background is %d bytes; expected %d", len(backgroundRGB), titleCardWidth*titleCardHeight*3)
	}
	pixels := make([]uint16, titleCardWidth*titleCardHeight)
	factor := float64(100-settings.Darkness) / 100
	for index := range pixels {
		r := uint8(math.Round(float64(backgroundRGB[index*3]) * factor))
		g := uint8(math.Round(float64(backgroundRGB[index*3+1]) * factor))
		b := uint8(math.Round(float64(backgroundRGB[index*3+2]) * factor))
		pixels[index] = titleCardRGB555(r, g, b)
	}
	titleR, titleG, titleB := titleCardHexRGB(settings.TitleTextColor, "#FFFFFF")
	titleOutlineR, titleOutlineG, titleOutlineB := titleCardHexRGB(settings.TitleOutlineColor, "#000000")
	subtitleR, subtitleG, subtitleB := titleCardHexRGB(settings.SubtitleTextColor, "#FFFFFF")
	subtitleOutlineR, subtitleOutlineG, subtitleOutlineB := titleCardHexRGB(settings.SubtitleOutlineColor, "#000000")
	titleColour := titleCardRGB555(titleR, titleG, titleB)
	titleOutlineColour := titleCardRGB555(titleOutlineR, titleOutlineG, titleOutlineB)
	subtitleColour := titleCardRGB555(subtitleR, subtitleG, subtitleB)
	subtitleOutlineColour := titleCardRGB555(subtitleOutlineR, subtitleOutlineG, subtitleOutlineB)
	titleScale, titleMaxChars, titleLineHeight := titleCardTextStyle(settings.TitleTextSize)
	subtitleScale, subtitleMaxChars, subtitleLineHeight := titleCardTextStyle(settings.SubtitleTextSize)
	gap := titleCardTypographyGap(settings.TitleTextSize, settings.SubtitleTextSize)
	titleLines := titleCardWrap(settings.Title, titleMaxChars, 2)
	subtitleLines := titleCardWrap(settings.Subtitle, subtitleMaxChars, 2)
	totalHeight := len(titleLines) * titleLineHeight
	if len(subtitleLines) > 0 {
		totalHeight += gap + len(subtitleLines)*subtitleLineHeight
	}
	startY := (titleCardHeight - totalHeight) / 2
	if startY < 10 {
		startY = 10
	}
	y := startY
	for _, line := range titleLines {
		titleCardDrawLine(pixels, line, y, titleScale, settings.TitleAlignment, titleColour, titleOutlineColour, settings.DrawOutline)
		y += titleLineHeight
	}
	if len(subtitleLines) > 0 {
		y += gap
		for _, line := range subtitleLines {
			titleCardDrawLine(pixels, line, y, subtitleScale, settings.SubtitleAlignment, subtitleColour, subtitleOutlineColour, settings.DrawOutline)
			y += subtitleLineHeight
		}
	}
	out := make([]byte, titleCardPixelBytes)
	for index, value := range pixels {
		binary.LittleEndian.PutUint16(out[index*2:index*2+2], value)
	}
	return out, nil
}

func solidTitleCardBackground(value string) []byte {
	r, g, b := titleCardHexRGB(value, "#000000")
	out := make([]byte, titleCardWidth*titleCardHeight*3)
	for index := 0; index < titleCardWidth*titleCardHeight; index++ {
		out[index*3], out[index*3+1], out[index*3+2] = r, g, b
	}
	return out
}

func extractTitleCardBackground(opt ProjectOptions, input ClipInput, settings TitleCardSettings, tempDir string) ([]byte, error) {
	if settings.BackgroundMode == "solid" {
		return solidTitleCardBackground(settings.SolidColor), nil
	}
	when := opt.Start
	if settings.BackgroundMode == "part-frame" {
		when += settings.FrameOffsetSeconds
	}
	end := opt.End
	if end > opt.Start && when >= end {
		when = math.Max(opt.Start, end-0.04)
	}
	outPath := filepath.Join(tempDir, "title-card-background.rgb")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := runCommandContext(ctx, opt.FFmpegPath,
		"-y", "-hide_banner", "-loglevel", "error",
		"-threads", "1", "-ss", fmt.Sprintf("%.6f", when), "-i", input.InputPath,
		"-map", "0:v:0", "-frames:v", "1", "-an", "-sn", "-dn",
		"-vf", makePreviewFilter(opt.FitMode), "-threads", "1",
		"-pix_fmt", "rgb24", "-f", "rawvideo", outPath)
	if err != nil {
		return nil, fmt.Errorf("title-card frame extraction failed: %s", strings.TrimSpace(string(output)))
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, err
	}
	if len(data) != titleCardWidth*titleCardHeight*3 {
		return nil, fmt.Errorf("title-card frame is %d bytes; expected %d", len(data), titleCardWidth*titleCardHeight*3)
	}
	return data, nil
}

func buildTitleCardAsset(backgroundRGB []byte, settings TitleCardSettings) ([]byte, error) {
	pixels, err := renderTitleCardPixels(backgroundRGB, settings)
	if err != nil {
		return nil, err
	}
	asset := make([]byte, titleCardHeaderSize+len(pixels))
	binary.LittleEndian.PutUint32(asset[0:4], titleCardMagic)
	binary.LittleEndian.PutUint16(asset[4:6], 1)
	flags := uint16(0)
	if settings.StartMode == "button" {
		flags |= titleCardFlagWaitForA
	}
	if settings.AllowSkip {
		flags |= titleCardFlagAllowSkip
	}
	if settings.Fade {
		flags |= titleCardFlagFade
	}
	binary.LittleEndian.PutUint16(asset[6:8], flags)
	binary.LittleEndian.PutUint32(asset[8:12], uint32(len(pixels)))
	duration := uint32(math.Round(settings.DurationSeconds * gbaRefresh))
	if duration < 1 {
		duration = 1
	}
	binary.LittleEndian.PutUint32(asset[12:16], duration)
	copy(asset[titleCardHeaderSize:], pixels)
	return asset, nil
}

func prepareTitleCardAsset(opt ProjectOptions, input ClipInput, tempDir string) ([]byte, error) {
	if opt.TitleCard == nil {
		return nil, nil
	}
	settings := *opt.TitleCard
	background, err := extractTitleCardBackground(opt, input, settings, tempDir)
	if err != nil {
		return nil, err
	}
	return buildTitleCardAsset(background, settings)
}

func validateTitleCardProject(project *TitleCardProjectSettings) error {
	if project == nil || !project.Enabled {
		return nil
	}
	if len(project.Parts) > 256 {
		return errors.New("title cards contain too many per-part settings")
	}
	seen := map[int]bool{}
	for _, part := range project.Parts {
		if part.Part < 1 || part.Part > 9999 {
			return errors.New("title-card part numbers must be between 1 and 9999")
		}
		if seen[part.Part] {
			return fmt.Errorf("duplicate title-card settings for part %d", part.Part)
		}
		seen[part.Part] = true
	}
	return nil
}

func cloneTitleCardAsset(asset []byte) []byte {
	return bytes.Clone(asset)
}
