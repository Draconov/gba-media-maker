import { FFmpeg } from "@ffmpeg/ffmpeg";
import { fetchFile, toBlobURL } from "@ffmpeg/util";
import { AUDIO_RATE, GBA_REFRESH, RGB_FRAME_BYTES, ROM_LIMIT } from "./rom-core.js";
import { buildStoredZip } from "./zip-store.js";
import { chooseChapterEnd, formatClock, parseClock } from "./split-utils.js";
import { createBuiltinTheme, decodeCustomFile, serializeTheme, deserializeTheme, startPreview, applyUI, settingsColours, rgb555ToHex, quantizeHexColor, describeColor, setupGBAColorPicker } from "./menu-themes.js";
import { buildTitleCardAsset, createTitleCardProject, defaultTitleCardSettings, normalizeTitleCardSettings, renderTitleCardPreview, resolveTitleCardSettings, sanitizeTitleCardText, TITLE_CARD_BYTES } from "./title-cards.js";
import { analyzeSmartScan } from "./smart-encoding.js";
import { glyphBits, glyphLength, sanitizeGBAText, unsupportedGBARunes } from "./gba-text.js";
import { encodeIMAADPCM, decodeIMAADPCM } from "./adpcm.js";
import "./style.css";

const FFMPEG_CORE_BASE = "https://cdn.jsdelivr.net/npm/@ffmpeg/core@0.12.10/dist/esm";
const MAX_RAW_FRAME_BYTES = 384 * 1024 * 1024;

const elements = {
  fileInput: document.querySelector("#fileInput"),
  dropZone: document.querySelector("#dropZone"),
  fileArea: document.querySelector("#fileArea"),
  fileList: document.querySelector("#fileList"),
  fileSummary: document.querySelector("#fileSummary"),
  clearButton: document.querySelector("#clearButton"),
  preset: document.querySelector("#preset"),
  extremeSection: document.querySelector("#extremeSection"),
  smartTarget: document.querySelector("#smartTarget"),
  smartPriority: document.querySelector("#smartPriority"),
  smartAnalyze: document.querySelector("#smartAnalyze"),
  smartCancel: document.querySelector("#smartCancel"),
  smartStatus: document.querySelector("#smartStatus"),
  smartResults: document.querySelector("#smartResults"),
  outputMode: document.querySelector("#outputMode"),
  menuSettingsGroup: document.querySelector("#menuSettingsGroup"),
  menuPreview: document.querySelector("#menuPreview"),
  menuBackground: document.querySelector("#menuBackground"),
  customMenuBackgroundRow: document.querySelector("#customMenuBackgroundRow"),
  customMenuBackground: document.querySelector("#customMenuBackground"),
  clearCustomMenuBackground: document.querySelector("#clearCustomMenuBackground"),
  menuUIColor: document.querySelector("#menuUIColor"),
  menuUIColorValue: document.querySelector("#menuUIColorValue"),
  menuSelectionColor: document.querySelector("#menuSelectionColor"),
  menuSelectionColorValue: document.querySelector("#menuSelectionColorValue"),
  menuOutline: document.querySelector("#menuOutline"),
  menuOutlineColor: document.querySelector("#menuOutlineColor"),
  menuOutlineColorValue: document.querySelector("#menuOutlineColorValue"),
  menuBackgroundStatus: document.querySelector("#menuBackgroundStatus"),
  vblanks: document.querySelector("#vblanks"),
  fitMode: document.querySelector("#fitMode"),
  paletteMode: document.querySelector("#paletteMode"),
  ditherMode: document.querySelector("#ditherMode"),
  compression: document.querySelector("#compression"),
  audioMode: document.querySelector("#audioMode"),
  audioQuality: document.querySelector("#audioQuality"),
  seekSeconds: document.querySelector("#seekSeconds"),
  defaultStart: document.querySelector("#defaultStart"),
  defaultEnd: document.querySelector("#defaultEnd"),
  defaultSpeed: document.querySelector("#defaultSpeed"),
  defaultVolume: document.querySelector("#defaultVolume"),
  defaultLoop: document.querySelector("#defaultLoop"),
  romTitle: document.querySelector("#romTitle"),
  normalize: document.querySelector("#normalize"),
  limiter: document.querySelector("#limiter"),
  resume: document.querySelector("#resume"),
  convertButton: document.querySelector("#convertButton"),
  cancelButton: document.querySelector("#cancelButton"),
  progressArea: document.querySelector("#progressArea"),
  progressBar: document.querySelector("#progressBar"),
  progressMessage: document.querySelector("#progressMessage"),
  progressPercent: document.querySelector("#progressPercent"),
  logOutput: document.querySelector("#logOutput"),
  resultArea: document.querySelector("#resultArea"),
  resultTitle: document.querySelector("#resultTitle"),
  resultDetails: document.querySelector("#resultDetails"),
  downloadButton: document.querySelector("#downloadButton"),
  compatibilityText: document.querySelector("#compatibilityText"),
  desktopLink: document.querySelector("#desktopLink"),
  saveProjectButton: document.querySelector("#saveProjectButton"),
  openProjectInput: document.querySelector("#openProjectInput"),
  projectNotice: document.querySelector("#projectNotice"),
  previewCard: document.querySelector("#previewCard"),
  selectedClipName: document.querySelector("#selectedClipName"),
  previewVideo: document.querySelector("#previewVideo"),
  titleEditor: document.querySelector("#titleEditor"),
  titlePreview: document.querySelector("#titlePreview"),
  titlePreviewInput: document.querySelector("#titlePreviewInput"),
  titleWarning: document.querySelector("#titleWarning"),
  resetClipTitleButton: document.querySelector("#resetClipTitleButton"),
  timelineThumbs: document.querySelector("#timelineThumbs"),
  timelinePlay: document.querySelector("#timelinePlay"),
  timelineStart: document.querySelector("#timelineStart"),
  timelineEnd: document.querySelector("#timelineEnd"),
  timelineCurrentText: document.querySelector("#timelineCurrentText"),
  timelineStartTimeInput: document.querySelector("#timelineStartTimeInput"),
  timelineEndTimeInput: document.querySelector("#timelineEndTimeInput"),
  inlineTimeline: document.querySelector("#inlineTimeline"),
  timelineTrack: document.querySelector("#timelineTrack"),
  timelineSelection: document.querySelector("#timelineSelection"),
  timelineCurrentMarker: document.querySelector("#timelineCurrentMarker"),
  timelineStartHandle: document.querySelector("#timelineStartHandle"),
  timelinePlayHandle: document.querySelector("#timelinePlayHandle"),
  timelineEndHandle: document.querySelector("#timelineEndHandle"),
  jumpBegin: document.querySelector("#jumpBegin"),
  jumpEnd: document.querySelector("#jumpEnd"),
  audioPreviewButton: document.querySelector("#audioPreviewButton"),
  audioPreviewPlayer: document.querySelector("#audioPreviewPlayer"),
  splitVideoRow: document.querySelector("#splitVideoRow"),
  splitVideo: document.querySelector("#splitVideo"),
  splitOptions: document.querySelector("#splitOptions"),
  splitBudget: document.querySelector("#splitBudget"),
  splitBudgetValue: document.querySelector("#splitBudgetValue"),
  maxPartDuration: document.querySelector("#maxPartDuration"),
  chapterAware: document.querySelector("#chapterAware"),
  partTitleScreens: document.querySelector("#partTitleScreens"),
  titleCardGroup: document.querySelector("#titleCardGroup"),
  titleCardControls: document.querySelector("#titleCardControls"),
  titleCardPrev: document.querySelector("#titleCardPrev"),
  titleCardNext: document.querySelector("#titleCardNext"),
  titleCardPartLabel: document.querySelector("#titleCardPartLabel"),
  titleCardPartSelect: document.querySelector("#titleCardPartSelect"),
  titleCardUseShared: document.querySelector("#titleCardUseShared"),
  titleCardCopyToAll: document.querySelector("#titleCardCopyToAll"),
  titleCardPreview: document.querySelector("#titleCardPreview"),
  titleCardBackground: document.querySelector("#titleCardBackground"),
  titleCardDarkness: document.querySelector("#titleCardDarkness"),
  titleCardDarknessValue: document.querySelector("#titleCardDarknessValue"),
  titleCardFrameOffsetField: document.querySelector("#titleCardFrameOffsetField"),
  titleCardFrameOffset: document.querySelector("#titleCardFrameOffset"),
  titleCardSolidColorField: document.querySelector("#titleCardSolidColorField"),
  titleCardSolidColor: document.querySelector("#titleCardSolidColor"),
  titleCardSolidColorValue: document.querySelector("#titleCardSolidColorValue"),
  titleCardTitle: document.querySelector("#titleCardTitle"),
  titleCardSubtitle: document.querySelector("#titleCardSubtitle"),
  titleCardAlignment: document.querySelector("#titleCardAlignment"),
  titleCardTextSize: document.querySelector("#titleCardTextSize"),
  titleCardTextColor: document.querySelector("#titleCardTextColor"),
  titleCardTextColorValue: document.querySelector("#titleCardTextColorValue"),
  titleCardSubtitleAlignment: document.querySelector("#titleCardSubtitleAlignment"),
  titleCardSubtitleTextSize: document.querySelector("#titleCardSubtitleTextSize"),
  titleCardSubtitleTextColor: document.querySelector("#titleCardSubtitleTextColor"),
  titleCardSubtitleTextColorValue: document.querySelector("#titleCardSubtitleTextColorValue"),
  titleCardOutline: document.querySelector("#titleCardOutline"),
  titleCardOutlineColor: document.querySelector("#titleCardOutlineColor"),
  titleCardOutlineColorValue: document.querySelector("#titleCardOutlineColorValue"),
  titleCardSubtitleOutlineColor: document.querySelector("#titleCardSubtitleOutlineColor"),
  titleCardSubtitleOutlineColorValue: document.querySelector("#titleCardSubtitleOutlineColorValue"),
  titleCardTextWarning: document.querySelector("#titleCardTextWarning"),
  titleCardStartMode: document.querySelector("#titleCardStartMode"),
  titleCardDurationField: document.querySelector("#titleCardDurationField"),
  titleCardDuration: document.querySelector("#titleCardDuration"),
  titleCardAllowSkip: document.querySelector("#titleCardAllowSkip"),
  titleCardFade: document.querySelector("#titleCardFade"),
  resumeLongSplit: document.querySelector("#resumeLongSplit"),
  estimateArea: document.querySelector("#estimateArea"),
  optimizerButton: document.querySelector("#optimizerButton"),
};

const PRESETS = {
  best: {
    vblanks: "4", compression: "delta", normalize: true, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "scene", ditherMode: "error", audioQuality: "pcm",
  },
  balanced: {
    vblanks: "5", compression: "delta", normalize: false, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "shared", ditherMode: "ordered", audioQuality: "pcm",
  },
  long: {
    vblanks: "8", compression: "delta", normalize: false, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "shared", ditherMode: "ordered", audioQuality: "pcm",
  },
  small: {
    vblanks: "8", compression: "delta", normalize: false, limiter: false,
    fitMode: "fit", audioMode: "none", paletteMode: "shared", ditherMode: "off", audioQuality: "pcm",
  },
  extreme: {
    vblanks: "5", compression: "delta", normalize: false, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "scene", ditherMode: "ordered", audioQuality: "auto",
  },
};

let entries = [];
let conversionRunning = false;
let conversionCancelled = false;
let ffmpeg = null;
let romWorker = null;
let romTaskCounter = 0;
const romTasks = new Map();
let resultURL = "";
let resultFileName = "";
let resultMime = "application/octet-stream";
let logLines = [];
let recentFFmpegLogs = [];
let ffmpegMetadataQueue = Promise.resolve();
let selectedEntryId = "";
let previewURL = "";
let audioPreviewURL = "";
let pendingProject = null;
let lastEstimate = null;
let thumbRenderToken = 0;
let lastPartialSplit = null;
let preferredOutputMode = "rom";
let customMenuTheme = null;
let activeMenuTheme = null;
let stopMenuPreview = null;
let titleCardProject = null;
let smartAnalysis = null;
let smartAnalysisRunning = false;
let smartAnalysisCancelled = false;
let menuTitleUnsupported = [];
let menuTitleUnsupportedEntryId = "";
let titleCardProjectSource = "";
let titleCardPart = 1;
let titleCardEstimatedParts = 1;
let titleCardPreviewVideo = null;
let titleCardPreviewURL = "";
let titleCardPreviewToken = 0;
let titleCardVisibilitySignature = "";

function menuStyleSettings() {
  return { uiColor: elements.menuUIColor?.value || "#FFFFFF", selectedColor: elements.menuSelectionColor?.value || "#FFDE00", outline: Boolean(elements.menuOutline?.checked), outlineColor: elements.menuOutlineColor?.value || "#000000" };
}
function updateMenuColorReadouts() {
  for (const [input, output, fallback] of [
    [elements.menuUIColor,elements.menuUIColorValue,"#FFFFFF"],
    [elements.menuSelectionColor,elements.menuSelectionColorValue,"#FFDE00"],
    [elements.menuOutlineColor,elements.menuOutlineColorValue,"#000000"],
  ]) {
    if (!input || !output) continue;
    const color=describeColor(input.value,fallback);
    output.textContent=`${color.hex} · RGB555 ${color.r},${color.g},${color.b}`;
    input._gbaColorPickerController?.sync();
  }
}
function snapMenuColor(input,fallback) {
  if (!input) return;
  input.value=quantizeHexColor(input.value,fallback);
  updateMenuColorReadouts();
}
function restoreMenuColors(settings={}) {
  const colors=settingsColours({
    uiColor:settings.menuUIColor ?? settings.menuTheme?.uiColor ?? "white",
    selectedColor:settings.menuSelectionColor ?? settings.menuTheme?.selectedColor,
    outlineColor:settings.menuOutlineColor ?? settings.menuTheme?.outlineColor ?? "black",
  });
  elements.menuUIColor.value=rgb555ToHex(colors.ui);
  elements.menuSelectionColor.value=rgb555ToHex(colors.selected);
  elements.menuOutlineColor.value=rgb555ToHex(colors.outline);
  updateMenuColorReadouts();
}
function rebuildMenuTheme() {
  if (!elements.menuBackground) return;
  const id = elements.menuBackground.value;
  if (id === "custom" && customMenuTheme) activeMenuTheme = applyUI(customMenuTheme, menuStyleSettings());
  else if (id === "custom") activeMenuTheme = createBuiltinTheme("classic-dark", menuStyleSettings());
  else activeMenuTheme = createBuiltinTheme(id, menuStyleSettings());
  elements.customMenuBackgroundRow.hidden = id !== "custom";
  elements.menuOutlineColor.disabled = !elements.menuOutline.checked || conversionRunning;
  elements.menuOutlineColor._gbaColorPickerController?.sync();
}
function serializedMenuTheme() { rebuildMenuTheme(); return activeMenuTheme ? serializeTheme(activeMenuTheme) : null; }
function menuThemeBytes() { return activeMenuTheme ? 64 + activeMenuTheme.palette.length + activeMenuTheme.frames.reduce((sum, frame) => sum + frame.length, 0) : 0; }
async function loadCustomMenuBackground(file) {
  if (!file) return;
  elements.menuBackgroundStatus.textContent = `Optimizing ${file.name}…`;
  try {
    customMenuTheme = await decodeCustomFile(file, menuStyleSettings(), fraction => { elements.menuBackgroundStatus.textContent = `Optimizing ${file.name}… ${Math.round(fraction * 100)}%`; });
    elements.menuBackgroundStatus.textContent = customMenuTheme.frames.length > 1 ? `${file.name} — ${customMenuTheme.frames.length} optimized animation frames` : `${file.name} — optimized static background`;
    rebuildMenuTheme(); resetResult(); updateEstimate();
  } catch (error) {
    customMenuTheme = null;
    elements.menuBackgroundStatus.textContent = `Could not read this image or GIF: ${error instanceof Error ? error.message : String(error)}`;
  }
}


