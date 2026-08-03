import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import {
  ASSET_OFFSET,
  FRAME_BYTES,
  METADATA_OFFSET,
  RGB_FRAME_BYTES,
  assembleROM,
  convertRawClip,
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
