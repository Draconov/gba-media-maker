# Third-party notices

## FFmpeg

FFmpeg is not stored in this source repository. Official portable release ZIPs include a pinned Windows x64 `ffmpeg.exe` beside the application so the app never downloads or installs an executable at runtime.

The release workflow uses the **LGPL** variant from BtbN/FFmpeg-Builds, pinned to release tag `autobuild-2026-08-07-13-13`. It downloads `ffmpeg-master-latest-win64-lgpl.zip`, verifies the archive against the `checksums.sha256` file published with that pinned release, extracts `ffmpeg.exe`, and verifies that the build contains a software AV1 decoder (`libdav1d` or `libaom`). This avoids depending on AV1-capable GPU hardware when converting AV1 source videos.

- FFmpeg project: <https://ffmpeg.org/>
- Binary build source: <https://github.com/BtbN/FFmpeg-Builds>
- Pinned binary release: <https://github.com/BtbN/FFmpeg-Builds/releases/tag/autobuild-2026-08-07-13-13>

FFmpeg and its linked libraries are distributed under their own licenses. The selected BtbN `lgpl` build intentionally excludes GPL-only dependencies. Review the FFmpeg and binary-provider licensing information before redistribution.
