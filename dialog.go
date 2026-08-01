package main

import "errors"

var (
	errDialogCancelled   = errors.New("dialog cancelled")
	errDialogUnsupported = errors.New("native file dialogs are unavailable on this platform")
)

const (
	videoDialogFilter   = "Video files\x00*.mp4;*.mkv;*.webm;*.avi;*.mov;*.m4v\x00All files\x00*.*\x00\x00"
	projectDialogFilter = "GBA Video Maker project\x00*.gbavideo\x00All files\x00*.*\x00\x00"
)
