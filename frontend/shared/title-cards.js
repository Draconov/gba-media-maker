import { glyphBits, glyphLength, sanitizeGBAText } from "./gba-text.js";
const WIDTH = 240;
const HEIGHT = 160;
const HEADER_SIZE = 32;
const MAGIC = 0x31444354;
const GBA_REFRESH = 59.727500569606;

const FLAG_WAIT_A = 1;
const FLAG_SKIP = 2;
const FLAG_FADE = 4;

function setU16(data, offset, value) {
  data[offset] = value & 0xff;
  data[offset + 1] = (value >>> 8) & 0xff;
}
function setU32(data, offset, value) {
  data[offset] = value & 0xff;
  data[offset + 1] = (value >>> 8) & 0xff;
  data[offset + 2] = (value >>> 16) & 0xff;
  data[offset + 3] = (value >>> 24) & 0xff;
}

export function sanitizeTitleCardText(value, maximum = 40) {
  return sanitizeGBAText(value, maximum).text;
}

function sourceBaseName(sourceName) {
  return String(sourceName || "GBA VIDEO").replace(/^.*[\\/]/, "").replace(/\.[^.]+$/, "");
}

export function defaultTitleCardSettings(sourceName = "GBA VIDEO") {
  return {
    title: sanitizeTitleCardText(sourceBaseName(sourceName), 36) || "GBA VIDEO",
    subtitle: "Part {part}",
    backgroundMode: "part-first-frame",
    frameOffsetSeconds: 0,
    darkness: 50,
    solidColor: "#000000",
    textColor: "#FFFFFF",
    outlineColor: "#000000",
    alignment: "center",
    textSize: "large",
    titleTextColor: "#FFFFFF",
    titleOutlineColor: "#000000",
    titleAlignment: "center",
    titleTextSize: "large",
    subtitleTextColor: "#FFFFFF",
    subtitleOutlineColor: "#000000",
    subtitleAlignment: "center",
    subtitleTextSize: "small",
    drawOutline: true,
    startMode: "button",
    durationSeconds: 3,
    allowSkip: true,
    fade: true,
  };
}

export function createTitleCardProject(sourceName = "GBA VIDEO") {
  return { enabled: true, useShared: true, shared: defaultTitleCardSettings(sourceName), parts: [] };
}

function normalizeHex(value, fallback) {
  const match = /^#?([0-9a-f]{6})$/i.exec(String(value || "").trim());
  return `#${(match ? match[1] : fallback.replace("#", "")).toUpperCase()}`;
}