function titleCardSourceName() { return entries[0]?.file?.name || "GBA VIDEO"; }
function ensureTitleCardProject(force = false) {
  const source = titleCardSourceName();
  if (force || !titleCardProject || titleCardProjectSource !== source) {
    titleCardProject = createTitleCardProject(source);
    titleCardProjectSource = source;
    titleCardPart = 1;
  }
}
function titleCardPartRecord(part, create = false) {
  ensureTitleCardProject();
  if (titleCardProject.useShared) return titleCardProject.shared;
  let record = (titleCardProject.parts || []).find((item) => Number(item.part) === Number(part));
  if (!record && create) {
    record = { part: Number(part), settings: structuredClone(titleCardProject.shared) };
    titleCardProject.parts.push(record);
  }
  return record?.settings || titleCardProject.shared;
}
function serializedTitleCards() {
  ensureTitleCardProject();
  const copy = structuredClone(titleCardProject || createTitleCardProject(titleCardSourceName()));
  copy.enabled = Boolean(elements.partTitleScreens.checked);
  copy.useShared = Boolean(elements.titleCardUseShared.checked);
  return copy;
}
function titleCardReadout(input, output, fallback) {
  if (!input || !output) return;
  const color = describeColor(input.value, fallback);
  output.textContent = `${color.hex} · RGB555 ${color.r},${color.g},${color.b}`;
  input._gbaColorPickerController?.sync();
}
function updateTitleCardReadouts() {
  titleCardReadout(elements.titleCardTextColor, elements.titleCardTextColorValue, "#FFFFFF");
  titleCardReadout(elements.titleCardOutlineColor, elements.titleCardOutlineColorValue, "#000000");
  titleCardReadout(elements.titleCardSubtitleTextColor, elements.titleCardSubtitleTextColorValue, "#FFFFFF");
  titleCardReadout(elements.titleCardSubtitleOutlineColor, elements.titleCardSubtitleOutlineColorValue, "#000000");
  titleCardReadout(elements.titleCardSolidColor, elements.titleCardSolidColorValue, "#000000");
}
function updateTitleCardTextWarning() {
  if (!elements.titleCardTextWarning) return;
  const subtitleForCheck=elements.titleCardSubtitle.value.replaceAll("{part}","1");
  const unsupported=[...new Set([...unsupportedGBARunes(elements.titleCardTitle.value), ...unsupportedGBARunes(subtitleForCheck)])];
  elements.titleCardTextWarning.textContent=unsupported.length ? `Unsupported GBA characters: ${unsupported.join(" ")}. They will be replaced in the ROM.` : "";
  elements.titleCardTextWarning.hidden=unsupported.length === 0;
}
function titleCardFormSettings() {
  return {
    title: elements.titleCardTitle.value,
    subtitle: elements.titleCardSubtitle.value,
    backgroundMode: elements.titleCardBackground.value,
    frameOffsetSeconds: Number(elements.titleCardFrameOffset.value) || 0,
    darkness: Number(elements.titleCardDarkness.value) || 0,
    solidColor: elements.titleCardSolidColor.value,
    titleTextColor: elements.titleCardTextColor.value,
    titleOutlineColor: elements.titleCardOutlineColor.value,
    titleAlignment: elements.titleCardAlignment.value,
    titleTextSize: elements.titleCardTextSize.value,
    subtitleTextColor: elements.titleCardSubtitleTextColor.value,
    subtitleOutlineColor: elements.titleCardSubtitleOutlineColor.value,
    subtitleAlignment: elements.titleCardSubtitleAlignment.value,
    subtitleTextSize: elements.titleCardSubtitleTextSize.value,
    drawOutline: elements.titleCardOutline.checked,
    startMode: elements.titleCardStartMode.value,
    durationSeconds: Number(elements.titleCardDuration.value) || 3,
    allowSkip: elements.titleCardAllowSkip.checked,
    fade: elements.titleCardFade.checked,
  };
}
function updateTitleCardConditionalFields() {
  const background = elements.titleCardBackground.value;
  elements.titleCardFrameOffsetField.hidden = background !== "part-frame";
  elements.titleCardSolidColorField.hidden = background !== "solid";
  elements.titleCardDurationField.hidden = elements.titleCardStartMode.value !== "timer";
  const enabled = elements.partTitleScreens.checked && !conversionRunning;
  elements.titleCardAllowSkip.disabled = !enabled || elements.titleCardStartMode.value !== "timer";
  elements.titleCardOutlineColor.disabled = !enabled || !elements.titleCardOutline.checked;
  elements.titleCardSubtitleOutlineColor.disabled = !enabled || !elements.titleCardOutline.checked;
  elements.titleCardOutlineColor._gbaColorPickerController?.sync();
  elements.titleCardSubtitleOutlineColor._gbaColorPickerController?.sync();
}
function saveTitleCardFields() {
  const target = titleCardPartRecord(titleCardPart, true);
  Object.assign(target, titleCardFormSettings());
  elements.titleCardDarknessValue.textContent = `${target.darkness}%`;
  updateTitleCardConditionalFields();
  updateTitleCardReadouts();
  updateTitleCardTextWarning();
  renderCurrentTitleCardPreview();
  resetResult();
}
function loadTitleCardFields() {
  ensureTitleCardProject();
  const settings = titleCardPartRecord(titleCardPart, false) || defaultTitleCardSettings(titleCardSourceName());
  elements.titleCardTitle.value = settings.title ?? "";
  elements.titleCardSubtitle.value = settings.subtitle ?? "";
  elements.titleCardBackground.value = settings.backgroundMode || "part-first-frame";
  elements.titleCardFrameOffset.value = Number(settings.frameOffsetSeconds) || 0;
  elements.titleCardDarkness.value = Number.isFinite(Number(settings.darkness)) ? Number(settings.darkness) : 50;
  elements.titleCardDarknessValue.textContent = `${elements.titleCardDarkness.value}%`;
  elements.titleCardSolidColor.value = settings.solidColor || "#000000";
  elements.titleCardTextColor.value = settings.titleTextColor || settings.textColor || "#FFFFFF";
  elements.titleCardOutlineColor.value = settings.titleOutlineColor || settings.outlineColor || "#000000";
  elements.titleCardSubtitleTextColor.value = settings.subtitleTextColor || settings.textColor || "#FFFFFF";
  elements.titleCardSubtitleOutlineColor.value = settings.subtitleOutlineColor || settings.outlineColor || "#000000";
  elements.titleCardOutline.checked = settings.drawOutline !== false;
  elements.titleCardAlignment.value = settings.titleAlignment || settings.alignment || "center";
  elements.titleCardTextSize.value = ["large", "medium", "small"].includes(settings.titleTextSize) ? settings.titleTextSize : (["medium", "small"].includes(settings.textSize) ? settings.textSize : "large");
  elements.titleCardSubtitleAlignment.value = settings.subtitleAlignment || settings.alignment || "center";
  elements.titleCardSubtitleTextSize.value = ["large", "medium", "small"].includes(settings.subtitleTextSize) ? settings.subtitleTextSize : (settings.textSize === "large" ? "medium" : "small");
  elements.titleCardStartMode.value = settings.startMode === "timer" ? "timer" : "button";
  elements.titleCardDuration.value = Number(settings.durationSeconds) || 3;
  elements.titleCardAllowSkip.checked = settings.allowSkip !== false;
  elements.titleCardFade.checked = settings.fade !== false;
  updateTitleCardConditionalFields();
  updateTitleCardReadouts();
  updateTitleCardTextWarning();
  renderCurrentTitleCardPreview();
}
function titleCardPartSourceTime(part) {
  const entry = entries[0];
  if (!entry) return 0;
  const start = entry.useProject ? clampNumber(elements.defaultStart.value, 0, entry.duration || 86400) : clampNumber(entry.start, 0, entry.duration || 86400);
  let end = entry.useProject ? Math.max(0, numericOr(elements.defaultEnd.value, 0)) : Math.max(0, numericOr(entry.end, 0));
  if (!(end > start)) end = entry.duration || start;
  const segment = Math.max(0, end - start) / Math.max(1, titleCardEstimatedParts);
  const settings = titleCardPartRecord(part, false) || {};
  const offset = settings.backgroundMode === "part-frame" ? Math.max(0, Number(settings.frameOffsetSeconds) || 0) : 0;
  return Math.min(Math.max(start, end - 0.04), start + (part - 1) * segment + offset);
}
async function ensureTitleCardPreviewVideo() {
  const entry = entries[0];
  if (!entry) return null;
  if (!titleCardPreviewVideo) {
    titleCardPreviewVideo = document.createElement("video");
    titleCardPreviewVideo.muted = true;
    titleCardPreviewVideo.playsInline = true;
    titleCardPreviewVideo.preload = "auto";
  }
  const key = `${entry.file.name}:${entry.file.size}:${entry.file.lastModified}`;
  if (titleCardPreviewVideo.dataset.key !== key) {
    if (titleCardPreviewURL) URL.revokeObjectURL(titleCardPreviewURL);
    titleCardPreviewURL = URL.createObjectURL(entry.file);
    titleCardPreviewVideo.dataset.key = key;
    titleCardPreviewVideo.src = titleCardPreviewURL;
    await new Promise((resolve, reject) => {
      titleCardPreviewVideo.onloadedmetadata = () => resolve();
      titleCardPreviewVideo.onerror = () => reject(new Error("Could not load title-card preview frame."));
    });
  }
  return titleCardPreviewVideo;
}
async function renderCurrentTitleCardPreview() {
  if (!elements.titleCardPreview || elements.titleCardGroup.hidden || !entries.length) return;
  const token = ++titleCardPreviewToken;
  const settings = titleCardPartRecord(titleCardPart, false);
  const fit = entries[0].useProject ? elements.fitMode.value : entries[0].fitMode;
  if (settings?.backgroundMode === "solid") {
    renderTitleCardPreview(elements.titleCardPreview, elements.titleCardPreview, fit, settings, titleCardPart, titleCardSourceName());
    return;
  }
  try {
    const video = await ensureTitleCardPreviewVideo();
    if (!video || token !== titleCardPreviewToken) return;
    const time = Math.max(0, Math.min(Math.max(0, video.duration - 0.04), titleCardPartSourceTime(titleCardPart)));
    if (Math.abs(video.currentTime - time) > 0.025) {
      await new Promise((resolve) => {
        const done = () => { video.removeEventListener("seeked", done); resolve(); };
        video.addEventListener("seeked", done);
        video.currentTime = time;
      });
    }
    if (token !== titleCardPreviewToken) return;
    renderTitleCardPreview(elements.titleCardPreview, video, fit, settings, titleCardPart, titleCardSourceName());
  } catch {
    const fallback = { ...settings, backgroundMode: "solid", solidColor: "#000000" };
    renderTitleCardPreview(elements.titleCardPreview, elements.titleCardPreview, fit, fallback, titleCardPart, titleCardSourceName());
  }
}
function updateTitleCardNavState() {
  elements.titleCardPartSelect.value = String(titleCardPart);
  elements.titleCardPartLabel.textContent = `of ${titleCardEstimatedParts}`;
  const enabled = !conversionRunning && elements.partTitleScreens.checked;
  elements.titleCardPrev.disabled = !enabled || titleCardPart <= 1;
  elements.titleCardNext.disabled = !enabled || titleCardPart >= titleCardEstimatedParts;
}
function setTitleCardPart(part, force = false) {
  const nextPart = Math.max(1, Math.min(titleCardEstimatedParts, Number(part) || 1));
  const changed = nextPart !== titleCardPart;
  titleCardPart = nextPart;
  updateTitleCardNavState();
  if (changed || force) loadTitleCardFields();
}
function updateTitleCardVisibility(estimate = lastEstimate) {
  const wasVisible = !elements.titleCardGroup.hidden;
  const visible = entries.length === 1 && elements.outputMode.value === "rom" && (Number(estimate?.parts || 1) > 1 || elements.splitVideo.checked);
  elements.titleCardGroup.hidden = !visible;
  if (!visible) {
    titleCardVisibilitySignature = "";
    return;
  }
  ensureTitleCardProject();
  const previousParts = titleCardEstimatedParts;
  titleCardEstimatedParts = Math.max(2, Number(estimate.parts) || 2);
  titleCardProject.enabled = elements.partTitleScreens.checked;
  titleCardProject.useShared = elements.titleCardUseShared.checked;
  if (elements.titleCardPartSelect.options.length !== titleCardEstimatedParts) {
    elements.titleCardPartSelect.replaceChildren(...Array.from({ length: titleCardEstimatedParts }, (_, index) => {
      const option = document.createElement("option"); option.value = String(index + 1); option.textContent = `Part ${index + 1}`; return option;
    }));
  }
  titleCardPart = Math.min(titleCardPart, titleCardEstimatedParts);
  const entry = entries[0];
  const start = entry.useProject ? elements.defaultStart.value : entry.start;
  const end = entry.useProject ? elements.defaultEnd.value : entry.end;
  const fit = entry.useProject ? elements.fitMode.value : entry.fitMode;
  const signature = `${entry.file.name}|${entry.file.size}|${entry.file.lastModified}|${titleCardEstimatedParts}|${start}|${end}|${fit}`;
  const sourceChanged = signature !== titleCardVisibilitySignature;
  titleCardVisibilitySignature = signature;
  const enabled = elements.partTitleScreens.checked && !conversionRunning;
  elements.titleCardControls.classList.toggle("disabled-panel", !enabled);
  for (const control of elements.titleCardControls.querySelectorAll("input,select,button")) control.disabled = !enabled;
  for (const control of [elements.titleCardUseShared, elements.titleCardOutline, elements.titleCardAllowSkip, elements.titleCardFade]) control.disabled = !enabled;
  elements.titleCardCopyToAll.hidden = elements.titleCardUseShared.checked;
  if (!wasVisible || previousParts !== titleCardEstimatedParts || sourceChanged) setTitleCardPart(titleCardPart, true);
  else updateTitleCardNavState();
  updateTitleCardConditionalFields();
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes < 1) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB"];
  const unit = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / 1024 ** unit).toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function numericOr(value, fallback) {
  const number = Number(value);
  return Number.isFinite(number) ? number : fallback;
}

function clampNumber(value, minimum, maximum) {
  return Math.min(maximum, Math.max(minimum, numericOr(value, minimum)));
}

function titleFromFile(file) {
  const base = file.name.replace(/\.[^.]+$/, "").replace(/[_-]+/g, " ").trim();
  return sanitizeMenuTitle(base) || "VIDEO";
}

function cleanFileBase(value, fallback = "GBA_VIDEO") {
  return String(value || fallback)
    .trim()
    .replace(/[^A-Za-z0-9_-]+/g, "_")
    .replace(/^_+|_+$/g, "") || fallback;
}

function outputFileName(romTitle, extension = "gba") {
  return `${cleanFileBase(romTitle, "gba-video")}.${extension}`;
}

function makeEntry(file) {
  return {
    id: crypto.randomUUID(),
    file,
    title: titleFromFile(file),
    useProject: true,
    start: 0,
    end: 0,
    speed: 1,
    fitMode: "fit",
    audioMode: "mix",
    audioTrack: 0,
    audioTracks: [],
    audioTracksKnown: false,
    volume: 1,
    loop: false,
    paletteMode: "shared",
    ditherMode: "ordered",
    duration: 0,
    hasAudio: true,
    channels: 0,
    chapters: [],
  };
}

function addFiles(fileList) {
  const existing = new Set(entries.map((entry) => `${entry.file.name}:${entry.file.size}:${entry.file.lastModified}`));
  for (const file of fileList) {
    if (!file.type.startsWith("video/") && !/\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(file.name)) continue;
    const key = `${file.name}:${file.size}:${file.lastModified}`;
    if (existing.has(key)) continue;
    const entry = makeEntry(file);
    entries.push(entry);
    if (!selectedEntryId) selectedEntryId = entry.id;
    existing.add(key);
    hydrateEntryMetadata(entry).then(() => { applyPendingProjectMatches(); renderFiles(); updateEstimate(); }).catch(() => {});
  }
  if (fileList.length) resetResult();
  renderFiles();
}



function sanitizeMenuTitle(value) {
  return sanitizeGBAText(value, 12).text;
}

async function hydrateEntryMetadata(entry) {
  if (!entry?.file) return;
  const duration = await readBrowserDuration(entry.file);
  if (duration > 0) entry.duration = duration;
}

function selectedEntry() {
  return entries.find((entry) => entry.id === selectedEntryId) || entries[0] || null;
}

function durationForEntry(entry, project = currentOptions(false)) {
  const clip = effectiveClipOptions(entry, project);
  const full = Math.max(0, entry.duration || 0);
  const start = clampNumber(clip.start, 0, full || 86400);
  const end = clip.end > start ? Math.min(clip.end, full || clip.end) : full;
  return Math.max(0, end - start) / clampNumber(clip.speed, 0.5, 3);
}

function updateOutputModes() {
  const previous = elements.outputMode.value || preferredOutputMode;
  const choices = entries.length <= 1
    ? [["rom", "Single ROM"]]
    : [["playlist", "One ROM — play clips in order"], ["menu", "One ROM — clip menu"], ["batch", "Separate ROMs in ZIP"]];
  elements.outputMode.replaceChildren();
  for (const [value, text] of choices) {
    const option = document.createElement("option");
    option.value = value;
    option.textContent = text;
    elements.outputMode.append(option);
  }
  const desired = choices.some(([value]) => value === previous) ? previous : (choices.some(([value]) => value === preferredOutputMode) ? preferredOutputMode : choices[0][0]);
  elements.outputMode.value = desired;
  preferredOutputMode = desired;
  const single = entries.length === 1 && elements.outputMode.value === "rom";
  elements.splitVideoRow.hidden = !single;
  if (!single) elements.splitVideo.checked = false;
  if (elements.menuSettingsGroup) elements.menuSettingsGroup.hidden = !(entries.length > 1 && elements.outputMode.value === "menu");
  if (elements.titleCardGroup && !single) elements.titleCardGroup.hidden = true;
  updateSplitVisibility();
}

function updateSplitVisibility() {
  const visible = entries.length === 1 && elements.outputMode.value === "rom" && elements.splitVideo.checked;
  elements.splitOptions.hidden = !visible;
  elements.splitBudgetValue.textContent = `${elements.splitBudget.value} MiB`;
}

function projectSettingsSnapshot() {
  const project = currentOptions();
  return {
    ...project,
    preset: elements.preset.value,
    smartAnalysis,
    splitVideo: elements.splitVideo.checked,
    splitBudgetMiB: Number(elements.splitBudget.value),
    maxPartDuration: elements.maxPartDuration.value,
    chapterAware: elements.chapterAware.checked,
    partTitleScreens: elements.partTitleScreens.checked,
    titleCards: serializedTitleCards(),
    resumeLongSplit: elements.resumeLongSplit.checked,
    menuTheme: includeMenuTheme && elements.outputMode.value === "menu" ? serializedMenuTheme() : null,
  };
}

