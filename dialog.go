package main

import "errors"

var (
	errDialogCancelled   = errors.New("dialog cancelled")
	errDialogUnsupported = errors.New("native file dialogs are unavailable on this platform")
)

const (
	mediaDialogFilter   = "Media files\x00*.mp4;*.mkv;*.webm;*.avi;*.mov;*.m4v;*.mpeg;*.mpg;*.gif;*.mp3;*.flac;*.wav;*.ogg;*.opus;*.m4a;*.aac;*.png;*.jpg;*.jpeg;*.webp;*.bmp;*.tga;*.tif;*.tiff\x00Video files\x00*.mp4;*.mkv;*.webm;*.avi;*.mov;*.m4v;*.mpeg;*.mpg;*.gif\x00Audio files\x00*.mp3;*.flac;*.wav;*.ogg;*.opus;*.m4a;*.aac;*.wma;*.aiff;*.aif;*.ape\x00Image files\x00*.png;*.jpg;*.jpeg;*.webp;*.bmp;*.tga;*.tif;*.tiff\x00All files\x00*.*\x00\x00"
	projectDialogFilter = "GBA Media Maker project\x00*.gbamedia;*.gbavideo\x00All files\x00*.*\x00\x00"
)
