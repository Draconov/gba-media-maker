import { encodeIMAADPCM, DEFAULT_ADPCM_BLOCK_SAMPLES } from "./adpcm.js";
import { encodeGBATextFixed, safeGBAHeaderTitle } from "./gba-text.js";

export const ROM_LIMIT = 32 * 1024 * 1024;
export const ROM_MIN_SIZE = 1 * 1024 * 1024;
export const METADATA_OFFSET = 0x7fc0;
export const ASSET_OFFSET = 0x8000;
export const CLIP_DESCRIPTOR_SIZE = 96;
export const FRAME_WIDTH = 120;
export const FRAME_HEIGHT = 80;
export const FRAME_BYTES = FRAME_WIDTH * FRAME_HEIGHT;
export const RGB_FRAME_BYTES = FRAME_BYTES * 3;
export const NATIVE_IMAGE_WIDTH = 240;
export const NATIVE_IMAGE_HEIGHT = 160;
export const NATIVE_IMAGE_PIXELS = NATIVE_IMAGE_WIDTH * NATIVE_IMAGE_HEIGHT;
export const NATIVE_IMAGE_RGB_BYTES = NATIVE_IMAGE_PIXELS * 3;
export const NATIVE_IMAGE_BYTES = NATIVE_IMAGE_PIXELS * 2;
export const AUDIO_RATE = 16384;
export const VIDEO_PALETTE_COLORS = 250;
export const GBA_REFRESH = 59.727500569606;
const MENU_THEME_HEADER_SIZE = 64;
const MENU_THEME_MAGIC = 0x3148544d;

const NINTENDO_LOGO = new Uint8Array([
  0x24, 0xff, 0xae, 0x51, 0x69, 0x9a, 0xa2, 0x21, 0x3d, 0x84, 0x82, 0x0a, 0x84, 0xe4, 0x09, 0xad,
  0x11, 0x24, 0x8b, 0x98, 0xc0, 0x81, 0x7f, 0x21, 0xa3, 0x52, 0xbe, 0x19, 0x93, 0x09, 0xce, 0x20,
  0x10, 0x46, 0x4a, 0x4a, 0xf8, 0x27, 0x31, 0xec, 0x58, 0xc7, 0xe8, 0x33, 0x82, 0xe3, 0xce, 0xbf,
  0x85, 0xf4, 0xdf, 0x94, 0xce, 0x4b, 0x09, 0xc1, 0x94, 0x56, 0x8a, 0xc0, 0x13, 0x72, 0xa7, 0xfc,
  0x9f, 0x84, 0x4d, 0x73, 0xa3, 0xca, 0x9a, 0x61, 0x58, 0x97, 0xa3, 0x27, 0xfc, 0x03, 0x98, 0x76,
  0x23, 0x1d, 0xc7, 0x61, 0x03, 0x04, 0xae, 0x56, 0xbf, 0x38, 0x84, 0x00, 0x40, 0xa7, 0x0e, 0xfd,
  0xff, 0x52, 0xfe, 0x03, 0x6f, 0x95, 0x30, 0xf1, 0x97, 0xfb, 0xc0, 0x85, 0x60, 0xd6, 0x80, 0x25,
  0xa9, 0x63, 0xbe, 0x03, 0x01, 0x4e, 0x38, 0xe2, 0xf9, 0xa2, 0x34, 0xff, 0xbb, 0x3e, 0x03, 0x44,
  0x78, 0x00, 0x90, 0xcb, 0x88, 0x11, 0x3a, 0x94, 0x65, 0xc0, 0x7c, 0x63, 0x87, 0xf0, 0x3c, 0xaf,
  0xd6, 0x25, 0xe4, 0x8b, 0x38, 0x0a, 0xac, 0x72, 0x21, 0xd4, 0xf8, 0x07,
]);

const BAYER_4X4 = new Int8Array([0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5]);

class ByteWriter {
  constructor(initialCapacity = 1024 * 1024) {
    this.buffer = new Uint8Array(initialCapacity);
    this.length = 0;
  }

  ensure(extra) {
    const required = this.length + extra;
    if (required <= this.buffer.length) return;
    let next = this.buffer.length;
    while (next < required) next *= 2;
    const grown = new Uint8Array(next);
    grown.set(this.buffer.subarray(0, this.length));
    this.buffer = grown;
  }

  write(bytes) {
    this.ensure(bytes.length);
    const offset = this.length;
    this.buffer.set(bytes, offset);
    this.length += bytes.length;
    return offset;
  }

  reserve(count) {
    this.ensure(count);
    const offset = this.length;
    this.buffer.fill(0, offset, offset + count);
    this.length += count;
    return offset;
  }

  align(alignment = 4) {
    const padding = (alignment - (this.length % alignment)) % alignment;
    if (padding) this.reserve(padding);
  }

  writeAligned(bytes, alignment = 4) {
    const offset = this.write(bytes);
    this.align(alignment);
    return offset;
  }

  finish() {
    return this.buffer.slice(0, this.length);
  }
}

function clamp(value, low, high) {
  return value < low ? low : value > high ? high : value;
}

function setU16(bytes, offset, value) {
  new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).setUint16(offset, value >>> 0, true);
}

function setU32(bytes, offset, value) {
  new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).setUint32(offset, value >>> 0, true);
}

function nextPowerOfTwo(value) {
  let result = ROM_MIN_SIZE;
  while (result < value) result *= 2;
  return result;
}

function asciiBytes(value) {
  return new TextEncoder().encode(value);
}

export function safeRomTitle(value) {
  return safeGBAHeaderTitle(value);
}