function clipSnapshot(entry) {
  return {
    source: { name: entry.file.name, size: entry.file.size, lastModified: entry.file.lastModified },
    title: entry.title,
    useProject: entry.useProject,
    start: entry.start,
    end: entry.end,
    speed: entry.speed,
    fitMode: entry.fitMode,
    audioMode: entry.audioMode,
    audioTrack: Number(entry.audioTrack) || 0,
    volume: entry.volume,
    loop: entry.loop,
    paletteMode: entry.paletteMode,
    ditherMode: entry.ditherMode,
  };
}

function downloadBlob(blob, fileName) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function saveProject() {
  const data = {
    format: "GBA Video Maker Project",
    version: 1,
    settings: projectSettingsSnapshot(),
    clips: entries.map(clipSnapshot),
  };
  const base = cleanFileBase(elements.romTitle.value || "GBA_VIDEO", "GBA_VIDEO");
  downloadBlob(new Blob([JSON.stringify(data, null, 2) + "\n"], { type: "application/json" }), `${base}.gbavideo`);
}

function applySettings(settings = {}) {
  const assign = (element, key, fallback = undefined) => {
    if (settings[key] !== undefined && settings[key] !== null) element.value = String(settings[key]);
    else if (fallback !== undefined) element.value = String(fallback);
  };
  preferredOutputMode = settings.outputMode || "rom";
  assign(elements.outputMode, "outputMode", preferredOutputMode);
  assign(elements.vblanks, "vblanks", 5);
  assign(elements.fitMode, "fitMode", "fit");
  assign(elements.paletteMode, "paletteMode", "shared");
  assign(elements.ditherMode, "ditherMode", "ordered");
  assign(elements.compression, "compression", "delta");
  assign(elements.audioMode, "audioMode", "mix");
  assign(elements.audioQuality, "audioQuality", "pcm");
  assign(elements.smartTarget, "smartTargetMiB", 32);
  assign(elements.smartPriority, "smartPriority", "balanced");
  assign(elements.seekSeconds, "seekSeconds", 5);
  assign(elements.defaultStart, "defaultStart", 0);
  assign(elements.defaultEnd, "defaultEnd", "");
  assign(elements.defaultSpeed, "defaultSpeed", 1);
  if (settings.defaultVolume !== undefined) {
    const savedVolume = Number(settings.defaultVolume);
    elements.defaultVolume.value = String(savedVolume <= 2 ? Math.round(savedVolume * 100) : savedVolume);
  } else elements.defaultVolume.value = "100";
  assign(elements.romTitle, "romTitle", "GBA VIDEO");
  elements.defaultLoop.checked = Boolean(settings.defaultLoop);
  elements.normalize.checked = Boolean(settings.normalize);
  elements.limiter.checked = settings.limiter !== false;
  elements.resume.checked = settings.resume !== false;
  elements.splitVideo.checked = Boolean(settings.splitVideo);
  assign(elements.splitBudget, "splitBudgetMiB", 31);
  assign(elements.maxPartDuration, "maxPartDuration", "0");
  elements.chapterAware.checked = settings.chapterAware !== false;
  elements.partTitleScreens.checked = settings.titleCards?.enabled ?? (settings.partTitleScreens !== false);
  titleCardProject = settings.titleCards ? structuredClone(settings.titleCards) : null;
  titleCardProjectSource = titleCardProject ? titleCardSourceName() : "";
  if (titleCardProject) elements.titleCardUseShared.checked = titleCardProject.useShared !== false;
  elements.resumeLongSplit.checked = settings.resumeLongSplit !== false;
  if (elements.menuBackground) {
    elements.menuBackground.value = settings.menuBackground || settings.menuTheme?.id || "ocean-wave-animated";
    restoreMenuColors(settings);
    elements.menuOutline.checked = settings.menuOutline !== false;
    customMenuTheme = elements.menuBackground.value === "custom" && settings.menuTheme ? deserializeTheme(settings.menuTheme) : null;
    rebuildMenuTheme();
  }
  if (settings.preset) elements.preset.value = settings.preset;
  if (elements.preset.value !== "extreme") elements.audioQuality.value = "pcm";
  smartAnalysis = settings.smartAnalysis || null;
  updateExtremeVisibility();
  if (smartAnalysis?.recommended) {
    renderSmartResults(smartAnalysis);
    elements.smartStatus.textContent = `Saved analysis restored · ${smartAnalysis.confidence || "unknown"} confidence`;
  }
}

function applyPendingProjectMatches() {
  if (!pendingProject?.clips?.length) return;
  const unmatched = [];
  const orderedMatches = [];
  const used = new Set();
  for (const saved of pendingProject.clips) {
    const match = entries.find((entry) => !used.has(entry.id) && entry.file.name === saved.source?.name && entry.file.size === saved.source?.size);
    if (!match) { unmatched.push(saved); continue; }
    used.add(match.id);
    Object.assign(match, {
      title: sanitizeMenuTitle(saved.title || titleFromFile(match.file)),
      useProject: saved.useProject !== false,
      start: numericOr(saved.start, 0),
      end: numericOr(saved.end, 0),
      speed: numericOr(saved.speed, 1),
      fitMode: saved.fitMode || "fit",
      audioMode: saved.audioMode || "mix",
      audioTrack: Number.isInteger(saved.audioTrack) ? saved.audioTrack : 0,
      volume: numericOr(saved.volume, 1),
      loop: Boolean(saved.loop),
      paletteMode: saved.paletteMode || "shared",
      ditherMode: saved.ditherMode || "ordered",
    });
    orderedMatches.push(match);
  }
  if (orderedMatches.length) entries = [...orderedMatches, ...entries.filter((entry) => !used.has(entry.id))];
  pendingProject.clips = unmatched;
  elements.projectNotice.textContent = unmatched.length
    ? `Project loaded. Select ${unmatched.length} missing source video${unmatched.length === 1 ? "" : "s"}.`
    : "Project loaded and source videos relinked.";
  if (!unmatched.length) pendingProject = null;
}

async function openProjectFile(file) {
  const parsed = JSON.parse(await file.text());
  if (!parsed || parsed.format !== "GBA Video Maker Project" || !Array.isArray(parsed.clips)) throw new Error("This is not a valid .gbavideo project.");
  applySettings(parsed.settings || {});
  pendingProject = parsed;
  applyPendingProjectMatches();
  updateOutputModes();
  renderFiles();
  updateEstimate();
  if (pendingProject) elements.projectNotice.textContent = `Project loaded. Select ${pendingProject.clips.length} source video${pendingProject.clips.length === 1 ? "" : "s"} to relink.`;
}

function estimateProject(project = currentOptions(false)) {
  const fps = GBA_REFRESH / project.vblanks;
  let frames = 0;
  let videoBytes = 0;
  let audioBytes = 0;
  let paletteBytes = 0;
  let totalDuration = 0;
  for (const entry of entries) {
    const clip = effectiveClipOptions(entry, project);
    const duration = durationForEntry(entry, project);
    const clipFrames = Math.max(1, Math.ceil(duration * fps));
    const raw = clipFrames * 120 * 80;
    const ratio = project.compression === "none" ? 1 : (clip.paletteMode === "scene" ? 0.43 : 0.34);
    frames += clipFrames;
    const adaptiveFactor = project.extremeOptimization && project.adaptiveKeyframes ? 0.91 : 1;
    videoBytes += (raw * ratio + (project.compression === "delta" ? clipFrames * 12 : 0)) * adaptiveFactor;
    if (clip.audioMode !== "none" && entry.hasAudio !== false) {
      const pcmBytes = duration * AUDIO_RATE;
      const codec = project.audioQuality === "auto" ? (smartAnalysis?.recommended?.audioCodec || "pcm") : project.audioQuality;
      audioBytes += codec === "adpcm" ? pcmBytes * 0.505 + 20 : pcmBytes;
    }
    paletteBytes += (clip.paletteMode === "scene" ? Math.max(1, Math.ceil(duration / 30)) : 1) * 512 + clipFrames * (clip.paletteMode === "scene" ? 2 : 0);
    totalDuration += duration;
  }
  const metadataBytes = 0x8000 + entries.length * 96 + frames * 8 + (project.outputMode === "menu" ? menuThemeBytes() : 0);
  let totalBytes = metadataBytes + videoBytes + audioBytes + paletteBytes;
  const budgetMiB = elements.splitVideo.checked ? Number(elements.splitBudget.value) : 32;
  const budget = Math.max(1, Math.min(32, budgetMiB)) * 1048576;
  const maxPartSeconds = elements.splitVideo.checked ? parseClock(elements.maxPartDuration.value) : 0;
  let partsBySize = Math.max(1, Math.ceil(totalBytes / budget));
  let partsByDuration = Number.isFinite(maxPartSeconds) && maxPartSeconds > 0 ? Math.max(1, Math.ceil(totalDuration / maxPartSeconds)) : 1;
  const needsSplit = entries.length === 1 && elements.outputMode.value === "rom" && (elements.splitVideo.checked || totalBytes > ROM_LIMIT);
  let parts = needsSplit ? Math.max(partsBySize, partsByDuration) : 1;
  if (needsSplit && elements.partTitleScreens.checked) {
    for (let pass = 0; pass < 2; pass += 1) {
      totalBytes = metadataBytes + videoBytes + audioBytes + paletteBytes + TITLE_CARD_BYTES * parts;
      partsBySize = Math.max(1, Math.ceil(totalBytes / budget));
      parts = Math.max(partsBySize, partsByDuration);
    }
  }
  return { fps, frames, videoBytes, audioBytes, paletteBytes, metadataBytes, totalBytes, totalDuration, parts, needsSplit, budget };
}

function updateEstimate() {
  if (!entries.length) {
    lastEstimate = null;
    elements.estimateArea.textContent = "Add a video to see an output estimate.";
    elements.optimizerButton.disabled = true;
    if (elements.titleCardGroup) elements.titleCardGroup.hidden = true;
    return;
  }
  const durationError = elements.splitVideo.checked && !Number.isFinite(parseClock(elements.maxPartDuration.value));
  if (durationError) {
    elements.estimateArea.textContent = "Maximum duration must be 0, seconds, MM:SS, or H:MM:SS.";
    elements.optimizerButton.disabled = false;
    return;
  }
  lastEstimate = estimateProject();
  const firstLine = lastEstimate.parts > 1 ? `Estimated output: ${lastEstimate.parts} ROM parts` : `Estimated output: 1 ROM`;
  elements.estimateArea.innerHTML = `<strong>${firstLine}</strong> · Cartridge target: ${formatBytes(lastEstimate.budget)}<br>` +
    `Estimated data: ${formatBytes(lastEstimate.totalBytes)} · ${lastEstimate.frames.toLocaleString()} frames · ${lastEstimate.fps.toFixed(2)} fps<br>` +
    `Video ${formatBytes(lastEstimate.videoBytes)} · Audio ${formatBytes(lastEstimate.audioBytes)} · Palettes/indexes ${formatBytes(lastEstimate.paletteBytes)}`;
  elements.optimizerButton.disabled = conversionRunning || !entries.length;
  updateTitleCardVisibility(lastEstimate);
}

function optimizeToFit() {
  if (!entries.length) return;
  const controls = [elements.compression, elements.paletteMode, elements.ditherMode, elements.vblanks, elements.audioMode];
  const original = controls.map((control) => control.value);
  let estimate = estimateProject();
  if (estimate.totalBytes <= ROM_LIMIT) {
    elements.projectNotice.textContent = "The current estimate already fits one 32 MiB ROM.";
    return;
  }
  const proposals = [
    [elements.compression, "delta", "enable delta + keyframe compression"],
    [elements.paletteMode, "shared", "use a shared palette"],
    [elements.ditherMode, "off", "disable dithering"],
    [elements.vblanks, "8", "use Compact 7.47 FPS"],
    [elements.audioMode, "none", "disable audio"],
  ];
  const changes = [];
  for (const [control, value, description] of proposals) {
    if (control.value === value) continue;
    control.value = value;
    changes.push(description);
    estimate = estimateProject();
    if (estimate.totalBytes <= ROM_LIMIT) break;
  }
  const proposalValues = controls.map((control) => control.value);
  controls.forEach((control, index) => { control.value = original[index]; });
  const message = `${changes.length ? changes.map((change) => `• ${change}`).join("\n") : "No smaller settings remain."}\n\nEstimated result: ${formatBytes(estimate.totalBytes)}.${estimate.totalBytes > ROM_LIMIT ? " Automatic splitting will still be needed." : ""}\n\nApply these changes?`;
  if (!changes.length || !confirm(message)) return;
  controls.forEach((control, index) => { control.value = proposalValues[index]; });
  elements.preset.value = "custom";
  updateEstimate();
  syncSelectedPreview();
  elements.projectNotice.textContent = estimate.totalBytes <= ROM_LIMIT
    ? "Applied the reviewed optimizer proposal."
    : "Applied the smallest reviewed proposal; automatic splitting is still required.";
}

function updateTitlePreview() {
  const entry = selectedEntry();
  if (entry?.id !== menuTitleUnsupportedEntryId) menuTitleUnsupported = [];
  const canvas = elements.titlePreview;
  const context = canvas.getContext("2d");
  context.imageSmoothingEnabled = false;
  context.fillStyle = "#000";
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = "#ffdd00";
  const scale = 4;
  const startX = 8;
  const startY = 6;
  const rawTitle = sanitizeMenuTitle(entry?.title || "");
  const title = rawTitle.padEnd(12, " ").slice(0, 12);
  [...title].forEach((character, index) => {
    const bits = glyphBits(character);
    for (let row = 0; row < 5; row += 1) {
      for (let column = 0; column < 3; column += 1) {
        const bit = 14 - (row * 3 + column);
        if (bits & (1 << bit)) context.fillRect(startX + index * 16 + column * scale, startY + row * scale, scale, scale);
      }
    }
  });

  if (document.activeElement === elements.titlePreviewInput) {
    const caretIndex = Math.min(elements.titlePreviewInput.selectionStart ?? glyphLength(rawTitle), 12);
    const caretX = Math.min(canvas.width - 4, startX + caretIndex * 16);
    context.fillRect(caretX, startY, 2, 20);
  }

  if (elements.titlePreviewInput && elements.titlePreviewInput.value !== rawTitle) {
    elements.titlePreviewInput.value = rawTitle;
  }
  const duplicates = entry ? entries.filter((candidate) => candidate.id !== entry.id && candidate.title === entry.title && entry.title).length : 0;
  elements.titleWarning.textContent = menuTitleUnsupported.length ? `Unsupported GBA characters: ${menuTitleUnsupported.join(" ")}. They were replaced.` : (duplicates ? "Another clip uses the same menu title." : "");
}

function revokePreviewURLs() {
  if (previewURL) URL.revokeObjectURL(previewURL);
  previewURL = "";
  if (audioPreviewURL) URL.revokeObjectURL(audioPreviewURL);
  audioPreviewURL = "";
}

function syncSelectedPreview(force = false) {
  const entry = selectedEntry();
  elements.previewCard.hidden = !entry;
  if (!entry) { revokePreviewURLs(); return; }
  elements.selectedClipName.textContent = entry.file.name;
  if (force || elements.previewVideo.dataset.entryId !== entry.id) {
    if (previewURL) URL.revokeObjectURL(previewURL);
    previewURL = URL.createObjectURL(entry.file);
    elements.previewVideo.src = previewURL;
    elements.previewVideo.dataset.entryId = entry.id;
    elements.previewVideo.muted = true;
  }
  const duration = Math.max(0.01, entry.duration || elements.previewVideo.duration || 1);
  const project = currentOptions(false);
  const clip = effectiveClipOptions(entry, project);
  const start = clampNumber(clip.start, 0, duration);
  const end = clip.end > start ? Math.min(clip.end, duration) : duration;
  for (const control of [elements.timelinePlay, elements.timelineStart, elements.timelineEnd]) {
    control.max = String(duration);
    control.step = "0.04";
  }
  elements.timelineStart.value = String(start);
  elements.timelineEnd.value = String(end);
  if (Number(elements.timelinePlay.value) < start || Number(elements.timelinePlay.value) > end) elements.timelinePlay.value = String(start);
  updateTimelineLabels();
  updateTitlePreview();
  renderTimelineThumbnails(entry);
}

function updateTimelineLabels() {
  const duration = Math.max(0.01, Number(elements.timelinePlay.max) || elements.previewVideo.duration || 1);
  const start = clampNumber(elements.timelineStart.value, 0, duration);
  const current = clampNumber(elements.timelinePlay.value, 0, duration);
  const end = clampNumber(elements.timelineEnd.value, 0, duration);
  const percent = (value) => `${Math.max(0, Math.min(100, (value / duration) * 100)).toFixed(4)}%`;

  if (document.activeElement !== elements.timelineStartTimeInput) elements.timelineStartTimeInput.value = formatClock(start, true);
  elements.timelineCurrentText.textContent = formatClock(current, true);
  if (document.activeElement !== elements.timelineEndTimeInput) elements.timelineEndTimeInput.value = formatClock(end, true);
  elements.timelineTrack.style.setProperty("--timeline-start", percent(start));
  elements.timelineTrack.style.setProperty("--timeline-current", percent(current));
  elements.timelineTrack.style.setProperty("--timeline-end", percent(end));
}

