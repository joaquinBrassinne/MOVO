import assert from "node:assert/strict";
import { test } from "node:test";

import {
  allowedDecisions,
  describeModerationFailure,
  formatReviewDate,
  truncateComment,
  withoutReview,
} from "./moderation.ts";

test("a reported review can be kept or hidden", () => {
  assert.deepEqual(allowedDecisions("reported"), { visible: true, hidden: true });
});

test("a hidden review can only be made visible again", () => {
  assert.deepEqual(allowedDecisions("hidden"), { visible: true, hidden: false });
});

test("a visible review can only be hidden", () => {
  assert.deepEqual(allowedDecisions("visible"), { visible: false, hidden: true });
});

test("withoutReview removes only the decided review", () => {
  assert.deepEqual(withoutReview([{ id: 1 }, { id: 2 }, { id: 3 }], 2), [{ id: 1 }, { id: 3 }]);
});

test("truncateComment keeps short text and cuts long text", () => {
  assert.equal(truncateComment("Muy bueno", 20), "Muy bueno");
  assert.equal(truncateComment("abcdefghij", 5), "abcde…");
  assert.equal(truncateComment(undefined), "");
});

test("formatReviewDate writes day, month and year", () => {
  assert.equal(formatReviewDate("2026-10-23T15:00:00Z"), "23 oct 2026");
  assert.equal(formatReviewDate("nope"), "");
});

test("a purged review explains why it cannot be restored", () => {
  assert.match(describeModerationFailure({ code: "review_purged" }).title, /no se puede restaurar/);
});

test("an unknown failure gets the generic message", () => {
  assert.equal(describeModerationFailure({ code: "boom" }).title, "No pudimos completar la acción");
});