export function normalizeTitleCardSettings(value = {}, sourceName = "GBA VIDEO", part = 1) {
  const defaults = defaultTitleCardSettings(sourceName);
  const raw = value && typeof value === "object" ? value : {};
  const settings = { ...defaults, ...raw };
  settings.title = sanitizeTitleCardText(settings.title || defaults.title, 36) || defaults.title;
  settings.subtitle = sanitizeTitleCardText(String(settings.subtitle ?? defaults.subtitle).replaceAll("{part}", String(part)), 40);
  settings.backgroundMode = ["part-first-frame", "part-frame", "solid"].includes(settings.backgroundMode) ? settings.backgroundMode : defaults.backgroundMode;
  settings.frameOffsetSeconds = Math.max(0, Number(settings.frameOffsetSeconds) || 0);
  settings.darkness = Math.max(0, Math.min(90, Math.round(Number(settings.darkness) || 0)));
  settings.solidColor = normalizeHex(settings.solidColor, defaults.solidColor);
  settings.textColor = normalizeHex(settings.textColor, defaults.textColor);
  settings.outlineColor = normalizeHex(settings.outlineColor, defaults.outlineColor);
  settings.alignment = ["left", "right"].includes(settings.alignment) ? settings.alignment : "center";
  settings.textSize = ["medium", "small"].includes(settings.textSize) ? settings.textSize : "large";
  settings.titleTextColor = normalizeHex(raw.titleTextColor ?? raw.textColor ?? defaults.titleTextColor, defaults.titleTextColor);
  settings.titleOutlineColor = normalizeHex(raw.titleOutlineColor ?? raw.outlineColor ?? defaults.titleOutlineColor, defaults.titleOutlineColor);
  settings.subtitleTextColor = normalizeHex(raw.subtitleTextColor ?? raw.textColor ?? defaults.subtitleTextColor, defaults.subtitleTextColor);
  settings.subtitleOutlineColor = normalizeHex(raw.subtitleOutlineColor ?? raw.outlineColor ?? defaults.subtitleOutlineColor, defaults.subtitleOutlineColor);
  settings.titleAlignment = ["left", "right"].includes(raw.titleAlignment) ? raw.titleAlignment : (raw.titleAlignment === "center" ? "center" : settings.alignment);
  settings.subtitleAlignment = ["left", "right"].includes(raw.subtitleAlignment) ? raw.subtitleAlignment : (raw.subtitleAlignment === "center" ? "center" : settings.alignment);
  settings.titleTextSize = ["large", "medium", "small"].includes(raw.titleTextSize) ? raw.titleTextSize : settings.textSize;
  settings.subtitleTextSize = ["large", "medium", "small"].includes(raw.subtitleTextSize)
    ? raw.subtitleTextSize
    : (settings.textSize === "large" ? "medium" : "small");
  settings.drawOutline = settings.drawOutline !== false;
  settings.startMode = settings.startMode === "timer" ? "timer" : "button";
  settings.durationSeconds = Math.max(0.1, Math.min(60, Number(settings.durationSeconds) || defaults.durationSeconds));
  settings.allowSkip = settings.allowSkip !== false;
  settings.fade = settings.fade !== false;
  return settings;
}

export function resolveTitleCardSettings(project, sourceName, part) {
  if (!project?.enabled) return null;
  let selected = project.shared || {};
  if (!project.useShared) {
    const match = (project.parts || []).find((item) => Number(item.part) === Number(part));
    if (match) selected = match.settings || match;
  }
  return normalizeTitleCardSettings(selected, sourceName, part);
}

function hexRGB(value, fallback) {
  const hex = normalizeHex(value, fallback).slice(1);
  return [Number.parseInt(hex.slice(0, 2), 16), Number.parseInt(hex.slice(2, 4), 16), Number.parseInt(hex.slice(4, 6), 16)];
}

function rgb555(r, g, b) {
  const r5 = Math.floor((r * 31 + 127) / 255);
  const g5 = Math.floor((g * 31 + 127) / 255);
  const b5 = Math.floor((b * 31 + 127) / 255);
  return r5 | (g5 << 5) | (b5 << 10);
}

function rgb555ToRGBA(value, rgba, offset) {
  rgba[offset] = Math.round((value & 31) * 255 / 31);
  rgba[offset + 1] = Math.round(((value >>> 5) & 31) * 255 / 31);
  rgba[offset + 2] = Math.round(((value >>> 10) & 31) * 255 / 31);
  rgba[offset + 3] = 255;
}

function wrapText(text, maxChars, maxLines) {
  const words = String(text || "").trim().split(/\s+/u).filter(Boolean);
  const lines = []; let current = "";
  for (let word of words) {
    let chars = Array.from(word);
    while (chars.length > maxChars) {
      if (current) { lines.push(current); current = ""; if (lines.length >= maxLines) return lines; }
      lines.push(chars.slice(0, maxChars).join("")); chars = chars.slice(maxChars);
      if (lines.length >= maxLines) return lines;
    }
    word = chars.join("");
    const candidate = current ? `${current} ${word}` : word;
    if (glyphLength(candidate) <= maxChars) current = candidate;
    else { if (current) lines.push(current); if (lines.length >= maxLines) return lines; current = word; }
  }
  if (current && lines.length < maxLines) lines.push(current);
  return lines;
}

