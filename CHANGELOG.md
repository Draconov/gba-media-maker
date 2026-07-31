# Changelog

All notable changes to this project are documented here.

## [0.6.0] - 2026-07-31

### Added

- `L` seeks backward by approximately five seconds during playback or from the end screen
- `R` seeks forward by approximately five seconds
- Per-frame audio seek offsets keep sound aligned after seeking
- Freestanding ARM7TDMI player source and reproducible LLVM build scripts

### Changed

- Generated ROM metadata format upgraded from GBV2 to GBV3
- GitHub Actions use Node.js 24-compatible action releases

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
