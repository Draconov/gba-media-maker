#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCH="${1:?usage: build-linux-slice.sh <amd64|arm64> <output-dir>}"
OUT="${2:?usage: build-linux-slice.sh <amd64|arm64> <output-dir>}"
FFMPEG_BIN="${FFMPEG_BIN:-}"

case "$ARCH" in
  amd64|arm64) ;;
  *) echo "Unsupported Linux architecture: $ARCH" >&2; exit 1 ;;
esac

rm -rf "$OUT"
mkdir -p "$OUT"
GOOS=linux GOARCH="$ARCH" CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o "$OUT/gba-media-maker" "$ROOT"
chmod +x "$OUT/gba-media-maker"

if [[ -n "$FFMPEG_BIN" ]]; then
  [[ -x "$FFMPEG_BIN" ]] || { echo "FFMPEG_BIN is not executable: $FFMPEG_BIN" >&2; exit 1; }
  if command -v file >/dev/null 2>&1; then
    info="$(file -b "$FFMPEG_BIN")"
    case "$ARCH" in
      amd64) grep -Eqi 'x86[-_ ]64|x86-64' <<<"$info" || { echo "FFmpeg is not x64: $info" >&2; exit 1; } ;;
      arm64) grep -Eqi 'aarch64|ARM64' <<<"$info" || { echo "FFmpeg is not ARM64: $info" >&2; exit 1; } ;;
    esac
  fi
  cp "$FFMPEG_BIN" "$OUT/ffmpeg"
  chmod +x "$OUT/ffmpeg"
fi
printf '%s\n' "$ARCH" > "$OUT/arch.txt"
