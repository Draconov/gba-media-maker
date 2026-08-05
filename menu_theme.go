package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	menuThemeHeaderSize = 64
	menuThemeMagic      = 0x3148544d // "MTH1"
	menuThemeStatic     = 0
	menuThemeShimmer    = 1
	menuThemeFrames     = 2
	menuThemeMaxFrames  = 16
)

type MenuShimmerOptions struct {
	SourceStart int `json:"sourceStart"`
	Count       int `json:"count"`
	Target1     int `json:"target1"`
	Interval1   int `json:"interval1"`
	Target2     int `json:"target2"`
	Interval2   int `json:"interval2"`
	Phases      int `json:"phases"`
}

type MenuThemeOptions struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Kind          string              `json:"kind"`
	Palette       []byte              `json:"palette"`
	Frames        [][]byte            `json:"frames"`
	FrameVBlanks  int                 `json:"frameVBlanks"`
	UIColor       uint16              `json:"uiColor"`
	SelectedColor uint16              `json:"selectedColor"`
	Outline       bool                `json:"outline"`
	OutlineColor  uint16              `json:"outlineColor"`
	Shimmer       *MenuShimmerOptions `json:"shimmer,omitempty"`
}

func (theme *MenuThemeOptions) validate() error {
	if theme == nil {
		return errors.New("menu theme is missing")
	}
	if len(theme.Palette) != 512 {
		return fmt.Errorf("menu palette is %d bytes; expected 512", len(theme.Palette))
	}
	if len(theme.Frames) < 1 || len(theme.Frames) > menuThemeMaxFrames {
		return fmt.Errorf("menu background must contain 1 to %d optimized frames", menuThemeMaxFrames)
	}
	for i, frame := range theme.Frames {
		if len(frame) != frameBytes {
			return fmt.Errorf("menu frame %d is %d bytes; expected %d", i+1, len(frame), frameBytes)
		}
		for _, index := range frame {
			if index >= 250 {
				return fmt.Errorf("menu frame %d uses reserved palette index %d", i+1, index)
			}
		}
	}
	switch theme.Kind {
	case "static":
		if len(theme.Frames) != 1 {
			return errors.New("static menu themes must contain exactly one frame")
		}
	case "frames":
		if theme.FrameVBlanks < 6 || theme.FrameVBlanks > 120 {
			return errors.New("animated menu frame interval must be between 6 and 120 VBlanks")
		}
	case "palette-shimmer":
		if len(theme.Frames) != 1 || theme.Shimmer == nil {
			return errors.New("palette shimmer themes require one frame and shimmer settings")
		}
		s := theme.Shimmer
		if s.Count < 1 || s.Count > 64 || s.SourceStart < 0 || s.SourceStart+s.Count > 250 || s.Target1 < 0 || s.Target1+s.Count > 250 || s.Target2 < 0 || s.Target2+s.Count > 250 {
			return errors.New("palette shimmer ranges are invalid")
		}
		if s.Interval1 < 1 || s.Interval2 < 1 || s.Interval1 > 240 || s.Interval2 > 240 {
			return errors.New("palette shimmer intervals are invalid")
		}
		if s.Phases != 2 && s.Phases != 4 && s.Phases != 8 {
			return errors.New("palette shimmer phases must be 2, 4 or 8")
		}
	default:
		return errors.New("unknown menu background type")
	}
	return nil
}

func appendMenuTheme(rom []byte, theme *MenuThemeOptions) ([]byte, int, error) {
	if err := theme.validate(); err != nil {
		return rom, 0, err
	}
	headerOffset := len(rom)
	rom = append(rom, make([]byte, menuThemeHeaderSize)...)
	paletteOffset := len(rom)
	rom = appendAligned(rom, theme.Palette)
	framesOffset := len(rom)
	for _, frame := range theme.Frames {
		rom = appendAligned(rom, frame)
	}

	kind := uint16(menuThemeStatic)
	if theme.Kind == "palette-shimmer" {
		kind = menuThemeShimmer
	} else if theme.Kind == "frames" {
		kind = menuThemeFrames
	}
	flags := uint16(0)
	if theme.Outline {
		flags |= 1
	}
	header := rom[headerOffset : headerOffset+menuThemeHeaderSize]
	binary.LittleEndian.PutUint32(header[0:4], menuThemeMagic)
	binary.LittleEndian.PutUint16(header[4:6], 1)
	binary.LittleEndian.PutUint16(header[6:8], kind)
	binary.LittleEndian.PutUint32(header[8:12], uint32(paletteOffset))
	binary.LittleEndian.PutUint32(header[12:16], uint32(framesOffset))
	binary.LittleEndian.PutUint16(header[16:18], uint16(len(theme.Frames)))
	binary.LittleEndian.PutUint16(header[18:20], uint16(theme.FrameVBlanks))
	binary.LittleEndian.PutUint16(header[20:22], flags)
	binary.LittleEndian.PutUint16(header[22:24], theme.UIColor)
	binary.LittleEndian.PutUint16(header[24:26], theme.SelectedColor)
	binary.LittleEndian.PutUint16(header[26:28], theme.OutlineColor)
	if theme.Shimmer != nil {
		s := theme.Shimmer
		binary.LittleEndian.PutUint16(header[28:30], uint16(s.SourceStart))
		binary.LittleEndian.PutUint16(header[30:32], uint16(s.Count))
		binary.LittleEndian.PutUint16(header[32:34], uint16(s.Target1))
		binary.LittleEndian.PutUint16(header[34:36], uint16(s.Interval1))
		binary.LittleEndian.PutUint16(header[36:38], uint16(s.Target2))
		binary.LittleEndian.PutUint16(header[38:40], uint16(s.Interval2))
		binary.LittleEndian.PutUint16(header[40:42], uint16(s.Phases))
	}
	binary.LittleEndian.PutUint32(header[44:48], frameBytes)
	binary.LittleEndian.PutUint32(header[48:52], uint32(len(rom)-headerOffset))
	return rom, headerOffset, nil
}
