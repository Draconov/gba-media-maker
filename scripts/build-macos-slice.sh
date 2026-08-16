#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCH="${1:?usage: build-macos-slice.sh <amd64|arm64> <output-dir>}"
OUT="${2:?usage: build-macos-slice.sh <amd64|arm64> <output-dir>}"
FFMPEG_DIR="${FFMPEG_DIR:-}"

case "$ARCH" in
  amd64|arm64) ;;
  *) echo "Unsupported macOS architecture: $ARCH" >&2; exit 1 ;;
esac

rm -rf "$OUT"
mkdir -p "$OUT/Frameworks"
GOOS=darwin GOARCH="$ARCH" CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o "$OUT/GBA Media Maker" "$ROOT"
chmod +x "$OUT/GBA Media Maker"

if [[ -n "$FFMPEG_DIR" ]]; then
  [[ -x "$FFMPEG_DIR/ffmpeg" ]] || { echo "FFMPEG_DIR has no executable ffmpeg: $FFMPEG_DIR" >&2; exit 1; }
  if command -v lipo >/dev/null 2>&1; then
    expected="$ARCH"
    [[ "$ARCH" != "amd64" ]] || expected="x86_64"
    arches="$(lipo -archs "$FFMPEG_DIR/ffmpeg")"
    grep -qw "$expected" <<<"$arches" || { echo "FFmpeg slice architecture mismatch: expected $expected, got $arches" >&2; exit 1; }
  fi
  cp "$FFMPEG_DIR/ffmpeg" "$OUT/ffmpeg"
  chmod +x "$OUT/ffmpeg"
  if [[ -d "$FFMPEG_DIR/Frameworks" ]]; then
    cp -a "$FFMPEG_DIR/Frameworks/." "$OUT/Frameworks/"
  fi
fi
printf '%s\n' "$ARCH" > "$OUT/arch.txt"
