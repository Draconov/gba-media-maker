# Architecture

This document describes the **GBA Media Maker v0.13.1** desktop, browser, converter, ROM, and GBA runtime architecture.

## Overview

GBA Media Maker has two front ends that target the same GBV5 ROM/player design:

```text
                         ┌──────────────────────────┐
                         │  GBA Media Maker v0.13  │
                         └────────────┬─────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
                    ▼                                   ▼
        Portable Windows app                  Browser / GitHub Pages
       Go + embedded web UI                  Vite + JavaScript UI
                    │                                   │
          native FFmpeg process                    ffmpeg.wasm
                    │                                   │
                    └──────────────┬────────────────────┘
                                   │
                         media-specific pipeline
                                   │
                 ┌─────────────────┼─────────────────┐
                 ▼                 ▼                 ▼
              Video             Audio             Image
           120×80 indexed     240×160 RGB555     240×160 RGB555
           + PCM/ADPCM        artwork + audio     native screen
                 │                 │                 │
                 └─────────────────┼─────────────────┘
                                   ▼
                         GBV5 v5 ROM assembly
                                   │
                         32 KiB ARM7TDMI player
                                   │
                                   ▼
                           Game Boy Advance
```

The desktop implementation is the reference converter. The website mirrors the same project model, naming rules, descriptor layout, menu/title-card formats, and current `assets/player_stub.bin` runtime.

## Repository layers

### Desktop application

- `main_windows.go` / `main_other.go` — application entry points
- `webapp.go` — loopback-only local HTTP API, session state, preview/conversion orchestration
- `web/` — embedded HTML, CSS, and JavaScript interface
- `converter.go` — FFmpeg inspection/extraction, encoding, splitting, and ROM assembly
- `audio_artwork.go` — default/custom audio artwork handling
- `menu_theme.go` — MTH1 menu-theme validation/embedding
- `title_card.go` — TCD1 title-card rendering/serialization
- `smart_encoding.go` — Extreme optimization analysis and recommendations

The Windows executable embeds the web interface and the prebuilt GBA runtime. Media decoding itself is delegated to a local `ffmpeg.exe`.

### Browser application

- `website/src/main.js` — UI state, ffmpeg.wasm jobs, conversion workflow, downloads
- `website/src/rom-core.js` — palette/compression logic and GBV5 assembly
- `website/src/rom.worker.js` — frame/palette conversion work
- `website/src/project-format.js` — `.gbamedia` v2 serialization/migration
- `website/src/parity-utils.js` — desktop-compatible naming/settings helpers
- `website/src/menu-themes.js` — stable menu theme/background conversion and preview
- `website/src/title-cards.js` — title-card editor/serialization
- `website/src/gba-text.js` — shared player-compatible glyph encoding
- `website/scripts/sync-player.mjs` — copies the authoritative player stub into the web build

The website is static. ffmpeg.wasm runs locally in the browser; there is no media-upload conversion backend.

## Desktop GUI lifecycle

1. `runWebApp` creates a random per-session token and temporary workspace.
2. The HTTP listener binds to `127.0.0.1` only.
3. The desktop shell launches a browser/app-mode window for the local interface.
4. Selected media is copied/registered in the local session and inspected with FFmpeg.
5. The UI requests previews, thumbnails, audio samples, menu/theme previews, and title-card previews from token-prefixed endpoints.
6. Conversion runs locally and reports progress through the same session API.
7. The finished ROM/ZIP/project file is downloaded from the local server.
8. Temporary session data is removed during normal shutdown.

## Media classification

The converter identifies each source as one of these runtime media kinds:

- **video** — normal video containers plus animated GIF
- **audio** — audio-only files
- **image** — static image files

Animated GIF is deliberately treated as video, not as a static image. GIF output always sets the loop flag.

Long-video automatic splitting applies only to video media. Audio and image entries are never routed through the long-video splitter.

## Shared project model

A project contains global defaults plus per-item overrides. Important per-item fields include:

- display/menu title;
- trim start/end;
- speed;
- framing mode;
- audio stream/channel/volume;
- loop;
- palette/dither settings for video;
- slideshow duration for images;
- song title/artist and artwork source for audio.

