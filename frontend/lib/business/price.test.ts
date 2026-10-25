import assert from "node:assert/strict";
import { test } from "node:test";

import { formatPrice, MAX_PRICE, parsePrice } from "./price.ts";

test("parsePrice reads whole pesos however the owner writes them", () => {
  assert.equal(parsePrice("5500"), 5500);
  assert.equal(parsePrice("$ 5.500"), 5500);
  assert.equal(parsePrice(" 12 000 "), 12000);
  assert.equal(parsePrice("0"), 0);
});

test("parsePrice treats an empty field as no price", () => {
  assert.equal(parsePrice(""), null);
  assert.equal(parsePrice("   "), null);
  assert.equal(parsePrice("$"), null);
});

test("parsePrice rejects what is not a price", () => {
  assert.equal(parsePrice("abc"), undefined);
  assert.equal(parsePrice("12,50"), undefined);
  assert.equal(parsePrice("-5"), undefined);
  assert.equal(parsePrice("5e3"), undefined);
});

test("parsePrice accepts the maximum and rejects one peso more", () => {
  assert.equal(parsePrice(String(MAX_PRICE)), MAX_PRICE);
  assert.equal(parsePrice(String(MAX_PRICE + 1)), undefined);
});

test("formatPrice groups thousands the Argentine way", () => {
  assert.match(formatPrice(5500), /^\$ 5\.500$/);
  assert.match(formatPrice(12000), /^\$ 12\.000$/);
});
