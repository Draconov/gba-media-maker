# Third-party notices

GBA Media Maker is distributed under the project's Non-Commercial Contribution License v1.0, while third-party components and assets remain under their own licences.

## FFmpeg — desktop conversion

FFmpeg is not committed to the source repository as a normal source dependency. Official portable Windows release packages may include a pinned Windows x64 `ffmpeg.exe` beside the application.

The release workflow currently pins BtbN/FFmpeg-Builds release:

```text
autobuild-2026-08-07-13-13
```

Packaging downloads that release's `checksums.sha256`, discovers the matching **non-shared win64 LGPL** ZIP from the manifest instead of assuming a floating `*-latest-*` filename, verifies its SHA-256 hash, extracts `ffmpeg.exe`, and checks for software AV1 decoding support (`libdav1d` or `libaom`).

- FFmpeg project: <https://ffmpeg.org/>
- BtbN/FFmpeg-Builds: <https://github.com/BtbN/FFmpeg-Builds>
- Pinned release: <https://github.com/BtbN/FFmpeg-Builds/releases/tag/autobuild-2026-08-07-13-13>

FFmpeg and its linked libraries are distributed under their own licences. The release packaging intentionally selects the LGPL build variant. Review the FFmpeg/BtbN licensing information before redistribution.

## ffmpeg.wasm — browser conversion

The browser edition uses the `@ffmpeg/ffmpeg` / `@ffmpeg/util` JavaScript packages and an FFmpeg WebAssembly core for local browser processing.

Current package versions are declared in `website/package.json` and remain subject to their respective upstream licences and the licences of the FFmpeg build they load/use.

- ffmpeg.wasm project: <https://github.com/ffmpegwasm/ffmpeg.wasm>
- FFmpeg project: <https://ffmpeg.org/>

## Built-in audio artwork presets

The 20 images under:

```text
assets/audio-artwork/
website/public/audio-artwork/
```

are third-party artwork from the **Grainient — Dither Gradient I** collection, supplied by the project maintainer under a separate licence that permits their inclusion/redistribution with this project.

These artwork files are **not relicensed under the project's Non-Commercial Contribution License**. Their use and redistribution remain subject to the applicable artwork licence.

- Collection source: <https://grainient.supply/collections/dither-gradient-i>

## Platform/trademark notice

Game Boy Advance, Nintendo, and related marks belong to their respective owners. GBA Media Maker is an independent project and is not affiliated with or endorsed by Nintendo.
