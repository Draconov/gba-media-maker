import { FFmpeg } from "@ffmpeg/ffmpeg";
import { fetchFile, toBlobURL } from "@ffmpeg/util";
import { AUDIO_RATE, GBA_REFRESH, RGB_FRAME_BYTES } from "./rom-core.js";
import { buildStoredZip } from "./zip-store.js";
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
  outputMode: document.querySelector("#outputMode"),
  vblanks: document.querySelector("#vblanks"),
  fitMode: document.querySelector("#fitMode"),
  paletteMode: document.querySelector("#paletteMode"),
  ditherMode: document.querySelector("#ditherMode"),
  compression: document.querySelector("#compression"),
  audioMode: document.querySelector("#audioMode"),
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
};

const PRESETS = {
  best: {
    vblanks: "4", compression: "delta", normalize: true, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "scene", ditherMode: "error",
  },
  balanced: {
    vblanks: "5", compression: "delta", normalize: false, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "shared", ditherMode: "ordered",
  },
  long: {
    vblanks: "8", compression: "delta", normalize: false, limiter: true,
    fitMode: "fit", audioMode: "mix", paletteMode: "shared", ditherMode: "ordered",
  },
  small: {
    vblanks: "8", compression: "delta", normalize: false, limiter: false,
    fitMode: "fit", audioMode: "none", paletteMode: "shared", ditherMode: "off",
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
  return file.name.replace(/\.[^.]+$/, "").replace(/[_-]+/g, " ").trim().slice(0, 12) || "VIDEO";
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
    volume: 1,
    loop: false,
    paletteMode: "shared",
    ditherMode: "ordered",
  };
}