function safeTitleScreenName(value) {
  return encodeGBATextFixed(value, 24);
}


function patchGBAHeader(rom, title) {
  setU32(rom, 0, 0xea00002e);
  rom.set(NINTENDO_LOGO, 4);
  rom.set(safeRomTitle(title), 0xa0);
  rom.set(asciiBytes("GM05"), 0xac);
  rom.set(asciiBytes("01"), 0xb0);
  rom[0xb2] = 0x96;
  rom.fill(0, 0xb3, 0xbd);
  let sum = 0;
  for (let i = 0xa0; i < 0xbd; i += 1) sum += rom[i];
  rom[0xbd] = (-0x19 - sum) & 0xff;
  rom[0xbe] = 0;
  rom[0xbf] = 0;
}

function newColorBox(points) {
  let total = 0;
  let minR = 31, maxR = 0, minG = 31, maxG = 0, minB = 31, maxB = 0;
  for (const point of points) {
    total += point.count;
    if (point.r < minR) minR = point.r;
    if (point.r > maxR) maxR = point.r;
    if (point.g < minG) minG = point.g;
    if (point.g > maxG) maxG = point.g;
    if (point.b < minB) minB = point.b;
    if (point.b > maxB) maxB = point.b;
  }
  return { points, total, minR, maxR, minG, maxG, minB, maxB };
}

function boxScore(box) {
  const range = Math.max(box.maxR - box.minR, box.maxG - box.minG, box.maxB - box.minB);
  return (range + 1) * box.total;
}

function splitColorBox(box) {
  if (box.points.length < 2) return null;
  const rr = box.maxR - box.minR;
  const gr = box.maxG - box.minG;
  const br = box.maxB - box.minB;
  let channel = "r";
  if (gr > rr && gr >= br) channel = "g";
  else if (br > rr && br > gr) channel = "b";

  const points = box.points.slice().sort((a, b) => a[channel] - b[channel] || a.index - b.index);
  const half = Math.floor(box.total / 2);
  let accumulated = 0;
  let split = 1;
  for (let i = 0; i < points.length - 1; i += 1) {
    accumulated += points[i].count;
    if (accumulated >= half) {
      split = i + 1;
      break;
    }
  }
  if (split <= 0 || split >= points.length) split = Math.floor(points.length / 2);
  return [newColorBox(points.slice(0, split)), newColorBox(points.slice(split))];
}

function quantizePalette(histogram) {
  const points = [];
  for (let index = 0; index < histogram.length; index += 1) {
    const count = histogram[index];
    if (count) {
      points.push({ index, count, r: index & 31, g: (index >>> 5) & 31, b: (index >>> 10) & 31 });
    }
  }
  if (!points.length) points.push({ index: 0, count: 1, r: 0, g: 0, b: 0 });

  const boxes = [newColorBox(points)];
  while (boxes.length < VIDEO_PALETTE_COLORS) {
    let best = -1;
    let bestScore = -1;
    for (let i = 0; i < boxes.length; i += 1) {
      if (boxes[i].points.length < 2) continue;
      const score = boxScore(boxes[i]);
      if (score > bestScore) {
        best = i;
        bestScore = score;
      }
    }
    if (best < 0) break;
    const split = splitColorBox(boxes[best]);
    if (!split) break;
    boxes[best] = split[0];
    boxes.push(split[1]);
  }

  const palette = new Int16Array(256 * 3);
  for (let i = 0; i < boxes.length; i += 1) {
    let rs = 0, gs = 0, bs = 0, total = 0;
    for (const point of boxes[i].points) {
      rs += point.r * point.count;
      gs += point.g * point.count;
      bs += point.b * point.count;
      total += point.count;
    }
    if (total) {
      palette[i * 3] = Math.floor((rs + total / 2) / total);
      palette[i * 3 + 1] = Math.floor((gs + total / 2) / total);
      palette[i * 3 + 2] = Math.floor((bs + total / 2) / total);
    }
  }

  // Reserved player UI colours.
  const reserved = [
    [0, 0, 0], [6, 6, 6], [31, 31, 31], [31, 27, 0], [31, 0, 0], [0, 31, 0],
  ];
  for (let i = 0; i < reserved.length; i += 1) {
    const index = 250 + i;
    palette[index * 3] = reserved[i][0];
    palette[index * 3 + 1] = reserved[i][1];
    palette[index * 3 + 2] = reserved[i][2];
  }
  return palette;
}

function createPaletteLookup(palette, report) {
  const lookup = new Uint8Array(32768);
  for (let index = 0; index < lookup.length; index += 1) {
    const r = index & 31;
    const g = (index >>> 5) & 31;
    const b = (index >>> 10) & 31;
    let best = 0;
    let bestDistance = Number.MAX_SAFE_INTEGER;
    for (let j = 0; j < VIDEO_PALETTE_COLORS; j += 1) {
      const dr = r - palette[j * 3];
      const dg = g - palette[j * 3 + 1];
      const db = b - palette[j * 3 + 2];
      const distance = dr * dr + dg * dg + db * db;
      if (distance < bestDistance) {
        best = j;
        bestDistance = distance;
        if (distance === 0) break;
      }
    }
    lookup[index] = best;
    if (report && index % 4096 === 0) report(index / lookup.length);
  }
  return lookup;
}

