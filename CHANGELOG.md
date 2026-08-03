# Changelog

## [0.10.1] - 2026-08-02

### Added

- Selectable video resolution at project-default and per-clip scope
- Efficient 120×80, Enhanced 180×120, and Native 240×160 modes
- Resolution-aware live previews, size estimates, project files, and conversion results
- Smart-optimizer resolution fallback from Native to Enhanced to Efficient
- Hybrid-codec tests for partial 8×8 tiles at 180×120

### Changed

- Mode 4 DMA uploads, affine matrices, frame buffers, HUD drawing, seeking, and menu thumbnails now use each clip's descriptor dimensions
- The Best quality preset now starts at Enhanced 180×120; the default remains Efficient 120×80
- GBV6 remains the ROM format because its descriptors already contain source width, source height, and frame-byte fields

## [0.10.0] - 2026-08-02

### Added

- GBV6 ROM format with 128-byte clip descriptors and codec/audio statistics
- Hybrid frame codec that automatically chooses raw, changed-byte, repeat, or changed-8×8-tile records per frame
- Adaptive keyframes for strong scene changes and bounded seek reconstruction
- Blocked IMA ADPCM audio option, enabled by default, using about half the storage of PCM
- PCM audio compatibility option
- Codec comparison results in the conversion UI and command-line harness
- Tile-based Mode 0 collection menu using separate gradient, text, thumbnail, and OBJ layers
- Sprite-based mute, volume, seek, loop, and menu-arrow feedback
- Double-buffered ADPCM decoder with timer-driven Direct Sound DMA

### Changed

- Mode 4 video is uploaded at 120×80 with DMA3 and enlarged to 240×160 using BG2 affine hardware instead of software 2×2 pixel expansion
- Collection-menu blinking and selection updates now use tilemap/OAM changes rather than full bitmap redraws
- Expanded the embedded player template to 44 KiB and moved GBV6 metadata/assets to `0xAF00`/`0xB000`
- Video palettes now reserve entries 246–255 for menu and HUD colours
- Balanced and size-focused presets now use hybrid video plus ADPCM audio

## [0.9.1] - 2026-08-02

### Added

- Blue-gradient clip-selection menu with white text and a yellow selected item
- Yellow pixel arrow that blinks approximately every 0.4 seconds
- Status line showing clip count, combined duration, and current selection index
- One-, two-, or three-column collection layouts with paging beyond 30 clips
- Optional right-side selected-video thumbnail
- Left/Right navigation between menu columns or pages

## [0.9.0] - 2026-08-01

### Added

- Per-video settings with project-default inheritance for title, trim, speed, framing, audio, volume, loop, palette, and dithering
- Drag-and-drop clip ordering plus accessible Move up and Move down controls
- `.gbavideo` save/open support using source file paths and relinking for moved files
- Timeline scrubbing with playhead, start/end handles, thumbnails, fine adjustment, and quick jumps
- Smart 32 MiB optimizer with a reviewable proposal before applying changes
- Live GBA-font clip-title preview, character validation, duplicate-title warnings, and filename reset
- Optional selected-clip thumbnail in menu ROMs
- Playlist previous/next controls on `SELECT + Left/Right`

### Changed

- Fit with bars is now the default framing mode
- Multi-clip conversion now resolves trim, framing, audio, palette, dithering, and looping independently for each clip

## [0.8.0] - 2026-07-31

### UI refinement update

- Tightened the volume popup so only one black pixel remains after the final digit.
- Added the user-provided app icon as the browser app-window icon and repository ICO asset.
- Added automatic system light/dark mode palettes.
- Centered the drag-and-drop screen at every window size and removed its redundant header and film emoji.
- Restored named frame-rate choices: Smooth, Balanced, Classic, and Compact.
- ROM titles now start from the source filename and are cropped to the 12-character GBA limit.
- Fixed End frame previews by seeking to the final decodable frame rather than exact EOF.
- Simplified the trim label to “End (blank = full video)”.

### Player controls

