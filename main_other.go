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
	speed := flag.Float64("speed", 1, "speed")
	loop := flag.Bool("loop", false, "loop when playback ends")
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
	opt := ConvertOptions{InputPath: *in, OutputPath: *out, FFmpegPath: ff, Start: *start, End: *end, Speed: *speed, VBlanks: 6, FitMode: "crop", AudioMode: "mix", Volume: 1, Loop: *loop, RomTitle: "PORTABLE"}
	res, err := convertVideo(opt, func(p int, s string) { fmt.Printf("[%3d%%] %s\n", p, s) })
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s (%d frames, %.2f fps, %d bytes)\n", res.OutputPath, res.FrameCount, res.FPS, res.PaddedSize)
}
