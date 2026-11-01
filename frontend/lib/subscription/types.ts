import type { PlanName } from "../business/types.ts";

export const SUBSCRIPTION_STATUS = {
  ACTIVE: "active",
  GRACE: "grace",
  EXPIRED: "expired",
} as const;

export type SubscriptionStatus = (typeof SUBSCRIPTION_STATUS)[keyof typeof SUBSCRIPTION_STATUS];

/** Dates are calendar dates (YYYY-MM-DD) in Argentina. */
export interface Subscription {
  plan: PlanName;
  paid_until: string;
  grace_until: string;
  status: SubscriptionStatus;
}

export interface SubscriptionView {
  pilot: boolean;
  subscription: Subscription | null;
}

export interface NamedItem {
  id: number;
  name: string;
}

export interface PlanDowngradePayload {
  plan_from: PlanName;
  plan_to: PlanName;
  branches: NamedItem[];
  services: NamedItem[];
}

/** Counts and rates only; `redeem_rate` and `threshold` are fractions (0.1 is 10 %). */
export interface UnredeemedCouponsPayload {
  period_days: number;
  coupons_generated: number;
  coupons_redeemed: number;
  redeem_rate: number;
  threshold: number;
}

/** The payload depends on `type`; it is validated by `alertView`, never trusted. */
export interface PlanAlert {
  id: number;
  type: string;
  payload: unknown;
  created_at: string;
}

export const ALERT_TYPE = {
  PLAN_DOWNGRADE: "plan_downgrade",
  UNREDEEMED_COUPONS: "unredeemed_coupons",
} as const;