function quantizeFrame(src, dst, palette, lookup, mode, errorCurrent, errorNext) {
  if (mode === "error") {
    errorCurrent.fill(0);
    errorNext.fill(0);
    for (let y = 0; y < FRAME_HEIGHT; y += 1) {
      errorNext.fill(0);
      for (let x = 0; x < FRAME_WIDTH; x += 1) {
        const source = (y * FRAME_WIDTH + x) * 3;
        const error = (x + 1) * 3;
        const r = clamp(src[source] + Math.trunc(errorCurrent[error] / 16), 0, 255);
        const g = clamp(src[source + 1] + Math.trunc(errorCurrent[error + 1] / 16), 0, 255);
        const b = clamp(src[source + 2] + Math.trunc(errorCurrent[error + 2] / 16), 0, 255);
        const r5 = Math.floor((r * 31 + 127) / 255);
        const g5 = Math.floor((g * 31 + 127) / 255);
        const b5 = Math.floor((b * 31 + 127) / 255);
        const paletteIndex = lookup[r5 | (g5 << 5) | (b5 << 10)];
        dst[y * FRAME_WIDTH + x] = paletteIndex;

        const er = r - Math.floor((palette[paletteIndex * 3] * 255) / 31);
        const eg = g - Math.floor((palette[paletteIndex * 3 + 1] * 255) / 31);
        const eb = b - Math.floor((palette[paletteIndex * 3 + 2] * 255) / 31);
        errorCurrent[error + 3] += er * 7;
        errorCurrent[error + 4] += eg * 7;
        errorCurrent[error + 5] += eb * 7;
        errorNext[error - 3] += er * 3;
        errorNext[error - 2] += eg * 3;
        errorNext[error - 1] += eb * 3;
        errorNext[error] += er * 5;
        errorNext[error + 1] += eg * 5;
        errorNext[error + 2] += eb * 5;
        errorNext[error + 3] += er;
        errorNext[error + 4] += eg;
        errorNext[error + 5] += eb;
      }
      const swap = errorCurrent;
      errorCurrent = errorNext;
      errorNext = swap;
    }
    return;
  }

  for (let y = 0; y < FRAME_HEIGHT; y += 1) {
    for (let x = 0; x < FRAME_WIDTH; x += 1) {
      const source = (y * FRAME_WIDTH + x) * 3;
      let r = Math.floor((src[source] * 31 + 127) / 255);
      let g = Math.floor((src[source + 1] * 31 + 127) / 255);
      let b = Math.floor((src[source + 2] * 31 + 127) / 255);
      if (mode === "ordered") {
        const delta = Math.trunc((BAYER_4X4[(y & 3) * 4 + (x & 3)] - 7) / 4);
        r = clamp(r + delta, 0, 31);
        g = clamp(g + delta, 0, 31);
        b = clamp(b + delta, 0, 31);
      }
      dst[y * FRAME_WIDTH + x] = lookup[r | (g << 5) | (b << 10)];
    }
  }
}

function paletteToBytes(palette) {
  const bytes = new Uint8Array(256 * 2);
  for (let i = 0; i < 256; i += 1) {
    const colour = palette[i * 3] | (palette[i * 3 + 1] << 5) | (palette[i * 3 + 2] << 10);
    setU16(bytes, i * 2, colour);
  }
  return bytes;
}

function appendU16(array, value) {
  array.push(value & 0xff, (value >>> 8) & 0xff);
}

function encodeDelta(previous, current) {
  const output = [];
  let position = 0;
  while (position < current.length) {
    let skip = 0;
    while (position + skip < current.length && previous[position + skip] === current[position + skip] && skip < 65535) skip += 1;
    position += skip;
    if (position >= current.length) {
      appendU16(output, skip);
      appendU16(output, 0);
      break;
    }
    const start = position;
    let unchanged = 0;
    while (position < current.length && position - start < 65535) {
      if (previous[position] === current[position]) unchanged += 1;
      else unchanged = 0;
      position += 1;
      if (unchanged >= 4) {
        position -= unchanged;
        break;
      }
    }
    const run = position - start;
    appendU16(output, skip);
    appendU16(output, run);
    for (let i = start; i < position; i += 1) output.push(current[i]);
  }
  return Uint8Array.from(output);
}

function makeRecord(type, payload) {
  const total = 8 + payload.length;
  const padded = (total + 3) & ~3;
  const record = new Uint8Array(padded);
  setU32(record, 0, type);
  setU32(record, 4, payload.length);
  record.set(payload, 8);
  return record;
}

function concatenate(parts, totalLength) {
  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const part of parts) {
    result.set(part, offset);
    offset += part.length;
  }
  return result;
}

function buildHistogram(framesRGB, startFrame, endFrame) {
  const histogram = new Uint32Array(32768);
  const frameCount = Math.max(1, endFrame - startFrame);
  const samples = Math.min(frameCount, 60);
  for (let sample = 0; sample < samples; sample += 1) {
    const relative = samples === 1 ? 0 : Math.round((sample * (frameCount - 1)) / (samples - 1));
    const frame = startFrame + relative;
    const start = frame * RGB_FRAME_BYTES;
    const end = start + RGB_FRAME_BYTES;
    for (let i = start; i < end; i += 3) {
      const r = Math.floor((framesRGB[i] * 31 + 127) / 255);
      const g = Math.floor((framesRGB[i + 1] * 31 + 127) / 255);
      const b = Math.floor((framesRGB[i + 2] * 31 + 127) / 255);
      histogram[r | (g << 5) | (b << 10)] += 1;
    }
  }
  return histogram;
}

