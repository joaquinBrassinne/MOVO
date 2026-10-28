import assert from "node:assert/strict";
import { test } from "node:test";

import { displayCode, normalizeCode } from "./code.ts";
import { describeReturning, PROBLEM, problemOf, problemOfStatus } from "./redeem.ts";

test("a code is read like the backend reads it", () => {
  assert.equal(normalizeCode("abcd-efgh"), "ABCDEFGH");
  assert.equal(normalizeCode(" ab cd 23 45 "), "ABCD2345");
  assert.equal(normalizeCode("ABCDEFGH"), "ABCDEFGH");
});

test("what cannot be a code is not looked up", () => {
  assert.equal(normalizeCode(""), null);
  assert.equal(normalizeCode("ABCDEFG"), null);
  assert.equal(normalizeCode("ABCDEFGHJ"), null);
  // 0, O, 1 e I no están en el alfabeto: la gente los confunde.
  for (const confusing of ["ABCD0FGH", "ABCDOFGH", "ABCD1FGH", "ABCDIFGH"]) {
    assert.equal(normalizeCode(confusing), null, confusing);
  }
  assert.equal(normalizeCode("ABCD#FGH"), null);
});

test("the field shows capitals with a hyphen in the middle and never more than 8 symbols", () => {
  assert.equal(displayCode("ab"), "AB");
  assert.equal(displayCode("abcd"), "ABCD");
  assert.equal(displayCode("abcde"), "ABCD-E");
  assert.equal(displayCode("ab cd-ef gh"), "ABCD-EFGH");
  assert.equal(displayCode("abcdefghij"), "ABCD-EFGH");
  assert.equal(displayCode("##"), "");
});

test("each failure of the API has its own words", () => {
  assert.equal(problemOf({ code: "coupon_not_found" }).kind, PROBLEM.NOT_FOUND);
  assert.equal(problemOf({ code: "coupon_expired" }).title, "Cupón vencido");
  assert.equal(problemOf({ code: "coupon_redeemed" }).title, "Ya fue canjeado");
  assert.equal(
    problemOf({ code: "validation_failed", details: { field: "branch_id", reason: "required" } })
      .kind,
    PROBLEM.BRANCH,
  );
  assert.equal(problemOf({ code: "network_error" }).title, "Sin conexión");
  assert.equal(problemOf({ code: "unauthorized" }).title, "Tu sesión venció");
});

test("an unknown failure never leaks the server text", () => {
  const problem = problemOf({ code: "something_new", details: { secret: "x" } });

  assert.equal(problem.kind, PROBLEM.OTHER);
  assert.doesNotMatch(`${problem.title} ${problem.body}`, /something_new|secret/);
});

test("a coupon that was seen but cannot be redeemed is explained by its status", () => {
  assert.equal(problemOfStatus("expired")?.kind, PROBLEM.EXPIRED);
  assert.equal(problemOfStatus("redeemed")?.kind, PROBLEM.REDEEMED);
  assert.equal(problemOfStatus("pending"), null);
});

test("how many coupons the customer already redeemed is said in plain words", () => {
  assert.match(describeReturning(0), /primera vez/);
  assert.equal(describeReturning(1), "Este cliente ya canjeó 1 cupón en tu negocio.");
  assert.equal(describeReturning(4), "Este cliente ya canjeó 4 cupones en tu negocio.");
});
