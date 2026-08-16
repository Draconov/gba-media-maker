import { copyFile, mkdir, stat } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = resolve(here, "..");
const repository = resolve(website, "..");
const sourceDir = resolve(repository, "assets", "audio-artwork");
const outputDir = resolve(website, "public", "audio-artwork");

await mkdir(outputDir, { recursive: true });
for (let index = 1; index <= 20; index += 1) {
  const name = `preset-${String(index).padStart(2, "0")}.png`;
  const source = resolve(sourceDir, name);
  const output = resolve(outputDir, name);
  const info = await stat(source);
  if (!info.isFile() || info.size < 8) throw new Error(`Invalid artwork preset: ${name}`);
  await copyFile(source, output);
}
console.log("Synced 20 browser artwork presets");
