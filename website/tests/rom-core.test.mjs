import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import {
  ASSET_OFFSET,
  FRAME_BYTES,
  METADATA_OFFSET,
  RGB_FRAME_BYTES,
  NATIVE_IMAGE_BYTES,
  NATIVE_IMAGE_RGB_BYTES,
  assembleROM,
  convertNativeMediaClip,
  convertRawClip,
  rgb24ToNativeRGB555,
  safeRomTitle,
} from "../src/rom-core.js";

function u16(bytes, offset) {
  return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint16(offset, true);
}

function u32(bytes, offset) {
  return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(offset, true);
}

function reconstructFrames(rom, descriptorOffset) {
  const frameCount = u32(rom, descriptorOffset);
  const videoOffset = u32(rom, descriptorOffset + 8);
  const videoIndexOffset = u32(rom, descriptorOffset + 12);
  const frames = [];
  let previous = new Uint8Array(FRAME_BYTES);

  for (let frame = 0; frame < frameCount; frame += 1) {
    const recordOffset = videoOffset + u32(rom, videoIndexOffset + frame * 4);
    const type = u32(rom, recordOffset);
    const payloadLength = u32(rom, recordOffset + 4);
    const payload = rom.subarray(recordOffset + 8, recordOffset + 8 + payloadLength);
    const current = previous.slice();
    if (type === 0) {
      assert.equal(payload.length, FRAME_BYTES);
      current.set(payload);
    } else {
      assert.equal(type, 1);
      let outputPosition = 0;
      let inputPosition = 0;
      while (inputPosition < payload.length) {
        const skip = u16(payload, inputPosition);
        const run = u16(payload, inputPosition + 2);
        inputPosition += 4;
        outputPosition += skip;
        current.set(payload.subarray(inputPosition, inputPosition + run), outputPosition);
        inputPosition += run;
        outputPosition += run;
      }
    }
    frames.push(current);
    previous = current;
  }
  return frames;
}

test("safeRomTitle sanitizes and pads to twelve bytes", () => {
  assert.equal(new TextDecoder().decode(safeRomTitle("My cat! 2026")), "MY CAT 2026 ");
});


test("native image conversion matches the v0.13 240x160 RGB555 descriptor", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const rgb = new Uint8Array(NATIVE_IMAGE_RGB_BYTES);
  for (let pixel = 0; pixel < rgb.length; pixel += 3) {
    rgb[pixel] = 255;
    rgb[pixel + 1] = 128;
    rgb[pixel + 2] = 0;
  }
  const native = rgb24ToNativeRGB555(rgb);
  assert.equal(native.length, NATIVE_IMAGE_BYTES);

  const clip = convertNativeMediaClip({ mediaKind: "image", nativeRGB: rgb, title: "PICTURE", imageSeconds: 0, vblanks: 5 });
  const result = assembleROM(playerStub, [clip], { romTitle: "IMAGE", outputMode: "rom", resume: true });
  const rom = result.rom;
  const flags = u16(rom, ASSET_OFFSET + 50);
  assert.equal(u32(rom, ASSET_OFFSET), 1);
  assert.equal(u32(rom, ASSET_OFFSET + 4), NATIVE_IMAGE_BYTES);
  assert.equal(u16(rom, ASSET_OFFSET + 46), 240);
  assert.equal(u16(rom, ASSET_OFFSET + 48), 160);
  assert.ok(flags & 0x0080, "image flag missing");
  assert.equal(flags & 0x0001, 0, "image must not advertise audio");
  assert.equal(u32(rom, ASSET_OFFSET + 84), 0, "manual image must preserve zero slideshow duration");
  assert.equal(new TextDecoder().decode(rom.subarray(0xac, 0xb0)), "GM05");
});

