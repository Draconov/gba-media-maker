//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func macBrowserCandidates() []string {
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, "Applications", "Google Chrome.app", "Contents", "MacOS", "Google Chrome"),
			filepath.Join(home, "Applications", "Chromium.app", "Contents", "MacOS", "Chromium"),
		)
	}
	candidates = append(candidates,
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	)
	return candidates
}

func launchAppWindow(url string) error {
	for _, browser := range macBrowserCandidates() {
		if st, err := os.Stat(browser); err != nil || st.IsDir() {
			continue
		}
		cmd := exec.Command(browser, "--app="+url, "--no-first-run")
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	if err := exec.Command("open", url).Start(); err != nil {
		return fmt.Errorf("could not open the app window: %w", err)
	}
	return nil
}
