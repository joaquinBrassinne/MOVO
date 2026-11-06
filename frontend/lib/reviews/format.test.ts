import assert from "node:assert/strict";
import { test } from "node:test";

import { formatRating, isNewBranch, starFills } from "./format.ts";

test("a branch with fewer than 3 reviews is new", () => {
  assert.equal(isNewBranch(0), true);
  assert.equal(isNewBranch(2), true);
  assert.equal(isNewBranch(3), false);
});

test("the rating uses a decimal comma", () => {
  assert.equal(formatRating(4.6), "4,6");
  assert.equal(formatRating(5), "5,0");
});

test("stars fill whole, half or empty", () => {
  assert.deepEqual(starFills(4.6), [1, 1, 1, 1, 0.5]);
  assert.deepEqual(starFills(3), [1, 1, 1, 0, 0]);
  assert.deepEqual(starFills(0), [0, 0, 0, 0, 0]);
  assert.deepEqual(starFills(9), [1, 1, 1, 1, 1]);
});
