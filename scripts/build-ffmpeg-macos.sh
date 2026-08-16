#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "build-ffmpeg-macos.sh must run on macOS." >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/ffmpeg-pins.env"
OUT="${1:-$ROOT/dist/ffmpeg-macos}"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/gbamm-ffmpeg.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

command -v brew >/dev/null || { echo "Homebrew is required to build the pinned macOS FFmpeg." >&2; exit 1; }
brew install dav1d pkg-config nasm >/dev/null

archive="$WORK/ffmpeg-${MACOS_FFMPEG_VERSION}.tar.xz"
curl --fail --location --retry 3 --retry-delay 2 \
  "https://ffmpeg.org/releases/ffmpeg-${MACOS_FFMPEG_VERSION}.tar.xz" \
  --output "$archive"
tar -xf "$archive" -C "$WORK"

src="$WORK/ffmpeg-${MACOS_FFMPEG_VERSION}"
dav1d_prefix="$(brew --prefix dav1d)"
cd "$src"
PKG_CONFIG_PATH="$dav1d_prefix/lib/pkgconfig" ./configure \
  --disable-doc \
  --disable-debug \
  --disable-ffplay \
  --disable-ffprobe \
  --disable-autodetect \
  --enable-libdav1d \
  --enable-zlib \
  --enable-static \
  --disable-shared \
  --extra-cflags="-I$dav1d_prefix/include" \
  --extra-ldflags="-L$dav1d_prefix/lib"
make -j"$(sysctl -n hw.logicalcpu)" ffmpeg

rm -rf "$OUT"
mkdir -p "$OUT/Frameworks"
cp ffmpeg "$OUT/ffmpeg"
chmod +x "$OUT/ffmpeg"

buildconf="$($OUT/ffmpeg -hide_banner -buildconf 2>&1)"
if grep -Eq -- '--enable-(gpl|version3|nonfree)' <<<"$buildconf"; then
  echo "Refusing to package a GPL/version3/nonfree FFmpeg build." >&2
  exit 1
fi
if ! "$OUT/ffmpeg" -hide_banner -decoders 2>&1 | grep -q 'libdav1d'; then
  echo "Pinned macOS FFmpeg build has no libdav1d AV1 decoder." >&2
  exit 1
fi

while IFS= read -r dep; do
  [[ -n "$dep" ]] || continue
  case "$dep" in
    /System/*|/usr/lib/*|@*) continue ;;
  esac
  base="$(basename "$dep")"
  cp -f "$dep" "$OUT/Frameworks/$base"
  install_name_tool -change "$dep" "@executable_path/../Frameworks/$base" "$OUT/ffmpeg"
done < <(otool -L "$OUT/ffmpeg" | tail -n +2 | awk '{print $1}')

if otool -L "$OUT/ffmpeg" | grep -Eq '/opt/homebrew|/usr/local/Cellar'; then
  echo "FFmpeg still references Homebrew after bundling dependencies:" >&2
  otool -L "$OUT/ffmpeg" >&2
  exit 1
fi

for dylib in "$OUT"/Frameworks/*.dylib; do
  [[ -e "$dylib" ]] || continue
  install_name_tool -id "@executable_path/../Frameworks/$(basename "$dylib")" "$dylib" || true
done

native_arch="$(uname -m)"
case "$native_arch" in
  x86_64) expected_arch="x86_64" ;;
  arm64) expected_arch="arm64" ;;
  *) echo "Unsupported macOS build architecture: $native_arch" >&2; exit 1 ;;
esac
actual_arches="$(lipo -archs "$OUT/ffmpeg")"
grep -qw "$expected_arch" <<<"$actual_arches" || { echo "FFmpeg architecture mismatch: expected $expected_arch, got $actual_arches" >&2; exit 1; }
printf '%s
' "$expected_arch" > "$OUT/arch.txt"

"$OUT/ffmpeg" -hide_banner -version | head -n 1
echo "Built portable macOS FFmpeg ($expected_arch) in $OUT"
