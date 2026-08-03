# GBA Video Maker

A portable Windows app that converts ordinary video files into self-playing Game Boy Advance ROMs.

Version **0.10.1** adds project-wide or per-clip output resolution choices—Efficient 120×80, Enhanced 180×120, and Native 240×160—on top of the v0.10 hardware renderer, tiled menu, sprite HUD, hybrid video codec, adaptive keyframes, and PCM/IMA ADPCM audio.

## Download and run

Download the portable ZIP from the repository's **Releases** page, extract it, and run `GBA Video Maker.exe`.

No installer, Python runtime, administrator access, or devkitARM setup is required. Microsoft Edge or Google Chrome is used for the local app window, with the default browser as a fallback.

Official portable releases include a pinned Windows x64 `ffmpeg.exe` beside the app. The application does **not** download, install, or update executables at runtime. Source builds can use `ffmpeg.exe` beside the app or an `ffmpeg` available through `PATH`.

> Windows may still warn about an unsigned executable. Building from source or code-signing release binaries are the reliable ways to establish publisher trust.

## Conversion features

- Select or drag one or several videos
- Live start/end preview using the chosen crop, fit, stretch, and output-resolution settings
- Shared settings for batch conversion
- Separate `.gba` files in a ZIP, or several clips in one ROM with a startup menu
- Trim start and end times
- Per-project or per-clip video resolution: Efficient 120×80, Enhanced 180×120, or Native 240×160
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
- Hybrid adaptive video compression with raw, byte-delta, repeat, and 8×8 tile-delta records
- PCM or blocked IMA ADPCM audio storage; ADPCM is the default
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

When several videos are converted as one menu ROM, the player opens a hardware-tiled clip-selection screen:

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
- Encoded frames: selectable 120×80, 180×120, or native 240×160
- Mode 4 BG2 affine scaling maps the selected source resolution to the 240×160 display
- DMA3 row uploads and page flipping
- Indexed RGB555 colour
- Palette entries 0–245 for video
- Palette entries 246–255 reserved for menu/HUD colours
- Signed 8-bit mono audio at 16,384 Hz
- Blocked IMA ADPCM by default, with PCM available
- Power-of-two ROM padding up to the 32 MiB GBA limit

Hybrid compression evaluates every frame and can store:

- a complete keyframe;
- changed-byte runs;
- a repeat-previous-frame marker;
- changed 8×8 tiles.

The encoder selects the smallest valid record automatically, inserts keyframes at strong scene changes, limits reconstruction-chain length, and falls back to a full frame whenever compression would not help. Low-motion animation, screen recordings, pixel art, and static scenes benefit most. Noisy or rapidly changing footage can remain close to raw size.

The result panel reports raw, byte-delta, tile-delta, and repeat-frame counts plus video and audio storage ratios.

## Palette modes

**Shared palette** uses one palette for the entire clip and is the default.

**Per-scene palette** analyzes coarse frame differences, begins a new palette at strong scene changes, and periodically refreshes very long scenes. The ROM switches palettes in sync with page flips.

Dithering choices:

- **Off** — cleanest pixels and fastest conversion
- **Ordered** — stable pattern and good default balance
- **Error diffusion** — highest detail, slower conversion, potentially busier pixels

## Resolution modes

- **Efficient — 120×80:** smallest frames and longest practical duration; each source pixel is displayed as a 2×2 block. This remains the default.
- **Enhanced — 180×120:** 2.25× as many source pixels as Efficient and a visible detail improvement, while still using hardware scaling.
- **Native — 240×160:** full GBA display resolution with no enlargement, but approximately four times the raw video data of Efficient.

Resolution can be inherited from project defaults or overridden for individual clips. The preview deliberately downsamples to the chosen source resolution and scales back with nearest-neighbour filtering, so it approximates the pixel detail that the ROM will display. The final result still depends on palette quantization, dithering, frame rate, compression, and the source video.

## Hardware renderer

The ROM no longer expands frames pixel-by-pixel in software. DMA copies each selected-resolution frame into the hidden Mode 4 page, and BG2 affine scaling maps 120×80 or 180×120 to the 240×160 display; native 240×160 uses a 1:1 affine matrix. Temporary mute, volume, seek, and loop feedback use OBJ sprites. Collection menus use tiled backgrounds plus a sprite arrow, so blinking and selection movement require only tilemap/OAM updates.

## Privacy model

The GUI is served by the executable on a random loopback address under `127.0.0.1`. A random session token is included in API paths. Video processing happens locally through FFmpeg; selected videos are not uploaded to a remote service.

Temporary files are stored in the operating system's temporary directory and removed when the session closes normally.

## Antivirus-friendly source layout

The browser interface is kept in readable files under `web/` and embedded at build time. The Go source no longer contains a compressed one-line browser script or any code that downloads an executable and starts it. Portable releases instead place a verified `ffmpeg.exe` beside the application.

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
./scripts/package-release.ps1 -Version 0.10.1
```

## Project layout

```text
player/                  ARM7TDMI playback-engine source and build scripts
assets/player_stub.bin   Prebuilt 44 KiB GBV6 player template
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