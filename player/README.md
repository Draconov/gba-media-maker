# Embedded GBA Media Player

Freestanding ARM7TDMI runtime for GBA Media Maker v0.13.0.

It remains compatible with GBV5 video descriptors and adds dispatch for audio-only and static-image descriptors. Video uses the existing indexed Mode 4 path. Audio and images use native 240×160 RGB555 Mode 3 screens; audio overlays a Now Playing UI and streams PCM or block IMA ADPCM through Direct Sound/DMA.

Build with `./player/build.sh` on Linux/macOS or `./player/build.ps1` in PowerShell. Requirements are Clang/LLVM (`clang`, `ld.lld`, `llvm-objcopy`). The resulting `assets/player_stub.bin` is padded to exactly 32768 bytes.

## Runtime controls

Video/audio playback uses `A` pause/resume, `B` restart/menu, `L`/`R` for previous/next media, D-pad Left/Right for seek (video frame-step while paused) with ~0.3 s hold-repeat, `START` for the three HUD modes, and `START+SELECT` for help. Up/Down volume and `SELECT` mute/unmute are available only when the current media actually has audio. GIFs, silent videos, videos converted with **No audio**, and images do not expose audio controls. For images, `A` pauses/resumes only an enabled slideshow and does nothing in manual-viewer mode. Menu navigation is column-based with Up/Down inside a column and Left/Right between columns.
