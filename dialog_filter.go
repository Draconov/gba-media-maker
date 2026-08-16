package main

import (
	"path/filepath"
	"strings"
)

type dialogFilterGroup struct {
	Name     string
	Patterns []string
}

func parseDialogFilter(filter string) []dialogFilterGroup {
	parts := strings.Split(filter, "\x00")
	groups := make([]dialogFilterGroup, 0, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		name := strings.TrimSpace(parts[i])
		patterns := strings.TrimSpace(parts[i+1])
		if name == "" || patterns == "" {
			break
		}
		items := strings.FieldsFunc(patterns, func(r rune) bool { return r == ';' || r == ',' })
		clean := make([]string, 0, len(items))
		for _, item := range items {
			if item = strings.TrimSpace(item); item != "" {
				clean = append(clean, item)
			}
		}
		if len(clean) > 0 {
			groups = append(groups, dialogFilterGroup{Name: name, Patterns: clean})
		}
	}
	return groups
}

func joinDialogOutput(output string) []string {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}

func ensureDialogExtension(path, ext string) string {
	if path == "" || ext == "" || filepath.Ext(path) != "" {
		return path
	}
	return path + "." + strings.TrimPrefix(ext, ".")
}
