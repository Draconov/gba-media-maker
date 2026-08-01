# GBA Video Maker

A portable Windows app that converts ordinary video files into self-playing Game Boy Advance ROMs.

Version **0.8.0** adds a substantially expanded player, real conversion previews, batch and multi-clip output, audio processing, scene palettes, and keyframe/delta video compression.

## Download and run

Download the portable ZIP from the repository's **Releases** page, extract it, and run `GBA Video Maker.exe`.

No installer, Python runtime, administrator access, or devkitARM setup is required. Microsoft Edge or Google Chrome is used for the local app window, with the default browser as a fallback.

On first use, the app can download a pinned Windows x64 FFmpeg executable and verify its SHA-256 checksum. You can instead place your own `ffmpeg.exe` beside the app.

> Windows may warn about an unsigned executable. Build it from source when you prefer not to run an unsigned download.

## Conversion features

- Select or drag one or several videos
- Live start/end preview using the chosen crop, fit, or stretch mode
- Shared settings for batch conversion
- Separate `.gba` files in a ZIP, or several clips in one ROM with a startup menu
- Trim start and end times
- Playback speed from 0.5× to 3×
- Four frame-rate choices
- Configurable 3, 5, 10, or 15-second seek step; default is 5 seconds
- Best quality, Balanced, Long video, Smallest ROM, and Custom presets
- Estimated cartridge size, frame count, and approximate duration limit
- Mixed mono, left channel, right channel, or no audio
- Audio-channel preview before conversion
- Optional loudness normalization and limiter
- Shared palette or scene-change palettes
- Dithering off, ordered dithering, or error-diffusion dithering
- Optional keyframe/delta video compression
- Optional looping
- Optional SRAM playback-position saving and resume prompt
- Editable 12-character ROM title

## Generated ROM controls

| Button | Action |
|---|---|
| `A` | Pause or resume |
| `B` | Restart the current clip |
| `L` / `R` | Seek backward or forward by the configured amount |
| Hold `L` / `R` | Repeat the seek approximately every 0.4 seconds |
| `D-pad Left` / `Right` while playing | Seek backward or forward |
| `D-pad Left` / `Right` while paused | Move one frame backward or forward |
| `D-pad Up` / `Down` | Raise or lower volume |
| `SELECT` | Mute or unmute |
| `START` | Cycle HUD: hidden, time only, time + progress/frame number |
| `L + R` | Quickly hide or restore the HUD |
| `START + SELECT` | Open the complete controls-help screen |

The GBA Direct Sound hardware provides two practical playback gain levels. The ROM therefore cycles among **0%, 50%, and 100%** volume while keeping mute as a separate `SELECT` control.

The seek popup displays the actual seek amount selected during conversion. The full HUD displays elapsed/total time, current frame number, progress, and the loop icon when looping is enabled.

## Multi-clip ROMs

When several videos are converted as one ROM, the player opens a clip-selection menu:

- `D-pad Up` / `Down` chooses a clip
- `A` starts the selected clip

Each clip keeps its own palette data, compressed frame stream, audio, seek table, duration, and title.

## Resume support

When **Save/resume position** is enabled, the generated ROM declares `SRAM_V113` save memory and stores the current clip and frame. On the next launch:

- `A` continues from the saved time
- `B` restarts that clip from the beginning

SRAM behavior should be tested on the intended flash cartridge because save handling varies among cartridges and emulator configurations.

## Video format and compression

- Display: 240×160
- Encoded frames: 120×80, expanded 2× by the player
- Indexed RGB555 colour
- Palette entries 0–249 for video
- Palette entries 250–255 reserved for stable HUD colours
- Signed 8-bit mono audio at 16,384 Hz
- Power-of-two ROM padding up to the 32 MiB GBA limit

Delta compression uses:

- periodic full keyframes;
- per-frame changed-byte runs between keyframes;
- automatic full-frame fallback when a delta would not be smaller;
- a frame index for seeking and keyframe reconstruction.

Low-motion animation and static scenes usually benefit most. Highly noisy or rapidly changing footage may remain close to its uncompressed size.

## Palette modes

**Shared palette** uses one palette for the entire clip and is the default.

**Per-scene palette** analyzes coarse frame differences, begins a new palette at strong scene changes, and periodically refreshes very long scenes. The ROM switches palettes in sync with page flips.

Dithering choices:

- **Off** — cleanest pixels and fastest conversion
- **Ordered** — stable pattern and good default balance
- **Error diffusion** — highest detail, slower conversion, potentially busier pixels

## Privacy model

The GUI is served by the executable on a random loopback address under `127.0.0.1`. A random session token is included in API paths. Video processing happens locally through FFmpeg; selected videos are not uploaded to a remote service.

Temporary files are stored in the operating system's temporary directory and removed when the session closes normally.

## Build from source

Requirements:

- Go 1.23 or newer
- FFmpeg for integration tests and conversions
- LLVM (`clang`, `ld.lld`, and `llvm-objcopy`) when rebuilding the embedded GBA player

Run checks:

```bash
go fmt ./...
go vet ./...
go test ./...
```

Rebuild the player after editing files under `player/`:

```bash
./player/build.sh
```

Build the Windows executable:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-H windowsgui -s -w" \
  -o "GBA Video Maker.exe" .
```

On PowerShell:

```powershell
./scripts/build-windows.ps1
./scripts/package-release.ps1 -Version 0.8.0
```

## Project layout

```text
player/                  ARM7TDMI playback-engine source and build scripts
assets/player_stub.bin   Prebuilt 16 KiB embedded player template
converter.go             FFmpeg pipeline, palettes, dithering, compression, ROM assembly
webapp.go                Local HTTP API and embedded Windows GUI
main_windows.go          Windows app-window launcher
main_other.go            Command-line test harness for non-Windows systems
process_*.go              Platform-specific command execution
webapp_test.go            Unit and end-to-end tests
.github/workflows         CI and release automation
```

More implementation detail is in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## FFmpeg and legal notes

FFmpeg is a separate open-source project distributed under its own licences and is not committed to this repository. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

Convert only video and audio you have permission to use. Game Boy Advance, Nintendo, and related marks belong to their respective owners. This project is independent and is not affiliated with or endorsed by Nintendo.

## License

The application source is available under the [MIT License](LICENSE).


The app follows the Windows light/dark appearance automatically and uses the included `assets/app_icon.png` artwork for its app-window icon.


When you add multiple videos, the app can now build one ROM that plays the clips in order, one ROM with a clip-selection menu, or a batch ZIP of separate ROMs.
The app icon also now uses the transparent-background version of the provided artwork.


The app icon now uses the latest user-provided icon artwork.

The Windows build embeds the app icon directly into the `.exe`; the included `tools/embedicon` helper is run automatically by the build scripts.


In multi-video menu ROMs, the selected clip uses a clear pixel-art arrow. Finishing a clip or pressing B returns to the menu.


Each loaded video now has a red remove button in the editor, making it easy to drop clips from a multi-video project.
