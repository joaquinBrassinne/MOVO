// Cómo se lee el cupo mensual del local. Funciones puras: se prueban sin React.

import type { CouponQuota } from "./types.ts";

export const QUOTA_LEVEL = {
  UNLIMITED: "unlimited",
  OK: "ok",
  LOW: "low",
  EXHAUSTED: "exhausted",
} as const;

export type QuotaLevel = (typeof QUOTA_LEVEL)[keyof typeof QUOTA_LEVEL];

/** Desde este porcentaje consumido el cupo se avisa como "quedan pocos" (SDD 4.10). */
export const LOW_QUOTA_PERCENT = 90;

export interface QuotaSummary {
  level: QuotaLevel;
  used: number;
  /** `null` sin límite. */
  limit: number | null;
  /** Porcentaje consumido, de 0 a 100; 0 sin límite. */
  percent: number;
  /** Cupones que quedan; `null` sin límite. */
  remaining: number | null;
}

export function summarizeQuota(quota: CouponQuota): QuotaSummary {
  const used = quota.generated_this_month;
  if (quota.limit === null) {
    return { level: QUOTA_LEVEL.UNLIMITED, used, limit: null, percent: 0, remaining: null };
  }

  const percent = Math.min(100, Math.floor((used * 100) / quota.limit));
  const remaining = Math.max(0, quota.limit - used);
  const level =
    remaining === 0
      ? QUOTA_LEVEL.EXHAUSTED
      : percent >= LOW_QUOTA_PERCENT
        ? QUOTA_LEVEL.LOW
        : QUOTA_LEVEL.OK;

  return { level, used, limit: quota.limit, percent, remaining };
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

/** "2026-11-01" se muestra como "1 de noviembre". Sin una fecha válida devuelve "". */
export function formatResetDate(date: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!match) return "";

  const month = MONTHS[Number(match[2]) - 1];
  return month ? `${Number(match[3])} de ${month}` : "";
}