test("native audio conversion writes artwork, MMD2 metadata, and audio flags", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const rgb = new Uint8Array(NATIVE_IMAGE_RGB_BYTES);
  for (let pixel = 0; pixel < rgb.length; pixel += 3) {
    rgb[pixel] = 25;
    rgb[pixel + 1] = 80;
    rgb[pixel + 2] = 180;
  }
  const audio = new Uint8Array(16384);
  for (let index = 0; index < audio.length; index += 1) audio[index] = (index * 7) & 0xff;
  const clip = convertNativeMediaClip({
    mediaKind: "audio", nativeRGB: rgb, audio, title: "MUSIC", musicTitle: "Track Name", artist: "Artist", album: "Album",
    durationSeconds: 1, vblanks: 5, audioCodec: "pcm", seekSeconds: 5, loop: true,
  });
  const result = assembleROM(playerStub, [clip], { romTitle: "MUSIC", outputMode: "rom", resume: true });
  const rom = result.rom;
  const flags = u16(rom, ASSET_OFFSET + 50);
  assert.ok(flags & 0x0001, "audio-present flag missing");
  assert.ok(flags & 0x0002, "loop flag missing");
  assert.ok(flags & 0x0040, "audio-only flag missing");
  assert.ok(flags & 0x0100, "media-metadata flag missing");
  assert.equal(u32(rom, ASSET_OFFSET + 4), NATIVE_IMAGE_BYTES);
  assert.equal(u16(rom, ASSET_OFFSET + 46), 240);
  assert.equal(u16(rom, ASSET_OFFSET + 48), 160);
  assert.equal(u32(rom, ASSET_OFFSET + 84), clip.audioSampleCount);
  assert.ok(u32(rom, ASSET_OFFSET + 16) > 0, "audio payload missing");
  assert.ok(u32(rom, ASSET_OFFSET + 32) > 0, "audio seek table missing");

  const metadataOffset = u32(rom, ASSET_OFFSET + 12);
  assert.ok(metadataOffset > ASSET_OFFSET);
  assert.equal(u32(rom, metadataOffset), 0x32444d4d);
  assert.equal(new TextDecoder().decode(rom.subarray(metadataOffset + 4, metadataOffset + 14)), "TRACK NAME");
  assert.equal(new TextDecoder().decode(rom.subarray(metadataOffset + 32, metadataOffset + 38)), "ARTIST");
  assert.equal(new TextDecoder().decode(rom.subarray(metadataOffset + 60, metadataOffset + 65)), "ALBUM");
});

test("mixed video, audio, and image clips assemble as one GBV5 media-menu ROM", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const video = convertRawClip({ framesRGB: new Uint8Array(RGB_FRAME_BYTES), title: "VIDEO", vblanks: 8, ditherMode: "off", compression: "none" });
  const rgb = new Uint8Array(NATIVE_IMAGE_RGB_BYTES);
  const audioBytes = new Uint8Array(16384);
  const audio = convertNativeMediaClip({ mediaKind: "audio", nativeRGB: rgb, audio: audioBytes, title: "AUDIO", musicTitle: "SONG", durationSeconds: 1, vblanks: 5 });
  const image = convertNativeMediaClip({ mediaKind: "image", nativeRGB: rgb, title: "IMAGE", imageSeconds: 5, vblanks: 5 });
  const result = assembleROM(playerStub, [video, audio, image], { romTitle: "MIXED", outputMode: "menu", resume: true });
  assert.equal(u16(result.rom, METADATA_OFFSET + 8), 3);
  assert.equal(u16(result.rom, METADATA_OFFSET + 6) & 0x0002, 0, "media menu must not use legacy playlist mode");
  assert.equal(u16(result.rom, ASSET_OFFSET + 50) & (0x0040 | 0x0080), 0, "video descriptor must remain video");
  assert.ok(u16(result.rom, ASSET_OFFSET + 96 + 50) & 0x0040, "second descriptor must be audio");
  assert.ok(u16(result.rom, ASSET_OFFSET + 192 + 50) & 0x0080, "third descriptor must be image");
});

test("browser core creates a structurally valid GBV5 ROM", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  assert.equal(playerStub.length, ASSET_OFFSET);

  const frames = new Uint8Array(RGB_FRAME_BYTES * 3);
  for (let pixel = 0; pixel < FRAME_BYTES; pixel += 1) {
    frames[pixel * 3] = 20;
    frames[pixel * 3 + 1] = 70;
    frames[pixel * 3 + 2] = 180;
  }
  frames.set(frames.subarray(0, RGB_FRAME_BYTES), RGB_FRAME_BYTES);
  frames.set(frames.subarray(0, RGB_FRAME_BYTES), RGB_FRAME_BYTES * 2);
  // Force small changes so delta records are exercised.
  frames[RGB_FRAME_BYTES + 3 * 44] = 250;
  frames[RGB_FRAME_BYTES * 2 + 3 * 45 + 1] = 240;

  const clip = convertRawClip({
    framesRGB: frames,
    title: "CAT TEST",
    vblanks: 6,
    ditherMode: "off",
    keyInterval: 30,
    seekSeconds: 5,
    loop: true,
  });
  const result = assembleROM(playerStub, [clip], { romTitle: "WEB TEST", outputMode: "rom", resume: true });
  const rom = result.rom;

  assert.equal(new TextDecoder().decode(rom.subarray(METADATA_OFFSET, METADATA_OFFSET + 4)), "GBV5");
  assert.equal(u16(rom, METADATA_OFFSET + 4), 5);
  assert.equal(u16(rom, METADATA_OFFSET + 8), 1);
  assert.equal(u32(rom, METADATA_OFFSET + 12), ASSET_OFFSET);
  assert.equal(u32(rom, ASSET_OFFSET), 3);
  assert.equal(u16(rom, ASSET_OFFSET + 46), 120);
  assert.equal(u16(rom, ASSET_OFFSET + 48), 80);
  assert.equal((u16(rom, ASSET_OFFSET + 50) & 0x0002) !== 0, true);
  assert.equal(result.paddedSize >= 1024 * 1024, true);
  assert.equal((result.paddedSize & (result.paddedSize - 1)) === 0, true);

  const decoded = reconstructFrames(rom, ASSET_OFFSET);
  assert.equal(decoded.length, 3);
  assert.equal(decoded.every((frame) => frame.length === FRAME_BYTES), true);

  let checksumSum = 0;
  for (let index = 0xa0; index < 0xbd; index += 1) checksumSum += rom[index];
  assert.equal(rom[0xbd], (-0x19 - checksumSum) & 0xff);
});

