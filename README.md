<div align="center">
  <img src="assets/app_icon.png" width="96" alt="GBA Media Maker icon">

# GBA Media Maker

**Turn videos, music, and images into playable Game Boy Advance ROMs.**

![Version](https://img.shields.io/badge/version-0.13.0-ffd600?style=for-the-badge&labelColor=20252d)

</div>

GBA Media Maker is the v0.13 evolution of GBA Video Maker. It keeps the existing video converter and GBV5-compatible ROM container while adding first-class audio and image entries, mixed-media collections, native 240×160 artwork/images, and a media-aware GBA runtime.

## What v0.13.0 adds

- **Video + Audio + Image** files in the same project and ROM.
- **Audio-only ROMs** from MP3, WAV, FLAC, OGG/Opus, M4A/AAC and other formats supported by FFmpeg.
- Embedded cover-art extraction for audio, with a generated fallback artwork screen.
- GBA **Now Playing** screen with title, artist, album, elapsed/total time, progress, pause, seek, volume and mute.
- **Native 240×160 static images** stored as RGB555 instead of 120×80 video frames.
- Image viewer/gallery controls and optional automatic slideshow timing (`0` = manual).
- Mixed media menu with `[V]`, `[A]`, and `[I]` entry tags.
- `.gbamedia` v2 project files; legacy `.gbavideo` v1 projects remain accepted.
- Existing video presets, title cards, menus, palette modes, long-video splitting, PCM and experimental IMA ADPCM are retained.

## Desktop workflow

1. Launch **GBA Media Maker**.
2. Choose or drag one or more media files.
3. Reorder them and choose **Single ROM**, **Playlist**, **Media menu**, or **Separate ROMs**.
4. Selecting a video, audio file, or image automatically exposes the settings relevant to that media type.
5. Create the output and test the `.gba` in an emulator or on hardware/flashcart.

The Windows app runs a local-only UI on `127.0.0.1` and invokes FFmpeg locally. Source media is not uploaded by the desktop application.

## GBA controls

### Video
- `A`: pause/resume
- `L` / `R`: seek
- `Up` / `Down`: volume
- `SELECT`: mute
- `START`: HUD
- `B`: return to media menu/restart, depending on ROM mode

### Audio
- `A`: pause/resume
- `L` / `R`: seek
- `Up` / `Down`: volume
- `SELECT`: mute
- `START`: Now Playing/HUD
- `SELECT + Left/Right`: previous/next media in playlist mode

### Images
- `Left` / `Right`: previous/next image
- `A`: toggle overlay
- `B`: return to media menu/restart
- Optional slideshow duration advances automatically; `0` keeps the image open manually.

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
