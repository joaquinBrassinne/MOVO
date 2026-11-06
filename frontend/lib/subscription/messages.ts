// What the owner reads about their subscription. Pure functions, tested without React.

import { formatLongDate, joinNames, planLabel } from "./format.ts";
import {
  ALERT_TYPE,
  SUBSCRIPTION_STATUS,
  type PlanAlert,
  type PlanDowngradePayload,
  type SubscriptionView,
  type UnredeemedCouponsPayload,
} from "./types.ts";

export type StatusTone = "calm" | "warning" | "danger";

export interface StatusCopy {
  tone: StatusTone;
  /** Short chip text, shown on the plan card. */
  chip: string;
  /** Longer explanation, shown under the card when the owner needs to know. */
  title?: string;
  body?: string;
}

export function statusCopy({ pilot, subscription }: SubscriptionView): StatusCopy {
  if (pilot) {
    return {
      tone: "calm",
      chip: "Local piloto",
      title: "Local piloto",
      body: "Tenés acceso piloto sin costo, con los límites del plan Pro.",
    };
  }
  if (!subscription) {
    return {
      tone: "calm",
      chip: "En revisión",
      body: "Tu suscripción se confirma cuando el equipo de MOVO revise tu alta.",
    };
  }

  const paid = formatLongDate(subscription.paid_until);
  const grace = formatLongDate(subscription.grace_until);
  switch (subscription.status) {
    case SUBSCRIPTION_STATUS.GRACE:
      return {
        tone: "warning",
        chip: `Venció el ${paid}`,
        title: "Tu suscripción venció",
        body: `Tu suscripción venció el ${paid}. Seguís visible hasta el ${grace}; si el cobro de Mercado Pago no se regulariza, tu local deja de aparecer en el mapa.`,
      };
    case SUBSCRIPTION_STATUS.EXPIRED:
      return {
        tone: "danger",
        chip: `Venció el ${paid}`,
        title: "Tu local dejó de aparecer en el mapa",
        body: `Tu suscripción venció el ${paid} y el plazo de gracia terminó el ${grace}. Si el cobro de Mercado Pago no se regulariza, tu local sigue sin aparecer en el mapa.`,
      };
    default:
      return { tone: "calm", chip: `Vigente hasta el ${paid}` };
  }
}

/** The notice for a plan change that switched things off. */
export function downgradeMessage(payload: PlanDowngradePayload): string {
  const items = [...payload.branches, ...payload.services].map((item) => item.name);
  const from = planLabel(payload.plan_from);
  const to = planLabel(payload.plan_to);
  return `Tu plan cambió de ${from} a ${to}. Se desactivaron ${joinNames(items)} porque superan los límites del nuevo plan. No se borró nada: podés reactivarlos mejorando de plan o desactivando otros.`;
}

/** The notice for coupons that were generated but rarely redeemed. Counts and a rate only. */
export function unredeemedMessage(payload: UnredeemedCouponsPayload): string {
  const percent = Math.round(payload.redeem_rate * 100);
  return `Solo ${payload.coupons_redeemed} de cada ${payload.coupons_generated} cupones de los últimos ${payload.period_days} días se canjearon (${percent} %). Revisá tus beneficios y si el personal sabe cómo canjearlos.`;
}

export interface AlertView {
  title: string;
  body: string;
  /** Where the owner can act on it, when there is such a place. */
  href?: string;
  linkLabel?: string;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const isCount = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value) && value >= 0;

/** What to show for an alert, or `null` when its type is unknown or its payload malformed. */
export function alertView(alert: PlanAlert): AlertView | null {
  const payload = alert.payload;
  if (!isRecord(payload)) return null;

  switch (alert.type) {
    case ALERT_TYPE.PLAN_DOWNGRADE:
      if (!Array.isArray(payload.branches) || !Array.isArray(payload.services)) return null;
      return {
        title: "Cambió tu plan",
        body: downgradeMessage(payload as unknown as PlanDowngradePayload),
      };
    case ALERT_TYPE.UNREDEEMED_COUPONS:
      if (
        !isCount(payload.period_days) ||
        !isCount(payload.coupons_generated) ||
        !isCount(payload.coupons_redeemed) ||
        !isCount(payload.redeem_rate)
      ) {
        return null;
      }
      return {
        title: "Cupones sin canjear",
        body: unredeemedMessage(payload as unknown as UnredeemedCouponsPayload),
        href: "/owners/metrics",
        linkLabel: "Ver métricas",
      };
    default:
      return null;
  }
}