function setPixel(pixels, x, y, colour) {
  if (x >= 0 && y >= 0 && x < WIDTH && y < HEIGHT) pixels[y * WIDTH + x] = colour;
}
function drawGlyph(pixels, x, y, scale, glyph, colour) {
  for (let row = 0; row < 5; row += 1) for (let col = 0; col < 3; col += 1) {
    const bit = 14 - (row * 3 + col);
    if (!(glyph & (1 << bit))) continue;
    for (let yy = 0; yy < scale; yy += 1) for (let xx = 0; xx < scale; xx += 1) setPixel(pixels, x + col * scale + xx, y + row * scale + yy, colour);
  }
}
function lineX(line, scale, alignment) {
  const length = glyphLength(line);
  const width = length ? length * 4 * scale - scale : 0;
  if (alignment === "left") return 12;
  if (alignment === "right") return WIDTH - 12 - width;
  return Math.floor((WIDTH - width) / 2);
}
function drawLine(pixels, line, y, scale, alignment, colour, outlineColour, outline) {
  const x = lineX(line, scale, alignment);
  if (outline) {
    const radius = scale >= 3 ? 2 : 1;
    for (const [ox, oy] of [[-radius,0],[radius,0],[0,-radius],[0,radius],[-1,-1],[1,-1],[-1,1],[1,1]]) {
      [...line].forEach((character, index) => drawGlyph(pixels, x + index * 4 * scale + ox, y + oy, scale, glyphBits(character), outlineColour));
    }
  }
  [...line].forEach((character, index) => drawGlyph(pixels, x + index * 4 * scale, y, scale, glyphBits(character), colour));
}

function textStyle(size) {
  if (size === "medium") return { scale: 3, maxChars: 19, lineHeight: 18 };
  if (size === "small") return { scale: 2, maxChars: 29, lineHeight: 12 };
  return { scale: 4, maxChars: 14, lineHeight: 24 };
}
function typographyGap(titleSize, subtitleSize) {
  if (titleSize === "large" || subtitleSize === "large") return 10;
  if (titleSize === "medium" || subtitleSize === "medium") return 8;
  return 6;
}

export function renderTitleCardPixels(backgroundRGB, rawSettings, part = 1, sourceName = "GBA VIDEO") {
  if (!(backgroundRGB instanceof Uint8Array) || backgroundRGB.length !== WIDTH * HEIGHT * 3) throw new Error(`Title-card background must contain ${WIDTH * HEIGHT * 3} RGB bytes.`);
  const settings = normalizeTitleCardSettings(rawSettings, sourceName, part);
  const pixels = new Uint16Array(WIDTH * HEIGHT);
  const factor = (100 - settings.darkness) / 100;
  for (let index = 0; index < pixels.length; index += 1) {
    pixels[index] = rgb555(
      Math.round(backgroundRGB[index * 3] * factor),
      Math.round(backgroundRGB[index * 3 + 1] * factor),
      Math.round(backgroundRGB[index * 3 + 2] * factor),
    );
  }
  const titleColour = rgb555(...hexRGB(settings.titleTextColor, "#FFFFFF"));
  const titleOutlineColour = rgb555(...hexRGB(settings.titleOutlineColor, "#000000"));
  const subtitleColour = rgb555(...hexRGB(settings.subtitleTextColor, "#FFFFFF"));
  const subtitleOutlineColour = rgb555(...hexRGB(settings.subtitleOutlineColor, "#000000"));
  const titleType = textStyle(settings.titleTextSize);
  const subtitleType = textStyle(settings.subtitleTextSize);
  const gap = typographyGap(settings.titleTextSize, settings.subtitleTextSize);
  const titleLines = wrapText(settings.title, titleType.maxChars, 2);
  const subtitleLines = wrapText(settings.subtitle, subtitleType.maxChars, 2);
  const totalHeight = titleLines.length * titleType.lineHeight + (subtitleLines.length ? gap + subtitleLines.length * subtitleType.lineHeight : 0);
  let y = Math.max(10, Math.floor((HEIGHT - totalHeight) / 2));
  for (const line of titleLines) { drawLine(pixels, line, y, titleType.scale, settings.titleAlignment, titleColour, titleOutlineColour, settings.drawOutline); y += titleType.lineHeight; }
  if (subtitleLines.length) {
    y += gap;
    for (const line of subtitleLines) { drawLine(pixels, line, y, subtitleType.scale, settings.subtitleAlignment, subtitleColour, subtitleOutlineColour, settings.drawOutline); y += subtitleType.lineHeight; }
  }
  return { pixels, settings };
}

