"use client";

import { useState, type ReactNode } from "react";

import { Notice } from "@/components/onboarding/fields";
import { XIcon } from "@/components/socios/icons";
import { describeAdminFailure } from "@/lib/admin/messages";
import { changePlan, registerPayment } from "@/lib/admin/subscriptions/api";
import { describeDaysLeft, formatDate, paymentState } from "@/lib/admin/subscriptions/format";
import type { PlanChange, PlanId, SubscriptionRow } from "@/lib/admin/subscriptions/types";
import type { ApiResult } from "@/lib/api/request";
import type { FormMessage } from "@/lib/business/messages";

import { PaymentStatePill, PLAN_NAME, PlanChip } from "./SubscriptionPills";

interface Props {
  row: SubscriptionRow;
  onClose: () => void;
  /** Called after a change, so the list is fetched again. */
  onChanged: () => void;
}

/** The side panel of one subscription: its data and the three billing actions. */
export function SubscriptionDetail({ row, onClose, onChanged }: Props) {
  const [plan, setPlan] = useState<PlanId>(row.plan);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<FormMessage | null>(null);
  const [paid, setPaid] = useState<string | null>(null);
  const [change, setChange] = useState<PlanChange | null>(null);

  async function run<T>(call: () => Promise<ApiResult<T>>, onOk: (data: T) => void) {
    if (pending) return;
    setPending(true);
    setError(null);
    setPaid(null);
    setChange(null);

    const result = await call();
    if (result.ok) onOk(result.data);
    else setError(describeAdminFailure(result));

    setPending(false);
    onChanged();
  }

  const state = paymentState(row);
  const deactivated = change
    ? [...change.deactivated.branches, ...change.deactivated.services]
    : [];

  return (
    <aside
      className="space-y-5 rounded-3xl bg-white p-6 shadow-sm"
      aria-label={`Suscripción de ${row.business_name}`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">
            Detalle de suscripción
          </p>
          <h2 className="mt-2 truncate font-display text-2xl font-extrabold">
            {row.business_name}
          </h2>
          <p className="mt-1 flex items-center gap-2">
            <PlanChip plan={row.plan} />
            <PaymentStatePill state={state} />
            {row.pilot ? <span className="text-xs font-bold text-movo-gold">Piloto</span> : null}
          </p>
        </div>
        <button
          type="button"
          aria-label="Cerrar el detalle"
          onClick={onClose}
          className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <XIcon />
        </button>
      </div>

      {error ? <Notice title={error.title} body={error.body} /> : null}

      <Section title="Datos de la suscripción">
        <dl className="space-y-1 text-sm">
          <Item label="Plan" value={PLAN_NAME[row.plan]} />
          <Item
            label="Vigente hasta"
            value={`${formatDate(row.paid_until)} (${describeDaysLeft(row.days_left)})`}
          />
          <Item label="Gracia hasta" value={formatDate(row.grace_until)} />
          <Item label="Referencia de Mercado Pago" value={row.mp_subscription_ref || "—"} />
          <Item label="Notas" value={row.notes || "—"} />
        </dl>
      </Section>

      <Section title="Acciones">
        <button
          type="button"
          disabled={pending}
          onClick={() =>
            void run(
              () => registerPayment(row.business_id),
              (sub) => setPaid(sub.paid_until),
            )
          }
          className="h-14 w-full rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
        >
          Registrar pago
        </button>
        <p className="text-xs text-movo-slate">Suma un mes a la vigencia.</p>
        {paid ? (
          <p role="status" className="text-sm font-semibold text-movo-ok">
            Pago registrado: ahora vence el {formatDate(paid)}.
          </p>
        ) : null}

        <div className="flex items-end gap-2 border-t border-movo-slate/20 pt-3">
          <label className="min-w-0 flex-1 text-sm font-semibold">
            Cambiar de plan
            <select
              value={plan}
              disabled={pending}
              onChange={(event) => setPlan(event.target.value as PlanId)}
              className="mt-1 h-12 w-full rounded-2xl bg-white px-4 text-base ring-1 ring-movo-slate/30 outline-none focus:ring-2 focus:ring-movo-accent"
            >
              <option value="starter">{PLAN_NAME.starter}</option>
              <option value="pro">{PLAN_NAME.pro}</option>
            </select>
          </label>
          <button
            type="button"
            disabled={pending}
            onClick={() =>
              void run(
                () => changePlan(row.business_id, plan),
                (result) => setChange(result),
              )
            }
            className="h-12 shrink-0 rounded-full bg-movo-card px-5 text-sm font-bold text-movo-accent outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
          >
            Cambiar plan
          </button>
        </div>
        {change ? (
          <div role="status" className="text-sm">
            <p className="font-semibold text-movo-ok">Plan cambiado a {PLAN_NAME[change.plan]}.</p>
            {deactivated.length > 0 ? (
              <>
                <p className="mt-1 text-movo-slate">Se desactivaron por el límite del plan:</p>
                <ul className="list-disc pl-5">
                  {change.deactivated.branches.map((item) => (
                    <li key={`b${item.id}`}>Sede: {item.name}</li>
                  ))}
                  {change.deactivated.services.map((item) => (
                    <li key={`s${item.id}`}>Servicio: {item.name}</li>
                  ))}
                </ul>
              </>
            ) : (
              <p className="text-movo-slate">No se desactivó nada.</p>
            )}
          </div>
        ) : null}

        {row.reminder_url ? (
          <a
            href={row.reminder_url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex h-12 items-center justify-center rounded-full bg-movo-ok/15 px-4 text-sm font-bold text-movo-ok"
          >
            Enviar recordatorio por WhatsApp
          </a>
        ) : null}
        <a
          href="/admin/businesses"
          className="inline-block text-sm font-bold underline underline-offset-4"
        >
          Ver el local en Locales (#{row.business_id})
        </a>
      </Section>
    </aside>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-3">
      <h3 className="text-xs font-bold tracking-wide text-movo-slate uppercase">{title}</h3>
      <div className="space-y-3 rounded-2xl bg-movo-cream p-4">{children}</div>
    </section>
  );
}

function Item({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="inline text-movo-slate">{label}: </dt>
      <dd className="inline font-semibold break-words">{value}</dd>
    </div>
  );
}
