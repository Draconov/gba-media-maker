# Architecture

## Overview

```text
One or more video files
        │
        ▼
Local web UI on 127.0.0.1
        │
        ├─ FFmpeg inspection
        ├─ start/end and title-card preview generation
        ├─ cached/coalesced FFmpeg preview jobs
        ├─ audio preview generation
        ├─ frame extraction and 120×80 scaling
        ├─ scene detection and palette quantization
        ├─ optional Extreme representative scan and candidate ranking
        ├─ optional ordered/error-diffusion dithering
        ├─ fixed or scene-aware adaptive keyframe/delta compression
        ├─ audio normalization, limiting, and resampling
        ├─ standard PCM or experimental block IMA ADPCM
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

## Split-part title-card pipeline

When one source video is split into multiple ROMs, each part may include a `TCD1` asset generated before ROM assembly. The converter extracts the selected part-relative source frame, fits it to 240×160, applies background darkening, renders title/subtitle text with RGB555 colours, and stores the completed native screen with timing and input flags.

Desktop preview extraction is debounced and keyed by source path, timestamp, and framing mode. Identical requests share one in-flight job, completed frames are cached, and selecting another part cancels obsolete work. Preview FFmpeg commands use fast input seeking and one decoder thread so the editor cannot fan out into several CPU-heavy processes.

The player displays the pre-rendered screen before initializing normal 120×80 playback. It may wait for `A` or a timer, optionally fade to black, initialize the video, and fade the first frame in.


## Extreme optimization pipeline

Only the Extreme preset enables the v0.12 analysis path. A bounded FFmpeg scan produces at most roughly 120 RGB24 frames at 120×80. Shared motion, detail, brightness, colourfulness, changed-pixel, and scene scores drive representative sample selection, enhanced scene boundaries, candidate generation, quality estimates, and size ranges. Candidate settings remain ordinary converter options after application; users can still edit them manually.

Existing presets force PCM, the legacy scene detector, and fixed keyframe intervals. This feature gate prevents experimental metadata or runtime work from leaking into stable conversions.

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

PCM clips contain frame-indexed byte offsets. Experimental ADPCM clips contain frame-indexed decoded-sample positions and independently decodable 2,048-sample blocks. The player reconstructs signed 8-bit PCM into a double buffer and switches Direct Sound DMA between 4,096-sample halves while refilling the inactive half.

Runtime volume uses the GBA Direct Sound A hardware gain bit, providing 50% and 100% levels, plus a routed-off 0% state. Mute is maintained separately.

## GBV5 ROM layout

```text
0x000000  GBA header and ARM player
0x007F00  64-byte GBV5 global metadata
0x008000  clip descriptor table and clip assets
```

The player template is exactly 20 KiB. Each 96-byte clip descriptor includes frame count, stream offsets, audio information, palette information, seek length, flags, title, and compressed/uncompressed sizes.

Clip assets may contain:

- one or more 512-byte palettes;
- a per-frame palette table;
- a compressed-frame offset table;
- compressed or raw video data;
- a frame-indexed audio seek table;
- PCM audio or experimental block IMA ADPCM.

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



### Unified GBA text encoding (v0.12.2)

User-facing project strings remain UTF-8. A shared glyph mapper accepts Latin plus the union of Ukrainian and Russian Cyrillic, normalizes common typographic punctuation, and measures limits by Unicode glyph count rather than encoded byte length. Runtime menu/title-fallback fields remain fixed-size single-byte records: ASCII keeps its normal value, Cyrillic is mapped to `0x80`–`0xA4`, and `№` uses `0xA5`. The ARM player maps those codes to the same 3×5 bitmaps used by Go and JavaScript previews. The cartridge header is a separate ASCII-only field and transliterates Cyrillic rather than storing custom glyph bytes.

### Independent title/subtitle typography

Title-card settings keep legacy shared typography fields for v0.12.0 project compatibility, but v0.12.1 stores explicit title and subtitle size, alignment, text colour, and outline colour fields. The converter resolves old shared fields into the new per-row settings before pre-rendering the native 240×160 RGB555 title card, so the GBA player format itself does not change.
