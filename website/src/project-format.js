export const PROJECT_FORMAT = "gba-media-maker-project";
export const PROJECT_VERSION = 2;
export const LEGACY_PROJECT_FORMAT = "gba-video-maker-project";
export const LEGACY_PROJECT_VERSION = 1;

const FPS_TO_VBLANKS = { smooth: 4, balanced: 5, classic: 6, compact: 8 };
const VBLANKS_TO_FPS = { 4: "smooth", 5: "balanced", 6: "classic", 8: "compact" };
const ARTWORK_MODES = new Set(["embedded", "default", "custom"]);
const ARTWORK_PRESET = /^preset-(0[1-9]|1[0-9]|20)$/;

function numberOr(value, fallback) {
  const number = Number(value);
  return Number.isFinite(number) ? number : fallback;
}

function normalizeArtworkMode(value) {
  return ARTWORK_MODES.has(String(value || "").toLowerCase()) ? String(value).toLowerCase() : "embedded";
}

function normalizeArtworkPreset(value) {
  const text = String(value || "").toLowerCase();
  return ARTWORK_PRESET.test(text) ? text : "preset-01";
}

function projectClock(seconds, blankWhenZero = false) {
  const value = Math.max(0, numberOr(seconds, 0));
  if (blankWhenZero && value <= 0) return "";
  const minutes = Math.floor(value / 60);
  const rest = value - minutes * 60;
  if (Math.abs(rest - Math.round(rest)) < 0.0005) return `${minutes}:${String(Math.round(rest)).padStart(2, "0")}`;
  return `${minutes}:${rest.toFixed(2).padStart(5, "0")}`;
}

function parseProjectClock(value, fallback = 0) {
  const text = String(value ?? "").trim();
  if (!text) return fallback;
  const parts = text.split(":").map(Number);
  if (!parts.length || parts.length > 3 || parts.some((part) => !Number.isFinite(part))) return fallback;
  if (parts.length === 3) return parts[0] * 3600 + parts[1] * 60 + parts[2];
  if (parts.length === 2) return parts[0] * 60 + parts[1];
  return parts[0];
}

function browserSettingsFromLegacy(settings = {}) {
  return {
    preset: settings.preset || "custom",
    audioQuality: settings.audioQuality || "pcm",
    smartTargetMiB: numberOr(settings.smartTargetMiB, 32),
    smartPriority: settings.smartPriority || "balanced",
    vblanks: numberOr(settings.vblanks, FPS_TO_VBLANKS[settings.fps] || 5),
    fitMode: settings.fitMode || settings.fit || "fit",
    paletteMode: settings.paletteMode || "shared",
    ditherMode: settings.ditherMode || "ordered",
    compression: settings.compression || "delta",
    audioMode: settings.audioMode || settings.audio || "mix",
    seekSeconds: numberOr(settings.seekSeconds, 5),
    defaultStart: numberOr(settings.defaultStart, parseProjectClock(settings.start, 0)),
    defaultEnd: numberOr(settings.defaultEnd, String(settings.end ?? "").trim() ? parseProjectClock(settings.end, 0) : 0),
    defaultSpeed: numberOr(settings.defaultSpeed, numberOr(settings.speed, 1)),
    defaultVolume: numberOr(settings.defaultVolume, numberOr(settings.volume, 100) / 100),
    defaultLoop: Boolean(settings.defaultLoop ?? settings.loop),
    defaultImageSeconds: Math.max(0, numberOr(settings.defaultImageSeconds ?? settings.imageSeconds, 5)),
    romTitle: settings.romTitle || "GBA MEDIA",
    normalize: Boolean(settings.normalize),
    limiter: settings.limiter !== false,
    resume: settings.resume !== false,
    outputMode: settings.outputMode === "batch" ? "batch" : (settings.outputMode === "menu" || settings.outputMode === "playlist" ? "menu" : "rom"),
    splitVideo: Boolean(settings.splitVideo),
    splitBudgetMiB: numberOr(settings.splitBudgetMiB, 31),
    maxPartDuration: String(settings.maxPartDuration ?? "0"),
    chapterAware: settings.chapterAware !== false,
    partTitleScreens: settings.titleCards?.enabled ?? (settings.partTitleScreens !== false),
    titleCards: settings.titleCards || null,
    resumeLongSplit: settings.resumeLongSplit !== false,
    menuBackground: settings.menuBackground || settings.menuTheme?.id || "ocean-wave-animated",
    menuUIColor: settings.menuUIColor || "#FFFFFF",
    menuSelectionColor: settings.menuSelectionColor || "#FFDE00",
    menuOutline: settings.menuOutline !== false,
    menuOutlineColor: settings.menuOutlineColor || "#000000",
    menuTheme: settings.menuTheme || null,
  };
}

