# Embedded GBA Media Player

Freestanding ARM7TDMI runtime used by **GBA Media Maker v0.13.0**.

The player is built once as a generic **32 KiB** stub and embedded at the beginning of every generated ROM. GBV5 metadata and media assets are appended by the desktop or browser converter.

## Supported media paths

| Media | Display path | Stored visual data | Audio |
|---|---|---|---|
| Video | Mode 4 | 120×80 indexed frames, expanded 2× to 240×160 | Optional PCM / experimental ADPCM |
| Animated GIF | Mode 4 | Video-frame path, forced loop | None |
| Music/audio | Mode 3 | Native 240×160 RGB555 artwork | PCM / experimental ADPCM |
| Static image | Mode 3 | Native 240×160 RGB555 image | None |

The runtime remains compatible with legacy GBV5 video descriptors and uses v0.13 descriptor flags to dispatch audio-only, image, and media-metadata entries.

## Build

Linux/macOS:

```bash
./player/build.sh
```

PowerShell:

```powershell
./player/build.ps1
```

Requirements:

- `clang`
- `ld.lld`
- `llvm-objcopy`

The build compiles `startup.S`, `metadata.S`, `main.c`, and `compiler_compat.S` for `arm-none-eabi`, links the freestanding runtime, converts it to a raw binary, and pads the result to exactly:

```text
32768 bytes / 0x8000
```

By default the output is `assets/player_stub.bin`. CI can set `PLAYER_STUB_OUT` to smoke-build a temporary copy without overwriting the committed stub.

## Current controls

### Video

| Button | Action |
|---|---|
| `A` | Pause / resume |
| `B` | Return to menu in menu ROMs; restart in a direct ROM |
| `L` / `R` | Previous / next media |
| `Left` / `Right` | Seek while playing; one-frame step while paused |
| Hold `Left` / `Right` | Repeat about every 18 VBlanks (~0.30 s) |
| `Up` / `Down` | Volume 0 / 50 / 100 only when audio exists |
| `SELECT` | Mute / unmute only when audio exists |
| `START` | HUD hidden → time only → full |
| `START + SELECT` | Help |

GIFs, silent videos, and videos converted with **No audio** do not expose volume/mute actions or badges.

### Audio

| Button | Action |
|---|---|
| `A` | Pause / resume |
| `B` | Return to menu in menu ROMs; restart in a direct ROM |
| `L` / `R` | Previous / next media |
| `Left` / `Right` | Seek while playing; one timeline step while paused |
| Hold `Left` / `Right` | Repeat about every 18 VBlanks (~0.30 s) |
| `Up` / `Down` | Volume 0 / 50 / 100 |
| `SELECT` | Mute / unmute |
| `START` | HUD hidden → time only → full |
| `START + SELECT` | Help |

Audio starts with the full Now Playing HUD visible.

### Images

| Button | Action |
|---|---|
| `A` | Pause/resume only when slideshow duration is non-zero |
| `B` | Return to menu in menu ROMs; restart in a direct ROM |
| `L` / `R` | Previous / next media |
| `START` | Cycle HUD visibility |
| `START + SELECT` | Help |

Manual images ignore `A`. Images have no seek, volume, or mute controls.

### Media menu

| Button | Action |
|---|---|
| `Up` / `Down` | Move within current column |
| `Left` / `Right` | Move between columns/pages |
| `A` | Play selected media |
| `START + SELECT` | Help |

The old `SELECT + L/R` media shortcut and `L + R` quick-HUD shortcut are intentionally removed.

## HUD behavior

Video uses the restored v0.12.2 presentation:

- elapsed/total time;
- full-HUD frame counter;
- progress line;
- yellow loop indicator;
- centered seek badge;
- volume and mute badges when audio exists.

Audio uses a native 240×160 artwork screen with title, artist, elapsed/total time, PLAY/PAUSE status, and a 4-pixel progress line.

Temporary UI timing:

```text
seek badge / temporary full HUD   6 VBlanks ≈ 0.10 s
mute badge                        6 VBlanks ≈ 0.10 s
volume badge                      6 VBlanks ≈ 0.10 s
held seek/step repeat            18 VBlanks ≈ 0.30 s
```

Audio temporary UI updates restore/redraw only affected HUD regions, avoiding full-screen artwork flashes.

## Save/resume

When the GBV5 global resume flag is set, the runtime stores:

- save magic and media count;
- remembered menu selection;
- one encoded resume position per media entry.

Before a valid saved video/audio position is used, the player shows:

```text
CONTINUE FROM
    MM:SS

 A CONTINUE
 B RESTART
```

Images do not show a playback-position prompt.

## GBV5 extensions used by v0.13

Current clip flag bits:

```text
0x0001  audio stream present
0x0002  loop
0x0004  compressed video
0x0008  scene palette table
0x0010  IMA ADPCM
0x0020  adaptive keyframes
0x0040  audio-only media
0x0080  static-image media
0x0100  media metadata record
```

Audio metadata supports current `MMD2` and legacy `MMD1`. New `MMD2` records contain 28-character title, 28-character artist, and 20-character album fields.

Menu themes use `MTH1`. Native title cards use `TCD1`.

## ROM/display invariants

- GBV5 global metadata: `0x7FC0`
- Media/descriptor region begins: `0x8000`
- Player stub size: exactly 32 KiB
- Video logical frame: 120×80 / 9,600 bytes
- Native artwork/image screen: 240×160 RGB555 / 76,800 bytes
- Audio sample rate: 16,384 Hz mono
- Maximum generated ROM: 32 MiB

See [`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md) for the complete ROM and converter architecture.
