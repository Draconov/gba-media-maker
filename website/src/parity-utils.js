export const FPS_VBLANKS = { smooth: 4, balanced: 5, classic: 6, compact: 8 };
export const VBLANKS_FPS = { 4: "smooth", 5: "balanced", 6: "classic", 8: "compact" };
export const FPS_ORDER = ["smooth", "balanced", "classic", "compact"];

export function sanitizeDesktopFilename(name) {
  const trimmed = String(name || "").trim();
  const leaf = trimmed.split(/[\\/]/).pop() || "";
  const safe = leaf.replace(/[<>:"/\\|?*\x00-\x1F]/g, "_").trim() || "video";
  return Array.from(safe).slice(0, 180).join("");
}

export function sourceBaseName(fileName, fallback = "video") {
  const safe = sanitizeDesktopFilename(fileName);
  const extension = /\.[^.]*$/.exec(safe)?.[0] || "";
  const base = extension ? safe.slice(0, -extension.length) : safe;
  return base.trim() || fallback;
}

export function conversionOutputFileName(fileNames, outputMode) {
  const names = Array.from(fileNames || []);
  const base = names.length > 1 ? "GBA_Video_Collection" : sourceBaseName(names[0], "video");
  return `${base}.${outputMode === "batch" ? "zip" : "gba"}`;
}

export function splitArchiveFileName(fileName) {
  return `${sourceBaseName(fileName, "video")}_PARTS.zip`;
}

export function splitPartFileName(fileName, part) {
  return `${sourceBaseName(fileName, "video")}_PART_${String(part).padStart(2, "0")}.gba`;
}

export function batchRomFileName(fileName) {
  return `${sourceBaseName(fileName, "video")}_GBA.gba`;
}

export function parsePartDuration(value) {
  const text = String(value ?? "").trim();
  if (text === "" || text === "0") return 0;
  const match = /^(\d+):([0-5]\d)$/.exec(text);
  if (!match) return NaN;
  const seconds = Number(match[1]) * 60 + Number(match[2]);
  return seconds > 0 && seconds <= 240 * 60 ? seconds : NaN;
}

export function resolveAudioCodec(requested, extreme, predictedPCMBytes, targetMiB = 32) {
  if (requested === "adpcm") return "adpcm";
  if (requested !== "auto") return "pcm";
  const target = Math.max(1, Math.min(32, Number(targetMiB) || 32)) * 1048576;
  return extreme && predictedPCMBytes > 0 && predictedPCMBytes > target / 3 ? "adpcm" : "pcm";
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function effective(model, id) {
  const config = model.clips[id];
  return config.useProject ? { ...model.defaults, title: config.title } : { ...config };
}

function makeCustom(model, id) {
  const config = model.clips[id];
  if (config.useProject) model.clips[id] = { ...model.defaults, title: config.title, useProject: false };
  return model.clips[id];
}

export function estimateOptimizerModel(model, entries, { romLimit = 32 * 1048576, menuThemeBytes = 0, audioRate = 16384 } = {}) {
  const vblanks = FPS_VBLANKS[model.global.fps] || 5;
  const fps = 59.727500569606 / vblanks;
  let player = 32768 + entries.length * 96 + (model.global.outputMode === "menu" ? menuThemeBytes : 0);
  let videoBytes = 0, audioBytes = 0, paletteBytes = 0, indexBytes = 0, frames = 0, sourceDuration = 0;
  for (const entry of entries) {
    if (!(entry.duration > 0)) continue;
    const clip = effective(model, entry.id);
    const start = Math.max(0, Number(clip.start) || 0);
    let end = Number(clip.end) > 0 ? Number(clip.end) : entry.duration;
    if (!Number.isFinite(end)) end = entry.duration;
    end = Math.min(entry.duration, end);
    if (end <= start || !Number.isFinite(Number(clip.speed)) || Number(clip.speed) <= 0) return { error: "Check trim settings." };
    const sourceClipDuration = end - start;
    sourceDuration += sourceClipDuration;
    const displayDuration = sourceClipDuration / Number(clip.speed);
    const frameCount = Math.max(1, Math.ceil(displayDuration * fps));
    frames += frameCount;
    const compressionFactor = model.global.compression === "delta" ? (model.global.preset === "extreme" ? 0.61 : 0.68) : 1;
    videoBytes += frameCount * 9600 * compressionFactor;
    if (model.global.compression === "delta") indexBytes += frameCount * 8;
    const palettes = clip.paletteMode === "scene" ? Math.max(1, Math.ceil(frameCount / 60)) : 1;
    paletteBytes += palettes * 512 + (palettes > 1 ? frameCount * 2 : 0);
    if (clip.audio !== "none" && entry.hasAudio !== false) {
      const requested = model.global.preset === "extreme" ? model.global.audioQuality : "pcm";
      const pcmBytes = displayDuration * audioRate;
      const codec = resolveAudioCodec(requested, model.global.preset === "extreme", pcmBytes, model.global.smartTargetMiB);
      audioBytes += pcmBytes * (codec === "adpcm" ? 0.505 : 1) + frameCount * 4;
    }
  }
  const bytes = Math.ceil(player + videoBytes + audioBytes + paletteBytes + indexBytes);
  let cartridge = 1 << 20;
  while (cartridge < bytes && cartridge < romLimit) cartridge *= 2;
  return { bytes, cartridge, frames, fps, sourceDuration, breakdown: { player, video: videoBytes, audio: audioBytes, palettes: paletteBytes, indexes: indexBytes } };
}

function clockValue(seconds) {
  const value = Math.max(0, Number(seconds) || 0);
  const minutes = Math.floor(value / 60);
  const rest = value - minutes * 60;
  return `${minutes}:${rest.toFixed(2).padStart(5, "0")}`;
}

export function buildOptimizerProposal(snapshot, entries, selectedId, options = {}) {
  const model = clone(snapshot);
  const changes = [];
  let result = estimateOptimizerModel(model, entries, options);
  if (result.error) return { model, changes: [], before: result, after: result, noop: true };
  if (result.bytes <= (options.romLimit || 32 * 1048576)) {
    return { model, changes: ["The current project already fits within 32 MiB."], before: result, after: result, noop: true };
  }
  const before = result;
  const limit = options.romLimit || 32 * 1048576;
  if (model.global.compression !== "delta") {
    model.global.compression = "delta";
    changes.push("Video compression: Uncompressed → Delta + keyframes");
    result = estimateOptimizerModel(model, entries, options);
  }
  while (result.bytes > limit) {
    const index = FPS_ORDER.indexOf(model.global.fps);
    if (index < 0 || index === FPS_ORDER.length - 1) break;
    const old = model.global.fps;
    const next = FPS_ORDER[index + 1];
    model.global.fps = next;
    changes.push(`Frame rate: ${(59.7275 / FPS_VBLANKS[old]).toFixed(2)} fps → ${(59.7275 / FPS_VBLANKS[next]).toFixed(2)} fps`);
    result = estimateOptimizerModel(model, entries, options);
  }
  if (result.bytes > limit) {
    if (model.defaults.paletteMode === "scene") {
      model.defaults.paletteMode = "shared";
      changes.push("Project palette: Per-scene → Shared");
    }
    for (const entry of entries) {
      const config = model.clips[entry.id];
      if (!config.useProject && config.paletteMode === "scene") {
        config.paletteMode = "shared";
        changes.push(`${entry.name}: Per-scene palette → Shared`);
      }
    }
    result = estimateOptimizerModel(model, entries, options);
  }
  if (result.bytes > limit) {
    const candidates = entries.map((entry) => {
      const clip = effective(model, entry.id);
      const start = Number(clip.start) || 0;
      const end = Number(clip.end) > 0 ? Math.min(Number(clip.end), entry.duration) : entry.duration;
      return { entry, bytes: clip.audio === "none" ? 0 : Math.max(0, (end - start) / Number(clip.speed || 1)) * 16384 };
    }).sort((a, b) => b.bytes - a.bytes);
    for (const candidate of candidates) {
      if (result.bytes <= limit) break;
      if (!candidate.bytes) continue;
      const config = makeCustom(model, candidate.entry.id);
      config.audio = "none";
      changes.push(`${candidate.entry.name}: Audio → None`);
      result = estimateOptimizerModel(model, entries, options);
    }
  }
  if (result.bytes > limit && entries.length) {
    const entry = entries.find((item) => item.id === selectedId) || entries[entries.length - 1];
    const config = makeCustom(model, entry.id);
    const start = Number(config.start) || 0;
    const originalEnd = Number(config.end) > 0 ? Math.min(Number(config.end), entry.duration) : entry.duration;
    let low = start + 0.2, high = originalEnd, best = low;
    for (let index = 0; index < 24; index += 1) {
      const mid = (low + high) / 2;
      config.end = mid;
      const test = estimateOptimizerModel(model, entries, options);
      if (test.bytes <= limit) { best = mid; low = mid; } else high = mid;
    }
    config.end = best;
    changes.push(`${entry.name}: End time shortened to ${clockValue(best)}`);
    result = estimateOptimizerModel(model, entries, options);
  }
  return { model, changes, before, after: result, noop: false };
}
