//go:build !windows

package main

import "errors"

func openFilesDialog(title, filter string, multi bool) ([]string, error) {
	return nil, errDialogUnsupported
}

func saveFileDialog(title, filter, defaultExt, initialName string) (string, error) {
	return "", errDialogUnsupported
}

var _ = errors.New