function timelineValueFromClientX(clientX) {
  const rect = elements.timelineTrack.getBoundingClientRect();
  const duration = Math.max(0.01, Number(elements.timelinePlay.max) || elements.previewVideo.duration || 1);
  const ratio = rect.width > 0 ? clampNumber((clientX - rect.left) / rect.width, 0, 1) : 0;
  return ratio * duration;
}

function setTimelinePreview(value) {
  const duration = Math.max(0.01, Number(elements.timelinePlay.max) || elements.previewVideo.duration || 1);
  const start = Number(elements.timelineStart.value);
  const end = Number(elements.timelineEnd.value);
  const clamped = clampNumber(value, Math.max(0, start), Math.min(duration, end));
  elements.timelinePlay.value = String(clamped);
  elements.previewVideo.currentTime = clamped;
  updateTimelineLabels();
}

function setTimelineBoundaryPreview(kind, value) {
  const duration = Math.max(0.01, Number(elements.timelinePlay.max) || elements.previewVideo.duration || 1);
  const start = Number(elements.timelineStart.value);
  const end = Number(elements.timelineEnd.value);
  let clamped = clampNumber(value, 0, duration);
  if (kind === "start") {
    clamped = Math.min(clamped, Math.max(0, end - 0.04));
    elements.timelineStart.value = String(clamped);
    if (Number(elements.timelinePlay.value) < clamped) setTimelinePreview(clamped);
  } else {
    clamped = Math.max(clamped, Math.min(duration, start + 0.04));
    elements.timelineEnd.value = String(clamped);
    if (Number(elements.timelinePlay.value) > clamped) setTimelinePreview(clamped);
  }
  updateTimelineLabels();
}

function beginTimelineDrag(kind, event) {
  if (conversionRunning) return;
  event.preventDefault();
  event.stopPropagation();
  const handle = event.currentTarget;
  handle.setPointerCapture?.(event.pointerId);

  const update = (pointerEvent) => {
    const value = timelineValueFromClientX(pointerEvent.clientX);
    if (kind === "current") setTimelinePreview(value);
    else setTimelineBoundaryPreview(kind, value);
  };
  const finish = (pointerEvent) => {
    update(pointerEvent);
    handle.releasePointerCapture?.(pointerEvent.pointerId);
    handle.removeEventListener("pointermove", update);
    handle.removeEventListener("pointerup", finish);
    handle.removeEventListener("pointercancel", finish);
    if (kind === "start") applyTimelineBoundary("start", elements.timelineStart.value);
    if (kind === "end") applyTimelineBoundary("end", elements.timelineEnd.value);
  };

  update(event);
  handle.addEventListener("pointermove", update);
  handle.addEventListener("pointerup", finish);
  handle.addEventListener("pointercancel", finish);
}

function nudgeTimeline(kind, direction) {
  const step = 0.04 * direction;
  if (kind === "current") setTimelinePreview(Number(elements.timelinePlay.value) + step);
  else {
    setTimelineBoundaryPreview(kind, Number(elements[kind === "start" ? "timelineStart" : "timelineEnd"].value) + step);
    applyTimelineBoundary(kind, kind === "start" ? elements.timelineStart.value : elements.timelineEnd.value);
  }
}

async function renderTimelineThumbnails(entry) {
  const token = ++thumbRenderToken;
  elements.timelineThumbs.replaceChildren();
  if (!entry?.duration || entry.duration <= 0) return;
  const video = document.createElement("video");
  const url = URL.createObjectURL(entry.file);
  video.src = url;
  video.muted = true;
  video.preload = "auto";
  try {
    await new Promise((resolve, reject) => { video.onloadedmetadata = resolve; video.onerror = reject; });
    for (let index = 0; index < 10; index += 1) {
      if (token !== thumbRenderToken) break;
      const time = entry.duration * ((index + 0.5) / 10);
      await new Promise((resolve) => {
        const done = () => { video.removeEventListener("seeked", done); resolve(); };
        video.addEventListener("seeked", done);
        video.currentTime = Math.min(Math.max(0, time), Math.max(0, entry.duration - 0.05));
      });
      const canvas = document.createElement("canvas");
      canvas.width = 120;
      canvas.height = 80;
      canvas.getContext("2d").drawImage(video, 0, 0, 120, 80);
      elements.timelineThumbs.append(canvas);
    }
  } catch { /* thumbnail preview is optional */ }
  URL.revokeObjectURL(url);
}

function makeNumberControl(text, value, min, max, step, onInput, placeholder = "") {
  const label = document.createElement("label");
  const caption = document.createElement("span");
  caption.textContent = text;
  const input = document.createElement("input");
  input.type = "number";
  input.min = String(min);
  input.max = String(max);
  input.step = String(step);
  input.value = value === 0 ? "0" : value ? String(value) : "";
  input.placeholder = placeholder;
  input.addEventListener("input", () => {
    onInput(input.value);
    resetResult();
    updateEstimate();
    syncSelectedPreview();
  });
  label.append(caption, input);
  return { label, input };
}

function makeSelectControl(text, value, choices, onChange) {
  const label = document.createElement("label");
  const caption = document.createElement("span");
  caption.textContent = text;
  const select = document.createElement("select");
  for (const [optionValue, optionText] of choices) {
    const option = document.createElement("option");
    option.value = optionValue;
    option.textContent = optionText;
    select.append(option);
  }
  select.value = value;
  select.addEventListener("change", () => {
    onChange(select.value);
    resetResult();
    updateEstimate();
    syncSelectedPreview();
  });
  label.append(caption, select);
  return { label, select };
}

function audioTrackLabel(track, fallbackIndex = 0) {
  const number = Number.isInteger(track?.index) ? track.index + 1 : fallbackIndex + 1;
  const details = [];
  const title = String(track?.title || "").trim();
  const language = String(track?.language || "").trim();
  if (title) details.push(title);
  else if (language) details.push(language.toUpperCase());
  if (title && language && !title.toLowerCase().includes(language.toLowerCase())) details.push(language.toUpperCase());
  if (Number(track?.channels) > 0) details.push(Number(track.channels) === 1 ? "mono" : Number(track.channels) === 2 ? "stereo" : `${track.channels} ch`);
  if (track?.default) details.push("default");
  return `Track ${number}${details.length ? ` — ${details.join(" · ")}` : ""}`;
}

function audioTrackChannels(probe, index) {
  const track = probe?.audioTracks?.[Number(index) || 0];
  return Number(track?.channels) || Number(probe?.channels) || 0;
}

function populateAudioTrackSelect(select, entry, loading = false) {
  select.replaceChildren();
  const tracks = entry.audioTracks || [];
  if (loading && !entry.audioTracksKnown) {
    const option = document.createElement("option");
    option.value = String(Number(entry.audioTrack) || 0);
    option.textContent = "Detecting audio tracks…";
    select.append(option); select.disabled = true; return;
  }
  if (entry.audioTracksKnown && tracks.length === 0) {
    const option = document.createElement("option"); option.value = "0"; option.textContent = "No audio tracks"; select.append(option);
    entry.audioTrack = 0; select.disabled = true; return;
  }
  const count = tracks.length || Math.max(1, (Number(entry.audioTrack) || 0) + 1);
  for (let index = 0; index < count; index += 1) {
    const option = document.createElement("option"); option.value = String(index); option.textContent = audioTrackLabel(tracks[index], index); select.append(option);
  }
  let selected = Number(entry.audioTrack) || 0;
  if (tracks.length && (selected < 0 || selected >= tracks.length)) selected = 0;
  entry.audioTrack = selected; select.value = String(selected); select.disabled = conversionRunning;
}

function queueMetadataTask(task) {
  const run = ffmpegMetadataQueue.then(task, task);
  ffmpegMetadataQueue = run.catch(() => {});
  return run;
}

async function ensureEntryAudioTracks(entry, select) {
  if (!entry?.file || entry.audioTracksKnown || entry.audioTrackProbePromise || conversionRunning) {
    populateAudioTrackSelect(select, entry);
    return;
  }
  populateAudioTrackSelect(select, entry, true);
  entry.audioTrackProbePromise = queueMetadataTask(async () => {
    await ensureFFmpeg();
    const inputName = `track-probe-${entry.id.replace(/[^A-Za-z0-9]/g, "").slice(0, 12)}${entry.file.name.match(/\.[A-Za-z0-9]{1,8}$/)?.[0] || ".mp4"}`;
    await ffmpeg.writeFile(inputName, await fetchFile(entry.file));
    try {
      const probe = await readProbe(inputName, `track-${entry.id}`, entry.file);
      entry.duration = probe.duration || entry.duration;
      entry.hasAudio = probe.hasAudio;
      entry.audioTracks = probe.audioTracks || [];
      entry.audioTracksKnown = !probe.audioUnknown;
      entry.channels = audioTrackChannels(probe, entry.audioTrack);
      entry.chapters = probe.chapters || entry.chapters || [];
      if (entry.audioTracksKnown && entry.audioTrack >= entry.audioTracks.length) entry.audioTrack = 0;
    } finally {
      try { await ffmpeg.deleteFile(inputName); } catch { /* already removed */ }
    }
  });
  try { await entry.audioTrackProbePromise; }
  catch (error) { appendLog(`Audio-track detection warning: ${error instanceof Error ? error.message : String(error)}`); }
  finally { entry.audioTrackProbePromise = null; populateAudioTrackSelect(select, entry); }
}

function moveEntry(index, direction) {
  const next = index + direction;
  if (next < 0 || next >= entries.length) return;
  [entries[index], entries[next]] = [entries[next], entries[index]];
  resetResult();
  renderFiles();
}

function renderFiles() {
  elements.fileList.replaceChildren();
  elements.fileArea.hidden = entries.length === 0;
  elements.convertButton.disabled = entries.length === 0 || conversionRunning;
  const totalBytes = entries.reduce((sum, entry) => sum + entry.file.size, 0);
  elements.fileSummary.textContent = `${entries.length} video${entries.length === 1 ? "" : "s"} · ${formatBytes(totalBytes)}`;
  if (entries.length && !entries.some((entry) => entry.id === selectedEntryId)) selectedEntryId = entries[0].id;
  updateOutputModes();

  entries.forEach((entry, index) => {
    const row = document.createElement("div");
    row.className = `file-row${entry.id === selectedEntryId ? " selected" : ""}`;
    row.addEventListener("click", (event) => {
      if (event.target.closest("button, input, select, summary, label")) return;
      selectedEntryId = entry.id;
      renderFiles();
    });

    const fileName = document.createElement("div");
    fileName.className = "file-name";
    const strong = document.createElement("strong");
    strong.textContent = entry.file.name;
    const small = document.createElement("small");
    small.textContent = `${formatBytes(entry.file.size)}${entry.duration ? ` · ${formatClock(entry.duration)}` : " · inspecting…"}`;
    fileName.append(strong, small);
    fileName.addEventListener("click", () => { selectedEntryId = entry.id; renderFiles(); });

    const titleLabel = document.createElement("label");
    const titleText = document.createElement("span");
    titleText.textContent = "Clip-menu title";
    const titleInput = document.createElement("input");
    titleInput.type = "text";
    titleInput.maxLength = 12;
    titleInput.value = entry.title;
    titleInput.addEventListener("input", () => {
      const cleaned = sanitizeMenuTitle(titleInput.value);
      titleInput.value = cleaned;
      entry.title = cleaned || "VIDEO";
      selectedEntryId = entry.id;
      resetResult();
      updateTitlePreview();
      updateEstimate();
    });
    titleLabel.append(titleText, titleInput);

    const moveGroup = document.createElement("div");
    moveGroup.className = "move-buttons";
    const up = document.createElement("button");
    up.type = "button";
    up.className = "secondary compact-button";
    up.textContent = "↑";
    up.title = "Move clip up";
    up.disabled = index === 0;
    up.addEventListener("click", () => moveEntry(index, -1));
    const down = document.createElement("button");
    down.type = "button";
    down.className = "secondary compact-button";
    down.textContent = "↓";
    down.title = "Move clip down";
    down.disabled = index + 1 === entries.length;
    down.addEventListener("click", () => moveEntry(index, 1));
    moveGroup.append(up, down);

    const remove = document.createElement("button");
    remove.className = "icon-button";
    remove.type = "button";
    remove.title = `Remove ${entry.file.name}`;
    remove.setAttribute("aria-label", remove.title);
    remove.textContent = "×";
    remove.addEventListener("click", () => {
      entries = entries.filter((candidate) => candidate.id !== entry.id);
      if (selectedEntryId === entry.id) selectedEntryId = entries[0]?.id || "";
      resetResult();
      renderFiles();
      updateEstimate();
    });

    const details = document.createElement("details");
    details.className = "clip-options";
    const summary = document.createElement("summary");
    summary.textContent = entry.useProject ? "Using project settings" : "Custom clip settings";
    const optionsGrid = document.createElement("div");
    optionsGrid.className = "clip-options-grid";

    const useProjectLabel = document.createElement("label");
    useProjectLabel.className = "clip-use-project";
    const useProject = document.createElement("input");
    useProject.type = "checkbox";
    useProject.checked = entry.useProject;
    const useProjectText = document.createElement("span");
    useProjectText.textContent = "Use project settings for this clip";
    useProject.addEventListener("change", () => {
      entry.useProject = useProject.checked;
      resetResult();
      renderFiles();
      updateEstimate();
    });
    useProjectLabel.append(useProject, useProjectText);
    optionsGrid.append(useProjectLabel);

    const controls = [];
    const start = makeNumberControl("Start (seconds)", entry.start, 0, 86400, 0.1, (value) => { entry.start = numericOr(value, 0); }, "0");
    const end = makeNumberControl("End (blank = full)", entry.end, 0, 86400, 0.1, (value) => { entry.end = numericOr(value, 0); }, "Full video");
    const speed = makeNumberControl("Speed", entry.speed, 0.5, 3, 0.05, (value) => { entry.speed = value === "" ? 1 : numericOr(value, 1); });
    const volume = makeNumberControl("Volume %", Math.round(entry.volume * 100), 0, 200, 5, (value) => { entry.volume = clampNumber(value, 0, 200) / 100; });
    const fit = makeSelectControl("Screen framing", entry.fitMode, [["fit", "Fit with bars"], ["crop", "Crop to fill"], ["stretch", "Stretch"]], (value) => { entry.fitMode = value; });
    const audio = makeSelectControl("Audio channel", entry.audioMode, [["mix", "Mix to mono"], ["left", "Left channel"], ["right", "Right channel"], ["none", "No audio"]], (value) => { entry.audioMode = value; });
    const audioTrack = makeSelectControl("Input audio track", String(Number(entry.audioTrack) || 0), [], (value) => { entry.audioTrack = Number(value) || 0; if (audioPreviewURL) { URL.revokeObjectURL(audioPreviewURL); audioPreviewURL = ""; elements.audioPreviewPlayer.removeAttribute("src"); elements.audioPreviewPlayer.hidden = true; } });
    populateAudioTrackSelect(audioTrack.select, entry);
    const palette = makeSelectControl("Palette", entry.paletteMode, [["shared", "Shared palette"], ["scene", "Per-scene palette"]], (value) => { entry.paletteMode = value; });
    const dither = makeSelectControl("Dithering", entry.ditherMode, [["off", "Off"], ["ordered", "Ordered"], ["error", "Error diffusion"]], (value) => { entry.ditherMode = value; });

    controls.push(start.input, end.input, speed.input, volume.input, fit.select, audio.select, palette.select, dither.select);
    optionsGrid.append(start.label, end.label, speed.label, fit.label, audioTrack.label, audio.label, volume.label, palette.label, dither.label);

    const loopLabel = document.createElement("label");
    loopLabel.className = "clip-loop clip-option-check";
    const loopInput = document.createElement("input");
    loopInput.type = "checkbox";
    loopInput.checked = entry.loop;
    loopInput.addEventListener("change", () => {
      entry.loop = loopInput.checked;
      resetResult();
      updateEstimate();
    });
    const loopText = document.createElement("span");
    loopText.textContent = "Loop playback";
    loopLabel.append(loopInput, loopText);
    optionsGrid.append(loopLabel);
    controls.push(loopInput);

    for (const control of controls) control.disabled = entry.useProject || conversionRunning;
    audioTrack.select.disabled = conversionRunning || (entry.audioTracksKnown && entry.audioTracks.length === 0);
    details.addEventListener("toggle", () => { if (details.open) ensureEntryAudioTracks(entry, audioTrack.select); });
    details.append(summary, optionsGrid);
    row.append(fileName, titleLabel, moveGroup, remove, details);
    elements.fileList.append(row);
  });
  syncSelectedPreview();
  updateEstimate();
}

function resetResult() {
  if (resultURL) URL.revokeObjectURL(resultURL);
  resultURL = "";
  resultFileName = "";
  resultMime = "application/octet-stream";
  elements.resultArea.hidden = true;
}

function updateProgress(percent, message) {
  elements.progressArea.hidden = false;
  elements.progressBar.value = percent;
  elements.progressPercent.textContent = `${Math.round(percent)}%`;
  elements.progressMessage.textContent = message;
}