test("browser core supports uncompressed video and per-scene palette metadata", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const frameCount = 130;
  const frames = new Uint8Array(RGB_FRAME_BYTES * frameCount);
  for (let frame = 0; frame < frameCount; frame += 1) {
    const base = frame * RGB_FRAME_BYTES;
    const red = frame < 65 ? 220 : 20;
    const blue = frame < 65 ? 20 : 220;
    for (let pixel = 0; pixel < FRAME_BYTES; pixel += 1) {
      frames[base + pixel * 3] = red;
      frames[base + pixel * 3 + 1] = 30;
      frames[base + pixel * 3 + 2] = blue;
    }
  }

  const clip = convertRawClip({
    framesRGB: frames,
    title: "SCENES",
    vblanks: 8,
    paletteMode: "scene",
    ditherMode: "off",
    compression: "none",
  });
  assert.equal(clip.compressed, false);
  assert.equal(clip.videoIndex.length, 0);
  assert.equal(clip.video.length, frameCount * FRAME_BYTES);
  assert.equal(clip.paletteCount >= 2, true);
  assert.equal(clip.paletteIndex.length, frameCount * 2);

  const result = assembleROM(playerStub, [clip], { romTitle: "SCENES", outputMode: "menu", resume: false });
  const flags = u16(result.rom, ASSET_OFFSET + 50);
  assert.equal((flags & 0x0004) === 0, true);
  assert.equal((flags & 0x0008) !== 0, true);
  assert.equal(u32(result.rom, ASSET_OFFSET + 12), 0);
  assert.equal(u32(result.rom, ASSET_OFFSET + 28) !== 0, true);
});

test("browser core writes split-part title-screen metadata", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const frames = new Uint8Array(RGB_FRAME_BYTES);
  const clip = convertRawClip({
    framesRGB: frames,
    title: "PART TEST",
    vblanks: 8,
    ditherMode: "off",
    compression: "none",
  });
  const result = assembleROM(playerStub, [clip], {
    romTitle: "PART 02",
    outputMode: "rom",
    resume: true,
    titleScreenPart: 2,
    titleScreenName: "A very long movie filename.mp4",
  });
  assert.equal((u16(result.rom, METADATA_OFFSET + 6) & 0x0004) !== 0, true);
  assert.equal(u32(result.rom, METADATA_OFFSET + 20), 2);
  assert.equal(new TextDecoder().decode(result.rom.subarray(METADATA_OFFSET + 24, METADATA_OFFSET + 48)).replace(/\0+$/, ""), "A VERY LONG MOVIE FILENA");
});

test("browser core embeds a configurable animated menu theme", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const framesRGB = new Uint8Array(RGB_FRAME_BYTES);
  const clipA = convertRawClip({ framesRGB, title: "ONE", vblanks: 8, ditherMode: "off", compression: "none" });
  const clipB = convertRawClip({ framesRGB: framesRGB.slice(), title: "TWO", vblanks: 8, ditherMode: "off", compression: "none" });
  const palette = new Uint8Array(512);
  const frame = new Uint8Array(FRAME_BYTES);
  const b64 = (bytes) => Buffer.from(bytes).toString("base64");
  const theme = {
    id: "test-wave", kind: "frames", palette: b64(palette), frames: [b64(frame), b64(frame)],
    frameVBlanks: 12, uiColor: 0x7fff, selectedColor: 0x037f, outline: true, outlineColor: 0,
  };
  const result = assembleROM(playerStub, [clipA, clipB], { romTitle: "THEME", outputMode: "menu", menuTheme: theme });
  const themeOffset = u32(result.rom, METADATA_OFFSET + 48);
  assert.ok(themeOffset > ASSET_OFFSET);
  assert.equal(u32(result.rom, themeOffset), 0x3148544d);
  assert.equal(u16(result.rom, themeOffset + 6), 2);
  assert.equal(u16(result.rom, themeOffset + 16), 2);
  assert.equal(u16(result.rom, themeOffset + 18), 12);
  assert.equal(u16(result.rom, themeOffset + 20), 1);
});

