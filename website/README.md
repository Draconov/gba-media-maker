<p align="center">
  <img src="public/icon.png" width="96" alt="GBA Video Maker icon">
</p>

<h1 align="center">GBA Video Maker Web</h1>

<p align="center">
  Convert videos into Game Boy Advance ROMs directly in your browser.
  <br>
  Processing stays on your device; source videos are never uploaded to a server.
</p>

<p align="center">
  <a href="https://draconov.github.io/gba-video-maker/"><strong>Open the web app</strong></a>
  ·
  <a href="https://github.com/draconov/gba-video-maker">Main repository</a>
</p>

---

## Overview

Version 0.10 follows the Windows application's conversion workflow and uses the same embedded GBA player. It supports single videos, collections, customizable menu ROMs, long-video splitting, project files, per-clip settings, and browser-side ROM generation through ffmpeg.wasm.

### Highlights

- Runs locally in the browser with no video uploads
- Produces Single ROM, playlist ROM, menu ROM, or separate-ROM ZIP output
- Uses the same fixed 120×80 playback format and `player_stub.bin` as the Windows app
- Supports `.gbavideo` project save/open with browser-safe source relinking
- Estimates ROM size and long-video part count before conversion
- Recovers completed split parts through IndexedDB
- Provides the same menu-design preview, built-in backgrounds, UI colours, outlines, and custom image/GIF support as the Windows app

## Output modes

| Mode | Result | Best for |
| --- | --- | --- |
| **Single ROM** | One `.gba` file; automatically splits if it cannot fit | One video |
| **Playlist ROM** | One ROM that plays clips in order | Episodes or compilations |
| **Menu ROM** | One ROM with a startup clip selector | Collections with manual selection |
| **Separate ROMs** | Numbered `.gba` files inside a ZIP | Independent clips or batch conversion |

## Main features

### Video and project workflow

- One or more source videos
- Drag-and-drop clip ordering
- Project defaults with per-clip overrides
- Full-width selected-clip preview
- Windows-style thumbnail/trim timeline below the video
- Draggable Start, Current, and End handles
- Directly editable Start and End timestamps
- Editable yellow GBA-font title field with automatic character filtering and 12-character truncation
- Selected-channel audio preview
- `.gbavideo` save/open support

### Encoding controls

- Named quality/frame-rate presets
- Crop, fit-with-bars, and stretch framing
- Shared or scene-change palettes
- Off, ordered, and error-diffusion dithering
- Raw or keyframe/delta compression
- Mono mix, left-channel, right-channel, or disabled audio
- Loudness normalization and limiter
- Per-clip speed, volume, looping, palette, dithering, trim, and title settings
- Reviewable 32 MiB optimization proposal

### Menu design

The **Menu design** section appears when multiple clips are loaded and **Menu ROM** is selected. Its preview is generated from the same 120×80 indexed background and UI settings that are embedded in the finished ROM.

- **Classic dark**, **Ocean Wave — static**, **Ocean Wave — animated**, and **Blue Wave — animated** presets
- Ocean Wave dual-rate palette shimmer: approximately 2 changes per second on the bright curl and 5 changes per second on the lower water
- Seven UI-colour presets
- Optional one-pixel outline with five outline colours
- Custom PNG, JPEG, WebP, or GIF upload
- Center-crop and resize to 120×80
- RGB555 indexed conversion with reserved menu UI colours
- Up to 16 sampled looping GIF frames at approximately 5 changes per second
- Integer-scaled preview using the player's exact 3×5 font, coordinates, divider lines, and selector shape
- Theme data embedded in the ROM as an `MTH1` record, so each exported menu ROM is self-contained
- Theme palette, frames, animation timing, colours, and outline settings included in size estimates and project files

Animated frame themes are prepared on the hidden Mode 4 page and displayed on VBlank to reduce tearing. Browser GIF animation requires support for the browser `ImageDecoder` API; browsers without it import the file as a static image when possible.

### Long-video splitting

Single-ROM conversion automatically falls back to numbered ROM parts when the source cannot safely fit on one cartridge. Enabling **Split the video** exposes additional controls:

- 1–32 MiB target size
- 20 MiB, 30 MiB, and Maximum shortcuts
- Optional maximum duration using seconds, `MM:SS`, or `H:MM:SS`
- Chapter-aware cut points
- Optional filename and `PART N` title screens
- Numbered ROM output plus `PARTS.txt`
- Estimated part count before conversion
- Progress such as `Part 3 of approximately 7` and `Source position: 18:42 / 50:00`
- Interrupted-job recovery through IndexedDB
- Partial output recovery when a later part fails

## Run locally

### Requirements

- Node.js 22
- npm
- A modern browser with WebAssembly support

### Development server

```bash
cd website
npm install
npm run dev
```

Vite prints a local address, usually `http://localhost:5173`.

### Test and build

```bash
cd website
npm test
npm run build
npm run preview
```

The deployable site is generated in `website/dist/`. Do not edit `dist/` manually; it is rebuilt by Vite and GitHub Actions.

## Project structure

```text
website/
├── index.html
├── package.json
├── vite.config.js
├── public/
│   ├── .nojekyll
│   ├── icon.png
│   └── player_stub.bin       copied from ../assets/
├── scripts/
│   └── sync-player.mjs
├── src/
│   ├── main.js               interface, conversion workflow, and downloads
│   ├── menu-themes.js        built-in themes, custom image/GIF conversion, and preview
│   ├── style.css             responsive light/dark interface
│   ├── rom.worker.js         palette and compression worker
│   └── rom-core.js           palette, compression, theme embedding, and ROM assembly
└── tests/
    └── rom-core.test.mjs
```

The GitHub Pages workflow is stored at:

```text
.github/workflows/pages.yml
```

## Publish with GitHub Pages

1. Commit the repository, including `website/` and `.github/workflows/pages.yml`.
2. Push to the `main` branch.
3. Open **Settings → Pages** in the GitHub repository.
4. Set **Build and deployment → Source** to **GitHub Actions**.
5. Open the **Actions** tab and wait for **Deploy web converter to GitHub Pages** to finish.

The public project URL is:

```text
https://draconov.github.io/gba-video-maker/
```

Pushes that change `website/`, `assets/player_stub.bin`, or the Pages workflow automatically rebuild the site.

## Keep the browser player synchronized

Rebuild the desktop/player project so `assets/player_stub.bin` is current. The website's `prebuild` script copies it into `website/public/player_stub.bin` and verifies that the file is exactly 32 KiB.

```bash
cd website
npm run build
```

Do not manually maintain a separate browser player binary.

## Browser limitations

The interface and ROM output follow the Windows app, but browsers have stricter memory, file-access, and storage limits.

- ffmpeg.wasm and active frame buffers consume browser memory.
- Long videos are processed one ROM part at a time.
- Very large files can exceed the browser's WebAssembly memory or storage quota.
- Browsers cannot silently reopen local files. After opening a `.gbavideo` project, reselect the source files so they can be relinked by name and size.
- Completed split parts are stored in IndexedDB only when recovery is enabled.
- The Windows build remains faster and more tolerant of multi-gigabyte sources.

## Privacy

All conversion work happens locally in the browser. The application does not upload source videos to a conversion server.