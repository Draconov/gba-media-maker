# Third-party notices

## FFmpeg

FFmpeg is not stored in this source repository. Official portable release ZIPs include a pinned Windows x64 `ffmpeg.exe` beside the application so the app never downloads or installs an executable at runtime.

The release workflow obtains `shaka-streamer-binaries==1.5.1` from PyPI, verifies the wheel SHA-256 value `240b5a649527f1ddd8b95e7856a0e830c5da02664be5d45032654d274ead7a8c`, extracts `ffmpeg.exe`, and places it into the portable release folder.

- FFmpeg project: <https://ffmpeg.org/>
- Binary build source: <https://github.com/shaka-project/static-ffmpeg-binaries>
- Packaging project: <https://pypi.org/project/shaka-streamer-binaries/1.5.1/>

FFmpeg is distributed under its own licenses. Review the FFmpeg and binary-provider licensing information before redistribution.
