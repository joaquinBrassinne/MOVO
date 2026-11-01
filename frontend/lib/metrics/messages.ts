// Pure copy and formatting helpers for the metrics page: no React, no Intl, no timezone.

import { formatLongDate } from "../subscription/format.ts";
import type { HeatCell, QuotaProjection, Records, SlotKey } from "./types.ts";

const DASH = "—";

function round(value: number, digits: number): number {
  const factor = 10 ** digits;
  const rounded = Math.round(value * factor) / factor;
  return rounded === 0 ? 0 : rounded; // never "-0"
}

const decimal = (value: number, digits: number) => String(round(value, digits)).replace(".", ",");

/** 0.084 -> "8,4 %"; 0.25 -> "25 %"; null -> "—". */
export function formatPercent(fraction: number | null | undefined, digits = 1): string {
  return fraction == null ? DASH : `${decimal(fraction * 100, digits)} %`;
}

/** Growth as a fraction: 0.12 -> "+12 %", -0.03 -> "-3 %", null -> null (no baseline). */
export function formatChange(change: number | null | undefined): string | null {
  if (change == null) return null;
  const text = formatPercent(Math.abs(change), 0);
  if (round(change * 100, 0) === 0) return text;
  return `${change > 0 ? "+" : "-"}${text}`;
}

/** The movement of a rate, in percentage points: 0.012 -> "+1,2 pp". */
export function formatPoints(change: number | null | undefined): string | null {
  if (change == null) return null;
  const points = round(change * 100, 1);
  return `${points > 0 ? "+" : ""}${decimal(points, 1)} pp`;
}

export type Direction = "up" | "down" | "flat";

export function direction(change: number | null | undefined): Direction | null {
  if (change == null) return null;
  if (change > 0) return "up";
  if (change < 0) return "down";
  return "flat";
}

/** 184000 -> "$184.000". */
export function formatArs(amount: number): string {
  return `$${Math.round(amount)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".")}`;
}

/** 4.56 -> "4,6". */
export function formatStars(value: number | null | undefined): string {
  return value == null ? DASH : decimal(value, 1);
}

/** "+0,2" / "-0,1" stars. */
export function formatStarsChange(change: number | null | undefined): string | null {
  if (change == null) return null;
  const value = round(change, 1);
  return `${value > 0 ? "+" : ""}${decimal(value, 1)}`;
}

const WEEKDAYS = ["Lun", "Mar", "Mié", "Jue", "Vie", "Sáb", "Dom"];
const WEEKDAYS_LONG = ["lunes", "martes", "miércoles", "jueves", "viernes", "sábado", "domingo"];

/** ISO weekday 1 (Monday) to 7 (Sunday). */
export const weekdayShort = (weekday: number) => WEEKDAYS[weekday - 1] ?? DASH;
export const weekdayLong = (weekday: number) => WEEKDAYS_LONG[weekday - 1] ?? DASH;

export const SLOT_KEYS: readonly SlotKey[] = ["overnight", "morning", "afternoon", "evening"];

const SLOT_NAMES: Record<SlotKey, string> = {
  overnight: "Madrugada",
  morning: "Mañana",
  afternoon: "Tarde",
  evening: "Noche",
};

// The four six-hour slots of the heat map (backend: `slots` of /metrics/heatmap).
const SLOT_RANGES: Record<SlotKey, string> = {
  overnight: "0 a 6 h",
  morning: "6 a 12 h",
  afternoon: "12 a 18 h",
  evening: "18 a 24 h",
};

export const slotName = (key: SlotKey) => SLOT_NAMES[key];
export const slotRange = (key: SlotKey) => SLOT_RANGES[key];

/** The busiest cell, or null when nothing was redeemed. A tie keeps the first (API order). */
export function peakCell(cells: readonly HeatCell[]): HeatCell | null {
  let peak: HeatCell | null = null;
  for (const cell of cells) {
    if (cell.redemptions > 0 && (peak === null || cell.redemptions > peak.redemptions)) peak = cell;
  }
  return peak;
}

