#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PLAYER="$ROOT/player"
BUILD="$PLAYER/build"
CLANG=${CLANG:-clang}
LLD=${LLD:-ld.lld}
OBJCOPY=${OBJCOPY:-llvm-objcopy}
OUTPUT=${PLAYER_STUB_OUT:-"$ROOT/assets/player_stub.bin"}

rm -rf "$BUILD"
mkdir -p "$BUILD"
mkdir -p "$(dirname -- "$OUTPUT")"

COMMON="--target=arm-none-eabi -mcpu=arm7tdmi -marm -ffreestanding -fno-builtin -fno-stack-protector -Oz -fno-inline-functions -Wall -Wextra"
$CLANG $COMMON -c "$PLAYER/startup.S" -o "$BUILD/startup.o"
$CLANG $COMMON -c "$PLAYER/metadata.S" -o "$BUILD/metadata.o"
$CLANG $COMMON -c "$PLAYER/compiler_compat.S" -o "$BUILD/compiler_compat.o"
$CLANG $COMMON -c "$PLAYER/main.c" -o "$BUILD/main.o"
$LLD -T "$PLAYER/linker.ld" --gc-sections -o "$BUILD/player.elf" "$BUILD/startup.o" "$BUILD/main.o" "$BUILD/compiler_compat.o" "$BUILD/metadata.o"
$OBJCOPY -O binary "$BUILD/player.elf" "$BUILD/player.bin"
cp "$BUILD/player.bin" "$OUTPUT"
truncate -s 32768 "$OUTPUT"
printf 'Built %s\n' "$OUTPUT"
