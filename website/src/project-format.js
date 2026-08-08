export const PROJECT_FORMAT = "gba-video-maker-project";
export const PROJECT_VERSION = 1;

const FPS_TO_VBLANKS = { smooth: 4, balanced: 5, classic: 6, compact: 8 };
const VBLANKS_TO_FPS = { 4: "smooth", 5: "balanced", 6: "classic", 8: "compact" };

function numberOr(value, fallback) {
  const number = Number(value);
  return Number.isFinite(number) ? number : fallback;
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
    vblanks: numberOr(settings.vblanks, 5),
    fitMode: settings.fitMode || "fit",
    paletteMode: settings.paletteMode || "shared",
    ditherMode: settings.ditherMode || "ordered",
    compression: settings.compression || "delta",
    audioMode: settings.audioMode || "mix",
    seekSeconds: numberOr(settings.seekSeconds, 5),
    defaultStart: numberOr(settings.defaultStart, 0),
    defaultEnd: numberOr(settings.defaultEnd, 0),
    defaultSpeed: numberOr(settings.defaultSpeed, 1),
    defaultVolume: numberOr(settings.defaultVolume, 1),
    defaultLoop: Boolean(settings.defaultLoop),
    romTitle: settings.romTitle || "GBA VIDEO",
    normalize: Boolean(settings.normalize),
    limiter: settings.limiter !== false,
    resume: settings.resume !== false,
    outputMode: settings.outputMode || "rom",
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

export function canonicalProjectFromBrowser({ settings, entries, appVersion = "0.12.2" }) {
  const fps = VBLANKS_TO_FPS[Number(settings.vblanks)] || "balanced";
  const doc = {
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
      romTitle: settings.romTitle || "GBA VIDEO",
      seekSeconds: numberOr(settings.seekSeconds, 5),
      normalize: Boolean(settings.normalize),
      limiter: Boolean(settings.limiter),
      resume: Boolean(settings.resume),
      compression: settings.compression || "delta",
      paletteMode: settings.paletteMode || "shared",
      ditherMode: settings.ditherMode || "ordered",
      outputMode: settings.outputMode || "rom",
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
        title: entry.title || "GBA VIDEO",
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
      },
    })),
  };
  return doc;
}

export function browserStateFromCanonicalProject(doc) {
  if (!doc || doc.format !== PROJECT_FORMAT || Number(doc.version) !== PROJECT_VERSION || !Array.isArray(doc.clips) || doc.clips.length === 0) {
    throw new Error("This is not a valid .gbavideo project.");
  }
  const settings = doc.settings || {};
  return {
    settings: {
      preset: settings.preset || "custom",
      audioQuality: settings.audioQuality || "pcm",
      smartTargetMiB: numberOr(settings.smartTargetMiB, 32),
      smartPriority: settings.smartPriority || "balanced",
      vblanks: FPS_TO_VBLANKS[settings.fps] || 5,
      fitMode: settings.fit || "fit",
      paletteMode: settings.paletteMode || "shared",
      ditherMode: settings.ditherMode || "ordered",
      compression: settings.compression || "delta",
      audioMode: settings.audio || "mix",
      seekSeconds: numberOr(settings.seekSeconds, 5),
      defaultStart: parseProjectClock(settings.start, 0),
      defaultEnd: String(settings.end ?? "").trim() ? parseProjectClock(settings.end, 0) : 0,
      defaultSpeed: numberOr(settings.speed, 1),
      defaultVolume: numberOr(settings.volume, 100) / 100,
      defaultLoop: Boolean(settings.loop),
      romTitle: settings.romTitle || "GBA VIDEO",
      normalize: Boolean(settings.normalize),
      limiter: settings.limiter !== false,
      resume: settings.resume !== false,
      outputMode: settings.outputMode || "rom",
      splitVideo: Boolean(settings.splitVideo),
      splitBudgetMiB: numberOr(settings.splitBudgetMiB, 31),
      maxPartDuration: String(settings.maxPartDuration ?? "0").trim() || "0",
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
    },
    clips: doc.clips.map((saved) => ({
      source: {
        name: saved.name || "video",
        size: Number(saved.size) || 0,
        lastModified: Number(saved.lastModified) || 0,
      },
      title: saved.settings?.title || "GBA VIDEO",
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
    })),
  };
}

export function normalizeBrowserProjectDocument(parsed) {
  if (parsed?.format === PROJECT_FORMAT) return browserStateFromCanonicalProject(parsed);
  if (parsed?.format === "GBA Video Maker Project" && Array.isArray(parsed.clips) && parsed.clips.length > 0) {
    return {
      settings: browserSettingsFromLegacy(parsed.settings || {}),
      clips: parsed.clips.map((saved) => ({
        source: {
          name: saved.source?.name || "video",
          size: Number(saved.source?.size) || 0,
          lastModified: Number(saved.source?.lastModified) || 0,
        },
        title: saved.title || "GBA VIDEO",
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
      })),
    };
  }
  throw new Error("This is not a valid .gbavideo project.");
}
