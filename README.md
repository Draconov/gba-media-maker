<div align="center">
  <img src="assets/app_icon.png" width="96" alt="GBA Video Maker icon">

# GBA Video Maker

**Turn ordinary videos into playable Game Boy Advance ROMs — in the browser or with the portable Windows app.**

[![Version](https://img.shields.io/badge/version-0.12.1-ffd600?style=for-the-badge&labelColor=20252d)](CHANGELOG.md)
[![Web App](https://img.shields.io/badge/TRY-WEB_APP-ffd600?style=for-the-badge&labelColor=20252d)](https://draconov.github.io/gba-video-maker/)
[![Desktop](https://img.shields.io/badge/DOWNLOAD-DESKTOP_APP-ffffff?style=for-the-badge&labelColor=20252d)](../../releases/latest)
[![License](https://img.shields.io/badge/license-MIT-ffffff?style=for-the-badge&labelColor=20252d)](LICENSE)
</div>

## Choose your version

| | Web app | Portable desktop app |
|---|---|---|
| Installation | None | Extract the ZIP and run the EXE |
| Processing | Local, inside the browser | Local, through native FFmpeg |
| Best for | Quick conversions and short/medium videos | Long videos, repeated jobs, and maximum speed |
| Privacy | Files stay on your device | Files stay on your device |
| Platform | Modern desktop browsers | Windows x64 |
| Open | **[Launch the web converter](https://draconov.github.io/gba-video-maker/)** | **[Open the latest release](../../releases/latest)** |

The browser edition uses WebAssembly and may consume considerably more memory than the desktop app. For large conversions, the portable app is the safer choice.

## What it can create

- A normal single-video ROM
- One ROM that plays several clips in sequence
- One ROM with a startup clip-selection menu
- Separate `.gba` files packaged into a ZIP
- Single-video conversions automatically switch to numbered ROM parts inside one ZIP when one cartridge is not enough

The stable presets keep the proven playback path, while the new experimental preset opts into a versioned extension:

- Fixed **120×80** indexed video, expanded cleanly to the GBA's 240×160 display
- Standard signed 8-bit mono PCM audio at **16,384 Hz** for every legacy preset
- Optional block-based 4-bit IMA ADPCM only under **Extreme optimization (Experimental)**
- Hardware DMA audio playback and hardware playback clock
- Keyframe/delta compression with raw-frame fallback
- Optional scene-aware adaptive keyframes with explicit seek metadata under the experimental preset

## Highlights

### v0.12.1 — Independent title-card typography

- The split-video title-card editor now uses a compact two-row typography table.
- Title and subtitle each have independent **Large / Medium / Small** size, alignment, text colour, and outline colour controls.
- Title-card previews and generated native 240×160 screens use the exact per-row typography settings.
- Existing v0.12.0 project files migrate their shared typography automatically when opened.
- Desktop and web editions use the same responsive layout and the same custom GBA colour picker.

### v0.12 — Extreme optimization (Experimental)

Version 0.12 adds an isolated smart-encoding path without changing the output of Best quality, Balanced, Long video, Smallest ROM, or Custom projects unless the experimental preset is explicitly selected.

- **Representative analysis:** scans a bounded low-resolution set of frames and selects typical, fast-motion, high-detail, dark, colourful, transition, and low-motion samples
- **Candidate recommendations:** compares frame rate, palette mode, dithering, adaptive keyframes, and audio storage against a requested 8–32 MiB target
- **Quality and size trade-offs:** reports visual, motion, temporal-stability, and audio scores with estimated minimum/maximum ROM size and confidence
- **Recommendation application:** applies a chosen candidate while leaving the normal manual controls editable
- **Enhanced scene detection:** combines changed-pixel, motion, brightness, fade, flash-rejection, and minimum-scene-length signals
- **Adaptive keyframes:** forces keyframes at confirmed scene boundaries and uses content-aware maximum intervals while preserving arbitrary seeking
- **Experimental compact audio:** block-based 4-bit IMA ADPCM at 16,384 Hz uses roughly half the audio storage, with a real codec preview before conversion
- **Backward-safe preset isolation:** every non-Extreme preset forces standard PCM, fixed keyframes, and the established v0.11 conversion path
- **Desktop/web parity:** both editions expose the same preset, target, priority, audio choices, analyzer results, and ROM metadata

> [!WARNING]
> Compact ADPCM and adaptive encoding are experimental. They are covered by encoder/decoder and ROM-structure tests, but this development build has not been qualified on every flash cartridge or long-duration real-hardware playback setup. Use Standard PCM when reliability matters more than capacity.

### Video and ROM controls

Version 0.11 adds native per-part title cards for automatically split single-video conversions, plus a lighter preview pipeline and compact one-row part navigation.

- **Automatic long-video split:** no special mode is required. Create a normal **Single ROM**; when it cannot safely fit, the app selects the largest safe source-time segment for each part, verifies the encoded size, continues from the exact ending timestamp, and exports `NAME_PART_01.gba`, `NAME_PART_02.gba`, and `PARTS.txt` in one ZIP
- **Before starting:** the estimator shows `Estimated output: N ROM parts` using the selected ROM-size target and optional duration cap
- **During conversion:** progress shows `Part N of approximately M` and the current source position, for example `18:42 / 50:00`
- **Optional split panel:** check **Split the video** to reveal the 1–32 MiB target, 20 MiB / 30 MiB / Maximum shortcuts, chapter rules, title screens, and recovery settings; oversized Single ROM jobs still split automatically when the checkbox is off
- **Duration-based parts:** enter a maximum duration as `MM:SS` (for example `1:05`); `0` leaves the duration automatic
- **Chapter-aware splitting:** when chapter metadata is present, the splitter prefers a nearby earlier chapter boundary instead of cutting in the middle of a chapter
- **Long-job recovery:** accepted parts are kept in a persistent recovery folder and reused when the same conversion is started again after an interruption
- **Native per-part title cards:** split ROMs can open on a full 240×160 title card using the first frame of that part at 50% default darkening, the source filename as the title, and `Part {part}` as the automatic subtitle
- **Shared or individual design:** edit one style for every part or navigate with the compact `Part N` / `of M` controls and override title, subtitle, background frame, darkness, colours, alignment, Large/Medium/Small text size, timing, and fade behaviour
- **Compact options:** all title-card checkboxes stay together in one horizontal row
- **Exact preview:** both editions show the title card with the same RGB555 rendering and 3×5 font data embedded in the ROM

- Drag in one or more videos
- Trim start and end times
- Playback speed controls
- Fit with bars, crop, or stretch
- Four GBA-friendly frame-rate choices
- Configurable 3, 5, 10, or 15-second seek step
- Optional looping per clip
- Optional SRAM save/resume support
- Editable 12-character ROM title
- Clip reordering and per-clip menu titles

### Image quality

- Shared palette or per-scene palettes
- Dithering off, ordered, or error diffusion
- Optional keyframe/delta compression
- Uncompressed-video mode
- Automatic raw-frame fallback when compression would be larger

### Audio

- Mono mix, left channel, right channel, or disabled audio
- Per-clip volume
- Optional loudness normalization
- Optional limiter
- Selected-channel audio preview in both the desktop and web editors
- Audio-quality dropdown: Standard PCM, Compact ADPCM (Experimental), or Auto for ROM target under Extreme optimization
- Codec previews use the same 16,384 Hz PCM/ADPCM implementation written into the ROM

### Menu design and multi-clip menu

Version 0.10 added a dedicated **Menu design** panel whenever **One ROM — clip menu** is selected. The desktop and browser editions use the same theme format and embed the chosen design directly into the exported ROM.

- Live pixel preview built from the same 120×80 indexed background data used by the GBA player
- Integer-scaled preview with the player's exact 3×5 font, text coordinates, divider lines, and selector shape
- Built-in **Classic dark**, **Ocean Wave — static**, **Ocean Wave — animated**, and **Blue Wave — animated** backgrounds
- Ocean Wave animation uses a lightweight palette shimmer: the bright curl changes about twice per second and the lower water about five times per second
- Selectable UI-colour presets for normal and selected text
- Optional one-pixel UI outline with a selectable outline colour
- Custom PNG, JPEG, WebP, or GIF backgrounds, cropped to 120×80 and optimized to a GBA RGB555 indexed palette
- GIF backgrounds sampled to at most 16 looping frames; frame animations are drawn on the hidden Mode 4 page and switched during VBlank
- Theme data, animation settings, palette, UI colours, and outline settings are stored inside each menu ROM instead of requiring a separate player binary
- Menu-theme storage is included in the pre-conversion size estimate
- Menu-design choices are preserved in `.gbavideo` project files
- One, two, or three title columns with total duration and current-selection status
- Four-direction D-pad navigation and an independent blinking OBJ-sprite selector
- The selected menu item is remembered when returning from playback and across restarts when SRAM resume is enabled
- Each clip stores its own resume frame instead of sharing one global position
- No unstable video thumbnails in the GBA menu


### Automatic long-video handling

For one source video, choose the normal **Single ROM** output and convert as usual. The converter checks the minimum required data before encoding and also verifies the real encoded size afterward. When one cartridge would exceed the selected target, the result automatically changes from a `.gba` file to a `_PARTS.zip` package containing sequential ROMs plus `PARTS.txt`. No manual split mode is exposed.

The automatic split panel provides:

- A 1–32 MiB ROM-data target slider, plus 20 MiB, 30 MiB, and Maximum presets
- An optional fixed maximum source duration per part
- Chapter-aware split points when the source contains chapters
- Optional native 240×160 title cards with shared or per-part settings
- First-frame backgrounds darkened behind the source filename and automatic `Part {part}` subtitle
- Wait-for-A or timed start, optional skip, and fade into video
- Persistent recovery of completed parts after cancellation or failure (desktop cache on Windows; IndexedDB in the web app)

The pre-conversion estimate reports the approximate part count. While encoding, the status area shows both the current part and source timestamp. The estimate can change as real compressed sizes become available.

## Generated ROM controls

| Button | During playback |
|---|---|
| `A` | Pause or resume |
| `B` | Restart the clip, or return to the menu in menu ROMs |
| `L` / `R` | Seek backward or forward |
| Hold `L` / `R` | Repeat seeking approximately every 0.4 seconds |
| `D-pad Left` / `Right` | Seek while playing; step one frame while paused |
| `D-pad Up` / `Down` | Change volume: 0%, 50%, or 100% |
| `SELECT` | Mute or unmute |
| `START` | Cycle HUD: hidden, time only, or full |
| `L + R` | Quickly hide or restore the HUD |
| `START + SELECT` | Open the controls-help screen |
| `SELECT + Left/Right` | Previous or next clip in playlist ROMs |

### Menu controls

| Button | In the clip menu |
|---|---|
| `D-pad Up` / `Down` | Move within a column |
| `D-pad Left` / `Right` | Move between columns |
| `A` | Play the selected clip |

The full HUD can show elapsed and total time, current frame number, progress, and the loop icon.

## Desktop app

Download the portable ZIP from the repository's **Releases** page, extract it, and run:

```text
GBA Video Maker.exe
```

No installer, Python runtime, administrator access, or devkitARM installation is needed. Official portable packages place a pinned Windows x64 `ffmpeg.exe` beside the application. The application does not download or update executables at runtime.

> [!NOTE]
> Windows may warn about an unsigned executable. Building from source or signing release binaries is the reliable way to establish publisher trust.

## Web app

The standalone browser edition lives entirely under [`website/`](website/). It uses the same embedded `assets/player_stub.bin` as the desktop converter, so generated ROMs share the same playback engine and menu.

The web edition now mirrors the Windows application's conversion workflow:

- Quality presets, custom mode, size estimates, and the 32 MiB optimizer
- Single ROM, menu, playlist, and separate-ROM ZIP outputs
- Automatic and manually configured long-video splitting
- Adjustable 1–32 MiB split target, `MM:SS` duration caps, chapter-aware cuts, part title screens, and numbered-ROM ZIP manifests
- Interrupted split recovery in IndexedDB; reselecting the same source video continues after completed parts
- `Part N of approximately M` and source-position progress
- Frame rate, framing, palette, dithering, compression, audio channel, volume, normalization, limiter, trimming, speed, looping, seek step, and SRAM resume
- Project save/open with source-file relinking, project defaults, per-clip overrides, clip reordering, a selected-clip timeline, GBA-font title preview, and audio preview
- Menu-design parity with the desktop app, including live preview, UI and outline colours, built-in static/animated backgrounds, and custom image/GIF backgrounds

Browser processing is local: selected videos are copied into FFmpeg's in-browser filesystem and are not uploaded to a project server. Browser security does not allow a saved project to silently reopen local files, so the web app relinks them by filename after the user selects them again.

### Run the website locally

Install Node.js, then:

```bash
cd website
npm install
npm test
npm run dev
```

Build the exact static site used by GitHub Pages:

```bash
cd website
npm run build
npm run preview
```

GitHub Pages deployment is configured in [`.github/workflows/pages.yml`](.github/workflows/pages.yml).

## Resume support

When **Save/resume position** is enabled, the ROM declares `SRAM_V113` save memory.

For a single-video ROM it stores the playback frame. For menu and playlist ROMs it stores:

- The most recently selected clip
- A separate playback frame for every clip

Returning from a menu clip preserves that clip's position. Selecting it again presents the normal resume prompt:

- `A` continues from the saved time
- `B` restarts only that clip from the beginning

Finishing a clip normally clears only that clip's saved frame. SRAM behavior should be tested on the intended flash cartridge because save handling differs among cartridges and emulator configurations.

## ROM format

| Property | Value |
|---|---|
| Display | 240×160 Mode 4 |
| Encoded video | 120×80 indexed frames |
| Video colours | RGB555 palette entries 0–249 |
| UI colours | Palette entries 250–255 |
| Audio | Signed 8-bit mono PCM, 16,384 Hz |
| Maximum ROM size | 32 MiB |
| Padding | Next power-of-two cartridge size |

Delta compression stores periodic full keyframes and changed-byte runs between them. Each clip includes frame and seek metadata so the player can reconstruct frames after seeking.

## Privacy and security

### Desktop

The desktop GUI is served locally on a random `127.0.0.1` address with a random session token. FFmpeg processes selected videos locally. Temporary files are created in the operating system's temporary directory and removed after normal shutdown.

### Browser

The GitHub Pages site is static. Conversion runs in the browser with WebAssembly. The project does not require a video-upload server.

See [`SECURITY.md`](SECURITY.md) for vulnerability-reporting guidance.

## Build from source

Requirements:

- Go 1.23 or newer
- FFmpeg for conversions and integration tests
- LLVM tools (`clang`, `ld.lld`, and `llvm-objcopy`) to rebuild the GBA player
- Node.js and npm for the standalone website

Run the Go checks:

```bash
go fmt ./...
go vet ./...
go test ./...
```

Rebuild the embedded GBA player after changing files under `player/`:

```bash
./player/build.sh
```

Build the Windows executable:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-H windowsgui -s -w" \
  -o "GBA Video Maker.exe" .
```

PowerShell helpers:

```powershell
./scripts/build-windows.ps1
./scripts/package-release.ps1 -Version 0.12.1
```

## Project layout

```text
player/                  ARM7TDMI player source and build scripts
assets/player_stub.bin   Prebuilt 32 KiB GBA player template
converter.go             FFmpeg pipeline, palettes, compression, and ROM assembly
menu_theme.go            MTH1 menu-theme validation and ROM embedding
title_card.go            Native 240×160 title-card rendering and TCD1 assets
web/                     Desktop interface, menu themes, and title-card editor
website/                 Standalone GitHub Pages converter with matching tools
webapp.go                Local desktop HTTP API and preview-job coordination
scripts/                 Desktop build and packaging helpers
docs/                    Architecture documentation
.github/workflows/       CI, release, and Pages deployment
```

More implementation details are available in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) and [`website/README.md`](website/README.md).

## FFmpeg and legal notes

FFmpeg is a separate open-source project distributed under its own licences and is not committed to the source repository. Official portable packages may include a verified FFmpeg binary. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

Convert only media you have permission to use. Game Boy Advance, Nintendo, and related marks belong to their respective owners. This independent project is not affiliated with or endorsed by Nintendo.

## License

GBA Video Maker is available under the [MIT License](LICENSE).
