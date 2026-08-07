export const ADPCM_MAGIC = "IAD1";
export const ADPCM_VERSION = 1;
export const ADPCM_HEADER_BYTES = 20;
export const DEFAULT_ADPCM_BLOCK_SAMPLES = 2048;

const INDEX_TABLE = [-1, -1, -1, -1, 2, 4, 6, 8, -1, -1, -1, -1, 2, 4, 6, 8];
const STEP_TABLE = [
  7, 8, 9, 10, 11, 12, 13, 14, 16, 17, 19, 21, 23, 25, 28, 31,
  34, 37, 41, 45, 50, 55, 60, 66, 73, 80, 88, 97, 107, 118, 130, 143,
  157, 173, 190, 209, 230, 253, 279, 307, 337, 371, 408, 449, 494, 544, 598, 658,
  724, 796, 876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066, 2272, 2499, 2749, 3024,
  3327, 3660, 4026, 4428, 4871, 5358, 5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899,
  15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767,
];

const clamp = (value, low, high) => Math.max(low, Math.min(high, value));
const signed8 = (value) => (value << 24) >> 24;

function setU16(target, offset, value) {
  target[offset] = value & 0xff;
  target[offset + 1] = (value >>> 8) & 0xff;
}
function setU32(target, offset, value) {
  target[offset] = value & 0xff;
  target[offset + 1] = (value >>> 8) & 0xff;
  target[offset + 2] = (value >>> 16) & 0xff;
  target[offset + 3] = (value >>> 24) & 0xff;
}
function getU16(source, offset) { return source[offset] | (source[offset + 1] << 8); }
function getS16(source, offset) { return (getU16(source, offset) << 16) >> 16; }
function getU32(source, offset) { return (source[offset] | (source[offset + 1] << 8) | (source[offset + 2] << 16) | (source[offset + 3] << 24)) >>> 0; }

function initialIndex(pcm, start, count) {
  let maxDelta = 0;
  let previous = signed8(pcm[start]) << 8;
  const limit = Math.min(count, 32);
  for (let i = 1; i < limit && start + i < pcm.length; i += 1) {
    const current = signed8(pcm[start + i]) << 8;
    maxDelta = Math.max(maxDelta, Math.abs(current - previous));
    previous = current;
  }
  let index = 0;
  while (index < STEP_TABLE.length - 1 && STEP_TABLE[index] < maxDelta / 2) index += 1;
  return index;
}

function encodeNibble(sample, state) {
  const step = STEP_TABLE[state.index];
  let difference = sample - state.predictor;
  let code = 0;
  if (difference < 0) { code = 8; difference = -difference; }
  let delta = step >> 3;
  if (difference >= step) { code |= 4; difference -= step; delta += step; }
  if (difference >= (step >> 1)) { code |= 2; difference -= step >> 1; delta += step >> 1; }
  if (difference >= (step >> 2)) { code |= 1; delta += step >> 2; }
  state.predictor = clamp(state.predictor + ((code & 8) ? -delta : delta), -32768, 32767);
  state.index = clamp(state.index + INDEX_TABLE[code], 0, 88);
  return code;
}

function decodeNibble(code, state) {
  const step = STEP_TABLE[state.index];
  let delta = step >> 3;
  if (code & 4) delta += step;
  if (code & 2) delta += step >> 1;
  if (code & 1) delta += step >> 2;
  state.predictor = clamp(state.predictor + ((code & 8) ? -delta : delta), -32768, 32767);
  state.index = clamp(state.index + INDEX_TABLE[code & 15], 0, 88);
  return state.predictor;
}

export function encodeIMAADPCM(pcm, blockSamples = DEFAULT_ADPCM_BLOCK_SAMPLES) {
  if (!(pcm instanceof Uint8Array)) throw new TypeError("PCM input must be a Uint8Array.");
  if (blockSamples < 64 || blockSamples > 16384) throw new Error("Invalid ADPCM block size.");
  const blockBytes = 4 + Math.ceil((blockSamples - 1) / 2);
  const blockCount = pcm.length ? Math.ceil(pcm.length / blockSamples) : 0;
  const output = new Uint8Array(ADPCM_HEADER_BYTES + blockCount * blockBytes);
  output.set(new TextEncoder().encode(ADPCM_MAGIC), 0);
  setU16(output, 4, ADPCM_VERSION);
  setU16(output, 6, blockSamples);
  setU32(output, 8, pcm.length);
  setU32(output, 12, blockBytes);
  setU32(output, 16, blockCount);
  for (let block = 0; block < blockCount; block += 1) {
    const start = block * blockSamples;
    const count = Math.min(blockSamples, pcm.length - start);
    const base = ADPCM_HEADER_BYTES + block * blockBytes;
    const state = { predictor: signed8(pcm[start]) << 8, index: initialIndex(pcm, start, count) };
    setU16(output, base, state.predictor & 0xffff);
    output[base + 2] = state.index;
    for (let i = 1; i < blockSamples; i += 1) {
      const sample = i < count ? signed8(pcm[start + i]) << 8 : state.predictor;
      const code = encodeNibble(sample, state);
      const position = base + 4 + ((i - 1) >> 1);
      if ((i - 1) & 1) output[position] |= code << 4;
      else output[position] = code;
    }
  }
  return { data: output, blockSamples, blockBytes, blockCount, sampleCount: pcm.length };
}

export function parseIMAADPCM(data) {
  if (!(data instanceof Uint8Array) || data.length < ADPCM_HEADER_BYTES) throw new Error("Invalid ADPCM stream.");
  if (new TextDecoder().decode(data.subarray(0, 4)) !== ADPCM_MAGIC) throw new Error("Invalid ADPCM stream.");
  if (getU16(data, 4) !== ADPCM_VERSION) throw new Error("Unsupported ADPCM version.");
  const info = { blockSamples: getU16(data, 6), sampleCount: getU32(data, 8), blockBytes: getU32(data, 12), blockCount: getU32(data, 16) };
  if (!info.blockSamples || info.blockBytes < 4 || ADPCM_HEADER_BYTES + info.blockBytes * info.blockCount > data.length) throw new Error("Truncated ADPCM stream.");
  return info;
}

export function decodeIMAADPCM(data) {
  const info = parseIMAADPCM(data);
  const pcm = new Uint8Array(info.sampleCount);
  let written = 0;
  for (let block = 0; block < info.blockCount && written < pcm.length; block += 1) {
    const base = ADPCM_HEADER_BYTES + block * info.blockBytes;
    const state = { predictor: getS16(data, base), index: clamp(data[base + 2], 0, 88) };
    pcm[written++] = (state.predictor >> 8) & 0xff;
    for (let i = 1; i < info.blockSamples && written < pcm.length; i += 1) {
      const packed = data[base + 4 + ((i - 1) >> 1)];
      const code = ((i - 1) & 1) ? (packed >> 4) & 15 : packed & 15;
      pcm[written++] = (decodeNibble(code, state) >> 8) & 0xff;
    }
  }
  return { pcm, ...info };
}
