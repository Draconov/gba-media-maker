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
	Title              string  `json:"title"`
	Subtitle           string  `json:"subtitle"`
	BackgroundMode     string  `json:"backgroundMode"` // part-first-frame, part-frame, solid
	FrameOffsetSeconds float64 `json:"frameOffsetSeconds"`
	Darkness           int     `json:"darkness"` // 0..90 percent
	SolidColor         string  `json:"solidColor"`
	TextColor          string  `json:"textColor"`
	OutlineColor       string  `json:"outlineColor"`
	DrawOutline        bool    `json:"drawOutline"`
	Alignment          string  `json:"alignment"` // left, center, right
	TextSize           string  `json:"textSize"`  // large, medium, small
	StartMode          string  `json:"startMode"` // button, timer
	DurationSeconds    float64 `json:"durationSeconds"`
	AllowSkip          bool    `json:"allowSkip"`
	Fade               bool    `json:"fade"`
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
		Title:              sanitizeTitleCardText(base, 36),
		Subtitle:           "Part {part}",
		BackgroundMode:     "part-first-frame",
		FrameOffsetSeconds: 0,
		Darkness:           50,
		SolidColor:         "#000000",
		TextColor:          "#FFFFFF",
		OutlineColor:       "#000000",
		DrawOutline:        true,
		Alignment:          "center",
		TextSize:           "large",
		StartMode:          "button",
		DurationSeconds:    3,
		AllowSkip:          true,
		Fade:               true,
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
	value = strings.ToUpper(value)
	var clean strings.Builder
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			clean.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '.' || r == ':' || r == '!' || r == '?' || r == '/' || r == '&' || r == '(' || r == ')' || r == '+':
			clean.WriteRune(r)
		default:
			clean.WriteByte(' ')
		}
	}
	text := strings.Join(strings.Fields(clean.String()), " ")
	if maximum > 0 && len(text) > maximum {
		text = strings.TrimSpace(text[:maximum])
	}
	return text
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

var titleCardGlyphs = map[byte]uint16{
	'0': 0x7B6F, '1': 0x2C97, '2': 0x73E7, '3': 0x73CF, '4': 0x5BC9,
	'5': 0x79CF, '6': 0x79EF, '7': 0x7292, '8': 0x7BEF, '9': 0x7BCF,
	'A': 0x2BED, 'B': 0x6BAE, 'C': 0x7927, 'D': 0x6B6E, 'E': 0x79E7,
	'F': 0x79E4, 'G': 0x79AF, 'H': 0x5BED, 'I': 0x7497, 'J': 0x124E,
	'K': 0x5D6D, 'L': 0x4927, 'M': 0x5FE9, 'N': 0x5F6D, 'O': 0x7B6F,
	'P': 0x7BE4, 'Q': 0x7B7B, 'R': 0x7BED, 'S': 0x79CF, 'T': 0x7492,
	'U': 0x5B6F, 'V': 0x5B6A, 'W': 0x5BFD, 'X': 0x5AAD, 'Y': 0x5A92,
	'Z': 0x72A7, ' ': 0,
	'-': 0x01C0, '_': 0x0007, '.': 0x0002, ':': 0x0410, '!': 0x2492,
	'?': 0x72C2, '/': 0x1248, '+': 0x05D0, '(': 0x2488, ')': 0x1112,
	'&': 0x2AAE,
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
		for len(word) > maxChars {
			if current != "" {
				lines = append(lines, current)
				current = ""
				if len(lines) >= maxLines {
					return lines
				}
			}
			lines = append(lines, word[:maxChars])
			word = word[maxChars:]
			if len(lines) >= maxLines {
				return lines
			}
		}
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if len(candidate) <= maxChars {
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
	if len(line) > 0 {
		width = len(line)*4*scale - scale
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
	x := titleCardTextX(line, scale, alignment)
	if drawOutline {
		radius := 1
		if scale >= 3 {
			radius = 2
		}
		offsets := [][2]int{{-radius, 0}, {radius, 0}, {0, -radius}, {0, radius}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}}
		for _, offset := range offsets {
			for index := 0; index < len(line); index++ {
				titleCardDrawGlyph(pixels, x+index*4*scale+offset[0], y+offset[1], scale, titleCardGlyphs[line[index]], outline)
			}
		}
	}
	for index := 0; index < len(line); index++ {
		titleCardDrawGlyph(pixels, x+index*4*scale, y, scale, titleCardGlyphs[line[index]], colour)
	}
}

func titleCardTypography(size string) (titleScale, subtitleScale, titleMaxChars, subtitleMaxChars, titleLineHeight, subtitleLineHeight, gap int) {
	switch size {
	case "medium":
		return 3, 2, 19, 29, 18, 12, 8
	case "small":
		return 2, 1, 29, 59, 12, 6, 6
	default:
		return 4, 3, 14, 19, 24, 18, 10
	}
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
	tr, tg, tb := titleCardHexRGB(settings.TextColor, "#FFFFFF")
	or, og, ob := titleCardHexRGB(settings.OutlineColor, "#000000")
	textColour := titleCardRGB555(tr, tg, tb)
	outlineColour := titleCardRGB555(or, og, ob)
	titleScale, subtitleScale, titleMaxChars, subtitleMaxChars, titleLineHeight, subtitleLineHeight, gap := titleCardTypography(settings.TextSize)
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
		titleCardDrawLine(pixels, line, y, titleScale, settings.Alignment, textColour, outlineColour, settings.DrawOutline)
		y += titleLineHeight
	}
	if len(subtitleLines) > 0 {
		y += gap
		for _, line := range subtitleLines {
			titleCardDrawLine(pixels, line, y, subtitleScale, settings.Alignment, textColour, outlineColour, settings.DrawOutline)
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
