<div align="center">
  <img src="assets/icon.png" width="96" alt="GBA Media Maker icon">

# GBA Media Maker

**Turn videos, music, and images into playable Game Boy Advance ROMs.**

![Version](https://img.shields.io/badge/version-0.13.0-ffd600?style=for-the-badge&labelColor=20252d)

</div>

GBA Media Maker is the v0.13 evolution of GBA Video Maker. It keeps the existing video converter and GBV5-compatible ROM container while adding first-class audio and image entries, mixed-media collections, native 240×160 artwork/images, and a media-aware GBA runtime.

## What v0.13.0 adds

- **Video + Audio + Image** files in the same project and ROM.
- **Audio-only ROMs** from MP3, WAV, FLAC, OGG/Opus, M4A/AAC and other formats supported by FFmpeg.
- Embedded cover-art extraction for audio, with a generated fallback artwork screen.
- GBA **Now Playing** screen with editable 28-character song title and artist subtitle, elapsed/total time, progress, pause, seek, volume and mute.
- **Native 240×160 static images** stored as RGB555 instead of 120×80 video frames.
- Image viewer/gallery controls with an explicit **Enable slideshow** switch; turn it off for a manual viewer.
- Every multi-item collection ROM starts in the media menu, whether it is mixed media, video-only, music-only, or image-only; entries use `[V]`, `[A]`, and `[I]` tags.
- `.gbamedia` v2 project files; legacy `.gbavideo` v1 projects remain accepted.
- Existing video presets, title cards, menus, palette modes, long-video splitting, PCM and experimental IMA ADPCM are retained.

## Desktop workflow

1. Launch **GBA Media Maker**.
2. Choose or drag one or more media files.
3. Reorder them. A one-item project opens directly; every multi-item collection uses the **Media menu** automatically (or choose **Separate ROMs**).
4. Selecting a video, music file, or image exposes a dedicated Video, Music, or Image settings panel.
5. Create the output and test the `.gba` in an emulator or on hardware/flashcart.

The Windows app runs a local-only UI on `127.0.0.1` and invokes FFmpeg locally. Source media is not uploaded by the desktop application.

## GBA controls

### During playback

| Button | Action |
|---|---|
| `A` | Pause or resume |
| `B` | Restart the current media, or return to the media menu in menu ROMs |
| `L` / `R` | Previous / next media |
| D-pad `Left` / `Right` | Seek while playing; step one frame while paused. Holding repeats about every 0.3 seconds |
| D-pad `Up` / `Down` | Volume 0% / 50% / 100% |
| `SELECT` | Mute / unmute |
| `START` | Cycle HUD: hidden → time only → full |
| `L + R` | Quickly hide the HUD or restore the previous HUD mode |
| `START + SELECT` | Open the controls-help screen |
| `SELECT + L/R` | Legacy playlist shortcut: previous / next media without opening a menu |

Static image entries use the same media-navigation/HUD shortcuts; `A` pauses or resumes an automatic slideshow when a slideshow duration is configured.

### Media menu

| Button | Action |
|---|---|
| D-pad `Up` / `Down` | Move within the current column |
| D-pad `Left` / `Right` | Move between columns (and across menu pages when needed) |
| `A` | Play the selected media |

## Media formats

The native desktop build delegates decoding to FFmpeg, so practical source support includes common MP4/MKV/MOV/AVI/WebM videos; MP3/WAV/FLAC/OGG/Opus/M4A/AAC audio; and PNG/JPEG/WebP/BMP/TIFF-style images.

Generated video remains 120×80 indexed graphics expanded to 240×160 on the GBA. Audio artwork and static image entries use full 240×160 RGB555 screens. Audio playback remains mono at 16,384 Hz, using signed 8-bit PCM or the existing experimental IMA ADPCM path.

## Build

Requirements: Go 1.23+, FFmpeg for conversion/tests, and LLVM (`clang`, `ld.lld`, `llvm-objcopy`) when rebuilding the GBA runtime.

```bash
go test ./...
./player/build.sh
go build ./...
```

Windows release helpers live under `scripts/`. The release workflow discovers the pinned BtbN non-shared `win64-lgpl` FFmpeg archive through its `checksums.sha256` manifest.

## ROM compatibility

The global ROM metadata remains **GBV5 version 5**. v0.13 uses previously available descriptor flag bits for audio-only, image, and media-metadata records rather than creating an unnecessary container-version break. Legacy video descriptors continue to use the existing path.

The GBA header game code is `GM05` for newly generated media ROMs.

## Repository layout

- `converter.go` — media inspection, conversion, packing, audio/image/video pipelines
- `web/` — embedded desktop UI
- `player/` — freestanding ARM7TDMI runtime
- `assets/player_stub.bin` — rebuilt 32 KiB runtime stub embedded in generated ROMs
- `website/` — browser-edition source scaffold
- `scripts/` — Windows build/release helpers
- `.github/workflows/` — CI/release/pages workflows

## License

MIT. See [LICENSE](LICENSE) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
