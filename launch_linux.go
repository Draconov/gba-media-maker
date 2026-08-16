//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func linuxBrowserCandidates() []string {
	return []string{
		"google-chrome-stable",
		"google-chrome",
		"chromium",
		"chromium-browser",
		"microsoft-edge-stable",
		"microsoft-edge",
		"brave-browser",
		"brave",
	}
}

func launchAppWindow(url string) error {
	for _, name := range linuxBrowserCandidates() {
		browser, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if err := exec.Command(browser, "--app="+url, "--no-first-run").Start(); err == nil {
			return nil
		}
	}
	for _, fallback := range [][]string{{"xdg-open", url}, {"gio", "open", url}} {
		if _, err := exec.LookPath(fallback[0]); err != nil {
			continue
		}
		if err := exec.Command(fallback[0], fallback[1:]...).Start(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("could not open the app window: install a supported browser or xdg-open")
}