Current project files use:

```text
format:  gba-media-maker-project
version: 2
ext:     .gbamedia
```

Legacy `.gbavideo` / `GBA Video Maker Project` data remains loadable. Legacy `playlist` output values are migrated to the v0.13 media-menu mode.

## Output-mode policy

New projects resolve output as follows:

```text
1 media item
    └─ direct ROM

2+ media items
    ├─ Media menu ROM      (default/current collection model)
    └─ Separate ROMs ZIP   (explicit batch mode)
```

The menu rule does not depend on whether the collection is mixed media. Video-only, music-only, image-only, GIF/video, and mixed collections all use the same menu runtime.

The legacy playlist flag remains readable by the GBV5 player/converter for compatibility, but it is not exposed as the normal v0.13 collection choice.

## Video pipeline

Video retains the stable indexed-video pipeline established in GBA Video Maker (pre v0.13.0)

### FFmpeg extraction

FFmpeg applies:

- start/end trim;
- playback-speed filters;
- fit/crop/stretch framing;
- selected GBA frame rate;
- optional source audio selection/filtering.

Video frames are emitted as RGB24 at **120×80**.

### Palette and quantization

The encoder:

1. optionally detects scene boundaries;
2. builds shared or per-scene RGB555 palettes;
3. reserves runtime/UI palette slots;
4. quantizes each frame using off, ordered, or error-diffusion dithering;
5. stores per-frame palette selection when scene palettes are used.

### Frame compression

Compressed video contains full-keyframe and delta records. A record has an eight-byte header:

```text
u32 record_type    0 = full frame, 1 = delta
u32 payload_size
u8  payload[payload_size]
padding to 4-byte alignment
```

Delta payloads store alternating unchanged/changed runs. The encoder falls back to a full frame whenever the delta would not be smaller.

Fixed keyframe intervals are used by the stable presets. Extreme optimization can use adaptive keyframes and enhanced scene detection.

### Video audio

When enabled, FFmpeg converts the selected stream/channel to **16,384 Hz mono signed 8-bit PCM**. Optional processing includes:

- playback-speed compensation;
- per-item gain;
- loudness normalization;
- limiter.

Extreme mode can use the existing block IMA ADPCM format. Frame-indexed seek data maps playback frames to PCM byte offsets or decoded ADPCM sample positions.

A video with no audio stream, an animated GIF, or a video converted with **No audio** does not set `CLIP_FLAG_AUDIO`; the runtime therefore suppresses mute/volume controls and feedback for that entry.

## Animated GIF pipeline

Animated GIFs use the video path:

1. FFmpeg decodes one animation cycle;
2. frames are processed as normal 120×80 indexed video;
3. source audio is absent;
4. `CLIP_FLAG_LOOP` is forced on.

This keeps GIF playback, seeking/frame stepping, menu navigation, and the v0.12.2-style video HUD on the same tested runtime path.

## Audio-only pipeline

Audio-only entries have a synthetic frame/timeline count so they can share GBV5 navigation, seek, and resume infrastructure with video.

### Artwork selection

Each audio item chooses one of three artwork modes:

```text
embedded
    ├─ embedded cover found -> use it
    └─ no embedded cover    -> chosen default preset

default
    └─ chosen preset-01 ... preset-20

custom
    └─ user PNG/JPEG/WebP image
```

The built-in presets are stored under `assets/audio-artwork/`. Website copies live under `website/public/audio-artwork/`.

All selected artwork is reduced/cropped/fitted to a native **240×160 RGB555** screen (76,800 bytes) before ROM assembly.

### Audio stream

The audio stream uses the same 16,384 Hz PCM/experimental ADPCM storage as video audio. A synthetic timeline is generated from duration and selected VBlanks-per-frame. Its seek table maps timeline frames to audio samples.

### Media metadata

New audio entries write an **MMD2** metadata record:

```text
Offset  Size  Meaning
0x00      4   "MMD2"
0x04     28   title (GBA text encoding)
0x20     28   artist
0x3C     20   album
```

