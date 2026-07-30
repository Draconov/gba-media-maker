# Changelog

All notable changes to this project are documented here.

## [0.5.0] - 2026-07-30

### Added

- Portable Windows GUI served locally by the Go executable
- Drag-and-drop and native browser file selection
- Trimming, playback speed, framing, audio-channel, volume, FPS, looping, and ROM-title controls
- Local FFmpeg download with pinned SHA-256 verification
- ROM-size estimation and downloadable `.gba` output
- End-to-end upload, inspection, conversion, and download test

### Changed

- Replaced the earlier custom Win32 GUI and callback implementation with a browser-based local interface
- Conversion now starts from a minimal centered video-selection screen

### Fixed

- Crashes caused by custom Win32 callback and file-dialog code
