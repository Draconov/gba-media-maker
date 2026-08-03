import { copyFile, mkdir, stat } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const source = resolve(here, "../../assets/player_stub.bin");
const destination = resolve(here, "../public/player_stub.bin");
await mkdir(dirname(destination), { recursive: true });
await copyFile(source, destination);
const info = await stat(destination);
if (info.size !== 0x8000) {
  throw new Error(`player_stub.bin is ${info.size} bytes; the web converter expects 32768 bytes.`);
}
console.log(`Synced ${destination} (${info.size} bytes)`);
