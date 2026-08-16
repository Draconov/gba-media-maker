//go:build darwin || linux

package main

import "os"

func writeStartupLog(message string) {
	_ = os.WriteFile(diagnosticLogPath(), []byte(message), 0o644)
}
