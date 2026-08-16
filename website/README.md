<p align="center">
  <img src="public/icon.png" width="96" alt="GBA Media Maker icon">
</p>

<h1 align="center">GBA Media Maker Web</h1>

<p align="center">
  Build video, music, GIF, image, and mixed-media Game Boy Advance ROMs directly in a modern desktop browser.
  <br>
  Media processing stays on the user's device; there is no conversion-upload backend.
</p>

<p align="center">
  <a href="https://draconov.github.io/gba-media-maker/"><strong>Open the web app</strong></a>
  ·
  <a href="https://github.com/Draconov/gba-media-maker">Main repository</a>
  ·
  <a href="https://github.com/Draconov/gba-media-maker/releases/latest">Desktop releases</a>
  ·
  <a href="README.uk.md"><strong>Українська</strong></a>
</p>

---

## Overview

The browser edition targets **current-release parity** with the portable desktop app. It uses ffmpeg.wasm for media work and the same synchronized 32 KiB GBA player stub for ROM playback.

## Languages

The website UI supports **English** and **Українська**. The language button sits on the same row as the desktop-download action and opens a dropdown menu. Browser language is used on first visit, the selection is stored locally, and English is the fallback. GBA ROM/player UI is not localized by this mechanism.

Canonical locale files live at the repository root:

```text
../locales/index.json
../locales/en.json
../locales/uk.json
```

The manifest drives the available language menu. `scripts/sync-locales.mjs` copies all locale JSON files into generated `public/locales/` before dev, test, and build.

### Current parity

| Feature | Browser | Desktop |
|---|:---:|:---:|
| Video ROMs | ✅ | ✅ |
| Animated GIF import/auto-loop | ✅ | ✅ |
| Audio-only ROMs | ✅ | ✅ |
| Native static-image ROMs | ✅ | ✅ |
| Mixed/same-media menu collections | ✅ | ✅ |
| Separate-ROM ZIP output | ✅ | ✅ |
| `.gbamedia` v2 projects | ✅ | ✅ |
| Legacy `.gbavideo` loading | ✅ | ✅ |
| v0.12.2 menu themes/custom colour picker | ✅ | ✅ |
| Custom image/GIF/video menu backgrounds | ✅ | ✅ |
| Split-video title cards | ✅ | ✅ |
| Long-video splitting | ✅ | ✅ |
| Extreme optimization controls | ✅ | ✅ |
| 20 default audio-artwork presets | ✅ | ✅ |
| Embedded/custom audio artwork | ✅ | ✅ |
| Current save/resume/runtime controls | ✅ | ✅ |
| Current output naming rules | ✅ | ✅ |

The main practical difference is resource access: native FFmpeg is faster and more tolerant of large sources, while browsers have WebAssembly memory/storage limits and cannot silently reopen arbitrary local paths from a saved project.

## Media workflow

### Video

The website keeps the existing 120×80 indexed video pipeline:

- trim start/end;
- playback speed;
- fit, crop, or stretch;
- GBA-friendly frame-rate presets;
- shared/scene palettes;
- off/ordered/error-diffusion dithering;
- delta or uncompressed video;
- selected audio stream/channel;
- volume, normalize, limiter;
- PCM or experimental ADPCM under supported settings;
- seek step, loop, save/resume;
- long-video splitting, chapters, title cards, and recovery tools.

### Animated GIF

GIFs are classified as video, decoded as animation frames, and always loop in the generated ROM.

### Audio

Music/audio items support:

- editable 28-character song title;
- editable 28-character artist subtitle;
- trim and playback speed;
- PCM/experimental ADPCM audio;
- looping and save/resume;
- current Now Playing player controls;
- artwork mode: **Embedded**, **Default**, or **Custom image**.

#### Artwork presets

The browser includes the same 20 native 240×160 default artwork presets as the desktop app:

```text
website/public/audio-artwork/preset-01.png
...
website/public/audio-artwork/preset-20.png
```

