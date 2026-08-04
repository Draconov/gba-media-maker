import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const website = resolve(here, "..");
const repository = resolve(website, "..");

test("website uses the desktop application icon", async () => {
  const [desktopPng, webPng, desktopIco, webIco, html, manifest] = await Promise.all([
    readFile(resolve(repository, "assets/app_icon.png")),
    readFile(resolve(website, "public/icon.png")),
    readFile(resolve(repository, "assets/app_icon.ico")),
    readFile(resolve(website, "public/favicon.ico")),
    readFile(resolve(website, "index.html"), "utf8"),
    readFile(resolve(website, "public/site.webmanifest"), "utf8"),
  ]);

  assert.deepEqual(webPng, desktopPng, "website PNG must exactly match assets/app_icon.png");
  assert.deepEqual(webIco, desktopIco, "website ICO must exactly match assets/app_icon.ico");
  assert.match(html, /rel="icon" href="\.\/favicon\.ico"/);
  assert.match(html, /rel="apple-touch-icon" href="\.\/apple-touch-icon\.png"/);
  assert.match(html, /rel="manifest" href="\.\/site\.webmanifest"/);
  assert.match(html, /class="site-icon" src="\.\/icon\.png"/);

  const parsed = JSON.parse(manifest);
  assert.equal(parsed.name, "GBA Video Maker Web");
  assert.ok(parsed.icons.some((icon) => icon.sizes === "192x192"));
  assert.ok(parsed.icons.some((icon) => icon.sizes === "512x512"));
});
