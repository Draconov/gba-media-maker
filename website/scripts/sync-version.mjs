import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = resolve(here, "..");
const repository = resolve(website, "..");
const source = resolve(repository, "VERSION");
const output = resolve(website, "src", "generated", "version.js");

const version = (await readFile(source, "utf8")).trim();
if (!/^\d+\.\d+\.\d+$/.test(version)) {
  throw new Error("VERSION must contain a semantic version such as 1.2.3.");
}

await mkdir(dirname(output), { recursive: true });
await writeFile(output, `// Generated from ../../VERSION. Do not edit.\nexport const APP_VERSION = ${JSON.stringify(version)};\n`, "utf8");
console.log(`Synced website version ${version}`);
