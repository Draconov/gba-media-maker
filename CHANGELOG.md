# Changelog

All notable user-visible changes are recorded here. Dates use `YYYY-MM-DD`.

> [!NOTE]
> Historical sections describe the controls and behavior that existed in that release. For the current v0.13.1 controls and media model, use the main [`README.md`](README.md).

## Release index

| Version | Date | Main theme |
|---|---|---|
| **0.13.1** | **2026-08-16** | Audio/image player polish: two-state image HUD, automatic built-in audio covers, cover-visible audio HUD |
| **0.13.0** | **2026-08-11** | GBA Media Maker: video + GIF + music + images, mixed-media menus, desktop/web parity |
| 0.12.2 | 2026-08-07 | Stable video player/menu refinements, custom menu video backgrounds, Cyrillic text |
| 0.12.1 | 2026-08-07 | Independent split-title typography and responsive editor fixes |
| 0.12.0 | 2026-08-06 | Extreme optimization, adaptive keyframes, experimental IMA ADPCM |
| 0.11.0 | 2026-08-06 | Native split-video title cards |
| 0.10.0 | 2026-08-05 | Custom MTH1 menu themes and stable menu preview parity |
| 0.9.0 | 2026-08-05 | Long-video splitting, projects, collection menus, browser parity |
| 0.8.0 | 2026-07-31 | Presets, compression, menu ROMs, advanced controls |
| 0.7.0 | 2026-07-31 | Playback clock/HUD, local-app hardening, stable release workflow |
| 0.6.0 | 2026-07-31 | Synchronized seeking and reproducible player builds |
| 0.5.0 | 2026-07-30 | First complete portable local-web conversion workflow |


## [0.13.1] - 2026-08-16

### Changed
- Image playback now has only two HUD states: hidden and shown. Images start with the HUD hidden.
- New audio tracks are assigned one of the 20 built-in artwork presets automatically; the selected preset remains editable and saved with the project.
- Audio HUD panels now dim the cover beneath them instead of replacing the lower part of the cover with solid black, keeping the full cover visible while playback information is shown.
- Video playback, video timing, seeking, PCM end protection, and the restored Game Pak wait-state configuration are unchanged from the validated v0.13.0 runtime.

## [0.13.0] - 2026-08-11

### Highlights

- Renamed the product experience to **GBA Media Maker** and expanded the converter from video-only projects to first-class **video, animated GIF, music/audio, and static-image** media.
- Preserved the established GBV5 version 5 container and the stable v0.12.2 video/menu presentation while adding media-specific descriptor flags and runtime dispatch.
- Brought the standalone website to the current desktop media model, ROM format, player runtime, output naming, menu/theme tools, title cards, and project schema.

### Added — media types

- Added audio-only conversion for FFmpeg-supported music/audio sources such as MP3, WAV, FLAC, OGG/Opus, M4A, and AAC.
- Added native **240×160 RGB555** audio artwork and a GBA Now Playing screen.
- Added editable **28-character song title** and **28-character artist** fields plus a compact `MMD2` record that also retains a 20-character album field.
- Added three per-track audio artwork modes:
  - **Embedded artwork** with fallback to the selected built-in preset;
  - **Default artwork** using one of 20 bundled 240×160 presets;
  - **Custom image** using PNG/JPEG/WebP artwork prepared for the GBA screen.
- Added native **240×160 RGB555** static-image media entries.
- Added explicit image **Enable slideshow** behavior; a zero duration is a manual viewer rather than an implicit five-second slideshow.
- Added animated GIF import through the video path. GIFs are always treated as animated video media and force native looping.

### Added — collections and projects

- Added mixed-media menu ROMs containing any combination of video, GIF, audio, and image entries.
- Changed the new collection model so **every project with two or more items uses the media menu**, even when every item has the same media type. Separate ROMs remain available as ZIP output.
- Added `[V]`, `[A]`, and `[I]` media labels to collection-menu entries.
- Added `.gbamedia` **project format v2** (`gba-media-maker-project`) for the mixed-media model.
- Kept legacy `.gbavideo` / GBA Video Maker project loading and migrated old playlist output values to the current media-menu mode.
- Added desktop/browser output naming parity for single ROMs, menu collections, separate-ROM ZIPs, split-video archives/parts, and project files.

### Added — audio player and HUD

- Added audio seeking, pause/resume, loop handling, 0/50/100 volume, mute/unmute, HUD modes, media switching, and save/resume integration.
- Audio starts with the full Now Playing HUD visible.
- Added video-style audio seek/mute/volume feedback without force-blanking or redrawing the full artwork screen for ordinary UI updates.
- Refined the audio Now Playing layout with title, artist, elapsed/total time, PLAY/PAUSE status, and a four-pixel progress line.

