# Changelog

## [0.12.1] - 2026-08-07

### Fixed
- Improved the split-video title/subtitle editor so the text fields receive more horizontal space in wide layouts, making longer titles visible without changing the project version.
- Title-card editor layout now sizes against the actual title-card panel instead of the whole browser/window, preventing controls from escaping the card in the desktop two-pane layout.
- The preview column shrinks responsively before the layout stacks, preserving the side-by-side editor whenever there is enough room.
- Container-query fallbacks stack the preview/settings and typography controls only when the title-card panel itself is genuinely narrow.

### Added
- Added a compact two-row typography table to split-video title cards.
- Added independent title and subtitle size, alignment, text colour, and outline colour settings.
- Added matching desktop/web custom colour pickers for both title-card text rows.
- Added migration tests so v0.12.0 projects with shared typography continue to render with the previous title/subtitle hierarchy.

### Changed
- Title cards now render title and subtitle typography independently in the native 240×160 preview and generated ROM asset.
- Default new-project typography is **Large** title and **Small** subtitle, both centred with white text and black outlines.
- Updated desktop/web responsive styling so the two typography rows remain compact at normal widths and reflow cleanly on narrow screens.

## [0.12.0] - 2026-08-06

### Added
- Added **Extreme optimization (Experimental)** as an isolated quality preset; all existing presets keep the v0.11 encoding path and standard PCM audio.
- Added bounded scene/motion/detail analysis, representative sample selection, candidate generation, quality metrics, sample-informed size ranges, confidence reporting, and recommendation application.
- Added enhanced scene-boundary detection with fade handling and flash rejection.
- Added adaptive keyframe placement with explicit keyframe records and frame-indexed seeking.
- Added block-based 4-bit IMA ADPCM at 16,384 Hz, independent block state, frame-to-sample seek tables, GBA-side decoding, double-buffered DMA feeding, and pause/resume/seek support.
- Added Standard PCM, Compact ADPCM (Experimental), and Auto for ROM target audio choices plus exact codec audio previews.
- Added equivalent Extreme controls and conversion metadata to the browser edition.
- Added Go and JavaScript ADPCM golden/round-trip tests, smart-analysis tests, adaptive/ADPCM ROM descriptor tests, and website parity assertions.

### Changed
- Smart-analysis FFmpeg work is bounded, cancellable, and uses one scan stream instead of starting concurrent preview jobs.
- The browser title-card editor remains available when manual splitting is enabled even while browser metadata is still unknown.
- Project files now preserve the selected experimental preset, audio mode, target, priority, and optional analysis result.
- README, website documentation, player documentation, architecture notes, release scripts, and visible version labels now identify v0.12.0.

### Compatibility
- Legacy presets write PCM and fixed-keyframe GBV5 clips exactly through the established path.
- ADPCM and adaptive flags use previously reserved descriptor fields; the bundled v0.12 player understands both legacy and experimental clips.

## [0.11.0] - 2026-08-06

### Added

#### Native title cards for split videos

- A **Title cards for split video** editor that appears only when one source video is expected to produce more than one ROM part.
- Previous/next buttons and a direct `Part N` selector followed by plain `of M` text for editing every generated part from one compact navigation row.
- Shared settings by default, with optional per-part overrides and **Copy this part to all**.
- Default title from the source filename and default subtitle `Part {part}`, resolved automatically for each ROM.
- First-frame-of-part and timestamp-in-part backgrounds, plus a solid-colour background option.
- Adjustable background darkening, title/subtitle text, Large/Medium/Small text sizing, alignment, start behaviour, timer, skip, and fade controls.
- Reuse of the existing RGB555 custom colour picker for title text, outline, and solid backgrounds.
- A pixel-accurate native **240×160** title-card preview in both desktop and browser editions.
- `TCD1` title-card assets containing a full 240×160 RGB555 screen, timing, input, and fade flags.
- Per-part title-card settings in `.gbavideo` project files.

### Changed

