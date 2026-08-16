import test from "node:test";
import assert from "node:assert/strict";
import { readFile, access } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = dirname(here);
const root = dirname(website);

async function json(path) { return JSON.parse(await readFile(path, "utf8")); }

test("website uses the canonical manifest-driven localization catalogs", async () => {
  const manifest = await json(join(root, "locales", "index.json"));
  const generatedManifest = await json(join(website, "public", "locales", "index.json"));
  const en = await json(join(root, "locales", "en.json"));
  const uk = await json(join(root, "locales", "uk.json"));
  const fr = await json(join(root, "locales", "fr.json"));
  const es = await json(join(root, "locales", "es.json"));
  const de = await json(join(root, "locales", "de.json"));
  const generatedUk = await json(join(website, "public", "locales", "uk.json"));
  const generatedFr = await json(join(website, "public", "locales", "fr.json"));
  const generatedEs = await json(join(website, "public", "locales", "es.json"));
  const generatedDe = await json(join(website, "public", "locales", "de.json"));
  assert.equal(manifest.fallback, "en");
  assert.deepEqual(manifest.languages.map(item => item.code), ["en", "uk", "fr", "es", "de"]);
  assert.deepEqual(manifest.languages.map(item => item.short), ["EN", "UA", "FR", "ES", "DE"]);
  assert.deepEqual(manifest.languages.map(item => item.flag), ["🇬🇧", "🇺🇦", "🇫🇷", "🇪🇸", "🇩🇪"]);
  assert.deepEqual(generatedManifest, manifest);
  assert.equal(en.meta.code, "en");
  assert.equal(uk.meta.code, "uk");
  assert.equal(uk.meta.fallback, "en");
  assert.equal(fr.meta.fallback, "en");
  assert.equal(es.meta.fallback, "en");
  assert.equal(de.meta.fallback, "en");
  assert.equal(uk.messages.Language, "Мова");
  assert.equal(uk.messages["Choose language"], "Обрати мову");
  assert.equal(fr.messages.Language, "Langue");
  assert.equal(es.messages.Language, "Idioma");
  assert.equal(de.messages.Language, "Sprache");
  assert.deepEqual(generatedUk, uk);
  assert.deepEqual(generatedFr, fr);
  assert.deepEqual(generatedEs, es);
  assert.deepEqual(generatedDe, de);
  for (const catalog of [uk, fr, es, de]) assert.ok(Object.keys(catalog.messages).length * 100 >= Object.keys(en.messages).length * 80);
  await assert.rejects(access(join(root, "locales", "ru.json")));
  await assert.rejects(access(join(website, "public", "locales", "flag-gb.svg")));
  await assert.rejects(access(join(website, "public", "locales", "flag-ua.svg")));
  await assert.rejects(access(join(website, "public", "locales", "flag-fr.svg")));
  await assert.rejects(access(join(website, "public", "locales", "flag-es.svg")));
  await assert.rejects(access(join(website, "public", "locales", "flag-de.svg")));
});

test("website bootstraps localization and exposes the expandable language menu", async () => {
  const html = await readFile(join(website, "index.html"), "utf8");
  const main = await readFile(join(website, "src", "main.js"), "utf8");
  const i18n = await readFile(join(website, "src", "i18n.js"), "utf8");
  const pkg = await json(join(website, "package.json"));
  assert.match(html, /data-i18n-language-host/);
  assert.ok(html.indexOf('id="desktopLink"') < html.indexOf("data-i18n-language-host"));
  assert.match(i18n, /MANIFEST_URL = "locales\/index\.json"/);
  assert.match(i18n, /language-menu-button/);
  assert.match(i18n, /language-menu-option/);
  assert.match(i18n, /aria-haspopup=\"menu\"/);
  assert.match(i18n, /textContent = display\.flag/);
  assert.doesNotMatch(i18n, /language-flag-image/);
  assert.doesNotMatch(i18n, /setLanguage\(language ===/);
  assert.match(main, /import \{ initI18n \} from "\.\/i18n\.js";/);
  assert.match(main, /await initI18n\(\);/);
  assert.match(pkg.scripts["sync-runtime"], /sync-locales/);
});
