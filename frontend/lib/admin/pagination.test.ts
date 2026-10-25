import assert from "node:assert/strict";
import { test } from "node:test";

import { pageCount, pageWindow } from "./pagination.ts";

test("pageWindow centers the current page", () => {
  assert.deepEqual(pageWindow(5, 10), [3, 4, 5, 6, 7]);
});

test("pageWindow stays inside the first and last pages", () => {
  assert.deepEqual(pageWindow(1, 10), [1, 2, 3, 4, 5]);
  assert.deepEqual(pageWindow(2, 10), [1, 2, 3, 4, 5]);
  assert.deepEqual(pageWindow(10, 10), [6, 7, 8, 9, 10]);
  assert.deepEqual(pageWindow(9, 10), [6, 7, 8, 9, 10]);
});

test("pageWindow shows every page when there are few", () => {
  assert.deepEqual(pageWindow(1, 3), [1, 2, 3]);
  assert.deepEqual(pageWindow(1, 1), [1]);
});

test("pageWindow of nothing is empty", () => {
  assert.deepEqual(pageWindow(1, 0), []);
});

test("pageCount rounds up", () => {
  assert.equal(pageCount(48, 10), 5);
  assert.equal(pageCount(50, 10), 5);
  assert.equal(pageCount(51, 10), 6);
  assert.equal(pageCount(0, 10), 0);
});
