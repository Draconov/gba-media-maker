#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PLAYER="$ROOT/player"
BUILD="$PLAYER/build"
CLANG=${CLANG:-clang}
LLD=${LLD:-ld.lld}
OBJCOPY=${OBJCOPY:-llvm-objcopy}

rm -rf "$BUILD"
mkdir -p "$BUILD"

COMMON="--target=arm-none-eabi -mcpu=arm7tdmi -marm -ffreestanding -fno-builtin -fno-stack-protector -Os -fno-inline-functions -Wall -Wextra"
$CLANG $COMMON -c "$PLAYER/startup.S" -o "$BUILD/startup.o"
$CLANG $COMMON -c "$PLAYER/metadata.S" -o "$BUILD/metadata.o"
$CLANG $COMMON -c "$PLAYER/main.c" -o "$BUILD/main.o"
$LLD -T "$PLAYER/linker.ld" --gc-sections -o "$BUILD/player.elf" "$BUILD/startup.o" "$BUILD/main.o" "$BUILD/metadata.o"
$OBJCOPY -O binary "$BUILD/player.elf" "$BUILD/player.bin"
cp "$BUILD/player.bin" "$ROOT/assets/player_stub.bin"
truncate -s 32768 "$ROOT/assets/player_stub.bin"
printf 'Built %s\n' "$ROOT/assets/player_stub.bin"
