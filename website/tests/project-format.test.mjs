import assert from "node:assert/strict";
import test from "node:test";
import {
  PROJECT_FORMAT,
  PROJECT_VERSION,
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
    defaultImageSeconds: 0,
    romTitle: "MEDIA",
    normalize: true,
    limiter: true,
    resume: true,
    outputMode: "menu",
    splitVideo: false,
    splitBudgetMiB: 30,
    maxPartDuration: "12:34",
    chapterAware: true,
    partTitleScreens: true,
    titleCards: { enabled: true, useShared: true, shared: { title: "Movie" }, parts: [] },
    resumeLongSplit: true,
    menuBackground: "ocean-wave-animated",
    menuUIColor: "#FFFFFF",
    menuSelectionColor: "#FFDE00",
    menuOutline: true,
    menuOutlineColor: "#000000",
    menuTheme: { id: "ocean-wave-animated" },
    ...overrides,
  };
}

function mediaEntry(overrides = {}) {
  return {
    file: fakeFile("song.flac"),
    title: "SONG",
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
    imageSeconds: 0,
    musicTitle: "Twenty Eight Character Song",
    musicArtist: "Artist Name",
    musicArtworkMode: "custom",
    musicArtworkPreset: "preset-20",
    musicArtworkCustom: "data:image/png;base64,AAAA",
    ...overrides,
  };
}

test("browser saves the canonical v2 GBA Media Maker .gbamedia schema", () => {
  const entry = mediaEntry();
  const doc = canonicalProjectFromBrowser({ settings: browserSettings(), entries: [entry] });
  assert.equal(doc.format, PROJECT_FORMAT);
  assert.equal(doc.version, PROJECT_VERSION);
  assert.equal(doc.version, 2);
  assert.equal(doc.settings.fps, "balanced");
  assert.equal(doc.settings.fit, "crop");
  assert.equal(doc.settings.audio, "mix");
  assert.equal(doc.settings.volume, 110);
  assert.equal(doc.settings.start, "0:02.50");
  assert.equal(doc.settings.end, "1:05");
  assert.equal(doc.settings.imageSeconds, 0, "manual-image project default must survive serialization");
  assert.equal("vblanks" in doc.settings, false);
  assert.equal("fitMode" in doc.settings, false);
  assert.equal("defaultVolume" in doc.settings, false);
  assert.equal(doc.clips[0].path, "");
  assert.equal(doc.clips[0].name, "song.flac");
  assert.equal(doc.clips[0].size, entry.file.size);
  assert.equal(doc.clips[0].settings.fit, "stretch");
  assert.equal(doc.clips[0].settings.audio, "right");
  assert.equal(doc.clips[0].settings.audioTrack, 2);
  assert.equal(doc.clips[0].settings.volume, 75);
  assert.equal(doc.clips[0].settings.imageSeconds, 0);
  assert.equal(doc.clips[0].settings.musicTitle, "Twenty Eight Character Song");
  assert.equal(doc.clips[0].settings.musicArtist, "Artist Name");
  assert.equal(doc.clips[0].settings.musicArtworkMode, "custom");
  assert.equal(doc.clips[0].settings.musicArtworkPreset, "preset-20");
  assert.equal(doc.clips[0].settings.musicArtworkCustom, "data:image/png;base64,AAAA");
});

test("canonical v2 projects restore media settings without changing semantics", () => {
  const original = canonicalProjectFromBrowser({
    settings: browserSettings(),
    entries: [mediaEntry({ audioTrack: 1, musicArtworkMode: "default", musicArtworkPreset: "preset-07" })],
  });
  const restored = browserStateFromCanonicalProject(original);
  assert.equal(restored.settings.vblanks, 5);
  assert.equal(restored.settings.fitMode, "crop");
  assert.equal(restored.settings.audioMode, "mix");
  assert.equal(restored.settings.defaultVolume, 1.1);
  assert.equal(restored.settings.defaultImageSeconds, 0);
  assert.equal(restored.settings.maxPartDuration, "12:34");
  assert.equal(restored.clips[0].source.name, "song.flac");
  assert.equal(restored.clips[0].fitMode, "stretch");
  assert.equal(restored.clips[0].audioTrack, 1);
  assert.equal(restored.clips[0].volume, 0.75);
  assert.equal(restored.clips[0].imageSeconds, 0);
  assert.equal(restored.clips[0].musicTitle, "Twenty Eight Character Song");
  assert.equal(restored.clips[0].musicArtist, "Artist Name");
  assert.equal(restored.clips[0].musicArtworkMode, "default");
  assert.equal(restored.clips[0].musicArtworkPreset, "preset-07");
});

test("browser accepts the previous canonical v1 video project format", () => {
  const restored = normalizeBrowserProjectDocument({
    format: "gba-video-maker-project",
    version: 1,
    settings: { fps: "classic", fit: "fit", audio: "left", volume: 80, romTitle: "OLD", outputMode: "playlist" },
    clips: [{ name: "old.mp4", size: 99, settings: { title: "OLD", fit: "crop", audio: "left", volume: 80 } }],
  });
  assert.equal(restored.settings.vblanks, 6);
  assert.equal(restored.settings.romTitle, "OLD");
  assert.equal(restored.settings.outputMode, "menu", "legacy playlist projects upgrade to the media menu");
  assert.equal(restored.clips[0].source.name, "old.mp4");
  assert.equal(restored.clips[0].fitMode, "crop");
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

test("empty or unsupported projects are rejected as GBA Media Maker projects", () => {
  assert.throws(() => normalizeBrowserProjectDocument({ format: PROJECT_FORMAT, version: 2, settings: {}, clips: [] }), /valid GBA Media Maker project/);
  assert.throws(() => normalizeBrowserProjectDocument({ format: "gba-video-maker-project", version: 1, settings: {}, clips: [] }), /valid GBA Media Maker project/);
  assert.throws(() => normalizeBrowserProjectDocument({ format: PROJECT_FORMAT, version: 999, settings: {}, clips: [{}] }), /valid GBA Media Maker project/);
});
