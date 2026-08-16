package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"regexp"
	"strings"
)

const (
	defaultAudioArtworkPreset = "preset-01"
	customArtworkDataPrefix   = "data:image/png;base64,"
	maxCustomArtworkPNGBytes  = 2 * 1024 * 1024
)

var audioArtworkPresetPattern = regexp.MustCompile(`^preset-(0[1-9]|1[0-9]|20)$`)

func normalizeAudioArtworkPreset(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !audioArtworkPresetPattern.MatchString(value) {
		return defaultAudioArtworkPreset
	}
	return value
}

func automaticAudioArtworkPreset(seed string) string {
	h := uint32(2166136261)
	for _, r := range seed {
		h ^= uint32(r)
		h *= 16777619
	}
	return fmt.Sprintf("preset-%02d", int(h%20)+1)
}

func audioArtworkPresetPNG(value string) ([]byte, error) {
	preset := normalizeAudioArtworkPreset(value)
	data, err := audioArtworkFS.ReadFile("assets/audio-artwork/" + preset + ".png")
	if err != nil {
		return nil, fmt.Errorf("load audio artwork %s: %w", preset, err)
	}
	return data, nil
}

func imageToNativeRGB555(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	if bounds.Dx() != nativeImageWidth || bounds.Dy() != nativeImageHeight {
		return nil, fmt.Errorf("audio artwork must be %d×%d pixels", nativeImageWidth, nativeImageHeight)
	}
	out := make([]byte, nativeImageBytes)
	for y := 0; y < nativeImageHeight; y++ {
		for x := 0; x < nativeImageWidth; x++ {
			r16, g16, b16, a16 := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Composite transparency over black before quantizing to RGB555.
			r16 = r16 * a16 / 0xffff
			g16 = g16 * a16 / 0xffff
			b16 = b16 * a16 / 0xffff
			r8 := (r16 + 128) / 257
			g8 := (g16 + 128) / 257
			b8 := (b16 + 128) / 257
			r5 := (r8*31 + 127) / 255
			g5 := (g8*31 + 127) / 255
			b5 := (b8*31 + 127) / 255
			value := uint16(r5 | (g5 << 5) | (b5 << 10))
			i := (y*nativeImageWidth + x) * 2
			out[i] = byte(value)
			out[i+1] = byte(value >> 8)
		}
	}
	return out, nil
}

func writePNGArtwork(data []byte, path string) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode audio artwork: %w", err)
	}
	native, err := imageToNativeRGB555(img)
	if err != nil {
		return err
	}
	return os.WriteFile(path, native, 0644)
}

func writePresetAudioArtwork(preset, path string) error {
	data, err := audioArtworkPresetPNG(preset)
	if err != nil {
		return err
	}
	return writePNGArtwork(data, path)
}

func decodeCustomAudioArtworkDataURL(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("custom audio artwork is missing")
	}
	if !strings.HasPrefix(strings.ToLower(value), customArtworkDataPrefix) {
		return nil, errors.New("custom audio artwork must be a PNG image")
	}
	encoded := value[len(customArtworkDataPrefix):]
	if len(encoded) > maxCustomArtworkPNGBytes*2 {
		return nil, errors.New("custom audio artwork is too large")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("custom audio artwork is invalid")
	}
	if len(data) == 0 || len(data) > maxCustomArtworkPNGBytes {
		return nil, errors.New("custom audio artwork is too large")
	}
	return data, nil
}

func writeCustomAudioArtwork(value, path string) error {
	data, err := decodeCustomAudioArtworkDataURL(value)
	if err != nil {
		return err
	}
	return writePNGArtwork(data, path)
}
