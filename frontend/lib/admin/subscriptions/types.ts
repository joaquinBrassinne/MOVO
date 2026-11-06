// Contract of the /admin/subscriptions endpoints of the Go API.

import type { BusinessStatus } from "../../business/types.ts";

export type PlanId = "starter" | "pro";
export type SubscriptionStatus = "active" | "grace" | "expired";

export interface Subscription {
  business_id: number;
  plan: PlanId;
  /** Calendar date in Argentina, `YYYY-MM-DD`. */
  paid_until: string;
  grace_until: string;
  status: SubscriptionStatus;
  mp_subscription_ref?: string;
  notes?: string;
}

export interface SubscriptionRow extends Subscription {
  business_name: string;
  business_status: BusinessStatus;
  pilot: boolean;
  expiring_soon: boolean;
  /** Negative once the paid period is over. */
  days_left: number;
  /** A wa.me link with the reminder text; empty when the business has no usable contact. */
  reminder_url: string;
  /** The plan (a pilot gets Pro's) includes priority support: only a mark for the admin. */
  priority_support: boolean;
}

export interface SubscriptionList {
  subscriptions: SubscriptionRow[];
  /** Entries that match the filters, before paging. */
  total: number;
}

/** `""` is every status; the values are the `status` query of the API. */
export type SubscriptionStatusFilter = "" | "expiring" | "grace" | "expired";

export interface SubscriptionFilters {
  status: SubscriptionStatusFilter;
  plan: "" | PlanId;
  query: string;
  /** 1-based. */
  page: number;
}

export interface NamedItem {
  id: number;
  name: string;
}

export interface PlanChange extends Subscription {
  deactivated: { branches: NamedItem[]; services: NamedItem[] };
}

export interface ConfirmPaymentInput {
  plan: PlanId;
  paid_until: string;
  mp_subscription_ref: string;
  notes: string;
}

/** What the admin sees for a subscription; it merges the paid status with the expiry warning. */
export type PaymentState = "ok" | "expiring" | "grace" | "expired";
