import assert from "node:assert/strict";
import test from "node:test";
import {
  PROJECT_FORMAT,
  browserStateFromCanonicalProject,
  canonicalProjectFromBrowser,
  normalizeBrowserProjectDocument,
} from "../src/project-format.js";

function fakeFile(name, size = 123456, lastModified = 1700000000000) {
  return { name, size, lastModified };
}

function browserSettings(overrides = {}) {
  return {
    preset: "extreme",
    audioQuality: "auto",
    smartTargetMiB: 28,
    smartPriority: "balanced",
    vblanks: 5,
    fitMode: "crop",
    paletteMode: "scene",
    ditherMode: "ordered",
    compression: "delta",
    audioMode: "mix",
    seekSeconds: 10,
    defaultStart: 2.5,
    defaultEnd: 65,
    defaultSpeed: 1.25,
    defaultVolume: 1.1,
    defaultLoop: true,
    romTitle: "MOVIE",
    normalize: true,
    limiter: true,
    resume: true,
    outputMode: "rom",
    splitVideo: true,
    splitBudgetMiB: 30,
    maxPartDuration: "12:34",
    chapterAware: true,
    partTitleScreens: true,
    titleCards: { enabled: true, useShared: true, shared: { title: "Movie" }, parts: [] },
    resumeLongSplit: true,
    menuBackground: "classic-dark",
    menuUIColor: "#FFFFFF",
    menuSelectionColor: "#FFDE00",
    menuOutline: true,
    menuOutlineColor: "#000000",
    menuTheme: null,
    ...overrides,
  };
}

test("browser saves the canonical desktop .gbavideo schema", () => {
  const entry = {
    file: fakeFile("movie.mkv"),
    title: "MOVIE",
    useProject: false,
    start: 4,
    end: 44.5,
    speed: 1.5,
    fitMode: "stretch",
    audioMode: "right",
    audioTrack: 2,
    volume: 0.75,
    loop: false,
    paletteMode: "shared",
    ditherMode: "off",
  };
  const doc = canonicalProjectFromBrowser({ settings: browserSettings(), entries: [entry] });
  assert.equal(doc.format, PROJECT_FORMAT);
  assert.equal(doc.version, 1);
  assert.equal(doc.settings.fps, "balanced");
  assert.equal(doc.settings.fit, "crop");
  assert.equal(doc.settings.audio, "mix");
  assert.equal(doc.settings.volume, 110);
  assert.equal(doc.settings.start, "0:02.50");
  assert.equal(doc.settings.end, "1:05");
  assert.equal("vblanks" in doc.settings, false);
  assert.equal("fitMode" in doc.settings, false);
  assert.equal("defaultVolume" in doc.settings, false);
  assert.equal(doc.clips[0].path, "");
  assert.equal(doc.clips[0].name, "movie.mkv");
  assert.equal(doc.clips[0].size, entry.file.size);
  assert.equal(doc.clips[0].settings.fit, "stretch");
  assert.equal(doc.clips[0].settings.audio, "right");
  assert.equal(doc.clips[0].settings.audioTrack, 2);
  assert.equal(doc.clips[0].settings.volume, 75);
});

test("canonical desktop projects restore into browser settings without changing semantics", () => {
  const original = canonicalProjectFromBrowser({
    settings: browserSettings({ outputMode: "menu", menuTheme: { id: "classic-dark" } }),
    entries: [{
      file: fakeFile("movie.mkv"), title: "MOVIE", useProject: false,
      start: 4, end: 44.5, speed: 1.5, fitMode: "stretch", audioMode: "right",
      audioTrack: 1, volume: 0.75, loop: true, paletteMode: "shared", ditherMode: "off",
    }],
  });
  const restored = browserStateFromCanonicalProject(original);
  assert.equal(restored.settings.vblanks, 5);
  assert.equal(restored.settings.fitMode, "crop");
  assert.equal(restored.settings.audioMode, "mix");
  assert.equal(restored.settings.defaultVolume, 1.1);
  assert.equal(restored.settings.maxPartDuration, "12:34");
  assert.equal(restored.clips[0].source.name, "movie.mkv");
  assert.equal(restored.clips[0].fitMode, "stretch");
  assert.equal(restored.clips[0].audioTrack, 1);
  assert.equal(restored.clips[0].volume, 0.75);
});

test("browser still migrates the previous browser-only project schema", () => {
  const migrated = normalizeBrowserProjectDocument({
    format: "GBA Video Maker Project",
    version: 1,
    settings: { vblanks: 8, fitMode: "fit", audioMode: "left", defaultVolume: 0.8, romTitle: "OLD" },
    clips: [{ source: { name: "old.mp4", size: 99 }, title: "OLD", fitMode: "crop", audioMode: "left", volume: 0.8 }],
  });
  assert.equal(migrated.settings.vblanks, 8);
  assert.equal(migrated.settings.romTitle, "OLD");
  assert.equal(migrated.clips[0].source.name, "old.mp4");
  assert.equal(migrated.clips[0].fitMode, "crop");
});


test("empty canonical and legacy projects are rejected like the desktop app", () => {
  assert.throws(() => normalizeBrowserProjectDocument({ format: "gba-video-maker-project", version: 1, settings: {}, clips: [] }), /valid \.gbavideo project/);
  assert.throws(() => normalizeBrowserProjectDocument({ format: "GBA Video Maker Project", version: 1, settings: {}, clips: [] }), /valid \.gbavideo project/);
});
