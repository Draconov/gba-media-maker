import assert from "node:assert/strict";
import test from "node:test";
import {
  batchRomFileName,
  buildOptimizerProposal,
  conversionOutputFileName,
  parsePartDuration,
  resolveAudioCodec,
  splitArchiveFileName,
  splitPartFileName,
} from "../src/parity-utils.js";

test("maximum split duration matches the desktop MM:SS validation", () => {
  assert.equal(parsePartDuration(""), 0);
  assert.equal(parsePartDuration("0"), 0);
  assert.equal(parsePartDuration("1:05"), 65);
  assert.equal(parsePartDuration("240:00"), 14400);
  for (const invalid of ["65", "1:5", "1:60", "1:02:03", "240:01", "-1:00", "abc"]) {
    assert.equal(Number.isNaN(parsePartDuration(invalid)), true, invalid);
  }
});

test("Extreme Auto audio uses the same one-third target threshold as the desktop converter", () => {
  const targetMiB = 30;
  const threshold = targetMiB * 1048576 / 3;
  assert.equal(resolveAudioCodec("auto", true, threshold, targetMiB), "pcm");
  assert.equal(resolveAudioCodec("auto", true, threshold + 1, targetMiB), "adpcm");
  assert.equal(resolveAudioCodec("auto", false, threshold * 2, targetMiB), "pcm");
  assert.equal(resolveAudioCodec("adpcm", true, 1, targetMiB), "adpcm");
  assert.equal(resolveAudioCodec("pcm", true, threshold * 2, targetMiB), "pcm");
});

test("browser output names match desktop naming", () => {
  assert.equal(conversionOutputFileName(["My Movie.mp4"], "rom"), "My Movie.gba");
  assert.equal(conversionOutputFileName(["one.mp4", "two.mkv"], "menu"), "GBA_Media_Collection.gba");
  assert.equal(conversionOutputFileName(["one.mp4", "two.mkv"], "playlist"), "GBA_Media_Collection.gba");
  assert.equal(conversionOutputFileName(["one.mp4", "two.mkv"], "batch"), "GBA_Media_Collection.zip");
  assert.equal(splitArchiveFileName("My Movie.mp4"), "My Movie_PARTS.zip");
  assert.equal(splitPartFileName("My Movie.mp4", 3), "My Movie_PART_03.gba");
  assert.equal(batchRomFileName("My Movie.mp4"), "My Movie_GBA.gba");
  assert.equal(batchRomFileName('bad:name?.mkv'), "bad_name__GBA.gba");
});

test("browser optimizer follows the desktop staged reduction order", () => {
  const entries = [{ id: "a", name: "long.mp4", duration: 5000, hasAudio: true }];
  const snapshot = {
    global: {
      fps: "smooth", compression: "none", outputMode: "rom", preset: "best",
      audioQuality: "pcm", smartTargetMiB: 32, normalize: true, limiter: true,
    },
    defaults: {
      start: 0, end: 0, speed: 1, fit: "fit", audio: "mix", volume: 1,
      loop: false, paletteMode: "scene", ditherMode: "error",
    },
    clips: {
      a: { title: "LONG", useProject: true, start: 0, end: 0, speed: 1, fit: "fit", audio: "mix", volume: 1, loop: false, paletteMode: "scene", ditherMode: "error" },
    },
  };
  const proposal = buildOptimizerProposal(snapshot, entries, "a");
  const text = proposal.changes.join("\n");
  const compression = text.indexOf("Video compression:");
  const fps = text.indexOf("Frame rate:");
  const palette = text.indexOf("Project palette:");
  const audio = text.indexOf("Audio → None");
  const end = text.indexOf("End time shortened");
  assert.ok(compression >= 0);
  assert.ok(fps > compression);
  assert.ok(palette > fps);
  assert.ok(audio > palette);
  assert.ok(end > audio);
  assert.equal(proposal.model.clips.a.useProject, false);
  assert.equal(proposal.model.clips.a.audio, "none");
  assert.ok(proposal.model.clips.a.end > 0 && proposal.model.clips.a.end < entries[0].duration);
});