export function canonicalProjectFromBrowser({ settings, entries, appVersion = "0.13.0" }) {
  const fps = VBLANKS_TO_FPS[Number(settings.vblanks)] || "balanced";
  return {
    format: PROJECT_FORMAT,
    version: PROJECT_VERSION,
    appVersion,
    settings: {
      preset: settings.preset || "custom",
      audioQuality: settings.audioQuality || "pcm",
      smartTargetMiB: numberOr(settings.smartTargetMiB, 32),
      smartPriority: settings.smartPriority || "balanced",
      start: projectClock(settings.defaultStart),
      end: projectClock(settings.defaultEnd, true),
      speed: numberOr(settings.defaultSpeed, 1),
      fps,
      fit: settings.fitMode || "fit",
      audio: settings.audioMode || "mix",
      volume: Math.round(numberOr(settings.defaultVolume, 1) * 10000) / 100,
      loop: Boolean(settings.defaultLoop),
      imageSeconds: Math.max(0, numberOr(settings.defaultImageSeconds, 5)),
      romTitle: settings.romTitle || "GBA MEDIA",
      seekSeconds: numberOr(settings.seekSeconds, 5),
      normalize: Boolean(settings.normalize),
      limiter: Boolean(settings.limiter),
      resume: Boolean(settings.resume),
      compression: settings.compression || "delta",
      paletteMode: settings.paletteMode || "shared",
      ditherMode: settings.ditherMode || "ordered",
      outputMode: settings.outputMode === "batch" ? "batch" : (settings.outputMode === "menu" || settings.outputMode === "playlist" ? "menu" : "rom"),
      splitVideo: Boolean(settings.splitVideo),
      splitBudgetMiB: numberOr(settings.splitBudgetMiB, 31),
      maxPartDuration: String(settings.maxPartDuration ?? "0").trim() || "0",
      chapterAware: settings.chapterAware !== false,
      partTitleScreens: Boolean(settings.partTitleScreens),
      resumeLongSplit: settings.resumeLongSplit !== false,
      titleCards: settings.titleCards || null,
      menuBackground: settings.menuBackground || "ocean-wave-animated",
      menuUIColor: settings.menuUIColor || "#FFFFFF",
      menuSelectionColor: settings.menuSelectionColor || "#FFDE00",
      menuOutline: settings.menuOutline !== false,
      menuOutlineColor: settings.menuOutlineColor || "#000000",
      menuTheme: settings.outputMode === "menu" ? (settings.menuTheme || null) : null,
    },
    clips: entries.map((entry) => ({
      path: "",
      name: entry.file.name,
      size: Number(entry.file.size) || 0,
      lastModified: Number(entry.file.lastModified) || 0,
      settings: {
        title: entry.title || "GBA MEDIA",
        useProject: entry.useProject !== false,
        start: projectClock(entry.start),
        end: projectClock(entry.end, true),
        speed: numberOr(entry.speed, 1),
        fit: entry.fitMode || "fit",
        audio: entry.audioMode || "mix",
        audioTrack: Number.isInteger(entry.audioTrack) ? entry.audioTrack : 0,
        volume: Math.round(numberOr(entry.volume, 1) * 10000) / 100,
        loop: Boolean(entry.loop),
        paletteMode: entry.paletteMode || "shared",
        ditherMode: entry.ditherMode || "ordered",
        imageSeconds: Math.max(0, numberOr(entry.imageSeconds, 5)),
        musicTitle: String(entry.musicTitle || "").slice(0, 28),
        musicArtist: String(entry.musicArtist || "").slice(0, 28),
        musicArtworkMode: normalizeArtworkMode(entry.musicArtworkMode),
        musicArtworkPreset: normalizeArtworkPreset(entry.musicArtworkPreset),
        musicArtworkCustom: String(entry.musicArtworkCustom || ""),
        musicSeekSeconds: [3, 5, 10, 15].includes(Number(entry.musicSeekSeconds)) ? Number(entry.musicSeekSeconds) : 5,
      },
    })),
  };
}

