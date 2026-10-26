import assert from "node:assert/strict";
import { test } from "node:test";

import { distanceBetween } from "./geo.ts";

test("distanceBetween is zero for the same point", () => {
  assert.equal(distanceBetween({ lat: -32.17, lng: -64.11 }, { lat: -32.17, lng: -64.11 }), 0);
});

test("distanceBetween measures a kilometer north in about a kilometer", () => {
  const km = distanceBetween(
    { lat: -32.17, lng: -64.11 },
    { lat: -32.17 + 1 / 111.195, lng: -64.11 },
  );
  assert.ok(Math.abs(km - 1000) < 5, `got ${km}`);
});

test("distanceBetween is symmetric", () => {
  const a = { lat: -32.17, lng: -64.11 };
  const b = { lat: -31.4, lng: -64.18 };
  assert.equal(distanceBetween(a, b), distanceBetween(b, a));
});
