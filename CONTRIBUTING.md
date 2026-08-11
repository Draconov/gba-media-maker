# Contributing

Bug reports, reproducible test cases, documentation fixes, and focused pull requests are welcome.

## Before opening an issue

1. Reproduce the problem with the latest release/source state.
2. State whether the problem affects the **Windows app**, **browser edition**, **generated GBA ROM**, or more than one layer.
3. For conversion problems, include the source media kind, container/codec, duration, audio-stream layout, and the relevant selected settings.
4. For generated-ROM problems, include the output mode and media mix (video/GIF/audio/image), plus the emulator/flash cartridge/hardware used for testing.
5. Attach `GBA Media Maker.log` when the desktop app exits unexpectedly.
6. Prefer a short synthetic or freely shareable reproduction. Do not upload copyrighted media unless you have permission to share it.

## Get the source

```bash
git clone https://github.com/Draconov/gba-media-maker.git
cd gba-media-maker
```

## Development requirements

Desktop/core work uses:

- Go 1.23+
- FFmpeg on `PATH` for conversion/integration tests
- LLVM (`clang`, `ld.lld`, `llvm-objcopy`) when rebuilding the GBA runtime

Website work additionally uses:

- Node.js 22
- npm

## Core checks

Run before submitting a change:

```bash
go fmt ./...
go vet ./...
go test -race ./...
```

FFmpeg-backed integration tests skip when FFmpeg is not available, but changes to conversion behavior should be tested with FFmpeg installed.

## GBA player changes

If anything under `player/` changes:

```bash
./player/build.sh
```

The authoritative output is:

```text
assets/player_stub.bin
```

It must remain exactly **32,768 bytes**. Do not hand-maintain a separate browser player; the website synchronizes this stub through `website/scripts/sync-player.mjs`.

When changing controls/HUD/runtime behavior, verify the relevant media paths:

- video with audio;
- silent video / No audio;
- animated GIF;
- audio-only media;
- manual image;
- slideshow image;
- multi-item media menu;
- save/resume when relevant.

## Website changes

```bash
cd website
npm install
npm test
npm run build
```

Keep browser behavior aligned with the desktop model when the feature is intended to have parity. In particular, preserve:

- project schema/migration;
- output naming;
- GBV5 descriptor/metadata layout;
- menu/theme serialization;
- title-card serialization;
- audio-artwork modes/presets;
- synchronized player runtime.

## Documentation changes

Permanent release/user information belongs in:

- `README.md`
- `CHANGELOG.md`
- `docs/ARCHITECTURE.md`
- `player/README.md`
- `website/README.md`
- `THIRD_PARTY_NOTICES.md` when third-party components/assets change

Do not add one-off root-level `*_UPDATE.txt` patch notes for normal repository history. Consolidate notable user-visible changes under the appropriate version/date in `CHANGELOG.md`.

## Pull requests

Keep changes small enough to review. Explain:

- user-visible behavior;
- affected media/front-end/runtime paths;
- tests performed;
- ROM/project-format compatibility impact;
- whether `assets/player_stub.bin` was intentionally rebuilt.

Generated executables, downloaded FFmpeg binaries, ad-hoc test videos, generated `.gba` files, and build directories should not be committed.

Third-party artwork/assets should be added only when the project has the rights needed to distribute them, and their separate licensing status should be documented.
