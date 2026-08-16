import { AUDIO_RATE, GBA_REFRESH, RGB_FRAME_BYTES, FRAME_BYTES } from "./rom-core.js";
import { ADPCM_HEADER_BYTES, DEFAULT_ADPCM_BLOCK_SAMPLES } from "./adpcm.js";

const clamp = (value, low, high) => Math.max(low, Math.min(high, value));

function metricsForFrame(frame, previous) {
  let motion = 0, detail = 0, brightness = 0, colour = 0, changed = 0;
  for (let y = 0; y < 80; y += 1) {
    for (let x = 0; x < 120; x += 1) {
      const i = (y * 120 + x) * 3;
      const r = frame[i], g = frame[i + 1], b = frame[i + 2];
      const luma = (77 * r + 150 * g + 29 * b) / 256;
      brightness += luma;
      colour += Math.max(r, g, b) - Math.min(r, g, b);
      if (x > 0) {
        const j = i - 3;
        detail += Math.abs(luma - (77 * frame[j] + 150 * frame[j + 1] + 29 * frame[j + 2]) / 256);
      }
      if (y > 0) {
        const j = i - 360;
        detail += Math.abs(luma - (77 * frame[j] + 150 * frame[j + 1] + 29 * frame[j + 2]) / 256);
      }
      if (previous) {
        const prior = (77 * previous[i] + 150 * previous[i + 1] + 29 * previous[i + 2]) / 256;
        const difference = Math.abs(luma - prior);
        motion += difference;
        if (difference > 48) changed += 1;
      }
    }
  }
  return {
    motion: Math.min(1, motion / FRAME_BYTES / 64),
    detail: Math.min(1, detail / FRAME_BYTES / 2 / 64),
    brightness: brightness / FRAME_BYTES / 255,
    colour: colour / FRAME_BYTES / 255,
    scene: changed / FRAME_BYTES,
  };
}

function selectSamples(metrics, sourceStart, sourceEnd) {
  const kinds = [
    ["Typical scene", (m) => 1 - Math.abs(m.motion - 0.35)],
    ["Fast motion", (m) => m.motion * 0.75 + m.scene * 0.25],
    ["High detail", (m) => m.detail * 0.8 + m.colour * 0.2],
    ["Dark scene", (m) => (1 - m.brightness) * 0.8 + m.detail * 0.2],
    ["Bright / colourful", (m) => m.brightness * 0.35 + m.colour * 0.65],
    ["Scene transition", (m) => m.scene * 0.7 + m.motion * 0.3],
    ["Low motion", (m) => (1 - m.motion) * 0.7 + m.detail * 0.3],
  ];
  const used = new Set();
  const gap = Math.max(2, Math.floor(metrics.length / 12));
  const selected = [];
  for (const [kind, score] of kinds) {
    let best = -1, bestScore = -Infinity;
    for (let i = 0; i < metrics.length; i += 1) {
      if ([...used].some((index) => Math.abs(index - i) < gap)) continue;
      const value = score(metrics[i]);
      if (value > bestScore) { bestScore = value; best = i; }
    }
    if (best >= 0) {
      const m = metrics[best];
      selected.push({
        time: metrics.length > 1 ? sourceStart + (best * (sourceEnd - sourceStart)) / (metrics.length - 1) : sourceStart,
        kind,
        motion: m.motion,
        detail: m.detail,
        brightness: m.brightness,
        colour: m.colour,
      });
      used.add(best);
    }
    if (selected.length >= 6) break;
  }
  return selected.sort((a, b) => a.time - b.time);
}

function averageMetrics(metrics) {
  const result = { motion: 0, detail: 0, colour: 0, variability: 0 };
  if (!metrics.length) return { motion: 0.35, detail: 0.4, colour: 0.35, variability: 0.2 };
  let brightness = 0;
  for (const m of metrics) {
    result.motion += m.motion; result.detail += m.detail; result.colour += m.colour; brightness += m.brightness;
  }
  for (const key of ["motion", "detail", "colour"]) result[key] /= metrics.length;
  brightness /= metrics.length;
  for (const m of metrics) result.variability += Math.abs(m.brightness - brightness) + Math.abs(m.motion - result.motion);
  result.variability /= metrics.length * 2;
  return result;
}

function quality(candidate, metrics) {
  const fpsScore = { 4: 1, 5: 0.84, 6: 0.7, 8: 0.52 }[candidate.vblanks];
  const paletteScore = candidate.paletteMode === "scene" ? 0.96 : 0.82;
  const ditherScore = candidate.ditherMode === "error" ? 0.98 - metrics.motion * 0.2 : candidate.ditherMode === "ordered" ? 0.92 : 0.76 + (1 - metrics.detail) * 0.12;
  let temporalStability = 92 - 22 * metrics.motion;
  if (candidate.ditherMode === "error") temporalStability -= 14 * metrics.motion;
  if (candidate.paletteMode === "scene") temporalStability += 4 * metrics.colour;
  return {
    visualQuality: clamp(Math.round(58 + 24 * paletteScore + 15 * ditherScore + 8 * (1 - metrics.detail * 0.15)), 0, 100),
    motionQuality: clamp(Math.round(48 + 47 * fpsScore - 10 * metrics.motion * (1 - fpsScore)), 0, 100),
    temporalStability: clamp(Math.round(temporalStability), 0, 100),
    audioQuality: candidate.audioCodec === "adpcm" ? 88 : 100,
  };
}

