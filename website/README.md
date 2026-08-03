# GBA Video Maker Web

This folder is the complete GitHub Pages edition of GBA Video Maker. It is deliberately isolated from the Windows desktop code.

## What the web edition currently supports

- One or more input videos
- The original fixed 120×80 v0.9 playback format
- Single-video ROMs, multi-video menus, and playlists
- Fit, crop, or stretch framing
- 14.93, 11.95, 9.95, or 7.47 FPS
- PCM mono audio at 16,384 Hz
- Shared 250-colour palette per clip
- Ordered, error-diffusion, or disabled dithering
- Delta-frame compression and seek tables
- The same embedded `assets/player_stub.bin`, including the custom menu background
- Fully local processing: source videos are not uploaded to a server

The desktop release remains recommended for long videos because native FFmpeg is much faster and browsers have stricter memory limits.

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
   https://YOUR-USERNAME.github.io/YOUR-REPOSITORY/
   ```

Future pushes that change `website/`, `assets/player_stub.bin`, or the Pages workflow automatically rebuild the site.

## Update the GBA player used by the website

Rebuild the desktop/player project normally so `assets/player_stub.bin` is updated. The website's `prebuild` script copies that file into `website/public/player_stub.bin` and verifies that it is exactly 32 KiB.

## Important browser limitation

ffmpeg.wasm and the raw 120×80 frame stream both live in browser memory during conversion. The worker rejects a clip when its estimated raw frame stream exceeds 384 MiB. This is intentional: huge jobs should use the desktop version instead of crashing the browser tab.

## Current browser controls

Project-wide controls include output behaviour, frame rate, screen framing, dithering, audio, seek step, ROM title, and save/resume behaviour.

Each uploaded clip also has its own:

- menu title
- loop toggle
- start and end trim times
- playback-speed multiplier from 0.25x to 4x
- volume multiplier from 0 to 2

The page follows the operating system light/dark preference and uses the same colour variables as the desktop interface.
