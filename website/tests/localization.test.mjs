import test from "node:test";
import assert from "node:assert/strict";
import { readFile, access } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = dirname(here);
const root = dirname(website);

async function json(path) { return JSON.parse(await readFile(path, "utf8")); }

test("website uses the canonical EN/UK localization catalogs", async () => {
  const en = await json(join(root, "locales", "en.json"));
  const uk = await json(join(root, "locales", "uk.json"));
  const generatedUk = await json(join(website, "public", "locales", "uk.json"));
  assert.equal(en.meta.code, "en");
  assert.equal(uk.meta.code, "uk");
  assert.equal(uk.meta.fallback, "en");
  assert.equal(uk.messages.Language, "Мова");
  assert.deepEqual(generatedUk, uk);
  assert.ok(Object.keys(uk.messages).length * 100 >= Object.keys(en.messages).length * 80);
  await assert.rejects(access(join(root, "locales", "ru.json")));
});

test("website bootstraps localization and exposes a language selector", async () => {
  const html = await readFile(join(website, "index.html"), "utf8");
  const main = await readFile(join(website, "src", "main.js"), "utf8");
  const pkg = await json(join(website, "package.json"));
  assert.match(html, /data-i18n-language-host/);
  assert.match(main, /import \{ initI18n \} from "\.\/i18n\.js";/);
  assert.match(main, /await initI18n\(\);/);
  assert.match(pkg.scripts["sync-runtime"], /sync-locales/);
});
