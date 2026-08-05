# Changelog

## [0.9.0] - 2026-08-05

### Added

#### Long-video conversion

- Automatic fallback from a single ROM to sequential numbered ROMs in a ZIP when the source cannot safely fit on one cartridge.
- Pre-conversion estimates such as `Estimated output: N ROM parts`.
- Split progress showing `Part N of approximately M` and the current source position.
- Optional manual splitting with:
  - a 1–32 MiB target-size slider;
  - 20 MiB, 30 MiB, and Maximum shortcuts;
  - maximum duration per part using seconds, `MM:SS`, or `H:MM:SS`;
  - chapter-aware cut points;
  - optional filename and `PART N` title screens.
- Adaptive part sizing that shortens oversized candidates and extends underfilled candidates.
- `PARTS.txt` manifests containing filenames, source ranges, encoded-data sizes, and cartridge sizes.
- Interrupted-conversion recovery that keeps and reuses completed parts.
- Early detection when PCM audio alone would exceed the selected cartridge budget.
- Disk-backed desktop splitting for long sources.

#### Project workflow

- Per-video settings with project-default inheritance for title, trim, speed, framing, audio, volume, looping, palette, and dithering.
- Drag-and-drop clip ordering plus accessible Move up and Move down controls.
- `.gbavideo` project save/open support with source-file relinking.
- Timeline scrubbing with thumbnails, start/end handles, a current-position handle, fine adjustment, and quick jumps.
- A reviewable 32 MiB optimization proposal.
- Live GBA-font title editing with character validation, duplicate-title warnings, filename reset, and automatic 12-character truncation.

#### Collection ROMs and playback

- Playlist previous/next controls using `SELECT + Left/Right`.
- A multi-column collection menu with total duration, current-selection status, D-pad navigation, and an OBJ-based selection arrow.
- Persistent selected-menu-item storage.
- Separate SRAM resume positions for every clip.

#### Web application

- Feature parity with the Windows conversion workflow, including Single ROM, playlist ROM, menu ROM, and separate-ROM ZIP outputs.
- Automatic and manually configured long-video splitting in the browser.
- IndexedDB recovery for completed split parts.
- Browser-safe `.gbavideo` project reopening and source relinking.
- Presets, per-scene palettes, uncompressed video, channel selection, normalization, limiter, per-clip overrides, and per-clip looping.
- Browser ROM-core tests for metadata, splitting, and collection behavior.

### Changed

- Advanced split controls are hidden until **Split the video** is enabled.
- **Single ROM** remains the compact default and still falls back to automatic splitting when required.
- Manual split limits are ignored while **Split the video** is disabled.
- Exported ROM and ZIP filenames no longer include an application-version suffix.
- Part-count estimates are refined from actual encoded sizes during conversion.
- Every accepted split part remains below the selected data budget, and the next part begins at the previous part's exact end timestamp.
- Fit with bars is now the default framing mode.
- Multi-clip conversion resolves trim, framing, audio, palette, dithering, and looping independently for each clip.
- Selected-video thumbnails were removed from the GBA menu.
- The embedded player area was expanded to 32 KiB; media begins at `0x8000`.
- The browser interface now follows the desktop light/dark palette and system theme.
- The desktop-download button now shares the website-title row.
- Browser media inspection falls back to browser metadata when `ffprobe` fails and reports clearer FFmpeg diagnostics.
- The ambiguous web-only Automatic output mode was removed.
- The web clip editor now uses a full-width video preview with a separate Windows-style timeline below it.
- Browser-native video controls are hidden; clicking the video or pressing Space/Enter toggles playback.
- Start and End timestamps are directly editable using seconds, `MM:SS`, or `H:MM:SS`.
- Start, Current, and End use matching rectangular handles that no longer shift or resize on hover.

### Fixed

- Converted the blinking menu arrow from framebuffer drawing to a 16×16 OBJ sprite.
- Arrow blinking now changes only OAM visibility instead of redrawing the full menu.
- Full menu redraws now occur only when the selected clip changes.
- Aligned the menu arrow with the selected title.
- Fixed hidden result and progress elements appearing before browser conversion.

### Compatibility

