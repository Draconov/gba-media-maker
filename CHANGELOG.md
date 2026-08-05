# Changelog

## [0.9.0 Web Feature Parity] - 2026-08-05

### Added
- Brought the public browser converter up to the Windows application's current workflow.
- Added automatic and manually configured long-video splitting with size targets, `MM:SS` duration caps, chapter-aware cuts, numbered ROMs, `PARTS.txt`, and optional filename / `PART N` title screens.
- Added estimated output size and ROM-part count, detailed split progress, and automatic 32 MiB overflow fallback.
- Added interrupted split recovery through IndexedDB and retained completed parts when a later part fails.
- Added `.gbavideo` project save/open with browser-safe source-file relinking.
- Added a selected-clip timeline, preview thumbnails, GBA-font title preview, selected-channel audio preview, and a 32 MiB optimizer.

### Changed
- The web app now dynamically exposes **Single ROM** for one source and playlist, menu, or batch outputs for collections.
- Browser and Windows ROM assembly now both support split-part title-screen metadata.

## [0.9.0] - 2026-08-05

### Added

#### Long-video conversion

- Added automatic fallback from a single ROM to sequential numbered ROMs in a ZIP when the video cannot safely fit on one cartridge.
- Added pre-conversion split estimates such as `Estimated output: N ROM parts`.
- Added long-conversion progress showing `Part N of approximately M` and the current source position.
- Added a 1–32 MiB target-size slider with 20 MiB, 30 MiB, and Maximum shortcuts.
- Added an optional maximum source duration per ROM part. The field accepts `MM:SS`, such as `1:05`; `0` keeps the duration automatic.
- Added chapter-aware splitting that prefers a nearby earlier chapter boundary when chapter metadata is available.
- Added optional split-ROM title screens showing the sanitized source filename and `PART N`.
- Added adaptive part sizing: oversized candidates are shortened and underfilled candidates are extended before being finalized.
- Added `PARTS.txt` to split ZIPs with part filenames, source-time ranges, encoded-data sizes, and cartridge sizes.
- Added persistent interrupted-conversion recovery. Completed parts are retained and reused when the same job is started again.
- Added early detection when the selected PCM audio alone would exceed the safe cartridge budget.
- Added a disk-backed desktop workflow suitable for long sources without retaining the entire conversion in browser memory.

#### Project workflow

- Added per-video settings with project-default inheritance for title, trim, speed, framing, audio, volume, looping, palette, and dithering.
- Added drag-and-drop clip ordering and accessible Move up and Move down controls.
- Added `.gbavideo` project save/open support using source paths, with relinking for moved files.
- Added timeline scrubbing with a playhead, start/end handles, thumbnails, fine adjustment, and quick jumps.
- Added a smart 32 MiB optimizer with a reviewable proposal before changes are applied.
- Added live GBA-font clip-title previews, character validation, duplicate-title warnings, and filename reset.

#### Collection ROMs and playback

- Added playlist previous/next controls using `SELECT + Left/Right`.
- Added a collection menu supporting up to three columns, total clip duration, current-selection status, D-pad column navigation, and a blinking selection arrow.
- Added persistent selected-menu-item storage.
- Added a separate SRAM resume frame for every clip.

#### Web interface

- Added browser presets, per-scene palettes, uncompressed video, left/right audio selection, normalization, limiter, project defaults, per-clip overrides, clip reordering, and per-clip looping.
- Added desktop-equivalent output choices for playlist ROMs, menu ROMs, and separate ROMs in a ZIP.
- Added a browser ROM-core test that verifies the loop flag in clip metadata.

### Changed

- Advanced long-video controls are hidden until **Split the video** is checked.
- Normal **Single ROM** conversion remains compact and still falls back to automatic splitting when required.
- Manual size and duration rules do not affect conversion while **Split the video** is unchecked.
- Exported ROM and ZIP filenames no longer receive an application-version suffix.
- Long-video part-count estimates are refined from actual encoded part sizes during conversion.
- Each accepted automatic-split part stays below a conservative encoded-data budget and the next part begins at the same source timestamp.
- Fit with bars is now the default framing mode.
- Multi-clip conversion resolves trim, framing, audio, palette, dithering, and looping independently for each clip.
- Selected-video thumbnails were removed from the GBA menu, along with the obsolete desktop checkbox.
- The embedded player area was expanded from 20 KiB to 32 KiB for the menu background; media begins at `0x8000`.
- The browser interface now follows the desktop app's light and dark colour palettes and system theme.
- The desktop-download button now shares the website-title row.
- Browser media inspection now falls back to browser metadata when `ffprobe` fails and reports clearer FFmpeg diagnostics.
- The ambiguous web-only Automatic output mode was removed.

### Fixed

- Converted the blinking menu arrow from framebuffer drawing into a real 16×16 OBJ sprite.
- Arrow blinking now changes only OAM visibility instead of redrawing the background, titles, status text, and selected-title highlight.
- Full menu-framebuffer redraws now occur only when the selected clip changes.
- Aligned the independent OBJ menu arrow with the selected title instead of the gap between rows.
- Fixed hidden result and progress elements appearing before browser conversion by enforcing the HTML `hidden` attribute.

