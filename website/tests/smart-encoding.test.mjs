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
  assert.ok(!Number.isNaN(Date.parse(result.analyzedAt)));
});

test("smart candidate quality and priority ranking match the desktop formulas", () => {
  const redFrames = new Uint8Array(RGB_FRAME_BYTES * 24);
  for (let i = 0; i < redFrames.length; i += 3) redFrames[i] = 255;
  const colourResult = analyzeSmartScan({ framesRGB: redFrames, duration: 30, targetBytes: 32 << 20, audioQuality: "pcm" });
  const balanced = colourResult.candidates.find((candidate) => candidate.id === "balanced");
  assert.equal(balanced.temporalStability, 96, "scene-palette stability includes the desktop colour bonus");

  const frames = new Uint8Array(RGB_FRAME_BYTES * 36);
  for (let frame = 0; frame < 36; frame += 1) {
    for (let i = 0; i < RGB_FRAME_BYTES; i += 3) {
      frames[frame * RGB_FRAME_BYTES + i] = (i + frame * 17) & 255;
      frames[frame * RGB_FRAME_BYTES + i + 1] = (i * 3 + frame * 9) & 255;
      frames[frame * RGB_FRAME_BYTES + i + 2] = (i * 7 + frame * 5) & 255;
    }
  }
  const motion = analyzeSmartScan({ framesRGB: frames, duration: 120, targetBytes: 8 << 20, priority: "motion", audioQuality: "auto" });
  const detail = analyzeSmartScan({ framesRGB: frames, duration: 120, targetBytes: 8 << 20, priority: "detail", audioQuality: "auto" });
  assert.equal(motion.recommended.id, "balanced");
  assert.equal(detail.recommended.id, "balanced");
});


test("smart sample timestamps stay on the original source timeline", () => {
  const frames = new Uint8Array(RGB_FRAME_BYTES * 24);
  for (let frame = 0; frame < 24; frame += 1) {
    frames.fill((frame * 11) & 255, frame * RGB_FRAME_BYTES, (frame + 1) * RGB_FRAME_BYTES);
  }
  const result = analyzeSmartScan({ framesRGB: frames, duration: 10, sourceStart: 20, sourceEnd: 40 });
  assert.ok(result.samples.length > 0);
  assert.ok(result.samples.every((sample) => sample.time >= 20 && sample.time <= 40));
  assert.ok(result.samples.some((sample) => sample.time > 30));
  assert.equal("scene" in result.samples[0], false, "browser result schema does not leak internal scan-only fields");
});
