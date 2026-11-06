// Pure helpers of the admin metrics page.

import type { AdminMetrics, CouponCounts, MetricsPeriod } from "./types.ts";

/** The coupon counts of the chosen period. */
export function couponsOf(metrics: AdminMetrics, period: MetricsPeriod): CouponCounts {
  return period === "month" ? metrics.coupons_this_month : metrics.coupons;
}

/** Redeemed over generated as a percentage; `null` when nothing was generated (no 0/0). */
export function redemptionRate({ generated, redeemed }: CouponCounts): number | null {
  if (generated <= 0) return null;
  return (redeemed / generated) * 100;
}

/** 47.04 -> "47,0%"; `null` -> "—". Argentine decimal comma, one decimal. */
export function formatPercent(value: number | null): string {
  if (value === null) return "—";
  return `${value.toFixed(1).replace(".", ",")}%`;
}

/** 3420 -> "3.420" (Argentine thousands dot, independent of the runtime locale). */
export function formatCount(value: number): string {
  return String(Math.trunc(value)).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}
