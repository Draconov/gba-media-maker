import test from "node:test";
import assert from "node:assert/strict";
import {
  encodeGBATextFixed,
  glyphBits,
  glyphLength,
  safeGBAHeaderTitle,
  sanitizeGBAText,
  unsupportedGBARunes,
} from "../../frontend/shared/gba-text.js";

test("one GBA font supports the union of Ukrainian and Russian Cyrillic", () => {
  const alphabet = "АБВГҐДЕЄЁЖЗИІЇЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ";
  const result = sanitizeGBAText(alphabet);
  assert.equal(result.text, alphabet);
  assert.deepEqual(result.unsupported, []);
  for (const character of alphabet) assert.notEqual(glyphBits(character), 0, character);
});

test("lowercase Cyrillic and typographic punctuation normalize to the shared pixel font", () => {
  assert.equal(sanitizeGBAText("Привіт — п’ять… Ёжик").text, "ПРИВІТ - П'ЯТЬ... ЁЖИК");
});

test("glyph limits count characters rather than UTF-8 bytes", () => {
  const result = sanitizeGBAText("Українськийт", 12).text;
  assert.equal(glyphLength(result), 12);
  assert.ok(new TextEncoder().encode(result).length > 12);
});

test("runtime Cyrillic menu text is one byte per GBA glyph", () => {
  assert.deepEqual(
    [...encodeGBATextFixed("АБВҐЄЇЁЯ№", 12)],
    [0x80, 0x81, 0x82, 0x84, 0x87, 0x8d, 0x88, 0xa4, 0xa5, 0, 0, 0],
  );
});

test("unsupported characters are reported while common Unicode punctuation is accepted", () => {
  assert.deepEqual(unsupportedGBARunes("Відео 😀 日本"), ["😀", "日", "本"]);
  assert.deepEqual(unsupportedGBARunes("п’ять — так…"), []);
});

test("GBA cartridge header remains ASCII-safe through Cyrillic transliteration", () => {
  assert.equal(new TextDecoder().decode(safeGBAHeaderTitle("Моє відео")), "MOYE VIDEO  ");
});


test("similar Cyrillic letters keep distinct pixel glyphs", () => {
  assert.notEqual(glyphBits("И"), glyphBits("Н"));
  assert.notEqual(glyphBits("Й"), glyphBits("А"));
  assert.notEqual(glyphBits("Щ"), glyphBits("Ц"));
});


test('slash glyph stays diagonal', () => {
  assert.equal(glyphBits('/'), 0x12A4);
});
