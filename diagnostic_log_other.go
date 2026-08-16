//go:build !windows && !darwin && !linux

package main

import "path/filepath"

func diagnosticLogPath() string {
	return filepath.Join(appDirectory(), "GBA Media Maker.log")
}