- Added hold-to-seek with an immediate first jump and repeated jumps approximately every 0.4 seconds
- Added configurable 3, 5, 10, and 15-second seek steps; the selected value appears in the seek popup
- Added D-pad Up/Down volume control with 0%, 50%, and 100% Direct Sound levels
- Kept `SELECT` for mute/unmute
- Moved quick HUD show/hide to the `L + R` combination
- Added frame-by-frame D-pad Left/Right movement while paused
- Changed `START` to cycle hidden, time-only, and full HUD modes
- Added current frame number to the full HUD
- Added a `START + SELECT` controls-help screen
- Kept `B` as restart

### Converter application

- Added Best quality, Balanced, Long video, Smallest ROM, and Custom output presets
- Added cartridge-size, frame-count, and approximate duration-limit estimates
- Added start/end picture previews using crop, fit-with-bars, or stretch processing
- Added audio-channel previews
- Added optional loudness normalization and limiter
- Added multi-file drag and drop
- Added batch output as separate ROMs in a ZIP
- Added multiple clips in one ROM with a startup selection menu

### Encoding

- Upgraded generated ROM metadata from GBV4 to GBV5
- Expanded the embedded player template from 8 KiB to 16 KiB
- Added keyframe/delta frame compression with automatic raw-frame fallback
- Added frame indexes for compressed seeking and reconstruction
- Added shared and scene-change palette modes
- Added off, ordered, and error-diffusion dithering modes
- Added per-frame palette-index synchronization
- Added optional SRAM playback-position saving and resume prompts
- Added per-clip assets, settings, titles, audio, palettes, and seek tables for collection ROMs

## [0.7.0] - 2026-07-31

### Added

- D-pad Left/Right seeking alongside L/R
- Mute feedback, playback HUD, progress line, seek feedback, and loop indicator
- Per-frame audio seek offsets

### Fixed

- Fixed audio/video drift and end-of-video silence by scheduling frames from a dedicated 16,384 Hz hardware playback clock.
- Audio conversion now normalizes source timestamps with asynchronous resampling, improving files with timestamp gaps or overlaps.
- Paused frame stepping now repositions the audio stream to the selected frame before playback resumes.

### Changed

- Audited the repository and removed unused converter helpers and constants
- Added strict loopback host checks, anti-CSRF token checks for local POST requests, and browser security headers
- Moved the session token from inline JavaScript into an escaped HTML meta element, allowing a strict script policy
- Parallelized RGB555 palette lookup generation and reduced allocations in scene detection and error-diffusion dithering
- Quantized independent video frames in parallel while preserving byte-for-byte ROM output
- Optimized delta-frame encoding and binary index/palette writing without changing ROM output
- Reduced the embedded browser icon from 69 KiB to 26 KiB without changing the Windows executable icon
- Avoided duplicate GitHub Actions work by letting the Release workflow validate pushes to main while CI handles pull requests and other branches
- Removed the runtime FFmpeg executable downloader; portable releases now bundle a pinned, verified FFmpeg binary.
- Moved the browser UI from a large inline Go string into readable embedded files under `web/` to reduce heuristic antivirus false positives and improve auditability.
- Automatic GitHub Actions builds now publish/update a normal stable release using the exact VERSION tag, without build-number suffixes or a prerelease badge.
- Added a red remove button for each loaded video in the editor, so clips can be removed without resetting the whole project.
- Replaced the clip-menu font chevron with a clear pixel-art right arrow
- Menu-ROM clips now return to the clip menu automatically when playback finishes
- In a menu ROM, pressing B during playback now returns to the clip menu instead of restarting the current clip
- Embedded the custom icon directly into the Windows executable so Explorer displays it on `GBA Video Maker.exe`.
- Replaced the app icon with the latest user-provided icon artwork.
- The app icon now uses the transparent-background version so it looks clean in the app window
- When multiple videos are loaded, you can now choose between **One ROM — play clips in order**, **One ROM — clip menu**, or **Separate ROMs in ZIP**

- Refined seek-popup spacing and loop-icon placement
- Upgraded generated ROM metadata to GBV4

## [0.6.0] - 2026-07-31

- Added L/R seeking and synchronized audio restart
- Added reproducible LLVM player builds

## [0.5.0] - 2026-07-30

- Added the portable local-web Windows GUI and complete first conversion workflow
