import assert from "node:assert/strict";
import test from "node:test";
import { describeColor, hexToRGB555, quantizeHexColor, rgb555ToHex, settingsColours } from "../src/menu-themes.js";

test("menu colours are quantized to the GBA 15-bit RGB palette", () => {
  assert.equal(hexToRGB555("#ffffff"), 0x7fff);
  assert.equal(rgb555ToHex(0x7fff), "#FFFFFF");
  assert.equal(quantizeHexColor("#123456"), rgb555ToHex(hexToRGB555("#123456")));
  const color = describeColor("#ffde00");
  assert.equal(color.rgb15, 0x037f);
  assert.deepEqual([color.r, color.g, color.b], [31, 27, 0]);
});

test("menu colour settings support independent colours and legacy presets", () => {
  const custom = settingsColours({ uiColor: "#102030", selectedColor: "#a0b0c0", outlineColor: "#405060" });
  assert.equal(custom.ui, hexToRGB555("#102030"));
  assert.equal(custom.selected, hexToRGB555("#a0b0c0"));
  assert.equal(custom.outline, hexToRGB555("#405060"));

  const legacy = settingsColours({ uiColor: "cyan", outlineColor: "navy" });
  assert.equal(legacy.ui, 0x7fe0);
  assert.equal(legacy.selected, 0x7fff);
  assert.equal(legacy.outline, 0x2400);
});
