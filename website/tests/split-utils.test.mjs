import assert from "node:assert/strict";
import test from "node:test";
import { chooseChapterEnd, formatClock, parseClock } from "../src/split-utils.js";

test("split duration accepts seconds, MM:SS and H:MM:SS", () => {
  assert.equal(parseClock("0"), 0);
  assert.equal(parseClock("65"), 65);
  assert.equal(parseClock("1:05"), 65);
  assert.equal(parseClock("1:02:03"), 3723);
  assert.equal(Number.isNaN(parseClock("1:xx")), true);
  assert.equal(formatClock(65), "1:05");
  assert.equal(formatClock(3723), "1:02:03");
});

test("chapter-aware splitting chooses a nearby earlier boundary", () => {
  assert.equal(chooseChapterEnd([60, 120, 180], 0, 128, 300), 120);
  assert.equal(chooseChapterEnd([60, 120, 180], 0, 170, 300), 170);
  assert.equal(chooseChapterEnd([], 0, 128, 300), 128);
});
