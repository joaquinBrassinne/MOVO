import assert from "node:assert/strict";
import { test } from "node:test";

import { formatDistance } from "./format.ts";

test("formatDistance shows meters under a kilometer, rounded to 50, and switches to km when it would round to 1000", () => {
  assert.equal(formatDistance(452), "450 m");
  assert.equal(formatDistance(649), "650 m");
  assert.equal(formatDistance(980), "1.0 km");
});

test("formatDistance never reports less than 50 m", () => {
  assert.equal(formatDistance(0), "50 m");
  assert.equal(formatDistance(12), "50 m");
});

test("formatDistance shows kilometers with one decimal from 1000 m", () => {
  assert.equal(formatDistance(1000), "1.0 km");
  assert.equal(formatDistance(1420), "1.4 km");
  assert.equal(formatDistance(4960), "5.0 km");
});
