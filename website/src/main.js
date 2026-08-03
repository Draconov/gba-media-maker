import { FFmpeg } from "@ffmpeg/ffmpeg";
import { fetchFile, toBlobURL } from "@ffmpeg/util";
import { AUDIO_RATE, GBA_REFRESH, RGB_FRAME_BYTES } from "./rom-core.js";
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
  outputMode: document.querySelector("#outputMode"),
  vblanks: document.querySelector("#vblanks"),
  fitMode: document.querySelector("#fitMode"),
  ditherMode: document.querySelector("#ditherMode"),
  audioMode: document.querySelector("#audioMode"),
  seekSeconds: document.querySelector("#seekSeconds"),
  romTitle: document.querySelector("#romTitle"),
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

let entries = [];
let conversionRunning = false;
let conversionCancelled = false;
let ffmpeg = null;
let romWorker = null;
let romTaskCounter = 0;
const romTasks = new Map();
let resultURL = "";
let resultFileName = "";
let logLines = [];
let lastFFmpegLog = "";

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes < 1) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB"];
  const unit = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / 1024 ** unit).toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function titleFromFile(file) {
  return file.name.replace(/\.[^.]+$/, "").replace(/[_-]+/g, " ").trim().slice(0, 12) || "VIDEO";
}

function addFiles(fileList) {
  const existing = new Set(entries.map((entry) => `${entry.file.name}:${entry.file.size}:${entry.file.lastModified}`));
  for (const file of fileList) {
    if (!file.type.startsWith("video/") && !/\.(mp4|mov|mkv|webm|avi|m4v|mpeg|mpg)$/i.test(file.name)) continue;
    const key = `${file.name}:${file.size}:${file.lastModified}`;
    if (existing.has(key)) continue;
    entries.push({ id: crypto.randomUUID(), file, title: titleFromFile(file) });
    existing.add(key);
  }
  renderFiles();
}

function renderFiles() {
  elements.fileList.replaceChildren();
  elements.fileArea.hidden = entries.length === 0;
  elements.convertButton.disabled = entries.length === 0 || conversionRunning;
  const totalBytes = entries.reduce((sum, entry) => sum + entry.file.size, 0);
  elements.fileSummary.textContent = `${entries.length} video${entries.length === 1 ? "" : "s"} · ${formatBytes(totalBytes)}`;

  for (const entry of entries) {
    const row = document.createElement("div");
    row.className = "file-row";

    const fileName = document.createElement("div");
    fileName.className = "file-name";
    const strong = document.createElement("strong");
    strong.textContent = entry.file.name;
    const small = document.createElement("small");
    small.textContent = formatBytes(entry.file.size);
    fileName.append(strong, small);

    const label = document.createElement("label");
    const labelText = document.createElement("span");
    labelText.textContent = "Menu title";
    const titleInput = document.createElement("input");
    titleInput.type = "text";
    titleInput.maxLength = 12;
    titleInput.value = entry.title;
    titleInput.addEventListener("input", () => { entry.title = titleInput.value; });
    label.append(labelText, titleInput);

    const remove = document.createElement("button");
    remove.className = "icon-button";
    remove.type = "button";
    remove.title = `Remove ${entry.file.name}`;
    remove.setAttribute("aria-label", remove.title);
    remove.textContent = "×";
    remove.addEventListener("click", () => {
      entries = entries.filter((candidate) => candidate.id !== entry.id);
      renderFiles();
    });

    row.append(fileName, label, remove);
    elements.fileList.append(row);
  }
}

function setBusy(busy) {
  conversionRunning = busy;
  elements.convertButton.disabled = busy || entries.length === 0;
  elements.cancelButton.hidden = !busy;
  elements.fileInput.disabled = busy;
  elements.clearButton.disabled = busy;
  for (const control of [elements.outputMode, elements.vblanks, elements.fitMode, elements.ditherMode, elements.audioMode, elements.seekSeconds, elements.romTitle, elements.resume]) {
    control.disabled = busy;
  }
}