The runtime displays title and artist in Now Playing. Album remains available in the record for metadata compatibility/future use.

The player can still read the earlier `MMD1` layout when encountered.

### Audio display/runtime

Audio uses GBA Mode 3 with the native artwork in VRAM. The full Now Playing HUD contains title, artist, elapsed/total time, PLAY/PAUSE state, and a 4-pixel progress line.

Temporary seek/mute/volume overlays restore/redraw only their affected UI regions. Ordinary audio controls do **not** force-blank and recopy the whole 240×160 artwork, preventing the former white-screen flash during seek feedback.

## Static-image pipeline

Images bypass the video palette/compression path entirely.

1. FFmpeg applies fit/crop/stretch to **240×160**.
2. The converter writes a native little-endian RGB555 screen (76,800 bytes).
3. The descriptor has one visual frame.
4. The descriptor's `audio_sample_count` extension field is repurposed for image slideshow duration in milliseconds.

```text
image duration > 0  -> timed slideshow
image duration = 0  -> manual viewer
```

Images have no audio flag, audio stream, seek controls, volume, or mute. `A` toggles pause only when slideshow duration is non-zero.

## Menu/theme subsystem (MTH1)

The v0.13 menu uses the stable v0.12.2 theme/rendering system with media-aware titles/tags.

An optional `MTH1` record contains:

- RGB555 menu palette;
- one or more 120×80 indexed background frames;
- frame timing for animated backgrounds;
- normal/selected/outline colours;
- outline flag;
- shimmer metadata for built-in animated wave themes.

Supported menu backgrounds include:

- built-in static/animated themes;
- PNG/JPEG/WebP images;
- GIF animation;
- sampled video animation (source audio ignored).

Custom animated backgrounds are sampled to at most 16 looping frames. The desktop and website previews use the same 3×5 font data, selection-arrow geometry, logical 120×80 coordinates, and integer scaling assumptions as the player.

At runtime, the selection arrow is an OBJ sprite. Animated backgrounds are prepared on the hidden Mode 4 page and page-flipped on VBlank.

## Split-video title cards (TCD1)

When a single video is split into numbered ROM parts, each part may include a native `TCD1` title-card asset.

The converter:

1. extracts the configured part-relative source frame or background;
2. fits it to 240×160;
3. applies darkening/solid background settings;
4. renders title/subtitle with the shared GBA glyph set and RGB555 colours;
5. stores the finished native screen with timing/input flags.

The player shows this pre-rendered Mode 3 screen before normal playback. It can wait for `A` or a timer, optionally allow skipping, and optionally fade.

## Unified GBA text encoding

Project/UI strings remain UTF-8. Before ROM serialization, the shared glyph layer:

- normalizes supported punctuation;
- counts visible glyphs rather than UTF-8 bytes;
- keeps ASCII as ordinary one-byte values;
- maps supported Cyrillic into compact custom single-byte player codes;
- supports Ukrainian `Ґ Є І Ї` and Russian `Ё Ъ Ы Э`;
- reports unsupported glyphs;
- transliterates the cartridge-header title separately because that field must remain ASCII-safe.

Go, desktop JavaScript, website JavaScript, and the ARM player use matching 3×5 glyph definitions for user-visible menu/title-card text.

## GBV5 ROM layout

The global container remains **GBV5 version 5**.

```text
0x000000 ┌────────────────────────────────────┐
         │ GBA header + ARM7TDMI player       │
         │ fixed 32 KiB template              │
0x007FC0 ├────────────────────────────────────┤
         │ 64-byte GBV5 global metadata       │
0x008000 ├────────────────────────────────────┤
         │ 96-byte clip/media descriptors     │
         │ optional MTH1 menu theme           │
         │ optional TCD1 title card           │
         │ per-media palettes/indexes/data    │
         │ seek tables / audio / metadata     │
         └────────────────────────────────────┘
```

`assets/player_stub.bin` is exactly **32,768 bytes** and media assets begin at `0x8000`.

The finished ROM is padded to the next power-of-two cartridge size, with a minimum output size of 1 MiB and a maximum of 32 MiB.

