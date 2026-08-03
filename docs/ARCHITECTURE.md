# Architecture

## Overview

```text
One or more video files
        │
        ▼
Local web UI on 127.0.0.1
        │
        ├─ FFmpeg inspection and previews
        ├─ trim, speed, framing, and selected-resolution extraction
        ├─ shared/per-scene palette quantization
        ├─ hybrid frame analysis and compression
        ├─ PCM or blocked IMA ADPCM audio encoding
        └─ single-ROM, playlist-ROM, menu-ROM, or batch-ZIP assembly
```

The Windows executable embeds the readable web UI, Go conversion code, and a reproducibly built ARM7TDMI player template. Official release ZIPs add a pinned, checksum-verified Windows FFmpeg executable.

## Local GUI lifecycle

1. `runWebApp` creates a random token and temporary workspace.
2. The server binds only to an automatically selected `127.0.0.1` port.
3. Windows launches Edge or Chrome in app mode, with the default browser as fallback.
4. One or several local videos are inspected and represented as ordered clips.
5. Preview endpoints invoke FFmpeg for framing, timeline thumbnails, and selected-channel audio samples.
6. Conversion progress and codec statistics are exposed through token-prefixed local endpoints.
7. The server exits after the app window disappears and the heartbeat expires.

## Frame conversion and palettes

FFmpeg applies per-clip trimming, speed, framing, the chosen fixed output rate, and the selected clip resolution, then writes RGB24 frames at 120×80, 180×120, or 240×160.

The Go encoder then:

1. detects visual scene changes when per-scene palettes are enabled;
2. samples each scene and builds a 246-colour RGB555 palette;
3. reserves indices 246–255 for the menu and playback UI;
4. quantizes frames with no dithering, ordered dithering, or error diffusion;
5. writes a per-frame palette index when several palettes exist.

Shared palette mode stores one 512-byte palette. Per-scene mode stores additional palettes and a synchronized frame-to-palette table.

## GBV6 hybrid video stream

Compressed clips contain a 32-bit frame-offset table. Each record begins with:

```text
u32 record_type
u32 payload_size
u8  payload[payload_size]
padding to 4-byte alignment
```

Record types:

```text
0  full width×height-byte keyframe
1  changed-byte runs
2  repeat the previous decoded frame
3  changed 8×8 tiles
```

Byte-delta payloads contain repeated runs:

```text
u16 unchanged_byte_count
u16 changed_byte_count
u8  changed_bytes[changed_byte_count]
```

Tile-delta payloads begin with a changed-tile bitmap sized for `ceil(width/8) × ceil(height/8)` tiles. Every set bit is followed by that tile’s pixels; edge tiles store only their valid width and height, which supports 180×120 as well as dimensions divisible by eight. The converter compares byte delta, tile delta, repeat, and raw storage for every frame and keeps the smallest safe representation. Strong scene changes and a maximum chain length trigger adaptive keyframes. Arbitrary seeking reconstructs from the nearest preceding full frame; sequential playback applies only the next record.

## Audio storage

FFmpeg converts the chosen stream/channel to signed 8-bit mono at 16,384 Hz. Optional filters include speed compensation, gain, loudness normalization, asynchronous timestamp normalization, and limiting.

Two storage modes are supported:

- **IMA ADPCM** (default): independent 1,024-sample blocks with a four-byte predictor/index header and 512 bytes of nibbles. Typical storage is about 50.4% of PCM.
- **PCM**: signed 8-bit samples stored directly for maximum fidelity and the simplest playback path.

Each clip has a frame-indexed seek table. ADPCM entries are sample positions; PCM entries are aligned byte offsets. The ADPCM player decodes into two alternating 2,048-sample EWRAM buffers while DMA feeds Direct Sound A. Timer 1 interrupts at buffer boundaries and the main loop refills freed buffers.

## GBV6 ROM layout

```text
0x000000  GBA header and ARM player code/data image
0x00AF00  64-byte GBV6 global metadata
0x00B000  128-byte clip descriptors followed by clip assets
```

The embedded player template is exactly 45,056 bytes (44 KiB). A clip descriptor records stream offsets, frame/audio settings, palette information, title, codec flags, compression statistics, ADPCM block parameters, and original/stored byte counts.

Clip assets may contain:

- one or more 512-byte palettes;
- a per-frame palette table;
- a compressed-frame offset table;
- hybrid or raw video data;
- a frame-indexed audio seek table;
- blocked ADPCM or PCM audio.

The completed ROM is padded to a power-of-two cartridge size and rejected above the standard 32 MiB GBA ROM limit.

## Hardware-assisted video renderer

The decoder keeps two maximum-size 38,400-byte indexed frame buffers in EWRAM and validates every descriptor against `source_width × source_height`. DMA3 copies each source row into the corresponding row of the hidden 240-byte-wide Mode 4 page. The BG2 affine matrix is calculated per clip: 0.5 source-pixel steps for 120×80, 0.75 for 180×120, and 1.0 for native 240×160.

Framebuffer upload work is therefore 9,600 bytes for Efficient, 21,600 bytes for Enhanced, or 38,400 bytes for Native. Palette changes are copied during VBlank immediately before page flipping.

Transient playback elements—mute, volume, seek feedback, and the loop mark—use OBJ sprites in bitmap mode. The full time/progress HUD remains in the framebuffer so it can be pinned without maintaining a large sprite-font system.

## Tile-based collection menu

Collection menus use Mode 0 rather than a bitmap redraw loop:

- BG0: four-band blue vertical gradient;
- BG1: white/yellow title, status, and clip-name tiles;
- BG2: optional 48×32 8-bit selected-clip thumbnail;
- OBJ: blinking yellow selection arrow.

Moving the selection updates tilemap entries and one OAM object instead of redrawing a 240×160 bitmap. The menu supports one, two, or three columns and pages larger collections.

## Playback clock

Timer 2 runs at 16,384 Hz and Timer 3 cascades to form a 32-bit playback clock. Frame deadlines are derived from the selected frame rate, so decoding and rendering cost does not accumulate as audio/video drift. Pause, seek, help, and frame-step operations restart both timelines from the same frame position.

## SRAM resume

Resume-enabled ROMs embed the `SRAM_V113` signature and store:

- save magic;
- selected clip index;
- current frame;
- integrity value.

The player validates these values before offering continue or restart.

## Security boundaries

- listener restricted to `127.0.0.1`;
- strict Host and loopback checks;
- random token in API paths and state-changing requests;
- upload-size cap;
- pinned FFmpeg release hash;
- argument-array process execution rather than shell commands;
- sanitized output names;
- temporary per-session workspace.