export function buildTitleCardAsset(backgroundRGB, rawSettings, part = 1, sourceName = "GBA VIDEO") {
  const { pixels, settings } = renderTitleCardPixels(backgroundRGB, rawSettings, part, sourceName);
  const asset = new Uint8Array(HEADER_SIZE + pixels.length * 2);
  setU32(asset, 0, MAGIC);
  setU16(asset, 4, 1);
  let flags = settings.startMode === "button" ? FLAG_WAIT_A : 0;
  if (settings.allowSkip) flags |= FLAG_SKIP;
  if (settings.fade) flags |= FLAG_FADE;
  setU16(asset, 6, flags);
  setU32(asset, 8, pixels.length * 2);
  setU32(asset, 12, Math.max(1, Math.round(settings.durationSeconds * GBA_REFRESH)));
  const view = new DataView(asset.buffer);
  pixels.forEach((value, index) => view.setUint16(HEADER_SIZE + index * 2, value, true));
  return asset;
}

function sourceToRGB(source, fitMode = "fit") {
  const canvas = document.createElement("canvas");
  canvas.width = WIDTH;
  canvas.height = HEIGHT;
  const context = canvas.getContext("2d", { willReadFrequently: true });
  context.imageSmoothingEnabled = true;
  context.fillStyle = "#000";
  context.fillRect(0, 0, WIDTH, HEIGHT);
  const sourceWidth = source.videoWidth || source.naturalWidth || source.width || WIDTH;
  const sourceHeight = source.videoHeight || source.naturalHeight || source.height || HEIGHT;
  if (fitMode === "stretch") context.drawImage(source, 0, 0, WIDTH, HEIGHT);
  else {
    const scale = fitMode === "crop" ? Math.max(WIDTH / sourceWidth, HEIGHT / sourceHeight) : Math.min(WIDTH / sourceWidth, HEIGHT / sourceHeight);
    const width = sourceWidth * scale;
    const height = sourceHeight * scale;
    context.drawImage(source, (WIDTH - width) / 2, (HEIGHT - height) / 2, width, height);
  }
  const rgba = context.getImageData(0, 0, WIDTH, HEIGHT).data;
  const rgb = new Uint8Array(WIDTH * HEIGHT * 3);
  for (let index = 0; index < WIDTH * HEIGHT; index += 1) {
    rgb[index * 3] = rgba[index * 4];
    rgb[index * 3 + 1] = rgba[index * 4 + 1];
    rgb[index * 3 + 2] = rgba[index * 4 + 2];
  }
  return rgb;
}

export function renderTitleCardPreview(canvas, source, fitMode, rawSettings, part = 1, sourceName = "GBA VIDEO") {
  if (!canvas || !source) return;
  canvas.width = WIDTH;
  canvas.height = HEIGHT;
  const settings = normalizeTitleCardSettings(rawSettings, sourceName, part);
  let rgb;
  if (settings.backgroundMode === "solid") {
    const [r, g, b] = hexRGB(settings.solidColor, "#000000");
    rgb = new Uint8Array(WIDTH * HEIGHT * 3);
    for (let index = 0; index < WIDTH * HEIGHT; index += 1) { rgb[index * 3] = r; rgb[index * 3 + 1] = g; rgb[index * 3 + 2] = b; }
  } else rgb = sourceToRGB(source, fitMode);
  const { pixels } = renderTitleCardPixels(rgb, settings, part, sourceName);
  const context = canvas.getContext("2d");
  context.imageSmoothingEnabled = false;
  const image = context.createImageData(WIDTH, HEIGHT);
  pixels.forEach((value, index) => rgb555ToRGBA(value, image.data, index * 4));
  context.putImageData(image, 0, 0);
}

export const TITLE_CARD_BYTES = HEADER_SIZE + WIDTH * HEIGHT * 2;
export const TITLE_CARD_WIDTH = WIDTH;
export const TITLE_CARD_HEIGHT = HEIGHT;
