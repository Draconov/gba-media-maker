import test from "node:test";
import assert from "node:assert/strict";
import { FRAME_BYTES, RGB24_FRAME_BYTES, decodeRGB24Frames, isVideoFile } from "../src/menu-themes.js";

test("custom menu backgrounds recognize common video files", () => {
  assert.equal(isVideoFile({ name: "background.mp4", type: "video/mp4" }), true);
  assert.equal(isVideoFile({ name: "background.mkv", type: "" }), true);
  assert.equal(isVideoFile({ name: "background.mov", type: "video/quicktime" }), true);
  assert.equal(isVideoFile({ name: "background.gif", type: "image/gif" }), false);
  assert.equal(isVideoFile({ name: "background.png", type: "image/png" }), false);
});

test("RGB24 menu-video frames use the existing indexed custom-theme format", () => {
  const bytes = new Uint8Array(RGB24_FRAME_BYTES * 2);
  for (let i = 0; i < FRAME_BYTES; i++) {
    bytes[i * 3] = 255;
    bytes[RGB24_FRAME_BYTES + i * 3 + 1] = 255;
  }
  const progress = [];
  const theme = decodeRGB24Frames(bytes, "menu.mp4", 15, {}, value => progress.push(value));
  assert.equal(theme.kind, "frames");
  assert.equal(theme.frames.length, 2);
  assert.equal(theme.frames[0].length, FRAME_BYTES);
  assert.equal(theme.frameVBlanks, 15);
  assert.equal(theme.name, "menu.mp4");
  assert.equal(progress.at(-1), 1);
  assert.notDeepEqual(theme.frames[0], theme.frames[1]);
});
