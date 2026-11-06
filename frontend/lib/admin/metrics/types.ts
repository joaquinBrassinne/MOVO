// Contract of GET /admin/metrics of the Go API.

export interface CouponCounts {
  generated: number;
  redeemed: number;
}

export interface AdminMetrics {
  active_businesses: number;
  /** All-time totals. */
  coupons: CouponCounts;
  /** Since `month_start`. */
  coupons_this_month: CouponCounts;
  /** First day of the current month (Argentina), `YYYY-MM-DD`. */
  month_start: string;
}

export type MetricsPeriod = "month" | "all";