- Kept the existing video decoder, audio DMA playback, timing, seeking, fixed 120×80 renderer, compression format, and palette capacity unchanged.

## [0.8.0] - 2026-07-31

### Added

#### Converter

- Best quality, Balanced, Long video, Smallest ROM, and Custom presets.
- Cartridge-size, frame-count, and approximate duration-limit estimates.
- Start/end picture previews using crop, fit-with-bars, or stretch processing.
- Audio-channel previews, optional loudness normalization, and a limiter.
- Multi-file drag and drop.
- Batch output as separate ROMs in a ZIP.
- Multiple clips in one ROM with a startup selection menu.

#### Player controls

- Hold-to-seek with an immediate jump and repeated jumps approximately every 0.4 seconds.
- Configurable 3, 5, 10, and 15-second seek steps with on-screen feedback.
- D-pad Up/Down volume control with 0%, 50%, and 100% Direct Sound levels.
- Frame-by-frame D-pad Left/Right movement while paused.
- Current frame number in the full HUD.
- A `START + SELECT` controls-help screen.

#### Encoding

- GBV5 ROM metadata.
- A 16 KiB embedded player template.
- Keyframe/delta compression with automatic raw-frame fallback.
- Frame indexes for compressed seeking and reconstruction.
- Shared and scene-change palette modes.
- Off, ordered, and error-diffusion dithering modes.
- Per-frame palette-index synchronization.
- Optional SRAM playback-position saving and resume prompts.
- Per-clip assets, settings, titles, audio, palettes, and seek tables for collection ROMs.

### Changed

- Added the custom app icon to the browser window and repository ICO asset.
- Added automatic system light/dark palettes.
- Centered the drag-and-drop screen and removed its redundant header and emoji.
- Restored named frame-rate choices: Smooth, Balanced, Classic, and Compact.
- ROM titles now begin with the source filename and are truncated to the 12-character GBA limit.
- Simplified the trim label to **End (blank = full video)**.
- Kept `SELECT` for mute/unmute, moved quick HUD visibility to `L + R`, and changed `START` to cycle HUD modes.

### Fixed

- Tightened volume-popup spacing.
- Fixed end-frame previews by seeking to the final decodable frame instead of exact EOF.

## [0.7.0] - 2026-07-31

### Added

- D-pad Left/Right seeking alongside L/R.
- Mute feedback, playback HUD, progress line, seek feedback, and a loop indicator.
- Per-frame audio seek offsets.
- Strict loopback host checks, anti-CSRF checks, and browser security headers.
- A remove button for each loaded video.
- Multi-video output choices for sequential playback, a clip menu, or separate ROMs in a ZIP.

### Changed

- Removed unused converter helpers and constants.
- Moved the session token from inline JavaScript to an escaped HTML meta element, enabling a strict script policy.
- Parallelized palette lookup generation and independent-frame quantization while preserving byte-for-byte ROM output.
- Reduced allocations in scene detection and error-diffusion dithering.
- Optimized delta-frame encoding and binary metadata writing.
- Reduced the embedded browser icon size without changing the Windows executable icon.
- Split GitHub Actions responsibilities between stable releases and pull-request CI.
- Removed the runtime FFmpeg downloader; portable releases now bundle a pinned, verified FFmpeg build.
- Moved the browser UI from an inline Go string into readable files under `web/`.
- Stable releases now use the exact `VERSION` tag without build-number suffixes or a prerelease badge.
- Replaced the menu chevron with a pixel-art arrow.
- Menu-ROM clips now return to the menu when playback finishes or when `B` is pressed.
- Embedded the latest transparent app icon directly into the Windows executable.
- Refined seek-popup spacing and loop-icon placement.
- Upgraded generated ROM metadata to GBV4.

### Fixed

- Fixed audio/video drift and end-of-video silence using a dedicated 16,384 Hz playback clock.
- Normalized source audio timestamps with asynchronous resampling.
- Paused frame stepping now repositions audio before playback resumes.

## [0.6.0] - 2026-07-31

### Added

- L/R seeking with synchronized audio restart.
- Reproducible LLVM player builds.

## [0.5.0] - 2026-07-30

### Added

- The portable local-web Windows interface and first complete conversion workflow.