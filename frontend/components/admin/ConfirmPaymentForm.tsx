"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { describeAdminFailure } from "@/lib/admin/messages";
import { confirmPayment } from "@/lib/admin/subscriptions/api";
import { argentineToday, defaultPaidUntil } from "@/lib/admin/subscriptions/format";
import { usePlanCatalog } from "@/lib/admin/subscriptions/hooks";
import type { PlanId } from "@/lib/admin/subscriptions/types";
import type { FormMessage } from "@/lib/business/messages";
import { formatPrice } from "@/lib/plans/format";

import { PLAN_NAME } from "./SubscriptionPills";

const CONTROL =
  "h-12 w-full rounded-2xl bg-white px-4 text-base outline-none ring-1 ring-movo-slate/30 focus:ring-2 focus:ring-movo-accent disabled:opacity-60";

interface Props {
  businessId: number;
  /** Primary button text: "Confirmar pago y publicar" or "Confirmar pago". */
  submitLabel: string;
  /** Called after the payment was confirmed, so the caller reloads. */
  onConfirmed: () => void;
}

/**
 * Confirms the first payment: the admin picks the plan and the initial validity. The API creates
 * the subscription and, for a business in review, publishes it in the same transaction.
 */
export function ConfirmPaymentForm({ businessId, submitLabel, onConfirmed }: Props) {
  const catalog = usePlanCatalog();
  const [plan, setPlan] = useState<PlanId>("starter");
  const [paidUntil, setPaidUntil] = useState(() => defaultPaidUntil());
  const [reference, setReference] = useState("");
  const [notes, setNotes] = useState("");
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<FormMessage | null>(null);

  const prices = new Map(
    catalog.state.status === "ready" ? catalog.state.data.map((p) => [p.plan, p.price]) : [],
  );
  const id = (name: string) => `confirm-${businessId}-${name}`;

  async function submit() {
    if (pending) return;
    setPending(true);
    setMessage(null);

    const result = await confirmPayment(businessId, {
      plan,
      paid_until: paidUntil,
      mp_subscription_ref: reference.trim(),
      notes: notes.trim(),
    });

    setPending(false);
    if (result.ok) {
      onConfirmed();
      return;
    }
    setMessage(describeAdminFailure(result));
    // A state change (already confirmed, no longer confirmable) needs the fresh data.
    if (result.status === 409) onConfirmed();
  }

  return (
    <div className="space-y-3 border-t border-movo-slate/20 pt-3">
      {message ? <Notice title={message.title} body={message.body} /> : null}

      <label htmlFor={id("plan")} className="block text-sm font-semibold">
        Plan asignado
        <select
          id={id("plan")}
          value={plan}
          disabled={pending}
          onChange={(event) => setPlan(event.target.value as PlanId)}
          className={`mt-1 ${CONTROL}`}
        >
          {(["starter", "pro"] as const).map((option) => {
            const price = prices.get(option);
            return (
              <option key={option} value={option}>
                {PLAN_NAME[option]}
                {price === undefined ? "" : ` (${formatPrice(price)}/mes)`}
              </option>
            );
          })}
        </select>
      </label>

      <label htmlFor={id("paid-until")} className="block text-sm font-semibold">
        Vigencia inicial (paid_until)
        <input
          id={id("paid-until")}
          type="date"
          value={paidUntil}
          min={argentineToday()}
          disabled={pending}
          onChange={(event) => setPaidUntil(event.target.value)}
          className={`mt-1 ${CONTROL}`}
        />
      </label>

      <label htmlFor={id("reference")} className="block text-sm font-semibold">
        Referencia de Mercado Pago <span className="font-normal text-movo-slate">(opcional)</span>
        <input
          id={id("reference")}
          type="text"
          value={reference}
          maxLength={100}
          disabled={pending}
          onChange={(event) => setReference(event.target.value)}
          className={`mt-1 ${CONTROL}`}
        />
      </label>

      <label htmlFor={id("notes")} className="block text-sm font-semibold">
        Notas <span className="font-normal text-movo-slate">(opcional)</span>
        <textarea
          id={id("notes")}
          value={notes}
          maxLength={500}
          rows={3}
          disabled={pending}
          onChange={(event) => setNotes(event.target.value)}
          className={`mt-1 resize-none rounded-2xl py-3 ${CONTROL} h-auto`}
        />
      </label>

      <button
        type="button"
        disabled={pending || paidUntil === ""}
        onClick={() => void submit()}
        className="h-14 w-full rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
      >
        {submitLabel}
      </button>
    </div>
  );
}
