# Contributing

Bug reports, reproducible test cases, documentation fixes, and focused pull requests are welcome.

## Before opening an issue

1. Use the latest release.
2. Check whether `ffmpeg.exe` exists beside the app or can be downloaded by it.
3. Attach `GBA Video Maker.log` when the app exits unexpectedly.
4. Include the source video's container, codec, resolution, duration, audio layout, and the selected conversion settings.
5. Do not upload copyrighted source videos unless you have permission to share them. A short synthetic reproduction is preferred.

## Development checks

```bash
go fmt ./...
go vet ./...
go test -race ./...
```

FFmpeg must be available on `PATH` for the generated-media integration test. Without FFmpeg, that test skips automatically.

## Pull requests

Keep changes small enough to review. Explain user-visible behaviour, tests performed, and any ROM-format compatibility impact. Generated executables, FFmpeg binaries, test videos, and `.gba` files should not be committed.
