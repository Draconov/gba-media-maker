# GBA Media Maker architecture — v0.13.0

## Media pipeline

```text
Video / Audio / Image inputs
          │
          ▼
      FFmpeg inspect
          │
     media kind dispatch
      ┌────┼────┐
      ▼    ▼    ▼
   Video Audio Image
  120×80  │   240×160 RGB555
 indexed  │
          ├─ 16,384 Hz mono PCM / IMA ADPCM
          └─ 240×160 RGB555 cover/fallback art
      └────┬────┘
           ▼
       GBV5 ROM
           ▼
  ARM7TDMI media runtime
```

The global container remains `GBV5`, version 5. v0.13 consumes descriptor flag bits that were previously unused:

- bit 6 (`0x0040`) — audio-only media
- bit 7 (`0x0080`) — static image media
- bit 8 (`0x0100`) — media metadata record

Legacy video descriptors are interpreted as before.

## Video

Video keeps the established 120×80 indexed-frame pipeline, scene/shared palettes, reserved UI colours, optional keyframe/delta compression, title cards, seek table, PCM/ADPCM audio, and long-video split support. The runtime expands the indexed source to the GBA's 240×160 Mode 4 display.

## Audio

Audio files no longer require a video stream. The converter:

1. inspects duration, audio streams/channels and global metadata;
2. extracts an embedded visual stream as cover artwork when available;
3. otherwise generates fallback artwork;
4. converts artwork to a native 240×160 RGB555 screen;
5. resamples selected audio to 16,384 Hz mono signed 8-bit PCM;
6. optionally encodes the existing block IMA ADPCM format under Extreme settings;
7. creates a frame-compatible seek/timeline table so GBV5 collection logic remains shared;
8. stores an optional `MMD1` record containing compact artist and album strings.

The GBA runtime uses a hardware playback timer as the audio clock and renders a Now Playing overlay over the native artwork.

## Images

Static images bypass the video quantizer entirely. FFmpeg applies the selected fit/crop/stretch framing and the converter writes a full 240×160 little-endian RGB555 screen (76,800 bytes). The descriptor uses one visual frame and stores slideshow duration in milliseconds in the otherwise media-specific extension field. A duration of `0` means manual viewer mode.

## Mixed media ROMs

Menu/playlist descriptors may freely combine video, audio and image entries. The runtime dispatches each descriptor by flags, while common title, loop, playlist navigation and SRAM selection state remain shared.

## Desktop UI

The embedded local UI uses **Add Media** and detects type from FFmpeg inspection/source extension. It hides video-only palette/compression/FPS controls for audio/images, hides audio controls for images, and exposes slideshow timing only for images. Project format v2 is `gba-media-maker-project`; v1 `gba-video-maker-project` files remain accepted.

## Runtime build

`player/build.sh` and `player/build.ps1` compile `startup.S`, `metadata.S`, `main.c`, and `compiler_compat.S` for `arm-none-eabi`, link with `ld.lld`, convert to binary, and pad `assets/player_stub.bin` to exactly 32 KiB. `compiler_compat.S` supplies the small EABI division/memclr helpers needed by the freestanding Clang build.
