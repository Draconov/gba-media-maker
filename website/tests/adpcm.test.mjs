import test from "node:test";
import assert from "node:assert/strict";
import { encodeIMAADPCM, decodeIMAADPCM } from "../src/adpcm.js";

test("IMA ADPCM cuts signed 8-bit mono storage roughly in half", () => {
  const pcm = new Uint8Array(32768 + 91);
  for (let i = 0; i < pcm.length; i += 1) pcm[i] = Math.round(Math.sin(i * 0.08) * 88) & 0xff;
  const encoded = encodeIMAADPCM(pcm);
  assert.ok(encoded.data.length < pcm.length * 0.6);
  const decoded = decodeIMAADPCM(encoded.data);
  assert.equal(decoded.pcm.length, pcm.length);
  let mse = 0;
  for (let i = 0; i < pcm.length; i += 1) {
    const a = (pcm[i] << 24) >> 24;
    const b = (decoded.pcm[i] << 24) >> 24;
    mse += (a - b) ** 2;
  }
  mse /= pcm.length;
  assert.ok(mse < 400, `MSE ${mse}`);
});