### Global metadata

The 64-byte global metadata record stores:

- `GBV5` magic and version 5;
- global flags;
- media count;
- descriptor table pointer/size;
- legacy split-title fallback fields;
- optional MTH1 pointer;
- optional TCD1 pointer.

Current global flag bits:

| Bit | Value | Meaning |
|---|---:|---|
| 0 | `0x0001` | save/resume enabled |
| 1 | `0x0002` | legacy playlist mode |
| 2 | `0x0004` | title-screen/title-card metadata present |

New v0.13 multi-item projects normally leave the playlist bit clear, which makes any 2+ item ROM a media-menu ROM.

### 96-byte media descriptor

The existing GBV5 96-byte clip descriptor is retained. Important flag bits are:

| Bit | Value | Meaning |
|---|---:|---|
| 0 | `0x0001` | audio stream present |
| 1 | `0x0002` | loop |
| 2 | `0x0004` | compressed video |
| 3 | `0x0008` | scene palettes |
| 4 | `0x0010` | IMA ADPCM |
| 5 | `0x0020` | adaptive keyframes |
| 6 | `0x0040` | audio-only media |
| 7 | `0x0080` | static-image media |
| 8 | `0x0100` | media metadata record |

Bits 6–8 were previously unused, allowing v0.13 media extensions without changing the GBV5 container version.

For video, the descriptor fields keep their original meanings. For native audio/image entries:

- frame bytes become 76,800;
- source dimensions become 240×160;
- video data pointer refers to the native artwork/image screen;
- audio-only entries can use the video-index pointer for MMD metadata;
- image entries store slideshow milliseconds in the auxiliary sample-count field.

## GBA display paths

### Video / menu: Mode 4

Video uses two indexed 120×80 logical pages. Each source pixel is expanded to a 2×2 block on the physical 240×160 display. Palette changes and page flips are synchronized around VBlank.

The full video HUD retains the v0.12.2 layout, including:

- elapsed/total time;
- frame counter;
- progress line;
- yellow loop icon;
- centered seek feedback;
- mute/volume feedback when audio exists.

### Audio / image: Mode 3

Audio artwork and static images use direct 240×160 RGB555 pixels in Mode 3. Audio overlays are drawn over artwork. Image HUD rendering is intentionally simpler and never exposes audio controls.

## Playback controls and UI timing

The runtime uses the current v0.13 control layout:

```text
A                 pause/resume (image only when slideshow is enabled)
B                 restart direct media / return to media menu
L / R             previous / next media
Left / Right      seek; video/audio timeline step while paused
Up / Down         volume 0/50/100 only when audio exists
SELECT            mute/unmute only when audio exists
START             cycle HUD: hidden / time / full
START + SELECT    help
```

Removed legacy shortcuts are not recognized:

- `SELECT + L/R` media navigation;
- `L + R` quick HUD hide/restore.

Temporary feedback timers are centralized:

```text
HUD hold after seek     6 VBlanks  ≈ 0.10 s
seek badge              6 VBlanks  ≈ 0.10 s
mute badge              6 VBlanks  ≈ 0.10 s
volume badge            6 VBlanks  ≈ 0.10 s
held seek repeat       18 VBlanks  ≈ 0.30 s
```

Input polling occurs before frame-deadline handling so late video frames cannot starve keypad polling. UI timers are also advanced even when frames are late, preventing temporary badges from remaining stuck on screen.

## Playback clock and pause/resume

Video timing uses the established hardware playback clock derived from GBA timers rather than accumulating decoder delays frame-by-frame.

The video pause path preserves prepared frames when possible and explicitly invalidates them only when operations such as help, frame step, seek, or HUD redraw make that necessary. Resume can use a valid prepared frame or safely resume from the current frame if the prepared page was invalidated.

Audio uses the same timer domain to map current playback samples back to its synthetic frame/timeline index.

## Save/resume storage

When save/resume is enabled, the runtime writes directly to GBA SRAM space.

Current SRAM state stores:

```text
0x00  save magic
0x04  media count
0x08  remembered menu selection
0x10  per-media encoded resume positions
```

