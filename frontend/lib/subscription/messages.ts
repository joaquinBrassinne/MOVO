// What the owner reads about their subscription. Pure functions, tested without React.

import { formatLongDate, joinNames, planLabel } from "./format.ts";
import { SUBSCRIPTION_STATUS, type PlanDowngradePayload, type SubscriptionView } from "./types.ts";

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
