import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = resolve(here, "..");
const repository = resolve(website, "..");

test("website ships GBA Media Maker branding and transparent web icons", async () => {
  const [desktopPng, webPng, html, manifest] = await Promise.all([
    readFile(resolve(repository, "assets/icon.png")),
    readFile(resolve(website, "public/icon.png")),
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "public/site.webmanifest"), "utf8"),
  ]);

  assert.deepEqual([...desktopPng.subarray(0, 8)], [137,80,78,71,13,10,26,10]);
  assert.deepEqual([...webPng.subarray(0, 8)], [137,80,78,71,13,10,26,10]);
  assert.match(html, /<title>GBA Media Maker/);
  assert.match(html, /rel="icon" href="\.\/favicon\.ico"/);
  assert.match(html, /rel="apple-touch-icon" href="\.\/apple-touch-icon\.png"/);
  assert.match(html, /rel="manifest" href="\.\/site\.webmanifest"/);
  assert.match(html, /class="site-icon" src="\.\/icon\.png"/);

  const parsed = JSON.parse(manifest);
  assert.equal(parsed.name, "GBA Media Maker Web");
  assert.ok(parsed.icons.some((icon) => icon.sizes === "192x192"));
  assert.ok(parsed.icons.some((icon) => icon.sizes === "512x512"));
});

test("web edition exposes current desktop media-maker parity controls", async () => {
  const [html, script, worker, romCore, projectFormat] = await Promise.all([
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "src/main.js"), "utf8"),
    readFile(resolve(website, "src/rom.worker.js"), "utf8"),
    readFile(resolve(website, "src/rom-core.js"), "utf8"),
    readFile(resolve(website, "src/project-format.js"), "utf8"),
  ]);
  for (const id of [
    "fileInput", "saveProjectButton", "openProjectInput", "previewVideo", "previewImage", "timelineStart", "timelineEnd",
    "inlineTimeline", "timelineTrack", "timelineStartHandle", "timelinePlayHandle", "timelineEndHandle",
    "timelineStartTimeInput", "timelineEndTimeInput",
    "titleEditor", "titlePreviewInput", "audioPreviewButton", "audioQuality",
    "extremeSection", "smartTarget", "smartPriority", "smartAnalyze", "smartCancel", "smartResults",
    "splitVideo", "splitBudget", "maxPartDuration", "chapterAware",
    "partTitleScreens", "resumeLongSplit", "estimateArea", "optimizerButton",
    "defaultImageSlideshow", "defaultImageSeconds",
    "titleCardGroup", "titleCardPreview", "titleCardPartSelect", "titleCardUseShared",
    "titleCardTitle", "titleCardSubtitle", "titleCardBackground", "titleCardDarkness", "titleCardTextSize",
    "titleCardTextColor", "titleCardOutlineColor", "titleCardSubtitleTextSize", "titleCardSubtitleAlignment",
    "titleCardSubtitleTextColor", "titleCardSubtitleOutlineColor", "titleCardStartMode", "titleCardFade",
    "menuSettingsGroup", "menuPreview", "menuBackground", "customMenuBackground", "customMenuVideoTiming", "customMenuVideoStart", "customMenuVideoDuration",
    "menuUIColor", "menuSelectionColor", "menuOutline", "menuOutlineColor",
  ]) assert.match(html, new RegExp(`id="${id}"`));

  assert.match(html, /accept="\.gbamedia,\.gbavideo,[^"]*"/);
  assert.match(html, /video\/\*,audio\/\*,image\/\*/);
  assert.match(html, /One ROM — media menu/);
  assert.match(html, /Separate ROMs in ZIP/);
  assert.match(html, /Enable slideshow/);
  assert.match(html, /Custom image, GIF or video/);
  assert.match(html, /Extreme optimization \(Experimental\)/);
  assert.match(html, /Compact ADPCM \(Experimental\)/);

  assert.match(script, /function guessMediaKind/);
  assert.match(script, /musicArtworkMode/);
  assert.match(script, /musicArtworkPreset/);
  assert.match(script, /musicArtworkCustom/);
  assert.match(script, /Embedded artwork/);
  assert.match(script, /Built-in artwork/);
  assert.match(script, /automaticArtworkPreset/);
  assert.match(script, /Custom image/);
  assert.match(script, /resolveAudioArtworkRGB/);
  assert.match(script, /encodeNativeMedia/);
  assert.match(script, /mediaKind: "image"/);
  assert.match(script, /mediaKind: "audio"/);
  assert.match(script, /\.gif\$\/i/);
  assert.match(script, /performLongSplit/);
  assert.match(script, /Estimated output:/);
  assert.match(script, /indexedDB\.open\("gba-video-maker"/);
  assert.match(script, /decodeCustomFile/);
  assert.match(script, /decodeRGB24Frames/);
  assert.match(script, /serializeTheme/);
  assert.match(script, /buildTitleCardAsset/);
  assert.match(script, /renderTitleCardPreview/);
  assert.match(script, /analyzeSmartScan/);
  assert.match(script, /Input audio track/);
  assert.match(script, /stream_tags=language,title/);
  assert.match(script, /canonicalProjectFromBrowser/);
  assert.match(script, /normalizeBrowserProjectDocument/);
  assert.match(script, /desktopOutputFileName/);
  assert.match(worker, /encodeNativeMedia/);
  assert.match(romCore, /convertNativeMediaClip/);
  assert.match(romCore, /MMD2/);
  assert.match(projectFormat, /gba-media-maker-project/);
  assert.match(projectFormat, /PROJECT_VERSION = 2/);
});

