// Pure helpers of the subscriptions list: the query string and the paging math.

import type { SubscriptionFilters, SubscriptionStatusFilter } from "./types.ts";

export const PAGE_SIZE = 20;

export const STATUS_FILTERS: { value: SubscriptionStatusFilter; label: string }[] = [
  { value: "", label: "Todas" },
  { value: "expiring", label: "Por vencer" },
  { value: "grace", label: "En gracia" },
  { value: "expired", label: "Vencidas" },
];

export const NO_FILTERS: SubscriptionFilters = { status: "", plan: "", query: "", page: 1 };

/** The `GET /admin/subscriptions` query string of a filter set; empty filters are left out. */
export function subscriptionsQuery(filters: SubscriptionFilters): string {
  const params = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String((filters.page - 1) * PAGE_SIZE),
  });
  if (filters.status) params.set("status", filters.status);
  if (filters.plan) params.set("plan", filters.plan);
  const query = filters.query.trim();
  if (query) params.set("q", query);
  return params.toString();
}

/** True when any filter narrows the list (the page does not count). */
export function hasActiveFilters(filters: SubscriptionFilters): boolean {
  return filters.status !== "" || filters.plan !== "" || filters.query.trim() !== "";
}

/** The 1-based range of a page, zeros when nothing matched: 45 entries, page 3 -> 41..45. */
export function pageRange(page: number, total: number): { from: number; to: number } {
  if (total <= 0) return { from: 0, to: 0 };
  return { from: (page - 1) * PAGE_SIZE + 1, to: Math.min(page * PAGE_SIZE, total) };
}
