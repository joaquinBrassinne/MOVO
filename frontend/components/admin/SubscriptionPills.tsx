import { STATE_LABEL } from "@/lib/admin/subscriptions/format";
import type { PaymentState, PlanId } from "@/lib/admin/subscriptions/types";

export const PLAN_NAME: Record<PlanId, string> = { starter: "Starter", pro: "Pro" };

const TONE: Record<PaymentState, string> = {
  ok: "bg-movo-ok/15 text-movo-ok",
  expiring: "bg-movo-accent/40 text-movo-ink",
  grace: "bg-orange-100 text-orange-900",
  expired: "bg-movo-slate/20 text-movo-ink",
};

/** The payment state as a labelled pill: the text carries the meaning, the color only helps. */
export function PaymentStatePill({ state }: { state: PaymentState }) {
  return (
    <span className={`inline-flex rounded-full px-3 py-1 text-xs font-bold ${TONE[state]}`}>
      {STATE_LABEL[state]}
    </span>
  );
}

export function PlanChip({ plan }: { plan: PlanId }) {
  return (
    <span
      className={`inline-flex rounded-full px-3 py-1 text-xs font-bold ${
        plan === "pro" ? "bg-movo-card text-movo-accent" : "bg-movo-sand text-movo-ink"
      }`}
    >
      {PLAN_NAME[plan]}
    </span>
  );
}
