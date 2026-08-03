import test from "node:test";
import assert from "node:assert/strict";
import { buildStoredZip } from "../src/zip-store.js";

function u16(bytes, offset) {
  return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint16(offset, true);
}
function u32(bytes, offset) {
  return new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(offset, true);
}

test("stored ZIP contains local headers, central directory and EOCD", () => {
  const zip = buildStoredZip([
    { name: "ONE.gba", data: new Uint8Array([1, 2, 3]) },
    { name: "TWO.gba", data: new Uint8Array([4, 5]) },
  ]);
  assert.equal(u32(zip, 0), 0x04034b50);
  const eocd = zip.length - 22;
  assert.equal(u32(zip, eocd), 0x06054b50);
  assert.equal(u16(zip, eocd + 10), 2);
  const centralOffset = u32(zip, eocd + 16);
  assert.equal(u32(zip, centralOffset), 0x02014b50);
});
