import assert from "node:assert/strict";
import { test } from "node:test";

import { couponsOf, formatCount, formatPercent, redemptionRate } from "./format.ts";
import type { AdminMetrics } from "./types.ts";

const metrics: AdminMetrics = {
  active_businesses: 48,
  coupons: { generated: 14850, redeemed: 6980 },
  coupons_this_month: { generated: 3420, redeemed: 1608 },
  month_start: "2026-11-01",
};

test("the period picks the monthly or the cumulative counts", () => {
  assert.equal(couponsOf(metrics, "month").generated, 3420);
  assert.equal(couponsOf(metrics, "all").generated, 14850);
});

test("the redemption rate is redeemed over generated, and empty when nothing was generated", () => {
  assert.equal(formatPercent(redemptionRate({ generated: 3420, redeemed: 1608 })), "47,0%");
  assert.equal(formatPercent(redemptionRate({ generated: 4, redeemed: 4 })), "100,0%");
  assert.equal(formatPercent(redemptionRate({ generated: 0, redeemed: 0 })), "—");
});

test("counts use the thousands dot", () => {
  assert.equal(formatCount(0), "0");
  assert.equal(formatCount(999), "999");
  assert.equal(formatCount(3420), "3.420");
  assert.equal(formatCount(1234567), "1.234.567");
});
