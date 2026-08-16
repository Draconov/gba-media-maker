//go:build darwin

package main

import "path/filepath"

func extraFFmpegCandidates() []string {
	appDir := appDirectory()
	return []string{
		filepath.Clean(filepath.Join(appDir, "..", "Resources", "ffmpeg")),
		"/opt/homebrew/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/opt/local/bin/ffmpeg",
		"/usr/bin/ffmpeg",
	}
}
