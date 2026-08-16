//go:build darwin || linux

package main

import (
	"os"
	"path/filepath"
)

func diagnosticLogPath() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "GBA Media Maker")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return filepath.Join(os.TempDir(), "GBA Media Maker.log")
	}
	return filepath.Join(dir, "GBA Media Maker.log")
}
