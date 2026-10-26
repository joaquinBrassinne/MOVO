import assert from "node:assert/strict";
import { test } from "node:test";

import { coverFirst, moveInOrder } from "./photoOrder.ts";

test("moveInOrder swaps a photo with its neighbour and leaves the rest", () => {
  assert.deepEqual(moveInOrder([1, 2, 3, 4], 3, -1), [1, 3, 2, 4]);
  assert.deepEqual(moveInOrder([1, 2, 3, 4], 2, 1), [1, 3, 2, 4]);
});

test("moveInOrder does not move past either end or an unknown photo", () => {
  assert.equal(moveInOrder([1, 2, 3], 1, -1), null);
  assert.equal(moveInOrder([1, 2, 3], 3, 1), null);
  assert.equal(moveInOrder([1, 2, 3], 9, 1), null);
  assert.equal(moveInOrder([], 1, 1), null);
});

test("moveInOrder never changes the list it was given", () => {
  const ids = [1, 2, 3];
  moveInOrder(ids, 2, -1);
  assert.deepEqual(ids, [1, 2, 3]);
});

test("coverFirst puts the photo at the front and keeps the order of the others", () => {
  assert.deepEqual(coverFirst([1, 2, 3, 4], 3), [3, 1, 2, 4]);
});

test("coverFirst is null when the photo already is the cover or does not exist", () => {
  assert.equal(coverFirst([1, 2, 3], 1), null);
  assert.equal(coverFirst([1, 2, 3], 9), null);
});
