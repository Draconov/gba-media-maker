import { assembleROM, convertRawClip } from "./rom-core.js";

function sendError(id, error) {
  self.postMessage({
    type: "error",
    id,
    message: error instanceof Error ? error.message : String(error),
  });
}

self.addEventListener("message", (event) => {
  const { id, action, payload } = event.data || {};
  try {
    if (action === "encodeClip") {
      const clip = convertRawClip({
        ...payload,
        report: (fraction, message) => {
          self.postMessage({ type: "progress", id, fraction, message });
        },
      });
      self.postMessage({ type: "clip", id, clip }, [
        clip.palette.buffer,
        clip.paletteIndex.buffer,
        clip.videoIndex.buffer,
        clip.video.buffer,
        clip.audio.buffer,
      ]);
      return;
    }

    if (action === "assembleROM") {
      const result = assembleROM(payload.playerStub, payload.clips, payload.options);
      self.postMessage({
        type: "rom",
        id,
        buffer: result.rom.buffer,
        details: {
          clipCount: result.clipCount,
          frameCount: result.frameCount,
          unpaddedSize: result.unpaddedSize,
          paddedSize: result.paddedSize,
        },
      }, [result.rom.buffer]);
      return;
    }

    throw new Error(`Unknown ROM worker action: ${action}`);
  } catch (error) {
    sendError(id, error);
  }
});