function detectSceneStarts(framesRGB, frameCount) {
  if (frameCount <= 1) return [0];
  const starts = [0];
  const signatureLength = 10 * 15 * 3;
  const previous = new Uint8Array(signatureLength);
  const current = new Uint8Array(signatureLength);
  let lastStart = 0;
  let havePrevious = false;

  for (let frame = 0; frame < frameCount; frame += 1) {
    const frameStart = frame * RGB_FRAME_BYTES;
    let position = 0;
    for (let y = 4; y < FRAME_HEIGHT; y += 8) {
      for (let x = 4; x < FRAME_WIDTH; x += 8) {
        const source = frameStart + (y * FRAME_WIDTH + x) * 3;
        current[position++] = framesRGB[source];
        current[position++] = framesRGB[source + 1];
        current[position++] = framesRGB[source + 2];
      }
    }
    if (havePrevious) {
      let difference = 0;
      for (let index = 0; index < signatureLength; index += 1) {
        difference += Math.abs(current[index] - previous[index]);
      }
      const average = Math.floor(difference / signatureLength);
      if ((frame - lastStart >= 10 && average >= 42) || frame - lastStart >= 120) {
        starts.push(frame);
        lastStart = frame;
      }
    }
    previous.set(current);
    havePrevious = true;
  }
  return starts;
}

function frameMetrics(frame, previous) {
  let motion = 0, detail = 0, brightness = 0, changed = 0;
  for (let y = 0; y < FRAME_HEIGHT; y += 1) {
    for (let x = 0; x < FRAME_WIDTH; x += 1) {
      const i = (y * FRAME_WIDTH + x) * 3;
      const luma = (77 * frame[i] + 150 * frame[i + 1] + 29 * frame[i + 2]) / 256;
      brightness += luma;
      if (x > 0) {
        const j = i - 3;
        const other = (77 * frame[j] + 150 * frame[j + 1] + 29 * frame[j + 2]) / 256;
        detail += Math.abs(luma - other);
      }
      if (y > 0) {
        const j = i - FRAME_WIDTH * 3;
        const other = (77 * frame[j] + 150 * frame[j + 1] + 29 * frame[j + 2]) / 256;
        detail += Math.abs(luma - other);
      }
      if (previous) {
        const other = (77 * previous[i] + 150 * previous[i + 1] + 29 * previous[i + 2]) / 256;
        const difference = Math.abs(luma - other);
        motion += difference;
        if (difference > 48) changed += 1;
      }
    }
  }
  const pixels = FRAME_BYTES;
  return {
    motion: Math.min(1, motion / pixels / 64),
    detail: Math.min(1, detail / pixels / 2 / 64),
    brightness: brightness / pixels / 255,
    scene: changed / pixels,
  };
}

function detectSceneStartsEnhanced(framesRGB, frameCount) {
  if (frameCount <= 1) return [0];
  const metrics = [];
  let previous = null;
  for (let frame = 0; frame < frameCount; frame += 1) {
    const start = frame * RGB_FRAME_BYTES;
    const current = framesRGB.subarray(start, start + RGB_FRAME_BYTES);
    metrics.push(frameMetrics(current, previous));
    previous = current;
  }
  const starts = [0];
  let lastStart = 0;
  for (let frame = 1; frame < frameCount; frame += 1) {
    const current = metrics[frame], prior = metrics[frame - 1], next = metrics[frame + 1];
    const brightnessJump = Math.abs(current.brightness - prior.brightness);
    let hardCut = current.scene >= 0.34 && current.motion >= 0.42 && ((next?.scene || 0) >= 0.08 || brightnessJump < 0.55);
    const fadeBoundary = (current.brightness < 0.07 && prior.brightness >= 0.16) || (prior.brightness < 0.07 && current.brightness >= 0.16);
    if (brightnessJump > 0.58 && (next?.scene || 0) < 0.08) hardCut = false;
    const gap = frame - lastStart;
    if (gap >= 10 && (hardCut || fadeBoundary || gap >= 180)) {
      starts.push(frame);
      lastStart = frame;
    }
  }
  return starts;
}

function alignAudio(audioInput, frameCount, vblanks) {
  const displaySeconds = (frameCount * vblanks) / GBA_REFRESH;
  const required = Math.ceil(displaySeconds * AUDIO_RATE);
  const aligned = Math.ceil(required / 16) * 16;
  const audio = new Uint8Array(aligned);
  audio.set(audioInput.subarray(0, aligned));
  return audio;
}


export function rgb24ToNativeRGB555(rgb) {
  if (!(rgb instanceof Uint8Array)) throw new TypeError("rgb must be a Uint8Array");
  if (rgb.length !== NATIVE_IMAGE_RGB_BYTES) {
    throw new Error(`Native image RGB data is ${rgb.length} bytes; expected ${NATIVE_IMAGE_RGB_BYTES}.`);
  }
  const out = new Uint8Array(NATIVE_IMAGE_BYTES);
  for (let pixel = 0; pixel < NATIVE_IMAGE_PIXELS; pixel += 1) {
    const source = pixel * 3;
    const r5 = Math.floor((rgb[source] * 31 + 127) / 255);
    const g5 = Math.floor((rgb[source + 1] * 31 + 127) / 255);
    const b5 = Math.floor((rgb[source + 2] * 31 + 127) / 255);
    const value = r5 | (g5 << 5) | (b5 << 10);
    out[pixel * 2] = value & 0xff;
    out[pixel * 2 + 1] = value >>> 8;
  }
  return out;
}

