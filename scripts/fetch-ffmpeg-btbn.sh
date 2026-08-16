#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/ffmpeg-pins.env"
OS="${1:?usage: fetch-ffmpeg-btbn.sh <windows|linux> <amd64|arm64> <output>}"
ARCH="${2:?usage: fetch-ffmpeg-btbn.sh <windows|linux> <amd64|arm64> <output>}"
OUTPUT="${3:?usage: fetch-ffmpeg-btbn.sh <windows|linux> <amd64|arm64> <output>}"

case "$OS/$ARCH" in
  windows/amd64) suffix='win64-lgpl\.zip'; binary='ffmpeg.exe' ;;
  linux/amd64) suffix='linux64-lgpl\.tar\.xz'; binary='ffmpeg' ;;
  linux/arm64) suffix='linuxarm64-lgpl\.tar\.xz'; binary='ffmpeg' ;;
  *) echo "Unsupported BtbN FFmpeg target: $OS/$ARCH" >&2; exit 1 ;;
esac

base="https://github.com/BtbN/FFmpeg-Builds/releases/download/${BTBN_FFMPEG_TAG}"
work="$(mktemp -d "${TMPDIR:-/tmp}/gbamm-btbn.XXXXXX")"
trap 'rm -rf "$work"' EXIT
curl --fail --location --retry 3 --retry-delay 2 "$base/checksums.sha256" --output "$work/checksums.sha256"
entry="$(grep -E "^[0-9a-fA-F]{64}[[:space:]]+\\*?ffmpeg-[^[:space:]]+-${suffix}$" "$work/checksums.sha256" | head -n 1 || true)"
if [[ -z "$entry" ]]; then
  echo "Could not find pinned BtbN asset for $OS/$ARCH ($suffix)." >&2
  exit 1
fi
expected="$(awk '{print $1}' <<<"$entry")"
archive_name="$(sed -E 's/^[0-9a-fA-F]{64}[[:space:]]+\*?//' <<<"$entry")"
archive="$work/$archive_name"
curl --fail --location --retry 3 --retry-delay 2 "$base/$archive_name" --output "$archive"
printf '%s  %s\n' "$expected" "$archive" | sha256sum -c -
mkdir -p "$work/extract"
case "$archive_name" in
  *.zip) unzip -q "$archive" -d "$work/extract" ;;
  *.tar.xz) tar -xJf "$archive" -C "$work/extract" ;;
  *) echo "Unsupported archive type: $archive_name" >&2; exit 1 ;;
esac
candidate="$(find "$work/extract" -type f -path "*/bin/$binary" -print -quit)"
if [[ -z "$candidate" ]]; then
  echo "No $binary found in $archive_name." >&2
  exit 1
fi
mkdir -p "$(dirname "$OUTPUT")"
cp "$candidate" "$OUTPUT"
chmod +x "$OUTPUT" || true
if [[ ! -s "$OUTPUT" ]]; then
  echo "Extracted FFmpeg is empty." >&2
  exit 1
fi
if [[ "$OS" == "linux" ]]; then
  decoders="$($OUTPUT -hide_banner -decoders 2>&1)"
  if ! grep -Eq 'libdav1d|libaom-av1' <<<"$decoders"; then
    echo "Pinned FFmpeg build has no verified software AV1 decoder." >&2
    exit 1
  fi
else
  if [[ "$(head -c 2 "$OUTPUT")" != "MZ" ]]; then
    echo "Pinned Windows FFmpeg is not a PE executable." >&2
    exit 1
  fi
  if ! grep -aEq -- '--enable-libdav1d|--enable-libaom' "$OUTPUT"; then
    echo "Pinned Windows FFmpeg has no embedded libdav1d/libaom build marker." >&2
    exit 1
  fi
fi
echo "$OUTPUT"
