# GBA player source

This directory contains the freestanding ARM7TDMI playback engine embedded in every generated ROM.

ROM controls:

- `A`: pause or resume
- `L`: seek backward by approximately five seconds
- `R`: seek forward by approximately five seconds
- `B` or `START`: restart

The build scripts require LLVM tools (`clang`, `ld.lld`, and `llvm-objcopy`). They compile without devkitARM or libgba.

```bash
./player/build.sh
```

```powershell
./player/build.ps1
```

The resulting `assets/player_stub.bin` must be exactly 8192 bytes. The converter fills the GBA header and metadata when assembling each ROM.