function audioStorage(audio, frameCount, vblanks, audioCodec) {
  const hasAudio = audio instanceof Uint8Array && audio.length > 0;
  const alignedPCM = hasAudio ? alignAudio(audio, frameCount, vblanks) : new Uint8Array();
  let storedAudio = alignedPCM;
  let resolvedAudioCodec = hasAudio ? "pcm" : "none";
  let audioBlockSamples = 0;
  let audioBlockBytes = 0;
  if (hasAudio && audioCodec === "adpcm") {
    const encoded = encodeIMAADPCM(alignedPCM, DEFAULT_ADPCM_BLOCK_SAMPLES);
    storedAudio = encoded.data;
    resolvedAudioCodec = "adpcm";
    audioBlockSamples = encoded.blockSamples;
    audioBlockBytes = encoded.blockBytes;
  }
  return { hasAudio, alignedPCM, storedAudio, resolvedAudioCodec, audioBlockSamples, audioBlockBytes };
}

export function convertNativeMediaClip({
  mediaKind,
  nativeRGB,
  audio = new Uint8Array(),
  title = "GBA MEDIA",
  musicTitle = "",
  artist = "",
  album = "",
  durationSeconds = 0,
  imageSeconds = 5,
  vblanks = 5,
  audioCodec = "pcm",
  seekSeconds = 5,
  loop = false,
  report,
}) {
  if (mediaKind !== "audio" && mediaKind !== "image") throw new Error("Native media kind must be audio or image.");
  report?.(0.08, mediaKind === "audio" ? "Converting 240×160 artwork…" : "Converting native 240×160 image…");
  const video = rgb24ToNativeRGB555(nativeRGB);
  if (mediaKind === "image") {
    const seconds = Math.max(0, Number(imageSeconds) || 0);
    report?.(1, "Native image ready.");
    return {
      mediaKind: "image", title, frameCount: 1, vblanks, keyInterval: 0, adaptiveKeyframes: false,
      seekSeconds, loop: Boolean(loop), hasAudio: false, audioCodec: "none", audioSampleCount: 0,
      audioBlockSamples: 0, audioBlockBytes: 0, compressed: false,
      palette: new Uint8Array(), paletteIndex: new Uint8Array(), paletteCount: 0, videoIndex: new Uint8Array(),
      video, audio: new Uint8Array(), rawVideoSize: NATIVE_IMAGE_BYTES, storedVideoSize: NATIVE_IMAGE_BYTES,
      imageSeconds: seconds, mediaMetadata: false, musicTitle: "", artist: "", album: "",
    };
  }

  const displaySeconds = Math.max(0, Number(durationSeconds) || 0);
  let frameCount = Math.ceil((displaySeconds * GBA_REFRESH) / Math.max(1, Number(vblanks) || 5));
  if (frameCount < 1) frameCount = 1;
  report?.(0.35, "Preparing audio storage…");
  const storage = audioStorage(audio, frameCount, vblanks, audioCodec);
  if (!storage.hasAudio) throw new Error("Audio media contains no decodable audio stream.");
  report?.(1, "Audio media ready.");
  return {
    mediaKind: "audio", title, musicTitle, artist, album, mediaMetadata: true,
    frameCount, vblanks, keyInterval: 0, adaptiveKeyframes: false, seekSeconds, loop: Boolean(loop),
    hasAudio: true, audioCodec: storage.resolvedAudioCodec, audioSampleCount: storage.alignedPCM.length,
    audioBlockSamples: storage.audioBlockSamples, audioBlockBytes: storage.audioBlockBytes, compressed: false,
    palette: new Uint8Array(), paletteIndex: new Uint8Array(), paletteCount: 0, videoIndex: new Uint8Array(),
    video, audio: storage.storedAudio, rawVideoSize: NATIVE_IMAGE_BYTES, storedVideoSize: NATIVE_IMAGE_BYTES,
    imageSeconds: 0,
  };
}