function appendLog(message) {
  const clean = String(message || "").trim();
  if (!clean) return;
  logLines.push(clean);
  if (logLines.length > 24) logLines = logLines.slice(-24);
  elements.logOutput.textContent = logLines.join("\n");
  elements.logOutput.scrollTop = elements.logOutput.scrollHeight;
}

function currentOptions(includeMenuTheme = true) {
  return {
    preset: elements.preset.value,
    extremeOptimization: elements.preset.value === "extreme",
    audioQuality: elements.preset.value === "extreme" ? elements.audioQuality.value : "pcm",
    adaptiveKeyframes: elements.preset.value === "extreme",
    enhancedSceneDetection: elements.preset.value === "extreme",
    smartTargetMiB: Number(elements.smartTarget?.value || 32),
    smartPriority: elements.smartPriority?.value || "balanced",
    outputMode: elements.outputMode.value,
    vblanks: Number(elements.vblanks.value),
    fitMode: elements.fitMode.value,
    paletteMode: elements.paletteMode.value,
    ditherMode: elements.ditherMode.value,
    compression: elements.compression.value,
    audioMode: elements.audioMode.value,
    seekSeconds: Number(elements.seekSeconds.value),
    defaultStart: clampNumber(elements.defaultStart.value, 0, 86400),
    defaultEnd: Math.max(0, numericOr(elements.defaultEnd.value, 0)),
    defaultSpeed: clampNumber(elements.defaultSpeed.value, 0.5, 3),
    defaultVolume: clampNumber(elements.defaultVolume.value, 0, 200) / 100,
    defaultLoop: elements.defaultLoop.checked,
    romTitle: elements.romTitle.value || "GBA VIDEO",
    normalize: elements.normalize.checked,
    limiter: elements.limiter.checked,
    resume: elements.resume.checked,
    splitVideo: elements.splitVideo.checked,
    splitBudgetMiB: Number(elements.splitBudget.value),
    maxPartSeconds: parseClock(elements.maxPartDuration.value),
    chapterAware: elements.chapterAware.checked,
    partTitleScreens: elements.partTitleScreens.checked,
    titleCards: serializedTitleCards(),
    resumeLongSplit: elements.resumeLongSplit.checked,
    menuBackground: elements.menuBackground?.value || "ocean-wave-animated",
    menuUIColor: elements.menuUIColor?.value || "#FFFFFF",
    menuSelectionColor: elements.menuSelectionColor?.value || "#FFDE00",
    menuOutline: Boolean(elements.menuOutline?.checked),
    menuOutlineColor: elements.menuOutlineColor?.value || "#000000",
    menuTheme: elements.outputMode.value === "menu" ? serializedMenuTheme() : null,
  };
}

function effectiveClipOptions(entry, project) {
  if (!entry.useProject) {
    return {
      start: entry.start, end: entry.end, speed: entry.speed,
      fitMode: entry.fitMode, audioMode: entry.audioMode, audioTrack: Number(entry.audioTrack) || 0, volume: entry.volume,
      loop: entry.loop, paletteMode: entry.paletteMode, ditherMode: entry.ditherMode,
    };
  }
  return {
    start: project.defaultStart, end: project.defaultEnd, speed: project.defaultSpeed,
    fitMode: project.fitMode, audioMode: project.audioMode, audioTrack: Number(entry.audioTrack) || 0, volume: project.defaultVolume,
    loop: project.defaultLoop, paletteMode: project.paletteMode, ditherMode: project.ditherMode,
  };
}

function applyPreset(name) {
  const preset = PRESETS[name];
  if (!preset) return;
  elements.vblanks.value = preset.vblanks;
  elements.compression.value = preset.compression;
  elements.normalize.checked = preset.normalize;
  elements.limiter.checked = preset.limiter;
  elements.fitMode.value = preset.fitMode;
  elements.audioMode.value = preset.audioMode;
  elements.paletteMode.value = preset.paletteMode;
  elements.ditherMode.value = preset.ditherMode;
  elements.audioQuality.value = preset.audioQuality || "pcm";
  smartAnalysis = null;
  updateExtremeVisibility();
  resetResult();
  updateEstimate();
  syncSelectedPreview();
}

function markPresetCustom() {
  if (elements.preset.value === "extreme") {
    smartAnalysis = null;
    elements.smartStatus.textContent = "Settings changed — analyze again";
    elements.smartResults.hidden = true;
  } else {
    elements.preset.value = "custom";
    updateExtremeVisibility();
  }
  resetResult();
}

function updateExtremeVisibility() {
  const extreme = elements.preset.value === "extreme";
  elements.extremeSection.hidden = !extreme;
  elements.audioQuality.disabled = !extreme || conversionRunning || smartAnalysisRunning;
  if (!extreme) {
    elements.audioQuality.value = "pcm";
    smartAnalysis = null;
    elements.smartResults.hidden = true;
    elements.smartStatus.textContent = "Not analyzed";
  }
}

function formatSmartBytes(bytes) {
  return `${(Number(bytes || 0) / 1048576).toFixed(1)} MiB`;
}

function applySmartCandidate(candidate) {
  if (!candidate) return;
  elements.preset.value = "extreme";
  elements.vblanks.value = String(candidate.vblanks);
  elements.paletteMode.value = candidate.paletteMode;
  elements.ditherMode.value = candidate.ditherMode;
  elements.compression.value = "delta";
  elements.audioQuality.value = candidate.audioCodec || "pcm";
  smartAnalysis = { ...(smartAnalysis || {}), appliedCandidate: candidate };
  elements.smartStatus.textContent = `${candidate.label} settings applied`;
  updateExtremeVisibility();
  resetResult();
  updateEstimate();
}

function renderSmartResults(result) {
  elements.smartResults.replaceChildren();
  const candidates = [result.recommended, ...(result.alternatives || [])].filter(Boolean);
  for (const [index, candidate] of candidates.entries()) {
    const card = document.createElement("article");
    card.className = `smart-result-card${index === 0 ? " recommended" : ""}`;
    const title = document.createElement("h4");
    title.textContent = index === 0 ? `Recommended — ${candidate.label}` : candidate.label;
    const summary = document.createElement("p");
    summary.textContent = candidate.summary || "Alternative encoding candidate.";
    const metrics = document.createElement("div");
    metrics.className = "smart-result-metrics";
    metrics.textContent = `Estimated ${formatSmartBytes(candidate.estimatedMinBytes)}–${formatSmartBytes(candidate.estimatedMaxBytes)} · Visual ${candidate.visualQuality}/100 · Motion ${candidate.motionQuality}/100 · ${candidate.fps.toFixed(2)} FPS · ${candidate.audioCodec === "adpcm" ? "Compact ADPCM" : "Standard PCM"}`;
    const button = document.createElement("button");
    button.type = "button";
    button.className = "secondary";
    button.textContent = "Apply settings";
    button.addEventListener("click", () => applySmartCandidate(candidate));
    card.append(title, summary, metrics, button);
    elements.smartResults.append(card);
  }
  elements.smartResults.hidden = false;
}

async function runSmartAnalysis() {
  if (smartAnalysisRunning || conversionRunning) return;
  if (elements.preset.value !== "extreme") return;
  if (entries.length !== 1) { alert("Extreme analysis currently requires exactly one source video."); return; }
  smartAnalysisRunning = true;
  smartAnalysisCancelled = false;
  elements.smartAnalyze.disabled = true;
  elements.smartCancel.hidden = false;
  elements.smartStatus.textContent = "Loading analysis engine…";
  updateExtremeVisibility();
  let inputName = "";
  const scanName = "smart-scan.rgb";
  try {
    await ensureFFmpeg();
    const entry = entries[0];
    inputName = "smart-input" + (entry.file.name.match(/\.[A-Za-z0-9]{1,8}$/)?.[0] || ".mp4");
    await ffmpeg.writeFile(inputName, await fetchFile(entry.file));
    const probe = await readProbe(inputName, 997, entry.file);
    if (smartAnalysisCancelled) throw new Error("Analysis cancelled.");
    const project = currentOptions(false);
    const clip = effectiveClipOptions(entry, project);
    const start = Math.max(0, clip.start || 0);
    const sourceEnd = clip.end > start ? Math.min(clip.end, probe.duration) : probe.duration;
    const duration = Math.max(0.25, (sourceEnd - start) / Math.max(0.5, clip.speed || 1));
    const scanFrames = duration < 60 ? Math.max(18, Math.ceil(duration * 2)) : 120;
    const scanFPS = Math.min(2, scanFrames / Math.max(0.25, sourceEnd - start));
    elements.smartStatus.textContent = "Scanning motion, detail and scene changes…";
    const args = ["-hide_banner", "-loglevel", "error", "-ss", start.toFixed(6), "-i", inputName, "-t", Math.max(0.25, sourceEnd - start).toFixed(6), "-vf", `fps=${scanFPS.toFixed(8)},scale=120:80:flags=area`, "-pix_fmt", "rgb24", "-f", "rawvideo", scanName];
    const code = await ffmpeg.exec(args);
    if (code !== 0) throw new Error("FFmpeg could not create the representative-scene scan.");
    if (smartAnalysisCancelled) throw new Error("Analysis cancelled.");
    const framesRGB = await ffmpeg.readFile(scanName);
    elements.smartStatus.textContent = "Comparing encoding candidates…";
    const result = analyzeSmartScan({
      framesRGB, duration, hasAudio: probe.hasAudio && clip.audioMode !== "none",
      targetBytes: Number(elements.smartTarget.value) * 1048576,
      priority: elements.smartPriority.value, audioQuality: elements.audioQuality.value,
    });
    smartAnalysis = result;
    renderSmartResults(result);
    elements.smartStatus.textContent = `${result.samples.length} representative samples analyzed · ${result.confidence} confidence`;
  } catch (error) {
    elements.smartStatus.textContent = error instanceof Error ? error.message : String(error);
    if (!smartAnalysisCancelled) alert(elements.smartStatus.textContent);
  } finally {
    if (ffmpeg) {
      if (scanName) await ffmpeg.deleteFile(scanName).catch(() => {});
      if (inputName) await ffmpeg.deleteFile(inputName).catch(() => {});
    }
    smartAnalysisRunning = false;
    elements.smartAnalyze.disabled = false;
    elements.smartCancel.hidden = true;
    updateExtremeVisibility();
  }
}

function updateConvertButton() {
  elements.convertButton.textContent = "Create output";
}

function setBusy(busy) {
  conversionRunning = busy;
  elements.convertButton.disabled = busy || entries.length === 0;
  elements.cancelButton.hidden = !busy;
  elements.fileInput.disabled = busy;
  elements.clearButton.disabled = busy;
  const settings = [
    elements.preset, elements.outputMode, elements.vblanks, elements.fitMode,
    elements.paletteMode, elements.ditherMode, elements.compression, elements.audioMode, elements.audioQuality,
    elements.smartTarget, elements.smartPriority, elements.smartAnalyze, elements.smartCancel,
    elements.seekSeconds, elements.defaultStart, elements.defaultEnd, elements.defaultSpeed,
    elements.defaultVolume, elements.defaultLoop, elements.romTitle, elements.normalize,
    elements.limiter, elements.resume, elements.splitVideo, elements.splitBudget,
    elements.maxPartDuration, elements.chapterAware, elements.partTitleScreens, elements.resumeLongSplit,
    elements.saveProjectButton, elements.openProjectInput, elements.optimizerButton,
    elements.timelinePlay, elements.timelineStart, elements.timelineEnd, elements.audioPreviewButton,
    elements.titlePreviewInput, elements.menuBackground, elements.customMenuBackground,
    elements.clearCustomMenuBackground, elements.menuUIColor, elements.menuSelectionColor, elements.menuOutline, elements.menuOutlineColor,
    elements.titleCardPrev, elements.titleCardNext, elements.titleCardPartSelect, elements.titleCardUseShared,
    elements.titleCardCopyToAll, elements.titleCardBackground, elements.titleCardDarkness, elements.titleCardFrameOffset,
    elements.titleCardSolidColor, elements.titleCardTitle, elements.titleCardSubtitle, elements.titleCardAlignment, elements.titleCardTextSize,
    elements.titleCardTextColor, elements.titleCardOutline, elements.titleCardOutlineColor, elements.titleCardStartMode,
    elements.titleCardDuration, elements.titleCardAllowSkip, elements.titleCardFade,
  ];
  for (const control of settings) if (control) control.disabled = busy;
  for (const control of elements.fileList.querySelectorAll("input, select, button")) control.disabled = busy;
  elements.optimizerButton.disabled = busy || !entries.length;
  if (!elements.titleCardGroup.hidden && lastEstimate) updateTitleCardVisibility(lastEstimate);
  updateExtremeVisibility();
}

function downloadResult() {
  if (!resultURL) return;
  const link = document.createElement("a");
  link.href = resultURL;
  link.download = resultFileName;
  document.body.append(link);
  link.click();
  link.remove();
}

function safeVirtualName(index, originalName) {
  const match = String(originalName).match(/\.[A-Za-z0-9]{1,8}$/);
  return `input-${index}${match ? match[0].toLowerCase() : ".mp4"}`;
}

async function ensureFFmpeg() {
  if (ffmpeg?.loaded) return;
  updateProgress(1, "Loading the browser video engine…");
  ffmpeg = new FFmpeg();
  ffmpeg.on("log", ({ message }) => {
    const clean = String(message || "").trim();
    if (!clean) return;
    recentFFmpegLogs.push(clean);
    if (recentFFmpegLogs.length > 60) recentFFmpegLogs = recentFFmpegLogs.slice(-60);
    appendLog(clean);
  });
  await ffmpeg.load({
    coreURL: await toBlobURL(`${FFMPEG_CORE_BASE}/ffmpeg-core.js`, "text/javascript"),
    wasmURL: await toBlobURL(`${FFMPEG_CORE_BASE}/ffmpeg-core.wasm`, "application/wasm"),
  });
  updateProgress(5, "Video engine ready.");
}

function getRomWorker() {
  if (romWorker) return romWorker;
  romWorker = new Worker(new URL("./rom.worker.js", import.meta.url), { type: "module" });
  romWorker.addEventListener("message", (event) => {
    const message = event.data || {};
    const task = romTasks.get(message.id);
    if (!task) return;
    if (message.type === "progress") {
      task.onProgress?.(message.fraction, message.message);
      return;
    }
    romTasks.delete(message.id);
    if (message.type === "error") task.reject(new Error(message.message));
    else if (message.type === "clip") task.resolve(message.clip);
    else if (message.type === "rom") task.resolve({ buffer: message.buffer, details: message.details });
  });
  romWorker.addEventListener("error", (event) => {
    for (const task of romTasks.values()) task.reject(new Error(event.message || "ROM worker crashed."));
    romTasks.clear();
    romWorker?.terminate();
    romWorker = null;
  });
  return romWorker;
}

function runRomTask(action, payload, transfer = [], onProgress) {
  const id = ++romTaskCounter;
  return new Promise((resolve, reject) => {
    romTasks.set(id, { resolve, reject, onProgress });
    getRomWorker().postMessage({ id, action, payload }, transfer);
  });
}

function readBrowserDuration(file) {
  return new Promise((resolve) => {
    const video = document.createElement("video");
    const url = URL.createObjectURL(file);
    const finish = (value) => {
      URL.revokeObjectURL(url);
      video.removeAttribute("src");
      video.load();
      resolve(value);
    };
    const timer = setTimeout(() => finish(0), 8000);
    video.preload = "metadata";
    video.muted = true;
    video.onloadedmetadata = () => {
      clearTimeout(timer);
      finish(Number.isFinite(video.duration) ? video.duration : 0);
    };
    video.onerror = () => {
      clearTimeout(timer);
      finish(0);
    };
    video.src = url;
  });
}

function decodeText(data) {
  return typeof data === "string" ? data : new TextDecoder().decode(data);
}

async function readProbe(inputName, index, file) {
  const outputName = `probe-${index}.json`;
  const browserDurationPromise = readBrowserDuration(file);
  try { await ffmpeg.deleteFile(outputName); } catch { /* absent */ }

  try {
    const exitCode = await ffmpeg.ffprobe([
      "-v", "error",
      "-show_entries", "format=duration:stream=index,codec_type,codec_name,channels:stream_tags=language,title:chapter=start_time,end_time",
      "-of", "json",
      inputName,
      "-o", outputName,
    ]);
    if (exitCode === 0) {
      const raw = await ffmpeg.readFile(outputName, "utf8");
      const probe = JSON.parse(decodeText(raw));
      const duration = Number(probe?.format?.duration);
      if (Number.isFinite(duration) && duration > 0) {
        const audioTracks = (Array.isArray(probe.streams) ? probe.streams : [])
          .filter((stream) => stream.codec_type === "audio")
          .map((stream, audioIndex) => ({
            index: audioIndex, streamIndex: Number(stream.index) || 0, codec: String(stream.codec_name || ""),
            channels: Number(stream.channels) || 0, language: String(stream.tags?.language || ""), title: String(stream.tags?.title || ""),
          }));
        const audioStream = audioTracks[0] || null;
        const chapters = Array.isArray(probe.chapters)
          ? probe.chapters.map((chapter) => Number(chapter.start_time)).filter((value) => Number.isFinite(value) && value > 0).sort((a, b) => a - b)
          : [];
        return { duration, hasAudio: Boolean(audioStream), channels: Number(audioStream?.channels) || 0, audioTracks, audioUnknown: false, chapters };
      }
    }
  } catch (error) {
    appendLog(`ffprobe warning: ${error instanceof Error ? error.message : String(error)}`);
  } finally {
    try { await ffmpeg.deleteFile(outputName); } catch { /* absent */ }
  }

  const browserDuration = await browserDurationPromise;
  if (browserDuration > 0) {
    appendLog("ffprobe could not inspect this file; using browser metadata and testing audio during conversion.");
    return { duration: browserDuration, hasAudio: true, channels: 0, audioTracks: [], audioUnknown: true, chapters: [] };
  }

  const detail = recentFFmpegLogs.slice(-3).join(" | ");
  throw new Error(`Could not inspect the selected video.${detail ? ` FFmpeg: ${detail}` : ""}`);
}

