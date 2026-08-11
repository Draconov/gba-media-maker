//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"
)

func browserCandidates() []string {
	var out []string
	for _, name := range []string{"msedge.exe", "chrome.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			out = append(out, p)
		}
	}
	envs := []string{os.Getenv("PROGRAMFILES(X86)"), os.Getenv("PROGRAMFILES"), os.Getenv("LOCALAPPDATA")}
	rels := []string{filepath.Join("Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join("Google", "Chrome", "Application", "chrome.exe")}
	for _, base := range envs {
		if base == "" {
			continue
		}
		for _, rel := range rels {
			p := filepath.Join(base, rel)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				out = append(out, p)
			}
		}
	}
	return out
}
func launchAppWindow(url string) error {
	seen := map[string]bool{}
	for _, browser := range browserCandidates() {
		key := filepath.Clean(browser)
		if seen[key] {
			continue
		}
		seen[key] = true
		cmd := exec.Command(browser, "--app="+url, "--no-first-run")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not open the app window: %w", err)
	}
	return nil
}
func main() {
	defer func() {
		if r := recover(); r != nil {
			path := filepath.Join(appDirectory(), "GBA Media Maker.log")
			_ = os.WriteFile(path, []byte(fmt.Sprintf("[%s] fatal startup panic: %v\n%s", time.Now().Format(time.RFC3339), r, debug.Stack())), 0644)
		}
	}()
	if err := runWebApp(launchAppWindow); err != nil {
		path := filepath.Join(appDirectory(), "GBA Media Maker.log")
		_ = os.WriteFile(path, []byte(fmt.Sprintf("[%s] startup error: %v\n", time.Now().Format(time.RFC3339), err)), 0644)
	}
}
