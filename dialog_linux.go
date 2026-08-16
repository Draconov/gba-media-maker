//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runLinuxDialog(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", errDialogCancelled
		}
		return "", fmt.Errorf("%s file dialog failed: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func zenityFilterArgs(filter string) []string {
	groups := parseDialogFilter(filter)
	args := make([]string, 0, len(groups))
	for _, group := range groups {
		args = append(args, "--file-filter="+group.Name+" | "+strings.Join(group.Patterns, " "))
	}
	return args
}

func kdialogFilter(filter string) string {
	groups := parseDialogFilter(filter)
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		parts = append(parts, group.Name+" ("+strings.Join(group.Patterns, " ")+")")
	}
	return strings.Join(parts, "\n")
}

func openFilesDialog(title, filter string, multi bool) ([]string, error) {
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--title=" + title}
		if multi {
			args = append(args, "--multiple", "--separator=\n")
		}
		args = append(args, zenityFilterArgs(filter)...)
		out, err := runLinuxDialog("zenity", args...)
		if err != nil {
			return nil, err
		}
		paths := joinDialogOutput(out)
		if len(paths) == 0 {
			return nil, errors.New("no file was selected")
		}
		return paths, nil
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--title", title}
		if multi {
			args = append(args, "--multiple", "--separate-output")
		}
		args = append(args, "--getopenfilename", ":gba-media-maker", kdialogFilter(filter))
		out, err := runLinuxDialog("kdialog", args...)
		if err != nil {
			return nil, err
		}
		paths := joinDialogOutput(out)
		if len(paths) == 0 {
			return nil, errors.New("no file was selected")
		}
		return paths, nil
	}
	return nil, errDialogUnsupported
}

func saveFileDialog(title, filter, defaultExt, initialName string) (string, error) {
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--save", "--confirm-overwrite", "--title=" + title, "--filename=" + initialName}
		args = append(args, zenityFilterArgs(filter)...)
		out, err := runLinuxDialog("zenity", args...)
		if err != nil {
			return "", err
		}
		return ensureDialogExtension(out, defaultExt), nil
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		start := initialName
		if home, homeErr := os.UserHomeDir(); homeErr == nil && home != "" {
			start = filepath.Join(home, initialName)
		}
		out, err := runLinuxDialog("kdialog", "--title", title, "--getsavefilename", start, kdialogFilter(filter))
		if err != nil {
			return "", err
		}
		return ensureDialogExtension(out, defaultExt), nil
	}
	return "", errDialogUnsupported
}