function clipTiming(clipOptions, sourceDuration) {
  const start = clampNumber(clipOptions.start, 0, Math.max(0, sourceDuration - 0.01));
  const requestedEnd = numericOr(clipOptions.end, 0);
  const end = requestedEnd > start ? Math.min(requestedEnd, sourceDuration) : sourceDuration;
  const speed = clampNumber(clipOptions.speed, 0.5, 3);
  const volume = clampNumber(clipOptions.volume, 0, 2);
  if (end <= start) throw new Error("End time must be after start time.");
  return { start, end, sourceDuration: end - start, outputDuration: (end - start) / speed, speed, volume };
}

function trimArguments(timing) {
  const args = [];
  if (timing.start > 0.0001) args.push("-ss", timing.start.toFixed(3));
  if (timing.sourceDuration > 0.0001) args.push("-t", timing.sourceDuration.toFixed(3));
  return args;
}

function atempoFilter(speed) {
  const factors = [];
  let remaining = speed;
  while (remaining > 2.0001) { factors.push(2); remaining /= 2; }
  while (remaining < 0.4999) { factors.push(0.5); remaining /= 0.5; }
  factors.push(remaining);
  return factors.map((factor) => `atempo=${factor.toFixed(8)}`).join(",");
}

function videoFilter(fitMode, vblanks, speed = 1) {
  const fps = GBA_REFRESH / vblanks;
  let scale;
  if (fitMode === "crop") scale = "scale=120:80:force_original_aspect_ratio=increase,crop=120:80";
  else if (fitMode === "stretch") scale = "scale=120:80";
  else scale = "scale=120:80:force_original_aspect_ratio=decrease,pad=120:80:(ow-iw)/2:(oh-ih)/2:black";
  return `${scale},setpts=PTS/${speed.toFixed(8)},fps=${fps.toFixed(10)},format=rgb24`;
}


function titleCardVideoFilter(fitMode) {
  if (fitMode === "crop") return "scale=240:160:force_original_aspect_ratio=increase,crop=240:160,format=rgb24";
  if (fitMode === "stretch") return "scale=240:160,format=rgb24";
  return "scale=240:160:force_original_aspect_ratio=decrease,pad=240:160:(ow-iw)/2:(oh-ih)/2:black,format=rgb24";
}
function solidTitleCardRGB(hex) {
  const match = /^#?([0-9a-f]{6})$/i.exec(String(hex || "#000000"));
  const value = Number.parseInt(match?.[1] || "000000", 16);
  const rgb = new Uint8Array(240 * 160 * 3);
  const r = value >>> 16, g = (value >>> 8) & 255, b = value & 255;
  for (let index = 0; index < 240 * 160; index += 1) { rgb[index * 3] = r; rgb[index * 3 + 1] = g; rgb[index * 3 + 2] = b; }
  return rgb;
}
async function extractTitleCardFrame(inputName, index, fitMode, start, end, settings) {
  if (settings.backgroundMode === "solid") return solidTitleCardRGB(settings.solidColor);
  const outputName = `title-card-${index}.rgb`;
  let when = start + (settings.backgroundMode === "part-frame" ? Math.max(0, Number(settings.frameOffsetSeconds) || 0) : 0);
  if (end > start && when >= end) when = Math.max(start, end - 0.04);
  const exitCode = await ffmpeg.exec([
    "-hide_banner", "-loglevel", "error", "-i", inputName,
    "-ss", when.toFixed(6), "-frames:v", "1", "-an", "-vf", titleCardVideoFilter(fitMode),
    "-pix_fmt", "rgb24", "-f", "rawvideo", outputName,
  ]);
  if (exitCode !== 0) throw new Error(`FFmpeg could not extract the title-card background. ${recentFFmpegLogs.slice(-1)[0] || ""}`.trim());
  const frame = await ffmpeg.readFile(outputName);
  await ffmpeg.deleteFile(outputName);
  if (!(frame instanceof Uint8Array) || frame.length !== 240 * 160 * 3) throw new Error("The title-card background frame is incomplete.");
  return frame;
}

async function extractFrames(inputName, index, project, clipOptions, timing) {
  const outputName = `frames-${index}.rgb`;
  const estimatedFrames = Math.max(1, Math.ceil(timing.outputDuration * (GBA_REFRESH / project.vblanks)));
  const estimatedBytes = estimatedFrames * RGB_FRAME_BYTES;
  if (estimatedBytes > MAX_RAW_FRAME_BYTES) {
    throw new Error(`This clip would need about ${(estimatedBytes / 1048576).toFixed(0)} MiB of raw browser memory. Use a shorter clip, a lower frame rate, or the desktop app.`);
  }
  const exitCode = await ffmpeg.exec([
    "-hide_banner", "-loglevel", "error", "-i", inputName,
    ...trimArguments(timing),
    "-an", "-vf", videoFilter(clipOptions.fitMode, project.vblanks, timing.speed),
    "-pix_fmt", "rgb24", "-f", "rawvideo", outputName,
  ]);
  if (exitCode !== 0) throw new Error(`FFmpeg could not decode the video frames. ${recentFFmpegLogs.slice(-1)[0] || ""}`.trim());
  const frames = await ffmpeg.readFile(outputName);
  await ffmpeg.deleteFile(outputName);
  if (!(frames instanceof Uint8Array) || frames.length < RGB_FRAME_BYTES) throw new Error("The converted video contains no usable frames.");
  return frames;
}

async function extractAudio(inputName, index, project, clipOptions, probe, timing) {
  if (clipOptions.audioMode === "none" || !probe.hasAudio) return new Uint8Array();
  const outputName = `audio-${index}.s8`;
  if (probe.audioTracks?.length && (clipOptions.audioTrack < 0 || clipOptions.audioTrack >= probe.audioTracks.length)) {
    throw new Error("The selected input audio track is not available in this file.");
  }
  const filters = [];
  if (clipOptions.audioMode === "left") filters.push("pan=mono|c0=c0");
  if (clipOptions.audioMode === "right") filters.push(audioTrackChannels(probe, clipOptions.audioTrack) === 1 ? "pan=mono|c0=c0" : "pan=mono|c0=c1");
  filters.push(`aresample=${AUDIO_RATE}:async=1:first_pts=0`);
  filters.push(atempoFilter(timing.speed));
  if (Math.abs(timing.volume - 1) > 0.000001) filters.push(`volume=${timing.volume.toFixed(6)}`);
  if (project.normalize) filters.push("loudnorm=I=-16:LRA=11:TP=-1.5");
  if (project.limiter) filters.push("alimiter=limit=0.95:attack=5:release=50");

  const exitCode = await ffmpeg.exec([
    "-hide_banner", "-loglevel", "error", "-i", inputName,
    ...trimArguments(timing),
    "-map", `0:a:${Number(clipOptions.audioTrack) || 0}`, "-vn", "-af", filters.join(","),
    "-ac", "1", "-ar", String(AUDIO_RATE), "-f", "s8", outputName,
  ]);
  if (exitCode !== 0) {
    try { await ffmpeg.deleteFile(outputName); } catch { /* absent */ }
    if (probe.audioUnknown) {
      appendLog("Audio stream was not available or could not be decoded; continuing without audio.");
      return new Uint8Array();
    }
    throw new Error(`FFmpeg could not decode the audio stream. ${recentFFmpegLogs.slice(-1)[0] || ""}`.trim());
  }
  const audio = await ffmpeg.readFile(outputName);
  await ffmpeg.deleteFile(outputName);
  return audio instanceof Uint8Array ? audio : new Uint8Array();
}

function clipTransferList(clip) {
  return [clip.palette.buffer, clip.paletteIndex.buffer, clip.videoIndex.buffer, clip.video.buffer, clip.audio.buffer];
}



function splitPartRomTitle(base, part) {
  const suffix = String(part).padStart(2, "0");
  return `${String(base || "VIDEO").slice(0, 9)}${suffix}`.slice(0, 12);
}

function recoveryFingerprint(entry, project, start, end) {
  const settings = [
    entry.file.name, entry.file.size, entry.file.lastModified, start, end,
    project.vblanks, project.fitMode, project.audioMode, Number(entry.audioTrack) || 0, project.defaultVolume,
    project.normalize, project.limiter, project.compression, project.paletteMode,
    project.ditherMode, project.splitBudgetMiB, project.maxPartSeconds,
    project.chapterAware, project.partTitleScreens, JSON.stringify(project.titleCards || null),
  ];
  let hash = 2166136261;
  for (const character of JSON.stringify(settings)) {
    hash ^= character.charCodeAt(0);
    hash = Math.imul(hash, 16777619);
  }
  return `split-${(hash >>> 0).toString(16)}`;
}

function openRecoveryDB() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open("gba-video-maker", 1);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains("split-jobs")) db.createObjectStore("split-jobs", { keyPath: "key" });
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error("Could not open browser recovery storage."));
  });
}

async function recoveryGet(key) {
  const db = await openRecoveryDB();
  try {
    return await new Promise((resolve, reject) => {
      const request = db.transaction("split-jobs", "readonly").objectStore("split-jobs").get(key);
      request.onsuccess = () => resolve(request.result || null);
      request.onerror = () => reject(request.error);
    });
  } finally { db.close(); }
}