export function convertRawClip({
  framesRGB,
  audio = new Uint8Array(),
  title,
  vblanks,
  paletteMode = "shared",
  ditherMode = "ordered",
  compression = "delta",
  keyInterval = 30,
  adaptiveKeyframes = false,
  enhancedSceneDetection = false,
  audioCodec = "pcm",
  seekSeconds = 5,
  loop = false,
  report,
}) {
  if (!(framesRGB instanceof Uint8Array)) throw new TypeError("framesRGB must be a Uint8Array");
  if (framesRGB.length % RGB_FRAME_BYTES !== 0) throw new Error("FFmpeg produced an incomplete frame stream.");
  const frameCount = framesRGB.length / RGB_FRAME_BYTES;
  if (frameCount < 1) throw new Error("No video frames were produced.");
  if (paletteMode !== "shared" && paletteMode !== "scene") throw new Error("Invalid palette mode.");
  if (compression !== "delta" && compression !== "none") throw new Error("Invalid compression mode.");

  report?.(0.02, paletteMode === "scene" ? "Detecting scene changes…" : "Building the shared video palette…");
  const sceneStarts = paletteMode === "scene" ? (enhancedSceneDetection ? detectSceneStartsEnhanced(framesRGB, frameCount) : detectSceneStarts(framesRGB, frameCount)) : [0];
  const paletteCount = sceneStarts.length;
  const frameScene = new Uint16Array(frameCount);
  const palettes = [];
  const lookups = [];
  const paletteParts = [];

  for (let scene = 0; scene < paletteCount; scene += 1) {
    const sceneStart = sceneStarts[scene];
    const sceneEnd = scene + 1 < paletteCount ? sceneStarts[scene + 1] : frameCount;
    for (let frame = sceneStart; frame < sceneEnd; frame += 1) frameScene[frame] = scene;
    report?.(0.04 + (scene / paletteCount) * 0.22, `Building palette ${scene + 1} of ${paletteCount}…`);
    const palette = quantizePalette(buildHistogram(framesRGB, sceneStart, sceneEnd));
    const lookup = createPaletteLookup(palette, (fraction) => {
      const sceneProgress = (scene + fraction) / paletteCount;
      report?.(0.1 + sceneProgress * 0.18, `Preparing palette ${scene + 1} of ${paletteCount}…`);
    });
    palettes.push(palette);
    lookups.push(lookup);
    paletteParts.push(paletteToBytes(palette));
  }

  const paletteBytes = concatenate(paletteParts, paletteParts.reduce((sum, part) => sum + part.length, 0));
  const paletteIndex = paletteCount > 1 ? new Uint8Array(frameCount * 2) : new Uint8Array();
  if (paletteCount > 1) {
    for (let frame = 0; frame < frameCount; frame += 1) setU16(paletteIndex, frame * 2, frameScene[frame]);
  }

  const compressed = compression === "delta";
  const videoIndex = compressed ? new Uint8Array(frameCount * 4) : new Uint8Array();
  const videoParts = [];
  let videoLength = 0;
  const previous = new Uint8Array(FRAME_BYTES);
  const current = new Uint8Array(FRAME_BYTES);
  const errorCurrent = new Int32Array((FRAME_WIDTH + 2) * 3);
  const errorNext = new Int32Array((FRAME_WIDTH + 2) * 3);
  let lastKeyframe = 0;
  let changeBudget = 0;

  for (let frame = 0; frame < frameCount; frame += 1) {
    const sourceStart = frame * RGB_FRAME_BYTES;
    const scene = frameScene[frame];
    quantizeFrame(
      framesRGB.subarray(sourceStart, sourceStart + RGB_FRAME_BYTES),
      current,
      palettes[scene],
      lookups[scene],
      ditherMode,
      errorCurrent,
      errorNext,
    );

    if (compressed) {
      setU32(videoIndex, frame * 4, videoLength);
      let type = 0;
      let payload = current;
      if (frame > 0) {
        const delta = encodeDelta(previous, current);
        const sceneBoundary = sceneStarts.includes(frame);
        const fixedKey = frame % Math.max(1, keyInterval) === 0;
        const maxAdaptive = Math.max(60, Math.min(150, Math.max(1, keyInterval) * 3));
        const distance = frame - lastKeyframe;
        let changed = 0;
        if (adaptiveKeyframes) {
          for (let i = 0; i < current.length; i += 1) if (current[i] !== previous[i]) changed += 1;
          changeBudget += changed;
        }
        const forceKey = adaptiveKeyframes
          ? sceneBoundary || distance >= maxAdaptive || (distance >= 8 && delta.length > FRAME_BYTES * 82 / 100) || (distance >= keyInterval && changeBudget > FRAME_BYTES * 5)
          : fixedKey;
        if (!forceKey && delta.length < current.length) {
          type = 1;
          payload = delta;
        } else {
          lastKeyframe = frame;
          changeBudget = 0;
        }
      } else {
        lastKeyframe = 0;
      }
      const record = makeRecord(type, payload);
      videoParts.push(record);
      videoLength += record.length;
    } else {
      const rawFrame = current.slice();
      videoParts.push(rawFrame);
      videoLength += rawFrame.length;
    }

    previous.set(current);
    if (frame % 8 === 0 || frame + 1 === frameCount) {
      report?.(0.3 + ((frame + 1) / frameCount) * 0.6, `Encoding frame ${frame + 1} of ${frameCount}…`);
    }
  }

  const storage = audioStorage(audio, frameCount, vblanks, audioCodec);
  const { hasAudio, alignedPCM, storedAudio, resolvedAudioCodec, audioBlockSamples, audioBlockBytes } = storage;
  report?.(0.94, "Finalizing clip data…");
  return {
    mediaKind: "video",
    title,
    frameCount,
    vblanks,
    keyInterval: adaptiveKeyframes ? 0 : keyInterval,
    adaptiveKeyframes,
    seekSeconds,
    loop,
    hasAudio,
    audioCodec: resolvedAudioCodec,
    audioSampleCount: alignedPCM.length,
    audioBlockSamples,
    audioBlockBytes,
    compressed,
    palette: paletteBytes,
    paletteIndex,
    paletteCount,
    videoIndex,
    video: concatenate(videoParts, videoLength),
    audio: storedAudio,
    rawVideoSize: frameCount * FRAME_BYTES,
    storedVideoSize: videoLength,
  };
}

