//go:build darwin

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return `"` + value + `"`
}

func runAppleScript(script string) (string, error) {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && strings.Contains(string(out), "(-128)") {
			return "", errDialogCancelled
		}
		return "", fmt.Errorf("macOS file dialog failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func openFilesDialog(title, filter string, multi bool) ([]string, error) {
	_ = filter
	prompt := appleScriptString(title)
	var script string
	if multi {
		script = `set picked to choose file with prompt ` + prompt + ` with multiple selections allowed` + "\n" +
			`set out to ""` + "\n" +
			`repeat with f in picked` + "\n" +
			`set out to out & POSIX path of f & linefeed` + "\n" +
			`end repeat` + "\n" +
			`return out`
	} else {
		script = `set picked to choose file with prompt ` + prompt + "\n" + `return POSIX path of picked`
	}
	out, err := runAppleScript(script)
	if err != nil {
		return nil, err
	}
	paths := joinDialogOutput(out)
	if len(paths) == 0 {
		return nil, errors.New("no file was selected")
	}
	return paths, nil
}

func saveFileDialog(title, filter, defaultExt, initialName string) (string, error) {
	_ = filter
	script := `set picked to choose file name with prompt ` + appleScriptString(title) + ` default name ` + appleScriptString(initialName) + "\n" +
		`return POSIX path of picked`
	out, err := runAppleScript(script)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", errors.New("no file was selected")
	}
	return ensureDialogExtension(out, defaultExt), nil
}
