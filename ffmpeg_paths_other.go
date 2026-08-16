//go:build !windows && !darwin && !linux

package main

func extraFFmpegCandidates() []string { return nil }