function writeClipDescriptor(rom, descriptorOffset, clip, offsets) {
  let flags = 0;
  if (clip.hasAudio) flags |= 1;
  if (clip.loop) flags |= 2;
  if (clip.mediaKind === "video" && clip.compressed) flags |= 4;
  if (clip.paletteCount > 1) flags |= 8;
  if (clip.audioCodec === "adpcm") flags |= 16;
  if (clip.mediaKind === "video" && clip.adaptiveKeyframes) flags |= 32;
  if (clip.mediaKind === "audio") flags |= 64;
  if (clip.mediaKind === "image") flags |= 128;
  if (clip.mediaMetadata) flags |= 256;
  let seekFrames = Math.round((clip.seekSeconds * GBA_REFRESH) / Math.max(1, clip.vblanks));
  if (seekFrames < 1) seekFrames = 1;

  const native = clip.mediaKind === "audio" || clip.mediaKind === "image";
  setU32(rom, descriptorOffset, clip.frameCount);
  setU32(rom, descriptorOffset + 4, native ? NATIVE_IMAGE_BYTES : FRAME_BYTES);
  setU32(rom, descriptorOffset + 8, offsets.video || 0);
  setU32(rom, descriptorOffset + 12, offsets.videoIndex || 0);
  setU32(rom, descriptorOffset + 16, offsets.audio || 0);
  setU32(rom, descriptorOffset + 20, clip.audio.length);
  setU32(rom, descriptorOffset + 24, offsets.palette || 0);
  setU32(rom, descriptorOffset + 28, offsets.paletteIndex || 0);
  setU32(rom, descriptorOffset + 32, offsets.seek || 0);
  setU32(rom, descriptorOffset + 36, AUDIO_RATE);
  setU32(rom, descriptorOffset + 40, seekFrames);
  setU16(rom, descriptorOffset + 44, clip.vblanks);
  setU16(rom, descriptorOffset + 46, native ? NATIVE_IMAGE_WIDTH : FRAME_WIDTH);
  setU16(rom, descriptorOffset + 48, native ? NATIVE_IMAGE_HEIGHT : FRAME_HEIGHT);
  setU16(rom, descriptorOffset + 50, flags);
  setU16(rom, descriptorOffset + 52, clip.seekSeconds);
  setU16(rom, descriptorOffset + 54, clip.paletteCount || 0);
  setU16(rom, descriptorOffset + 56, clip.adaptiveKeyframes ? 0 : (clip.keyInterval || 0));
  rom.set(encodeGBATextFixed(clip.title, 12), descriptorOffset + 60);
  setU32(rom, descriptorOffset + 72, clip.rawVideoSize);
  setU32(rom, descriptorOffset + 76, clip.storedVideoSize);
  setU32(rom, descriptorOffset + 80, clip.hasAudio ? (clip.audioCodec === "adpcm" ? 2 : 1) : 0);
  const auxCount = clip.mediaKind === "image" ? Math.round(Math.max(0, Number(clip.imageSeconds) || 0) * 1000) : (clip.audioSampleCount || 0);
  setU32(rom, descriptorOffset + 84, auxCount);
  setU32(rom, descriptorOffset + 88, clip.audioBlockSamples || 0);
  setU32(rom, descriptorOffset + 92, clip.audioBlockBytes || 0);
}

const MEDIA_METADATA_MAGIC = 0x32444d4d; // MMD2
const MEDIA_METADATA_SIZE = 80;

export function encodeMediaMetadata(title = "", artist = "", album = "") {
  const data = new Uint8Array(MEDIA_METADATA_SIZE);
  setU32(data, 0, MEDIA_METADATA_MAGIC);
  data.set(encodeGBATextFixed(title, 28), 4);
  data.set(encodeGBATextFixed(artist, 28), 32);
  data.set(encodeGBATextFixed(album, 20), 60);
  return data;
}

function decodeMenuBytes(value) {
  if (value instanceof Uint8Array) return value;
  if (typeof value !== "string") throw new Error("Menu theme data is missing.");
  if (typeof atob === "function") {
    const raw = atob(value); const out = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i += 1) out[i] = raw.charCodeAt(i);
    return out;
  }
  return Uint8Array.from(Buffer.from(value, "base64"));
}

function appendMenuTheme(writer, theme) {
  if (!theme) return 0;
  const palette = decodeMenuBytes(theme.palette);
  const frames = (theme.frames || []).map(decodeMenuBytes);
  if (palette.length !== 512) throw new Error(`Menu palette is ${palette.length} bytes; expected 512.`);
  if (frames.length < 1 || frames.length > 16) throw new Error("Menu background must contain 1 to 16 optimized frames.");
  for (const [index, frame] of frames.entries()) {
    if (frame.length !== FRAME_BYTES) throw new Error(`Menu frame ${index + 1} is ${frame.length} bytes; expected ${FRAME_BYTES}.`);
    for (const colour of frame) if (colour >= 250) throw new Error(`Menu frame ${index + 1} uses reserved palette index ${colour}.`);
  }
  let kind = 0;
  if (theme.kind === "palette-shimmer") kind = 1;
  else if (theme.kind === "frames") kind = 2;
  else if (theme.kind !== "static") throw new Error("Unknown menu background type.");
  const frameVBlanks = Number(theme.frameVBlanks || 0);
  if (kind === 2 && (frameVBlanks < 6 || frameVBlanks > 120)) throw new Error("Animated menu frame interval must be between 6 and 120 VBlanks.");
  const shimmer = theme.shimmer || null;
  if (kind === 1 && !shimmer) throw new Error("Palette shimmer theme is missing its animation settings.");

  const headerOffset = writer.reserve(MENU_THEME_HEADER_SIZE);
  const paletteOffset = writer.writeAligned(palette);
  const framesOffset = writer.length;
  for (const frame of frames) writer.writeAligned(frame);
  const header = writer.buffer;
  setU32(header, headerOffset, MENU_THEME_MAGIC);
  setU16(header, headerOffset + 4, 1);
  setU16(header, headerOffset + 6, kind);
  setU32(header, headerOffset + 8, paletteOffset);
  setU32(header, headerOffset + 12, framesOffset);
  setU16(header, headerOffset + 16, frames.length);
  setU16(header, headerOffset + 18, frameVBlanks);
  setU16(header, headerOffset + 20, theme.outline ? 1 : 0);
  setU16(header, headerOffset + 22, Number(theme.uiColor || 0x7fff));
  setU16(header, headerOffset + 24, Number(theme.selectedColor || 0x037f));
  setU16(header, headerOffset + 26, Number(theme.outlineColor || 0));
  if (shimmer) {
    setU16(header, headerOffset + 28, Number(shimmer.sourceStart || 0));
    setU16(header, headerOffset + 30, Number(shimmer.count || 0));
    setU16(header, headerOffset + 32, Number(shimmer.target1 || 0));
    setU16(header, headerOffset + 34, Number(shimmer.interval1 || 0));
    setU16(header, headerOffset + 36, Number(shimmer.target2 || 0));
    setU16(header, headerOffset + 38, Number(shimmer.interval2 || 0));
    setU16(header, headerOffset + 40, Number(shimmer.phases || 0));
  }
  setU32(header, headerOffset + 44, FRAME_BYTES);
  setU32(header, headerOffset + 48, writer.length - headerOffset);
  return headerOffset;
}