function estimateCandidate(candidate, duration, hasAudio, targetBytes, metrics) {
  const fps = GBA_REFRESH / candidate.vblanks;
  const frames = Math.max(1, Math.ceil(duration * fps));
  let factor = 0.22 + 0.58 * metrics.motion + 0.12 * metrics.detail;
  if (candidate.ditherMode === "error") factor += 0.13;
  if (candidate.ditherMode === "off") factor -= 0.05;
  if (candidate.paletteMode === "scene") factor -= 0.04 * metrics.colour;
  if (candidate.adaptiveKeyframes) factor *= 0.88 + 0.08 * metrics.motion;
  factor = clamp(factor, 0.16, 0.94);
  const video = frames * FRAME_BYTES * factor;
  const indexes = frames * 8;
  let palettes = 512;
  if (candidate.paletteMode === "scene") {
    const scenes = Math.max(1, Math.ceil((duration / 18) * (0.7 + metrics.variability * 2)));
    palettes = scenes * 512 + frames * 2;
  }
  let audio = 0;
  if (hasAudio) {
    audio = duration * AUDIO_RATE;
    if (candidate.audioCodec === "adpcm") {
      const blockBytes = 4 + Math.ceil((DEFAULT_ADPCM_BLOCK_SAMPLES - 1) / 2);
      audio = ADPCM_HEADER_BYTES + Math.ceil(audio / DEFAULT_ADPCM_BLOCK_SAMPLES) * blockBytes;
    }
    audio += frames * 4;
  }
  const total = 32768 + 96 + 4096 + video + indexes + palettes + audio;
  const uncertainty = 0.06 + metrics.variability * 0.12;
  return {
    ...candidate,
    fps,
    estimatedBytes: Math.ceil(total),
    estimatedMinBytes: Math.floor(total * (1 - uncertainty)),
    estimatedMaxBytes: Math.ceil(total * (1 + uncertainty)),
    fitsTarget: Math.ceil(total * (1 + uncertainty)) <= targetBytes,
    ...quality(candidate, metrics),
  };
}

function score(candidate, priority, targetBytes) {
  let value = candidate.visualQuality * 0.45 + candidate.motionQuality * 0.3 + candidate.temporalStability * 0.2 + candidate.audioQuality * 0.05;
  const fitPenalty = candidate.estimatedMaxBytes > targetBytes
    ? 80 + ((candidate.estimatedMaxBytes - targetBytes) / targetBytes) * 100
    : 0;
  const headroom = (targetBytes - candidate.estimatedBytes) / targetBytes;
  if (priority === "longest") value = value * 0.55 + headroom * 50;
  else if (priority === "motion") value = candidate.motionQuality * 0.55 + candidate.visualQuality * 0.25 + candidate.temporalStability * 0.2;
  else if (priority === "detail") value = candidate.visualQuality * 0.65 + candidate.motionQuality * 0.2 + candidate.temporalStability * 0.15;
  else if (priority === "quality") value *= 1.08;
  else value += Math.min(0.08, Math.max(0, headroom)) * 20;
  return value - fitPenalty;
}

export function analyzeSmartScan({ framesRGB, duration, sourceStart = 0, sourceEnd = sourceStart + duration, hasAudio = true, targetBytes = 32 * 1024 * 1024, priority = "balanced", audioQuality = "auto" }) {
  if (!(framesRGB instanceof Uint8Array) || framesRGB.length % RGB_FRAME_BYTES !== 0) throw new Error("Smart scan frames are invalid.");
  const frameCount = framesRGB.length / RGB_FRAME_BYTES;
  const metrics = [];
  let previous = null;
  for (let frame = 0; frame < frameCount; frame += 1) {
    const current = framesRGB.subarray(frame * RGB_FRAME_BYTES, (frame + 1) * RGB_FRAME_BYTES);
    metrics.push(metricsForFrame(current, previous));
    previous = current;
  }
  const averages = averageMetrics(metrics);
  const templates = [
    { id: "sharp", label: "Sharper", vblanks: 4, paletteMode: "scene", ditherMode: "error", audioCodec: "pcm", summary: "Preserves fine detail and colour." },
    { id: "balanced", label: "Recommended", vblanks: 5, paletteMode: "scene", ditherMode: "ordered", audioCodec: "pcm", summary: "Balances detail, motion and stability." },
    { id: "stable", label: "Stable", vblanks: 5, paletteMode: "shared", ditherMode: "ordered", audioCodec: "pcm", summary: "Reduces palette flicker and delta noise." },
    { id: "compact", label: "Smaller", vblanks: 6, paletteMode: "shared", ditherMode: "ordered", audioCodec: "pcm", summary: "Trades some motion for ROM space." },
    { id: "longest", label: "Longest", vblanks: 8, paletteMode: "shared", ditherMode: "off", audioCodec: "adpcm", summary: "Prioritizes maximum duration." },
    { id: "smooth", label: "Smoother", vblanks: 4, paletteMode: "shared", ditherMode: "ordered", audioCodec: "adpcm", summary: "Keeps a higher frame rate with compact audio." },
  ].map((candidate) => {
    let audioCodec = candidate.audioCodec;
    if (audioQuality === "pcm") audioCodec = "pcm";
    if (audioQuality === "adpcm") audioCodec = "adpcm";
    return estimateCandidate({ ...candidate, audioCodec, adaptiveKeyframes: true }, duration, hasAudio, targetBytes, averages);
  });
  templates.sort((a, b) => score(b, priority, targetBytes) - score(a, priority, targetBytes));
  return {
    version: 1,
    targetBytes,
    priority,
    duration,
    confidence: frameCount >= 80 && averages.variability < 0.28 ? "high" : frameCount < 30 || averages.variability > 0.45 ? "low" : "medium",
    analyzedAt: new Date().toISOString(),
    samples: selectSamples(metrics, sourceStart, sourceEnd),
    recommended: templates[0],
    alternatives: templates.slice(1, 4),
    candidates: templates,
  };
}
