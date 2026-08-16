#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(tr -d '[:space:]' < "$ROOT/VERSION")"
AMD64_SLICE="${1:?usage: package-linux.sh <amd64-slice-dir>}"
DIST="$ROOT/dist"
FOLDER="GBA_Media_Maker_v${VERSION}_Linux_x86_64"
STAGE="$DIST/$FOLDER"
ARCHIVE="$DIST/${FOLDER}.tar.gz"

require_file() {
  [[ -f "$1" ]] || { echo "Missing required release file: $1" >&2; exit 1; }
}

verify_x64() {
  local file_path="$1" description="$2" info
  command -v file >/dev/null || return 0
  info="$(file -b "$file_path")"
  grep -Eqi 'x86[-_ ]64|x86-64' <<<"$info" || { echo "$description is not x86_64: $info" >&2; exit 1; }
}

require_file "$AMD64_SLICE/gba-media-maker"
require_file "$AMD64_SLICE/ffmpeg"
verify_x64 "$AMD64_SLICE/gba-media-maker" "Linux app"
verify_x64 "$AMD64_SLICE/ffmpeg" "Linux FFmpeg"

rm -rf "$STAGE" "$ARCHIVE" "$ARCHIVE.sha256"
mkdir -p "$STAGE"
cp "$AMD64_SLICE/gba-media-maker" "$STAGE/gba-media-maker"
cp "$AMD64_SLICE/ffmpeg" "$STAGE/ffmpeg"
chmod +x "$STAGE/gba-media-maker" "$STAGE/ffmpeg"
cp "$ROOT/assets/icon.png" "$STAGE/gba-media-maker.png"
cp "$ROOT/README.md" "$ROOT/README.uk.md" "$ROOT/LICENSE" "$ROOT/THIRD_PARTY_NOTICES.md" "$STAGE/"

cat > "$STAGE/install-user.sh" <<'INSTALL'
#!/usr/bin/env bash
set -euo pipefail
case "$(uname -m)" in
  x86_64|amd64) ;;
  *)
    echo "This GBA Media Maker release requires 64-bit x86 Linux (x86_64). Detected: $(uname -m)" >&2
    exit 1
    ;;
esac
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${HOME}/.local/bin"
DATA_DIR="${XDG_DATA_HOME:-${HOME}/.local/share}"
APP_DIR="$DATA_DIR/gba-media-maker"
mkdir -p "$BIN_DIR" "$APP_DIR" "$DATA_DIR/applications" "$DATA_DIR/icons/hicolor/256x256/apps"
cp "$HERE/gba-media-maker" "$APP_DIR/gba-media-maker"
cp "$HERE/ffmpeg" "$APP_DIR/ffmpeg"
chmod +x "$APP_DIR/gba-media-maker" "$APP_DIR/ffmpeg"
ln -sfn "$APP_DIR/gba-media-maker" "$BIN_DIR/gba-media-maker"
cp "$HERE/gba-media-maker.png" "$DATA_DIR/icons/hicolor/256x256/apps/gba-media-maker.png"
cat > "$DATA_DIR/applications/gba-media-maker.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=GBA Media Maker
Comment=Convert video, audio and images into Game Boy Advance ROMs
Exec=$APP_DIR/gba-media-maker
Icon=gba-media-maker
Terminal=false
Categories=AudioVideo;Utility;
StartupNotify=true
DESKTOP
echo "Installed GBA Media Maker for the current user."
echo "App files are in $APP_DIR; ~/.local/bin/gba-media-maker is a convenience symlink."
INSTALL
chmod +x "$STAGE/install-user.sh"

cat > "$STAGE/README-Linux.txt" <<'TXT'
GBA Media Maker — Linux x86_64

This release is for modern 64-bit Intel/AMD Linux PCs (x86_64).
Run ./gba-media-maker to start the desktop UI.
Run ./install-user.sh to install it for the current user and add a desktop launcher.
The installer keeps the bundled FFmpeg inside the app-specific user data folder; it does not replace your system FFmpeg.
The desktop UI uses Chrome/Chromium/Edge/Brave app mode when available and falls back to your default browser.
Native file dialogs use Zenity when available, then KDialog.
TXT

mkdir -p "$DIST"
tar -C "$DIST" -czf "$ARCHIVE" "$FOLDER"
sha256sum "$ARCHIVE" | awk '{print $1}' > "$ARCHIVE.sha256"
echo "$ARCHIVE"
