//go:build !windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	in := flag.String("input", "", "input video")
	out := flag.String("output", "", "output ROM")
	start := flag.Float64("start", 0, "start seconds")
	end := flag.Float64("end", 0, "end seconds")
	speed := flag.Float64("speed", 1, "playback speed")
	loop := flag.Bool("loop", false, "loop when playback ends")
	seek := flag.Int("seek", 5, "seek step: 3, 5, 10 or 15 seconds")
	vblanks := flag.Int("vblanks", 6, "VBlanks per frame: 4, 5, 6 or 8")
	fit := flag.String("fit", "fit", "fit, crop or stretch")
	audio := flag.String("audio", "mix", "mix, left, right or none")
	normalize := flag.Bool("normalize", false, "normalize audio")
	limiter := flag.Bool("limiter", true, "apply audio limiter")
	resume := flag.Bool("resume", true, "save and resume playback position")
	compression := flag.String("compression", "hybrid", "hybrid, delta or none")
	audioCodec := flag.String("audio-codec", "adpcm", "adpcm or pcm")
	palette := flag.String("palette", "shared", "shared or scene")
	dither := flag.String("dither", "ordered", "off, ordered or error")
	resolution := flag.String("resolution", "efficient", "efficient, enhanced or native")
	title := flag.String("title", "GBA VIDEO", "ROM title")
	flag.Parse()
	if *in == "" {
		fmt.Println("Linux test harness: use -input video -output out.gba")
		return
	}
	if *out == "" {
		*out = filepath.Join(filepath.Dir(*in), "test_output.gba")
	}
	ff := commandExists("ffmpeg")
	if ff == "" {
		fmt.Fprintln(os.Stderr, "ffmpeg not found")
		os.Exit(1)
	}
	opt := ConvertOptions{InputPath: *in, OutputPath: *out, FFmpegPath: ff, Start: *start, End: *end, Speed: *speed, VBlanks: *vblanks, FitMode: *fit, AudioMode: *audio, Volume: 1, Loop: *loop, RomTitle: *title, SeekSeconds: *seek, Normalize: *normalize, Limiter: *limiter, Resume: *resume, Compression: *compression, AudioCodec: *audioCodec, PaletteMode: *palette, DitherMode: *dither, Resolution: *resolution, KeyInterval: 30}
	res, err := convertVideo(opt, func(p int, s string) { fmt.Printf("[%3d%%] %s\n", p, s) })
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s (%d frames, %.2f fps, %d bytes; video %.1f%% of raw; audio %.1f%% of PCM; raw=%d byte-delta=%d tile-delta=%d repeat=%d)\n", res.OutputPath, res.FrameCount, res.FPS, res.PaddedSize, 100*float64(res.CompressedBytes)/float64(max64(res.UncompressedBytes, 1)), 100*float64(res.AudioStoredBytes)/float64(max64(res.AudioPCMBytes, 1)), res.CodecStats.RawFrames, res.CodecStats.ByteDeltaFrames, res.CodecStats.TileDeltaFrames, res.CodecStats.RepeatFrames)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