function resetResult() {
  if (resultURL) URL.revokeObjectURL(resultURL);
  resultURL = "";
  resultFileName = "";
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
  if (logLines.length > 18) logLines = logLines.slice(-18);
  elements.logOutput.textContent = logLines.join("\n");
  elements.logOutput.scrollTop = elements.logOutput.scrollHeight;
}

function currentOptions() {
  return {
    outputMode: elements.outputMode.value,
    vblanks: Number(elements.vblanks.value),
    fitMode: elements.fitMode.value,
    ditherMode: elements.ditherMode.value,
    audioMode: elements.audioMode.value,
    seekSeconds: Number(elements.seekSeconds.value),
    romTitle: elements.romTitle.value || "GBA VIDEO",
    resume: elements.resume.checked,
  };
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

function outputFileName(romTitle) {
  const clean = String(romTitle || "GBA VIDEO")
    .trim()
    .replace(/[^A-Za-z0-9_-]+/g, "_")
    .replace(/^_+|_+$/g, "") || "gba-video";
  return `${clean}.gba`;
}

async function ensureFFmpeg() {
  if (ffmpeg?.loaded) return;
  updateProgress(1, "Loading the browser video engine…");
  ffmpeg = new FFmpeg();
  ffmpeg.on("log", ({ message }) => {
    const clean = String(message || "").trim();
    if (clean && clean !== lastFFmpegLog) {
      lastFFmpegLog = clean;
      appendLog(clean);
    }
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

async function readProbe(inputName, index) {
  const outputName = `probe-${index}.json`;
  const exitCode = await ffmpeg.ffprobe([
    "-v", "error",
    "-show_entries", "format=duration:stream=codec_type,channels",
    "-of", "json",
    inputName,
    "-o", outputName,
  ]);
  if (exitCode !== 0) throw new Error("Could not inspect the selected video.");
  const raw = await ffmpeg.readFile(outputName, "utf8");
  await ffmpeg.deleteFile(outputName);
  const probe = JSON.parse(String(raw));
  const duration = Number(probe?.format?.duration);
  if (!Number.isFinite(duration) || duration <= 0) throw new Error("Could not read the video duration.");
  const audioStream = Array.isArray(probe.streams) ? probe.streams.find((stream) => stream.codec_type === "audio") : null;
  return { duration, hasAudio: Boolean(audioStream) };
}

function videoFilter(fitMode, vblanks) {
  const fps = GBA_REFRESH / vblanks;
  let scale;
  if (fitMode === "crop") scale = "scale=120:80:force_original_aspect_ratio=increase,crop=120:80";
  else if (fitMode === "stretch") scale = "scale=120:80";
  else scale = "scale=120:80:force_original_aspect_ratio=decrease,pad=120:80:(ow-iw)/2:(oh-ih)/2:black";
  return `${scale},fps=${fps.toFixed(10)},format=rgb24`;
}

async function extractFrames(inputName, index, options, duration) {
  const outputName = `frames-${index}.rgb`;
  const estimatedFrames = Math.max(1, Math.ceil(duration * (GBA_REFRESH / options.vblanks)));
  const estimatedBytes = estimatedFrames * RGB_FRAME_BYTES;
  if (estimatedBytes > MAX_RAW_FRAME_BYTES) {
    throw new Error(`This clip would need about ${(estimatedBytes / 1048576).toFixed(0)} MiB of raw browser memory. Use a shorter clip, a lower frame rate, or the desktop app.`);
  }
  const exitCode = await ffmpeg.exec([
    "-hide_banner", "-loglevel", "error", "-i", inputName, "-an",
    "-vf", videoFilter(options.fitMode, options.vblanks),
    "-pix_fmt", "rgb24", "-f", "rawvideo", outputName,
  ]);
  if (exitCode !== 0) throw new Error("FFmpeg could not decode the video frames.");
  const frames = await ffmpeg.readFile(outputName);
  await ffmpeg.deleteFile(outputName);
  if (!(frames instanceof Uint8Array) || frames.length < RGB_FRAME_BYTES) throw new Error("The converted video contains no usable frames.");
  return frames;
}

async function extractAudio(inputName, index, options, probe) {
  if (options.audioMode === "none" || !probe.hasAudio) return new Uint8Array();
  const outputName = `audio-${index}.s8`;
  const exitCode = await ffmpeg.exec([
    "-hide_banner", "-loglevel", "error", "-i", inputName, "-map", "0:a:0", "-vn",
    "-af", `aresample=${AUDIO_RATE}:async=1:first_pts=0,alimiter=limit=0.95:attack=5:release=50`,
    "-ac", "1", "-ar", String(AUDIO_RATE), "-f", "s8", outputName,
  ]);
  if (exitCode !== 0) throw new Error("FFmpeg could not decode the audio stream.");
  const audio = await ffmpeg.readFile(outputName);
  await ffmpeg.deleteFile(outputName);
  return audio instanceof Uint8Array ? audio : new Uint8Array();
}

function clipTransferList(clip) {
  return [clip.palette.buffer, clip.videoIndex.buffer, clip.video.buffer, clip.audio.buffer];
}

async function performConversion() {
  const options = currentOptions();
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
    const base = 8 + index * clipSpan;
    const mapped = (fraction, message) => updateProgress(base + fraction * clipSpan, `${entry.title || "VIDEO"} — ${message}`);
    const inputName = safeVirtualName(index, entry.file.name);

    mapped(0, "Loading video into browser memory…");
    await ffmpeg.writeFile(inputName, await fetchFile(entry.file));
    try {
      mapped(0.03, "Inspecting media…");
      const probe = await readProbe(inputName, index);
      mapped(0.08, "Extracting 120×80 frames…");
      const framesRGB = await extractFrames(inputName, index, options, probe.duration);
      mapped(0.31, probe.hasAudio && options.audioMode !== "none" ? "Extracting audio…" : "No audio selected.");
      const audio = await extractAudio(inputName, index, options, probe);
      mapped(0.38, "Encoding the GBA clip…");
      const clip = await runRomTask("encodeClip", {
        framesRGB,
        audio,
        title: entry.title || "VIDEO",
        vblanks: options.vblanks,
        ditherMode: options.ditherMode,
        keyInterval: 30,
        seekSeconds: options.seekSeconds,
        loop: false,
      }, [framesRGB.buffer, audio.buffer], (fraction, message) => mapped(0.38 + fraction * 0.6, message));
      clips.push(clip);
    } finally {
      try { await ffmpeg.deleteFile(inputName); } catch { /* already removed */ }
    }
  }

  updateProgress(97, "Assembling and validating the ROM…");
  let outputMode = options.outputMode;
  if (outputMode === "auto") outputMode = clips.length > 1 ? "menu" : "rom";
  if (clips.length === 1) outputMode = "rom";
  const transfers = [playerStub.buffer];
  for (const clip of clips) transfers.push(...clipTransferList(clip));
  const assembled = await runRomTask("assembleROM", {
    playerStub,
    clips,
    options: { romTitle: options.romTitle, outputMode, resume: options.resume },
  }, transfers);
  updateProgress(100, "ROM ready.");
  return { ...assembled, fileName: outputFileName(options.romTitle) };
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
  elements.logOutput.textContent = "";
  conversionCancelled = false;
  setBusy(true);
  updateProgress(0, "Starting browser converter…");

  try {
    const result = await performConversion();
    if (conversionCancelled) return;
    const blob = new Blob([result.buffer], { type: "application/octet-stream" });
    resultURL = URL.createObjectURL(blob);
    resultFileName = result.fileName;
    elements.resultTitle.textContent = `${resultFileName} is ready`;
    elements.resultDetails.textContent = `${result.details.clipCount} clip${result.details.clipCount === 1 ? "" : "s"}, ${result.details.frameCount.toLocaleString()} frames, ${formatBytes(result.details.paddedSize)} ROM`;
    elements.resultArea.hidden = false;
    downloadResult();
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    appendLog(`ERROR: ${message}`);
    updateProgress(0, conversionCancelled ? "Conversion cancelled." : "Conversion failed.");
    if (!conversionCancelled) alert(message);
  } finally {
    setBusy(false);
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

elements.fileInput.addEventListener("change", () => {
  addFiles(elements.fileInput.files);
  elements.fileInput.value = "";
});
elements.clearButton.addEventListener("click", () => { entries = []; renderFiles(); });
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
renderFiles();
