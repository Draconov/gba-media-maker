import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { assembleROM, METADATA_OFFSET, RGB_FRAME_BYTES, convertRawClip } from "../src/rom-core.js";
import {
  TITLE_CARD_BYTES,
  buildTitleCardAsset,
  createTitleCardProject,
  defaultTitleCardSettings,
  resolveTitleCardSettings,
} from "../src/title-cards.js";

function u16(bytes, offset) {
  return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint16(offset, true);
}
function u32(bytes, offset) {
  return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(offset, true);
}

function solidRGB(r, g, b) {
  const rgb = new Uint8Array(240 * 160 * 3);
  for (let pixel = 0; pixel < 240 * 160; pixel += 1) {
    rgb[pixel * 3] = r;
    rgb[pixel * 3 + 1] = g;
    rgb[pixel * 3 + 2] = b;
  }
  return rgb;
}

test("title-card defaults use the video filename and automatic part token", () => {
  const defaults = defaultTitleCardSettings("My holiday film.mp4");
  assert.equal(defaults.title, "MY HOLIDAY FILM");
  assert.equal(defaults.subtitle, "Part {part}");
  assert.equal(defaults.backgroundMode, "part-first-frame");
  assert.equal(defaults.darkness, 50);
  assert.equal(defaults.textSize, "large");

  const project = createTitleCardProject("My holiday film.mp4");
  assert.equal(resolveTitleCardSettings(project, "My holiday film.mp4", 4).subtitle, "PART 4");
});

test("native title-card asset contains a 240 by 160 RGB555 screen and timing flags", () => {
  const settings = { ...defaultTitleCardSettings("movie.mp4"), startMode: "timer", durationSeconds: 2 };
  const asset = buildTitleCardAsset(solidRGB(40, 80, 120), settings, 2, "movie.mp4");
  assert.equal(asset.length, TITLE_CARD_BYTES);
  assert.equal(u32(asset, 0), 0x31444354);
  assert.equal(u16(asset, 4), 1);
  assert.equal(u16(asset, 6) & 1, 0);
  assert.notEqual(u16(asset, 6) & 2, 0);
  assert.notEqual(u16(asset, 6) & 4, 0);
  assert.equal(u32(asset, 8), 240 * 160 * 2);
  assert.ok(u32(asset, 12) >= 119 && u32(asset, 12) <= 120);
});

test("title-card text sizes render distinct native screens", () => {
  const background = solidRGB(0, 0, 0);
  const base = { ...defaultTitleCardSettings("A very long video title.mp4"), subtitle: "PART 1" };
  const large = buildTitleCardAsset(background, { ...base, textSize: "large" }, 1, "movie.mp4");
  const small = buildTitleCardAsset(background, { ...base, textSize: "small" }, 1, "movie.mp4");
  assert.notDeepEqual(large, small);
});

test("browser ROM assembly stores the native title-card pointer in GBV5 metadata", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const clip = convertRawClip({
    framesRGB: new Uint8Array(RGB_FRAME_BYTES),
    title: "PART ONE",
    vblanks: 8,
    ditherMode: "off",
    compression: "none",
  });
  const titleCard = buildTitleCardAsset(solidRGB(10, 20, 30), defaultTitleCardSettings("movie.mp4"), 1, "movie.mp4");
  const result = assembleROM(playerStub, [clip], { romTitle: "PART 01", outputMode: "rom", titleCard });
  const pointer = u32(result.rom, METADATA_OFFSET + 52);
  assert.ok(pointer > 0);
  assert.equal(u32(result.rom, pointer), 0x31444354);
  assert.notEqual(u16(result.rom, METADATA_OFFSET + 6) & 0x0004, 0);
});