function addFiles(fileList) {
  const existing = new Set(entries.map((entry) => `${entry.file.name}:${entry.file.size}:${entry.file.lastModified}`));
  for (const file of fileList) {
    if (!file.type.startsWith("video/") && !/\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(file.name)) continue;
    const key = `${file.name}:${file.size}:${file.lastModified}`;
    if (existing.has(key)) continue;
    entries.push(makeEntry(file));
    existing.add(key);
  }
  if (fileList.length) resetResult();
  renderFiles();
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
  });
  label.append(caption, select);
  return { label, select };
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

  entries.forEach((entry, index) => {
    const row = document.createElement("div");
    row.className = "file-row";

    const fileName = document.createElement("div");
    fileName.className = "file-name";
    const strong = document.createElement("strong");
    strong.textContent = entry.file.name;
    const small = document.createElement("small");
    small.textContent = formatBytes(entry.file.size);
    fileName.append(strong, small);

    const titleLabel = document.createElement("label");
    const titleText = document.createElement("span");
    titleText.textContent = "Clip-menu title";
    const titleInput = document.createElement("input");
    titleInput.type = "text";
    titleInput.maxLength = 12;
    titleInput.value = entry.title;
    titleInput.addEventListener("input", () => {
      entry.title = titleInput.value;
      resetResult();
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
      resetResult();
      renderFiles();
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
    const palette = makeSelectControl("Palette", entry.paletteMode, [["shared", "Shared palette"], ["scene", "Per-scene palette"]], (value) => { entry.paletteMode = value; });
    const dither = makeSelectControl("Dithering", entry.ditherMode, [["off", "Off"], ["ordered", "Ordered"], ["error", "Error diffusion"]], (value) => { entry.ditherMode = value; });

    controls.push(start.input, end.input, speed.input, volume.input, fit.select, audio.select, palette.select, dither.select);
    optionsGrid.append(start.label, end.label, speed.label, fit.label, audio.label, volume.label, palette.label, dither.label);

    const loopLabel = document.createElement("label");
    loopLabel.className = "clip-loop clip-option-check";
    const loopInput = document.createElement("input");
    loopInput.type = "checkbox";
    loopInput.checked = entry.loop;
    loopInput.addEventListener("change", () => {
      entry.loop = loopInput.checked;
      resetResult();
    });
    const loopText = document.createElement("span");
    loopText.textContent = "Loop playback";
    loopLabel.append(loopInput, loopText);
    optionsGrid.append(loopLabel);
    controls.push(loopInput);

    for (const control of controls) control.disabled = entry.useProject || conversionRunning;
    details.append(summary, optionsGrid);
    row.append(fileName, titleLabel, moveGroup, remove, details);
    elements.fileList.append(row);
  });
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

function currentOptions() {
  return {
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
  };
}

function effectiveClipOptions(entry, project) {
  if (!entry.useProject) {
    return {
      start: entry.start, end: entry.end, speed: entry.speed,
      fitMode: entry.fitMode, audioMode: entry.audioMode, volume: entry.volume,
      loop: entry.loop, paletteMode: entry.paletteMode, ditherMode: entry.ditherMode,
    };
  }
  return {
    start: project.defaultStart, end: project.defaultEnd, speed: project.defaultSpeed,
    fitMode: project.fitMode, audioMode: project.audioMode, volume: project.defaultVolume,
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
  resetResult();
}

function markPresetCustom() {
  elements.preset.value = "custom";
  resetResult();
}

function updateConvertButton() {
  elements.convertButton.textContent = elements.outputMode.value === "batch" ? "Convert to .zip" : "Convert to .gba";
}

function setBusy(busy) {
  conversionRunning = busy;
  elements.convertButton.disabled = busy || entries.length === 0;
  elements.cancelButton.hidden = !busy;
  elements.fileInput.disabled = busy;
  elements.clearButton.disabled = busy;
  const settings = [
    elements.preset, elements.outputMode, elements.vblanks, elements.fitMode,
    elements.paletteMode, elements.ditherMode, elements.compression, elements.audioMode,
    elements.seekSeconds, elements.defaultStart, elements.defaultEnd, elements.defaultSpeed,
    elements.defaultVolume, elements.defaultLoop, elements.romTitle, elements.normalize,
    elements.limiter, elements.resume,
  ];
  for (const control of settings) control.disabled = busy;
  for (const control of elements.fileList.querySelectorAll("input, select, button")) control.disabled = busy;
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
      "-show_entries", "format=duration:stream=codec_type,channels",
      "-of", "json",
      inputName,
      "-o", outputName,
    ]);
    if (exitCode === 0) {
      const raw = await ffmpeg.readFile(outputName, "utf8");
      const probe = JSON.parse(decodeText(raw));
      const duration = Number(probe?.format?.duration);
      if (Number.isFinite(duration) && duration > 0) {
        const audioStream = Array.isArray(probe.streams) ? probe.streams.find((stream) => stream.codec_type === "audio") : null;
        return { duration, hasAudio: Boolean(audioStream), channels: Number(audioStream?.channels) || 0, audioUnknown: false };
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
    return { duration: browserDuration, hasAudio: true, channels: 0, audioUnknown: true };
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
  const filters = [];
  if (clipOptions.audioMode === "left") filters.push("pan=mono|c0=c0");
  if (clipOptions.audioMode === "right") filters.push(probe.channels === 1 ? "pan=mono|c0=c0" : "pan=mono|c0=c1");
  filters.push(`aresample=${AUDIO_RATE}:async=1:first_pts=0`);
  filters.push(atempoFilter(timing.speed));
  if (Math.abs(timing.volume - 1) > 0.000001) filters.push(`volume=${timing.volume.toFixed(6)}`);
  if (project.normalize) filters.push("loudnorm=I=-16:LRA=11:TP=-1.5");
  if (project.limiter) filters.push("alimiter=limit=0.95:attack=5:release=50");

  const exitCode = await ffmpeg.exec([
    "-hide_banner", "-loglevel", "error", "-i", inputName,
    ...trimArguments(timing),
    "-map", "0:a:0", "-vn", "-af", filters.join(","),
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
  await ensureFFmpeg();
  if (conversionCancelled) throw new Error("Conversion cancelled.");

  updateProgress(6, "Loading the embedded v0.9 GBA player…");
  const playerResponse = await fetch(new URL("player_stub.bin", document.baseURI));
  if (!playerResponse.ok) throw new Error("Could not load player_stub.bin from the website.");
  const playerStub = new Uint8Array(await playerResponse.arrayBuffer());

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

  const transfers = [playerStub.buffer];
  for (const clip of clips) transfers.push(...clipTransferList(clip));
  const assembled = await runRomTask("assembleROM", {
    playerStub,
    clips,
    options: { romTitle: project.romTitle, outputMode: project.outputMode, resume: project.resume },
  }, transfers);
  updateProgress(100, "ROM ready.");
  return {
    ...assembled,
    fileName: outputFileName(project.romTitle, "gba"),
    mime: "application/octet-stream",
  };
}

async function startConversion() {
  if (!entries.length || conversionRunning) return;
  const totalBytes = entries.reduce((sum, entry) => sum + entry.file.size, 0);
  if (totalBytes > 1024 * 1024 * 1024) {
    alert("The selected source files exceed 1 GiB. Use the desktop app for a job this large.");
    return;
  }

  resetResult();
  logLines = [];
  recentFFmpegLogs = [];
  elements.logOutput.textContent = "";
  conversionCancelled = false;
  setBusy(true);
  updateProgress(0, "Starting browser converter…");

  try {
    const result = await performConversion();
    if (conversionCancelled) return;
    const blob = new Blob([result.buffer], { type: result.mime || "application/octet-stream" });
    resultURL = URL.createObjectURL(blob);
    resultFileName = result.fileName;
    resultMime = result.mime || "application/octet-stream";
    elements.resultTitle.textContent = `${resultFileName} is ready`;
    const noun = result.details.outputKind === "zip" ? "combined ROM data" : "ROM";
    elements.resultDetails.textContent = `${result.details.clipCount} clip${result.details.clipCount === 1 ? "" : "s"}, ${result.details.frameCount.toLocaleString()} frames, ${formatBytes(result.details.paddedSize)} ${noun}`;
    elements.resultArea.hidden = false;
    downloadResult();
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    appendLog(`ERROR: ${message}`);
    updateProgress(0, conversionCancelled ? "Conversion cancelled." : "Conversion failed.");
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

const presetFields = [elements.vblanks, elements.fitMode, elements.paletteMode, elements.ditherMode, elements.compression, elements.audioMode, elements.normalize, elements.limiter];
for (const control of presetFields) control.addEventListener("change", markPresetCustom);

elements.preset.addEventListener("change", () => applyPreset(elements.preset.value));
elements.outputMode.addEventListener("change", () => { updateConvertButton(); resetResult(); });

const ordinarySettings = [
  elements.seekSeconds, elements.defaultStart, elements.defaultEnd, elements.defaultSpeed,
  elements.defaultVolume, elements.defaultLoop, elements.romTitle, elements.resume,
];
for (const control of ordinarySettings) {
  control.addEventListener("change", resetResult);
  if (control.tagName === "INPUT" && ["text", "number"].includes(control.type)) control.addEventListener("input", resetResult);
}

elements.fileInput.addEventListener("change", () => {
  addFiles(elements.fileInput.files);
  elements.fileInput.value = "";
});
elements.clearButton.addEventListener("click", () => { entries = []; resetResult(); renderFiles(); });
elements.convertButton.addEventListener("click", startConversion);
elements.cancelButton.addEventListener("click", cancelConversion);
elements.downloadButton.addEventListener("click", downloadResult);

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
  if (resultURL) URL.revokeObjectURL(resultURL);
});

configureDesktopLink();
configureCompatibilityMessage();
updateConvertButton();
renderFiles();
