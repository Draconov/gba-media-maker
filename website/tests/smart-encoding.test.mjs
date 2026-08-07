import test from "node:test";
import assert from "node:assert/strict";
import { analyzeSmartScan } from "../src/smart-encoding.js";
import { RGB_FRAME_BYTES } from "../src/rom-core.js";

test("smart analysis returns representative samples and ranked candidates", () => {
  const frames = new Uint8Array(RGB_FRAME_BYTES * 36);
  for (let frame = 0; frame < 36; frame += 1) {
    for (let i = 0; i < RGB_FRAME_BYTES; i += 3) {
      frames[frame * RGB_FRAME_BYTES + i] = (i + frame * 17) & 255;
      frames[frame * RGB_FRAME_BYTES + i + 1] = (i * 3 + frame * 9) & 255;
      frames[frame * RGB_FRAME_BYTES + i + 2] = (i * 7 + frame * 5) & 255;
    }
  }
  const result = analyzeSmartScan({ framesRGB: frames, duration: 600, targetBytes: 32 << 20, audioQuality: "auto" });
  assert.ok(result.samples.length >= 4);
  assert.ok(result.candidates.length >= 5);
  assert.ok(result.recommended.estimatedBytes > 0);
  assert.match(result.confidence, /^(low|medium|high)$/);
});