function stateFromDocument(doc) {
  const settings = browserSettingsFromLegacy(doc.settings || {});
  return {
    settings,
    clips: doc.clips.map((saved) => ({
      source: {
        name: saved.name || "media",
        size: Number(saved.size) || 0,
        lastModified: Number(saved.lastModified) || 0,
      },
      title: saved.settings?.title || "GBA MEDIA",
      useProject: saved.settings?.useProject !== false,
      start: parseProjectClock(saved.settings?.start, 0),
      end: String(saved.settings?.end ?? "").trim() ? parseProjectClock(saved.settings?.end, 0) : 0,
      speed: numberOr(saved.settings?.speed, 1),
      fitMode: saved.settings?.fit || "fit",
      audioMode: saved.settings?.audio || "mix",
      audioTrack: Number.isInteger(saved.settings?.audioTrack) ? saved.settings.audioTrack : 0,
      volume: numberOr(saved.settings?.volume, 100) / 100,
      loop: Boolean(saved.settings?.loop),
      paletteMode: saved.settings?.paletteMode || "shared",
      ditherMode: saved.settings?.ditherMode || "ordered",
      imageSeconds: Math.max(0, numberOr(saved.settings?.imageSeconds, settings.defaultImageSeconds)),
      musicTitle: String(saved.settings?.musicTitle || "").slice(0, 28),
      musicArtist: String(saved.settings?.musicArtist || "").slice(0, 28),
      musicArtworkMode: normalizeArtworkMode(saved.settings?.musicArtworkMode),
      musicArtworkPreset: normalizeArtworkPreset(saved.settings?.musicArtworkPreset),
      musicArtworkCustom: String(saved.settings?.musicArtworkCustom || ""),
      musicSeekSeconds: [3, 5, 10, 15].includes(Number(saved.settings?.musicSeekSeconds)) ? Number(saved.settings.musicSeekSeconds) : numberOr(settings.seekSeconds, 5),
    })),
  };
}

export function browserStateFromCanonicalProject(doc) {
  if (!doc || !Array.isArray(doc.clips) || doc.clips.length === 0) throw new Error("This is not a valid GBA Media Maker project.");
  const current = doc.format === PROJECT_FORMAT && Number(doc.version) === PROJECT_VERSION;
  const legacy = doc.format === LEGACY_PROJECT_FORMAT && Number(doc.version) === LEGACY_PROJECT_VERSION;
  if (!current && !legacy) throw new Error("This is not a supported GBA Media Maker project.");
  return stateFromDocument(doc);
}

export function normalizeBrowserProjectDocument(parsed) {
  if ((parsed?.format === PROJECT_FORMAT && Number(parsed.version) === PROJECT_VERSION) ||
      (parsed?.format === LEGACY_PROJECT_FORMAT && Number(parsed.version) === LEGACY_PROJECT_VERSION)) {
    return browserStateFromCanonicalProject(parsed);
  }
  if (parsed?.format === "GBA Video Maker Project" && Array.isArray(parsed.clips) && parsed.clips.length > 0) {
    return {
      settings: browserSettingsFromLegacy(parsed.settings || {}),
      clips: parsed.clips.map((saved) => ({
        source: {
          name: saved.source?.name || "media",
          size: Number(saved.source?.size) || 0,
          lastModified: Number(saved.source?.lastModified) || 0,
        },
        title: saved.title || "GBA MEDIA",
        useProject: saved.useProject !== false,
        start: numberOr(saved.start, 0),
        end: numberOr(saved.end, 0),
        speed: numberOr(saved.speed, 1),
        fitMode: saved.fitMode || "fit",
        audioMode: saved.audioMode || "mix",
        audioTrack: Number.isInteger(saved.audioTrack) ? saved.audioTrack : 0,
        volume: numberOr(saved.volume, 1),
        loop: Boolean(saved.loop),
        paletteMode: saved.paletteMode || "shared",
        ditherMode: saved.ditherMode || "ordered",
        imageSeconds: Math.max(0, numberOr(saved.imageSeconds, 5)),
        musicTitle: String(saved.musicTitle || "").slice(0, 28),
        musicArtist: String(saved.musicArtist || "").slice(0, 28),
        musicArtworkMode: normalizeArtworkMode(saved.musicArtworkMode),
        musicArtworkPreset: normalizeArtworkPreset(saved.musicArtworkPreset),
        musicArtworkCustom: String(saved.musicArtworkCustom || ""),
      })),
    };
  }
  throw new Error("This is not a valid GBA Media Maker project.");
}
