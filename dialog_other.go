//go:build !windows && !darwin && !linux

package main

func openFilesDialog(title, filter string, multi bool) ([]string, error) {
	return nil, errDialogUnsupported
}

func saveFileDialog(title, filter, defaultExt, initialName string) (string, error) {
	return "", errDialogUnsupported
}
