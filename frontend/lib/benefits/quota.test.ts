import assert from "node:assert/strict";
import { test } from "node:test";

import { formatResetDate, summarizeQuota } from "./quota.ts";

const quota = (limit: number | null, generated: number) => ({
  limit,
  generated_this_month: generated,
  resets_on: "2026-11-01",
});

test("a quota without a limit has nothing to run out of", () => {
  assert.deepEqual(summarizeQuota(quota(null, 500)), {
    level: "unlimited",
    used: 500,
    limit: null,
    percent: 0,
    remaining: null,
  });
});

test("the quota is fine, low from 90 % and exhausted when none is left", () => {
  assert.deepEqual(summarizeQuota(quota(30, 18)), {
    level: "ok",
    used: 18,
    limit: 30,
    percent: 60,
    remaining: 12,
  });
  assert.equal(summarizeQuota(quota(30, 26)).level, "ok");
  assert.equal(summarizeQuota(quota(30, 27)).level, "low");
  assert.equal(summarizeQuota(quota(30, 29)).level, "low");
  assert.equal(summarizeQuota(quota(30, 30)).level, "exhausted");
  assert.equal(summarizeQuota(quota(1, 0)).level, "ok");
});

test("a quota exceeded by old data never shows more than 100 % or a negative remainder", () => {
  const over = summarizeQuota(quota(10, 14));
  assert.equal(over.percent, 100);
  assert.equal(over.remaining, 0);
  assert.equal(over.level, "exhausted");
});

test("the reset date reads like a person says it", () => {
  assert.equal(formatResetDate("2026-11-01"), "1 de noviembre");
  assert.equal(formatResetDate("2027-01-01"), "1 de enero");
  assert.equal(formatResetDate("2026-12-15"), "15 de diciembre");
  assert.equal(formatResetDate("nope"), "");
  assert.equal(formatResetDate("2026-13-01"), "");
});
