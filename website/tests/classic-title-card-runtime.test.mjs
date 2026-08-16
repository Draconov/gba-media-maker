import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import * as GBAText from "../../frontend/shared/gba-text.js";
import * as TitleCardTools from "../../frontend/shared/title-cards.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../..");

test("shared desktop title-card runtime loads with the canonical GBA text module", () => {
  assert.equal(typeof GBAText.sanitizeGBAText, "function");
  assert.equal(typeof TitleCardTools.createTitleCardProject, "function");
  assert.equal(typeof TitleCardTools.buildTitleCardAsset, "function");

  const project = TitleCardTools.createTitleCardProject("Example.mp4");
  assert.ok(project);
  assert.ok(project.shared);
});

test("desktop serializer uses the imported shared TitleCardTools module", () => {
  const source = fs.readFileSync(path.join(root, "web/app.js"), "utf8");

  assert.match(source, /const \[GBAText, MenuThemeTools, TitleCardTools\] = await Promise\.all\(\[/);
  assert.match(source, /import\('\.\/shared\/title-cards\.js'\)/);
  assert.match(source, /function serializeTitleCards\(\) \{\s*ensureTitleCardProject\(\);/);
  assert.match(source, /TitleCardTools\.createTitleCardProject\(titleCardSourceName\(\)\)/);
  assert.doesNotMatch(source, /window\.TitleCardTools/);
});
