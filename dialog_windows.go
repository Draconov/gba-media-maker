//go:build windows

package main

import (
	"errors"
	"path/filepath"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	ofnAllowMultiSelect = 0x00000200
	ofnExplorer         = 0x00080000
	ofnFileMustExist    = 0x00001000
	ofnPathMustExist    = 0x00000800
	ofnOverwritePrompt  = 0x00000002
	ofnNoChangeDir      = 0x00000008
)

type openFileNameW struct {
	StructSize      uint32
	Owner           uintptr
	Instance        uintptr
	Filter          *uint16
	CustomFilter    *uint16
	MaxCustomFilter uint32
	FilterIndex     uint32
	File            *uint16
	MaxFile         uint32
	FileTitle       *uint16
	MaxFileTitle    uint32
	InitialDir      *uint16
	Title           *uint16
	Flags           uint32
	FileOffset      uint16
	FileExtension   uint16
	DefaultExt      *uint16
	CustomData      uintptr
	Hook            uintptr
	TemplateName    *uint16
	Reserved        uintptr
	Reserved32      uint32
	FlagsEx         uint32
}

var (
	comdlg32           = syscall.NewLazyDLL("comdlg32.dll")
	getOpenFileNameW   = comdlg32.NewProc("GetOpenFileNameW")
	getSaveFileNameW   = comdlg32.NewProc("GetSaveFileNameW")
	commDlgExtendedErr = comdlg32.NewProc("CommDlgExtendedError")
)

func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func utf16Buffer(s string) []uint16 {
	return utf16.Encode([]rune(s))
}

func dialogError() error {
	code, _, _ := commDlgExtendedErr.Call()
	if code == 0 {
		return errDialogCancelled
	}
	return syscall.Errno(code)
}

func parseMultiSelect(buf []uint16) []string {
	parts := make([]string, 0, 8)
	start := 0
	for i, v := range buf {
		if v != 0 {
			continue
		}
		if i == start {
			break
		}
		parts = append(parts, string(utf16.Decode(buf[start:i])))
		start = i + 1
	}
	if len(parts) <= 1 {
		return parts
	}
	base := parts[0]
	out := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		out = append(out, filepath.Join(base, name))
	}
	return out
}

func openFilesDialog(title, filter string, multi bool) ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	buf := make([]uint16, 65536)
	filterBuffer := utf16Buffer(filter)
	flags := uint32(ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir)
	if multi {
		flags |= ofnAllowMultiSelect
	}
	ofn := openFileNameW{
		StructSize:  uint32(unsafe.Sizeof(openFileNameW{})),
		Filter:      &filterBuffer[0],
		FilterIndex: 1,
		File:        &buf[0],
		MaxFile:     uint32(len(buf)),
		Title:       utf16Ptr(title),
		Flags:       flags,
	}
	ok, _, _ := getOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ok == 0 {
		return nil, dialogError()
	}
	paths := parseMultiSelect(buf)
	if len(paths) == 0 {
		return nil, errors.New("no file was selected")
	}
	return paths, nil
}

func saveFileDialog(title, filter, defaultExt, initialName string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	buf := make([]uint16, 32768)
	filterBuffer := utf16Buffer(filter)
	name := utf16.Encode([]rune(initialName))
	if len(name) >= len(buf) {
		name = name[:len(buf)-1]
	}
	copy(buf, name)
	ofn := openFileNameW{
		StructSize:  uint32(unsafe.Sizeof(openFileNameW{})),
		Filter:      &filterBuffer[0],
		FilterIndex: 1,
		File:        &buf[0],
		MaxFile:     uint32(len(buf)),
		Title:       utf16Ptr(title),
		Flags:       ofnExplorer | ofnPathMustExist | ofnOverwritePrompt | ofnNoChangeDir,
		DefaultExt:  utf16Ptr(defaultExt),
	}
	ok, _, _ := getSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ok == 0 {
		return "", dialogError()
	}
	for i, v := range buf {
		if v == 0 {
			return string(utf16.Decode(buf[:i])), nil
		}
	}
	return "", errors.New("selected path is too long")
}
