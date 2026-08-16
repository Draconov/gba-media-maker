import { cp, mkdir, rm } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = dirname(here);
const source = join(website, "..", "locales");
const target = join(website, "public", "locales");
await rm(target, { recursive: true, force: true });
await mkdir(target, { recursive: true });
for (const name of ["en.json", "uk.json"]) await cp(join(source, name), join(target, name));
console.log("Synced EN/UK locale catalogs.");
