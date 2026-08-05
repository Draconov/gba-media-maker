# GBA Video Maker Web

This folder is the complete GitHub Pages edition of GBA Video Maker. It is deliberately isolated from the Windows desktop code.

## What the web edition currently supports

The web edition now follows the Windows application's conversion workflow and uses the same embedded GBA player:

- One or more input videos with clip ordering and project/per-clip settings
- Single ROM, menu ROM, playlist ROM, and separate-ROM ZIP output
- Automatic long-video fallback plus an optional **Split the video** panel
- 1–32 MiB part targets, `MM:SS` duration caps, chapter-aware cuts, filename / `PART N` title screens, `PARTS.txt`, and numbered ROMs
- Estimated ROM size and part count before conversion
- `Part N of approximately M` and source-position progress during splitting
- Persistent completed-part recovery through IndexedDB; reselect the same source file to resume
- Save/open `.gbavideo` projects with browser-safe source relinking
- Full-width selected-clip player with integrated start/end controls, eight preview frames, a directly editable yellow GBA-font title field, and selected-channel audio preview
- Presets, 32 MiB optimizer, framing, four frame rates, palettes, dithering, compression, PCM audio options, normalization, limiter, speed, volume, looping, seeking, and SRAM resume
- The original fixed 120×80 v0.9 playback format and the same `assets/player_stub.bin` as Windows
- Fully local processing: source videos are not uploaded to a server

## Folder layout

```text
website/
├── index.html
├── package.json
├── vite.config.js
├── public/
│   ├── .nojekyll
│   └── player_stub.bin       copied automatically from ../assets/
├── scripts/
│   └── sync-player.mjs
├── src/
│   ├── main.js               interface and downloads
│   ├── style.css
│   ├── rom.worker.js         palette/compression worker
│   └── rom-core.js           palette, compression, and ROM assembly
└── tests/
    └── rom-core.test.mjs
```

The deployment workflow is stored at:

```text
.github/workflows/pages.yml
```

## Run it locally

Install Node.js 22, then run:

```bash
cd website
npm install
npm run dev
```

Vite prints a local address such as `http://localhost:5173`.

To test and build the exact static files GitHub Pages will receive:

```bash
cd website
npm test
npm run build
npm run preview
```

The deployable site is generated in `website/dist/`. Do not edit `dist` manually; GitHub Actions rebuilds it.

## Publish on GitHub Pages

1. Commit the whole repository, including `website/` and `.github/workflows/pages.yml`.
2. Push it to the repository's `main` branch.
3. Open the repository on GitHub.
4. Open **Settings → Pages**.
5. Under **Build and deployment**, choose **GitHub Actions** as the source.
6. Open the **Actions** tab and wait for **Deploy web converter to GitHub Pages** to finish.
7. The deployment job displays the public site URL. For a project repository it normally looks like:

   ```text
   https://draconov.github.io/gba-video-maker/
   ```

Future pushes that change `website/`, `assets/player_stub.bin`, or the Pages workflow automatically rebuild the site.

## Update the GBA player used by the website

Rebuild the desktop/player project normally so `assets/player_stub.bin` is updated. The website's `prebuild` script copies that file into `website/public/player_stub.bin` and verifies that it is exactly 32 KiB.

## Browser platform limits

The controls and output behavior match the Windows app, but the execution environment is different. ffmpeg.wasm and each active 120×80 frame stream use browser memory, so the splitter processes long sources one ROM part at a time. Very large source files can still exceed a browser's WebAssembly or storage quota; the Windows build remains faster and more tolerant of multi-gigabyte jobs.

Saved browser projects cannot silently reopen local files because browsers block that for privacy. After opening a `.gbavideo` project, select the same source files and the app relinks them by name and size. Completed split parts are stored in IndexedDB when recovery is enabled.

## Current browser controls

Project-wide controls include output behaviour, frame rate, screen framing, dithering, audio, seek step, ROM title, and save/resume behaviour.

Each uploaded clip also has its own:

- menu title
- loop toggle
- start and end trim times
- playback-speed multiplier from 0.25x to 4x
- volume multiplier from 0 to 2