<div align="center">
  <img src="assets/app_icon.png" width="96" alt="GBA Video Maker icon">

# GBA Video Maker

**Turn ordinary videos into playable Game Boy Advance ROMs — in the browser or with the portable Windows app.**

[![Version](https://img.shields.io/badge/version-0.9.0-ffd600?style=for-the-badge&labelColor=20252d)](CHANGELOG.md)
[![Web App](https://img.shields.io/badge/TRY-WEB_APP-ffd600?style=for-the-badge&labelColor=20252d)](https://draconov.github.io/gba-video-maker/)
[![Desktop](https://img.shields.io/badge/DOWNLOAD-DESKTOP_APP-ffffff?style=for-the-badge&labelColor=20252d)](../../releases/latest)
[![License](https://img.shields.io/badge/license-MIT-ffffff?style=for-the-badge&labelColor=20252d)](LICENSE)

Fixed **120×80 v0.9 playback**, PCM audio, seeking, playlists, multi-clip menus, custom palettes, dithering, compression, and a custom GBA menu background.

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
- Single-video conversions automatically switch to numbered ROM parts inside one ZIP when one cartridge is not enough (desktop app)

The generated player keeps the proven v0.9 playback path:

- Fixed **120×80** indexed video, expanded cleanly to the GBA's 240×160 display
- Signed 8-bit mono PCM audio at **16,384 Hz**
- Hardware DMA audio playback and hardware playback clock
- Keyframe/delta compression with raw-frame fallback
- No newer ADPCM decoder, multi-resolution renderer, or aggressive frame-skipping scheduler

## Highlights

### Video and ROM controls

- **Automatic long-video split (desktop):** no special mode is required. Create a normal single-video ROM; when it cannot safely fit, the app selects the largest safe source-time segment for each part, verifies the encoded size, continues from the exact ending timestamp, and exports `NAME_PART_01.gba`, `NAME_PART_02.gba`, and `PARTS.txt` in one ZIP

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
- Audio preview in the desktop editor

### Multi-clip menu

- Custom blue-wave background
- One, two, or three title columns
- Total clip duration and current selection status
- Four-direction D-pad navigation
- Independent blinking OBJ-sprite arrow
- No unstable video thumbnails in the GBA menu


### Automatic long-video handling

For one source video, choose the normal **Single ROM** output and convert as usual. The desktop app checks the minimum required data before encoding and also verifies the real encoded size afterward. When one cartridge would exceed the safe limit, the result automatically changes from a `.gba` file to a `_PARTS.zip` package containing sequential ROMs plus `PARTS.txt`. No manual split mode is exposed.

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

The web edition currently includes the desktop converter's ROM-impacting options:

- Quality presets and custom mode
- Menu, playlist, and separate-ROM ZIP outputs
- Frame rate, framing, palette, dithering, and compression
- Audio channel, volume, normalization, limiter, and disabled audio
- Trimming, speed, looping, seek step, and resume
- Project defaults, per-clip overrides, and clip reordering

Browser processing is local: selected videos are copied into FFmpeg's in-browser filesystem and are not uploaded to a project server.

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

When **Save/resume position** is enabled, the ROM declares `SRAM_V113` save memory and stores the current clip and frame.

On the next launch:

- `A` continues from the saved time
- `B` restarts from the beginning

SRAM behavior should be tested on the intended flash cartridge because save handling differs among cartridges and emulator configurations.

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
./scripts/package-release.ps1 -Version 0.9.0
```

## Project layout

```text
player/                  ARM7TDMI player source and build scripts
assets/player_stub.bin   Prebuilt 32 KiB GBA player template
converter.go             FFmpeg pipeline, palettes, compression, and ROM assembly
web/                     Interface embedded in the Windows app
website/                 Standalone GitHub Pages converter
webapp.go                Local desktop HTTP API
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