Embedded mode uses source cover art when present and otherwise falls back to the selected preset. Custom PNG/JPEG/WebP artwork is cropped/prepared for the 240×160 GBA screen and stored in `.gbamedia` project data.

### Images

Static images are converted to native 240×160 RGB555 and support:

- fit/crop/stretch;
- timed slideshow mode;
- manual viewer mode when slideshow is disabled.

## Output modes

Current v0.13 behavior is deliberately simple:

| Input count | Choice | Result |
|---|---|---|
| 1 item | Single ROM | `<source>.gba` |
| 2+ items | Media menu | `GBA_Media_Collection.gba` |
| 2+ items | Separate ROMs | `GBA_Media_Collection.zip` |

Every 2+ item collection uses the media menu regardless of whether it contains mixed media or only videos/music/images. Legacy playlist project values are migrated to menu mode.

## Output naming parity

The browser uses the same naming helpers as the desktop workflow:

```text
single source             My Movie.gba
multi-item menu           GBA_Media_Collection.gba
multi-item batch ZIP      GBA_Media_Collection.zip
ROM inside batch ZIP      My Song_GBA.gba
split-video archive       My Movie_PARTS.zip
split-video part          My Movie_PART_01.gba
project                   My Project.gbamedia
```

The browser/operating system can still append `(1)` or another duplicate-download suffix after GBA Media Maker has supplied the intended filename.

## Menu design

The website uses the stable v0.12.2 menu/theme implementation carried forward into v0.13:

- Classic dark
- Ocean Wave — static
- Ocean Wave — animated
- Blue Wave — animated
- custom PNG/JPEG/WebP background
- custom GIF/video background sampled to at most 16 looping frames
- stable 120×80 MTH1 data
- exact 3×5 player font/coordinates
- media-type labels
- pixel-art selection arrow
- normal, selection, and outline colours
- optional outline
- full custom GBA colour picker with saturation/value square, hue slider, eyedropper, RGB, HEX, and preset swatches

The preview is generated from the same logical menu data that is embedded into the finished ROM.

## Project files

Browser projects use the same canonical v2 schema as the desktop app:

```text
extension: .gbamedia
format:    gba-media-maker-project
version:   2
```

Saved data includes:

- project defaults and per-item overrides;
- media ordering/titles;
- image slideshow state;
- music title/artist;
- artwork mode/preset/custom artwork;
- menu theme/background settings;
- title-card settings;
- current encoding/splitting settings.

Legacy `.gbavideo` project formats remain loadable.

### Browser source relinking

A website cannot silently reopen arbitrary local files stored in a project path. After reopening `.gbamedia`, the browser may ask the user to select the original source media again so it can relink them by available file information.

## GBA runtime parity

The website does **not** maintain a separate player fork.

```text
../player source
      │
      ▼
../assets/player_stub.bin
      │
      ▼
scripts/sync-player.mjs
      │
      ▼
public/player_stub.bin
```

The synchronized runtime provides the current v0.13 behavior, including:

- v0.12.2-style video HUD and loop icon;
- frame counter;
- 0.10-second temporary seek/mute/volume feedback;
- ~0.30-second held-seek repeat;
- current L/R media navigation;
- audio Now Playing HUD;
- no volume/mute on silent video/GIF/image entries;
- slideshow-aware image controls;
- restored save/resume confirmation prompt;
- current media menu.

## Long-video splitting

Single-video conversion can automatically fall back to numbered parts when the selected cartridge/data target cannot safely fit.

Available controls include:

- 1–32 MiB target;
- 20 MiB, 30 MiB, and Maximum shortcuts;
- optional maximum part duration;
- chapter-aware boundaries;
- native title cards;
- completed-part recovery;
- `PARTS.txt` manifest;
- exact numbered filenames matching desktop output.

## Extreme optimization

Extreme remains an explicitly experimental preset. The browser performs bounded low-resolution analysis through ffmpeg.wasm, ranks candidate settings, and can apply adaptive-keyframe/experimental ADPCM recommendations.

