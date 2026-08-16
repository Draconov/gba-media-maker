import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import * as GBAText from "../../frontend/shared/gba-text.js";
import * as MenuThemeTools from "../../frontend/shared/menu-themes.js";
import * as TitleCardTools from "../../frontend/shared/title-cards.js";

const here = dirname(fileURLToPath(import.meta.url));
const repository = resolve(here, "../..");

async function missing(path) {
  try {
    await access(path);
    return false;
  } catch {
    return true;
  }
}

test("desktop and website share one canonical frontend implementation", async () => {
  assert.equal(typeof GBAText.sanitizeGBAText, "function");
  assert.equal(typeof MenuThemeTools.createBuiltinTheme, "function");
  assert.equal(typeof TitleCardTools.createTitleCardProject, "function");
  assert.equal(typeof TitleCardTools.buildTitleCardAsset, "function");

  for (const duplicate of [
    "web/gba-text.js",
    "web/menu-themes.js",
    "web/title-cards.js",
    "website/src/gba-text.js",
    "website/src/menu-themes.js",
    "website/src/title-cards.js",
  ]) {
    assert.equal(await missing(resolve(repository, duplicate)), true, `${duplicate} must not be a second source copy`);
  }
});

test("both frontends consume the canonical shared modules", async () => {
  const [desktop, website] = await Promise.all([
    readFile(resolve(repository, "web/app.js"), "utf8"),
    readFile(resolve(repository, "website/src/main.js"), "utf8"),
  ]);

  for (const module of ["gba-text.js", "menu-themes.js", "title-cards.js"]) {
    assert.match(desktop, new RegExp(`import\\('\\./shared/${module.replace(".", "\\.")}\\'\\)`));
    assert.match(website, new RegExp(`frontend/shared/${module.replace(".", "\\.")}`));
  }
  assert.doesNotMatch(desktop, /window\.(?:GBAText|MenuThemeTools|TitleCardTools)/);
});
