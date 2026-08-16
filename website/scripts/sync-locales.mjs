import { cp, mkdir, readdir, rm } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = dirname(here);
const source = join(website, "..", "locales");
const target = join(website, "public", "locales");
await rm(target, { recursive: true, force: true });
await mkdir(target, { recursive: true });
const files = (await readdir(source)).filter(name => name.endsWith(".json")).sort();
for (const name of files) await cp(join(source, name), join(target, name));
console.log(`Synced ${files.length} website locale JSON files.`);