/** "Pico: sábado, noche (18 a 24 h)". */
export function peakText(cell: HeatCell | null): string {
  if (!cell) return "Todavía no hay un pico";
  const key = SLOT_KEYS[cell.slot];
  if (!key) return "Todavía no hay un pico";
  return `Pico: ${weekdayLong(cell.weekday)}, ${slotName(key).toLowerCase()} (${slotRange(key)})`;
}

/** Shade 0 (none) to 4 (the peak), relative to the busiest cell. */
export function heatLevel(value: number, max: number): 0 | 1 | 2 | 3 | 4 {
  if (value <= 0 || max <= 0) return 0;
  return Math.min(4, Math.max(1, Math.ceil((value / max) * 4))) as 1 | 2 | 3 | 4;
}

const MONTHS = [
  "enero",
  "febrero",
  "marzo",
  "abril",
  "mayo",
  "junio",
  "julio",
  "agosto",
  "septiembre",
  "octubre",
  "noviembre",
  "diciembre",
];

/** "2026-10" -> "octubre de 2026". */
export function monthLabel(month: string): string {
  const match = /^(\d{4})-(\d{2})/.exec(month);
  const name = match ? MONTHS[Number(match[2]) - 1] : undefined;
  return match && name ? `${name} de ${match[1]}` : month;
}

/** The progress of the month in course against the record: "Vas al 65 %" or "Récord en curso". */
export function recordProgressText(current: NonNullable<Records["current_month"]>): string {
  if (current.record_in_progress) return "Récord en curso";
  return `Vas al ${formatPercent(current.progress, 0)}`;
}

/** Copy for the quota projection; `tone` drives the colour of the note. */
export function quotaCopy(projection: QuotaProjection): { text: string; tone: "calm" | "warning" } {
  switch (projection.status) {
    case "no_quota":
      return { text: "Tu plan no tiene un tope de cupones este mes.", tone: "calm" };
    case "insufficient_data":
      return {
        text: `Todavía es pronto para estimarlo: hace falta que pasen ${projection.minimum_days ?? 3} días del mes y que se genere algún cupón.`,
        tone: "calm",
      };
    case "exhausted":
      return {
        text: `Ya usaste todo el cupo del mes${projection.resets_on ? `: se renueva el ${formatLongDate(projection.resets_on)}` : ""}.`,
        tone: "warning",
      };
    case "will_run_out":
      return {
        text: `A este ritmo, tu cupo se agota el ${projection.exhausts_on ? formatLongDate(projection.exhausts_on) : "mes que viene"}.`,
        tone: "warning",
      };
    case "on_track":
      return {
        text: `A este ritmo, tu cupo te alcanza hasta fin de mes${projection.resets_on ? ` (se renueva el ${formatLongDate(projection.resets_on)})` : ""}.`,
        tone: "calm",
      };
  }
}

/** How many times the plan price the estimate covers, only for a 30-day period (a monthly price). */
export function returnMultiple(estimated: number, planPrice: number, days: number): number | null {
  if (days !== 30 || planPrice <= 0) return null;
  return estimated / planPrice;
}

export const formatMultiple = (value: number) => `${decimal(value, 1)}x`;

/** "17 canjes × $6.500 de ticket promedio". */
export function returnFormula(redemptions: number, ticket: number): string {
  return `${redemptions} ${redemptions === 1 ? "canje" : "canjes"} × ${formatArs(ticket)} de ticket promedio`;
}

/** The text of an insufficient_data card. */
export function insufficientText(minimumSample: number | undefined): string {
  return `Sin datos suficientes todavía (mínimo ${minimumSample ?? 20} cupones)`;
}

/** Records need history, not a sample of coupons: they say how much history is missing. */
export function recordsInsufficientText(records: Records): string {
  return `Sin datos suficientes todavía: hacen falta ${records.minimum_history_months} meses completos de historia y al menos ${records.minimum_best_month_redemptions} canjes en tu mejor mes.`;
}

/** The message of a card whose request failed. */
export function failureText(code: string): string {
  if (code === "plan_upgrade_required") return "Esta métrica es parte del Plan Pro.";
  return "No pudimos cargar esta métrica.";
}

export const periodLabel = (days: number) => `${days} días`;
