<div align="center">
  <img src="assets/app_icon.png" width="96" alt="GBA Video Maker icon">

# GBA Video Maker

**Turn ordinary videos into playable Game Boy Advance ROMs — in the browser or with the portable Windows app.**

[![Version](https://img.shields.io/badge/version-0.12.2-ffd600?style=for-the-badge&labelColor=20252d)](CHANGELOG.md)
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

The browser edition uses WebAssembly and can use considerably more memory than the desktop app. For large conversions, the portable app is usually the safer choice.

## What it can create

- A normal single-video ROM
- One ROM that plays several clips in sequence
- One ROM with a startup clip-selection menu
- Separate `.gba` files packaged into a ZIP
- Numbered ROM parts when a single video cannot safely fit on one cartridge

Generated video uses fixed **120×80** indexed frames and expands them to the GBA's **240×160** display. Legacy presets use signed 8-bit mono PCM at **16,384 Hz**, hardware DMA audio playback, and keyframe/delta video compression with raw-frame fallback.

**Extreme optimization (Experimental)** can additionally use scene-aware adaptive keyframes and block-based 4-bit IMA ADPCM.

## Features

### Video and output

- Drag in one or more videos
- Trim start and end times
- Playback-speed controls
- Fit with bars, crop, or stretch
- Four GBA-friendly frame-rate choices
- Shared or per-scene palettes
- Dithering off, ordered, or error diffusion
- Keyframe/delta compression or uncompressed video
- Automatic raw-frame fallback when compression would be larger
- Configurable 3, 5, 10, or 15-second seek step
- Optional looping per clip
- Editable 12-character ROM title
- Clip reordering and per-clip menu titles
- Optional SRAM save/resume support

### Audio

- Mono mix, left channel, right channel, or disabled audio
- Per-clip volume
- Optional loudness normalization
- Optional limiter
- Selected-channel preview in both desktop and web editions
- Standard PCM, Compact ADPCM (Experimental), or Auto for ROM target under Extreme optimization
- Codec previews use the same 16,384 Hz PCM/ADPCM implementation written into the ROM

### Long videos and split ROMs

Single-video conversions can automatically split into multiple ROMs when the selected cartridge target is exceeded. The converter estimates the part count before encoding, verifies the real encoded size of each accepted part, and continues from the exact ending timestamp.

Optional split controls provide:

- A **1–32 MiB** ROM-data target with 20 MiB, 30 MiB, and Maximum shortcuts
- Optional maximum source duration per part using `MM:SS`
- Chapter-aware split points when chapter metadata is available
- Persistent recovery of completed parts after cancellation or failure
- Progress such as `Part N of approximately M` and `18:42 / 50:00`
- Output as `NAME_PART_01.gba`, `NAME_PART_02.gba`, and `PARTS.txt` inside one ZIP

### Native title cards

Split ROMs can show a native **240×160** title card before each part.

- First frame of the part as the default background
- Adjustable background darkening
- Source filename as the default title
- Automatic `Part {part}` subtitle
- Shared settings or per-part overrides
- Independent title/subtitle size, alignment, text colour, and outline colour
- Wait for `A` or timed start
- Optional skip and fade into video
- Pixel-accurate RGB555 preview in both editions

### Menu design

When **One ROM — clip menu** is selected, the menu can be customized directly in the editor.

- Live 120×80 pixel preview using the same indexed data as the GBA player
- Exact 3×5 font, divider lines, text coordinates, and selector shape
- Built-in **Classic dark**, **Ocean Wave — static**, **Ocean Wave — animated**, and **Blue Wave — animated** backgrounds
- Selectable normal, selected, and outline colours
- Optional one-pixel UI outline
- Custom PNG, JPEG, WebP, or GIF backgrounds
- GIF backgrounds sampled to at most 16 looping frames
- One, two, or three title columns
- Four-direction D-pad navigation
- Remembered menu selection and separate resume position for each clip when SRAM resume is enabled

Menu theme data is embedded directly into the ROM and included in the size estimate.

### Latin and Cyrillic text

Version **0.12.2** expands the existing 3×5 pixel font into one shared Latin + Cyrillic font, allowing mixed text in menu titles and title cards.

- One shared character set instead of separate language fonts
- Full Ukrainian support, including `Ґ Є І Ї`
- UTF-8 text remains readable in `.gbavideo` project files
- Menu titles and title cards count visible characters rather than UTF-8 bytes
- `{part}` works in strings such as `Частина {part}`
- Common typographic apostrophes, quotes, dashes, and ellipses are normalized
- Unsupported characters are reported instead of silently disappearing
- The internal 12-byte cartridge header remains ASCII-safe through transliteration

Lowercase Cyrillic input is accepted and rendered in the same uppercase-style 3×5 pixel font used by the player.

### Extreme optimization (Experimental)

Extreme optimization is isolated from the stable presets. **Best quality**, **Balanced**, **Long video**, **Smallest ROM**, and **Custom** keep the established PCM/fixed-keyframe path unless Extreme optimization is explicitly selected.

The experimental mode can:

- Analyze representative low-resolution samples
- Compare frame rate, palette, dithering, adaptive keyframes, and audio storage
- Estimate ROM-size ranges and quality trade-offs
- Apply a recommended candidate while keeping manual controls editable
- Detect scene boundaries using motion, brightness, fades, flash rejection, and minimum scene length
- Use adaptive keyframes while preserving seeking
- Use 4-bit IMA ADPCM at 16,384 Hz to reduce audio storage

> [!WARNING]
> Compact ADPCM and adaptive encoding are experimental. They are covered by encoder/decoder and ROM-structure tests, but have not been qualified on every flash cartridge or long-duration real-hardware setup. Use Standard PCM when reliability matters more than capacity.

For detailed release history, see [`CHANGELOG.md`](CHANGELOG.md).

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

## Save/resume support

When **Save/resume position** is enabled, the ROM declares `SRAM_V113` save memory.

A single-video ROM stores its playback frame. Menu and playlist ROMs additionally store the most recently selected clip and a separate resume frame for each clip.

When a saved position is available:

- `A` continues from the saved time
- `B` restarts that clip from the beginning

Finishing a clip normally clears only that clip's saved frame. SRAM behavior should be tested on the intended flash cartridge because save handling differs among cartridges and emulator configurations.

## Desktop app

Download the portable ZIP from the repository's **Releases** page, extract it, and run:

```text
GBA Video Maker.exe
```

No installer, Python runtime, administrator access, or devkitARM installation is required. Official portable packages place a pinned Windows x64 `ffmpeg.exe` beside the application, and the app does not download or update executables at runtime.

> [!NOTE]
> Windows may warn about an unsigned executable. Building from source or signing release binaries is the reliable way to establish publisher trust.

## Web app

The standalone browser edition lives under [`website/`](website/) and uses the same embedded `assets/player_stub.bin` as the desktop converter, so both editions generate ROMs with the same playback engine.

The web edition supports the same main conversion workflow, including output modes, quality controls, long-video splitting, title cards, project save/open, menu design, audio preview, and Extreme optimization controls.

Browser processing is local. Selected videos are copied into FFmpeg's in-browser filesystem and are not uploaded to a project server. Browser security does not allow a saved project to silently reopen local files, so source files must be selected again and are relinked by filename.

### Run the website locally

```bash
cd website
npm install
npm test
npm run dev
```

Build the static site used by GitHub Pages:

```bash
cd website
npm run build
npm run preview
```

GitHub Pages deployment is configured in [`.github/workflows/pages.yml`](.github/workflows/pages.yml).

## ROM format

| Property | Value |
|---|---|
| Display | 240×160 Mode 4 |
| Encoded video | 120×80 indexed frames |
| Video colours | RGB555 palette entries 0–249 |
| UI colours | Palette entries 250–255 |
| Audio | Signed 8-bit mono PCM at 16,384 Hz; optional experimental IMA ADPCM |
| Maximum ROM size | 32 MiB |
| Padding | Next power-of-two cartridge size |

Delta compression stores periodic full keyframes and changed-byte runs between them. Each clip includes frame and seek metadata so the player can reconstruct frames after seeking.

## Privacy and security

### Desktop

The desktop GUI is served locally on a random `127.0.0.1` address with a random session token. FFmpeg processes selected videos locally. Temporary files are created in the operating system's temporary directory and removed after normal shutdown.

### Browser

The GitHub Pages site is static. Conversion runs locally in the browser with WebAssembly and does not require a video-upload server.

See [`SECURITY.md`](SECURITY.md) for vulnerability-reporting guidance.

## Build from source

### Requirements

- Go 1.23 or newer
- FFmpeg for conversions and integration tests
- LLVM tools (`clang`, `ld.lld`, and `llvm-objcopy`) to rebuild the GBA player
- Node.js and npm for the standalone website

### Go checks

```bash
go fmt ./...
go vet ./...
go test ./...
```

### Rebuild the GBA player

After changing files under `player/`:

```bash
./player/build.sh
```

### Build the Windows executable

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-H windowsgui -s -w" \
  -o "GBA Video Maker.exe" .
```

PowerShell helpers:

```powershell
./scripts/build-windows.ps1
./scripts/package-release.ps1 -Version 0.12.2
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
