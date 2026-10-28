import assert from "node:assert/strict";
import { test } from "node:test";

import { formatCountdown, formatExpiresIn, formatShortDate, nextQuotaReset } from "./format.ts";
import { describeGenerateFailure, GENERATE_PROBLEM } from "./messages.ts";
import { isToken, parseSaved } from "./saved.ts";

const TOKEN_A = "a".repeat(43);
const TOKEN_B = "B-_".repeat(14) + "x";

test("a token is 43 characters of base64 for URLs and nothing else", () => {
  assert.equal(isToken(TOKEN_A), true);
  assert.equal(isToken(TOKEN_B), true);
  for (const bad of [
    "",
    "short",
    "a".repeat(42),
    "a".repeat(44),
    `${"a".repeat(42)}!`,
    "ABCDEFGH",
  ]) {
    assert.equal(isToken(bad), false, bad);
  }
});

test("saved coupons keep only real tokens, once each, newest first", () => {
  const raw = JSON.stringify([
    { token: TOKEN_A, savedAt: "2026-10-20T10:00:00Z" },
    { token: TOKEN_B, savedAt: "2026-10-27T10:00:00Z" },
    { token: TOKEN_A, savedAt: "2026-10-28T10:00:00Z" },
    { token: "not-a-token", savedAt: "2026-10-29T10:00:00Z" },
    { savedAt: "x" },
    null,
  ]);

  assert.deepEqual(parseSaved(raw), [
    { token: TOKEN_B, savedAt: "2026-10-27T10:00:00Z" },
    { token: TOKEN_A, savedAt: "2026-10-20T10:00:00Z" },
  ]);
});

test("whatever the browser holds that is not a list of tokens is an empty list", () => {
  assert.deepEqual(parseSaved(null), []);
  assert.deepEqual(parseSaved(""), []);
  assert.deepEqual(parseSaved("{not json"), []);
  assert.deepEqual(parseSaved('{"token":"x"}'), []);
});

test("the countdown reads hours and minutes and never goes below zero", () => {
  const now = new Date("2026-10-27T12:00:00Z");
  const at = (ms: number) => new Date(now.getTime() + ms).toISOString();
  const minute = 60_000;

  assert.equal(formatCountdown(at(48 * 60 * minute - 1), now), "47:59");
  assert.equal(formatCountdown(at(5 * minute), now), "00:05");
  assert.equal(formatCountdown(at(-1), now), "00:00");
  assert.equal(formatCountdown("nope", now), "00:00");
});

test("the short summary of what is left", () => {
  const now = new Date("2026-10-27T12:00:00Z");
  const at = (ms: number) => new Date(now.getTime() + ms).toISOString();
  const minute = 60_000;

  assert.equal(formatExpiresIn(at(31 * 60 * minute), now), "Vence en 31 hs");
  assert.equal(formatExpiresIn(at(25 * minute), now), "Vence en 25 min");
  assert.equal(formatExpiresIn(at(-1), now), "Vencido");
});

test("a moment is dated in Argentina, three hours behind UTC", () => {
  assert.equal(formatShortDate("2026-10-24T15:00:00Z"), "24 de oct");
  // 01:00 UTC del 25 todavía es el 24 a la noche en Argentina.
  assert.equal(formatShortDate("2026-10-25T01:00:00Z"), "24 de oct");
  assert.equal(formatShortDate("nope"), "");
});

test("the monthly quota starts again on the first of next month in Argentina", () => {
  assert.equal(nextQuotaReset(new Date("2026-10-27T12:00:00Z")), "2026-11-01");
  assert.equal(nextQuotaReset(new Date("2026-12-31T20:00:00Z")), "2027-01-01");
  // 02:00 UTC del 1 de noviembre todavía es 31 de octubre en Argentina: se reinicia hoy a la noche.
  assert.equal(nextQuotaReset(new Date("2026-11-01T02:00:00Z")), "2026-11-01");
});

test("each failure of generating a coupon has its own words", () => {
  const now = new Date("2026-10-27T12:00:00Z");

  assert.equal(
    describeGenerateFailure({ code: "pending_coupon" }, now).problem?.kind,
    GENERATE_PROBLEM.PENDING,
  );
  const quota = describeGenerateFailure({ code: "quota_reached" }, now).problem;
  assert.equal(quota?.title, "Cupos mensuales alcanzados");
  assert.match(quota?.body ?? "", /1 de noviembre/);
  assert.equal(
    describeGenerateFailure({ code: "benefit_not_found" }, now).problem?.kind,
    GENERATE_PROBLEM.UNAVAILABLE,
  );
  assert.match(
    describeGenerateFailure({ code: "network_error" }, now).problem?.title ?? "",
    /Sin conexión/,
  );
});

test("a rate limit says how long to wait, rounded up to minutes", () => {
  const now = new Date("2026-10-27T12:00:00Z");

  const some = describeGenerateFailure(
    { code: "rate_limited", details: { retry_after_seconds: 130 } },
    now,
  );
  assert.match(some.problem?.body ?? "", /3 minutos/);
  const one = describeGenerateFailure(
    { code: "rate_limited", details: { retry_after_seconds: 20 } },
    now,
  );
  assert.match(one.problem?.body ?? "", /1 minuto\./);
  assert.match(
    describeGenerateFailure({ code: "rate_limited" }, now).problem?.body ?? "",
    /en un rato/,
  );
});

test("a field error lands on its field and an unknown one never leaks the server text", () => {
  const now = new Date("2026-10-27T12:00:00Z");

  const name = describeGenerateFailure(
    { code: "validation_failed", details: { field: "name", reason: "invalid" } },
    now,
  );
  assert.equal(name.fields.name, "Ingresá tu nombre");
  assert.equal(name.problem, undefined);
  assert.match(
    describeGenerateFailure(
      { code: "validation_failed", details: { field: "phone", reason: "invalid" } },
      now,
    ).fields.phone ?? "",
    /característica/,
  );

  const unknown = describeGenerateFailure({ code: "something_new", details: { secret: "x" } }, now);
  assert.equal(unknown.problem?.kind, GENERATE_PROBLEM.OTHER);
  assert.doesNotMatch(JSON.stringify(unknown), /something_new|secret/);
});
