#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "package-macos.sh must run on macOS because it creates Universal Mach-O binaries." >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"
AMD64_SLICE="${1:?usage: package-macos.sh <amd64-slice-dir> <arm64-slice-dir>}"
ARM64_SLICE="${2:?usage: package-macos.sh <amd64-slice-dir> <arm64-slice-dir>}"
DIST="$ROOT/dist"
APP="$DIST/GBA Media Maker.app"
ZIP="$DIST/GBA_Media_Maker_v${VERSION}_macOS.zip"
BIN="$APP/Contents/MacOS/GBA Media Maker"

require_file() {
  [[ -f "$1" ]] || { echo "Missing required release file: $1" >&2; exit 1; }
}
verify_universal() {
  local file_path="$1" arches
  arches="$(lipo -archs "$file_path")"
  grep -qw x86_64 <<<"$arches" || { echo "Universal binary is missing x86_64: $file_path ($arches)" >&2; exit 1; }
  grep -qw arm64 <<<"$arches" || { echo "Universal binary is missing arm64: $file_path ($arches)" >&2; exit 1; }
}
merge_macho() {
  local intel="$1" arm="$2" out="$3"
  require_file "$intel"
  require_file "$arm"
  mkdir -p "$(dirname "$out")"
  lipo -create "$intel" "$arm" -output "$out"
  chmod +x "$out"
  verify_universal "$out"
}

for slice in "$AMD64_SLICE" "$ARM64_SLICE"; do
  require_file "$slice/GBA Media Maker"
  require_file "$slice/ffmpeg"
done

rm -rf "$APP" "$ZIP" "$ZIP.sha256"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$APP/Contents/Frameworks"
merge_macho "$AMD64_SLICE/GBA Media Maker" "$ARM64_SLICE/GBA Media Maker" "$BIN"
merge_macho "$AMD64_SLICE/ffmpeg" "$ARM64_SLICE/ffmpeg" "$APP/Contents/MacOS/ffmpeg"

framework_list="$(mktemp "${TMPDIR:-/tmp}/gbamm-frameworks.XXXXXX")"
trap 'rm -f "$framework_list"' EXIT
{
  find "$AMD64_SLICE/Frameworks" -maxdepth 1 -type f -name '*.dylib' -exec basename {} \;
  find "$ARM64_SLICE/Frameworks" -maxdepth 1 -type f -name '*.dylib' -exec basename {} \;
} | sort -u > "$framework_list"
while IFS= read -r base; do
  [[ -n "$base" ]] || continue
  intel="$AMD64_SLICE/Frameworks/$base"
  arm="$ARM64_SLICE/Frameworks/$base"
  out="$APP/Contents/Frameworks/$base"
  if [[ -f "$intel" && -f "$arm" ]]; then
    merge_macho "$intel" "$arm" "$out"
  elif [[ -f "$intel" ]]; then
    cp "$intel" "$out"
    chmod +x "$out"
  else
    cp "$arm" "$out"
    chmod +x "$out"
  fi
done < "$framework_list"

# Architecture slices are laid out as <slice>/ffmpeg + <slice>/Frameworks, so
# their FFmpeg load commands use @executable_path/Frameworks. In the final
# .app, FFmpeg moves to Contents/MacOS while frameworks live in Contents/Frameworks.
# Rewrite only at the final packaging stage so both slice smoke tests and the
# Universal app resolve the same bundled libraries correctly.
while IFS= read -r base; do
  [[ -n "$base" ]] || continue
  install_name_tool -change \
    "@executable_path/Frameworks/$base" \
    "@executable_path/../Frameworks/$base" \
    "$APP/Contents/MacOS/ffmpeg" 2>/dev/null || true
done < "$framework_list"

cp "$ROOT/assets/icon.icns" "$APP/Contents/Resources/AppIcon.icns"
cp "$ROOT/README.md" "$ROOT/README.uk.md" "$ROOT/LICENSE" "$ROOT/THIRD_PARTY_NOTICES.md" "$APP/Contents/Resources/"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleDisplayName</key><string>GBA Media Maker</string>
  <key>CFBundleExecutable</key><string>GBA Media Maker</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundleIdentifier</key><string>com.draconov.gbamediamaker</string>
  <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  <key>CFBundleName</key><string>GBA Media Maker</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>LSApplicationCategoryType</key><string>public.app-category.video</string>
  <key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
plutil -lint "$APP/Contents/Info.plist" >/dev/null

if otool -L "$APP/Contents/MacOS/ffmpeg" | grep -Eq '/opt/homebrew|/usr/local/(Cellar|opt)|@executable_path/Frameworks/'; then
  echo "Universal FFmpeg still has an invalid external/slice dependency path:" >&2
  otool -L "$APP/Contents/MacOS/ffmpeg" >&2
  exit 1
fi

if [[ -n "${MACOS_CODESIGN_IDENTITY:-}" ]]; then
  find "$APP/Contents/Frameworks" -type f -name '*.dylib' -exec codesign --force --options runtime --timestamp --sign "$MACOS_CODESIGN_IDENTITY" {} \;
  codesign --force --options runtime --timestamp --sign "$MACOS_CODESIGN_IDENTITY" "$APP/Contents/MacOS/ffmpeg"
  codesign --force --deep --options runtime --timestamp --sign "$MACOS_CODESIGN_IDENTITY" "$APP"
else
  codesign --force --deep --sign - "$APP"
fi
codesign --verify --deep --strict "$APP"
verify_universal "$BIN"
verify_universal "$APP/Contents/MacOS/ffmpeg"

mkdir -p "$DIST"
if command -v ditto >/dev/null 2>&1; then
  (cd "$DIST" && ditto -c -k --sequesterRsrc --keepParent "$(basename "$APP")" "$(basename "$ZIP")")
else
  (cd "$DIST" && zip -9 -qr "$(basename "$ZIP")" "$(basename "$APP")")
fi
shasum -a 256 "$ZIP" | awk '{print $1}' > "$ZIP.sha256"
echo "$ZIP"