test("website includes all twenty built-in audio artwork presets", async () => {
  const expectedPngMagic = [137,80,78,71,13,10,26,10];
  for (let index = 1; index <= 20; index += 1) {
    const name = `preset-${String(index).padStart(2, "0")}.png`;
    const image = await readFile(resolve(website, "public/audio-artwork", name));
    assert.deepEqual([...image.subarray(0, 8)], expectedPngMagic, name);
  }
  const script = await readFile(resolve(website, "src/main.js"), "utf8");
  assert.match(script, /Array\.from\(\{ length: 20 \}/);
});

test("title-card navigation stays in one row and avoids redundant reloads", async () => {
  const [style, script] = await Promise.all([
    readFile(resolve(website, "src/style.css"), "utf8"),
    readFile(resolve(website, "src/main.js"), "utf8"),
  ]);
  assert.match(style, /\.title-card-checkbox-row\{[^}]*flex-wrap:nowrap;/);
  assert.match(style, /\.title-card-nav\{[^}]*flex-wrap:nowrap;/);
  assert.match(style, /\.title-card-nav select\{[^}]*width:132px;/);
  assert.match(script, /function updateTitleCardNavState\(\)/);
  assert.match(script, /sourceChanged/);
  assert.match(script, /setTitleCardPart\(titleCardPart, true\)/);
  const html = await readFile(resolve(website, "index.html"), "utf8");
  assert.match(html, /Show title card at start/);
  assert.match(html, /Use same settings for each part/);
  assert.match(html, /id="titleCardPartLabel"[^>]*>of 2</);
  assert.match(html, /class="title-card-type-header"/);
  assert.match(html, /id="titleCardSubtitleTextSize"/);
  assert.match(style, /\.title-card-type-row\{[^}]*grid-template-columns:/);
  assert.match(style, /\.title-card-panel\{container-type:inline-size\}/);
  assert.match(style, /\.title-card-layout\{[^}]*grid-template-columns:clamp\(300px,38%,400px\) minmax\(0,1fr\)/);
  assert.match(style, /\.title-card-type-row\{[^}]*grid-template-columns:50px minmax\(176px,1fr\) 66px 72px 44px 44px/);
  assert.match(style, /\.typography-text input\{width:100%;min-width:0\}/);
  assert.match(style, /@container \(max-width:800px\)\{[\s\S]*?\.title-card-layout\{grid-template-columns:1fr\}/);
});


test("selected clip editor uses a full-width player and an editable GBA title field", async () => {
  const [html, style, script] = await Promise.all([
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "src/style.css"), "utf8"),
    readFile(resolve(website, "src/main.js"), "utf8"),
  ]);
  assert.match(html, /class="video-editor-stage"/);
  assert.match(html, /class="video-stage"/);
  assert.match(html, /id="titlePreviewInput"[^>]*maxlength="12"/);
  assert.match(html, /id="inlineTimeline"/);
  assert.match(html, /id="timelineTrack"/);
  assert.match(html, /id="timelineStartHandle"/);
  assert.match(html, /id="timelinePlayHandle"/);
  assert.match(html, /id="timelineEndHandle"/);
  assert.match(style, /\.preview-layout\s*\{\s*display:\s*block;/);
  assert.match(style, /#previewVideo[\s\S]*width:\s*100%;/);
  assert.match(style, /#previewVideo[\s\S]*aspect-ratio:\s*3 \/ 2;/);
  assert.match(style, /#previewVideo[\s\S]*object-fit:\s*contain;/);
  assert.match(html, /id="previewVideo"(?![^>]*\scontrols(?:\s|=|>))/);
  assert.match(html, /id="timelineStartTimeInput"/);
  assert.match(html, /id="timelineEndTimeInput"/);
  assert.match(style, /\.inline-timeline[\s\S]*position:\s*relative;/);
  assert.match(style, /\.timeline-handle-start[\s\S]*--timeline-start/);
  assert.match(script, /const raw = elements\.titlePreviewInput\.value;[\s\S]*sanitizeMenuTitle\(raw\)/);
  assert.match(script, /document\.activeElement === elements\.titlePreviewInput/);
  assert.match(style, /#titlePreviewInput\s*\{[^}]*position:\s*absolute;[^}]*color:\s*transparent;[^}]*background:\s*transparent;/s);
  assert.match(style, /\.title-editor\s*\{[^}]*background:\s*#000;/s);
  assert.match(script, /text === "" \? \(kind === "start" \? 0 : duration\)/);
  assert.match(script, /beginTimelineDrag\("start"/);
  assert.match(script, /timelineValueFromClientX/);
  assert.match(script, /parseClock\(text\)/);
  assert.match(script, /togglePreviewPlayback/);
  assert.match(style, /\.timeline-handle-current[\s\S]*background:\s*#fff;/);
  assert.doesNotMatch(style, /\.timeline-handle-current[\s\S]*border-radius:\s*50%/);
});

test("every queried website element exists in the HTML", async () => {
  const [html, script] = await Promise.all([
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "src/main.js"), "utf8"),
  ]);
  const ids = new Set([...html.matchAll(/\bid="([^"]+)"/g)].map((match) => match[1]));
  const queried = [...script.matchAll(/document\.querySelector\("#([^"]+)"\)/g)].map((match) => match[1]);
  assert.ok(queried.length > 30);
  for (const id of queried) assert.ok(ids.has(id), `missing #${id} in website/index.html`);
});

test("menu colour picker embeds the full custom picker and sizes its panel to content", async () => {
  const [desktopScript, webScript, desktopStyle, webStyle] = await Promise.all([
    readFile(resolve(repository, "web/menu-themes.js"), "utf8"),
    readFile(resolve(website, "src/menu-themes.js"), "utf8"),
    readFile(resolve(repository, "web/style.css"), "utf8"),
    readFile(resolve(website, "src/style.css"), "utf8"),
  ]);

  for (const script of [desktopScript, webScript]) {
    assert.match(script, /className='gba-sv-area'/);
    assert.match(script, /pickerRow\.append\(eyedropperButton,currentSwatch,hueSlider\)/);
    assert.match(script, /channelLabel\.append\(caption,numberInput\)/);
    assert.match(script, /trigger\.style\.setProperty\('--gba-swatch-color',colour\.hex\)/);
    assert.match(script, /M18\.5 2\.5/);
    assert.doesNotMatch(script, /Open full colour picker/);
    assert.doesNotMatch(script, /Common GBA colours/);
  }

  for (const style of [desktopStyle, webStyle]) {
    assert.match(style, /\.gba-color-popover\{[^}]*inline-size:fit-content;/s);
    assert.match(style, /\.gba-picker-content\{[^}]*max-width:100%;/s);
    assert.match(style, /\.gba-sv-area\{[^}]*linear-gradient\(to top,#000,transparent\)/s);
    assert.match(style, /\.gba-color-control\{[^}]*flex-wrap:wrap;/s);
    assert.match(style, /\.gba-color-trigger\{[^}]*border-radius:(?:8|9)px;[^}]*--gba-swatch-color/s);
    assert.match(style, /\.gba-color-trigger-swatch\{[^}]*position:absolute;[^}]*border-radius:(?:5|6)px/s);
    assert.match(style, /\.gba-current-swatch\{[^}]*border-radius:8px/s);
    assert.match(style, /\.gba-eyedropper svg\{[^}]*fill:none;[^}]*stroke:currentColor/s);
  }
});
