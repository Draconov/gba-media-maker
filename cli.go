package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var cliFlagNames = map[string]struct{}{
	"input": {}, "output": {}, "start": {}, "end": {}, "speed": {}, "loop": {}, "seek": {},
	"vblanks": {}, "fit": {}, "audio": {}, "normalize": {}, "limiter": {}, "resume": {},
	"compression": {}, "palette": {}, "dither": {}, "title": {}, "preset": {}, "audio-codec": {},
	"cli": {}, "h": {}, "help": {},
}

func wantsCLI(args []string) bool {
	for _, arg := range args {
		if arg == "cli" || arg == "--cli" || arg == "-cli" {
			return true
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if split := strings.IndexByte(name, '='); split >= 0 {
			name = name[:split]
		}
		if _, ok := cliFlagNames[name]; ok {
			return true
		}
	}
	return false
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "cli" || arg == "--cli" || arg == "-cli" {
			continue
		}
		filtered = append(filtered, arg)
	}

	fs := flag.NewFlagSet("gba-media-maker", flag.ContinueOnError)
	fs.SetOutput(stderr)
	in := fs.String("input", "", "input media (video, audio, or image)")
	out := fs.String("output", "", "output ROM")
	start := fs.Float64("start", 0, "start seconds")
	end := fs.Float64("end", 0, "end seconds")
	speed := fs.Float64("speed", 1, "playback speed")
	loop := fs.Bool("loop", false, "loop when playback ends")
	seek := fs.Int("seek", 5, "seek step: 3, 5, 10 or 15 seconds")
	vblanks := fs.Int("vblanks", 6, "VBlanks per video frame: 4, 5, 6 or 8")
	fit := fs.String("fit", "fit", "fit, crop or stretch")
	audio := fs.String("audio", "mix", "mix, left, right or none")
	normalize := fs.Bool("normalize", false, "normalize audio")
	limiter := fs.Bool("limiter", true, "apply audio limiter")
	resume := fs.Bool("resume", true, "save and resume playback position")
	compression := fs.String("compression", "delta", "delta or none (video)")
	palette := fs.String("palette", "shared", "shared or scene (video)")
	dither := fs.String("dither", "ordered", "off, ordered or error (video)")
	title := fs.String("title", "GBA MEDIA", "ROM title")
	preset := fs.String("preset", "balanced", "best, balanced, long, small, extreme or custom")
	audioCodec := fs.String("audio-codec", "pcm", "pcm, adpcm or auto")
	if err := fs.Parse(filtered); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *in == "" {
		fmt.Fprintln(stderr, "GBA Media Maker CLI: use -input media -output out.gba")
		fs.PrintDefaults()
		return 2
	}
	if *out == "" {
		*out = filepath.Join(filepath.Dir(*in), "media_output.gba")
	}
	ff := locateFFmpeg()
	if ff == "" {
		fmt.Fprintln(stderr, "ffmpeg not found; place it beside GBA Media Maker or install it on PATH")
		return 1
	}
	extreme := *preset == "extreme"
	opt := ConvertOptions{InputPath: *in, OutputPath: *out, FFmpegPath: ff, Start: *start, End: *end, Speed: *speed,
		VBlanks: *vblanks, FitMode: *fit, AudioMode: *audio, Volume: 1, Loop: *loop, RomTitle: *title,
		SeekSeconds: *seek, Normalize: *normalize, Limiter: *limiter, Resume: *resume, Compression: *compression,
		PaletteMode: *palette, DitherMode: *dither, KeyInterval: 30, Preset: *preset, AudioCodec: *audioCodec,
		ExtremeOptimization: extreme, SmartTargetMiB: 32, SmartPriority: "balanced"}
	res, err := convertVideo(opt, func(p int, s string) { fmt.Fprintf(stdout, "[%3d%%] %s\n", p, s) })
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Created %s (%d media frames, %d bytes)\n", res.OutputPath, res.FrameCount, res.PaddedSize)
	return 0
}

func runCLIFromOSArgs() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}