test("Extreme encoding writes adaptive-keyframe and ADPCM metadata", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const frameCount = 20;
  const frames = new Uint8Array(RGB_FRAME_BYTES * frameCount);
  for (let frame = 0; frame < frameCount; frame += 1) {
    const base = frame * RGB_FRAME_BYTES;
    for (let pixel = 0; pixel < FRAME_BYTES; pixel += 1) {
      frames[base + pixel * 3] = (frame * 19 + pixel) & 255;
      frames[base + pixel * 3 + 1] = (frame * 11) & 255;
      frames[base + pixel * 3 + 2] = 180;
    }
  }
  const audio = new Uint8Array(16384);
  for (let index = 0; index < audio.length; index += 1) audio[index] = Math.round(Math.sin(index / 19) * 80) & 255;
  const clip = convertRawClip({
    framesRGB: frames, audio, title: "EXTREME", vblanks: 5,
    paletteMode: "scene", ditherMode: "ordered", compression: "delta",
    adaptiveKeyframes: true, enhancedSceneDetection: true, audioCodec: "adpcm",
  });
  assert.equal(clip.audioCodec, "adpcm");
  assert.equal(clip.adaptiveKeyframes, true);
  assert.ok(clip.audio.length < clip.audioSampleCount * 0.55);
  const result = assembleROM(playerStub, [clip], { romTitle: "EXTREME", outputMode: "rom" });
  const flags = u16(result.rom, ASSET_OFFSET + 50);
  assert.ok(flags & 0x0010, "ADPCM flag missing");
  assert.ok(flags & 0x0020, "adaptive-keyframe flag missing");
  assert.equal(u16(result.rom, ASSET_OFFSET + 56), 0);
  assert.equal(u32(result.rom, ASSET_OFFSET + 80), 2);
  assert.equal(u32(result.rom, ASSET_OFFSET + 84), clip.audioSampleCount);
  assert.equal(u32(result.rom, ASSET_OFFSET + 88), 2048);
  assert.ok(u32(result.rom, ASSET_OFFSET + 92) > 4);
  const seekOffset = u32(result.rom, ASSET_OFFSET + 32);
  assert.ok(seekOffset > 0);
  assert.ok(u32(result.rom, seekOffset + 4) > 0, "ADPCM seek table should store sample positions");
});

test("browser ROM stores Cyrillic clip titles as compact one-byte GBA glyph codes", async () => {
  const playerStub = new Uint8Array(await readFile(new URL("../public/player_stub.bin", import.meta.url)));
  const clip = convertRawClip({
    framesRGB: new Uint8Array(RGB_FRAME_BYTES),
    title: "ВІДЕО ҐЄЇЁ",
    vblanks: 8,
    ditherMode: "off",
    compression: "none",
  });
  const result = assembleROM(playerStub, [clip], { romTitle: "Моє відео", outputMode: "menu" });
  assert.deepEqual([...result.rom.subarray(ASSET_OFFSET + 60, ASSET_OFFSET + 72)], [0x82,0x8c,0x85,0x86,0x93,0x20,0x84,0x87,0x8d,0x88,0x00,0x00]);
  assert.equal(new TextDecoder().decode(result.rom.subarray(0xa0, 0xac)), "MOYE VIDEO  ");
});

test("adaptive keyframes use the desktop accumulated change budget", () => {
  const frameCount = 40;
  const frames = new Uint8Array(RGB_FRAME_BYTES * frameCount);
  const chunkPixels = FRAME_BYTES / 5;
  for (let frame = 0; frame < frameCount; frame += 1) {
    const activeChunk = frame % 5;
    const frameBase = frame * RGB_FRAME_BYTES;
    const startPixel = activeChunk * chunkPixels;
    const endPixel = startPixel + chunkPixels;
    for (let pixel = startPixel; pixel < endPixel; pixel += 1) {
      const offset = frameBase + pixel * 3;
      frames[offset] = 255;
      frames[offset + 1] = 255;
      frames[offset + 2] = 255;
    }
  }
  const clip = convertRawClip({
    framesRGB: frames,
    title: "BUDGET",
    vblanks: 5,
    paletteMode: "shared",
    ditherMode: "off",
    compression: "delta",
    keyInterval: 30,
    adaptiveKeyframes: true,
    enhancedSceneDetection: false,
  });
  const frame30Offset = u32(clip.videoIndex, 30 * 4);
  assert.equal(u32(clip.video, frame30Offset), 0, "frame 30 should be forced to a keyframe by accumulated changes");
});