Each playable video/audio entry can have an independent resume position. Static images do not use a meaningful playback resume position.

Before playback, a valid non-zero saved video/audio position shows the restored confirmation screen:

```text
CONTINUE FROM
    MM:SS

 A CONTINUE
 B RESTART
```

Normal completion clears only that media item's saved position. Returning to the menu or switching media preserves the current position.

## Output naming parity

Desktop and website naming helpers intentionally match.

```text
single input:          <source base>.gba
multi-item menu:       GBA_Media_Collection.gba
multi-item batch ZIP:  GBA_Media_Collection.zip
batch ROM entry:       <source base>_GBA.gba
long-video archive:    <source base>_PARTS.zip
long-video part:       <source base>_PART_01.gba
project file:          <project name>.gbamedia
```

Unsafe filesystem characters are sanitized before output naming. Browser-added duplicate-download suffixes such as `(1)` are outside the application naming layer.

## Website/player synchronization

There is only one authoritative GBA runtime source/binary:

```text
player/*.S + player/main.c
          │
          ▼
assets/player_stub.bin
          │
          └── website/scripts/sync-player.mjs
                         │
                         ▼
               website/public/player_stub.bin
```

`website/public/player_stub.bin` is generated for build/test and should not be maintained as a separate hand-edited runtime.

GitHub Pages builds the player first, synchronizes the stub, runs website ROM/project tests, and builds the Vite site.

## Runtime build

`player/build.sh` and `player/build.ps1` compile:

- `startup.S`;
- `metadata.S`;
- `main.c`;
- `compiler_compat.S`.

The code is compiled for `arm-none-eabi` / ARM7TDMI with LLVM/Clang and linked with `ld.lld`. `llvm-objcopy` produces the binary and the build script pads it to exactly 32 KiB.

The current size-oriented build uses `-Oz`. The build must fail if the player code/data exceeds the fixed pre-asset region.

## Long-video splitting

Only a single video in direct-ROM mode can automatically enter the split pipeline.

The splitter:

1. estimates whether the configured cartridge/data target can fit;
2. finds a candidate source range;
3. converts/validates the actual encoded size;
4. shortens or extends candidate parts as needed;
5. continues from the previous accepted part's exact end time;
6. writes numbered ROMs plus `PARTS.txt` into `<name>_PARTS.zip`.

Optional chapter-aware boundaries, manual maximum part duration, recovery, and native title cards remain available from the v0.12.x video workflow.

## Extreme optimization

Extreme optimization remains isolated from stable presets.

A bounded low-resolution scan measures motion/detail/brightness/colour/scene-change properties, selects representative samples, ranks candidate settings, and can enable:

- adaptive keyframes;
- enhanced scene detection;
- experimental block IMA ADPCM;
- target-size-aware recommendations.

Stable named presets continue to use the established fixed-keyframe/PCM path unless Extreme is explicitly selected.

## Security boundaries

### Desktop

- Loopback listener only (`127.0.0.1`)
- Random per-session URL/API token
- Upload-size limits
- Temporary per-session workspace
- Argument-array process execution rather than shell interpolation
- Sanitized output filenames
- Official release packaging pins a BtbN FFmpeg release, discovers the correct non-shared win64 LGPL archive from its checksum manifest, verifies SHA-256, and verifies a software AV1 decoder

### Website

- Static application; no conversion upload API
- Source media remains in browser memory/virtual filesystem
- ffmpeg.wasm performs media processing locally
- Browser file security prevents silent reopening of arbitrary project source paths

## Compatibility strategy

v0.13 intentionally avoids unnecessary format churn:

- global container remains GBV5 version 5;
- legacy video descriptors remain valid;
- legacy MMD1 audio metadata remains readable;
- legacy project files are migrated on load;
- legacy playlist ROM flag remains understood by the runtime;
- stable v0.12.2 menu/theme and video HUD behavior is retained where it already worked well;
- new audio/image behavior is distinguished through descriptor flags rather than a new global ROM version.

This keeps existing video conversion/runtime behavior isolated from the new media features while allowing desktop and browser editions to share one current player.