Stable presets remain on the established fixed-keyframe/PCM path unless the user explicitly selects Extreme.

## Run locally

### Requirements

- Node.js 22
- npm
- Modern desktop browser with WebAssembly support
- LLVM only if you also intend to rebuild the GBA player from `../player/`

### Install and develop

```bash
cd website
npm install
npm test
npm run dev
```

Vite prints the local development URL.

### Production build

```bash
cd website
npm install
npm test
npm run build
npm run preview
```

The deployable static site is generated under:

```text
website/dist/
```

Do not hand-edit `dist/`; it is a build product.

## Player synchronization

The npm lifecycle hooks synchronize both the GBA player stub and the app version from the repository root before development, testing, and production build:

```json
"predev": "npm run sync-runtime",
"pretest": "npm run sync-runtime",
"prebuild": "npm run sync-runtime"
```

The runtime sync scripts copy `../assets/player_stub.bin`, copy the 20 shared artwork presets from `../assets/audio-artwork/`, and generate the website app version from `../VERSION`. Change `VERSION` once when preparing a release; generated browser assets are not committed.

## Tests

The browser test suite covers ROM assembly, media descriptors, project migration, naming parity, menu/title-card data, site assets, and current media behavior.

```bash
cd website
npm test
```

The GitHub Pages workflow also:

1. installs LLVM;
2. rebuilds the GBA player;
3. synchronizes the player stub;
4. installs website dependencies;
5. runs browser tests;
6. builds the Vite site;
7. deploys the static artifact.

See [`.github/workflows/pages.yml`](../.github/workflows/pages.yml).

## Project structure

```text
website/
├── index.html
├── package.json
├── vite.config.js
├── public/
│   ├── icon.png
│   ├── audio-artwork/
│   │   ├── preset-01.png
│   │   └── ... preset-20.png
│   └── player_stub.bin        generated/synchronized for build
├── scripts/
│   ├── sync-player.mjs
│   ├── sync-artwork.mjs
│   └── sync-version.mjs
├── src/
│   ├── main.js                UI, ffmpeg.wasm workflow, conversion/downloads
│   ├── rom-core.js            GBV5 assembly, palette/compression/media records
│   ├── rom.worker.js          conversion worker
│   ├── project-format.js      .gbamedia v2 save/load/migration
│   ├── parity-utils.js        desktop-compatible naming/settings helpers
│   ├── menu-themes.js         stable theme/background preview + MTH1 data
│   ├── title-cards.js         title-card state/preview/TCD1 data
│   ├── gba-text.js            player-compatible text encoding
│   ├── adpcm.js               browser IMA ADPCM codec path
│   ├── smart-encoding.js      Extreme optimization analysis
│   └── style.css
└── tests/
    └── *.test.mjs
```

## GitHub Pages

Deployment is configured in:

```text
.github/workflows/pages.yml
```

Enable **Settings → Pages → Build and deployment → GitHub Actions** for the repository. The final public URL depends on the GitHub account/repository where this project is deployed.

## Browser limitations

- ffmpeg.wasm and active frame buffers can use substantial memory.
- Very large sources can exceed WebAssembly memory or browser storage quotas.
- Native desktop FFmpeg is usually much faster.
- Browsers cannot silently restore arbitrary local source paths from project files.
- Some browser capabilities vary by engine/version; test the intended browser before relying on very large jobs.

## Privacy

The static website does not upload source media to a GBA Media Maker conversion server. Selected files are processed locally through the browser conversion stack.

The page may still fetch normal application dependencies/resources required by the deployed site (including the configured FFmpeg WebAssembly core); this is separate from uploading the user's source media.

## More documentation

- [Main README](../README.md)
- [Architecture](../docs/ARCHITECTURE.md)
- [Release history](../CHANGELOG.md)
- [Player runtime](../player/README.md)
- [Third-party notices](../THIRD_PARTY_NOTICES.md)
