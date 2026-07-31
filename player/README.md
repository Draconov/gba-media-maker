# GBA player source

This directory contains the freestanding ARM7TDMI playback engine embedded in every generated ROM.

ROM controls:

- `A`: pause or resume
- `L` or `D-pad Left`: seek backward by approximately five seconds
- `R` or `D-pad Right`: seek forward by approximately five seconds
- `SELECT`: mute or unmute audio
- `D-pad Up`: show or hide the progress/time HUD
- `B` or `START`: restart

The build scripts require LLVM tools (`clang`, `ld.lld`, and `llvm-objcopy`). They compile without devkitARM or libgba.

```bash
./player/build.sh
```

```powershell
./player/build.ps1
```

The resulting `assets/player_stub.bin` must be exactly 8192 bytes. The converter fills the GBA header and metadata when assembling each ROM.


Mute/unmute feedback is shown as a compact top-right badge: a red `X` when muted and a green `V` when unmuted. Temporary seek and mute feedback now lasts about 0.4 seconds, while the playback HUD remains visible during pause.
