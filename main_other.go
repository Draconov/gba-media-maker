//go:build !windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	in := flag.String("input", "", "input media (video, audio, or image)")
	out := flag.String("output", "", "output ROM")
	start := flag.Float64("start", 0, "start seconds")
	end := flag.Float64("end", 0, "end seconds")
	speed := flag.Float64("speed", 1, "playback speed")
	loop := flag.Bool("loop", false, "loop when playback ends")
	seek := flag.Int("seek", 5, "seek step: 3, 5, 10 or 15 seconds")
	vblanks := flag.Int("vblanks", 6, "VBlanks per video frame: 4, 5, 6 or 8")
	fit := flag.String("fit", "fit", "fit, crop or stretch")
	audio := flag.String("audio", "mix", "mix, left, right or none")
	normalize := flag.Bool("normalize", false, "normalize audio")
	limiter := flag.Bool("limiter", true, "apply audio limiter")
	resume := flag.Bool("resume", true, "save and resume playback position")
	compression := flag.String("compression", "delta", "delta or none (video)")
	palette := flag.String("palette", "shared", "shared or scene (video)")
	dither := flag.String("dither", "ordered", "off, ordered or error (video)")
	title := flag.String("title", "GBA MEDIA", "ROM title")
	preset := flag.String("preset", "balanced", "best, balanced, long, small, extreme or custom")
	audioCodec := flag.String("audio-codec", "pcm", "pcm, adpcm or auto")
	flag.Parse()
	if *in == "" {
		fmt.Println("GBA Media Maker CLI: use -input media -output out.gba")
		return
	}
	if *out == "" {
		*out = filepath.Join(filepath.Dir(*in), "media_output.gba")
	}
	ff := locateFFmpeg()
	if ff == "" {
		fmt.Fprintln(os.Stderr, "ffmpeg not found")
		os.Exit(1)
	}
	extreme := *preset == "extreme"
	opt := ConvertOptions{InputPath: *in, OutputPath: *out, FFmpegPath: ff, Start: *start, End: *end, Speed: *speed,
		VBlanks: *vblanks, FitMode: *fit, AudioMode: *audio, Volume: 1, Loop: *loop, RomTitle: *title,
		SeekSeconds: *seek, Normalize: *normalize, Limiter: *limiter, Resume: *resume, Compression: *compression,
		PaletteMode: *palette, DitherMode: *dither, KeyInterval: 30, Preset: *preset, AudioCodec: *audioCodec,
		ExtremeOptimization: extreme, SmartTargetMiB: 32, SmartPriority: "balanced"}
	res, err := convertVideo(opt, func(p int, s string) { fmt.Printf("[%3d%%] %s\n", p, s) })
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s (%d media frames, %d bytes)\n", res.OutputPath, res.FrameCount, res.PaddedSize)
}
