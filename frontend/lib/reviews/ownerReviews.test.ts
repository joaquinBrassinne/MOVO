import assert from "node:assert/strict";
import { test } from "node:test";

import { canReport, describeReportFailure, markReported } from "./ownerReviews.ts";
import type { ModerationReview } from "./types.ts";

const review = (id: number, status: ModerationReview["status"]): ModerationReview => ({
  id,
  branch_id: 1,
  branch_name: "Centro",
  business_id: 1,
  business_name: "Studio",
  rating: 4,
  display_name: "Ana G.",
  status,
  created_at: "2026-10-29T10:00:00Z",
});

test("a report marks only that review as reported", () => {
  const list = [review(1, "visible"), review(2, "visible")];
  const next = markReported(list, 2);
  assert.deepEqual(
    next.map((r) => r.status),
    ["visible", "reported"],
  );
  assert.equal(list[1].status, "visible", "the original list is not changed");
});

test("only a visible review can be reported", () => {
  assert.equal(canReport({ status: "visible" }), true);
  assert.equal(canReport({ status: "reported" }), false);
  assert.equal(canReport({ status: "hidden" }), false);
});

test("a failed report is worded in Spanish and never repeats the server text", () => {
  assert.equal(describeReportFailure({ code: "review_not_found" }).title, "La reseña no existe");
  assert.equal(
    describeReportFailure({ code: "business_deleted" }).title,
    "Tu negocio fue eliminado",
  );
  assert.equal(
    describeReportFailure({ code: "boom: stack trace" }).title,
    "No pudimos reportar la reseña",
  );
});
