// Pure helpers for subscription dates and tabs. Dates are Argentine calendar days
// (`YYYY-MM-DD`); they are never turned into local instants, so no timezone can shift them.

import type { PaymentState, SubscriptionRow, SubscriptionTab } from "./types.ts";

const MONTHS = ["ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"];

const ARGENTINA_OFFSET_MS = 3 * 60 * 60 * 1000;
const DAY_MS = 24 * 60 * 60 * 1000;
export const INITIAL_TERM_DAYS = 30;

/** "2026-11-30" -> "30 nov 2026". Empty or malformed input is returned as it came. */
export function formatDate(date: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!match) return date;
  const month = MONTHS[Number(match[2]) - 1];
  if (!month) return date;
  return `${Number(match[3])} ${month} ${match[1]}`;
}

/** The Argentine calendar day (UTC-3, no daylight saving) of an instant, as `YYYY-MM-DD`. */
export function argentineToday(now: Date = new Date()): string {
  return new Date(now.getTime() - ARGENTINA_OFFSET_MS).toISOString().slice(0, 10);
}

/** `date` plus `days` calendar days, as `YYYY-MM-DD`. */
export function addDays(date: string, days: number): string {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(Date.UTC(year, month - 1, day) + days * DAY_MS).toISOString().slice(0, 10);
}

/** The pre-filled initial validity: 30 days from the Argentine today. */
export function defaultPaidUntil(now: Date = new Date()): string {
  return addDays(argentineToday(now), INITIAL_TERM_DAYS);
}

/** "en 5 días", "mañana", "hoy", "venció ayer", "vencido hace 3 días". */
export function describeDaysLeft(daysLeft: number): string {
  if (daysLeft === 0) return "hoy";
  if (daysLeft === 1) return "mañana";
  if (daysLeft === -1) return "venció ayer";
  return daysLeft > 0 ? `en ${daysLeft} días` : `vencido hace ${-daysLeft} días`;
}

export function paymentState(row: Pick<SubscriptionRow, "status" | "expiring_soon">): PaymentState {
  if (row.status === "expired") return "expired";
  if (row.status === "grace") return "grace";
  return row.expiring_soon ? "expiring" : "ok";
}

export const STATE_LABEL: Record<PaymentState, string> = {
  ok: "Al día",
  expiring: "Por vencer",
  grace: "En gracia",
  expired: "Vencida",
};

export const TAB_LABEL: Record<SubscriptionTab, string> = {
  all: "Todas",
  expiring: "Por vencer (7 días)",
  grace: "En gracia",
  expired: "Vencidas",
  ok: "Al día",
};

export const TAB_ORDER: SubscriptionTab[] = ["all", "expiring", "grace", "expired", "ok"];

/** How many rows each tab holds. */
export function tabCounts(rows: SubscriptionRow[]): Record<SubscriptionTab, number> {
  const counts: Record<SubscriptionTab, number> = {
    all: rows.length,
    ok: 0,
    expiring: 0,
    grace: 0,
    expired: 0,
  };
  for (const row of rows) counts[paymentState(row)] += 1;
  return counts;
}

function normalize(text: string): string {
  return text
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .trim();
}

/** The rows of a tab whose business name contains the search text (case and accent blind). */
export function filterRows(
  rows: SubscriptionRow[],
  tab: SubscriptionTab,
  search: string,
): SubscriptionRow[] {
  const needle = normalize(search);
  return rows.filter(
    (row) =>
      (tab === "all" || paymentState(row) === tab) &&
      (needle === "" || normalize(row.business_name).includes(needle)),
  );
}

/** Up to two initials for the avatar: "Alma Haus Training" -> "AH". */
export function initials(name: string): string {
  const letters = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? "");
  return letters.join("") || "?";
}