async function recoveryPut(value) {
  const db = await openRecoveryDB();
  try {
    await new Promise((resolve, reject) => {
      const request = db.transaction("split-jobs", "readwrite").objectStore("split-jobs").put(value);
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  } finally { db.close(); }
}

async function recoveryDelete(key) {
  const db = await openRecoveryDB();
  try {
    await new Promise((resolve, reject) => {
      const request = db.transaction("split-jobs", "readwrite").objectStore("split-jobs").delete(key);
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
  } finally { db.close(); }
}

async function encodeLoadedRange({ inputName, entry, probe, project, start, end, part, playerStub, progressBase = 0, progressSpan = 1 }) {
  const clipOptions = effectiveClipOptions(entry, project);
  const rangeOptions = { ...clipOptions, start, end, loop: false };
  const timing = clipTiming(rangeOptions, probe.duration);
  const mapped = (fraction, message) => updateProgress(progressBase + fraction * progressSpan, message);
  mapped(0.02, `Part ${part}: extracting 120×80 frames…`);
  const framesRGB = await extractFrames(inputName, `part-${part}`, project, rangeOptions, timing);
  mapped(0.30, probe.hasAudio && rangeOptions.audioMode !== "none" ? `Part ${part}: extracting audio…` : `Part ${part}: no audio selected.`);
  const audio = await extractAudio(inputName, `part-${part}`, project, rangeOptions, probe, timing);
  mapped(0.38, `Part ${part}: encoding GBA video…`);
  const clip = await runRomTask("encodeClip", {
    framesRGB,
    audio,
    title: entry.title || "VIDEO",
    vblanks: project.vblanks,
    paletteMode: rangeOptions.paletteMode,
    ditherMode: rangeOptions.ditherMode,
    compression: project.compression,
    keyInterval: 30,
    adaptiveKeyframes: Boolean(project.extremeOptimization && project.adaptiveKeyframes),
    enhancedSceneDetection: Boolean(project.extremeOptimization && project.enhancedSceneDetection),
    audioCodec: project.extremeOptimization ? (project.audioQuality === "auto" ? (smartAnalysis?.recommended?.audioCodec || "pcm") : project.audioQuality) : "pcm",
    seekSeconds: project.seekSeconds,
    loop: false,
  }, [framesRGB.buffer, audio.buffer], (fraction, message) => mapped(0.38 + fraction * 0.54, `Part ${part}: ${message}`));
  let titleCard = null;
  if (project.partTitleScreens) {
    const settings = resolveTitleCardSettings(project.titleCards, entry.file.name, part) || normalizeTitleCardSettings({}, entry.file.name, part);
    mapped(0.92, `Part ${part}: rendering native 240×160 title card…`);
    const background = await extractTitleCardFrame(inputName, `part-${part}`, rangeOptions.fitMode, start, end, settings);
    titleCard = buildTitleCardAsset(background, settings, part, entry.file.name);
  }
  const stub = playerStub.slice();
  mapped(0.95, `Part ${part}: assembling ROM…`);
  const assembled = await runRomTask("assembleROM", {
    playerStub: stub,
    clips: [clip],
    options: {
      romTitle: splitPartRomTitle(project.romTitle, part),
      outputMode: "rom",
      resume: project.resume,
      titleScreenPart: project.partTitleScreens ? part : 0,
      titleScreenName: project.partTitleScreens ? entry.file.name.replace(/\.[^.]+$/, "") : "",
      titleCard,
    },
  }, [stub.buffer, ...clipTransferList(clip), ...(titleCard ? [titleCard.buffer] : [])]);
  return { ...assembled, start, end };
}

function splitManifest(parts, sourceName) {
  const lines = [`Source: ${sourceName}`, "", "Part files:"];
  for (const part of parts) {
    lines.push(`${part.name}  ${formatClock(part.start)} - ${formatClock(part.end)}  ${formatBytes(part.unpaddedSize)} data  ${formatBytes(part.data.byteLength)} cartridge`);
  }
  return new TextEncoder().encode(lines.join("\n") + "\n");
}

function splitZipResult(parts, entry, estimatedParts, partial = false) {
  const files = parts.map((part) => ({ name: part.name, data: new Uint8Array(part.data) }));
  files.push({ name: "PARTS.txt", data: splitManifest(parts, entry.file.name) });
  const zip = buildStoredZip(files);
  const base = cleanFileBase(entry.file.name.replace(/\.[^.]+$/, ""), "MY_VIDEO");
  return {
    buffer: zip.buffer,
    fileName: `${base}_PARTS.zip`,
    mime: "application/zip",
    details: {
      clipCount: parts.length,
      frameCount: parts.reduce((sum, part) => sum + (part.frameCount || 0), 0),
      paddedSize: parts.reduce((sum, part) => sum + part.data.byteLength, 0),
      outputKind: "zip",
      estimatedParts,
      partial,
    },
  };
}

async function performLongSplit(playerStub, project) {
  if (entries.length !== 1) throw new Error("Video splitting requires exactly one source video.");
  const entry = entries[0];
  const inputName = safeVirtualName(0, entry.file.name);
  await ffmpeg.writeFile(inputName, await fetchFile(entry.file));
  let recovery = null;
  try {
    const probe = await readProbe(inputName, 0, entry.file);
    entry.duration = probe.duration;
    entry.hasAudio = probe.hasAudio;
    entry.audioTracks = probe.audioTracks || [];
    entry.audioTracksKnown = !probe.audioUnknown;
    if (entry.hasAudio && entry.audioTracksKnown && entry.audioTrack >= entry.audioTracks.length) throw new Error("The selected input audio track is not available in this file.");
    entry.channels = audioTrackChannels(probe, entry.audioTrack);
    entry.chapters = probe.chapters || [];
    const clipOptions = effectiveClipOptions(entry, project);
    const whole = clipTiming(clipOptions, probe.duration);
    const start = whole.start;
    const finalEnd = whole.end;
    const totalSourceDuration = finalEnd - start;
    const budget = Math.max(1, Math.min(32, project.splitBudgetMiB || 32)) * 1048576;
    let estimatedParts = Math.max(2, estimateProject(project).parts);
    const key = recoveryFingerprint(entry, project, start, finalEnd);
    let parts = [];
    let cursor = start;
    let partNumber = 1;
    if (project.resumeLongSplit) {
      recovery = await recoveryGet(key);
      if (recovery?.parts?.length && recovery.sourceName === entry.file.name) {
        parts = recovery.parts;
        cursor = recovery.cursor;
        partNumber = recovery.nextPart;
        appendLog(`Resuming after ${parts.length} completed part${parts.length === 1 ? "" : "s"}.`);
      } else recovery = { key, sourceName: entry.file.name, cursor, nextPart: partNumber, parts: [] };
    }

    let guess = Math.min(project.maxPartSeconds > 0 ? project.maxPartSeconds : 480, finalEnd - cursor);
    if (parts.length) {
      const last = parts[parts.length - 1];
      guess = Math.max(2, (last.end - last.start) * (budget / Math.max(1, last.unpaddedSize)) * 0.94);
    } else if (lastEstimate?.totalDuration > 0 && lastEstimate?.totalBytes > 0) {
      guess = Math.min(guess, Math.max(2, totalSourceDuration * budget / lastEstimate.totalBytes * 0.92));
    }
    if (project.maxPartSeconds > 0) guess = Math.min(guess, project.maxPartSeconds);

    while (cursor < finalEnd - 0.001) {
      if (conversionCancelled) throw new Error("Conversion cancelled.");
      const overallFraction = (cursor - start) / totalSourceDuration;
      const progressPrefix = `Part ${partNumber} of approximately ${Math.max(estimatedParts, partNumber)}\nSource position: ${formatClock(cursor - start)} / ${formatClock(totalSourceDuration)}`;
      updateProgress(Math.min(98, overallFraction * 98), `${progressPrefix}\nPreparing candidate…`);
      let candidateSeconds = Math.min(guess, finalEnd - cursor);
      if (project.maxPartSeconds > 0) candidateSeconds = Math.min(candidateSeconds, project.maxPartSeconds);
      let accepted = null;

      for (let attempt = 1; attempt <= 7; attempt += 1) {
        if (conversionCancelled) throw new Error("Conversion cancelled.");
        let candidateEnd = Math.min(finalEnd, cursor + Math.max(1, candidateSeconds));
        if (project.chapterAware) candidateEnd = chooseChapterEnd(probe.chapters, cursor, candidateEnd, finalEnd);
        if (candidateEnd <= cursor + 0.05) candidateEnd = Math.min(finalEnd, cursor + Math.max(1, candidateSeconds));
        const partProgressBase = ((cursor - start) / totalSourceDuration) * 96;
        const partProgressSpan = Math.max(1, ((candidateEnd - cursor) / totalSourceDuration) * 96);
        try {
          const result = await encodeLoadedRange({ inputName, entry, probe, project, start: cursor, end: candidateEnd, part: partNumber, playerStub, progressBase: partProgressBase, progressSpan: partProgressSpan });
          const size = result.details.unpaddedSize;
          if (size > budget) {
            candidateSeconds *= Math.max(0.2, Math.min(0.9, budget / size * 0.92));
            appendLog(`Part ${partNumber} candidate used ${formatBytes(size)}; retrying with a shorter range.`);
            continue;
          }
          const canExtend = project.maxPartSeconds <= 0 && candidateEnd < finalEnd - 0.001 && size < budget * 0.72 && attempt < 4;
          if (canExtend) {
            candidateSeconds *= Math.min(1.55, budget / Math.max(1, size) * 0.88);
            appendLog(`Part ${partNumber} candidate used ${formatBytes(size)}; extending to use more cartridge space.`);
            continue;
          }
          accepted = result;
          break;
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          if (/32 MiB|cartridge size|raw browser memory|exceed/i.test(message) && candidateSeconds > 2) {
            candidateSeconds *= 0.62;
            appendLog(`Part ${partNumber} was too large; retrying a shorter range.`);
            continue;
          }
          throw error;
        }
      }
      if (!accepted) throw new Error(`Could not fit part ${partNumber} within the selected ROM-size target.`);

      const base = cleanFileBase(entry.file.name.replace(/\.[^.]+$/, ""), "MY_VIDEO");
      const name = `${base}_PART_${String(partNumber).padStart(2, "0")}.gba`;
      const data = accepted.buffer.slice(0);
      parts.push({
        name,
        start: cursor,
        end: accepted.end,
        frameCount: accepted.details.frameCount,
        unpaddedSize: accepted.details.unpaddedSize,
        data,
      });
      cursor = accepted.end;
      partNumber += 1;
      const acceptedDuration = Math.max(0.001, accepted.end - accepted.start);
      const bytesPerSourceSecond = accepted.details.unpaddedSize / acceptedDuration;
      const remainingParts = Math.ceil(Math.max(0, finalEnd - cursor) * bytesPerSourceSecond / budget);
      estimatedParts = Math.max(parts.length, parts.length + remainingParts);
      guess = acceptedDuration;
      if (accepted.details.unpaddedSize > 0) guess *= budget / accepted.details.unpaddedSize * 0.94;
      if (project.maxPartSeconds > 0) guess = Math.min(guess, project.maxPartSeconds);

      if (project.resumeLongSplit) {
        recovery = { key, sourceName: entry.file.name, cursor, nextPart: partNumber, parts };
        try { await recoveryPut(recovery); } catch (error) { appendLog(`Recovery warning: ${error instanceof Error ? error.message : String(error)}`); }
      }
    }

    if (project.resumeLongSplit) await recoveryDelete(key).catch(() => {});
    updateProgress(100, `Completed ${parts.length} ROM parts.`);
    return splitZipResult(parts, entry, Math.max(estimatedParts, parts.length));
  } catch (error) {
    if (recovery?.parts?.length) {
      lastPartialSplit = splitZipResult(recovery.parts, entry, recovery.parts.length, true);
      appendLog(`${recovery.parts.length} completed part${recovery.parts.length === 1 ? " was" : "s were"} kept in browser recovery storage.`);
    }
    throw error;
  } finally {
    try { await ffmpeg.deleteFile(inputName); } catch { /* already removed */ }
  }
}

async function assembleBatch(playerStub, clips, project) {
  const files = [];
  const usedNames = new Map();
  let totalFrames = 0;
  let totalSize = 0;
  for (let index = 0; index < clips.length; index += 1) {
    updateProgress(96 + ((index + 1) / clips.length) * 3, `Building ROM ${index + 1} of ${clips.length}…`);
    const clip = clips[index];
    const stub = playerStub.slice();
    const transfers = [stub.buffer, ...clipTransferList(clip)];
    const assembled = await runRomTask("assembleROM", {
      playerStub: stub,
      clips: [clip],
      options: { romTitle: clip.title, outputMode: "menu", resume: project.resume },
    }, transfers);
    let base = cleanFileBase(clip.title, `VIDEO_${index + 1}`);
    const count = usedNames.get(base) || 0;
    usedNames.set(base, count + 1);
    if (count) base = `${base}_${count + 1}`;
    const data = new Uint8Array(assembled.buffer);
    files.push({ name: `${base}.gba`, data });
    totalFrames += assembled.details.frameCount;
    totalSize += data.length;
  }
  const zip = buildStoredZip(files);
  return {
    buffer: zip.buffer,
    fileName: outputFileName(project.romTitle, "zip"),
    mime: "application/zip",
    details: { clipCount: clips.length, frameCount: totalFrames, paddedSize: totalSize, outputKind: "zip" },
  };
}

async function performConversion() {
  const project = currentOptions();
  if (project.splitVideo && !Number.isFinite(project.maxPartSeconds)) {
    throw new Error("Maximum duration must be 0, seconds, MM:SS, or H:MM:SS.");
  }
  await ensureFFmpeg();
  if (conversionCancelled) throw new Error("Conversion cancelled.");

  updateProgress(6, "Loading the embedded GBA player…");
  const playerResponse = await fetch(new URL("player_stub.bin", document.baseURI));
  if (!playerResponse.ok) throw new Error("Could not load player_stub.bin from the website.");
  const playerStub = new Uint8Array(await playerResponse.arrayBuffer());

  const singleRom = entries.length === 1 && project.outputMode === "rom";
  if (singleRom && (project.splitVideo || lastEstimate?.needsSplit)) {
    return performLongSplit(playerStub, {
      ...project,
      splitBudgetMiB: project.splitVideo ? project.splitBudgetMiB : 32,
      maxPartSeconds: project.splitVideo ? project.maxPartSeconds : 0,
      chapterAware: project.splitVideo ? project.chapterAware : true,
      partTitleScreens: project.partTitleScreens,
      titleCards: project.titleCards,
      resumeLongSplit: project.splitVideo ? project.resumeLongSplit : true,
    });
  }

  try {
    const clips = [];
    const clipSpan = 88 / entries.length;
    for (let index = 0; index < entries.length; index += 1) {
      if (conversionCancelled) throw new Error("Conversion cancelled.");
      const entry = entries[index];
      const clipOptions = effectiveClipOptions(entry, project);
      const base = 8 + index * clipSpan;
      const mapped = (fraction, message) => updateProgress(base + fraction * clipSpan, `${entry.title || "VIDEO"} — ${message}`);
      const inputName = safeVirtualName(index, entry.file.name);

      mapped(0, "Loading video into browser memory…");
      await ffmpeg.writeFile(inputName, await fetchFile(entry.file));
      try {
        mapped(0.03, "Inspecting media…");
        const probe = await readProbe(inputName, index, entry.file);
        entry.duration = probe.duration;
        entry.hasAudio = probe.hasAudio;
        entry.audioTracks = probe.audioTracks || [];
        entry.audioTracksKnown = !probe.audioUnknown;
        if (entry.hasAudio && entry.audioTracksKnown && entry.audioTrack >= entry.audioTracks.length) throw new Error("The selected input audio track is not available in this file.");
        entry.channels = audioTrackChannels(probe, entry.audioTrack);
        entry.chapters = probe.chapters || [];
        const timing = clipTiming(clipOptions, probe.duration);
        mapped(0.08, "Extracting 120×80 frames…");
        const framesRGB = await extractFrames(inputName, index, project, clipOptions, timing);
        mapped(0.31, probe.hasAudio && clipOptions.audioMode !== "none" ? "Extracting audio…" : "No audio selected.");
        const audio = await extractAudio(inputName, index, project, clipOptions, probe, timing);
        mapped(0.38, "Encoding the GBA clip…");
        const clip = await runRomTask("encodeClip", {
          framesRGB,
          audio,
          title: entry.title || "VIDEO",
          vblanks: project.vblanks,
          paletteMode: clipOptions.paletteMode,
          ditherMode: clipOptions.ditherMode,
          compression: project.compression,
          keyInterval: 30,
          adaptiveKeyframes: Boolean(project.extremeOptimization && project.adaptiveKeyframes),
          enhancedSceneDetection: Boolean(project.extremeOptimization && project.enhancedSceneDetection),
          audioCodec: project.extremeOptimization ? (project.audioQuality === "auto" ? (smartAnalysis?.recommended?.audioCodec || "pcm") : project.audioQuality) : "pcm",
          seekSeconds: project.seekSeconds,
          loop: Boolean(clipOptions.loop),
        }, [framesRGB.buffer, audio.buffer], (fraction, message) => mapped(0.38 + fraction * 0.6, message));
        clips.push(clip);
      } finally {
        try { await ffmpeg.deleteFile(inputName); } catch { /* already removed */ }
      }
    }

    updateProgress(96, "Assembling and validating output…");
    if (project.outputMode === "batch") return assembleBatch(playerStub, clips, project);

    const stub = playerStub.slice();
    const transfers = [stub.buffer];
    for (const clip of clips) transfers.push(...clipTransferList(clip));
    const assembled = await runRomTask("assembleROM", {
      playerStub: stub,
      clips,
      options: { romTitle: project.romTitle, outputMode: project.outputMode, resume: project.resume, menuTheme: project.menuTheme },
    }, transfers);
    updateProgress(100, "ROM ready.");
    return {
      ...assembled,
      fileName: outputFileName(project.romTitle, "gba"),
      mime: "application/octet-stream",
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    if (singleRom && /32 MiB|cartridge size|raw browser memory|exceed/i.test(message)) {
      appendLog("One ROM could not fit safely. Switching to automatic numbered-ROM splitting.");
      return performLongSplit(playerStub, {
        ...project,
        splitBudgetMiB: 32,
        maxPartSeconds: 0,
        chapterAware: true,
        partTitleScreens: project.partTitleScreens,
        titleCards: project.titleCards,
        resumeLongSplit: true,
      });
    }
    throw error;
  }
}

function publishConversionResult(result, autoDownload = true) {
  if (resultURL) URL.revokeObjectURL(resultURL);
  const blob = new Blob([result.buffer], { type: result.mime || "application/octet-stream" });
  resultURL = URL.createObjectURL(blob);
  resultFileName = result.fileName;
  resultMime = result.mime || "application/octet-stream";
  elements.resultTitle.textContent = result.details.partial
    ? `${resultFileName} contains the completed parts`
    : `${resultFileName} is ready`;
  const noun = result.details.outputKind === "zip" ? "combined ROM data" : "ROM";
  const estimated = result.details.estimatedParts ? ` · estimated ${result.details.estimatedParts} part${result.details.estimatedParts === 1 ? "" : "s"}` : "";
  elements.resultDetails.textContent = `${result.details.clipCount} clip${result.details.clipCount === 1 ? "" : "s"}, ${(result.details.frameCount || 0).toLocaleString()} frames, ${formatBytes(result.details.paddedSize)} ${noun}${estimated}`;
  elements.resultArea.hidden = false;
  if (autoDownload) downloadResult();
}

async function startConversion() {
  if (!entries.length || conversionRunning) return;
  const totalBytes = entries.reduce((sum, entry) => sum + entry.file.size, 0);
  if (totalBytes > 1536 * 1024 * 1024) {
    const proceed = confirm("These source files exceed 1.5 GiB. The browser may run out of memory. Continue anyway?");
    if (!proceed) return;
  }

  resetResult();
  lastPartialSplit = null;
  logLines = [];
  recentFFmpegLogs = [];
  elements.logOutput.textContent = "";
  conversionCancelled = false;
  setBusy(true);
  updateProgress(0, "Starting browser converter…");

  try {
    const result = await performConversion();
    if (conversionCancelled) return;
    publishConversionResult(result, true);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    appendLog(`ERROR: ${message}`);
    updateProgress(0, conversionCancelled ? "Conversion cancelled." : "Conversion failed.");
    if (lastPartialSplit) {
      publishConversionResult(lastPartialSplit, false);
      elements.projectNotice.textContent = "A later part failed. Completed ROM parts were kept and can be downloaded or resumed.";
    }
    if (!conversionCancelled) alert(message);
  } finally {
    setBusy(false);
    renderFiles();
  }
}

function cancelConversion() {
  if (!conversionRunning) return;
  conversionCancelled = true;
  try { ffmpeg?.terminate(); } catch { /* nothing */ }
  ffmpeg = null;
  try { romWorker?.terminate(); } catch { /* nothing */ }
  romWorker = null;
  for (const task of romTasks.values()) task.reject(new Error("Conversion cancelled."));
  romTasks.clear();
  appendLog("Conversion cancelled.");
}


function applyTimelineBoundary(kind, rawValue) {
  const entry = selectedEntry();
  if (!entry) return;
  const duration = Math.max(0, entry.duration || elements.previewVideo.duration || 0);
  let value = clampNumber(rawValue, 0, duration);
  const start = Number(elements.timelineStart.value);
  const end = Number(elements.timelineEnd.value);
  if (kind === "start") value = Math.min(value, Math.max(0, end - 0.04));
  if (kind === "end") value = Math.max(value, Math.min(duration, start + 0.04));
  if (entry.useProject) {
    if (kind === "start") elements.defaultStart.value = value.toFixed(2);
    else elements.defaultEnd.value = Math.abs(value - duration) < 0.05 ? "" : value.toFixed(2);
  } else if (kind === "start") entry.start = value;
  else entry.end = Math.abs(value - duration) < 0.05 ? 0 : value;
  if (kind === "start") elements.timelineStart.value = String(value);
  else elements.timelineEnd.value = String(value);
  resetResult();
  updateTimelineLabels();
  updateEstimate();
  renderFiles();
}

function commitTimelineTimeInput(kind, input) {
  const text = input.value.trim();
  const duration = Math.max(0, Number(elements.timelineEnd.max) || elements.previewVideo.duration || selectedEntry()?.duration || 0);
  const parsed = text === "" ? (kind === "start" ? 0 : duration) : parseClock(text);
  if (!Number.isFinite(parsed)) {
    input.setCustomValidity("Enter time as MM:SS, H:MM:SS, or seconds.");
    input.reportValidity();
    input.setCustomValidity("");
    updateTimelineLabels();
    return;
  }
  setTimelineBoundaryPreview(kind, parsed);
  const range = kind === "start" ? elements.timelineStart : elements.timelineEnd;
  applyTimelineBoundary(kind, range.value);
  input.value = formatClock(Number(range.value), true);
}

function restoreTimelineTimeInput(kind, input) {
  const range = kind === "start" ? elements.timelineStart : elements.timelineEnd;
  input.value = formatClock(Number(range.value), true);
}

function togglePreviewPlayback() {
  if (!elements.previewVideo.src) return;
  if (elements.previewVideo.paused) elements.previewVideo.play().catch(() => {});
  else elements.previewVideo.pause();
}

function wavFromSigned8(pcm, sampleRate = AUDIO_RATE) {
  const dataBytes = pcm.length * 2;
  const buffer = new ArrayBuffer(44 + dataBytes);
  const view = new DataView(buffer);
  const bytes = new Uint8Array(buffer);
  const text = (offset, value) => { for (let i = 0; i < value.length; i += 1) bytes[offset + i] = value.charCodeAt(i); };
  text(0, "RIFF"); view.setUint32(4, 36 + dataBytes, true); text(8, "WAVE"); text(12, "fmt ");
  view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true); view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true); view.setUint16(34, 16, true); text(36, "data"); view.setUint32(40, dataBytes, true);
  for (let i = 0; i < pcm.length; i += 1) view.setInt16(44 + i * 2, ((pcm[i] << 24) >> 24) << 8, true);
  return new Uint8Array(buffer);
}

async function createAudioPreview() {
  const entry = selectedEntry();
  if (!entry || conversionRunning) return;
  const project = currentOptions(false);
  const clip = effectiveClipOptions(entry, project);
  if (clip.audioMode === "none") { alert("Audio is disabled for this clip."); return; }
  elements.audioPreviewButton.disabled = true;
  let inputName = "";
  const outputName = "audio-preview.s8";
  try {
    await ensureFFmpeg();
    inputName = "audio-preview-input" + (entry.file.name.match(/\.[A-Za-z0-9]{1,8}$/)?.[0] || ".mp4");
    await ffmpeg.writeFile(inputName, await fetchFile(entry.file));
    const probe = await readProbe(inputName, 999, entry.file);
    const position = clampNumber(elements.timelinePlay.value, 0, probe.duration);
    const duration = Math.min(10, Math.max(0.25, probe.duration - position));
    const filters = [];
    if (clip.audioMode === "left") filters.push("pan=mono|c0=c0");
    if (clip.audioMode === "right") filters.push(audioTrackChannels(probe, clip.audioTrack) === 1 ? "pan=mono|c0=c0" : "pan=mono|c0=c1");
    if (Math.abs(clip.volume - 1) > 0.000001) filters.push(`volume=${clip.volume.toFixed(6)}`);
    if (project.normalize) filters.push("loudnorm=I=-16:LRA=11:TP=-1.5");
    if (project.limiter) filters.push("alimiter=limit=0.95:attack=5:release=50");
    if (probe.audioTracks?.length && (clip.audioTrack < 0 || clip.audioTrack >= probe.audioTracks.length)) throw new Error("The selected input audio track is not available in this file.");
    const args = ["-hide_banner", "-loglevel", "error", "-ss", position.toFixed(3), "-t", duration.toFixed(3), "-i", inputName, "-map", `0:a:${Number(clip.audioTrack) || 0}`, "-vn"];
    if (filters.length) args.push("-af", filters.join(","));
    args.push("-ac", "1", "-ar", String(AUDIO_RATE), "-f", "s8", outputName);
    const code = await ffmpeg.exec(args);
    if (code !== 0) throw new Error("Could not create the selected-channel audio preview.");
    let pcm = await ffmpeg.readFile(outputName);
    if (!(pcm instanceof Uint8Array)) pcm = new Uint8Array(pcm);
    const selected = project.extremeOptimization ? (project.audioQuality === "auto" ? (smartAnalysis?.recommended?.audioCodec || "pcm") : project.audioQuality) : "pcm";
    if (selected === "adpcm") pcm = decodeIMAADPCM(encodeIMAADPCM(pcm).data).pcm;
    const data = wavFromSigned8(pcm);
    if (audioPreviewURL) URL.revokeObjectURL(audioPreviewURL);
    audioPreviewURL = URL.createObjectURL(new Blob([data], { type: "audio/wav" }));
    elements.audioPreviewPlayer.src = audioPreviewURL;
    elements.audioPreviewPlayer.hidden = false;
    await elements.audioPreviewPlayer.play().catch(() => {});
  } catch (error) {
    alert(error instanceof Error ? error.message : String(error));
  } finally {
    if (ffmpeg) {
      await ffmpeg.deleteFile(outputName).catch(() => {});
      if (inputName) await ffmpeg.deleteFile(inputName).catch(() => {});
    }
    elements.audioPreviewButton.disabled = false;
  }
}

function configureDesktopLink() {
  if (!location.hostname.endsWith("github.io")) return;
  const owner = location.hostname.split(".")[0];
  const repo = location.pathname.split("/").filter(Boolean)[0];
  if (!owner || !repo) return;
  elements.desktopLink.href = `https://github.com/${owner}/${repo}/releases/latest`;
  elements.desktopLink.hidden = false;
}

function configureCompatibilityMessage() {
  const mobile = matchMedia("(pointer: coarse)").matches || /Android|iPhone|iPad/i.test(navigator.userAgent);
  if (mobile) elements.compatibilityText.textContent = "Mobile browsers can run out of memory quickly. Short clips only; use the desktop app for longer videos.";
}

if (elements.menuBackground) {
  for (const [input,label] of [[elements.menuUIColor,"UI text colour"],[elements.menuSelectionColor,"Selection colour"],[elements.menuOutlineColor,"Outline colour"]]) {
    setupGBAColorPicker(input,{label});
  }
  updateMenuColorReadouts();
  rebuildMenuTheme();
  stopMenuPreview = startPreview(elements.menuPreview, () => activeMenuTheme, menuStyleSettings);
  elements.menuBackground.addEventListener("change", () => { rebuildMenuTheme(); resetResult(); updateEstimate(); });
  for (const [input,fallback] of [[elements.menuUIColor,"#FFFFFF"],[elements.menuSelectionColor,"#FFDE00"],[elements.menuOutlineColor,"#000000"]]) {
    input.addEventListener("input", () => { updateMenuColorReadouts(); rebuildMenuTheme(); resetResult(); });
    input.addEventListener("change", () => { snapMenuColor(input,fallback); rebuildMenuTheme(); resetResult(); updateEstimate(); });
  }
  elements.menuOutline.addEventListener("change", () => { rebuildMenuTheme(); resetResult(); updateEstimate(); });
  elements.customMenuBackground.addEventListener("change", event => loadCustomMenuBackground(event.target.files?.[0]));
  elements.clearCustomMenuBackground.addEventListener("click", () => {
    customMenuTheme = null; elements.customMenuBackground.value = "";
    elements.menuBackgroundStatus.textContent = "Choose a built-in background or upload a custom image/GIF.";
    rebuildMenuTheme(); resetResult(); updateEstimate();
  });
}

if (elements.titleCardPreview) {
  for (const [input, label] of [
    [elements.titleCardTextColor, "Title text colour"],
    [elements.titleCardOutlineColor, "Title outline colour"],
    [elements.titleCardSubtitleTextColor, "Subtitle text colour"],
    [elements.titleCardSubtitleOutlineColor, "Subtitle outline colour"],
    [elements.titleCardSolidColor, "Title-card background colour"],
  ]) setupGBAColorPicker(input, { label });

  const colorFields = new Map([
    [elements.titleCardTextColor, "#FFFFFF"],
    [elements.titleCardOutlineColor, "#000000"],
    [elements.titleCardSubtitleTextColor, "#FFFFFF"],
    [elements.titleCardSubtitleOutlineColor, "#000000"],
    [elements.titleCardSolidColor, "#000000"],
  ]);
  for (const [input, fallback] of colorFields) {
    input.addEventListener("input", saveTitleCardFields);
    input.addEventListener("change", () => {
      input.value = quantizeHexColor(input.value, fallback);
      saveTitleCardFields();
    });
  }

  elements.titleCardPrev.addEventListener("click", () => setTitleCardPart(titleCardPart - 1));
  elements.titleCardNext.addEventListener("click", () => setTitleCardPart(titleCardPart + 1));
  elements.titleCardPartSelect.addEventListener("change", () => setTitleCardPart(Number(elements.titleCardPartSelect.value)));
  elements.titleCardUseShared.addEventListener("change", () => {
    ensureTitleCardProject();
    titleCardProject.useShared = elements.titleCardUseShared.checked;
    elements.titleCardCopyToAll.hidden = titleCardProject.useShared;
    loadTitleCardFields();
    resetResult();
    updateEstimate();
  });
  elements.titleCardCopyToAll.addEventListener("click", () => {
    ensureTitleCardProject();
    const source = structuredClone(titleCardPartRecord(titleCardPart, false));
    titleCardProject.parts = Array.from({ length: titleCardEstimatedParts }, (_, index) => ({ part: index + 1, settings: structuredClone(source) }));
    loadTitleCardFields();
    resetResult();
    updateEstimate();
  });

  for (const input of [
    elements.titleCardTitle, elements.titleCardSubtitle, elements.titleCardBackground,
    elements.titleCardFrameOffset, elements.titleCardDarkness, elements.titleCardOutline,
    elements.titleCardAlignment, elements.titleCardTextSize, elements.titleCardSubtitleAlignment, elements.titleCardSubtitleTextSize,
    elements.titleCardStartMode, elements.titleCardDuration, elements.titleCardAllowSkip, elements.titleCardFade,
  ]) {
    input.addEventListener("input", saveTitleCardFields);
    input.addEventListener("change", saveTitleCardFields);
  }
  updateTitleCardReadouts();
}

const presetFields = [elements.vblanks, elements.fitMode, elements.paletteMode, elements.ditherMode, elements.compression, elements.audioMode, elements.normalize, elements.limiter];
for (const control of presetFields) control.addEventListener("change", () => { markPresetCustom(); updateEstimate(); syncSelectedPreview(); });

elements.preset.addEventListener("change", () => applyPreset(elements.preset.value));
elements.smartAnalyze.addEventListener("click", runSmartAnalysis);
elements.smartCancel.addEventListener("click", () => {
  smartAnalysisCancelled = true;
  elements.smartStatus.textContent = "Cancelling analysis…";
  try { ffmpeg?.terminate(); } catch { /* best effort */ }
  ffmpeg = null;
});
for (const control of [elements.smartTarget, elements.smartPriority, elements.audioQuality]) {
  control.addEventListener("change", () => {
    if (elements.preset.value === "extreme") {
      smartAnalysis = null;
      elements.smartResults.hidden = true;
      elements.smartStatus.textContent = "Options changed — analyze again";
      resetResult();
      updateEstimate();
    }
  });
}
elements.outputMode.addEventListener("change", () => { preferredOutputMode = elements.outputMode.value; updateOutputModes(); updateConvertButton(); resetResult(); updateEstimate(); });

const ordinarySettings = [
  elements.seekSeconds, elements.defaultStart, elements.defaultEnd, elements.defaultSpeed,
  elements.defaultVolume, elements.defaultLoop, elements.romTitle, elements.resume,
];
for (const control of ordinarySettings) {
  const changed = () => { resetResult(); updateEstimate(); syncSelectedPreview(); };
  control.addEventListener("change", changed);
  if (control.tagName === "INPUT" && ["text", "number"].includes(control.type)) control.addEventListener("input", changed);
}

const splitSettings = [elements.splitBudget, elements.maxPartDuration, elements.chapterAware, elements.partTitleScreens, elements.resumeLongSplit];
for (const control of splitSettings) {
  const changed = () => { updateSplitVisibility(); resetResult(); updateEstimate(); };
  control.addEventListener("change", changed);
  if (control.tagName === "INPUT" && ["text", "range"].includes(control.type)) control.addEventListener("input", changed);
}
elements.splitVideo.addEventListener("change", () => { updateSplitVisibility(); resetResult(); updateEstimate(); });
for (const button of document.querySelectorAll(".split-preset")) {
  button.addEventListener("click", () => {
    elements.splitBudget.value = button.dataset.size || "31";
    updateSplitVisibility();
    resetResult();
    updateEstimate();
  });
}

elements.fileInput.addEventListener("change", () => {
  addFiles(elements.fileInput.files);
  elements.fileInput.value = "";
});
elements.clearButton.addEventListener("click", () => {
  entries = [];
  selectedEntryId = "";
  titleCardProject = null;
  titleCardProjectSource = "";
  titleCardPart = 1;
  titleCardVisibilitySignature = "";
  if (titleCardPreviewURL) URL.revokeObjectURL(titleCardPreviewURL);
  titleCardPreviewURL = "";
  if (titleCardPreviewVideo) { titleCardPreviewVideo.removeAttribute("src"); titleCardPreviewVideo.dataset.key = ""; }
  revokePreviewURLs();
  resetResult();
  renderFiles();
});
elements.saveProjectButton.addEventListener("click", saveProject);
elements.openProjectInput.addEventListener("change", async () => {
  const file = elements.openProjectInput.files?.[0];
  elements.openProjectInput.value = "";
  if (!file) return;
  try { await openProjectFile(file); }
  catch (error) { alert(error instanceof Error ? error.message : String(error)); }
});
elements.optimizerButton.addEventListener("click", optimizeToFit);
elements.convertButton.addEventListener("click", startConversion);
elements.cancelButton.addEventListener("click", cancelConversion);
elements.downloadButton.addEventListener("click", downloadResult);
elements.resetClipTitleButton.addEventListener("click", () => {
  menuTitleUnsupported = [];
  menuTitleUnsupportedEntryId = "";
  const entry = selectedEntry();
  if (!entry) return;
  entry.title = titleFromFile(entry.file);
  resetResult();
  renderFiles();
});
elements.titlePreviewInput.addEventListener("input", () => {
  const entry = selectedEntry();
  if (!entry) return;
  const raw = elements.titlePreviewInput.value;
  menuTitleUnsupported = unsupportedGBARunes(raw);
  menuTitleUnsupportedEntryId = entry.id;
  const cleaned = sanitizeMenuTitle(raw);
  if (elements.titlePreviewInput.value !== cleaned) elements.titlePreviewInput.value = cleaned;
  entry.title = cleaned;
  resetResult();
  updateTitlePreview();
  updateEstimate();
});
elements.titlePreviewInput.addEventListener("focus", () => {
  elements.titlePreviewInput.setSelectionRange(elements.titlePreviewInput.value.length, elements.titlePreviewInput.value.length);
  updateTitlePreview();
});
elements.titlePreviewInput.addEventListener("blur", () => {
  const entry = selectedEntry();
  if (entry && !entry.title) entry.title = "VIDEO";
  renderFiles();
});
elements.titlePreviewInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    elements.titlePreviewInput.blur();
  }
});
for (const eventName of ["click", "keyup", "select"]) {
  elements.titlePreviewInput.addEventListener(eventName, updateTitlePreview);
}

