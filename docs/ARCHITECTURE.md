# Architecture

## Overview

```text
One or more video files
        │
        ▼
Local web UI on 127.0.0.1
        │
        ├─ FFmpeg inspection
        ├─ start/end preview generation
        ├─ audio preview generation
        ├─ frame extraction and 120×80 scaling
        ├─ scene detection and palette quantization
        ├─ optional ordered/error-diffusion dithering
        ├─ keyframe/delta compression
        ├─ audio normalization, limiting, and resampling
        └─ single-ROM, collection-ROM, or batch-ZIP assembly
```

The Windows executable embeds the HTML, CSS, JavaScript, conversion code, and a prebuilt ARM7TDMI player stub.

## Local GUI lifecycle

1. `runWebApp` creates a random token and temporary workspace.
2. The server binds only to an automatically selected `127.0.0.1` port.
3. Windows launches Edge or Chrome in app mode, with the default browser as fallback.
4. Multipart uploads can contain one or several videos.
5. Files are inspected locally and exposed to the UI as a clip list.
6. Preview endpoints invoke FFmpeg for exact framing and selected-channel audio samples.
7. Conversion progress is polled through token-prefixed local endpoints.
8. The server exits after the app window disappears and the heartbeat expires.

## Frame conversion

FFmpeg applies trimming, speed, framing, and frame-rate filters and writes RGB24 frames at 120×80.

The Go encoder then:

1. detects visual scene changes when per-scene palettes are enabled;
2. samples each scene and builds a 250-colour median-cut palette;
3. reserves palette entries 250–255 for black, dark gray, white, yellow, red, and green UI colours;
4. quantizes each frame with off, ordered, or Floyd–Steinberg-style error-diffusion dithering;
5. writes a palette index for each frame when several palettes exist.

## Delta stream

Compressed clips contain a frame-offset table. Each indexed record has an eight-byte header:

```text
u32 record_type    0 = full keyframe, 1 = delta
u32 payload_size
u8  payload[payload_size]
padding to 4-byte alignment
```

A full keyframe stores all 9,600 indexed pixels. A delta stores repeated structures:

```text
u16 unchanged_byte_count
u16 changed_byte_count
u8  changed_bytes[changed_byte_count]
```

The converter forces periodic keyframes and chooses a full frame whenever a delta is not smaller. The player reconstructs arbitrary seek targets from the nearest preceding keyframe and applies subsequent deltas. Normal sequential playback copies the current frame buffer and applies only the next record.

## Audio

FFmpeg converts the selected audio stream/channel to signed 8-bit mono PCM at 16,384 Hz. Optional filters include:

- playback-speed compensation through chained `atempo` filters;
- user gain;
- single-pass loudness normalization;
- peak limiting.

Each clip contains a frame-indexed audio-offset table. Seeking restarts DMA at the sample corresponding to the target frame.

Runtime volume uses the GBA Direct Sound A hardware gain bit, providing 50% and 100% levels, plus a routed-off 0% state. Mute is maintained separately.

## GBV5 ROM layout

```text
0x000000  GBA header and ARM player
0x003F00  64-byte GBV5 global metadata
0x004000  clip descriptor table and clip assets
```

The player template is exactly 16 KiB. Each 96-byte clip descriptor includes frame count, stream offsets, audio information, palette information, seek length, flags, title, and compressed/uncompressed sizes.

Clip assets may contain:

- one or more 512-byte palettes;
- a per-frame palette table;
- a compressed-frame offset table;
- compressed or raw video data;
- a frame-indexed audio seek table;
- PCM audio.

The final ROM is padded to a power-of-two cartridge size and rejected above 32 MiB.

## Player memory and display

The player keeps two 9,600-byte indexed frame buffers in EWRAM. One represents the current frame and the other is prepared for the next frame. Mode 4 page flipping displays each logical 120×80 pixel as a 2×2 block on the physical 240×160 screen.

Palette changes are copied during VBlank immediately before page flipping, preventing the visible page from temporarily using the next scene's palette.

## SRAM resume

Resume-enabled ROMs embed the `SRAM_V113` signature and store:

- save magic;
- selected clip index;
- current frame;
- integrity value.

The player validates these values before showing the continue/restart prompt.

## Security boundaries

- Listener restricted to `127.0.0.1`
- Random token in all API paths
- Upload-size cap
- Pinned FFmpeg download hash
- Argument-array process execution rather than shell commands
- Sanitized output names
- Temporary per-session workspace


In multi-video menu ROMs, the selected clip uses a clear pixel-art arrow. Finishing a clip or pressing B returns to the menu.


## Playback clock

Timer 2 runs at 16,384 Hz and Timer 3 cascades to form a 32-bit clock. Frame deadlines are derived from the selected frame rate, so decoding and rendering time is included rather than accumulating as drift. Audio timestamps are normalized during conversion with FFmpeg asynchronous resampling.
