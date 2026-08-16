//go:build linux

package main

func extraFFmpegCandidates() []string {
	return []string{
		"/usr/local/bin/ffmpeg",
		"/usr/bin/ffmpeg",
		"/snap/bin/ffmpeg",
	}
}
