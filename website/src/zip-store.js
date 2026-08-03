function crc32(bytes) {
  let crc = 0xffffffff;
  for (const byte of bytes) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) crc = (crc >>> 1) ^ (0xedb88320 & -(crc & 1));
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function set16(view, offset, value) { view.setUint16(offset, value, true); }
function set32(view, offset, value) { view.setUint32(offset, value >>> 0, true); }

export function buildStoredZip(files) {
  if (!Array.isArray(files) || files.length < 1) throw new Error("At least one ZIP entry is required.");
  if (files.length > 65535) throw new Error("Too many ZIP entries.");
  const encoder = new TextEncoder();
  const encoded = files.map((file) => {
    if (!(file.data instanceof Uint8Array)) throw new TypeError("ZIP entry data must be a Uint8Array.");
    const nameBytes = encoder.encode(String(file.name || "file.bin"));
    if (nameBytes.length > 65535) throw new Error("ZIP entry name is too long.");
    return { ...file, nameBytes, crc: crc32(file.data) };
  });
  const localSize = encoded.reduce((sum, file) => sum + 30 + file.nameBytes.length + file.data.length, 0);
  const centralSize = encoded.reduce((sum, file) => sum + 46 + file.nameBytes.length, 0);
  const out = new Uint8Array(localSize + centralSize + 22);
  const view = new DataView(out.buffer);
  let position = 0;
  const central = [];

  for (const file of encoded) {
    const localOffset = position;
    set32(view, position, 0x04034b50); position += 4;
    set16(view, position, 20); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set32(view, position, file.crc); position += 4;
    set32(view, position, file.data.length); position += 4;
    set32(view, position, file.data.length); position += 4;
    set16(view, position, file.nameBytes.length); position += 2;
    set16(view, position, 0); position += 2;
    out.set(file.nameBytes, position); position += file.nameBytes.length;
    out.set(file.data, position); position += file.data.length;
    central.push({ file, localOffset });
  }

  const centralOffset = position;
  for (const { file, localOffset } of central) {
    set32(view, position, 0x02014b50); position += 4;
    set16(view, position, 20); position += 2;
    set16(view, position, 20); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set32(view, position, file.crc); position += 4;
    set32(view, position, file.data.length); position += 4;
    set32(view, position, file.data.length); position += 4;
    set16(view, position, file.nameBytes.length); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set16(view, position, 0); position += 2;
    set32(view, position, 0); position += 4;
    set32(view, position, localOffset); position += 4;
    out.set(file.nameBytes, position); position += file.nameBytes.length;
  }

  set32(view, position, 0x06054b50); position += 4;
  set16(view, position, 0); position += 2;
  set16(view, position, 0); position += 2;
  set16(view, position, files.length); position += 2;
  set16(view, position, files.length); position += 2;
  set32(view, position, centralSize); position += 4;
  set32(view, position, centralOffset); position += 4;
  set16(view, position, 0);
  return out;
}
