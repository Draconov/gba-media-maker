# Architecture

## Overview

```text
Video file
   │
   ▼
Local web UI ──multipart upload──▶ Go HTTP server on 127.0.0.1
                                      │
                                      ├─ FFmpeg inspection
                                      ├─ frame extraction and scaling
                                      ├─ audio extraction and resampling
                                      ├─ palette quantization
                                      └─ ROM assembly
                                                │
                                                ▼
                                         downloadable .gba
```

The executable contains the HTML, CSS, JavaScript, conversion code, and the GBA player stub. It does not require a separate web server.

## GUI lifecycle

1. `runWebApp` creates a random session token and temporary working directory.
2. The server binds to an automatically selected loopback port.
3. Windows launches Edge or Chrome with `--app=<local URL>`. The default browser is the fallback.
4. The opening page remains minimal until a video is selected.
5. Upload, inspection, conversion, progress polling, and download use token-prefixed local endpoints.
6. The server shuts down after the app closes or the heartbeat expires.

## Conversion pipeline

FFmpeg decodes the selected section, applies speed and framing filters, and emits RGB24 frames at the selected playback rate. The Go encoder quantizes colours to the GBA RGB555 palette format and stores 120×80 indexed frames. Palette entries 250-255 are reserved for a stable black, dark, white, yellow, red, and green HUD palette, while source pixels use entries 0-249.

Audio is decoded to signed 8-bit mono PCM at 16,384 Hz. The converter also writes an aligned audio-offset entry for every video frame. This lets the player restart DMA at the matching sample when `L` or `R` seeks by approximately five seconds. Muting changes Direct Sound routing while DMA continues, so unmuting resumes at the correct position. Each 120×80 source pixel is expanded to a 2×2 block on the 240×160 display, after which the player draws its progress bar, time display, mute badge, and temporary seek indicator.

The ROM builder combines:

- embedded player code
- metadata header
- palette data
- per-frame audio seek table
- indexed frames
- optional PCM audio
- power-of-two cartridge padding

The output is rejected when it would exceed 32 MiB.

## Security boundaries

- The HTTP listener uses `127.0.0.1`, not all interfaces.
- API routes include a random session token.
- Upload size is capped.
- FFmpeg is downloaded over HTTPS and accepted only after SHA-256 verification.
- FFmpeg commands are executed with argument arrays rather than through a shell.
- Temporary and output filenames are generated or sanitized by the backend.


The playback HUD leaves a black 1-pixel row above and below the loop icon so it sits cleanly above the progress line.