- Split ROMs can display the native title card before the 120×80 video player is initialized.
- Title-card storage is included in split estimates and exact ROM-size checks.
- The GBA player can fade the native title card to black and fade the first video frame in.
- Legacy text-only split title screens remain supported for older project data.
- Title-card previews use fast input seeking and a single FFmpeg decoder thread, keeping extraction lightweight even for later parts of long videos.
- Desktop preview requests are debounced, cancelled when obsolete, cached by source frame, and coalesced so duplicate UI refreshes cannot launch multiple FFmpeg processes.
- The title-card editor no longer reloads the selected part during every 500 ms state poll or unchanged size estimate.
- Part navigation stays in one compact row in both the Windows app and browser edition.
- All title-card checkboxes are grouped into one compact options row.
- The default title-card background darkening is now 50%.
- Labels were shortened to **Show title card at start** and **Use same settings for each part**.

### Fixed

- Fixed several simultaneous `ffmpeg.exe` preview processes driving CPU usage close to 100% while the split-video title-card editor was open.
- Fixed the part selector expanding to the full panel width and pushing the previous/next buttons onto separate rows.
- Fixed rapid part switching leaving obsolete preview extraction jobs running after a newer part was selected.
- Fixed automatic split mode failing to preserve an explicitly disabled title-card preference.
- Kept desktop and browser title-card defaults, rendering rules, RGB555 quantization, and ROM metadata layout synchronized.

## [0.10.0] - 2026-08-05

### Added

#### Menu design system

- A dedicated **Menu design** settings section for **One ROM — clip menu** output in both the Windows app and browser edition.
- A live 120×80 indexed menu preview showing the actual GBA font, divider lines, clip titles, total duration, selected item, and selector.
- Built-in menu backgrounds:
  - **Classic dark**;
  - **Ocean Wave — static**;
  - **Ocean Wave — animated**;
  - **Blue Wave — animated**.
- Seven UI-colour presets controlling normal and selected menu text.
- An optional one-pixel UI outline with selectable black, dark navy, white, blue, or yellow colour.
- Custom PNG, JPEG, WebP, and GIF menu backgrounds.
- Automatic center-crop and resize to 120×80, RGB555 conversion, indexed-palette optimization, and reservation of player UI palette entries.
- GIF sampling to at most 16 looping optimized frames.
- `MTH1` menu-theme records embedded directly into generated menu ROMs, including the palette, frame data, animation type, timing, UI colours, outline settings, and palette-shimmer configuration.
- Menu-theme validation and ROM-assembly tests in both the Go and browser ROM builders.
- Menu-design settings in `.gbavideo` project save/open data.

#### Animated menu backgrounds

- A dual-rate palette shimmer for **Ocean Wave — animated**: the bright wave curl changes approximately twice per second while the lower water changes approximately five times per second.
- A generated multi-frame **Blue Wave — animated** design.
- Mode 4 hidden-page drawing and VBlank page switching for tear-resistant image/GIF animation.
- ROM-size estimation that includes embedded menu-theme data.

### Changed

- Menu backgrounds are selected during ROM assembly and stored in the exported ROM instead of requiring a separately compiled player for each design.
- The generic 32 KiB player now reads optional menu-theme metadata and falls back to its built-in menu background when no valid theme is present.
- Desktop and website menu-theme conversion and preview behavior now share matching logic and built-in assets.
- The menu preview is rendered only at exact integer multiples of the logical 120×80 resolution.
- Preview text uses the same 3×5 glyph data and coordinates as the GBA player.
- The preview selector now uses the same `2, 3, 4, 3, 2` pixel shape and position as the OBJ sprite in the ROM.

### Fixed

- Fixed uneven, distorted preview lettering caused by non-integer browser scaling.
- Fixed the preview selector pointing in a different shape and position from the real ROM selector.
- Fixed custom menu-theme bytes being omitted from output-size estimates.

### Compatibility

- Kept the v0.9 video decoder, audio DMA path, playback clock, seeking, compression, clip metadata, and fixed 120×80 video renderer unchanged.
- Existing non-menu, playlist, batch, and long-video splitting workflows continue to use the same media format and controls.

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