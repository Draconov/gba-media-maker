//go:build darwin || linux

package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

func main() {
	if wantsCLI(os.Args[1:]) {
		runCLIFromOSArgs()
		return
	}
	defer func() {
		if r := recover(); r != nil {
			writeStartupLog(fmt.Sprintf("[%s] fatal startup panic: %v\n%s", time.Now().Format(time.RFC3339), r, debug.Stack()))
		}
	}()
	if err := runWebApp(launchAppWindow); err != nil {
		writeStartupLog(fmt.Sprintf("[%s] startup error: %v\n", time.Now().Format(time.RFC3339), err))
	}
}