### Compatibility

- Kept the existing video decoder, audio DMA playback, timing, seeking, fixed 120×80 renderer, compression, and palette capacity unchanged.

## [0.8.0] - 2026-07-31

### Added

#### Converter application

- Added Best quality, Balanced, Long video, Smallest ROM, and Custom output presets.
- Added cartridge-size, frame-count, and approximate duration-limit estimates.
- Added start/end picture previews using crop, fit-with-bars, or stretch processing.
- Added audio-channel previews.
- Added optional loudness normalization and a limiter.
- Added multi-file drag and drop.
- Added batch output as separate ROMs in a ZIP.
- Added multiple clips in one ROM with a startup selection menu.

#### Player controls

- Added hold-to-seek with an immediate first jump and repeated jumps approximately every 0.4 seconds.
- Added configurable 3, 5, 10, and 15-second seek steps; the selected value appears in the seek popup.
- Added D-pad Up/Down volume control with 0%, 50%, and 100% Direct Sound levels.
- Added frame-by-frame D-pad Left/Right movement while paused.
- Added the current frame number to the full HUD.
- Added a `START + SELECT` controls-help screen.

#### Encoding

- Upgraded generated ROM metadata from GBV4 to GBV5.
- Expanded the embedded player template from 8 KiB to 16 KiB.
- Added keyframe/delta compression with automatic raw-frame fallback.
- Added frame indexes for compressed seeking and reconstruction.
- Added shared and scene-change palette modes.
- Added off, ordered, and error-diffusion dithering modes.
- Added per-frame palette-index synchronization.
- Added optional SRAM playback-position saving and resume prompts.
- Added per-clip assets, settings, titles, audio, palettes, and seek tables for collection ROMs.

### Changed

- Added the user-provided app icon as the browser app-window icon and repository ICO asset.
- Added automatic system light/dark mode palettes.
- Centered the drag-and-drop screen at every window size and removed its redundant header and film emoji.
- Restored named frame-rate choices: Smooth, Balanced, Classic, and Compact.
- ROM titles now start from the source filename and are cropped to the 12-character GBA limit.
- Simplified the trim label to “End (blank = full video)”.
- Kept `SELECT` for mute/unmute.
- Moved quick HUD show/hide to `L + R`.
- Changed `START` to cycle hidden, time-only, and full HUD modes.
- Kept `B` as restart.

### Fixed

- Tightened the volume popup so only one black pixel remains after the final digit.
- Fixed End-frame previews by seeking to the final decodable frame rather than exact EOF.

## [0.7.0] - 2026-07-31

### Added

- Added D-pad Left/Right seeking alongside L/R.
- Added mute feedback, playback HUD, progress line, seek feedback, and a loop indicator.
- Added per-frame audio seek offsets.
- Added strict loopback host checks, anti-CSRF token checks for local POST requests, and browser security headers.
- Added a red remove button for each loaded video.
- Added multi-video output choices for sequential playback, a clip menu, or separate ROMs in a ZIP.

### Changed

- Audited the repository and removed unused converter helpers and constants.
- Moved the session token from inline JavaScript into an escaped HTML meta element, allowing a strict script policy.
- Parallelized RGB555 palette lookup generation and reduced allocations in scene detection and error-diffusion dithering.
- Quantized independent video frames in parallel while preserving byte-for-byte ROM output.
- Optimized delta-frame encoding and binary index/palette writing without changing ROM output.
- Reduced the embedded browser icon from 69 KiB to 26 KiB without changing the Windows executable icon.
- Avoided duplicate GitHub Actions work by letting the Release workflow validate pushes to `main`, while CI handles pull requests and other branches.
- Removed the runtime FFmpeg downloader; portable releases now bundle a pinned, verified FFmpeg binary.
- Moved the browser UI from a large inline Go string into readable embedded files under `web/`.
- Stable GitHub Actions releases now use the exact `VERSION` tag without build-number suffixes or a prerelease badge.
- Replaced the clip-menu font chevron with a clear pixel-art right arrow.
- Menu-ROM clips now return to the clip menu automatically when playback finishes.
- In menu ROMs, pressing `B` during playback now returns to the clip menu instead of restarting the current clip.
- Embedded the custom icon directly into the Windows executable.
- Replaced the app icon with the latest transparent-background artwork.
- Refined seek-popup spacing and loop-icon placement.
- Upgraded generated ROM metadata to GBV4.

### Fixed

- Fixed audio/video drift and end-of-video silence by scheduling frames from a dedicated 16,384 Hz hardware playback clock.
- Normalized source audio timestamps with asynchronous resampling, improving files with timestamp gaps or overlaps.
- Paused frame stepping now repositions the audio stream to the selected frame before playback resumes.

## [0.6.0] - 2026-07-31

### Added

- Added L/R seeking with synchronized audio restart.
- Added reproducible LLVM player builds.

## [0.5.0] - 2026-07-30

### Added

- Added the portable local-web Windows GUI and the complete first conversion workflow.