### Changed — ROM controls

- `L` / `R` now change to previous/next media directly.
- D-pad Left/Right owns seek while playing and frame/timeline stepping while paused.
- Held D-pad Left/Right repeats every **18 VBlanks (~0.30 s)**.
- D-pad Up/Down and `SELECT` are available only when the current media actually contains audio.
- `START` cycles hidden → time-only → full HUD.
- `START + SELECT` opens the controls-help screen.
- Removed the redundant `SELECT + L/R` media shortcut everywhere.
- Removed the redundant `L + R` quick-HUD shortcut everywhere.
- Manual images no longer react to `A`; slideshow images still use `A` to pause/resume slideshow timing.
- GIFs, silent videos, videos converted with **No audio**, and images no longer expose mute/volume controls or badges.

### Changed — video GUI/runtime

- Restored the stable **v0.12.2 video HUD presentation**, including the frame counter, progress layout, seek indicator, full-HUD behavior while paused, and yellow loop icon.
- Temporary seek-arrow feedback, temporary full HUD after seek, mute/unmute feedback, and volume feedback now each last **6 VBlanks (~0.10 s)**.
- Preserved the newer input-polling order so late video frames cannot starve keypad input.
- Fixed temporary HUD timer advancement so seek/mute/volume indicators cannot remain stuck when video decoding is behind schedule.
- Removed obsolete shoulder-combo grace timing after `L + R` stopped being a command, making ordinary L/R media switching immediate.

### Changed — save/resume

- Restored the v0.12.2-style resume confirmation instead of silently auto-resuming a saved position:

  ```text
  CONTINUE FROM
      MM:SS

   A CONTINUE
   B RESTART
  ```

- `A` resumes the saved video/audio position; `B` clears that position and restarts the current media.
- Static images do not show a playback-position prompt.
- Menu selection and per-media resume positions remain independent.

### Changed — menu/theme system

- Restored the complete stable v0.12.2 menu-theme/rendering stack instead of the simplified interim v0.13 implementation.
- Restored Ocean Wave and Blue Wave background behavior, animated MTH1 timing, outline rendering, selection-arrow behavior, and exact logical 120×80 preview geometry.
- Restored the full custom v0.12.2 GBA colour picker with saturation/value area, hue strip, eyedropper, RGB/HEX input, and preset swatches.
- Retained v0.13 media-aware `SELECT MEDIA` wording and media tags on top of the stable renderer.

### Added — website parity

- Added first-class browser video/GIF/audio/image handling with the same v0.13 project model as the EXE.
- Added the same 20 audio-artwork presets, embedded/default/custom artwork modes, fallback behavior, and project persistence.
- Added native browser audio/image GBV5 descriptors and `MMD2` metadata generation.
- Added mixed-media/same-media menu collection output and Separate ROMs ZIP output.
- Added current `.gbamedia` v2 save/open and legacy-project migration.
- Added desktop-compatible output naming helpers.
- Synchronized the website player from the authoritative `assets/player_stub.bin` rather than maintaining a separate browser runtime.
- Expanded browser tests for ROM structure, project format/migration, naming parity, media assets, and current UI/ROM behavior.

### Fixed

- Fixed a regression that made the custom menu colour inputs appear as thin browser-native bars instead of the v0.12.2 custom picker.
- Fixed the interim menu restore that had overwritten newer v0.13 playback-control handling.
- Fixed keypad input starvation when the video decoder/render loop was already late.
- Fixed seek feedback that could remain visible indefinitely on late/heavy video playback.
- Fixed audio seek/mute/volume feedback causing a white full-screen flash by updating only affected native UI regions.
- Fixed silent video/GIF/image entries accepting meaningless audio controls.
- Fixed save/resume-enabled ROMs silently jumping to saved positions without the continue/restart screen.
- Fixed Windows icon assets whose rounded corners were stored as opaque black pixels; current PNG/ICO/web icon assets preserve transparency.
- Fixed image project loading so an explicit zero-second/manual-viewer slideshow value is not changed back to five seconds.
- Fixed legacy browser playlist projects so they upgrade to the current menu collection mode.

### Packaging and build

- Kept the GBA player stub fixed at **32 KiB** with media beginning at `0x8000` and global metadata at `0x7FC0`.
- Current player builds use size-oriented `-Oz` so the media runtime, stable menu system, HUD, and restored resume prompt fit the fixed pre-asset region.
- CI verifies the committed stub size, smoke-builds the player to a temporary output, synchronizes the website stub, runs Go tests/vet, and cross-builds the Windows executable.
- GitHub Pages rebuilds/synchronizes the player before website tests/build.
- Release packaging continues to pin the BtbN FFmpeg release, discover the correct non-shared win64 LGPL archive from `checksums.sha256`, verify SHA-256, and verify software AV1 decoding support.

