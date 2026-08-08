import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = resolve(here, "..");
const repository = resolve(website, "..");

test("website uses the desktop application icon", async () => {
  const [desktopPng, webPng, desktopIco, webIco, html, manifest] = await Promise.all([
    readFile(resolve(repository, "assets/app_icon.png")),
    readFile(resolve(website, "public/icon.png")),
    readFile(resolve(repository, "assets/app_icon.ico")),
    readFile(resolve(website, "public/favicon.ico")),
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "public/site.webmanifest"), "utf8"),
  ]);

  assert.deepEqual(webPng, desktopPng, "website PNG must exactly match assets/app_icon.png");
  assert.deepEqual(webIco, desktopIco, "website ICO must exactly match assets/app_icon.ico");
  assert.match(html, /rel="icon" href="\.\/favicon\.ico"/);
  assert.match(html, /rel="apple-touch-icon" href="\.\/apple-touch-icon\.png"/);
  assert.match(html, /rel="manifest" href="\.\/site\.webmanifest"/);
  assert.match(html, /class="site-icon" src="\.\/icon\.png"/);

  const parsed = JSON.parse(manifest);
  assert.equal(parsed.name, "GBA Video Maker Web");
  assert.ok(parsed.icons.some((icon) => icon.sizes === "192x192"));
  assert.ok(parsed.icons.some((icon) => icon.sizes === "512x512"));
});

test("web edition exposes desktop parity controls", async () => {
  const [html, script] = await Promise.all([
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "src/main.js"), "utf8"),
  ]);
  for (const id of [
    "saveProjectButton", "openProjectInput", "previewVideo", "timelineStart", "timelineEnd",
    "inlineTimeline", "timelineTrack", "timelineStartHandle", "timelinePlayHandle", "timelineEndHandle",
    "timelineStartTimeInput", "timelineEndTimeInput",
    "titleEditor", "titlePreviewInput", "audioPreviewButton", "audioQuality",
    "extremeSection", "smartTarget", "smartPriority", "smartAnalyze", "smartCancel", "smartResults",
    "splitVideo", "splitBudget", "maxPartDuration", "chapterAware",
    "partTitleScreens", "resumeLongSplit", "estimateArea", "optimizerButton",
    "titleCardGroup", "titleCardPreview", "titleCardPartSelect", "titleCardUseShared",
    "titleCardTitle", "titleCardSubtitle", "titleCardBackground", "titleCardDarkness", "titleCardTextSize",
    "titleCardTextColor", "titleCardOutlineColor", "titleCardSubtitleTextSize", "titleCardSubtitleAlignment",
    "titleCardSubtitleTextColor", "titleCardSubtitleOutlineColor", "titleCardStartMode", "titleCardFade",
    "menuSettingsGroup", "menuPreview", "menuBackground", "customMenuBackground",
    "menuUIColor", "menuSelectionColor", "menuOutline", "menuOutlineColor",
  ]) assert.match(html, new RegExp(`id="${id}"`));
  assert.match(script, /performLongSplit/);
  assert.match(script, /Estimated output:/);
  assert.match(script, /Part \$\{partNumber\} of approximately/);
  assert.match(script, /indexedDB\.open\("gba-video-maker"/);
  assert.match(script, /decodeCustomFile/);
  assert.match(script, /serializeTheme/);
  assert.match(script, /buildTitleCardAsset/);
  assert.match(script, /renderTitleCardPreview/);
  assert.match(script, /analyzeSmartScan/);
  assert.match(script, /encodeIMAADPCM/);
  assert.match(script, /Input audio track/);
  assert.match(script, /const showSelector = entry\.audioTracksKnown && tracks\.length > 1/);
  assert.match(script, /select\.parentElement\.hidden = !showSelector/);
  assert.match(script, /entry\.audioTracksKnown && tracks\.length <= 1/);
  assert.match(script, /stream_tags=language,title/);
  assert.match(script, /`0:a:\$\{Number\(clipOptions\.audioTrack\) \|\| 0\}`/);
  assert.match(script, /audioTrack: Number\(entry\.audioTrack\) \|\| 0/);
  assert.match(script, /canonicalProjectFromBrowser/);
  assert.match(script, /normalizeBrowserProjectDocument/);
  assert.match(script, /parsePartDuration/);
  assert.match(script, /resolveAudioCodec/);
  assert.match(script, /buildOptimizerProposal/);
  assert.match(script, /applyPreviewFraming/);
  assert.match(script, /drawFittedPreviewFrame/);
  assert.match(script, /desktopOutputFileName/);
  assert.match(html, /Extreme optimization \(Experimental\)/);
  assert.match(html, /Compact ADPCM \(Experimental\)/);
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
