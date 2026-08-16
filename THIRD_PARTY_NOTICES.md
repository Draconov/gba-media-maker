# Third-party notices

GBA Media Maker is distributed under the project's Non-Commercial Contribution License v1.0, while third-party components and assets remain under their own licences.

## FFmpeg — desktop conversion

FFmpeg is not committed to the source repository as a normal source dependency. Official desktop release packages bundle a native FFmpeg executable for their target platform.

### Windows and Linux

The release workflow pins the retained BtbN/FFmpeg-Builds release:

```text
autobuild-2026-07-31-14-10
```

Packaging downloads that release's `checksums.sha256`, discovers the matching **non-shared LGPL** archive for Windows x64 or Linux x86_64, verifies the archive SHA-256, extracts FFmpeg, and checks software AV1 decoding support where the target binary can be executed on the build runner.

- FFmpeg project: <https://ffmpeg.org/>
- BtbN/FFmpeg-Builds: <https://github.com/BtbN/FFmpeg-Builds>
- Pinned release: <https://github.com/BtbN/FFmpeg-Builds/releases/tag/autobuild-2026-07-31-14-10>

### macOS

macOS packages build FFmpeg from pinned upstream source instead of redistributing a third-party macOS binary. The current source version is declared in `scripts/ffmpeg-pins.env`. The build disables dependency autodetection, enables `libdav1d` for software AV1 decoding, and rejects FFmpeg configurations containing `--enable-gpl`, `--enable-version3`, or `--enable-nonfree`. Any non-system dylibs used by the resulting executable are copied into the app bundle and their load paths are rewritten before packaging.

`libdav1d` remains under its own upstream licence.

FFmpeg and its linked libraries are distributed under their own licences. Review the relevant FFmpeg and dependency licensing information before redistribution.

## ffmpeg.wasm — browser conversion

The browser edition uses the `@ffmpeg/ffmpeg` / `@ffmpeg/util` JavaScript packages and an FFmpeg WebAssembly core for local browser processing.

Current package versions are declared in `website/package.json` and remain subject to their respective upstream licences and the licences of the FFmpeg build they load/use.

- ffmpeg.wasm project: <https://github.com/ffmpegwasm/ffmpeg.wasm>
- FFmpeg project: <https://ffmpeg.org/>

## Built-in audio artwork presets

The 20 images under:

```text
assets/audio-artwork/
```

are third-party artwork from the **Grainient — Dither Gradient I** collection, supplied by the project maintainer under a separate licence that permits their inclusion/redistribution with this project. Website copies are generated from this authoritative directory during web dev/test/build.

These artwork files are **not relicensed under the project's Non-Commercial Contribution License**. Their use and redistribution remain subject to the applicable artwork licence.

- Collection source: <https://grainient.supply/collections/dither-gradient-i>

## Platform/trademark notice

Game Boy Advance, Nintendo, and related marks belong to their respective owners. GBA Media Maker is an independent project and is not affiliated with or endorsed by Nintendo.