function buildSeekTable(clip) {
  const seek = new Uint8Array(clip.frameCount * 4);
  for (let frame = 0; frame < clip.frameCount; frame += 1) {
    let value = Math.floor((frame * clip.vblanks * AUDIO_RATE) / GBA_REFRESH);
    if (clip.audioCodec === "adpcm") {
      if (clip.audioSampleCount > 0 && value >= clip.audioSampleCount) value = clip.audioSampleCount - 1;
    } else {
      value &= ~3;
      if (clip.audio.length >= 4 && value > clip.audio.length - 4) value = (clip.audio.length - 4) & ~3;
    }
    setU32(seek, frame * 4, value);
  }
  return seek;
}

export function assembleROM(playerStub, clips, { romTitle = "GBA VIDEO", outputMode = "menu", resume = true, titleScreenPart = 0, titleScreenName = "", titleCard = null, menuTheme = null } = {}) {
  if (!(playerStub instanceof Uint8Array)) throw new TypeError("playerStub must be a Uint8Array");
  if (playerStub.length !== ASSET_OFFSET) {
    throw new Error(`Player template is ${playerStub.length} bytes; expected ${ASSET_OFFSET}.`);
  }
  if (!clips.length) throw new Error("At least one clip is required.");

  const writer = new ByteWriter(Math.max(ROM_MIN_SIZE, playerStub.length + 1024 * 1024));
  writer.write(playerStub);
  const clipTableOffset = writer.reserve(clips.length * CLIP_DESCRIPTOR_SIZE);
  const menuThemeOffset = outputMode === "menu" && clips.length > 1 && menuTheme ? appendMenuTheme(writer, menuTheme) : 0;
  let titleCardOffset = 0;
  if (titleCard) {
    const asset = titleCard instanceof Uint8Array ? titleCard : new Uint8Array(titleCard);
    if (asset.length < 32) throw new Error("Title-card asset is incomplete.");
    titleCardOffset = writer.writeAligned(asset);
  }

  const clipOffsets = [];
  for (const clip of clips) {
    const offsets = {};
    if (clip.palette?.length) offsets.palette = writer.writeAligned(clip.palette);
    if (clip.paletteIndex?.length) offsets.paletteIndex = writer.writeAligned(clip.paletteIndex);
    if (clip.mediaKind === "audio" && clip.mediaMetadata) {
      offsets.videoIndex = writer.writeAligned(encodeMediaMetadata(clip.musicTitle, clip.artist, clip.album));
    } else if (clip.videoIndex?.length) {
      offsets.videoIndex = writer.writeAligned(clip.videoIndex);
    }
    offsets.video = writer.writeAligned(clip.video);
    if (clip.hasAudio) {
      offsets.seek = writer.writeAligned(buildSeekTable(clip));
      offsets.audio = writer.writeAligned(clip.audio);
    }
    clipOffsets.push(offsets);
  }

  if (writer.length > ROM_LIMIT) {
    throw new Error(`The ROM needs ${(writer.length / 1048576).toFixed(2)} MiB, exceeding the 32 MiB GBA limit.`);
  }

  const unpaddedSize = writer.length;
  const paddedSize = nextPowerOfTwo(unpaddedSize);
  if (paddedSize > ROM_LIMIT) throw new Error("The next cartridge size exceeds 32 MiB.");
  writer.reserve(paddedSize - writer.length);
  writer.buffer.fill(0xff, unpaddedSize, paddedSize);
  const rom = writer.finish();

  for (let index = 0; index < clips.length; index += 1) {
    writeClipDescriptor(rom, clipTableOffset + index * CLIP_DESCRIPTOR_SIZE, clips[index], clipOffsets[index]);
  }

  rom.set(asciiBytes("GBV5"), METADATA_OFFSET);
  setU16(rom, METADATA_OFFSET + 4, 5);
  let flags = resume ? 1 : 0;
  if (outputMode === "playlist") flags |= 2;
  if (titleScreenPart > 0 || titleCardOffset > 0) flags |= 4;
  setU16(rom, METADATA_OFFSET + 6, flags);
  setU16(rom, METADATA_OFFSET + 8, clips.length);
  setU32(rom, METADATA_OFFSET + 12, clipTableOffset);
  setU32(rom, METADATA_OFFSET + 16, CLIP_DESCRIPTOR_SIZE);
  if (titleScreenPart > 0) {
    setU32(rom, METADATA_OFFSET + 20, titleScreenPart);
    rom.set(safeTitleScreenName(titleScreenName), METADATA_OFFSET + 24);
  }
  if (menuThemeOffset > 0) setU32(rom, METADATA_OFFSET + 48, menuThemeOffset);
  if (titleCardOffset > 0) setU32(rom, METADATA_OFFSET + 52, titleCardOffset);
  patchGBAHeader(rom, romTitle);

  return {
    rom,
    unpaddedSize,
    paddedSize,
    frameCount: clips.reduce((sum, clip) => sum + clip.frameCount, 0),
    clipCount: clips.length,
  };
}