### Compatibility

- The global ROM format remains **GBV5 version 5**.
- Existing video entries continue to use the established 120×80 indexed playback path.
- v0.13 consumes previously unused clip flag bits for audio-only (`0x0040`), image (`0x0080`), and media metadata (`0x0100`) instead of creating a new container version.
- The current runtime reads both `MMD2` and legacy `MMD1` audio metadata.
- Legacy playlist ROM flags remain understood even though new multi-item projects use the media menu.
- Existing v0.12.x video conversion, splitting, title-card, menu-theme, text, PCM, and experimental ADPCM features are retained.

## [0.12.2] - 2026-08-07

- Added custom video menu backgrounds for menu ROMs in both desktop and web editions. Videos are center-cropped to 120×80, sampled to at most 16 looping MTH1 frames, support configurable start time and 1–32 second sample duration, and ignore source audio.
- Fixed pinned BtbN FFmpeg release discovery for dated autobuilds whose master snapshot is named `ffmpeg-N-...-win64-lgpl.zip` rather than `ffmpeg-master-...`.
- Fixed the pinned BtbN FFmpeg release download: dated releases now discover their build-specific Windows x64 LGPL archive name from `checksums.sha256` instead of incorrectly requesting the floating `*-latest-*` filename.
- Fixed AV1 source conversion on PCs without hardware AV1 decoding by switching portable releases to a pinned FFmpeg build with a verified software AV1 decoder; incompatible manual FFmpeg builds now produce a clear AV1-specific error.
- Added per-source audio-track selection for video containers with multiple audio streams; track choice is preserved in project files and used by conversion and audio preview in both desktop and web editions.
- Replaced the loose pause boolean with an explicit `RUNNING -> PAUSED -> RESUME_ARMED -> RUNNING` playback state machine so pause/resume work cannot leak into unrelated UI/decode paths.
- Added explicit prepared-next-frame validity tracking. Ordinary pause keeps both the visible front page and rendered back page intact; help, HUD redraw, frame-step, and seek invalidate the prepared page deliberately.
- Normal resume performs no video decode or current-frame redraw: it uses only the validated prepared back frame, positions audio while stopped, flips on VBlank, then starts the playback/audio clocks together.
- Hardened rapid pause/unpause input with a dedicated A-button release latch: pause toggles only on a real up-to-down edge and cannot repeat while A remains held, even across resume/UI state transitions.
- Preserved the already-rendered next video frame across ordinary pause/unpause cycles instead of leaving the frame wait loop and rebuilding it.
- Added a dedicated `paused -> resume pending -> running` player path, bypassing the generic UI-refresh/redraw route during unpause.
- Resume still prepares frame-aligned audio while stopped, flips the prepared frame on VBlank, then starts the playback/audio clocks together.
- Fixed the bundled GBA player asset so generated ROMs use the compact five-step `/` glyph again, restoring the nicer slash in HUD time displays and menu clip counters.
- Fixed rapid pause/unpause responsiveness: the player now prepares the resume audio stream while it is still paused, waits for the next VBlank, presents the already-prepared video frame, then starts the playback clock and audio on that same display boundary. This prevents audio from starting before the visual resume point is ready on heavy menu ROMs.
- Moved the bundled GBV5 metadata block from `0x7F00` to `0x7FC0`, giving the player stub more headroom for runtime fixes while keeping the ROM payload start at `0x8000`.
- Player changes are now rebuilt and checked by local build scripts and GitHub workflows so `player/main.c` cannot silently drift away from `assets/player_stub.bin`.
- Fixed the desktop/local web title-card runtime so `TitleCardTools` is created correctly after the shared GBA text module loads.
- Menu and other output modes no longer fail serialization if the optional title-card runtime is unavailable.

### Added
- Added one shared Latin + Ukrainian/Russian Cyrillic 3×5 GBA font instead of separate language fonts.
- Added Ukrainian-specific `Ґ Є І Ї`, Russian-specific `Ё Ъ Ы Э`, and the shared Cyrillic alphabet to menu titles, title-card titles/subtitles, previews, and runtime menu rendering.
- Added UTF-8-safe glyph counting, compact one-byte Cyrillic ROM codes, unsupported-character reporting, typographic punctuation normalization, and ASCII cartridge-header transliteration.
- Added desktop/web/player parity tests for Ukrainian, Russian, mixed Cyrillic, title-card rendering, and browser ROM descriptor encoding.

### Changed
- Menu and title-card text limits now operate on visible characters rather than UTF-8 byte length.
- Lowercase Cyrillic input is accepted and rendered in the existing uppercase-style pixel font.
- The desktop and website editors now warn when text contains characters the GBA font cannot display.

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