for (const [input, kind] of [
  [elements.timelineStartTimeInput, "start"],
  [elements.timelineEndTimeInput, "end"],
]) {
  input.addEventListener("focus", () => input.select());
  input.addEventListener("change", () => commitTimelineTimeInput(kind, input));
  input.addEventListener("blur", () => restoreTimelineTimeInput(kind, input));
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      commitTimelineTimeInput(kind, input);
      input.blur();
    } else if (event.key === "Escape") {
      event.preventDefault();
      restoreTimelineTimeInput(kind, input);
      input.blur();
    }
  });
}

elements.previewVideo.addEventListener("click", togglePreviewPlayback);
elements.previewVideo.addEventListener("keydown", (event) => {
  if (event.key !== " " && event.key !== "Enter") return;
  event.preventDefault();
  togglePreviewPlayback();
});

elements.timelineStartHandle.addEventListener("pointerdown", (event) => beginTimelineDrag("start", event));
elements.timelinePlayHandle.addEventListener("pointerdown", (event) => beginTimelineDrag("current", event));
elements.timelineEndHandle.addEventListener("pointerdown", (event) => beginTimelineDrag("end", event));
elements.timelineTrack.addEventListener("pointerdown", (event) => {
  if (event.target.closest(".timeline-handle")) return;
  event.preventDefault();
  event.stopPropagation();
  setTimelinePreview(timelineValueFromClientX(event.clientX));
});
for (const [handle, kind] of [
  [elements.timelineStartHandle, "start"],
  [elements.timelinePlayHandle, "current"],
  [elements.timelineEndHandle, "end"],
]) {
  handle.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    nudgeTimeline(kind, event.key === "ArrowLeft" ? -1 : 1);
  });
}

elements.timelinePlay.addEventListener("input", () => {
  elements.previewVideo.currentTime = Number(elements.timelinePlay.value);
  updateTimelineLabels();
});
elements.timelineStart.addEventListener("input", () => { updateTimelineLabels(); });
elements.timelineStart.addEventListener("change", () => applyTimelineBoundary("start", elements.timelineStart.value));
elements.timelineEnd.addEventListener("input", () => { updateTimelineLabels(); });
elements.timelineEnd.addEventListener("change", () => applyTimelineBoundary("end", elements.timelineEnd.value));
elements.previewVideo.addEventListener("timeupdate", () => {
  if (!elements.previewVideo.seeking && Number.isFinite(elements.previewVideo.currentTime)) {
    elements.timelinePlay.value = String(elements.previewVideo.currentTime);
    updateTimelineLabels();
  }
});
elements.previewVideo.addEventListener("loadedmetadata", () => {
  const entry = selectedEntry();
  if (entry && Number.isFinite(elements.previewVideo.duration)) {
    entry.duration = elements.previewVideo.duration;
    syncSelectedPreview();
    updateEstimate();
  }
});
elements.jumpBegin.addEventListener("click", () => {
  elements.timelinePlay.value = elements.timelineStart.value;
  elements.previewVideo.currentTime = Number(elements.timelineStart.value);
  updateTimelineLabels();
});
elements.jumpEnd.addEventListener("click", () => {
  elements.timelinePlay.value = elements.timelineEnd.value;
  elements.previewVideo.currentTime = Number(elements.timelineEnd.value);
  updateTimelineLabels();
});
elements.audioPreviewButton.addEventListener("click", createAudioPreview);

for (const eventName of ["dragenter", "dragover"]) {
  elements.dropZone.addEventListener(eventName, (event) => {
    event.preventDefault();
    if (!conversionRunning) elements.dropZone.classList.add("dragging");
  });
}
for (const eventName of ["dragleave", "drop"]) {
  elements.dropZone.addEventListener(eventName, (event) => {
    event.preventDefault();
    elements.dropZone.classList.remove("dragging");
  });
}
elements.dropZone.addEventListener("drop", (event) => {
  if (!conversionRunning) addFiles(event.dataTransfer.files);
});

window.addEventListener("beforeunload", () => {
  try { ffmpeg?.terminate(); } catch { /* nothing */ }
  romWorker?.terminate();
  revokePreviewURLs();
  if (titleCardPreviewURL) URL.revokeObjectURL(titleCardPreviewURL);
  if (resultURL) URL.revokeObjectURL(resultURL);
});

configureDesktopLink();
configureCompatibilityMessage();
updateConvertButton();
updateExtremeVisibility();
updateSplitVisibility();
renderFiles();
