import assert from "node:assert/strict";
import { test } from "node:test";

import {
  clampComment,
  COMMENT_MAX,
  describeSubmitFailure,
  formatDay,
  placeLine,
  ratingLabel,
} from "./reviewFormLogic.ts";

test("the rating label reads '4 de 5 · ¡Muy buena!' and is empty without a rating", () => {
  assert.equal(ratingLabel(4), "4 de 5 · ¡Muy buena!");
  assert.equal(ratingLabel(1), "1 de 5 · Mala");
  assert.equal(ratingLabel(0), "");
});

test("the comment is cut at 500 characters", () => {
  assert.equal(clampComment("a".repeat(600)).length, COMMENT_MAX);
  assert.equal(clampComment("hola"), "hola");
});

test("the redeemed day is short, in Argentina time", () => {
  assert.equal(formatDay("2026-10-24T15:00:00Z"), "24 oct");
  assert.equal(formatDay("2026-10-25T01:00:00Z"), "24 oct");
  assert.equal(formatDay(null), "");
});

test("the place skips what is missing", () => {
  assert.equal(placeLine("Palermo Soho", "CABA"), "Palermo Soho, CABA");
  assert.equal(placeLine(null, "CABA"), "CABA");
  assert.equal(placeLine(null, null), "");
});

test("submit errors switch the state or point at the field, never the server text", () => {
  assert.deepEqual(describeSubmitFailure({ code: "already_reviewed" }), {
    kind: "state",
    state: "reviewed",
  });
  assert.deepEqual(describeSubmitFailure({ code: "review_window_closed" }), {
    kind: "state",
    state: "window_closed",
  });
  assert.deepEqual(describeSubmitFailure({ code: "coupon_not_redeemed" }), {
    kind: "state",
    state: "not_redeemed",
  });
  assert.equal(
    describeSubmitFailure({ code: "validation_failed", details: { field: "comment" } }).kind,
    "comment",
  );
  assert.equal(
    describeSubmitFailure({ code: "validation_failed", details: { field: "rating" } }).kind,
    "rating",
  );
  assert.equal(describeSubmitFailure({ code: "network_error" }).kind, "generic");
});
