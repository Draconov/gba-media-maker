# Changelog

All notable changes to this project are documented here.

## [0.7.0] - 2026-07-31

### Added

- `D-pad Left` and `D-pad Right` now mirror `L` and `R` for five-second seeking
- `SELECT` toggles audio mute without losing playback position
- `D-pad Up` toggles a playback HUD with current time, total duration, and a progress line
- Temporary backward/forward arrow indicators after `L` or `R` seeking
- Muted-audio badge shown briefly after mute-related UI activity

### Changed

- The seek popup is tighter now: the number and arrow have only a 2-pixel gap, so the box is less intrusive
- The loop icon now uses the latest provided 8x8 arrow exactly
- The HUD now leaves a black 1-pixel row above and below the loop icon
- The progress line now spans nearly the full screen width, with about 5-pixel margins on both sides
- The loop icon now sits above the progress line instead of beside it
- The seek popup uses a filled black box again, trimmed to about a 2-pixel margin around the number and arrow
- Loop-enabled exports now show a small yellow repeat/loop icon using the exact 7x6 arrow pattern you provided, positioned above the progress line at the right side of the playback HUD
- Mute/unmute now shows only a compact badge instead of also opening the playback HUD
- The mute badge is tighter, with a red `X` for muted audio and a green check-style `V` for unmuted audio, while the playback progress line and seek arrows use yellow again
- Refined the muted-audio badge to use a simple, clearly recognizable X
- Temporary seek and mute feedback now disappear after about 0.4 seconds, and automatic playback-HUD popups are shorter overall
- Playback progress remains visible while paused
- Five palette entries are now reserved for consistently readable overlays, allowing dedicated red and green status icons
- Generated ROM metadata format upgraded from GBV3 to GBV4
- Embedded player metadata moved within the 8 KiB template to make room for the HUD renderer

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
