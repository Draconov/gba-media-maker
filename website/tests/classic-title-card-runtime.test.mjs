import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import vm from "node:vm";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../..");

function runClassicAsset(context, relativePath) {
  const source = fs.readFileSync(path.join(root, relativePath), "utf8");
  vm.runInContext(source, context, { filename: relativePath });
}

test("classic desktop title-card script creates TitleCardTools after GBAText", () => {
  const sandbox = {
    console,
    Uint8Array,
    Uint16Array,
    DataView,
    TextEncoder,
    Math,
    Object,
    Array,
    String,
    Number,
    RegExp,
    Set,
  };
  sandbox.window = sandbox;
  const context = vm.createContext(sandbox);
  runClassicAsset(context, "web/gba-text.js");
  assert.ok(context.GBAText, "GBAText global was not created");
  runClassicAsset(context, "web/title-cards.js");
  assert.ok(context.TitleCardTools, "TitleCardTools global was not created");
  assert.equal(typeof context.TitleCardTools.createTitleCardProject, "function");
  assert.equal(typeof context.TitleCardTools.buildTitleCardAsset, "function");
});

test("desktop serializer checks optional TitleCardTools through window", () => {
  const source = fs.readFileSync(path.join(root, "web/app.js"), "utf8");
  assert.match(source, /function serializeTitleCards\(\) \{\s*if \(!window\.TitleCardTools\) return null;/);
  assert.doesNotMatch(source, /(^|[^.\w])TitleCardTools\./m);
});
