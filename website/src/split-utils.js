export function formatClock(seconds, precise = false) {
  const value = Math.max(0, Number(seconds) || 0);
  const hours = Math.floor(value / 3600);
  const minutes = Math.floor(value / 60) % 60;
  const wholeSeconds = Math.floor(value) % 60;
  const fraction = precise ? `.${Math.floor((value - Math.floor(value)) * 100).toString().padStart(2, "0")}` : "";
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(wholeSeconds).padStart(2, "0")}${fraction}`
    : `${minutes}:${String(wholeSeconds).padStart(2, "0")}${fraction}`;
}

export function parseClock(value) {
  const text = String(value ?? "").trim();
  if (!text || text === "0") return 0;
  if (/^\d+(?:\.\d+)?$/.test(text)) return Number(text);
  const parts = text.split(":");
  if (parts.length < 2 || parts.length > 3 || parts.some((part) => !/^\d+(?:\.\d+)?$/.test(part))) return NaN;
  let total = 0;
  for (const part of parts) total = total * 60 + Number(part);
  return total;
}

export function chooseChapterEnd(chapters, cursor, candidateEnd, finalEnd) {
  if (!Array.isArray(chapters) || !chapters.length || candidateEnd >= finalEnd - 0.001) return candidateEnd;
  let best = 0;
  for (const chapter of chapters) {
    if (chapter <= cursor + 2) continue;
    if (chapter > candidateEnd + 0.001) break;
    best = chapter;
  }
  if (!best) return candidateEnd;
  const threshold = Math.max(30, (candidateEnd - cursor) * 0.25);
  return candidateEnd - best <= threshold ? best : candidateEnd;
}
