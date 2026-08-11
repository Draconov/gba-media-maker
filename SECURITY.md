# Security policy

## Supported version

Security fixes target the latest published release.

## Reporting a vulnerability

Please avoid filing a public issue for vulnerabilities involving:

- arbitrary command execution;
- path traversal or unsafe file extraction;
- local API/session-token bypass;
- unintended non-loopback network exposure;
- downloaded-binary or checksum verification failures;
- malicious project/media files that escape intended local processing boundaries.

Use GitHub's private vulnerability-reporting feature when it is enabled for the repository. Include reproduction steps, affected version, operating system/browser, and impact. Do not include private media or personal filesystem paths unless they are necessary to reproduce the issue.

## Desktop security boundaries

The portable desktop app is designed around a local-only conversion boundary:

- HTTP listener restricted to `127.0.0.1`;
- random per-session token in local API paths;
- upload/request size limits;
- temporary per-session workspace;
- FFmpeg launched with argument arrays rather than shell command interpolation;
- sanitized output names;
- official portable packaging uses a pinned BtbN FFmpeg release;
- the release workflow discovers the correct non-shared win64 LGPL archive from that release's `checksums.sha256`, verifies SHA-256, and verifies software AV1 decoder support;
- portable packages do not rely on a runtime executable downloader.

Changes that weaken these properties require explicit review.

## Browser security/privacy boundary

The browser edition is a static application. It has no GBA Media Maker media-upload conversion backend. Source files are processed locally through browser APIs and ffmpeg.wasm.

Normal site/application resources can still be fetched from the deployed host/CDN configuration. This is separate from uploading the user's source media.

Project files must not be treated as permission to silently access arbitrary local paths; browsers intentionally require the user to reselect/relink source files when needed.

## Generated ROMs

Generated ROMs directly access GBA hardware resources such as VRAM, DMA, timers, and SRAM. Test save/resume behavior on the intended emulator/flash cartridge/hardware, especially when changing SRAM layout or runtime initialization.
