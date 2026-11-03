// Shapes of the owner metrics endpoints (backend/api/openapi.yaml, `/me/business/metrics/*`).

export const PERIODS = [7, 30, 90] as const;
export type Period = (typeof PERIODS)[number];
export const DEFAULT_PERIOD: Period = 30;

export interface PeriodRange {
  days: number;
  from: string;
  to: string;
}

/** A fraction (0.25 is 25 %) with its movement against the previous period. */
export interface Rate {
  numerator: number;
  denominator: number;
  rate: number | null;
  previous_rate: number | null;
  change: number | null;
}

export interface Count {
  value: number;
  previous: number;
  change: number | null;
}

/** Every Pro metric with a period answers this envelope; insufficient_data carries nothing else. */
interface Envelope {
  status: "ok" | "insufficient_data";
  period: PeriodRange;
  previous_period: PeriodRange;
  coupons_generated: number;
  minimum_sample: number;
}

export interface Essentials {
  period: PeriodRange;
  profile_views: number;
  coupons: { generated: number; redeemed: number };
}

export interface Activity {
  service_id: number;
  name: string;
  views: number;
  whatsapp_clicks: number;
}

export interface Activities {
  period: PeriodRange;
  activities: Activity[];
}

export interface Conversion extends Envelope {
  views_to_coupons?: Rate;
  coupons_to_redemptions?: Rate;
}

export type SlotKey = "overnight" | "morning" | "afternoon" | "evening";

export interface HeatCell {
  weekday: number;
  slot: number;
  redemptions: number;
}

export interface Heatmap extends Envelope {
  redemptions?: number;
  previous_redemptions?: number;
  cells?: HeatCell[];
}

export interface BenefitRow {
  benefit_id: number;
  name: string;
  generated: number;
  redeemed: number;
  expired: number;
  pending: number;
  rate: number | null;
  previous_rate: number | null;
  change: number | null;
}

export interface BenefitPerformance extends Envelope {
  benefits?: BenefitRow[];
}

export interface TimeToRedeem extends Envelope {
  redemptions?: number;
  same_day?: Rate;
  within_24h?: Rate;
  within_48h?: Rate;
}

export interface ReturningCustomers extends Envelope {
  customers?: number;
  previous_customers?: number;
  one_redemption?: number;
  two_or_more?: number;
  repeat_rate?: Rate;
}

export type QuotaStatus =
  "no_quota" | "insufficient_data" | "exhausted" | "on_track" | "will_run_out";

export interface QuotaProjection {
  status: QuotaStatus;
  limit?: number;
  generated?: number;
  days_elapsed?: number;
  days_in_month?: number;
  minimum_days?: number;
  projected_total?: number;
  exhausts_on?: string;
  resets_on?: string;
}

export interface EstimatedReturn extends Omit<Envelope, "status"> {
  status: "ok" | "no_ticket" | "insufficient_data";
  estimated?: boolean;
  redemptions?: number;
  average_ticket_ars?: number;
  estimated_return_ars?: number;
  previous_estimated_return_ars?: number | null;
  plan_price_ars?: number;
}

export interface BranchRow {
  branch_id: number;
  name: string;
  active: boolean;
  service_views: Count;
  whatsapp_clicks: Count;
  redemptions: Count;
}

export interface BranchComparison extends Envelope {
  branches?: BranchRow[];
}

export interface ReviewsOverTime extends Envelope {
  rating?: {
    reviews: number;
    previous_reviews: number;
    average: number | null;
    previous_average: number | null;
    change: number | null;
  };
  review_rate?: Rate;
}

export interface Records {
  status: "ok" | "insufficient_data";
  history_months: number;
  minimum_history_months: number;
  minimum_best_month_redemptions: number;
  best_month_redemptions?: number;
  best_month?: { month: string; redemptions: number };
  current_month?: {
    month: string;
    redemptions: number;
    progress: number;
    record_in_progress: boolean;
  };
  strong_weekday?: number;
  strong_slot?: { slot: number; key: SlotKey };
  star_benefit?: { benefit_id: number; name: string; redemptions: number } | null;
}

/** Metric name to its path under `/me/business/metrics` and its answer. */
export interface MetricMap {
  essentials: Essentials;
  activities: Activities;
  conversion: Conversion;
  heatmap: Heatmap;
  benefits: BenefitPerformance;
  "time-to-redeem": TimeToRedeem;
  "returning-customers": ReturningCustomers;
  "quota-projection": QuotaProjection;
  "estimated-return": EstimatedReturn;
  branches: BranchComparison;
  reviews: ReviewsOverTime;
  records: Records;
}

export type MetricName = keyof MetricMap;

/** Metrics that take no `period`: the month in course, or the last 12 complete months. */
export const PERIODLESS: readonly MetricName[] = ["quota-projection", "records"];
